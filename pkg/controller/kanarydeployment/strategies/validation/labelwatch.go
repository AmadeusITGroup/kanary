package validation

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"

	appsv1beta1 "k8s.io/api/apps/v1beta1"
	corev1 "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	kanaryv1alpha1 "github.com/amadeusitgroup/kanary/pkg/apis/kanary/v1alpha1"
	"github.com/amadeusitgroup/kanary/pkg/controller/kanarydeployment/utils"
)

// NewLabelWatch returns new validation.LabelWatch instance
func NewLabelWatch(list *kanaryv1alpha1.KanaryDeploymentSpecValidationList, s *kanaryv1alpha1.KanaryDeploymentSpecValidation) Interface {
	return &labelWatchImpl{
		validationPeriod:  list.ValidationPeriod,
		maxIntervalPeriod: list.MaxIntervalPeriod,
		dryRun:            list.NoUpdate,
		config:            s.LabelWatch,
	}
}

type labelWatchImpl struct {
	validationPeriod  *metav1.Duration
	maxIntervalPeriod *metav1.Duration
	dryRun            bool
	config            *kanaryv1alpha1.KanaryDeploymentSpecValidationLabelWatch
}

func (l *labelWatchImpl) Validation(kclient client.Client, reqLogger logr.Logger, kd *kanaryv1alpha1.KanaryDeployment, dep, canaryDep *appsv1beta1.Deployment) (status *kanaryv1alpha1.KanaryDeploymentStatus, result reconcile.Result, err error) {
	status = kd.Status.DeepCopy()
	var needUpdateDeployment bool
	// By default a Deployement is valid until a Label is discovered on pod or deployment.
	validationStatus := true

	if l.config.DeploymentInvalidationLabels != nil {
		var selector labels.Selector
		selector, err = metav1.LabelSelectorAsSelector(l.config.DeploymentInvalidationLabels)
		if err != nil {
			// TODO improve error handling
			return
		}
		if selector.Matches(labels.Set(canaryDep.Labels)) {
			validationStatus = false
		}
	}

	// watch pods label
	if l.config.PodInvalidationLabels != nil {
		var selector labels.Selector
		selector, err = metav1.LabelSelectorAsSelector(l.config.PodInvalidationLabels)
		if err != nil {
			return status, result, fmt.Errorf("unable to create the label selector from PodInvalidationLabels: %v", err)
		}
		var pods []corev1.Pod
		pods, err = getPods(kclient, reqLogger, kd.Name, kd.Namespace)
		if err != nil {
			return status, result, fmt.Errorf("unable to list pods: %v", err)
		}
		for _, pod := range pods {
			if selector.Matches(labels.Set(pod.Labels)) {
				validationStatus = false
				break
			}
		}
	}

	var deadlineReached bool
	if canaryDep != nil {
		var requeueAfter time.Duration
		requeueAfter, deadlineReached = isDeadlinePeriodDone(l.validationPeriod.Duration, l.maxIntervalPeriod.Duration, canaryDep.CreationTimestamp.Time, time.Now())
		if !deadlineReached {
			result.RequeueAfter = requeueAfter
		}
		if deadlineReached && validationStatus {
			needUpdateDeployment = true
		}
	}

	if needUpdateDeployment && !l.dryRun {
		var newDep *appsv1beta1.Deployment
		newDep, err = utils.UpdateDeploymentWithKanaryDeploymentTemplate(kd, dep)
		if err != nil {
			reqLogger.Error(err, "failed to update the Deployment artifact", "Namespace", newDep.Namespace, "Deployment", newDep.Name)
			return status, result, err
		}
		err = kclient.Update(context.TODO(), newDep)
		if err != nil {
			reqLogger.Error(err, "failed to update the Deployment", "Namespace", newDep.Namespace, "Deployment", newDep.Name, "newDep", *newDep)
			return status, result, err
		}
	}

	if validationStatus && needUpdateDeployment {
		utils.UpdateKanaryDeploymentStatusCondition(status, metav1.Now(), kanaryv1alpha1.SucceededKanaryDeploymentConditionType, corev1.ConditionTrue, "Deployment updated successfully")
	}

	if !validationStatus {
		utils.UpdateKanaryDeploymentStatusCondition(status, metav1.Now(), kanaryv1alpha1.FailedKanaryDeploymentConditionType, corev1.ConditionTrue, "KanaryDeployment failed, labelWatch has detected invalidation labels")
	}
	return status, result, err
}

func (l *labelWatchImpl) ValidationV2(kclient client.Client, reqLogger logr.Logger, kd *kanaryv1alpha1.KanaryDeployment, dep, canaryDep *appsv1beta1.Deployment) (*Result, error) {
	var err error
	result := &Result{}
	// By default a Deployement is valid until a Label is discovered on pod or deployment.
	isSucceed := true

	if l.config.DeploymentInvalidationLabels != nil {
		var selector labels.Selector
		selector, err = metav1.LabelSelectorAsSelector(l.config.DeploymentInvalidationLabels)
		if err != nil {
			// TODO improve error handling
			return result, err
		}
		if selector.Matches(labels.Set(canaryDep.Labels)) {
			isSucceed = false
		}
	}

	// watch pods label
	if l.config.PodInvalidationLabels != nil {
		var selector labels.Selector
		selector, err = metav1.LabelSelectorAsSelector(l.config.PodInvalidationLabels)
		if err != nil {
			return result, fmt.Errorf("unable to create the label selector from PodInvalidationLabels: %v", err)
		}
		var pods []corev1.Pod
		pods, err = getPods(kclient, reqLogger, kd.Name, kd.Namespace)
		if err != nil {
			return result, fmt.Errorf("unable to list pods: %v", err)
		}
		for _, pod := range pods {
			if selector.Matches(labels.Set(pod.Labels)) {
				isSucceed = false
				break
			}
		}
	}

	var deadlineReached bool
	if canaryDep != nil {
		var requeueAfter time.Duration
		requeueAfter, deadlineReached = isDeadlinePeriodDone(l.validationPeriod.Duration, l.maxIntervalPeriod.Duration, canaryDep.CreationTimestamp.Time, time.Now())
		if !deadlineReached {
			result.RequeueAfter = requeueAfter
		}
		if deadlineReached && isSucceed {
			result.NeedUpdateDeployment = true
		}
	}

	if !isSucceed {
		result.IsFailed = true
		result.Comment = "labelWatch has detected invalidation labels"
	}

	return result, err
}
