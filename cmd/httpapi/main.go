package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"

	osbuildv1alpha1 "github.com/project-flotta/osbuild-operator/api/v1alpha1"
	"github.com/project-flotta/osbuild-operator/internal/httpapi"
	operatorlogger "github.com/project-flotta/osbuild-operator/internal/logger"
	"github.com/project-flotta/osbuild-operator/internal/manifests"
	osbuildconfiginternal "github.com/project-flotta/osbuild-operator/internal/osbuildconfig"
	"github.com/project-flotta/osbuild-operator/internal/repository/configmap"
	"github.com/project-flotta/osbuild-operator/internal/repository/osbuild"
	"github.com/project-flotta/osbuild-operator/internal/repository/osbuildconfig"
	"github.com/project-flotta/osbuild-operator/internal/repository/osbuildconfigtemplate"
	secretrepository "github.com/project-flotta/osbuild-operator/internal/repository/secret"
	"github.com/project-flotta/osbuild-operator/restapi"
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(osbuildv1alpha1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	err := httpapi.Load()
	if err != nil {
		panic(err.Error())
	}
	logger, err := operatorlogger.Logger(httpapi.GlobalHttpAPIConf.LogLevel)
	if err != nil {
		panic(err.Error())
	}

	clientConfig, err := getRestConfig(httpapi.GlobalHttpAPIConf.Kubeconfig)
	if err != nil {
		logger.Error(err, "Cannot prepare k8s client config")
		panic(err.Error())
	}

	c, err := getClient(clientConfig, client.Options{Scheme: scheme})
	if err != nil {
		logger.Error(err, "Cannot create k8s client")
		panic(err.Error())
	}

	osBuildConfigRepository := osbuildconfig.NewOSBuildConfigRepository(c)
	secretRepository := secretrepository.NewSecretRepository(c)
	osBuildRepository := osbuild.NewOSBuildRepository(c)
	osBuildConfigTemplateRepository := osbuildconfigtemplate.NewOSBuildConfigTemplateRepository(c)
	configMapRepository := configmap.NewConfigMapRepository(c)
	osBuildCRCreator := manifests.NewOSBuildCRCreator(osBuildConfigRepository, osBuildRepository, scheme, osBuildConfigTemplateRepository, configMapRepository)

	h := restapi.Handler(osbuildconfiginternal.NewOSBuildConfigHandler(osBuildConfigRepository, secretRepository, osBuildCRCreator))
	server := &http.Server{
		Addr:              fmt.Sprintf(":%v", httpapi.GlobalHttpAPIConf.HttpPort),
		ReadHeaderTimeout: time.Minute,
		Handler:           h,
	}
	go func() {
		logger.Info("Starting listening to OSBuildConfigHandler server")
		logger.Fatal(server.ListenAndServe())
	}()

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", httpOK)
	mux.HandleFunc("/readyz", httpOK)
	logger.Info("Starting listening to probes services")
	logger.Fatal(http.ListenAndServe(fmt.Sprintf(":%v", httpapi.GlobalHttpAPIConf.ProbesPort), mux))
}

func getRestConfig(kubeconfigPath string) (*rest.Config, error) {
	if kubeconfigPath != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	}
	return ctrl.GetConfig()
}

func getClient(config *rest.Config, options client.Options) (client.Client, error) {
	c, err := client.New(config, options)
	if err != nil {
		return nil, err
	}

	cacheOpts := cache.Options{
		Scheme: options.Scheme,
		Mapper: options.Mapper,
	}
	objCache, err := cache.New(config, cacheOpts)
	if err != nil {
		return nil, err
	}
	background := context.Background()
	go func() {
		err = objCache.Start(background)
	}()
	if err != nil {
		return nil, err
	}
	if !objCache.WaitForCacheSync(background) {
		return nil, errors.New("cannot sync cache")
	}
	return client.NewDelegatingClient(client.NewDelegatingClientInput{
		CacheReader:     objCache,
		Client:          c,
		UncachedObjects: []client.Object{},
	})
}

func httpOK(writer http.ResponseWriter, _ *http.Request) {
	writer.WriteHeader(http.StatusOK)
}
