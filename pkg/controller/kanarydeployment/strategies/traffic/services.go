package traffic

import (
	"context"
	"time"

	"github.com/go-logr/logr"

	appsv1beta1 "k8s.io/api/apps/v1beta1"
	corev1 "k8s.io/api/core/v1"

	apiequality "k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	kanaryv1alpha1 "github.com/amadeusitgroup/kanary/pkg/apis/kanary/v1alpha1"
	"github.com/amadeusitgroup/kanary/pkg/controller/kanarydeployment/utils"
)

// NewKanaryService returns new traffic.KanaryService instance
func NewKanaryService(s *kanaryv1alpha1.KanaryDeploymentSpecTraffic) Interface {
	return &kanaryServiceImpl{
		conf: s,
	}
}

type kanaryServiceImpl struct {
	conf *kanaryv1alpha1.KanaryDeploymentSpecTraffic
}

func (k *kanaryServiceImpl) Traffic(kclient client.Client, reqLogger logr.Logger, kd *kanaryv1alpha1.KanaryDeployment, canaryDep *appsv1beta1.Deployment) (*kanaryv1alpha1.KanaryDeploymentStatus, reconcile.Result, error) {
	// Retrieve and create service if defined
	_, needsRequeue, result, err := k.manageServices(kclient, reqLogger, kd)
	status := kd.Status.DeepCopy()
	utils.UpdateKanaryDeploymentStatusConditionsFailure(status, metav1.Now(), err)
	if needsRequeue {
		result.Requeue = true
	}
	return status, result, err
}

func (k *kanaryServiceImpl) manageServices(kclient client.Client, reqLogger logr.Logger, kd *kanaryv1alpha1.KanaryDeployment) (*corev1.Service, bool, reconcile.Result, error) {
	var service *corev1.Service
	var err error

	if kd.Spec.ServiceName != "" {
		service = &corev1.Service{}
		err = kclient.Get(context.TODO(), types.NamespacedName{Name: kd.Spec.ServiceName, Namespace: kd.Namespace}, service)
		if err != nil && errors.IsNotFound(err) {
			// TODO update status, to say that the service didn't exist
			return service, true, reconcile.Result{Requeue: true, RequeueAfter: time.Second}, err
		} else if err != nil {
			reqLogger.Error(err, "failed to get Deployment")
			return service, true, reconcile.Result{}, err
		}
	}

	if service != nil {
		switch k.conf.Source {
		case kanaryv1alpha1.BothKanaryDeploymentSpecTrafficSource, kanaryv1alpha1.KanaryServiceKanaryDeploymentSpecTrafficSource:
			kanaryService := utils.NewCanaryServiceForKanaryDeployment(kd, service, NewOverwriteSelector(kd))
			currentKanaryService := &corev1.Service{}
			err = kclient.Get(context.TODO(), types.NamespacedName{Name: kanaryService.Name, Namespace: kanaryService.Namespace}, currentKanaryService)
			if err != nil && errors.IsNotFound(err) {
				reqLogger.Info("Creating a new Canary Service", "Namespace", kanaryService.Namespace, "Service.Name", kanaryService.Name)
				err = kclient.Create(context.TODO(), kanaryService)
				if err != nil {
					reqLogger.Error(err, "failed to create new CanaryService", "Namespace", kanaryService.Namespace, "Service.Name", kanaryService.Name)
					return service, true, reconcile.Result{}, err
				}
				// Deployment created successfully - return and requeue
				return service, true, reconcile.Result{Requeue: true}, nil
			} else if err != nil {
				reqLogger.Error(err, "failed to get Service")
				return service, true, reconcile.Result{}, err
			} else {
				compareKanaryServiceSpec := kanaryService.Spec.DeepCopy()
				compareCurrentServiceSpec := currentKanaryService.Spec.DeepCopy()
				{
					// remove potential values updated in service.Spec
					compareCurrentServiceSpec.ClusterIP = ""
					compareCurrentServiceSpec.LoadBalancerIP = ""
				}
				if !apiequality.Semantic.DeepEqual(compareKanaryServiceSpec, compareCurrentServiceSpec) {
					updatedService := currentKanaryService.DeepCopy()
					updatedService.Spec = *compareKanaryServiceSpec
					updatedService.Spec.ClusterIP = currentKanaryService.Spec.ClusterIP
					updatedService.Spec.LoadBalancerIP = currentKanaryService.Spec.LoadBalancerIP
					err = kclient.Update(context.TODO(), updatedService)
					if err != nil {
						reqLogger.Error(err, "unable to update the kanary service")
						return service, true, reconcile.Result{}, err
					}
				}
			}
		}
	}
	return service, false, reconcile.Result{}, err
}

// NewOverwriteSelector used to know if the Deployment.Spec.Selector needs to be overwrited for the kanary deployment
func NewOverwriteSelector(kd *kanaryv1alpha1.KanaryDeployment) bool {
	// if we dont want that
	switch kd.Spec.Traffic.Source {
	case kanaryv1alpha1.ServiceKanaryDeploymentSpecTrafficSource, kanaryv1alpha1.BothKanaryDeploymentSpecTrafficSource:
		return true
	default:
		return false
	}
}
