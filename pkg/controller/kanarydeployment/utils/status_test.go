package utils

import (
	"reflect"
	"testing"

	kanaryv1alpha1 "github.com/amadeusitgroup/kanary/pkg/apis/kanary/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

func TestIsKanaryDeploymentFailed(t *testing.T) {
	type args struct {
		status *kanaryv1alpha1.KanaryDeploymentStatus
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "failed",
			args: args{
				status: &kanaryv1alpha1.KanaryDeploymentStatus{
					Conditions: []kanaryv1alpha1.KanaryDeploymentCondition{
						{
							Type:   kanaryv1alpha1.FailedKanaryDeploymentConditionType,
							Status: corev1.ConditionTrue,
						},
					},
				},
			},
			want: true,
		},
		{
			name: "not failed",
			args: args{
				status: &kanaryv1alpha1.KanaryDeploymentStatus{},
			},
			want: false,
		},
		{
			name: "not failed, conditionFalse",
			args: args{
				status: &kanaryv1alpha1.KanaryDeploymentStatus{
					Conditions: []kanaryv1alpha1.KanaryDeploymentCondition{
						{
							Type:   kanaryv1alpha1.FailedKanaryDeploymentConditionType,
							Status: corev1.ConditionFalse,
						},
					},
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsKanaryDeploymentFailed(tt.args.status); got != tt.want {
				t.Errorf("IsKanaryDeploymentFailed() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsKanaryDeploymentSucceeded(t *testing.T) {
	type args struct {
		status *kanaryv1alpha1.KanaryDeploymentStatus
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "succeed",
			args: args{
				status: &kanaryv1alpha1.KanaryDeploymentStatus{
					Conditions: []kanaryv1alpha1.KanaryDeploymentCondition{
						{
							Type:   kanaryv1alpha1.SucceededKanaryDeploymentConditionType,
							Status: corev1.ConditionTrue,
						},
					},
				},
			},
			want: true,
		},
		{
			name: "not succeed",
			args: args{
				status: &kanaryv1alpha1.KanaryDeploymentStatus{},
			},
			want: false,
		},
		{
			name: "not succeed, conditionFalse",
			args: args{
				status: &kanaryv1alpha1.KanaryDeploymentStatus{
					Conditions: []kanaryv1alpha1.KanaryDeploymentCondition{
						{
							Type:   kanaryv1alpha1.SucceededKanaryDeploymentConditionType,
							Status: corev1.ConditionFalse,
						},
					},
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsKanaryDeploymentSucceeded(tt.args.status); got != tt.want {
				t.Errorf("IsKanaryDeploymentSucceeded() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_updateStatusWithReport(t *testing.T) {
	type args struct {
		kd     *kanaryv1alpha1.KanaryDeployment
		status *kanaryv1alpha1.KanaryDeploymentStatus
	}
	tests := []struct {
		name string
		args args
		want *kanaryv1alpha1.KanaryDeploymentStatus
	}{
		{
			name: "default report",
			args: args{
				kd: &kanaryv1alpha1.KanaryDeployment{
					Spec: kanaryv1alpha1.KanaryDeploymentSpec{
						Traffic: kanaryv1alpha1.KanaryDeploymentSpecTraffic{},
					},
				},
				status: &kanaryv1alpha1.KanaryDeploymentStatus{
					Conditions: []kanaryv1alpha1.KanaryDeploymentCondition{
						kanaryv1alpha1.KanaryDeploymentCondition{
							Status: corev1.ConditionTrue,
							Type:   kanaryv1alpha1.ScheduledKanaryDeploymentConditionType,
						},
					},

					Report: kanaryv1alpha1.KanaryDeploymentStatusReport{},
				},
			},
			want: &kanaryv1alpha1.KanaryDeploymentStatus{
				Report: kanaryv1alpha1.KanaryDeploymentStatusReport{
					Status:     string(kanaryv1alpha1.ScheduledKanaryDeploymentConditionType),
					Scale:      "static",
					Validation: "unknow",
				},
			},
		},
		{
			name: "promQL validation",
			args: args{
				kd: &kanaryv1alpha1.KanaryDeployment{
					Spec: kanaryv1alpha1.KanaryDeploymentSpec{
						Traffic: kanaryv1alpha1.KanaryDeploymentSpecTraffic{
							Mirror: &kanaryv1alpha1.KanaryDeploymentSpecTrafficMirror{},
						},
						Validations: kanaryv1alpha1.KanaryDeploymentSpecValidationList{
							Items: []kanaryv1alpha1.KanaryDeploymentSpecValidation{
								{PromQL: &kanaryv1alpha1.KanaryDeploymentSpecValidationPromQL{}},
							},
						},
					},
				},
				status: &kanaryv1alpha1.KanaryDeploymentStatus{
					Conditions: []kanaryv1alpha1.KanaryDeploymentCondition{
						kanaryv1alpha1.KanaryDeploymentCondition{
							Status: corev1.ConditionTrue,
							Type:   kanaryv1alpha1.ScheduledKanaryDeploymentConditionType,
						},
						kanaryv1alpha1.KanaryDeploymentCondition{
							Status: corev1.ConditionTrue,
							Type:   kanaryv1alpha1.RunningKanaryDeploymentConditionType,
						},
					},
					Report: kanaryv1alpha1.KanaryDeploymentStatusReport{},
				},
			},
			want: &kanaryv1alpha1.KanaryDeploymentStatus{
				Report: kanaryv1alpha1.KanaryDeploymentStatusReport{
					Status:     string(kanaryv1alpha1.RunningKanaryDeploymentConditionType),
					Scale:      "static",
					Validation: "promQL",
				},
			},
		},
		{
			name: "labelWatch validation",
			args: args{
				kd: &kanaryv1alpha1.KanaryDeployment{
					Spec: kanaryv1alpha1.KanaryDeploymentSpec{
						Traffic: kanaryv1alpha1.KanaryDeploymentSpecTraffic{
							Mirror: &kanaryv1alpha1.KanaryDeploymentSpecTrafficMirror{},
						},
						Validations: kanaryv1alpha1.KanaryDeploymentSpecValidationList{
							Items: []kanaryv1alpha1.KanaryDeploymentSpecValidation{
								{LabelWatch: &kanaryv1alpha1.KanaryDeploymentSpecValidationLabelWatch{}},
							},
						},
					},
				},
				status: &kanaryv1alpha1.KanaryDeploymentStatus{
					Conditions: []kanaryv1alpha1.KanaryDeploymentCondition{
						kanaryv1alpha1.KanaryDeploymentCondition{
							Status: corev1.ConditionTrue,
							Type:   kanaryv1alpha1.ScheduledKanaryDeploymentConditionType,
						},
						kanaryv1alpha1.KanaryDeploymentCondition{
							Status: corev1.ConditionTrue,
							Type:   kanaryv1alpha1.RunningKanaryDeploymentConditionType,
						},
						kanaryv1alpha1.KanaryDeploymentCondition{
							Status: corev1.ConditionTrue,
							Type:   kanaryv1alpha1.FailedKanaryDeploymentConditionType,
						},
					},
					Report: kanaryv1alpha1.KanaryDeploymentStatusReport{},
				},
			},
			want: &kanaryv1alpha1.KanaryDeploymentStatus{
				Report: kanaryv1alpha1.KanaryDeploymentStatusReport{
					Status:     string(kanaryv1alpha1.FailedKanaryDeploymentConditionType),
					Scale:      "static",
					Validation: "labelWatch",
				},
			},
		},
		{
			name: "manual validation",
			args: args{
				kd: &kanaryv1alpha1.KanaryDeployment{
					Spec: kanaryv1alpha1.KanaryDeploymentSpec{
						Traffic: kanaryv1alpha1.KanaryDeploymentSpecTraffic{
							Mirror: &kanaryv1alpha1.KanaryDeploymentSpecTrafficMirror{},
						},
						Validations: kanaryv1alpha1.KanaryDeploymentSpecValidationList{
							Items: []kanaryv1alpha1.KanaryDeploymentSpecValidation{
								{Manual: &kanaryv1alpha1.KanaryDeploymentSpecValidationManual{}},
							},
						},
					},
				},
				status: &kanaryv1alpha1.KanaryDeploymentStatus{
					Conditions: []kanaryv1alpha1.KanaryDeploymentCondition{
						kanaryv1alpha1.KanaryDeploymentCondition{
							Status: corev1.ConditionTrue,
							Type:   kanaryv1alpha1.ScheduledKanaryDeploymentConditionType,
						},
						kanaryv1alpha1.KanaryDeploymentCondition{
							Status: corev1.ConditionTrue,
							Type:   kanaryv1alpha1.RunningKanaryDeploymentConditionType,
						},
						kanaryv1alpha1.KanaryDeploymentCondition{
							Status: corev1.ConditionTrue,
							Type:   kanaryv1alpha1.SucceededKanaryDeploymentConditionType,
						},
					},
					Report: kanaryv1alpha1.KanaryDeploymentStatusReport{},
				},
			},
			want: &kanaryv1alpha1.KanaryDeploymentStatus{
				Report: kanaryv1alpha1.KanaryDeploymentStatusReport{
					Status:     string(kanaryv1alpha1.SucceededKanaryDeploymentConditionType),
					Scale:      "static",
					Validation: "manual",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if updateStatusReport(tt.args.kd, tt.args.status); !reflect.DeepEqual(tt.args.status.Report, tt.want.Report) {
				t.Errorf("updateStatusWithReport() = %v, want %v", tt.args.status, tt.want)
			}
		})
	}
}
