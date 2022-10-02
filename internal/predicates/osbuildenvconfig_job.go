package predicates

import (
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	batchv1 "k8s.io/api/batch/v1"
)

type OSBuildEnvConfigJobFinished struct {
	predicate.Funcs
}

func (OSBuildEnvConfigJobFinished) Create(e event.CreateEvent) bool {
	return false
}

func (OSBuildEnvConfigJobFinished) Delete(e event.DeleteEvent) bool {
	return false
}

func (OSBuildEnvConfigJobFinished) Generic(e event.GenericEvent) bool {
	return false
}

func (OSBuildEnvConfigJobFinished) Update(e event.UpdateEvent) bool {
	if e.ObjectOld == nil {
		return false
	}
	if e.ObjectNew == nil {
		return false
	}

	newJob, ok := e.ObjectNew.(*batchv1.Job)
	if !ok {
		return false
	}

	return newJob.Status.Active == 0
}
