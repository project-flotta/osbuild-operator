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
	buildv1 "github.com/openshift/api/build/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// OSBuildConfigSpec defines the desired state of OSBuildConfig
type OSBuildConfigSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Details defines what to build
	Details BuildDetails `json:"details"`
	// Triggers defines when to build
	Triggers BuildTriggers `json:"triggers"`
	// Template specifying template configuration to use
	Template *Template `json:"template,omitempty"`
}

// Template contains OSBuildConfigTemplate configuration
type Template struct {
	// OSBuildConfigTemplateRef specifies the name of OSBuildConfigTemplate resource
	OSBuildConfigTemplateRef string `json:"osBuildConfigTemplateRef"`
	// Parameters list parameter values for OS Build Config processing
	Parameters []ParameterValue `json:"parameters,omitempty"`
}

// ParameterValue specifies a name-value pair
type ParameterValue struct {
	// Name of a parameter
	Name string `json:"name"`
	// Value of a parameter
	Value string `json:"value"`
}

// BuildDetails includes all the information needed to build the image
type BuildDetails struct {
	// Distribution is the name of the O/S distribution
	Distribution string `json:"distribution"`
	// Customizations defines the changes to be applied on top of the base image (optional)
	Customizations *Customizations `json:"customizations,omitempty"`
	// TargetImage defines the requested output image
	TargetImage TargetImage `json:"targetImage"`
}

// Customizations defines the changes to be applied on top of the base image
type Customizations struct {
	// Packages is a list of RPM packages to install (optional)
	Packages []string `json:"packages,omitempty"`
	// Users is the list of Users to add to the image (optional)
	Users []User `json:"users,omitempty"`
	// Services defines the services to enable or disable (optional)
	Services *Services `json:"services,omitempty"`
}

// User defines a single user to be configured
type User struct {
	// Name is the username for the new user
	Name string `json:"name"`
	// Groups is the groups to add the user to (optional)
	Groups []string `json:"groups,omitempty"`
	// Key is the user's SSH public key (optional)
	PubKey *string `json:"pubKey,omitempty"`
}

type Services struct {
	// Enabled is the list of services to enabled (optional)
	Enabled []string `json:"enabled,omitempty"`
	// Disabled is the list of services to disabled (optional)
	Disabled []string `json:"disabled,omitempty"`
}

type TargetImage struct {
	// Architecture defines target architecture of the image
	Architecture Architecture `json:"architecture"`
	// ImageType defines the target image type
	TargetImageType TargetImageType `json:"targetImageType"`
	// OSTree is the OSTree configuration of the build (optional)
	OSTree *OSTreeConfig `json:"osTree,omitempty"`
}

// +kubebuilder:validation:Enum=x86_64;aarch64
type Architecture string

// +kubebuilder:validation:Enum=edge-container;edge-installer
type TargetImageType string

// OSTreeConfig defines the OSTree ref details
type OSTreeConfig struct {
	// URL is the URL of the target build (Optional)
	URL *string `json:"url,omitempty"`
	// Ref is the ref of the target build (Optional)
	Ref *string `json:"ref,omitempty"`
	// Parent is the ref of the parent of target build (Optional)
	Parent *string `json:"parent,omitempty"`
}

type BuildTriggers struct {
	// ConfigChange if True trigger a new build upon any change in this BuildConfig CR (optional)
	ConfigChange *bool `json:"configChange,omitempty"`
	// WebHook defines the way to trigger a build using a REST call (optional)
	WebHook *buildv1.WebHookTrigger `json:"webHook,omitempty"`
	// TemplateConfigChange if True trigger a new build upon any change to associated BuildConfigTemplate CR (optional).
	// Default: True.
	TemplateConfigChange *bool `json:"templateConfigChange,omitempty"`
}

// OSBuildConfigStatus defines the observed state of OSBuildConfig
type OSBuildConfigStatus struct {
	// LastVersion denotes the number of the last OSBuild CR created for this OSBuildConfig CR
	LastVersion *int `json:"lastVersion,omitempty"`

	// LastTemplateResourceVersion denotes the version of the last OSBuildConfigTemplate resource used by this
	// OSBuildConfig (value of OSBuildConfigTemplate's metadata.resourceVersion) to generate an OSBuild.
	LastTemplateResourceVersion *string `json:"LastTemplateResourceVersion,omitempty"`

	// CurrentTemplateResourceVersion denotes the most current version of the OSBuildConfigTemplate resource used by this
	// OSBuildConfig (value of OSBuildConfigTemplate's metadata.resourceVersion).
	CurrentTemplateResourceVersion *string `json:"CurrentTemplateResourceVersion,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// OSBuildConfig is the Schema for the osbuildconfigs API
type OSBuildConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OSBuildConfigSpec   `json:"spec,omitempty"`
	Status OSBuildConfigStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// OSBuildConfigList contains a list of OSBuildConfig
type OSBuildConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OSBuildConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&OSBuildConfig{}, &OSBuildConfigList{})
}
