package strategies

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"k8s.io/api/core/v1"

	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	kanaryv1alpha1 "github.com/amadeusitgroup/kanary/pkg/apis/kanary/v1alpha1"
	"github.com/amadeusitgroup/kanary/pkg/controller/kanarydeployment/strategies/validation"
)

func Test_computeStatus(t *testing.T) {
	type args struct {
		results []*validation.Result
		status  *kanaryv1alpha1.KanaryDeploymentStatus
	}
	tests := []struct {
		name                 string
		args                 args
		wantStatusFunc       func(status *kanaryv1alpha1.KanaryDeploymentStatus) error
		wantResult           reconcile.Result
		wantFailed           bool
		wantDeploymentUpdate bool
	}{
		{
			name: "empty result",
			args: args{
				results: []*validation.Result{},
				status:  &kanaryv1alpha1.KanaryDeploymentStatus{},
			},
			wantResult:           reconcile.Result{},
			wantFailed:           false,
			wantDeploymentUpdate: true,
		},
		{
			name: "one success result",
			args: args{
				results: []*validation.Result{
					{
						IsFailed:             false,
						NeedUpdateDeployment: true,
					},
				},
				status: &kanaryv1alpha1.KanaryDeploymentStatus{},
			},
			wantResult:           reconcile.Result{},
			wantFailed:           false,
			wantDeploymentUpdate: true,
		},
		{
			name: "one success and requeue result",
			args: args{
				results: []*validation.Result{
					{
						IsFailed:             false,
						NeedUpdateDeployment: true,
						Result: reconcile.Result{
							Requeue: true,
						},
					},
				},
				status: &kanaryv1alpha1.KanaryDeploymentStatus{},
			},
			wantResult: reconcile.Result{
				Requeue: true,
			},
			wantFailed:           false,
			wantDeploymentUpdate: true,
		},
		{
			name: "one success and requeueAfter result",
			args: args{
				results: []*validation.Result{
					{
						IsFailed:             false,
						NeedUpdateDeployment: true,
						Result: reconcile.Result{
							RequeueAfter: 15 * time.Second,
						},
					},
				},
				status: &kanaryv1alpha1.KanaryDeploymentStatus{},
			},
			wantResult: reconcile.Result{
				RequeueAfter: 15 * time.Second,
			},
			wantFailed:           false,
			wantDeploymentUpdate: true,
		},
		{
			name: "two requeueAfter results",
			args: args{
				results: []*validation.Result{
					{
						IsFailed:             false,
						NeedUpdateDeployment: true,
						Result: reconcile.Result{
							RequeueAfter: 30 * time.Second,
						},
					},
					{
						IsFailed:             false,
						NeedUpdateDeployment: true,
						Result: reconcile.Result{
							RequeueAfter: 15 * time.Second,
						},
					},
				},
				status: &kanaryv1alpha1.KanaryDeploymentStatus{},
			},
			wantResult: reconcile.Result{
				RequeueAfter: 15 * time.Second,
			},
			wantFailed:           false,
			wantDeploymentUpdate: true,
		},
		{
			name: "one success and on failure result",
			args: args{
				results: []*validation.Result{
					{
						IsFailed:             false,
						NeedUpdateDeployment: true,
					},
					{
						IsFailed:             true,
						NeedUpdateDeployment: false,
					},
				},
				status: &kanaryv1alpha1.KanaryDeploymentStatus{},
			},
			wantStatusFunc: func(status *kanaryv1alpha1.KanaryDeploymentStatus) error {
				for _, condition := range status.Conditions {
					if condition.Type == kanaryv1alpha1.FailedKanaryDeploymentConditionType && condition.Status == v1.ConditionTrue {
						return nil
					}
				}
				return fmt.Errorf("unable to found condition failure")
			},
			wantResult:           reconcile.Result{},
			wantFailed:           true,
			wantDeploymentUpdate: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, got2, got3 := computeStatus(tt.args.results, tt.args.status)
			if tt.wantStatusFunc != nil {
				if err := tt.wantStatusFunc(got); err != nil {
					t.Errorf("computeStatus() StatusFunc error = %v", err)
				}
			}
			if !reflect.DeepEqual(got1, tt.wantResult) {
				t.Errorf("computeStatus() Result = %v, want %v", got1, tt.wantResult)
			}
			if got2 != tt.wantFailed {
				t.Errorf("computeStatus() success = %v, want %v", got2, tt.wantFailed)
			}
			if got3 != tt.wantDeploymentUpdate {
				t.Errorf("computeStatus() needDeploymentUpdate = %v, want %v", got3, tt.wantDeploymentUpdate)
			}
		})
	}
}
