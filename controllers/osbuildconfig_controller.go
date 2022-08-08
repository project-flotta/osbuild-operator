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
	"fmt"
	"sort"

	"github.com/go-logr/logr"
	"github.com/google/go-cmp/cmp"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	osbuilderv1alpha1 "github.com/project-flotta/osbuild-operator/api/v1alpha1"
	"github.com/project-flotta/osbuild-operator/internal/manifests"
	"github.com/project-flotta/osbuild-operator/internal/predicates"
	"github.com/project-flotta/osbuild-operator/internal/repository/osbuild"
	"github.com/project-flotta/osbuild-operator/internal/repository/osbuildconfig"
)

// OSBuildConfigReconciler reconciles a OSBuildConfig object
type OSBuildConfigReconciler struct {
	OSBuildConfigRepository osbuildconfig.Repository
	OSBuildRepository       osbuild.Repository
	OSBuildCRCreator        manifests.OSBuildCRCreator
}

const (
	// Common errors
	FailedPatchLastTargetType = "Failed to set the OSBuildConfig lastTargetType"

	// Annotations
	webHookAnnotationKey = "last_webhook_trigger_ts"
)

//+kubebuilder:rbac:groups=osbuilder.project-flotta.io,resources=osbuildconfigs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=osbuilder.project-flotta.io,resources=osbuildconfigs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=osbuilder.project-flotta.io,resources=osbuildconfigs/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the OSBuildConfig object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *OSBuildConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Reconciling", "OSBuildConfig", req)

	osBuildConfig, err := r.OSBuildConfigRepository.Read(ctx, req.Name, req.Namespace)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{Requeue: true, RequeueAfter: RequeueForShortDuration}, nil
	}

	if osBuildConfig.DeletionTimestamp != nil {
		// The OSBuild CRs that were created by that OSBuildConfig would be deleted
		// thanks to setting controller reference for each OSBuild CR
		return ctrl.Result{}, nil
	}

	newOSBuildInstanceIsNeeded, err := r.checkIfNewOSBuildInstanceIsNeeded(ctx, osBuildConfig, logger)
	if err != nil {
		return ctrl.Result{Requeue: true, RequeueAfter: RequeueForShortDuration}, nil
	}

	if newOSBuildInstanceIsNeeded {
		logger.Info("new OSBuild instance need to be created for OSBuildConfig", "OSBuildConfig", osBuildConfig.Name)
		// create new OSBuild instance
		targetBuild := osBuildConfig.Spec.Details.TargetImage.TargetImageType
		if targetBuild == osbuilderv1alpha1.EdgeInstallerImageType {
			// start with edge-container OSBuild instance
			targetBuild = osbuilderv1alpha1.EdgeContainerImageType
		}
		return r.createOSBuildInstance(ctx, logger, osBuildConfig, targetBuild)
	} else {
		logger.Info("update OSBuildConfig current status")
		return r.updateOSBuildConfigCurrentStatus(ctx, logger, osBuildConfig)
	}
}

func (r *OSBuildConfigReconciler) checkIfNewOSBuildInstanceIsNeeded(ctx context.Context, osBuildConfig *osbuilderv1alpha1.OSBuildConfig, logger logr.Logger) (bool, error) {
	userConfiguration := r.getSortedUserConfiguration(osBuildConfig)
	userConfigOrWebhookAnnotationWereChanged := false
	patch := client.MergeFrom(osBuildConfig.DeepCopy())

	if osBuildConfig.Status.LastKnownUserConfiguration == nil || !cmp.Equal(osBuildConfig.Status.LastKnownUserConfiguration, &userConfiguration) {
		logger.Info("LastKnownUserConfiguration is nil or different from the last known user configuration")
		osBuildConfig.Status.LastKnownUserConfiguration = &userConfiguration
		userConfigOrWebhookAnnotationWereChanged = true
	}

	if osBuildConfig.Annotations != nil {
		webhookTriggerTS, ok := osBuildConfig.Annotations[webHookAnnotationKey]
		if ok {
			if osBuildConfig.Status.LastWebhookTriggerTS == "" || webhookTriggerTS != osBuildConfig.Status.LastWebhookTriggerTS {
				logger.Info("LastWebhookTriggerTS OR LastWebhookTriggerTS were changed")
				osBuildConfig.Status.LastWebhookTriggerTS = webhookTriggerTS
				userConfigOrWebhookAnnotationWereChanged = true
			}
		}
	}

	if userConfigOrWebhookAnnotationWereChanged {
		errPatch := r.OSBuildConfigRepository.PatchStatus(ctx, osBuildConfig, &patch)
		if errPatch != nil {
			logger.Error(errPatch, "Failed to patch OSBuildConfig status")
			return false, errPatch
		}
	}

	return userConfigOrWebhookAnnotationWereChanged, nil
}

func (r *OSBuildConfigReconciler) getSortedUserConfiguration(osBuildConfig *osbuilderv1alpha1.OSBuildConfig) osbuilderv1alpha1.UserConfiguration {
	userConfiguration := osbuilderv1alpha1.UserConfiguration{}
	if osBuildConfig.Spec.Details.Customizations != nil {
		userConfiguration.Customizations = osBuildConfig.Spec.Details.Customizations.DeepCopy()
		sort.Strings(userConfiguration.Customizations.Packages)

		if userConfiguration.Customizations.Users != nil {
			sort.SliceStable(userConfiguration.Customizations.Users, func(i, j int) bool {
				return userConfiguration.Customizations.Users[i].Name < userConfiguration.Customizations.Users[j].Name
			})
		}

		if userConfiguration.Customizations.Services != nil {
			sort.Strings(userConfiguration.Customizations.Services.Disabled)
			sort.Strings(userConfiguration.Customizations.Services.Enabled)

		}

	}
	if osBuildConfig.Spec.Template != nil {
		userConfiguration.Template = osBuildConfig.Spec.Template
		if userConfiguration.Template.Parameters != nil {
			sort.SliceStable(userConfiguration.Template.Parameters, func(i, j int) bool {
				return userConfiguration.Template.Parameters[i].Name < userConfiguration.Template.Parameters[j].Name
			})
		}
	}

	return userConfiguration
}

func (r *OSBuildConfigReconciler) updateOSBuildConfigCurrentStatus(ctx context.Context, logger logr.Logger, osBuildConfig *osbuilderv1alpha1.OSBuildConfig) (ctrl.Result, error) {
	osBuildName := fmt.Sprintf("%s-%d", osBuildConfig.Name, *osBuildConfig.Status.LastVersion)
	osBuild, err := r.OSBuildRepository.Read(ctx, osBuildName, osBuildConfig.Namespace)
	if err != nil {
		return ctrl.Result{Requeue: true, RequeueAfter: RequeueForShortDuration}, nil
	}

	osBuildStatus := getCondition(osBuild.Status.Conditions)

	switch osBuildStatus {
	case osbuilderv1alpha1.ConditionFailed:
		logger.Info("Last OSBuild instance has failed")
		return ctrl.Result{}, nil

	case osbuilderv1alpha1.ConditionInProgress:
		logger.Info("Last OSBuild instance still in progress")
		return ctrl.Result{Requeue: true, RequeueAfter: RequeueForLongDuration}, nil

	case osbuilderv1alpha1.ConditionReady:
		if osBuildConfig.Spec.Details.TargetImage.TargetImageType != osbuilderv1alpha1.EdgeInstallerImageType || *osBuildConfig.Status.LastBuildType == osbuilderv1alpha1.EdgeInstallerImageType {
			return ctrl.Result{}, nil
		}

		// last build was edge-container - now need to create OSBuild instance for edge-installer
		return r.createOSBuildInstance(ctx, logger, osBuildConfig, osbuilderv1alpha1.EdgeInstallerImageType)

	default:
		return ctrl.Result{Requeue: true, RequeueAfter: RequeueForShortDuration}, nil
	}
}

func getCondition(conditions []osbuilderv1alpha1.Condition) osbuilderv1alpha1.ConditionType {
	for _, c := range conditions {
		if c.Status == metav1.ConditionTrue {
			return c.Type
		}
	}
	return ""
}

func (r *OSBuildConfigReconciler) setOSBuildConfigLastBuildTargetType(ctx context.Context, logger logr.Logger, osBuildConfig *osbuilderv1alpha1.OSBuildConfig, targetType osbuilderv1alpha1.TargetImageType) error {
	patch := client.MergeFrom(osBuildConfig.DeepCopy())
	osBuildConfig.Status.LastBuildType = &targetType
	if errPatch := r.OSBuildConfigRepository.PatchStatus(ctx, osBuildConfig, &patch); errPatch != nil {
		return errPatch
	}

	return nil
}

func (r *OSBuildConfigReconciler) createOSBuildInstance(ctx context.Context, logger logr.Logger, osBuildConfig *osbuilderv1alpha1.OSBuildConfig, targetImageType osbuilderv1alpha1.TargetImageType) (ctrl.Result, error) {
	//Set status to InProgress order to avoid multiple OSBuild instances creation
	err := r.OSBuildCRCreator.Create(ctx, osBuildConfig, targetImageType)
	if err != nil {
		return ctrl.Result{Requeue: true, RequeueAfter: RequeueForShortDuration}, nil
	}

	errPatch := r.setOSBuildConfigLastBuildTargetType(ctx, logger, osBuildConfig, targetImageType)
	if errPatch != nil {
		logger.Error(errPatch, FailedPatchLastTargetType)
	}

	return ctrl.Result{Requeue: true, RequeueAfter: RequeueForLongDuration}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *OSBuildConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&osbuilderv1alpha1.OSBuildConfig{}).
		// Process only spec changes or when related template versions diverge
		WithEventFilter(predicates.OSBuildConfigChangedPredicate{}).
		Complete(r)
}
