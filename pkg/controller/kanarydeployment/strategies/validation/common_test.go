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
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "before deadline",
			args: args{
				validationPeriod: 10 * time.Minute,
				startTime:        now.Add(-5 * time.Minute),
				now:              now,
			},
			want: false,
		},
		{
			name: "after deadline",
			args: args{
				validationPeriod: 10 * time.Minute,
				startTime:        now.Add(-15 * time.Minute),
				now:              now,
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, got := isDeadlinePeriodDone(tt.args.validationPeriod, tt.args.startTime, tt.args.now); got != tt.want {
				t.Errorf("isDeadlinePeriodDone() = %v, want %v", got, tt.want)
			}
		})
	}
}
