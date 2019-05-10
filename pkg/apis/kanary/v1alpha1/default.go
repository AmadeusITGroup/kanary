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
	if !IsDefaultedKanaryDeploymentSpecValidationList(&kd.Spec.Validations) {
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
func IsDefaultedKanaryDeploymentSpecValidationList(list *KanaryDeploymentSpecValidationList) bool {
	if list.ValidationPeriod == nil {
		return false
	}

	if list.InitialDelay == nil {
		return false
	}

	if list.MaxIntervalPeriod == nil {
		return false
	}

	if list.Items == nil {
		return false
	}
	if len(list.Items) == 0 {
		return false
	}

	for _, v := range list.Items {
		if isInit := IsDefaultedKanaryDeploymentSpecValidation(&v); !isInit {
			return false
		}
	}

	return true
}

// IsDefaultedKanaryDeploymentSpecValidation used to know if a KanaryDeploymentSpecValidation is already defaulted
// returns true if yes, else no
func IsDefaultedKanaryDeploymentSpecValidation(v *KanaryDeploymentSpecValidation) bool {
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

	if v.PromQL != nil {
		if !isDefaultedKanaryDeploymentSpecValidationPromQL(v.PromQL) {
			return false
		}
	}

	return true
}

func isDefaultedKanaryDeploymentSpecValidationPromQL(pq *KanaryDeploymentSpecValidationPromQL) bool {
	if pq.PrometheusService == "" {
		return false
	}
	if pq.PodNameKey == "" {
		return false
	}
	if pq.DiscreteValueOutOfList != nil && !isDefaultedKanaryDeploymentSpecValidationPromQLDiscrete(pq.DiscreteValueOutOfList) {
		return false
	}
	if pq.ContinuousValueDeviation != nil && !isDefaultedKanaryDeploymentSpecValidationPromQLContinuous(pq.ContinuousValueDeviation) {
		return false
	}
	if pq.ValueInRange != nil && !isDefaultedKanaryDeploymentSpecValidationPromQLValueInRange(pq.ValueInRange) {
		return false
	}

	return true
}
func isDefaultedKanaryDeploymentSpecValidationPromQLValueInRange(c *ValueInRange) bool {
	return c.Min != nil && c.Max != nil
}

func isDefaultedKanaryDeploymentSpecValidationPromQLContinuous(c *ContinuousValueDeviation) bool {
	return c.MaxDeviationPercent != nil
}

func isDefaultedKanaryDeploymentSpecValidationPromQLDiscrete(d *DiscreteValueOutOfList) bool {
	return d.TolerancePercent != nil
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
	defaultKanaryDeploymentSpecValidationList(&spec.Validations)
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

func defaultKanaryDeploymentSpecValidationList(list *KanaryDeploymentSpecValidationList) {
	if list == nil {
		return
	}
	if list.ValidationPeriod == nil {
		list.ValidationPeriod = &metav1.Duration{
			Duration: 15 * time.Minute,
		}
	}
	if list.InitialDelay == nil {
		list.InitialDelay = &metav1.Duration{
			Duration: 0 * time.Minute,
		}
	}
	if list.MaxIntervalPeriod == nil {
		list.MaxIntervalPeriod = &metav1.Duration{
			Duration: 20 * time.Second,
		}
	}

	if list.Items == nil || len(list.Items) == 0 {
		list.Items = []KanaryDeploymentSpecValidation{
			{},
		}
	}
	for id, value := range list.Items {
		defaultKanaryDeploymentSpecValidation(&value)
		list.Items[id] = value
	}
}

func defaultKanaryDeploymentSpecValidation(v *KanaryDeploymentSpecValidation) {
	if v.Manual == nil && v.LabelWatch == nil && v.PromQL == nil {
		defaultKanaryDeploymentSpecScaleValidationManual(v)
	}
	if v.Manual != nil {
		if v.Manual.StatusAfterDealine == "" {
			v.Manual.StatusAfterDealine = NoneKanaryDeploymentSpecValidationManualDeadineStatus
		}
	}
	if v.PromQL != nil {
		defaultKanaryDeploymentSpecValidationPromQL(v.PromQL)

	}
}
func defaultKanaryDeploymentSpecValidationPromQL(pq *KanaryDeploymentSpecValidationPromQL) {
	if pq.PrometheusService == "" {
		pq.PrometheusService = "prometheus:9090"
	}
	if pq.PodNameKey == "" {
		pq.PodNameKey = "pod"
	}
	if pq.ContinuousValueDeviation != nil {
		defaultKanaryDeploymentSpecValidationPromQLContinuous(pq.ContinuousValueDeviation)
	}
	if pq.DiscreteValueOutOfList != nil {
		defaultKanaryDeploymentSpecValidationPromQLDiscreteValueOutOfList(pq.DiscreteValueOutOfList)
	}
	if pq.ValueInRange != nil {
		defaultKanaryDeploymentSpecValidationPromQLValueInRange(pq.ValueInRange)
	}
}
func defaultKanaryDeploymentSpecValidationPromQLValueInRange(c *ValueInRange) {
	if c.Min == nil {
		c.Min = NewFloat64(0)
	}
	if c.Max == nil {
		c.Max = NewFloat64(1)
	}
}
func defaultKanaryDeploymentSpecValidationPromQLContinuous(c *ContinuousValueDeviation) {
	if c.MaxDeviationPercent == nil {
		c.MaxDeviationPercent = NewFloat64(10)
	}
}
func defaultKanaryDeploymentSpecValidationPromQLDiscreteValueOutOfList(d *DiscreteValueOutOfList) {
	if d.TolerancePercent == nil {
		d.TolerancePercent = NewUInt(0)
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

// NewUInt returns new uint pointer instance
func NewUInt(i uint) *uint {
	return &i
}

// NewFloat64 return a pointer to a float64
func NewFloat64(val float64) *float64 {
	return &val
}
