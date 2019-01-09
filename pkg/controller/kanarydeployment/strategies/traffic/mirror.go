package traffic

import (
	"github.com/go-logr/logr"

	appsv1beta1 "k8s.io/api/apps/v1beta1"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	kanaryv1alpha1 "github.com/amadeusitgroup/kanary/pkg/apis/kanary/v1alpha1"
)

// NewMirror returns new traffic.Live instance
func NewMirror(s *kanaryv1alpha1.KanaryDeploymentSpecTraffic) Interface {
	return &mirrorImpl{
		conf: s.Shadow,
	}
}

type mirrorImpl struct {
	conf *kanaryv1alpha1.KanaryDeploymentSpecTrafficShadow
}

func (s *mirrorImpl) Traffic(kclient client.Client, reqLogger logr.Logger, kd *kanaryv1alpha1.KanaryDeployment, canaryDep *appsv1beta1.Deployment) (status *kanaryv1alpha1.KanaryDeploymentStatus, result reconcile.Result, err error) {
	return
}

func (s *mirrorImpl) Cleanup(kclient client.Client, reqLogger logr.Logger, kd *kanaryv1alpha1.KanaryDeployment) (status *kanaryv1alpha1.KanaryDeploymentStatus, result reconcile.Result, err error) {
	return
}
