package validation

import (
	"testing"
	"time"
)

func Test_isDeadlinePeriodDone(t *testing.T) {
	now := time.Now()
	type args struct {
		validationPeriod time.Duration
		startTime        time.Time
		now              time.Time
		maxDuration      time.Duration
	}
	tests := []struct {
		name         string
		args         args
		want         bool
		wantDuration time.Duration
	}{
		{
			name: "before deadline, after maxQueueDuration",
			args: args{
				validationPeriod: 10 * time.Minute,
				startTime:        now.Add(-5 * time.Minute),
				now:              now,
				maxDuration:      15 * time.Second,
			},
			want:         false,
			wantDuration: 15 * time.Second,
		},
		{
			name: "before deadline",
			args: args{
				validationPeriod: 10 * time.Minute,
				startTime:        now.Add(-5 * time.Minute),
				now:              now,
				maxDuration:      15 * time.Minute,
			},
			want:         false,
			wantDuration: 5 * time.Minute,
		},
		{
			name: "after deadline",
			args: args{
				validationPeriod: 10 * time.Minute,
				startTime:        now.Add(-15 * time.Minute),
				now:              now,
				maxDuration:      15 * time.Second,
			},
			want:         true,
			wantDuration: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotDuration, got := isDeadlinePeriodDone(tt.args.validationPeriod, tt.args.maxDuration, tt.args.startTime, tt.args.now)
			if got != tt.want {
				t.Errorf("isDeadlinePeriodDone().bool = %v, want %v", got, tt.want)
			}
			if gotDuration != tt.wantDuration {
				t.Errorf("isDeadlinePeriodDone().duration = %v, want %v", gotDuration, tt.wantDuration)
			}
		})
	}
}
