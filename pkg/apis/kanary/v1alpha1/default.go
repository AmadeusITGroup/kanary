package v1alpha1

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// IsDefaultedKanaryDeployment used to know if a KanaryDeployment is already defaulted
// returns true if yes, else no
func IsDefaultedKanaryDeployment(kd *KanaryDeployment) bool {
	if !IsDefaultedKanaryDeploymentSpecScale(&kd.Spec.Scale) {
		return false
	}
	if !IsDefaultedKanaryDeploymentSpecTraffic(&kd.Spec.Traffic) {
		return false
	}
	if !IsDefaultedKanaryDeploymentSpecValidation(&kd.Spec.Validation) {
		return false
	}

	return true
}

// IsDefaultedKanaryDeploymentSpecScale used to know if a KanaryDeploymentSpecScale is already defaulted
// returns true if yes, else no
func IsDefaultedKanaryDeploymentSpecScale(scale *KanaryDeploymentSpecScale) bool {
	if scale.Static == nil {
		return false
	}

	if scale.Static != nil {
		if scale.Static.Replicas == nil {
			return false
		}
	}

	return true
}

// IsDefaultedKanaryDeploymentSpecTraffic used to know if a KanaryDeploymentSpecTraffic is already defaulted
// returns true if yes, else no
func IsDefaultedKanaryDeploymentSpecTraffic(t *KanaryDeploymentSpecTraffic) bool {
	if t.Source == NoneKanaryDeploymentSpecTrafficSource ||
		t.Source == ServiceKanaryDeploymentSpecTrafficSource ||
		t.Source == KanaryServiceKanaryDeploymentSpecTrafficSource ||
		t.Source == BothKanaryDeploymentSpecTrafficSource ||
		t.Source == ShadowKanaryDeploymentSpecTrafficSource {
		return true
	}
	return false
}

// IsDefaultedKanaryDeploymentSpecValidation used to know if a KanaryDeploymentSpecValidation is already defaulted
// returns true if yes, else no
func IsDefaultedKanaryDeploymentSpecValidation(v *KanaryDeploymentSpecValidation) bool {
	if v.ValidationPeriod == nil {
		return false
	}

	if v.Manual == nil && v.LabelWatch == nil && v.PromQL == nil {
		return false
	}

	if v.Manual != nil {
		if !(v.Manual.Deadline == NoneKanaryDeploymentSpecValidationManualDeadine ||
			v.Manual.Deadline == ValidKanaryDeploymentSpecValidationManualDeadine ||
			v.Manual.Deadline == InvalidKanaryDeploymentSpecValidationManualDeadine) {
			return false
		}
	}
	return true
}

// DefaultKanaryDeployment used to default a KanaryDeployment
// return a list of errors in case of unvalid fields.
func DefaultKanaryDeployment(kd *KanaryDeployment) *KanaryDeployment {
	defaultedKD := kd.DeepCopy()
	defaultKanaryDeploymentSpec(&defaultedKD.Spec)
	return defaultedKD
}

// defaultKanaryDeploymentSpec used to default a KanaryDeploymentSpec
// return a list of errors in case of unvalid Spec.
func defaultKanaryDeploymentSpec(spec *KanaryDeploymentSpec) {
	defaultKanaryDeploymentSpecScale(&spec.Scale)
	defaultKanaryDeploymentSpecTraffic(&spec.Traffic)
	defaultKanaryDeploymentSpecValidation(&spec.Validation)
}

func defaultKanaryDeploymentSpecScale(s *KanaryDeploymentSpecScale) {
	if s.Static == nil {
		s.Static = &KanaryDeploymentSpecScaleStatic{}
		defaultKanaryDeploymentSpecScaleStatic(s.Static)
	}
}

func defaultKanaryDeploymentSpecScaleStatic(s *KanaryDeploymentSpecScaleStatic) {
	if s.Replicas == nil {
		s.Replicas = NewInt32(1)
	}
}

func defaultKanaryDeploymentSpecTraffic(t *KanaryDeploymentSpecTraffic) {
	if !(t.Source == NoneKanaryDeploymentSpecTrafficSource ||
		t.Source == ServiceKanaryDeploymentSpecTrafficSource ||
		t.Source == KanaryServiceKanaryDeploymentSpecTrafficSource ||
		t.Source == BothKanaryDeploymentSpecTrafficSource ||
		t.Source == ShadowKanaryDeploymentSpecTrafficSource) {
		t.Source = NoneKanaryDeploymentSpecTrafficSource
	}

	if t.Shadow != nil {
		defaultKanaryDeploymentSpecScaleTrafficShadow(t.Shadow)
	}
}

func defaultKanaryDeploymentSpecScaleTrafficShadow(t *KanaryDeploymentSpecTrafficShadow) {
	// TODO nothing todo for the moment
}

func defaultKanaryDeploymentSpecValidation(v *KanaryDeploymentSpecValidation) {
	if v.ValidationPeriod == nil {
		v.ValidationPeriod = &metav1.Duration{
			Duration: 15 * time.Minute,
		}
	}
	if v.Manual == nil && v.LabelWatch == nil && v.PromQL == nil {
		defaultKanaryDeploymentSpecScaleValidationManual(v)
	}
}

func defaultKanaryDeploymentSpecScaleValidationManual(v *KanaryDeploymentSpecValidation) {
	v.Manual = &KanaryDeploymentSpecValidationManual{
		Deadline: NoneKanaryDeploymentSpecValidationManualDeadine,
	}
}

// NewInt32 returns new int32 pointer instance
func NewInt32(i int32) *int32 {
	return &i
}
