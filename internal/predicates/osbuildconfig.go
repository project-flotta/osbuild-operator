package predicates

import (
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/project-flotta/osbuild-operator/api/v1alpha1"
)

type OSBuildConfigChangedPredicate struct {
	predicate.Funcs
}

func (OSBuildConfigChangedPredicate) Update(e event.UpdateEvent) bool {
	if e.ObjectOld == nil {
		return false
	}
	if e.ObjectNew == nil {
		return false
	}

	newConfig, ok := e.ObjectNew.(*v1alpha1.OSBuildConfig)
	if !ok {
		return false
	}

	generationChanged := e.ObjectNew.GetGeneration() != e.ObjectOld.GetGeneration()
	configTriggerEnabled := newConfig.Spec.Triggers.ConfigChange == nil || *newConfig.Spec.Triggers.ConfigChange
	generationChanged = generationChanged && configTriggerEnabled

	var templateChanged bool
	if newConfig.Status.LastTemplateResourceVersion != nil && newConfig.Status.CurrentTemplateResourceVersion != nil {
		templateChanged = *newConfig.Status.LastTemplateResourceVersion != *newConfig.Status.CurrentTemplateResourceVersion
	}

	templateTriggerEnabled := newConfig.Spec.Triggers.TemplateConfigChange == nil || *newConfig.Spec.Triggers.TemplateConfigChange
	templateChanged = templateChanged && templateTriggerEnabled

	return generationChanged || templateChanged
}
