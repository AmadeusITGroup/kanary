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
					Validation: KanaryDeploymentSpecValidation{
						ValidationPeriod: &metav1.Duration{
							Duration: 15 * time.Minute,
						},
						InitialDelay: &metav1.Duration{
							Duration: 5 * time.Minute,
						},
						Manual: &KanaryDeploymentSpecValidationManual{
							StatusAfterDealine: NoneKanaryDeploymentSpecValidationManualDeadineStatus,
						},
					},
				},
			},
			want: true,
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
					Validation: KanaryDeploymentSpecValidation{
						ValidationPeriod: &metav1.Duration{
							Duration: 15 * time.Minute,
						},
						InitialDelay: &metav1.Duration{
							Duration: 0 * time.Minute,
						},
						Manual: &KanaryDeploymentSpecValidationManual{
							StatusAfterDealine: NoneKanaryDeploymentSpecValidationManualDeadineStatus,
						},
					},
				},
			},
		},

		{
			name: "already some configuratin",
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
					Validation: KanaryDeploymentSpecValidation{
						ValidationPeriod: &metav1.Duration{
							Duration: 30 * time.Minute,
						},
						InitialDelay: &metav1.Duration{
							Duration: 5 * time.Minute,
						},
						PromQL: &KanaryDeploymentSpecValidationPromQL{
							ServerURL: "prometheus",
							Query:     "foo",
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
					Validation: KanaryDeploymentSpecValidation{
						ValidationPeriod: &metav1.Duration{
							Duration: 30 * time.Minute,
						},
						InitialDelay: &metav1.Duration{
							Duration: 5 * time.Minute,
						},
						PromQL: &KanaryDeploymentSpecValidationPromQL{
							ServerURL: "prometheus",
							Query:     "foo",
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
