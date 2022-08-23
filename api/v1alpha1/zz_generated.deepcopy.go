//go:build !ignore_autogenerated
// +build !ignore_autogenerated

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

// Code generated by controller-gen. DO NOT EDIT.

package v1alpha1

import (
	"github.com/openshift/api/build/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AWSS3ServiceConfig) DeepCopyInto(out *AWSS3ServiceConfig) {
	*out = *in
	out.CredsSecretReference = in.CredsSecretReference
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AWSS3ServiceConfig.
func (in *AWSS3ServiceConfig) DeepCopy() *AWSS3ServiceConfig {
	if in == nil {
		return nil
	}
	out := new(AWSS3ServiceConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *BuildDetails) DeepCopyInto(out *BuildDetails) {
	*out = *in
	if in.Customizations != nil {
		in, out := &in.Customizations, &out.Customizations
		*out = new(Customizations)
		(*in).DeepCopyInto(*out)
	}
	in.TargetImage.DeepCopyInto(&out.TargetImage)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new BuildDetails.
func (in *BuildDetails) DeepCopy() *BuildDetails {
	if in == nil {
		return nil
	}
	out := new(BuildDetails)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *BuildTriggers) DeepCopyInto(out *BuildTriggers) {
	*out = *in
	if in.ConfigChange != nil {
		in, out := &in.ConfigChange, &out.ConfigChange
		*out = new(bool)
		**out = **in
	}
	if in.WebHook != nil {
		in, out := &in.WebHook, &out.WebHook
		*out = new(v1.WebHookTrigger)
		(*in).DeepCopyInto(*out)
	}
	if in.TemplateConfigChange != nil {
		in, out := &in.TemplateConfigChange, &out.TemplateConfigChange
		*out = new(bool)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new BuildTriggers.
func (in *BuildTriggers) DeepCopy() *BuildTriggers {
	if in == nil {
		return nil
	}
	out := new(BuildTriggers)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ComposerConfig) DeepCopyInto(out *ComposerConfig) {
	*out = *in
	if in.PSQL != nil {
		in, out := &in.PSQL, &out.PSQL
		*out = new(ComposerDBConfig)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ComposerConfig.
func (in *ComposerConfig) DeepCopy() *ComposerConfig {
	if in == nil {
		return nil
	}
	out := new(ComposerConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ComposerDBConfig) DeepCopyInto(out *ComposerDBConfig) {
	*out = *in
	out.ConnectionSecretReference = in.ConnectionSecretReference
	if in.SSLMode != nil {
		in, out := &in.SSLMode, &out.SSLMode
		*out = new(DBSSLMode)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ComposerDBConfig.
func (in *ComposerDBConfig) DeepCopy() *ComposerDBConfig {
	if in == nil {
		return nil
	}
	out := new(ComposerDBConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Condition) DeepCopyInto(out *Condition) {
	*out = *in
	if in.Message != nil {
		in, out := &in.Message, &out.Message
		*out = new(string)
		**out = **in
	}
	if in.LastTransitionTime != nil {
		in, out := &in.LastTransitionTime, &out.LastTransitionTime
		*out = (*in).DeepCopy()
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Condition.
func (in *Condition) DeepCopy() *Condition {
	if in == nil {
		return nil
	}
	out := new(Condition)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ContainerRegistryServiceConfig) DeepCopyInto(out *ContainerRegistryServiceConfig) {
	*out = *in
	out.CredsSecretReference = in.CredsSecretReference
	if in.CABundleSecretReference != nil {
		in, out := &in.CABundleSecretReference, &out.CABundleSecretReference
		*out = new(v1.SecretLocalReference)
		**out = **in
	}
	if in.SkipSSLVerification != nil {
		in, out := &in.SkipSSLVerification, &out.SkipSSLVerification
		*out = new(bool)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ContainerRegistryServiceConfig.
func (in *ContainerRegistryServiceConfig) DeepCopy() *ContainerRegistryServiceConfig {
	if in == nil {
		return nil
	}
	out := new(ContainerRegistryServiceConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Customizations) DeepCopyInto(out *Customizations) {
	*out = *in
	if in.Packages != nil {
		in, out := &in.Packages, &out.Packages
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.Users != nil {
		in, out := &in.Users, &out.Users
		*out = make([]User, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Services != nil {
		in, out := &in.Services, &out.Services
		*out = new(Services)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Customizations.
func (in *Customizations) DeepCopy() *Customizations {
	if in == nil {
		return nil
	}
	out := new(Customizations)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EdgeInstallerBuildDetails) DeepCopyInto(out *EdgeInstallerBuildDetails) {
	*out = *in
	in.OSTree.DeepCopyInto(&out.OSTree)
	if in.Kickstart != nil {
		in, out := &in.Kickstart, &out.Kickstart
		*out = new(NameRef)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EdgeInstallerBuildDetails.
func (in *EdgeInstallerBuildDetails) DeepCopy() *EdgeInstallerBuildDetails {
	if in == nil {
		return nil
	}
	out := new(EdgeInstallerBuildDetails)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ExternalWorkerConfig) DeepCopyInto(out *ExternalWorkerConfig) {
	*out = *in
	out.SSHKeySecretReference = in.SSHKeySecretReference
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ExternalWorkerConfig.
func (in *ExternalWorkerConfig) DeepCopy() *ExternalWorkerConfig {
	if in == nil {
		return nil
	}
	out := new(ExternalWorkerConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *GenericS3ServiceConfig) DeepCopyInto(out *GenericS3ServiceConfig) {
	*out = *in
	if in.AWSS3ServiceConfig != nil {
		in, out := &in.AWSS3ServiceConfig, &out.AWSS3ServiceConfig
		*out = new(AWSS3ServiceConfig)
		**out = **in
	}
	if in.CABundleSecretReference != nil {
		in, out := &in.CABundleSecretReference, &out.CABundleSecretReference
		*out = new(v1.SecretLocalReference)
		**out = **in
	}
	if in.SkipSSLVerification != nil {
		in, out := &in.SkipSSLVerification, &out.SkipSSLVerification
		*out = new(bool)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new GenericS3ServiceConfig.
func (in *GenericS3ServiceConfig) DeepCopy() *GenericS3ServiceConfig {
	if in == nil {
		return nil
	}
	out := new(GenericS3ServiceConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *IsoConfiguration) DeepCopyInto(out *IsoConfiguration) {
	*out = *in
	if in.Kickstart != nil {
		in, out := &in.Kickstart, &out.Kickstart
		*out = new(KickstartFile)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new IsoConfiguration.
func (in *IsoConfiguration) DeepCopy() *IsoConfiguration {
	if in == nil {
		return nil
	}
	out := new(IsoConfiguration)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KickstartFile) DeepCopyInto(out *KickstartFile) {
	*out = *in
	if in.Raw != nil {
		in, out := &in.Raw, &out.Raw
		*out = new(string)
		**out = **in
	}
	if in.ConfigMapName != nil {
		in, out := &in.ConfigMapName, &out.ConfigMapName
		*out = new(string)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KickstartFile.
func (in *KickstartFile) DeepCopy() *KickstartFile {
	if in == nil {
		return nil
	}
	out := new(KickstartFile)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NameRef) DeepCopyInto(out *NameRef) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NameRef.
func (in *NameRef) DeepCopy() *NameRef {
	if in == nil {
		return nil
	}
	out := new(NameRef)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OSBuild) DeepCopyInto(out *OSBuild) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OSBuild.
func (in *OSBuild) DeepCopy() *OSBuild {
	if in == nil {
		return nil
	}
	out := new(OSBuild)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *OSBuild) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OSBuildConfig) DeepCopyInto(out *OSBuildConfig) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OSBuildConfig.
func (in *OSBuildConfig) DeepCopy() *OSBuildConfig {
	if in == nil {
		return nil
	}
	out := new(OSBuildConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *OSBuildConfig) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OSBuildConfigList) DeepCopyInto(out *OSBuildConfigList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]OSBuildConfig, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OSBuildConfigList.
func (in *OSBuildConfigList) DeepCopy() *OSBuildConfigList {
	if in == nil {
		return nil
	}
	out := new(OSBuildConfigList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *OSBuildConfigList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OSBuildConfigSpec) DeepCopyInto(out *OSBuildConfigSpec) {
	*out = *in
	in.Details.DeepCopyInto(&out.Details)
	in.Triggers.DeepCopyInto(&out.Triggers)
	if in.Template != nil {
		in, out := &in.Template, &out.Template
		*out = new(Template)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OSBuildConfigSpec.
func (in *OSBuildConfigSpec) DeepCopy() *OSBuildConfigSpec {
	if in == nil {
		return nil
	}
	out := new(OSBuildConfigSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OSBuildConfigStatus) DeepCopyInto(out *OSBuildConfigStatus) {
	*out = *in
	if in.LastKnownUserConfiguration != nil {
		in, out := &in.LastKnownUserConfiguration, &out.LastKnownUserConfiguration
		*out = new(UserConfiguration)
		(*in).DeepCopyInto(*out)
	}
	if in.LastVersion != nil {
		in, out := &in.LastVersion, &out.LastVersion
		*out = new(int)
		**out = **in
	}
	if in.LastBuildType != nil {
		in, out := &in.LastBuildType, &out.LastBuildType
		*out = new(TargetImageType)
		**out = **in
	}
	if in.LastTemplateResourceVersion != nil {
		in, out := &in.LastTemplateResourceVersion, &out.LastTemplateResourceVersion
		*out = new(string)
		**out = **in
	}
	if in.CurrentTemplateResourceVersion != nil {
		in, out := &in.CurrentTemplateResourceVersion, &out.CurrentTemplateResourceVersion
		*out = new(string)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OSBuildConfigStatus.
func (in *OSBuildConfigStatus) DeepCopy() *OSBuildConfigStatus {
	if in == nil {
		return nil
	}
	out := new(OSBuildConfigStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OSBuildConfigTemplate) DeepCopyInto(out *OSBuildConfigTemplate) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	out.Status = in.Status
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OSBuildConfigTemplate.
func (in *OSBuildConfigTemplate) DeepCopy() *OSBuildConfigTemplate {
	if in == nil {
		return nil
	}
	out := new(OSBuildConfigTemplate)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *OSBuildConfigTemplate) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OSBuildConfigTemplateList) DeepCopyInto(out *OSBuildConfigTemplateList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]OSBuildConfigTemplate, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OSBuildConfigTemplateList.
func (in *OSBuildConfigTemplateList) DeepCopy() *OSBuildConfigTemplateList {
	if in == nil {
		return nil
	}
	out := new(OSBuildConfigTemplateList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *OSBuildConfigTemplateList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OSBuildConfigTemplateSpec) DeepCopyInto(out *OSBuildConfigTemplateSpec) {
	*out = *in
	if in.Customizations != nil {
		in, out := &in.Customizations, &out.Customizations
		*out = new(Customizations)
		(*in).DeepCopyInto(*out)
	}
	if in.Iso != nil {
		in, out := &in.Iso, &out.Iso
		*out = new(IsoConfiguration)
		(*in).DeepCopyInto(*out)
	}
	if in.Parameters != nil {
		in, out := &in.Parameters, &out.Parameters
		*out = make([]Parameter, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OSBuildConfigTemplateSpec.
func (in *OSBuildConfigTemplateSpec) DeepCopy() *OSBuildConfigTemplateSpec {
	if in == nil {
		return nil
	}
	out := new(OSBuildConfigTemplateSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OSBuildConfigTemplateStatus) DeepCopyInto(out *OSBuildConfigTemplateStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OSBuildConfigTemplateStatus.
func (in *OSBuildConfigTemplateStatus) DeepCopy() *OSBuildConfigTemplateStatus {
	if in == nil {
		return nil
	}
	out := new(OSBuildConfigTemplateStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OSBuildEnvConfig) DeepCopyInto(out *OSBuildEnvConfig) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OSBuildEnvConfig.
func (in *OSBuildEnvConfig) DeepCopy() *OSBuildEnvConfig {
	if in == nil {
		return nil
	}
	out := new(OSBuildEnvConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *OSBuildEnvConfig) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OSBuildEnvConfigList) DeepCopyInto(out *OSBuildEnvConfigList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]OSBuildEnvConfig, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OSBuildEnvConfigList.
func (in *OSBuildEnvConfigList) DeepCopy() *OSBuildEnvConfigList {
	if in == nil {
		return nil
	}
	out := new(OSBuildEnvConfigList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *OSBuildEnvConfigList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OSBuildEnvConfigSpec) DeepCopyInto(out *OSBuildEnvConfigSpec) {
	*out = *in
	if in.Composer != nil {
		in, out := &in.Composer, &out.Composer
		*out = new(ComposerConfig)
		(*in).DeepCopyInto(*out)
	}
	if in.Workers != nil {
		in, out := &in.Workers, &out.Workers
		*out = make(WorkersConfig, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	out.RedHatCredsSecretReference = in.RedHatCredsSecretReference
	in.S3Service.DeepCopyInto(&out.S3Service)
	in.ContainerRegistryService.DeepCopyInto(&out.ContainerRegistryService)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OSBuildEnvConfigSpec.
func (in *OSBuildEnvConfigSpec) DeepCopy() *OSBuildEnvConfigSpec {
	if in == nil {
		return nil
	}
	out := new(OSBuildEnvConfigSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OSBuildEnvConfigStatus) DeepCopyInto(out *OSBuildEnvConfigStatus) {
	*out = *in
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]Condition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OSBuildEnvConfigStatus.
func (in *OSBuildEnvConfigStatus) DeepCopy() *OSBuildEnvConfigStatus {
	if in == nil {
		return nil
	}
	out := new(OSBuildEnvConfigStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OSBuildList) DeepCopyInto(out *OSBuildList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]OSBuild, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OSBuildList.
func (in *OSBuildList) DeepCopy() *OSBuildList {
	if in == nil {
		return nil
	}
	out := new(OSBuildList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *OSBuildList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OSBuildSpec) DeepCopyInto(out *OSBuildSpec) {
	*out = *in
	if in.Details != nil {
		in, out := &in.Details, &out.Details
		*out = new(BuildDetails)
		(*in).DeepCopyInto(*out)
	}
	if in.EdgeInstallerDetails != nil {
		in, out := &in.EdgeInstallerDetails, &out.EdgeInstallerDetails
		*out = new(EdgeInstallerBuildDetails)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OSBuildSpec.
func (in *OSBuildSpec) DeepCopy() *OSBuildSpec {
	if in == nil {
		return nil
	}
	out := new(OSBuildSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OSBuildStatus) DeepCopyInto(out *OSBuildStatus) {
	*out = *in
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]Condition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Output != nil {
		in, out := &in.Output, &out.Output
		*out = new(string)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OSBuildStatus.
func (in *OSBuildStatus) DeepCopy() *OSBuildStatus {
	if in == nil {
		return nil
	}
	out := new(OSBuildStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OSTreeConfig) DeepCopyInto(out *OSTreeConfig) {
	*out = *in
	if in.Parent != nil {
		in, out := &in.Parent, &out.Parent
		*out = new(string)
		**out = **in
	}
	if in.Ref != nil {
		in, out := &in.Ref, &out.Ref
		*out = new(string)
		**out = **in
	}
	if in.Url != nil {
		in, out := &in.Url, &out.Url
		*out = new(string)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OSTreeConfig.
func (in *OSTreeConfig) DeepCopy() *OSTreeConfig {
	if in == nil {
		return nil
	}
	out := new(OSTreeConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Parameter) DeepCopyInto(out *Parameter) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Parameter.
func (in *Parameter) DeepCopy() *Parameter {
	if in == nil {
		return nil
	}
	out := new(Parameter)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ParameterValue) DeepCopyInto(out *ParameterValue) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ParameterValue.
func (in *ParameterValue) DeepCopy() *ParameterValue {
	if in == nil {
		return nil
	}
	out := new(ParameterValue)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Repository) DeepCopyInto(out *Repository) {
	*out = *in
	if in.Baseurl != nil {
		in, out := &in.Baseurl, &out.Baseurl
		*out = new(string)
		**out = **in
	}
	if in.CheckGpg != nil {
		in, out := &in.CheckGpg, &out.CheckGpg
		*out = new(bool)
		**out = **in
	}
	if in.Gpgkey != nil {
		in, out := &in.Gpgkey, &out.Gpgkey
		*out = new(string)
		**out = **in
	}
	if in.IgnoreSsl != nil {
		in, out := &in.IgnoreSsl, &out.IgnoreSsl
		*out = new(bool)
		**out = **in
	}
	if in.Metalink != nil {
		in, out := &in.Metalink, &out.Metalink
		*out = new(string)
		**out = **in
	}
	if in.Mirrorlist != nil {
		in, out := &in.Mirrorlist, &out.Mirrorlist
		*out = new(string)
		**out = **in
	}
	if in.PackageSets != nil {
		in, out := &in.PackageSets, &out.PackageSets
		*out = new([]string)
		if **in != nil {
			in, out := *in, *out
			*out = make([]string, len(*in))
			copy(*out, *in)
		}
	}
	if in.Rhsm != nil {
		in, out := &in.Rhsm, &out.Rhsm
		*out = new(bool)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Repository.
func (in *Repository) DeepCopy() *Repository {
	if in == nil {
		return nil
	}
	out := new(Repository)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *S3ServiceConfig) DeepCopyInto(out *S3ServiceConfig) {
	*out = *in
	if in.AWS != nil {
		in, out := &in.AWS, &out.AWS
		*out = new(AWSS3ServiceConfig)
		**out = **in
	}
	if in.GenericS3 != nil {
		in, out := &in.GenericS3, &out.GenericS3
		*out = new(GenericS3ServiceConfig)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new S3ServiceConfig.
func (in *S3ServiceConfig) DeepCopy() *S3ServiceConfig {
	if in == nil {
		return nil
	}
	out := new(S3ServiceConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Services) DeepCopyInto(out *Services) {
	*out = *in
	if in.Disabled != nil {
		in, out := &in.Disabled, &out.Disabled
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.Enabled != nil {
		in, out := &in.Enabled, &out.Enabled
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Services.
func (in *Services) DeepCopy() *Services {
	if in == nil {
		return nil
	}
	out := new(Services)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TargetImage) DeepCopyInto(out *TargetImage) {
	*out = *in
	if in.OSTree != nil {
		in, out := &in.OSTree, &out.OSTree
		*out = new(OSTreeConfig)
		(*in).DeepCopyInto(*out)
	}
	if in.Repositories != nil {
		in, out := &in.Repositories, &out.Repositories
		*out = new([]Repository)
		if **in != nil {
			in, out := *in, *out
			*out = make([]Repository, len(*in))
			for i := range *in {
				(*in)[i].DeepCopyInto(&(*out)[i])
			}
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TargetImage.
func (in *TargetImage) DeepCopy() *TargetImage {
	if in == nil {
		return nil
	}
	out := new(TargetImage)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Template) DeepCopyInto(out *Template) {
	*out = *in
	if in.Parameters != nil {
		in, out := &in.Parameters, &out.Parameters
		*out = make([]ParameterValue, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Template.
func (in *Template) DeepCopy() *Template {
	if in == nil {
		return nil
	}
	out := new(Template)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *User) DeepCopyInto(out *User) {
	*out = *in
	if in.Groups != nil {
		in, out := &in.Groups, &out.Groups
		*out = new([]string)
		if **in != nil {
			in, out := *in, *out
			*out = make([]string, len(*in))
			copy(*out, *in)
		}
	}
	if in.Key != nil {
		in, out := &in.Key, &out.Key
		*out = new(string)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new User.
func (in *User) DeepCopy() *User {
	if in == nil {
		return nil
	}
	out := new(User)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *UserConfiguration) DeepCopyInto(out *UserConfiguration) {
	*out = *in
	if in.Customizations != nil {
		in, out := &in.Customizations, &out.Customizations
		*out = new(Customizations)
		(*in).DeepCopyInto(*out)
	}
	if in.Template != nil {
		in, out := &in.Template, &out.Template
		*out = new(Template)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new UserConfiguration.
func (in *UserConfiguration) DeepCopy() *UserConfiguration {
	if in == nil {
		return nil
	}
	out := new(UserConfiguration)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VMWorkerConfig) DeepCopyInto(out *VMWorkerConfig) {
	*out = *in
	if in.Architecture != nil {
		in, out := &in.Architecture, &out.Architecture
		*out = new(Architecture)
		**out = **in
	}
	in.DataVolumeSource.DeepCopyInto(&out.DataVolumeSource)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VMWorkerConfig.
func (in *VMWorkerConfig) DeepCopy() *VMWorkerConfig {
	if in == nil {
		return nil
	}
	out := new(VMWorkerConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *WorkerConfig) DeepCopyInto(out *WorkerConfig) {
	*out = *in
	if in.VMWorkerConfig != nil {
		in, out := &in.VMWorkerConfig, &out.VMWorkerConfig
		*out = new(VMWorkerConfig)
		(*in).DeepCopyInto(*out)
	}
	if in.ExternalWorkerConfig != nil {
		in, out := &in.ExternalWorkerConfig, &out.ExternalWorkerConfig
		*out = new(ExternalWorkerConfig)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new WorkerConfig.
func (in *WorkerConfig) DeepCopy() *WorkerConfig {
	if in == nil {
		return nil
	}
	out := new(WorkerConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in WorkersConfig) DeepCopyInto(out *WorkersConfig) {
	{
		in := &in
		*out = make(WorkersConfig, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new WorkersConfig.
func (in WorkersConfig) DeepCopy() WorkersConfig {
	if in == nil {
		return nil
	}
	out := new(WorkersConfig)
	in.DeepCopyInto(out)
	return *out
}
