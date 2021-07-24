// +build !ignore_autogenerated

/*
Copyright 2021 Cortex Labs, Inc.

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
	"k8s.io/api/core/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AutoscalingSpec) DeepCopyInto(out *AutoscalingSpec) {
	*out = *in
	out.Window = in.Window
	out.DownscaleStabilizationPeriod = in.DownscaleStabilizationPeriod
	out.UpscaleStabilizationPeriod = in.UpscaleStabilizationPeriod
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AutoscalingSpec.
func (in *AutoscalingSpec) DeepCopy() *AutoscalingSpec {
	if in == nil {
		return nil
	}
	out := new(AutoscalingSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ComputeSpec) DeepCopyInto(out *ComputeSpec) {
	*out = *in
	if in.CPU != nil {
		in, out := &in.CPU, &out.CPU
		x := (*in).DeepCopy()
		*out = &x
	}
	if in.Mem != nil {
		in, out := &in.Mem, &out.Mem
		x := (*in).DeepCopy()
		*out = &x
	}
	if in.Shm != nil {
		in, out := &in.Shm, &out.Shm
		x := (*in).DeepCopy()
		*out = &x
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ComputeSpec.
func (in *ComputeSpec) DeepCopy() *ComputeSpec {
	if in == nil {
		return nil
	}
	out := new(ComputeSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ContainerSpec) DeepCopyInto(out *ContainerSpec) {
	*out = *in
	if in.Command != nil {
		in, out := &in.Command, &out.Command
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.Args != nil {
		in, out := &in.Args, &out.Args
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.Env != nil {
		in, out := &in.Env, &out.Env
		*out = make([]v1.EnvVar, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Compute != nil {
		in, out := &in.Compute, &out.Compute
		*out = new(ComputeSpec)
		(*in).DeepCopyInto(*out)
	}
	if in.ReadinessProbe != nil {
		in, out := &in.ReadinessProbe, &out.ReadinessProbe
		*out = new(v1.Probe)
		(*in).DeepCopyInto(*out)
	}
	if in.LivenessProbe != nil {
		in, out := &in.LivenessProbe, &out.LivenessProbe
		*out = new(v1.Probe)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ContainerSpec.
func (in *ContainerSpec) DeepCopy() *ContainerSpec {
	if in == nil {
		return nil
	}
	out := new(ContainerSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NetworkingSpec) DeepCopyInto(out *NetworkingSpec) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NetworkingSpec.
func (in *NetworkingSpec) DeepCopy() *NetworkingSpec {
	if in == nil {
		return nil
	}
	out := new(NetworkingSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PodSpec) DeepCopyInto(out *PodSpec) {
	*out = *in
	if in.Containers != nil {
		in, out := &in.Containers, &out.Containers
		*out = make([]ContainerSpec, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PodSpec.
func (in *PodSpec) DeepCopy() *PodSpec {
	if in == nil {
		return nil
	}
	out := new(PodSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RealtimeAPI) DeepCopyInto(out *RealtimeAPI) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	out.Status = in.Status
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RealtimeAPI.
func (in *RealtimeAPI) DeepCopy() *RealtimeAPI {
	if in == nil {
		return nil
	}
	out := new(RealtimeAPI)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *RealtimeAPI) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RealtimeAPIList) DeepCopyInto(out *RealtimeAPIList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]RealtimeAPI, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RealtimeAPIList.
func (in *RealtimeAPIList) DeepCopy() *RealtimeAPIList {
	if in == nil {
		return nil
	}
	out := new(RealtimeAPIList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *RealtimeAPIList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RealtimeAPISpec) DeepCopyInto(out *RealtimeAPISpec) {
	*out = *in
	in.Pod.DeepCopyInto(&out.Pod)
	out.Autoscaling = in.Autoscaling
	if in.NodeGroups != nil {
		in, out := &in.NodeGroups, &out.NodeGroups
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	out.UpdateStrategy = in.UpdateStrategy
	out.Networking = in.Networking
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RealtimeAPISpec.
func (in *RealtimeAPISpec) DeepCopy() *RealtimeAPISpec {
	if in == nil {
		return nil
	}
	out := new(RealtimeAPISpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RealtimeAPIStatus) DeepCopyInto(out *RealtimeAPIStatus) {
	*out = *in
	out.ReplicaCounts = in.ReplicaCounts
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RealtimeAPIStatus.
func (in *RealtimeAPIStatus) DeepCopy() *RealtimeAPIStatus {
	if in == nil {
		return nil
	}
	out := new(RealtimeAPIStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *UpdateStratagySpec) DeepCopyInto(out *UpdateStratagySpec) {
	*out = *in
	out.MaxSurge = in.MaxSurge
	out.MaxUnavailable = in.MaxUnavailable
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new UpdateStratagySpec.
func (in *UpdateStratagySpec) DeepCopy() *UpdateStratagySpec {
	if in == nil {
		return nil
	}
	out := new(UpdateStratagySpec)
	in.DeepCopyInto(out)
	return out
}
