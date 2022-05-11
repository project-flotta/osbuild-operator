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
	"github.com/go-logr/logr"
	osbuilderprojectflottaiov1alpha1 "github.com/project-flotta/osbuild-operator/api/v1alpha1"
	"github.com/project-flotta/osbuild-operator/internal/repository/osbuild"
	"github.com/project-flotta/osbuild-operator/internal/repository/osbuildconfig"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// OSBuildConfigReconciler reconciles a OSBuildConfig object
type OSBuildConfigReconciler struct {
	client.Client
	Scheme                  *runtime.Scheme
	OSBuildConfigRepository osbuildconfig.Repository
	OSBuildRepository       osbuild.Repository
}

//+kubebuilder:rbac:groups=osbuilder.project-flotta.io,resources=osbuildconfigs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=osbuilder.project-flotta.io,resources=osbuildconfigs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=osbuilder.project-flotta.io,resources=osbuildconfigs/finalizers,verbs=update

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
		return ctrl.Result{Requeue: true}, err
	}

	return ctrl.Result{}, nil
}

func (r *OSBuildConfigReconciler) createNewOSBuildCR(ctx context.Context, osBuildConfig *osbuilderprojectflottaiov1alpha1.OSBuildConfig, logger logr.Logger) error {
	osBuildNewVersion := *osBuildConfig.Status.LastVersion + 1

	osBuildConfigSpecDetails := osBuildConfig.Spec.Details.DeepCopy()
	osBuild := &osbuilderprojectflottaiov1alpha1.OSBuild{
		ObjectMeta: metav1.ObjectMeta{
			Name: osBuildConfig.Name + "-" + string(rune(osBuildNewVersion)),
		},
		Spec: osbuilderprojectflottaiov1alpha1.OSBuildSpec{
			Details:     *osBuildConfigSpecDetails,
			TriggeredBy: "UpdateCR",
		},
	}

	// Set the owner of the osBuild CR to be osBuildConfig in order to manage lifecycle of the osBuild CR.
	// Especially in deletion of osBuildConfig CR
	err := controllerutil.SetControllerReference(osBuildConfig, osBuild, r.Scheme)
	if err != nil {
		logger.Error(err, "cannot create osBuild")
		return err
	}

	patch := client.MergeFrom(osBuildConfig.DeepCopy())
	osBuildConfig.Status.LastVersion = &osBuildNewVersion
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

	logger.Info("A new OSBuild CR was created", osBuild.Name)

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *OSBuildConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&osbuilderprojectflottaiov1alpha1.OSBuildConfig{}).
		Complete(r)
}
