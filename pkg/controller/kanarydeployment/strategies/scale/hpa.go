package scale

import (
	"context"

	"github.com/go-logr/logr"

	appsv1beta1 "k8s.io/api/apps/v1beta1"
	"k8s.io/api/autoscaling/v2beta1"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	kanaryv1alpha1 "github.com/amadeusitgroup/kanary/pkg/apis/kanary/v1alpha1"
	"github.com/amadeusitgroup/kanary/pkg/controller/kanarydeployment/utils"
)

// NewHPA returns new scale.HPA instance
func NewHPA(s *kanaryv1alpha1.HorizontalPodAutoscalerSpec) Interface {
	return &hpaImpl{}
}

type hpaImpl struct {
}

func (h *hpaImpl) Scale(kclient client.Client, reqLogger logr.Logger, kd *kanaryv1alpha1.KanaryDeployment, canaryDep *appsv1beta1.Deployment) (*kanaryv1alpha1.KanaryDeploymentStatus, reconcile.Result, error) {
	status := &kd.Status
	// don't update the canary deployment replicas if the KanaryDeployment has failed
	if utils.IsKanaryDeploymentFailed(status) {
		return status, reconcile.Result{}, nil
	}

	// check if the HPA is already created, if not create it.
	hpa := &v2beta1.HorizontalPodAutoscaler{}
	objKey := types.NamespacedName{Name: utils.GetCanaryDeploymentName(kd), Namespace: kd.Namespace}
	err := kclient.Get(context.TODO(), objKey, hpa)
	var requeue bool
	if err != nil && errors.IsNotFound(err) {
		hpa = &v2beta1.HorizontalPodAutoscaler{
			ObjectMeta: metav1.ObjectMeta{
				Name:      objKey.Name,
				Namespace: objKey.Namespace,
				Labels: map[string]string{
					kanaryv1alpha1.KanaryDeploymentKanaryNameLabelKey: kd.Name,
				},
			},
			Spec: v2beta1.HorizontalPodAutoscalerSpec{
				MinReplicas: kd.Spec.Scale.HPA.MinReplicas,
				MaxReplicas: kd.Spec.Scale.HPA.MaxReplicas,
				Metrics:     kd.Spec.Scale.HPA.Metrics,
				ScaleTargetRef: v2beta1.CrossVersionObjectReference{
					APIVersion: appsv1beta1.SchemeGroupVersion.String(),
					Kind:       "Deployment",
					Name:       utils.GetCanaryDeploymentName(kd),
				},
			},
		}
		if err = kclient.Create(context.TODO(), hpa); err != nil {
			reqLogger.Error(err, "failed to create new HorizontalPodAutoscaler")
		}
		requeue = true

	} else if err != nil {
		reqLogger.Error(err, "failed to get HorizontalPodAutoscaler")
		requeue = true
	}

	return status, reconcile.Result{Requeue: requeue}, nil
}

func (h *hpaImpl) Clear(kclient client.Client, reqLogger logr.Logger, kd *kanaryv1alpha1.KanaryDeployment, canaryDep *appsv1beta1.Deployment) (*kanaryv1alpha1.KanaryDeploymentStatus, reconcile.Result, error) {
	status := &kd.Status

	// check if the HPA is already created, if not create it.
	hpa := &v2beta1.HorizontalPodAutoscaler{}
	objKey := types.NamespacedName{Name: utils.GetCanaryDeploymentName(kd), Namespace: kd.Namespace}
	err := kclient.Get(context.TODO(), objKey, hpa)
	if err != nil && errors.IsNotFound(err) {
		return status, reconcile.Result{}, nil
	} else if err != nil {
		reqLogger.Error(err, "failed to get HorizontalPodAutoscaler")
		return status, reconcile.Result{Requeue: true}, err
	}

	// hpa is present, needs to delete it
	hpa = &v2beta1.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      utils.GetCanaryDeploymentName(kd),
			Namespace: objKey.Namespace,
		},
	}
	err = kclient.Delete(context.TODO(), hpa)
	return status, reconcile.Result{Requeue: true}, err
}
