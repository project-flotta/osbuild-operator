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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// OSBuildConfigTemplateSpec defines the desired state of OSBuildConfigTemplate
type OSBuildConfigTemplateSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Customizations defines the changes to be applied on top of the base image (optional)
	Customizations *Customizations `json:"customizations,omitempty"`

	// Iso specifies ISO-level customizations
	Iso *IsoConfiguration `json:"iso,omitempty"`

	// Parameters that are required by the template configuration (i.e. kickstart content)
	Parameters []Parameter `json:"parameters,omitempty"`
}

type Parameter struct {
	// Name of the parameter
	Name string `json:"name"`
	// Type of the parameter. Allowed values: string, int, bool.
	// +kubebuilder:validation:Enum={string,int,bool}
	Type string `json:"type"`
	// DefaultValue specifies what parameter value should be used, if the parameter is not provided
	DefaultValue string `json:"defaultValue"`
}

type IsoConfiguration struct {
	// Kickstart provides content of Kickstart file that has to be added to the target ISO
	Kickstart *KickstartFile `json:"kickstart,omitempty"`
}

type KickstartFile struct {
	// Raw inline content of the Kickstart file
	Raw *string `json:"raw,omitempty"`
	// ConfigMapName name of a config map containing the Kickstart file under `kickstart` key
	ConfigMapName *string `json:"configMapName,omitempty"`
}

// OSBuildConfigTemplateStatus defines the observed state of OSBuildConfigTemplate
type OSBuildConfigTemplateStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// OSBuildConfigTemplate is the Schema for the osbuildconfigtemplates API
type OSBuildConfigTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OSBuildConfigTemplateSpec   `json:"spec,omitempty"`
	Status OSBuildConfigTemplateStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// OSBuildConfigTemplateList contains a list of OSBuildConfigTemplate
type OSBuildConfigTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OSBuildConfigTemplate `json:"items"`
}

func init() {
	SchemeBuilder.Register(&OSBuildConfigTemplate{}, &OSBuildConfigTemplateList{})
}
