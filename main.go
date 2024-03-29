/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"
	"path"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"go.uber.org/zap/zapcore"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	routev1 "github.com/openshift/api/route/v1"
	kubevirtv1 "kubevirt.io/api/core/v1"

	"github.com/project-flotta/osbuild-operator/api/v1alpha1"
	"github.com/project-flotta/osbuild-operator/controllers"
	"github.com/project-flotta/osbuild-operator/internal/composer"
	"github.com/project-flotta/osbuild-operator/internal/conf"
	"github.com/project-flotta/osbuild-operator/internal/indexer"
	"github.com/project-flotta/osbuild-operator/internal/manifests"
	"github.com/project-flotta/osbuild-operator/internal/repository/certificate"
	"github.com/project-flotta/osbuild-operator/internal/repository/configmap"
	"github.com/project-flotta/osbuild-operator/internal/repository/deployment"
	"github.com/project-flotta/osbuild-operator/internal/repository/job"
	"github.com/project-flotta/osbuild-operator/internal/repository/osbuild"
	"github.com/project-flotta/osbuild-operator/internal/repository/osbuildconfig"
	"github.com/project-flotta/osbuild-operator/internal/repository/osbuildconfigtemplate"
	"github.com/project-flotta/osbuild-operator/internal/repository/osbuildenvconfig"
	"github.com/project-flotta/osbuild-operator/internal/repository/route"
	"github.com/project-flotta/osbuild-operator/internal/repository/secret"
	"github.com/project-flotta/osbuild-operator/internal/repository/service"
	"github.com/project-flotta/osbuild-operator/internal/repository/virtualmachine"
	"github.com/project-flotta/osbuild-operator/internal/sshkey"
	//+kubebuilder:scaffold:imports
)

const (
	osBuildCertsDir          = "/etc/osbuild/certs"
	composerFormatServerName = "https://%s/api/image-builder-composer/v2/"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(v1alpha1.AddToScheme(scheme))

	utilruntime.Must(certmanagerv1.AddToScheme(scheme))

	utilruntime.Must(kubevirtv1.AddToScheme(scheme))

	utilruntime.Must(routev1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	err := conf.Load()
	if err != nil {
		setupLog.Error(err, "failed to load the configuration")
		os.Exit(1)
	}

	var level zapcore.Level
	err = level.UnmarshalText([]byte(conf.GlobalConf.LogLevel))
	if err != nil {
		setupLog.Error(err, "unable to unmarshal log level", "log level", conf.GlobalConf.LogLevel)
		os.Exit(1)
	}
	opts := zap.Options{}
	opts.Level = level
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	setupLog = ctrl.Log
	setupLog.Info("Started with configuration", "configuration", conf.GlobalConf)

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     conf.GlobalConf.MetricsAddr,
		Port:                   conf.GlobalConf.WebhookPort,
		HealthProbeBindAddress: conf.GlobalConf.ProbeAddr,
		LeaderElection:         conf.GlobalConf.EnableLeaderElection,
		LeaderElectionID:       conf.GlobalConf.LeaderElectionResourceName,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	ctx := context.Background()
	err = mgr.GetFieldIndexer().IndexField(ctx, &v1alpha1.OSBuildConfig{}, indexer.ConfigByConfigTemplate, indexer.ConfigByTemplateIndexFunc)
	if err != nil {
		setupLog.Error(err, "Failed to create indexer for OSBuildConfig")
		os.Exit(1)
	}

	osBuildEnvConfigRepository := osbuildenvconfig.NewOSBuildEnvConfigRepository(mgr.GetClient())
	osBuildConfigRepository := osbuildconfig.NewOSBuildConfigRepository(mgr.GetClient())
	osBuildRepository := osbuild.NewOSBuildRepository(mgr.GetClient())
	osBuildConfigTemplateRepository := osbuildconfigtemplate.NewOSBuildConfigTemplateRepository(mgr.GetClient())
	configMapRepository := configmap.NewConfigMapRepository(mgr.GetClient())
	certificateRepository := certificate.NewCertificateRepository(mgr.GetClient())
	deploymentRepository := deployment.NewDeploymentRepository(mgr.GetClient())
	jobRepository := job.NewJobRepository(mgr.GetClient())
	serviceRepository := service.NewServiceRepository(mgr.GetClient())
	secretRepository := secret.NewSecretRepository(mgr.GetClient())
	routeRepository := route.NewRouteRepository(mgr.GetClient())
	virtualMachineRepository := virtualmachine.NewVirtualMachineRepository(mgr.GetClient())
	sshkeyGenerator := sshkey.NewSSHKeyGenerator()

	osBuildCRCreator := manifests.NewOSBuildCRCreator(osBuildConfigRepository, osBuildRepository, scheme, osBuildConfigTemplateRepository, configMapRepository)

	if err = (&controllers.OSBuildConfigReconciler{
		OSBuildConfigRepository: osBuildConfigRepository,
		OSBuildRepository:       osBuildRepository,
		OSBuildCRCreator:        osBuildCRCreator,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "OSBuildConfig")
		os.Exit(1)
	}

	// webhooks
	if conf.GlobalConf.EnableWebhooks {
		if err = (&v1alpha1.OSBuildConfig{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "OSBuildConfig")
			os.Exit(1)
		}
	}

	setupLog.Info("Create a composer client")
	composerClient, err := createClient()
	if err != nil {
		setupLog.Error(err, "unable to create composer client")
		os.Exit(1)
	}

	if err = (&controllers.OSBuildReconciler{
		Scheme:            mgr.GetScheme(),
		OSBuildRepository: osBuildRepository,
		ComposerClient:    composerClient,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "OSBuild")
		os.Exit(1)
	}
	if err = (&controllers.OSBuildEnvConfigReconciler{
		Scheme:                     mgr.GetScheme(),
		OSBuildEnvConfigRepository: osBuildEnvConfigRepository,
		CertificateRepository:      certificateRepository,
		ConfigMapRepository:        configMapRepository,
		DeploymentRepository:       deploymentRepository,
		JobRepository:              jobRepository,
		ServiceRepository:          serviceRepository,
		SecretRepository:           secretRepository,
		RouteRepository:            routeRepository,
		VirtualMachineRepository:   virtualMachineRepository,
		SSHKeyGenerator:            sshkeyGenerator,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "OSBuildEnvConfig")
		os.Exit(1)
	}
	if conf.GlobalConf.EnableWebhooks {
		if err = (&v1alpha1.OSBuildEnvConfig{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "OSBuildEnvConfig")
			os.Exit(1)
		}
	}
	if err = (&controllers.OSBuildConfigTemplateReconciler{
		Client:                          mgr.GetClient(),
		Scheme:                          mgr.GetScheme(),
		OSBuildConfigRepository:         osBuildConfigRepository,
		OSBuildConfigTemplateRepository: osBuildConfigTemplateRepository,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "OSBuildConfigTemplate")
		os.Exit(1)
	}
	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

func createClient() (composer.ClientWithResponsesInterface, error) {
	ca := path.Join(osBuildCertsDir, "ca.crt")
	tlsCert := path.Join(osBuildCertsDir, "tls.crt")
	tlsKey := path.Join(osBuildCertsDir, "tls.key")
	var tlsConfig *tls.Config

	caCert, err := os.ReadFile(ca)
	if err != nil {
		return nil, err
	}

	cert, err := tls.LoadX509KeyPair(tlsCert, tlsKey)
	if err != nil {
		return nil, err
	}

	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	tlsConfig = &tls.Config{
		MinVersion:   tls.VersionTLS12,
		RootCAs:      caCertPool,
		Certificates: []tls.Certificate{cert},
	}

	transport := &http.Transport{TLSClientConfig: tlsConfig}
	httpClient, err := &http.Client{Transport: transport}, nil
	if err != nil {
		setupLog.Error(err, "unable to create http client")
		return nil, err
	}

	composerClient := &composer.Client{
		Server:         fmt.Sprintf(composerFormatServerName, controllers.ComposerComposerAPIServiceName),
		Client:         httpClient,
		RequestEditors: nil,
	}

	return &composer.ClientWithResponses{
		ClientInterface: composerClient,
	}, nil
}
