// +build !ignore_autogenerated

// Code generated by deepcopy-gen. DO NOT EDIT.

package v1alpha1

import (
	v2beta1 "k8s.io/api/autoscaling/v2beta1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ContinuousValueDeviation) DeepCopyInto(out *ContinuousValueDeviation) {
	*out = *in
	if in.MaxDeviationPercent != nil {
		in, out := &in.MaxDeviationPercent, &out.MaxDeviationPercent
		*out = new(float64)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ContinuousValueDeviation.
func (in *ContinuousValueDeviation) DeepCopy() *ContinuousValueDeviation {
	if in == nil {
		return nil
	}
	out := new(ContinuousValueDeviation)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DeploymentTemplate) DeepCopyInto(out *DeploymentTemplate) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DeploymentTemplate.
func (in *DeploymentTemplate) DeepCopy() *DeploymentTemplate {
	if in == nil {
		return nil
	}
	out := new(DeploymentTemplate)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DiscreteValueOutOfList) DeepCopyInto(out *DiscreteValueOutOfList) {
	*out = *in
	if in.GoodValues != nil {
		in, out := &in.GoodValues, &out.GoodValues
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.BadValues != nil {
		in, out := &in.BadValues, &out.BadValues
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.TolerancePercent != nil {
		in, out := &in.TolerancePercent, &out.TolerancePercent
		*out = new(uint)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DiscreteValueOutOfList.
func (in *DiscreteValueOutOfList) DeepCopy() *DiscreteValueOutOfList {
	if in == nil {
		return nil
	}
	out := new(DiscreteValueOutOfList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *HorizontalPodAutoscalerSpec) DeepCopyInto(out *HorizontalPodAutoscalerSpec) {
	*out = *in
	if in.MinReplicas != nil {
		in, out := &in.MinReplicas, &out.MinReplicas
		*out = new(int32)
		**out = **in
	}
	if in.Metrics != nil {
		in, out := &in.Metrics, &out.Metrics
		*out = make([]v2beta1.MetricSpec, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new HorizontalPodAutoscalerSpec.
func (in *HorizontalPodAutoscalerSpec) DeepCopy() *HorizontalPodAutoscalerSpec {
	if in == nil {
		return nil
	}
	out := new(HorizontalPodAutoscalerSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KanaryDeployment) DeepCopyInto(out *KanaryDeployment) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KanaryDeployment.
func (in *KanaryDeployment) DeepCopy() *KanaryDeployment {
	if in == nil {
		return nil
	}
	out := new(KanaryDeployment)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *KanaryDeployment) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KanaryDeploymentCondition) DeepCopyInto(out *KanaryDeploymentCondition) {
	*out = *in
	in.LastUpdateTime.DeepCopyInto(&out.LastUpdateTime)
	in.LastTransitionTime.DeepCopyInto(&out.LastTransitionTime)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KanaryDeploymentCondition.
func (in *KanaryDeploymentCondition) DeepCopy() *KanaryDeploymentCondition {
	if in == nil {
		return nil
	}
	out := new(KanaryDeploymentCondition)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KanaryDeploymentList) DeepCopyInto(out *KanaryDeploymentList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	out.ListMeta = in.ListMeta
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]KanaryDeployment, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KanaryDeploymentList.
func (in *KanaryDeploymentList) DeepCopy() *KanaryDeploymentList {
	if in == nil {
		return nil
	}
	out := new(KanaryDeploymentList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *KanaryDeploymentList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KanaryDeploymentSpec) DeepCopyInto(out *KanaryDeploymentSpec) {
	*out = *in
	in.Template.DeepCopyInto(&out.Template)
	in.Scale.DeepCopyInto(&out.Scale)
	in.Traffic.DeepCopyInto(&out.Traffic)
	in.Validations.DeepCopyInto(&out.Validations)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KanaryDeploymentSpec.
func (in *KanaryDeploymentSpec) DeepCopy() *KanaryDeploymentSpec {
	if in == nil {
		return nil
	}
	out := new(KanaryDeploymentSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KanaryDeploymentSpecScale) DeepCopyInto(out *KanaryDeploymentSpecScale) {
	*out = *in
	if in.Static != nil {
		in, out := &in.Static, &out.Static
		*out = new(KanaryDeploymentSpecScaleStatic)
		(*in).DeepCopyInto(*out)
	}
	if in.HPA != nil {
		in, out := &in.HPA, &out.HPA
		*out = new(HorizontalPodAutoscalerSpec)
		(*in).DeepCopyInto(*out)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KanaryDeploymentSpecScale.
func (in *KanaryDeploymentSpecScale) DeepCopy() *KanaryDeploymentSpecScale {
	if in == nil {
		return nil
	}
	out := new(KanaryDeploymentSpecScale)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KanaryDeploymentSpecScaleStatic) DeepCopyInto(out *KanaryDeploymentSpecScaleStatic) {
	*out = *in
	if in.Replicas != nil {
		in, out := &in.Replicas, &out.Replicas
		*out = new(int32)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KanaryDeploymentSpecScaleStatic.
func (in *KanaryDeploymentSpecScaleStatic) DeepCopy() *KanaryDeploymentSpecScaleStatic {
	if in == nil {
		return nil
	}
	out := new(KanaryDeploymentSpecScaleStatic)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KanaryDeploymentSpecTraffic) DeepCopyInto(out *KanaryDeploymentSpecTraffic) {
	*out = *in
	if in.Mirror != nil {
		in, out := &in.Mirror, &out.Mirror
		*out = new(KanaryDeploymentSpecTrafficMirror)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KanaryDeploymentSpecTraffic.
func (in *KanaryDeploymentSpecTraffic) DeepCopy() *KanaryDeploymentSpecTraffic {
	if in == nil {
		return nil
	}
	out := new(KanaryDeploymentSpecTraffic)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KanaryDeploymentSpecTrafficMirror) DeepCopyInto(out *KanaryDeploymentSpecTrafficMirror) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KanaryDeploymentSpecTrafficMirror.
func (in *KanaryDeploymentSpecTrafficMirror) DeepCopy() *KanaryDeploymentSpecTrafficMirror {
	if in == nil {
		return nil
	}
	out := new(KanaryDeploymentSpecTrafficMirror)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KanaryDeploymentSpecValidation) DeepCopyInto(out *KanaryDeploymentSpecValidation) {
	*out = *in
	if in.Manual != nil {
		in, out := &in.Manual, &out.Manual
		*out = new(KanaryDeploymentSpecValidationManual)
		**out = **in
	}
	if in.LabelWatch != nil {
		in, out := &in.LabelWatch, &out.LabelWatch
		*out = new(KanaryDeploymentSpecValidationLabelWatch)
		(*in).DeepCopyInto(*out)
	}
	if in.PromQL != nil {
		in, out := &in.PromQL, &out.PromQL
		*out = new(KanaryDeploymentSpecValidationPromQL)
		(*in).DeepCopyInto(*out)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KanaryDeploymentSpecValidation.
func (in *KanaryDeploymentSpecValidation) DeepCopy() *KanaryDeploymentSpecValidation {
	if in == nil {
		return nil
	}
	out := new(KanaryDeploymentSpecValidation)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KanaryDeploymentSpecValidationLabelWatch) DeepCopyInto(out *KanaryDeploymentSpecValidationLabelWatch) {
	*out = *in
	if in.PodInvalidationLabels != nil {
		in, out := &in.PodInvalidationLabels, &out.PodInvalidationLabels
		*out = new(v1.LabelSelector)
		(*in).DeepCopyInto(*out)
	}
	if in.DeploymentInvalidationLabels != nil {
		in, out := &in.DeploymentInvalidationLabels, &out.DeploymentInvalidationLabels
		*out = new(v1.LabelSelector)
		(*in).DeepCopyInto(*out)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KanaryDeploymentSpecValidationLabelWatch.
func (in *KanaryDeploymentSpecValidationLabelWatch) DeepCopy() *KanaryDeploymentSpecValidationLabelWatch {
	if in == nil {
		return nil
	}
	out := new(KanaryDeploymentSpecValidationLabelWatch)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KanaryDeploymentSpecValidationList) DeepCopyInto(out *KanaryDeploymentSpecValidationList) {
	*out = *in
	if in.InitialDelay != nil {
		in, out := &in.InitialDelay, &out.InitialDelay
		*out = new(v1.Duration)
		**out = **in
	}
	if in.ValidationPeriod != nil {
		in, out := &in.ValidationPeriod, &out.ValidationPeriod
		*out = new(v1.Duration)
		**out = **in
	}
	if in.MaxIntervalPeriod != nil {
		in, out := &in.MaxIntervalPeriod, &out.MaxIntervalPeriod
		*out = new(v1.Duration)
		**out = **in
	}
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]KanaryDeploymentSpecValidation, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KanaryDeploymentSpecValidationList.
func (in *KanaryDeploymentSpecValidationList) DeepCopy() *KanaryDeploymentSpecValidationList {
	if in == nil {
		return nil
	}
	out := new(KanaryDeploymentSpecValidationList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KanaryDeploymentSpecValidationManual) DeepCopyInto(out *KanaryDeploymentSpecValidationManual) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KanaryDeploymentSpecValidationManual.
func (in *KanaryDeploymentSpecValidationManual) DeepCopy() *KanaryDeploymentSpecValidationManual {
	if in == nil {
		return nil
	}
	out := new(KanaryDeploymentSpecValidationManual)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KanaryDeploymentSpecValidationPromQL) DeepCopyInto(out *KanaryDeploymentSpecValidationPromQL) {
	*out = *in
	if in.ValueInRange != nil {
		in, out := &in.ValueInRange, &out.ValueInRange
		*out = new(ValueInRange)
		(*in).DeepCopyInto(*out)
	}
	if in.DiscreteValueOutOfList != nil {
		in, out := &in.DiscreteValueOutOfList, &out.DiscreteValueOutOfList
		*out = new(DiscreteValueOutOfList)
		(*in).DeepCopyInto(*out)
	}
	if in.ContinuousValueDeviation != nil {
		in, out := &in.ContinuousValueDeviation, &out.ContinuousValueDeviation
		*out = new(ContinuousValueDeviation)
		(*in).DeepCopyInto(*out)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KanaryDeploymentSpecValidationPromQL.
func (in *KanaryDeploymentSpecValidationPromQL) DeepCopy() *KanaryDeploymentSpecValidationPromQL {
	if in == nil {
		return nil
	}
	out := new(KanaryDeploymentSpecValidationPromQL)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KanaryDeploymentStatus) DeepCopyInto(out *KanaryDeploymentStatus) {
	*out = *in
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]KanaryDeploymentCondition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	out.Report = in.Report
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KanaryDeploymentStatus.
func (in *KanaryDeploymentStatus) DeepCopy() *KanaryDeploymentStatus {
	if in == nil {
		return nil
	}
	out := new(KanaryDeploymentStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KanaryDeploymentStatusReport) DeepCopyInto(out *KanaryDeploymentStatusReport) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KanaryDeploymentStatusReport.
func (in *KanaryDeploymentStatusReport) DeepCopy() *KanaryDeploymentStatusReport {
	if in == nil {
		return nil
	}
	out := new(KanaryDeploymentStatusReport)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ValueInRange) DeepCopyInto(out *ValueInRange) {
	*out = *in
	if in.Min != nil {
		in, out := &in.Min, &out.Min
		*out = new(float64)
		**out = **in
	}
	if in.Max != nil {
		in, out := &in.Max, &out.Max
		*out = new(float64)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ValueInRange.
func (in *ValueInRange) DeepCopy() *ValueInRange {
	if in == nil {
		return nil
	}
	out := new(ValueInRange)
	in.DeepCopyInto(out)
	return out
}
