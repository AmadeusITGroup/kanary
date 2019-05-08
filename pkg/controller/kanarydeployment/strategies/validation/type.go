package validation

import (
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// ValidationResult returns result of a Validation
type Result struct {
	reconcile.Result
	IsFailed             bool
	NeedUpdateDeployment bool
	Comment              string
}
