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

package v1alpha1

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

type osBuildEnvConfigError struct {
	error string
}

// Error returns non-empty string if there was an error.
func (e osBuildEnvConfigError) Error() string {
	return e.error
}

var (
	crAlreadyExists = osBuildEnvConfigError{
		error: "an OSBuildEnvConfig already exists",
	}
	workerNamesNotUnique = osBuildEnvConfigError{
		error: "worker names must be unique",
	}
	updateNotSupported = osBuildEnvConfigError{
		error: "OSBuildEnvConfig cannot be updated",
	}
)

// log is for logging in this package.
var osbuildenvconfiglog = logf.Log.WithName("osbuildenvconfig-resource")
var kClient client.Client

func (r *OSBuildEnvConfig) SetupWebhookWithManager(mgr ctrl.Manager) error {
	kClient = mgr.GetClient()
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

//+kubebuilder:webhook:path=/mutate-osbuilder-project-flotta-io-v1alpha1-osbuildenvconfig,mutating=true,failurePolicy=fail,sideEffects=None,groups=osbuilder.project-flotta.io,resources=osbuildenvconfigs,verbs=create;update,versions=v1alpha1,name=mosbuildenvconfig.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &OSBuildEnvConfig{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *OSBuildEnvConfig) Default() {
	osbuildenvconfiglog.Info("default", "name", r.Name)

	// TODO(user): fill in your defaulting logic.
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-osbuilder-project-flotta-io-v1alpha1-osbuildenvconfig,mutating=false,failurePolicy=fail,sideEffects=None,groups=osbuilder.project-flotta.io,resources=osbuildenvconfigs,verbs=create;update,versions=v1alpha1,name=vosbuildenvconfig.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &OSBuildEnvConfig{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *OSBuildEnvConfig) ValidateCreate() error {
	osbuildenvconfiglog.Info("validate create", "name", r.Name)

	err := validateSingleton()
	if err != nil {
		return err
	}

	err = validateUniqueWorkerNames(r.Spec.Workers)
	if err != nil {
		return err
	}

	return nil
}

func validateSingleton() error {
	ctx := context.Background()
	osBuildEnvConfigList := OSBuildEnvConfigList{}
	err := kClient.List(ctx, &osBuildEnvConfigList)
	if err != nil {
		return err
	}

	if len(osBuildEnvConfigList.Items) > 0 {
		return crAlreadyExists
	}

	return nil
}

func validateUniqueWorkerNames(workers []WorkerConfig) error {
	workerNames := make(map[string]struct{})
	for _, worker := range workers {
		if _, exists := workerNames[worker.Name]; exists {
			return workerNamesNotUnique
		}
		workerNames[worker.Name] = struct{}{}
	}
	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *OSBuildEnvConfig) ValidateUpdate(old runtime.Object) error {
	osbuildenvconfiglog.Info("validate update", "name", r.Name)

	// TODO(user): fill in your validation logic upon object update.
	return updateNotSupported
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *OSBuildEnvConfig) ValidateDelete() error {
	osbuildenvconfiglog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}
