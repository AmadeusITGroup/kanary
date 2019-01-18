package validation

import (
	"context"
	"time"

	"github.com/go-logr/logr"

	appsv1beta1 "k8s.io/api/apps/v1beta1"
	corev1 "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	kanaryv1alpha1 "github.com/amadeusitgroup/kanary/pkg/apis/kanary/v1alpha1"
	"github.com/amadeusitgroup/kanary/pkg/controller/kanarydeployment/utils"
)

// NewManual returns new validation.Manual instance
func NewManual(s *kanaryv1alpha1.KanaryDeploymentSpecValidation) Interface {
	return &manualImpl{
		deadlineStatus:         s.Manual.StatusAfterDealine,
		validationManualStatus: s.Manual.Status,
		validationPeriod:       s.ValidationPeriod.Duration,
		dryRun:                 s.NoUpdate,
	}
}

type manualImpl struct {
	deadlineStatus         kanaryv1alpha1.KanaryDeploymentSpecValidationManualDeadineStatus
	validationManualStatus kanaryv1alpha1.KanaryDeploymentSpecValidationManualStatus
	validationPeriod       time.Duration
	dryRun                 bool
}

func (m *manualImpl) Validation(kclient client.Client, reqLogger logr.Logger, kd *kanaryv1alpha1.KanaryDeployment, dep, canaryDep *appsv1beta1.Deployment) (status *kanaryv1alpha1.KanaryDeploymentStatus, result reconcile.Result, err error) {
	status = kd.Status.DeepCopy()
	var needUpdateDeployment bool
	if m.validationManualStatus == kanaryv1alpha1.ValidKanaryDeploymentSpecValidationManualStatus {
		needUpdateDeployment = true
	}

	var deadlineReached bool
	if canaryDep != nil {
		var requeueAfter time.Duration
		requeueAfter, deadlineReached = isDeadlinePeriodDone(m.validationPeriod, canaryDep.CreationTimestamp.Time, time.Now())
		if !deadlineReached {
			result.RequeueAfter = requeueAfter
		}
		if deadlineReached && m.deadlineStatus == kanaryv1alpha1.ValidKanaryDeploymentSpecValidationManualDeadineStatus {
			needUpdateDeployment = true
		}
	}

	if needUpdateDeployment && !m.dryRun {
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

	if m.validationManualStatus == kanaryv1alpha1.ValidKanaryDeploymentSpecValidationManualStatus {
		utils.UpdateKanaryDeploymentStatusCondition(status, metav1.Now(), kanaryv1alpha1.SucceededKanaryDeploymentConditionType, corev1.ConditionTrue, "Deployment updated successfully")
	} else if m.validationManualStatus == kanaryv1alpha1.InvalidKanaryDeploymentSpecValidationManualStatus {
		utils.UpdateKanaryDeploymentStatusCondition(status, metav1.Now(), kanaryv1alpha1.FailedKanaryDeploymentConditionType, corev1.ConditionTrue, "KanaryDeployment validation failed, manual.status=invalid")
	} else if deadlineReached && m.deadlineStatus == kanaryv1alpha1.InvalidKanaryDeploymentSpecValidationManualDeadineStatus {
		utils.UpdateKanaryDeploymentStatusCondition(status, metav1.Now(), kanaryv1alpha1.FailedKanaryDeploymentConditionType, corev1.ConditionTrue, "KanaryDeployment failed, deadline activated with 'invalid' status")
	} else if deadlineReached && m.deadlineStatus == kanaryv1alpha1.ValidKanaryDeploymentSpecValidationManualDeadineStatus {
		utils.UpdateKanaryDeploymentStatusCondition(status, metav1.Now(), kanaryv1alpha1.SucceededKanaryDeploymentConditionType, corev1.ConditionTrue, "Deployment updated successfully, dealine activated with 'valid' status")
	}
	return status, result, err
}
