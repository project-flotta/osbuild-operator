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

// User defines a single user to be configured
type User struct {
	// Name is the username for the new user
	Name string `json:"name"`
	// Groups is the groups to add the user to (optional)
	Groups []string `json:"groups,omitempty"`
	// Key is the user's SSH public key (optional)
	Key *string `json:"key,omitempty"`
}

// OSTreeConfig defines the OSTree ref details
type OSTreeConfig struct {
	// Url is the URL of the target build
	Url *string `json:"url"`
	// Ref is the ref of the target build
	Ref *string `json:"ref"`
	// Parent is the ref of the parent of target build (Optional)
	Parent *string `json:"parent"`
}

// +kubebuilder:validation:Enum=edge-container;edge-installer
type ImageType string

// BuildDetails includes all the information needed to build the image
type BuildDetails struct {
	// Distribution is the name of the O/S distribution
	Distribution string `json:"distribution"`
	// Packages is a list of RPM packages to install (optional)
	Packages []string `json:"packages,omitempty"`
	// Users is the list of Users to add to the image (optional)
	Users []User `json:"users,omitempty"`
	// EnabledServices is the list of services to enabled (optional)
	EnabledServices []string `json:"enabled_services,omitempty"`
	// DisabledServices is the list of services to disabled (optional)
	DisabledServices []string `json:"disabled_services,omitempty"`
	// Architecture defines target architecture of the image
	Architecture string `json:"architecture"`
	// OSTree is the OSTree configuration of the build
	OSTree *OSTreeConfig `json:"ostree,omitempty"`
	// ImageType defines the target image type
	ImageType ImageType `json:"image_type"`
}

type BuildTriggers struct {
	// ConfigChange if True trigger a new build upon any change in this BuildConfig CR
	ConfigChange *bool `json:"config_change,omitempty"`
}

// OSBuildConfigSpec defines the desired state of OSBuildConfig
type OSBuildConfigSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Details defines what to build
	Details BuildDetails `json:"build_details"`
	// Triggers defines when to build
	Triggers BuildTriggers `json:"build_triggers"`
}

// OSBuildConfigStatus defines the observed state of OSBuildConfig
type OSBuildConfigStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	LastVersion *int `json:"lastversion,omitempty"`
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
