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
	runtime "k8s.io/apimachinery/pkg/runtime"
)

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
	if in.LastVersion != nil {
		in, out := &in.LastVersion, &out.LastVersion
		*out = new(int)
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
func (in *OSTreeConfig) DeepCopyInto(out *OSTreeConfig) {
	*out = *in
	if in.URL != nil {
		in, out := &in.URL, &out.URL
		*out = new(string)
		**out = **in
	}
	if in.Ref != nil {
		in, out := &in.Ref, &out.Ref
		*out = new(string)
		**out = **in
	}
	if in.Parent != nil {
		in, out := &in.Parent, &out.Parent
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
func (in *Services) DeepCopyInto(out *Services) {
	*out = *in
	if in.Enabled != nil {
		in, out := &in.Enabled, &out.Enabled
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.Disabled != nil {
		in, out := &in.Disabled, &out.Disabled
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
func (in *User) DeepCopyInto(out *User) {
	*out = *in
	if in.Groups != nil {
		in, out := &in.Groups, &out.Groups
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.PubKey != nil {
		in, out := &in.PubKey, &out.PubKey
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