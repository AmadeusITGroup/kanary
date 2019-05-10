package v1alpha1

import (
	"reflect"
	"testing"
	"time"

	"k8s.io/api/autoscaling/v2beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestIsDefaultedKanaryDeployment(t *testing.T) {

	tests := []struct {
		name string
		kd   *KanaryDeployment
		want bool
	}{
		{
			name: "not defaulted",
			kd:   &KanaryDeployment{},
			want: false,
		},
		{
			name: "is defaulted",
			kd: &KanaryDeployment{
				Spec: KanaryDeploymentSpec{
					Scale: KanaryDeploymentSpecScale{
						Static: &KanaryDeploymentSpecScaleStatic{
							Replicas: NewInt32(1),
						},
					},
					Traffic: KanaryDeploymentSpecTraffic{
						Source: ServiceKanaryDeploymentSpecTrafficSource,
					},
					Validations: KanaryDeploymentSpecValidationList{
						ValidationPeriod: &metav1.Duration{
							Duration: 15 * time.Minute,
						},
						InitialDelay: &metav1.Duration{
							Duration: 5 * time.Minute,
						},
						MaxIntervalPeriod: &metav1.Duration{
							Duration: 5 * time.Minute,
						},
						Items: []KanaryDeploymentSpecValidation{
							{
								Manual: &KanaryDeploymentSpecValidationManual{
									StatusAfterDealine: NoneKanaryDeploymentSpecValidationManualDeadineStatus,
								},
							},
						},
					},
				},
			},
			want: true,
		},
		{
			name: "is defaulted",
			kd: &KanaryDeployment{
				Spec: KanaryDeploymentSpec{
					Scale: KanaryDeploymentSpecScale{
						Static: &KanaryDeploymentSpecScaleStatic{
							Replicas: NewInt32(1),
						},
					},
					Traffic: KanaryDeploymentSpecTraffic{
						Source: ServiceKanaryDeploymentSpecTrafficSource,
					},
					Validations: KanaryDeploymentSpecValidationList{
						ValidationPeriod: &metav1.Duration{
							Duration: 15 * time.Minute,
						},
						InitialDelay: &metav1.Duration{
							Duration: 5 * time.Minute,
						},
						Items: []KanaryDeploymentSpecValidation{
							{
								PromQL: &KanaryDeploymentSpecValidationPromQL{
									PrometheusService:        "s",
									PodNameKey:               "pod",
									ContinuousValueDeviation: &ContinuousValueDeviation{},
								},
							},
						},
					},
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsDefaultedKanaryDeployment(tt.kd); got != tt.want {
				t.Errorf("IsDefaultedKanaryDeployment() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDefaultKanaryDeployment(t *testing.T) {

	tests := []struct {
		name string
		kd   *KanaryDeployment
		want *KanaryDeployment
	}{
		{
			name: "not defaulted",
			kd: &KanaryDeployment{
				Spec: KanaryDeploymentSpec{},
			},
			want: &KanaryDeployment{
				Spec: KanaryDeploymentSpec{
					Scale: KanaryDeploymentSpecScale{
						Static: &KanaryDeploymentSpecScaleStatic{
							Replicas: NewInt32(1),
						},
					},
					Traffic: KanaryDeploymentSpecTraffic{
						Source: NoneKanaryDeploymentSpecTrafficSource,
					},
					Validations: KanaryDeploymentSpecValidationList{
						ValidationPeriod: &metav1.Duration{
							Duration: 15 * time.Minute,
						},
						InitialDelay: &metav1.Duration{
							Duration: 0 * time.Minute,
						},
						MaxIntervalPeriod: &metav1.Duration{
							Duration: 20 * time.Second,
						},
						Items: []KanaryDeploymentSpecValidation{
							{
								Manual: &KanaryDeploymentSpecValidationManual{
									StatusAfterDealine: NoneKanaryDeploymentSpecValidationManualDeadineStatus,
								},
							},
						},
					},
				},
			},
		},

		{
			name: "already some configuration",
			kd: &KanaryDeployment{
				Spec: KanaryDeploymentSpec{
					Scale: KanaryDeploymentSpecScale{
						Static: &KanaryDeploymentSpecScaleStatic{
							Replicas: NewInt32(1),
						},
					},
					Traffic: KanaryDeploymentSpecTraffic{
						Source: KanaryServiceKanaryDeploymentSpecTrafficSource,
					},
					Validations: KanaryDeploymentSpecValidationList{
						ValidationPeriod: &metav1.Duration{
							Duration: 30 * time.Minute,
						},
						InitialDelay: &metav1.Duration{
							Duration: 5 * time.Minute,
						},
						MaxIntervalPeriod: &metav1.Duration{
							Duration: 5 * time.Minute,
						},
						Items: []KanaryDeploymentSpecValidation{
							{
								PromQL: &KanaryDeploymentSpecValidationPromQL{
									Query:                    "foo",
									ContinuousValueDeviation: &ContinuousValueDeviation{},
								},
							},
						},
					},
				},
			},
			want: &KanaryDeployment{
				Spec: KanaryDeploymentSpec{
					Scale: KanaryDeploymentSpecScale{
						Static: &KanaryDeploymentSpecScaleStatic{
							Replicas: NewInt32(1),
						},
					},
					Traffic: KanaryDeploymentSpecTraffic{
						Source: KanaryServiceKanaryDeploymentSpecTrafficSource,
					},
					Validations: KanaryDeploymentSpecValidationList{
						ValidationPeriod: &metav1.Duration{
							Duration: 30 * time.Minute,
						},
						InitialDelay: &metav1.Duration{
							Duration: 5 * time.Minute,
						},
						MaxIntervalPeriod: &metav1.Duration{
							Duration: 5 * time.Minute,
						},
						Items: []KanaryDeploymentSpecValidation{
							{
								PromQL: &KanaryDeploymentSpecValidationPromQL{
									PrometheusService: "prometheus:9090",
									Query:             "foo",
									PodNameKey:        "pod",
									ContinuousValueDeviation: &ContinuousValueDeviation{
										MaxDeviationPercent: NewFloat64(10),
									},
								},
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DefaultKanaryDeployment(tt.kd); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DefaultKanaryDeployment() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsDefaultedKanaryDeploymentSpecScale(t *testing.T) {
	type args struct {
		scale *KanaryDeploymentSpecScale
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "already defaulted with static",
			args: args{
				scale: &KanaryDeploymentSpecScale{
					Static: &KanaryDeploymentSpecScaleStatic{
						Replicas: NewInt32(1),
					},
				},
			},
			want: true,
		},
		{
			name: "not defaulted at all",
			args: args{
				scale: &KanaryDeploymentSpecScale{},
			},
			want: false,
		},
		{
			name: "HPA not defaulted (minReplicas, maxReplicas)",
			args: args{
				scale: &KanaryDeploymentSpecScale{
					HPA: &HorizontalPodAutoscalerSpec{},
				},
			},
			want: false,
		},
		{
			name: "HPA not defaulted (Metrics)",
			args: args{
				scale: &KanaryDeploymentSpecScale{
					HPA: &HorizontalPodAutoscalerSpec{
						MinReplicas: NewInt32(1),
						MaxReplicas: int32(5),
					},
				},
			},
			want: false,
		},
		{
			name: "HPA not defaulted (Metrics empty slice)",
			args: args{
				scale: &KanaryDeploymentSpecScale{
					HPA: &HorizontalPodAutoscalerSpec{
						MinReplicas: NewInt32(1),
						MaxReplicas: int32(5),
						Metrics:     []v2beta1.MetricSpec{},
					},
				},
			},
			want: false,
		},
		{
			name: "HPA defaulted ",
			args: args{
				scale: &KanaryDeploymentSpecScale{
					HPA: &HorizontalPodAutoscalerSpec{
						MinReplicas: NewInt32(1),
						MaxReplicas: int32(5),
						Metrics:     []v2beta1.MetricSpec{{}},
					},
				},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsDefaultedKanaryDeploymentSpecScale(tt.args.scale); got != tt.want {
				t.Errorf("IsDefaultedKanaryDeploymentSpecScale() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_defaultKanaryDeploymentSpecValidationList(t *testing.T) {
	tests := []struct {
		name string
		list *KanaryDeploymentSpecValidationList
		want *KanaryDeploymentSpecValidationList
	}{
		{
			name: "nil list",
			list: &KanaryDeploymentSpecValidationList{},
			want: &KanaryDeploymentSpecValidationList{
				ValidationPeriod: &metav1.Duration{
					Duration: 15 * time.Minute,
				},
				InitialDelay: &metav1.Duration{
					Duration: 0 * time.Minute,
				},
				MaxIntervalPeriod: &metav1.Duration{
					Duration: 20 * time.Second,
				},
				Items: []KanaryDeploymentSpecValidation{
					{
						Manual: &KanaryDeploymentSpecValidationManual{
							StatusAfterDealine: NoneKanaryDeploymentSpecValidationManualDeadineStatus,
						},
					},
				},
			},
		},
		{
			name: "one element not defaulted",
			list: &KanaryDeploymentSpecValidationList{
				Items: []KanaryDeploymentSpecValidation{{}},
			},
			want: &KanaryDeploymentSpecValidationList{
				ValidationPeriod: &metav1.Duration{
					Duration: 15 * time.Minute,
				},
				InitialDelay: &metav1.Duration{
					Duration: 0 * time.Minute,
				},
				MaxIntervalPeriod: &metav1.Duration{
					Duration: 20 * time.Second,
				},
				Items: []KanaryDeploymentSpecValidation{
					{
						Manual: &KanaryDeploymentSpecValidationManual{
							StatusAfterDealine: NoneKanaryDeploymentSpecValidationManualDeadineStatus,
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defaultKanaryDeploymentSpecValidationList(tt.list)
			if !reflect.DeepEqual(tt.list, tt.want) {
				t.Errorf("defaultKanaryDeploymentSpecValidationList() = %#v, want %#v", tt.list, tt.want)
			}
		})
	}
}
