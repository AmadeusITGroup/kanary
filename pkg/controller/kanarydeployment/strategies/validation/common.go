package validation

import (
	"time"
)

func isDeadlinePeriodDone(validationPeriod time.Duration, startTime, now time.Time) (time.Duration, bool) {
	if startTime.Add(validationPeriod).Before(now) {
		return time.Duration(0), true
	}

	return startTime.Add(validationPeriod).Sub(now), false
}
