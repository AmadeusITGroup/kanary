package v1alpha1

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/api/autoscaling/v2beta1"
	corev1 "k8s.io/api/core/v1"
)

// DefaultCPUUtilization is the default value for CPU utilization, provided no other
// metrics are present.  This is here because it's used by both the v2beta1 defaulting
// logic, and the pseudo-defaulting done in v1 conversion.
const DefaultCPUUtilization = 80

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
	if scale.Static == nil && scale.HPA == nil {
		return false
	}

	if scale.Static != nil {
		if scale.Static.Replicas == nil {
			return false
		}
	}

	if scale.HPA != nil {
		if scale.HPA.MinReplicas == nil {
			return false
		}

		if scale.HPA.MaxReplicas == 0 {
			return false
		}

		if scale.HPA.Metrics == nil {
			return false
		}

		if len(scale.HPA.Metrics) == 0 {
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
		t.Source == MirrorKanaryDeploymentSpecTrafficSource {
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

	if v.InitialDelay == nil {
		return false
	}

	if v.Manual == nil && v.LabelWatch == nil && v.PromQL == nil {
		return false
	}

	if v.Manual != nil {
		if !(v.Manual.StatusAfterDealine == NoneKanaryDeploymentSpecValidationManualDeadineStatus ||
			v.Manual.StatusAfterDealine == ValidKanaryDeploymentSpecValidationManualDeadineStatus ||
			v.Manual.StatusAfterDealine == InvalidKanaryDeploymentSpecValidationManualDeadineStatus) {
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
	if s.Static == nil && s.HPA == nil {
		s.Static = &KanaryDeploymentSpecScaleStatic{}
	}
	if s.Static != nil {
		defaultKanaryDeploymentSpecScaleStatic(s.Static)
	}
	if s.HPA != nil {
		defaultKanaryDeploymentSpecScaleHPA(s.HPA)
	}
}

// defaultKanaryDeploymentSpecScaleHPA used to default HorizontalPodAutoscaler spec
func defaultKanaryDeploymentSpecScaleHPA(s *HorizontalPodAutoscalerSpec) {
	if s.MinReplicas == nil {
		s.MinReplicas = NewInt32(1)
	}
	if s.MaxReplicas == 0 {
		s.MaxReplicas = int32(10)
	}
	if s.Metrics == nil {
		s.Metrics = []v2beta1.MetricSpec{
			{
				Type: v2beta1.ResourceMetricSourceType,
				Resource: &v2beta1.ResourceMetricSource{
					Name:                     corev1.ResourceCPU,
					TargetAverageUtilization: NewInt32(DefaultCPUUtilization),
				},
			},
		}
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
		t.Source == MirrorKanaryDeploymentSpecTrafficSource) {
		t.Source = NoneKanaryDeploymentSpecTrafficSource
	}

	if t.Mirror != nil {
		defaultKanaryDeploymentSpecScaleTrafficMirror(t.Mirror)
	}
}

func defaultKanaryDeploymentSpecScaleTrafficMirror(t *KanaryDeploymentSpecTrafficMirror) {
	// TODO nothing todo for the moment
}

func defaultKanaryDeploymentSpecValidation(v *KanaryDeploymentSpecValidation) {
	if v.ValidationPeriod == nil {
		v.ValidationPeriod = &metav1.Duration{
			Duration: 15 * time.Minute,
		}
	}
	if v.InitialDelay == nil {
		v.InitialDelay = &metav1.Duration{
			Duration: 0 * time.Minute,
		}
	}
	if v.Manual == nil && v.LabelWatch == nil && v.PromQL == nil {
		defaultKanaryDeploymentSpecScaleValidationManual(v)
	}
}

func defaultKanaryDeploymentSpecScaleValidationManual(v *KanaryDeploymentSpecValidation) {
	v.Manual = &KanaryDeploymentSpecValidationManual{
		StatusAfterDealine: NoneKanaryDeploymentSpecValidationManualDeadineStatus,
	}
}

// NewInt32 returns new int32 pointer instance
func NewInt32(i int32) *int32 {
	return &i
}
