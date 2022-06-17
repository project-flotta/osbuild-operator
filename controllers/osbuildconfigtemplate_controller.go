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

package controllers

import (
	"context"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/project-flotta/osbuild-operator/api/v1alpha1"
	"github.com/project-flotta/osbuild-operator/internal/repository/osbuildconfig"
	"github.com/project-flotta/osbuild-operator/internal/repository/osbuildconfigtemplate"
)

// OSBuildConfigTemplateReconciler reconciles a OSBuildConfigTemplate object
type OSBuildConfigTemplateReconciler struct {
	client.Client
	Scheme                          *runtime.Scheme
	OSBuildConfigRepository         osbuildconfig.Repository
	OSBuildConfigTemplateRepository osbuildconfigtemplate.Repository
}

//+kubebuilder:rbac:groups=osbuilder.project-flotta.io,resources=osbuildconfigtemplates,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=osbuilder.project-flotta.io,resources=osbuildconfigtemplates/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=osbuilder.project-flotta.io,resources=osbuildconfigtemplates/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the OSBuildConfigTemplate object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.2/pkg/reconcile
func (r *OSBuildConfigTemplateReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Reconciling", "OSBuildConfigTemplate", req)
	template, err := r.OSBuildConfigTemplateRepository.Read(ctx, req.Name, req.Namespace)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Cannot get template")
		return ctrl.Result{}, err
	}

	if template.DeletionTimestamp != nil {
		return ctrl.Result{}, nil
	}

	configs, err := r.OSBuildConfigRepository.ListByOSBuildConfigTemplate(ctx, req.Name, req.Namespace)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Info("no configs found")
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Cannot get configs by template")
		return ctrl.Result{}, err
	}

	for i := range configs {
		config := configs[i]
		err := r.patchOSBuildConfig(ctx, &config, template)
		if err != nil {
			logger.Error(err, "cannot patch OSBuildConfig status with current template version",
				"OSBuildConfig", config.Name, "Namespace", config.Namespace)
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *OSBuildConfigTemplateReconciler) patchOSBuildConfig(ctx context.Context, config *v1alpha1.OSBuildConfig, template *v1alpha1.OSBuildConfigTemplate) error {
	if config.Status.CurrentTemplateResourceVersion == nil ||
		*config.Status.CurrentTemplateResourceVersion == template.ResourceVersion {
		return nil
	}

	patch := client.MergeFrom(config.DeepCopy())
	config.Status.CurrentTemplateResourceVersion = &template.ResourceVersion
	return r.OSBuildConfigRepository.PatchStatus(ctx, config, &patch)
}

// SetupWithManager sets up the controller with the Manager.
func (r *OSBuildConfigTemplateReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.OSBuildConfigTemplate{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(r)
}
