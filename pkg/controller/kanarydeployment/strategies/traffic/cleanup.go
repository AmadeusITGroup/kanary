package traffic

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"

	appsv1beta1 "k8s.io/api/apps/v1beta1"
	corev1 "k8s.io/api/core/v1"

	"k8s.io/apimachinery/pkg/labels"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	kanaryv1alpha1 "github.com/amadeusitgroup/kanary/pkg/apis/kanary/v1alpha1"
)

// NewCleanup returns new traffic.Traffic instance
// Aim of the cleanup is to remove created resources if they are not needed anymore
// according to the KanaryDeploymentSpec.
func NewCleanup(s *kanaryv1alpha1.KanaryDeploymentSpecTraffic) Interface {
	return &cleanupImpl{
		conf: s,
	}
}

type cleanupImpl struct {
	conf *kanaryv1alpha1.KanaryDeploymentSpecTraffic
}

func (t *cleanupImpl) Traffic(kclient client.Client, reqLogger logr.Logger, kd *kanaryv1alpha1.KanaryDeployment, canaryDep *appsv1beta1.Deployment) (status *kanaryv1alpha1.KanaryDeploymentStatus, result reconcile.Result, err error) {
	if t.conf.Source == kanaryv1alpha1.ShadowKanaryDeploymentSpecTrafficSource || t.conf.Source == kanaryv1alpha1.NoneKanaryDeploymentSpecTrafficSource {
		var needsReturn bool
		needsReturn, result, err = t.clearServices(kclient, reqLogger, kd)
		if needsReturn {
			result.Requeue = true
		}
	}

	if t.conf.Source != kanaryv1alpha1.ShadowKanaryDeploymentSpecTrafficSource || t.conf.Source == kanaryv1alpha1.NoneKanaryDeploymentSpecTrafficSource {
		// TODO implement this method when traffic strategy shadow is implemented
	}
	return &kd.Status, result, err
}

func (t *cleanupImpl) clearServices(kclient client.Client, reqLogger logr.Logger, kd *kanaryv1alpha1.KanaryDeployment) (needsReturn bool, result reconcile.Result, err error) {
	services := &corev1.ServiceList{}

	selector := labels.Set{
		kanaryv1alpha1.KanaryDeploymentKanaryNameLabelKey: kd.Name,
	}

	listOptions := &client.ListOptions{
		LabelSelector: selector.AsSelector(),
		Namespace:     kd.Namespace,
	}
	// use List instead of Get because the spec.ServiceName can empty after a spec configuration change
	err = kclient.List(context.TODO(), listOptions, services)
	if err != nil {
		reqLogger.Error(err, "failed to list Service")
		return true, reconcile.Result{Requeue: true}, err
	}

	var errs []error
	if len(services.Items) > 0 {
		reqLogger.Info(fmt.Sprintf("nbItem: %d", len(services.Items)))
		for _, service := range services.Items {
			err = kclient.Delete(context.TODO(), &service)
			if err != nil {
				reqLogger.Error(err, "unable to delete the kanary service")
				errs = append(errs, err)
			}
			needsReturn = true
			result.Requeue = true
		}
	}
	if errs != nil {
		err = utilerrors.NewAggregate(errs)
	}

	return needsReturn, result, err
}
