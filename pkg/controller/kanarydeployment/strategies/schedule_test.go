package strategies

import (
	"reflect"
	"testing"
	"time"

	kanaryv1alpha1 "github.com/amadeusitgroup/kanary/pkg/apis/kanary/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

func Test_getScheduleTime(t *testing.T) {

	future := time.Now().Add(10 * time.Minute).Round(time.Second)
	past := time.Now().Add(-10 * time.Minute).Round(time.Second)
	nearpast := time.Now().Add(-10 * time.Second).Round(time.Second)

	tests := []struct {
		name        string
		strSchedule string
		want        time.Time
		wantErr     bool
	}{
		{
			name:        "future",
			strSchedule: future.Format(time.RFC3339),
			want:        future,
			wantErr:     false,
		},
		{
			name:        "past",
			strSchedule: past.Format(time.RFC3339),
			want:        time.Time{},
			wantErr:     true,
		},
		{
			name:        "nearpast",
			strSchedule: nearpast.Format(time.RFC3339),
			want:        nearpast,
			wantErr:     false,
		},
		{
			name:        "formatErr",
			strSchedule: "Not a time in good format",
			want:        time.Time{},
			wantErr:     true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getScheduleTime(tt.strSchedule)
			if (err != nil) != tt.wantErr {
				t.Errorf("getScheduleTime() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got.Unix(), tt.want.Unix()) { // using Unix to not be affected by Locale settings
				t.Errorf("getScheduleTime() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_applyScheduling(t *testing.T) {
	inFuture := 10 * time.Minute
	future := time.Now().Add(inFuture).Round(time.Second)
	logger := logf.ZapLogger(true)
	tests := []struct {
		name       string
		kd         *kanaryv1alpha1.KanaryDeployment
		wantStatus *kanaryv1alpha1.KanaryDeploymentStatus
		wantResult *reconcile.Result
	}{
		{
			name: "no scheduled date - already scheduled",
			kd: &kanaryv1alpha1.KanaryDeployment{
				Status: kanaryv1alpha1.KanaryDeploymentStatus{
					Conditions: []kanaryv1alpha1.KanaryDeploymentCondition{
						kanaryv1alpha1.KanaryDeploymentCondition{
							Status: corev1.ConditionTrue,
							Type:   kanaryv1alpha1.ScheduledKanaryDeploymentConditionType,
						},
					},
				},
			},
			wantStatus: nil,
			wantResult: nil,
		},
		{
			name: "bad scheduled date - already scheduled - theoric case",
			kd: &kanaryv1alpha1.KanaryDeployment{
				Spec: kanaryv1alpha1.KanaryDeploymentSpec{
					Schedule: "bad Format",
				},

				Status: kanaryv1alpha1.KanaryDeploymentStatus{
					Conditions: []kanaryv1alpha1.KanaryDeploymentCondition{
						kanaryv1alpha1.KanaryDeploymentCondition{
							Status: corev1.ConditionTrue,
							Type:   kanaryv1alpha1.ScheduledKanaryDeploymentConditionType,
						},
					},
				},
			},
			wantStatus: &kanaryv1alpha1.KanaryDeploymentStatus{
				Conditions: []kanaryv1alpha1.KanaryDeploymentCondition{
					kanaryv1alpha1.KanaryDeploymentCondition{
						Status:  corev1.ConditionFalse,
						Type:    kanaryv1alpha1.ScheduledKanaryDeploymentConditionType,
						Message: "can't parse Schedule field: parsing time \"bad Format\" as \"2006-01-02T15:04:05Z07:00\": cannot parse \"bad Format\" as \"2006\"",
					},
				},
			},
			wantResult: &reconcile.Result{},
		},
		{
			name: "scheduled date future - already scheduled",
			kd: &kanaryv1alpha1.KanaryDeployment{
				Spec: kanaryv1alpha1.KanaryDeploymentSpec{
					Schedule: future.Format(time.RFC3339),
				},

				Status: kanaryv1alpha1.KanaryDeploymentStatus{
					Conditions: []kanaryv1alpha1.KanaryDeploymentCondition{
						kanaryv1alpha1.KanaryDeploymentCondition{
							Status: corev1.ConditionTrue,
							Type:   kanaryv1alpha1.ScheduledKanaryDeploymentConditionType,
						},
					},
				},
			},
			wantStatus: &kanaryv1alpha1.KanaryDeploymentStatus{
				Conditions: []kanaryv1alpha1.KanaryDeploymentCondition{
					kanaryv1alpha1.KanaryDeploymentCondition{
						Status:  corev1.ConditionTrue,
						Type:    kanaryv1alpha1.ScheduledKanaryDeploymentConditionType,
						Message: future.Format(time.RFC3339),
					},
				},
			},
			wantResult: &reconcile.Result{RequeueAfter: inFuture},
		},

		{
			name: "schedule on the fly - not yet scheduled",
			kd:   &kanaryv1alpha1.KanaryDeployment{},

			wantStatus: &kanaryv1alpha1.KanaryDeploymentStatus{
				Conditions: []kanaryv1alpha1.KanaryDeploymentCondition{
					kanaryv1alpha1.KanaryDeploymentCondition{
						Status:  corev1.ConditionTrue,
						Type:    kanaryv1alpha1.ScheduledKanaryDeploymentConditionType,
						Message: schedOnTheFlyMessage,
					},
				},
			},
			wantResult: &reconcile.Result{Requeue: true},
		},
		{
			name: "error in str Schedule - not yet scheduled",
			kd: &kanaryv1alpha1.KanaryDeployment{
				Spec: kanaryv1alpha1.KanaryDeploymentSpec{
					Schedule: "bad Format",
				},
			},

			wantStatus: &kanaryv1alpha1.KanaryDeploymentStatus{
				Conditions: []kanaryv1alpha1.KanaryDeploymentCondition{
					kanaryv1alpha1.KanaryDeploymentCondition{
						Status:  corev1.ConditionFalse,
						Type:    kanaryv1alpha1.ScheduledKanaryDeploymentConditionType,
						Message: "can't parse Schedule field: parsing time \"bad Format\" as \"2006-01-02T15:04:05Z07:00\": cannot parse \"bad Format\" as \"2006\"",
					},
				},
			},
			wantResult: &reconcile.Result{},
		},
		{
			name: "schedule in future - not yet scheduled",
			kd: &kanaryv1alpha1.KanaryDeployment{
				Spec: kanaryv1alpha1.KanaryDeploymentSpec{
					Schedule: future.Format(time.RFC3339),
				},
			},

			wantStatus: &kanaryv1alpha1.KanaryDeploymentStatus{
				Conditions: []kanaryv1alpha1.KanaryDeploymentCondition{
					kanaryv1alpha1.KanaryDeploymentCondition{
						Status:  corev1.ConditionTrue,
						Type:    kanaryv1alpha1.ScheduledKanaryDeploymentConditionType,
						Message: future.Format(time.RFC3339),
					},
				},
			},
			wantResult: &reconcile.Result{RequeueAfter: inFuture},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := ApplyScheduling(logger, tt.kd)
			if got != nil {
				clearConditionTimes(got.Conditions)
			}

			if !reflect.DeepEqual(got, tt.wantStatus) {
				t.Errorf("applyScheduling() got = %v, want %v", got, tt.wantStatus)
			}

			if got1 != nil {
				got1.RequeueAfter = got1.RequeueAfter.Round(time.Minute)
			}

			if !reflect.DeepEqual(got1, tt.wantResult) {
				t.Errorf("applyScheduling() got1 = %v, want %v", got1, tt.wantResult)
			}
		})
	}
}

func clearConditionTimes(conditions []kanaryv1alpha1.KanaryDeploymentCondition) {
	for i, c := range conditions {
		c.LastTransitionTime = metav1.Time{}
		c.LastUpdateTime = metav1.Time{}
		conditions[i] = c
	}
}
