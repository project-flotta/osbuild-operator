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
	"github.com/project-flotta/osbuild-operator/internal/customizations"
	"time"

	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	osbuilderprojectflottaiov1alpha1 "github.com/project-flotta/osbuild-operator/api/v1alpha1"
	"github.com/project-flotta/osbuild-operator/internal/predicates"
	"github.com/project-flotta/osbuild-operator/internal/repository/osbuild"
	"github.com/project-flotta/osbuild-operator/internal/repository/osbuildconfig"
	"github.com/project-flotta/osbuild-operator/internal/repository/osbuildconfigtemplate"
	"github.com/project-flotta/osbuild-operator/internal/templates"
)

var zero int

// OSBuildConfigReconciler reconciles a OSBuildConfig object
type OSBuildConfigReconciler struct {
	client.Client
	Scheme                          *runtime.Scheme
	OSBuildConfigRepository         osbuildconfig.Repository
	OSBuildRepository               osbuild.Repository
	OSBuildConfigTemplateRepository osbuildconfigtemplate.Repository
}

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
		return ctrl.Result{Requeue: true}, err
	}

	if osBuildConfig.DeletionTimestamp != nil {
		// The OSBuild CRs that were created by that OSBuildConfig would be deleted
		// thanks to setting controller reference for each OSBuild CR
		return ctrl.Result{}, nil
	}

	err = r.createNewOSBuildCR(ctx, osBuildConfig, logger)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{RequeueAfter: 30 * time.Second}, err
		}
		return ctrl.Result{Requeue: true}, err
	}

	return ctrl.Result{}, nil
}

func (r *OSBuildConfigReconciler) createNewOSBuildCR(ctx context.Context, osBuildConfig *osbuilderprojectflottaiov1alpha1.OSBuildConfig, logger logr.Logger) error {
	lastVersion := osBuildConfig.Status.LastVersion
	if lastVersion == nil {
		lastVersion = &zero
	}
	osBuildNewVersion := *lastVersion + 1

	osBuildName := fmt.Sprintf("%s-%d", osBuildConfig.Name, osBuildNewVersion)
	osBuild := &osbuilderprojectflottaiov1alpha1.OSBuild{
		ObjectMeta: metav1.ObjectMeta{
			Name:      osBuildName,
			Namespace: osBuildConfig.Namespace,
		},
		Spec: osbuilderprojectflottaiov1alpha1.OSBuildSpec{
			TriggeredBy: "UpdateCR",
		},
	}

	osBuildConfigSpecDetails := osBuildConfig.Spec.Details.DeepCopy()
	kickstartConfigMap, osConfigTemplate, err := r.applyTemplate(ctx, osBuildConfig, osBuildConfigSpecDetails, osBuildName, osBuild)
	if err != nil {
		logger.Error(err, "cannot apply template to osBuild")
		return err
	}
	osBuild.Spec.Details = *osBuildConfigSpecDetails

	// Set the owner of the osBuild CR to be osBuildConfig in order to manage lifecycle of the osBuild CR.
	// Especially in deletion of osBuildConfig CR
	err = controllerutil.SetControllerReference(osBuildConfig, osBuild, r.Scheme)
	if err != nil {
		logger.Error(err, "cannot create osBuild")
		return err
	}

	patch := client.MergeFrom(osBuildConfig.DeepCopy())
	osBuildConfig.Status.LastVersion = &osBuildNewVersion
	if osConfigTemplate != nil {
		osBuildConfig.Status.CurrentTemplateResourceVersion = &osConfigTemplate.ResourceVersion
		osBuildConfig.Status.LastTemplateResourceVersion = &osConfigTemplate.ResourceVersion
	}
	err = r.OSBuildConfigRepository.PatchStatus(ctx, osBuildConfig, &patch)
	if err != nil {
		logger.Error(err, "cannot update the field lastVersion of osBuildConfig")
		return err
	}

	err = r.OSBuildRepository.Create(ctx, osBuild)
	if err != nil {
		logger.Error(err, "cannot create osBuild")
		return err
	}

	if kickstartConfigMap != nil {
		err = r.setKickstartConfigMapOwner(ctx, kickstartConfigMap, osBuild)
		if err != nil {
			logger.Error(err, "cannot set controller reference to kickstart config map")
			return err
		}
	}

	logger.Info("A new OSBuild CR was created", "OSBuild", osBuild.Name)

	return nil
}

func (r *OSBuildConfigReconciler) applyTemplate(ctx context.Context, osBuildConfig *osbuilderprojectflottaiov1alpha1.OSBuildConfig, osBuildConfigSpecDetails *osbuilderprojectflottaiov1alpha1.BuildDetails, osBuildName string, osBuild *osbuilderprojectflottaiov1alpha1.OSBuild) (*v1.ConfigMap, *osbuilderprojectflottaiov1alpha1.OSBuildConfigTemplate, error) {
	var kickstartConfigMap *v1.ConfigMap
	var osConfigTemplate *osbuilderprojectflottaiov1alpha1.OSBuildConfigTemplate
	if template := osBuildConfig.Spec.Template; template != nil {
		var err error
		osConfigTemplate, err = r.OSBuildConfigTemplateRepository.Read(ctx, template.OSBuildConfigTemplateRef, osBuildConfig.Namespace)
		if err != nil {
			return nil, nil, err
		}

		osBuildConfigSpecDetails.Customizations = customizations.MergeCustomizations(osConfigTemplate.Spec.Customizations, osBuildConfigSpecDetails.Customizations)

		kickstartConfigMap, err = r.createKickstartConfigMap(ctx, osBuildConfig, osConfigTemplate, osBuildName, osBuild.Namespace)
		if err != nil {
			return nil, nil, err
		}
		if kickstartConfigMap != nil {
			osBuild.Spec.Kickstart = &osbuilderprojectflottaiov1alpha1.NameRef{Name: osBuildName}
		}
	}
	return kickstartConfigMap, osConfigTemplate, nil
}

func (r *OSBuildConfigReconciler) setKickstartConfigMapOwner(ctx context.Context, kickstartConfigMap *v1.ConfigMap, osBuild *osbuilderprojectflottaiov1alpha1.OSBuild) error {
	patch := client.MergeFrom(kickstartConfigMap)
	err := controllerutil.SetOwnerReference(osBuild, kickstartConfigMap, r.Scheme)
	if err != nil {
		return err
	}
	return r.Client.Patch(ctx, kickstartConfigMap, patch)
}

func (r *OSBuildConfigReconciler) createKickstartConfigMap(ctx context.Context, osBuildConfig *osbuilderprojectflottaiov1alpha1.OSBuildConfig, osConfigTemplate *osbuilderprojectflottaiov1alpha1.OSBuildConfigTemplate, name, namespace string) (*v1.ConfigMap, error) {
	kickstart, err := r.getKickstart(ctx, osConfigTemplate, osBuildConfig)
	if err != nil {
		return nil, err
	}

	if kickstart == nil {
		return nil, nil
	}

	cm := &v1.ConfigMap{}
	err = r.Client.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, cm)
	if err == nil {
		// CM has already been created, returning it
		return cm, nil
	}
	if !errors.IsNotFound(err) {
		return nil, err
	}

	cm = &v1.ConfigMap{
		ObjectMeta: ctrl.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: map[string]string{
			"kickstart": *kickstart,
		},
	}

	err = r.Client.Create(ctx, cm)
	if err != nil {
		return nil, err
	}
	return cm, nil
}

func (r *OSBuildConfigReconciler) getKickstart(ctx context.Context, osConfigTemplate *osbuilderprojectflottaiov1alpha1.OSBuildConfigTemplate, osBuildConfig *osbuilderprojectflottaiov1alpha1.OSBuildConfig) (*string, error) {
	if osConfigTemplate.Spec.Iso == nil || osConfigTemplate.Spec.Iso.Kickstart == nil {
		return nil, nil
	}
	if osConfigTemplate.Spec.Iso.Kickstart.Raw == nil && osConfigTemplate.Spec.Iso.Kickstart.ConfigMapName == nil {
		return nil, nil
	}

	var kickstartTemplate string
	if osConfigTemplate.Spec.Iso.Kickstart.Raw != nil {
		kickstartTemplate = *osConfigTemplate.Spec.Iso.Kickstart.Raw
	} else {
		cm := v1.ConfigMap{}
		err := r.Client.Get(ctx, types.NamespacedName{Name: *osConfigTemplate.Spec.Iso.Kickstart.ConfigMapName, Namespace: osBuildConfig.Namespace}, &cm)
		if err != nil {
			return nil, err
		}
		var ok bool
		if kickstartTemplate, ok = cm.Data["kickstart"]; !ok {
			return nil, errors.NewNotFound(schema.GroupResource{Group: "configmap", Resource: "key"}, "kickstart")
		}
	}

	finalKickstart, err := templates.ProcessOSBuildConfigTemplate(kickstartTemplate, osConfigTemplate.Spec.Parameters, osBuildConfig.Spec.Template.Parameters)
	if err != nil {
		return nil, err
	}
	return &finalKickstart, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *OSBuildConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&osbuilderprojectflottaiov1alpha1.OSBuildConfig{}).
		// Process only spec changes or when related template versions diverge
		WithEventFilter(predicates.OSBuildConfigChangedPredicate{}).
		Complete(r)
}
