package strategies

import (
	"testing"

	"github.com/amadeusitgroup/kanary/pkg/controller/kanarydeployment/strategies/validation"
)

func Test_computeStatus(t *testing.T) {
	type args struct {
		results []*validation.Result
	}
	tests := []struct {
		name               string
		args               args
		wantFailureMessage string
		wantForceSuccess   bool
	}{
		{
			name: "empty result",
			args: args{
				results: []*validation.Result{},
			},
			wantFailureMessage: "",
			wantForceSuccess:   false,
		},
		{
			name: "one success result",
			args: args{
				results: []*validation.Result{
					{
						IsFailed: false,
					},
				},
			},
			wantFailureMessage: "",
			wantForceSuccess:   false,
		},
		{
			name: "one success and requeue result",
			args: args{
				results: []*validation.Result{
					{
						IsFailed: false,
					},
				},
			},
			wantFailureMessage: "",
			wantForceSuccess:   false,
		},
		{
			name: "one success and requeueAfter result",
			args: args{
				results: []*validation.Result{
					{
						IsFailed: false,
					},
				},
			},
			wantFailureMessage: "",
			wantForceSuccess:   false,
		},
		{
			name: "two requeueAfter results",
			args: args{
				results: []*validation.Result{
					{
						IsFailed: false,
					},
					{
						IsFailed: false,
					},
				},
			},
			wantFailureMessage: "",
			wantForceSuccess:   false,
		},
		{
			name: "one success and on failure result",
			args: args{
				results: []*validation.Result{
					{
						IsFailed: false,
					},
					{
						IsFailed: true,
					},
				},
			},
			wantFailureMessage: unknownFailureReason,
			wantForceSuccess:   false,
		},
		{
			name: "one success and on failure result and one force",
			args: args{
				results: []*validation.Result{
					{
						IsFailed: false,
					},
					{
						IsFailed: true,
					},
					{
						IsFailed:        false,
						ForceSuccessNow: true,
					},
				},
			},
			wantFailureMessage: unknownFailureReason,
			wantForceSuccess:   false,
		},
		{
			name: "one force",
			args: args{
				results: []*validation.Result{
					{
						IsFailed:        false,
						ForceSuccessNow: true,
					},
				},
			},
			wantFailureMessage: "",
			wantForceSuccess:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotFailMessage, gotForceSuccessNow := computeStatus(tt.args.results)
			if gotFailMessage != tt.wantFailureMessage {
				t.Errorf("computeStatus() success = %v, want %v", gotFailMessage, tt.wantFailureMessage)
			}
			if gotForceSuccessNow != tt.wantForceSuccess {
				t.Errorf("computeStatus() needDeploymentUpdate = %v, want %v", gotForceSuccessNow, tt.wantForceSuccess)
			}
		})
	}
}
