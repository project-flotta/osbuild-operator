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

// OSBuildEnvConfigSpec defines the desired state of OSBuildEnvConfig
type OSBuildEnvConfigSpec struct {
	// Composer contains all the required configuration values for the Composer service
	// +kubebuilder:validation:Optional
	Composer *ComposerConfig `json:"composer"`
	// Workers is a list of WorkerConfig each providing the configuration required for a worker
	// +kubebuilder:validation:Required
	Workers WorkersConfig `json:"workers"`
	// RedHatCredsSecretReference is a reference to a secret in the same namespace,
	// containing the RedHat Portal credentials to be used by the Worker machines
	// The expected keys are username and password
	// +kubebuilder:validation:Required
	RedHatCredsSecretReference buildv1.SecretLocalReference `json:"redHatCredsSecretReference"`
	// S3Service holds the configuration needed to connect to the S3 service
	// +kubebuilder:validation:Required
	S3Service S3ServiceConfig `json:"s3Service"`
}

type ComposerConfig struct {
	// PSQL is the configuration of the DB server (optional)
	// +kubebuilder:validation:Optional
	PSQL *ComposerDBConfig `json:"psql,omitempty"`
}

type ComposerDBConfig struct {
	// RedHatCredsSecretReference is a reference to a secret in the same namespace,
	// containing the connection details to the PSQL service
	// The expected keys are: host, port, dbname, user, password
	ConnectionSecretReference buildv1.SecretLocalReference `json:"connectionSecretReference"`
	// SSLMode is the SSL mode to use when connecting to the PSQL server
	// As defined here: https://www.postgresql.org/docs/current/libpq-connect.html#LIBPQ-CONNECT-SSLMODE
	// (optional - default is prefer)
	// +kubebuilder:validation:Optional
	SSLMode *DBSSLMode `json:"sslMode,omitempty"`
}

// +kubebuilder:validation:Enum=disable;allow;prefer;require;verify-ca;verify-full
type DBSSLMode string

// +kubebuilder:validation:MinItems=1
type WorkersConfig []WorkerConfig

type WorkerConfig struct {
	// Name is a unique identifier for the Worker
	// +kubebuilder:validation:Required
	Name string `json:"name"`
	// VMWorkerConfig hold the configuration needed to start a managed VM to act as a Worker
	//
	// +kubebuilder:validation:OneOf
	VMWorkerConfig *VMWorkerConfig `json:"vmWorkerConfig,omitempty"`

	// ExternalWorkerConfig hold the configuration needed to configure an existing machine to act as a Worker
	//
	// +kubebuilder:validation:OneOf
	ExternalWorkerConfig *ExternalWorkerConfig `json:"externalWorkerConfig,omitempty"`
}

type VMWorkerConfig struct {
	// Architecture defines the architecture of the worker machine
	// +kubebuilder:validation:Optional
	Architecture *Architecture `json:"architecture"`
	// ImageURL is the location of the rhel qcow2 file to run on the worker
	// +kubebuilder:validation:Required
	ImageURL string `json:"imageURL"`
}

type ExternalWorkerConfig struct {
	// Address is the hostname or IP address of the external worker machine
	// +kubebuilder:validation:Required
	Address string `json:"address"`
	// User is the user to connect with to external worker machine
	// +kubebuilder:validation:Required
	User string `json:"user"`
	// RedHatCredsSecretReference is a reference to a secret in the same namespace,
	// containing the private key that may be used to connect to the external worker machine
	// the expected key is ssh-privatekey
	// +kubebuilder:validation:Required
	SSHKeySecretReference buildv1.SecretLocalReference `json:"redHatCredsSecretReference"`
}

type S3ServiceConfig struct {
	// AWSS3ServiceConfig hold the configuration needed to connect to AWS S3
	//
	// +kubebuilder:validation:OneOf
	AWS *AWSS3ServiceConfig `json:"awsS3ServiceConfig,omitempty"`

	// GenericS3ServiceConfig hold the configuration needed to connect to a generic S3 service
	//
	// +kubebuilder:validation:OneOf
	GenericS3 *GenericS3ServiceConfig `json:"genericS3ServiceConfig,omitempty"`
}

type AWSS3ServiceConfig struct {
	// CredsSecretReference is a reference to a secret in the same namespace,
	// containing the connection credentials for the S3 service
	// The required keys are access-key-id and secret-access-key
	// +kubebuilder:validation:Required
	CredsSecretReference buildv1.SecretLocalReference `json:"credsSecretReference"`
	// Region is the region to use when connecting to the S3 service
	// +kubebuilder:validation:Required
	Region string `json:"region"`
	// Bucket is the bucket to store images in
	// +kubebuilder:validation:Required
	Bucket string `json:"bucket"`
}

type GenericS3ServiceConfig struct {
	*AWSS3ServiceConfig `json:",inline"`

	// Endpoint is the Url of the S3 service
	// +kubebuilder:validation:Required
	Endpoint string `json:"endpoint"`
	// CABundleSecretReference is a reference to a secret in the same namespace,
	// containing the CA certificate to use when connecting to the S3 service (optional, default empty)
	// +kubebuilder:validation:Optional
	CABundleSecretReference *buildv1.SecretLocalReference `json:"caBundleSecretReference,omitempty"`
	// SkipSSLVerification when set to true the SSL certificate will not be verified (optional, default False)
	// +kubebuilder:validation:Optional
	SkipSSLVerification *bool `json:"skipSSLVerification,omitempty"`
}

// OSBuildEnvConfigStatus defines the observed state of OSBuildEnvConfig
type OSBuildEnvConfigStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope=Cluster

// OSBuildEnvConfig is the Schema for the osbuildenvconfigs API
type OSBuildEnvConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OSBuildEnvConfigSpec   `json:"spec,omitempty"`
	Status OSBuildEnvConfigStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// OSBuildEnvConfigList contains a list of OSBuildEnvConfig
type OSBuildEnvConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OSBuildEnvConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&OSBuildEnvConfig{}, &OSBuildEnvConfigList{})
}
