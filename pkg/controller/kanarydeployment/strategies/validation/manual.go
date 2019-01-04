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
		deadline:         s.Manual.Deadline,
		status:           s.Manual.Status,
		validationPeriod: s.ValidationPeriod.Duration,
	}
}

type manualImpl struct {
	deadline         kanaryv1alpha1.KanaryDeploymentSpecValidationManualDeadine
	status           kanaryv1alpha1.KanaryDeploymentSpecValidationManualStatus
	validationPeriod time.Duration
}

func (m *manualImpl) Validation(kclient client.Client, reqLogger logr.Logger, kd *kanaryv1alpha1.KanaryDeployment, dep, canaryDep *appsv1beta1.Deployment) (status *kanaryv1alpha1.KanaryDeploymentStatus, result reconcile.Result, err error) {
	status = kd.Status.DeepCopy()

	var needUpdateDeployment bool
	if m.status == kanaryv1alpha1.ValidKanaryDeploymentSpecValidationManualStatus {
		needUpdateDeployment = true
	}
	deadlineActivated := m.isDeadlinePeriodDone(kd.CreationTimestamp.Time, time.Now())
	if deadlineActivated && m.deadline == kanaryv1alpha1.ValidKanaryDeploymentSpecValidationManualDeadine {
		needUpdateDeployment = true
	}

	if needUpdateDeployment {
		newDep, err := utils.UpdateDeploymentForKanaryDeployment(kd, dep)
		if err != nil {
			reqLogger.Error(err, "failed to update the Deployment artifact", "Deployment.Namespace", newDep.Namespace, "Deployment.Name", newDep.Name)
			return status, reconcile.Result{}, err
		}

		reqLogger.Info("Updating Deployment", "Deployment.Namespace", newDep.Namespace, "Deployment.Name", newDep.Name)
		err = kclient.Update(context.TODO(), newDep)
		if err != nil {
			reqLogger.Error(err, "failed to update the Deployment", "Deployment.Namespace", newDep.Namespace, "Deployment.Name", newDep.Name)
			return status, reconcile.Result{}, err
		}
	}

	if m.status == kanaryv1alpha1.ValidKanaryDeploymentSpecValidationManualStatus {
		utils.UpdateKanaryDeploymentStatusCondition(status, metav1.Now(), kanaryv1alpha1.SucceededKanaryDeploymentConditionType, corev1.ConditionTrue, "Deployment updated successfully")
	} else if m.status == kanaryv1alpha1.InvalidKanaryDeploymentSpecValidationManualStatus {
		utils.UpdateKanaryDeploymentStatusCondition(status, metav1.Now(), kanaryv1alpha1.FailedKanaryDeploymentConditionType, corev1.ConditionTrue, "KanaryDeployment validation failed, manual.status=invalid")
	} else if deadlineActivated && m.deadline == kanaryv1alpha1.InvalidKanaryDeploymentSpecValidationManualDeadine {
		utils.UpdateKanaryDeploymentStatusCondition(status, metav1.Now(), kanaryv1alpha1.FailedKanaryDeploymentConditionType, corev1.ConditionTrue, "KanaryDeployment failed, deadline activated with 'invalid' status")
	} else if deadlineActivated && m.deadline == kanaryv1alpha1.ValidKanaryDeploymentSpecValidationManualDeadine {
		utils.UpdateKanaryDeploymentStatusCondition(status, metav1.Now(), kanaryv1alpha1.SucceededKanaryDeploymentConditionType, corev1.ConditionTrue, "Deployment updated successfully, dealine activated with 'valid' status")
	}

	return status, result, err
}

func (m *manualImpl) isDeadlinePeriodDone(startTime, now time.Time) bool {
	if startTime.Add(m.validationPeriod).Before(now) {
		return true
	}

	return false
}
