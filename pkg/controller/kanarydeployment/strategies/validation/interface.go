package validation

import (
	"github.com/go-logr/logr"

	appsv1beta1 "k8s.io/api/apps/v1beta1"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	kanaryv1alpha1 "github.com/amadeusitgroup/kanary/pkg/apis/kanary/v1alpha1"
)

// Interface validation strategy interface
type Interface interface {
	Validation(kclient client.Client, reqLogger logr.Logger, kd *kanaryv1alpha1.KanaryDeployment, dep, canaryDep *appsv1beta1.Deployment) (*kanaryv1alpha1.KanaryDeploymentStatus, reconcile.Result, error)
	ValidationV2(kclient client.Client, reqLogger logr.Logger, kd *kanaryv1alpha1.KanaryDeployment, dep, canaryDep *appsv1beta1.Deployment) (*Result, error)
}

// ValidationResult returns result of a Validation
type Result struct {
	reconcile.Result
	IsFailed             bool
	NeedUpdateDeployment bool
	Comment              string
}

func (vr Result) IsDone() bool {
	if vr.Requeue || vr.RequeueAfter > 0 {
		return false
	}
	return true
}
