package validation

import (
	"time"

	"github.com/amadeusitgroup/kanary/pkg/apis/kanary/v1alpha1"
)

func isDeadlinePeriodDone(validationPeriod time.Duration, startTime, now time.Time) (time.Duration, bool) {
	if startTime.Add(validationPeriod).Before(now) {
		return time.Duration(0), true
	}

	return startTime.Add(validationPeriod).Sub(now), false
}

// IsValidationDelayPeriodDone returns true if the InitialDelay validation periode is over.
func IsValidationDelayPeriodDone(kd *v1alpha1.KanaryDeployment) bool {
	now := time.Now()
	_, done := isDeadlinePeriodDone(kd.Spec.Validation.InitialDelay.Duration, kd.CreationTimestamp.Time, now)
	return done
}
