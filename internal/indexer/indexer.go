package indexer

import (
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/project-flotta/osbuild-operator/api/v1alpha1"
)

const (
	// ConfigByConfigTemplate is the name of the indexer for OSBuildConfig by OSBuildConfigTemplate name
	ConfigByConfigTemplate = "config-by-template"
)

func ConfigByTemplateIndexFunc(obj client.Object) []string {
	config, ok := obj.(*v1alpha1.OSBuildConfig)
	if !ok {
		return []string{}
	}
	if config.Spec.Template == nil {
		return []string{}
	}
	if config.Spec.Template.OSBuildConfigTemplateRef == "" {
		return []string{}
	}

	return []string{config.Spec.Template.OSBuildConfigTemplateRef}
}
