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
	"os"

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

	"github.com/project-flotta/osbuild-operator/api/v1alpha1"
	"github.com/project-flotta/osbuild-operator/controllers"
	"github.com/project-flotta/osbuild-operator/internal/conf"
	"github.com/project-flotta/osbuild-operator/internal/indexer"
	"github.com/project-flotta/osbuild-operator/internal/repository/osbuild"
	"github.com/project-flotta/osbuild-operator/internal/repository/osbuildconfig"
	"github.com/project-flotta/osbuild-operator/internal/repository/osbuildconfigtemplate"
	//+kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(v1alpha1.AddToScheme(scheme))
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

	osBuildConfigRepository := osbuildconfig.NewOSBuildConfigRepository(mgr.GetClient())
	osBuildRepository := osbuild.NewOSBuildRepository(mgr.GetClient())
	osBuildConfigTemplateRepository := osbuildconfigtemplate.NewOSBuildConfigTemplateRepository(mgr.GetClient())

	if err = (&controllers.OSBuildConfigReconciler{
		Client:                          mgr.GetClient(),
		Scheme:                          mgr.GetScheme(),
		OSBuildConfigRepository:         osBuildConfigRepository,
		OSBuildRepository:               osBuildRepository,
		OSBuildConfigTemplateRepository: osBuildConfigTemplateRepository,
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

	if err = (&controllers.OSBuildReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "OSBuild")
		os.Exit(1)
	}
	if err = (&controllers.OSBuildEnvConfigReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
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
		Client:                  mgr.GetClient(),
		Scheme:                  mgr.GetScheme(),
		OSBuildConfigRepository: osBuildConfigRepository,
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
