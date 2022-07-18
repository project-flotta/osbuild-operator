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

// OSBuildSpec defines the desired state of OSBuild
type OSBuildSpec struct {
	// Details defines what to build
	Details BuildDetails `json:"details"`

	// Kickstart is a reference to a configmap that may store content of a
	// kickstart file to be used in the target image
	Kickstart *NameRef `json:"kickstart,omitempty" protobuf:"bytes,2,opt,name=kickstart"`

	// TriggeredBy explains what triggered the build out
	TriggeredBy TriggeredBy `json:"triggeredBy"`
}

type NameRef struct {
	// The ConfigMap to select from.
	Name string `json:"name"`
}

// +kubebuilder:validation:Enum=UpdateCR;Webhook
type TriggeredBy string

// OSBuildStatus defines the observed state of OSBuild
type OSBuildStatus struct {
	// The conditions present the latest available observations of a build's current state
	Conditions []OSBuildCondition `json:"conditions,omitempty"`

	// +optional
	Output *string `json:"output,omitempty"`

	// ComposeId presents compose id that was already started, for tracking a job of edge-container
	// +optional
	ContainerComposeId string `json:"containerComposeId,omitempty"`

	// IsoComposeId presents compose id that was already started, for tracking a job of edge-installe
	// +optional
	IsoComposeId string `json:"isoComposeId,omitempty"`

	// ContainerUrl presents the url of the edge-container image
	// +optional
	ContainerUrl string `json:"containerUrl,omitempty"`

	// IsoUrl presents the url of the edge-installer image
	// +optional
	IsoUrl string `json:"isoUrl,omitempty"`
}

type OSBuildCondition struct {
	// Type of status
	// +kubebuilder:validation:Enum=startedContainerBuild;failedContainerBuild;containerBuildDone;startedIsoBuild;failedIsoBuild;isoBuildDone;
	Type OSBuildConditionType `json:"type" description:"type of OSBuildCondition condition"`

	// Status of the condition, one of True, False, Unknown
	Status metav1.ConditionStatus `json:"status" description:"status of the condition, one of True, False, Unknown"`

	// A human-readable message indicating details about last transition
	// +kubebuilder:optional
	Message *string `json:"message,omitempty" description:"one-word CamelCase reason for the condition's last transition"`

	// The last time the condition transit from one status to another
	// +optional
	LastTransitionTime *metav1.Time `json:"lastTransitionTime,omitempty" description:"last time the condition transit from one status to another"`
}

type OSBuildConditionType string

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// OSBuild is the Schema for the osbuilds API
type OSBuild struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OSBuildSpec   `json:"spec,omitempty"`
	Status OSBuildStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// OSBuildList contains a list of OSBuild
type OSBuildList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OSBuild `json:"items"`
}

func init() {
	SchemeBuilder.Register(&OSBuild{}, &OSBuildList{})
}
