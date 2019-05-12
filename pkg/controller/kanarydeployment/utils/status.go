package utils

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-logr/logr"

	corev1 "k8s.io/api/core/v1"

	apiequality "k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/amadeusitgroup/kanary/pkg/apis/kanary/v1alpha1"
	kanaryv1alpha1 "github.com/amadeusitgroup/kanary/pkg/apis/kanary/v1alpha1"
)

// UpdateKanaryDeploymentStatus used to update the KanaryDeployment.Status if it has changed.
func UpdateKanaryDeploymentStatus(kclient client.StatusWriter, reqLogger logr.Logger, kd *kanaryv1alpha1.KanaryDeployment, newStatus *kanaryv1alpha1.KanaryDeploymentStatus, result reconcile.Result, err error) (reconcile.Result, error) {
	//No need to go further if the same pointer is given
	if &kd.Status == newStatus {
		return result, err
	}

	updateStatusReport(kd, newStatus)
	if !apiequality.Semantic.DeepEqual(&kd.Status, newStatus) {
		updatedKd := kd.DeepCopy()
		updatedKd.Status = *newStatus
		err2 := kclient.Update(context.TODO(), updatedKd)
		if err2 != nil {
			reqLogger.Error(err2, "failed to update KanaryDeployment status", "KanaryDeployment.Namespace", updatedKd.Namespace, "KanaryDeployment.Name", updatedKd.Name)
			return reconcile.Result{}, err2
		}
	}
	return result, err
}

// UpdateKanaryDeploymentStatusConditionsFailure used to update the failre StatusConditions
func UpdateKanaryDeploymentStatusConditionsFailure(status *kanaryv1alpha1.KanaryDeploymentStatus, now metav1.Time, err error) {
	if err != nil {
		UpdateKanaryDeploymentStatusCondition(status, now, kanaryv1alpha1.ErroredKanaryDeploymentConditionType, corev1.ConditionTrue, fmt.Sprintf("%v", err))
	} else {
		UpdateKanaryDeploymentStatusCondition(status, now, kanaryv1alpha1.ErroredKanaryDeploymentConditionType, corev1.ConditionFalse, "")
	}
}

// UpdateKanaryDeploymentStatusCondition used to update a specific KanaryDeploymentConditionType
func UpdateKanaryDeploymentStatusCondition(status *kanaryv1alpha1.KanaryDeploymentStatus, now metav1.Time, t kanaryv1alpha1.KanaryDeploymentConditionType, conditionStatus corev1.ConditionStatus, desc string) {
	idConditionComplete := getIndexForConditionType(status, t)
	if idConditionComplete >= 0 {
		if status.Conditions[idConditionComplete].Status != conditionStatus {
			status.Conditions[idConditionComplete].LastTransitionTime = now
			status.Conditions[idConditionComplete].Status = conditionStatus
		}
		status.Conditions[idConditionComplete].LastUpdateTime = now
		status.Conditions[idConditionComplete].Message = desc
	} else if conditionStatus == corev1.ConditionTrue {
		// Only add if the condition is True
		status.Conditions = append(status.Conditions, NewKanaryDeploymentStatusCondition(t, now, "", desc))
	}
}

// NewKanaryDeploymentStatusCondition returns new KanaryDeploymentCondition instance
func NewKanaryDeploymentStatusCondition(conditionType kanaryv1alpha1.KanaryDeploymentConditionType, now metav1.Time, reason, message string) kanaryv1alpha1.KanaryDeploymentCondition {
	return kanaryv1alpha1.KanaryDeploymentCondition{
		Type:               conditionType,
		Status:             corev1.ConditionTrue,
		LastUpdateTime:     now,
		LastTransitionTime: now,
		Reason:             reason,
		Message:            message,
	}
}

// IsKanaryDeploymentErrored returns true if the KanaryDeployment has failed, else returns false
func IsKanaryDeploymentErrored(status *kanaryv1alpha1.KanaryDeploymentStatus) bool {
	if status == nil {
		return false
	}
	id := getIndexForConditionType(status, kanaryv1alpha1.ErroredKanaryDeploymentConditionType)
	if id >= 0 && status.Conditions[id].Status == corev1.ConditionTrue {
		return true
	}
	return false
}

// IsKanaryDeploymentFailed returns true if the KanaryDeployment has failed, else returns false
func IsKanaryDeploymentFailed(status *kanaryv1alpha1.KanaryDeploymentStatus) bool {
	if status == nil {
		return false
	}
	id := getIndexForConditionType(status, kanaryv1alpha1.FailedKanaryDeploymentConditionType)
	if id >= 0 && status.Conditions[id].Status == corev1.ConditionTrue {
		return true
	}
	return false
}

// IsKanaryDeploymentSucceeded returns true if the KanaryDeployment has succeeded, else return false
func IsKanaryDeploymentSucceeded(status *kanaryv1alpha1.KanaryDeploymentStatus) bool {
	if status == nil {
		return false
	}
	id := getIndexForConditionType(status, kanaryv1alpha1.SucceededKanaryDeploymentConditionType)
	if id >= 0 && status.Conditions[id].Status == corev1.ConditionTrue {
		return true
	}
	return false
}

// IsKanaryDeploymentDeploymentUpdated returns true if the KanaryDeployment has lead to deployment update
func IsKanaryDeploymentDeploymentUpdated(status *kanaryv1alpha1.KanaryDeploymentStatus) bool {
	if status == nil {
		return false
	}
	id := getIndexForConditionType(status, kanaryv1alpha1.DeploymentUpdatedKanaryDeploymentConditionType)
	if id >= 0 && status.Conditions[id].Status == corev1.ConditionTrue {
		return true
	}
	return false
}

// IsKanaryDeploymentRunning returns true if the KanaryDeployment is runnning
func IsKanaryDeploymentRunning(status *kanaryv1alpha1.KanaryDeploymentStatus) bool {
	return !IsKanaryDeploymentFailed(status) && !IsKanaryDeploymentSucceeded(status) && !IsKanaryDeploymentDeploymentUpdated(status)
}

func getIndexForConditionType(status *kanaryv1alpha1.KanaryDeploymentStatus, t kanaryv1alpha1.KanaryDeploymentConditionType) int {
	idCondition := -1
	if status == nil {
		return idCondition
	}
	for i, condition := range status.Conditions {
		if condition.Type == t {
			idCondition = i
			break
		}
	}
	return idCondition
}

func getReportStatus(status *kanaryv1alpha1.KanaryDeploymentStatus) string {

	// Order matters compare to the lifecycle of the kanary during validation

	if IsKanaryDeploymentFailed(status) {
		return string(v1alpha1.FailedKanaryDeploymentConditionType)
	}

	if IsKanaryDeploymentDeploymentUpdated(status) {
		return string(v1alpha1.DeploymentUpdatedKanaryDeploymentConditionType)
	}

	if IsKanaryDeploymentSucceeded(status) {
		return string(v1alpha1.SucceededKanaryDeploymentConditionType)
	}
	return string(v1alpha1.RunningKanaryDeploymentConditionType)
}

func getValidation(kd *kanaryv1alpha1.KanaryDeployment) string {
	var list []string
	for _, v := range kd.Spec.Validations.Items {
		if v.LabelWatch != nil {
			list = append(list, "labelWatch")
		}
		if v.PromQL != nil {
			list = append(list, "promQL")
		}
		if v.Manual != nil {
			list = append(list, "manual")
		}
	}
	if len(list) == 0 {
		return "unknow"
	}
	return strings.Join(list, ",")
}

func getScale(kd *kanaryv1alpha1.KanaryDeployment) string {
	if kd.Spec.Scale.HPA == nil {
		return "static"
	}
	return "hpa"
}

func getTraffic(kd *kanaryv1alpha1.KanaryDeployment) string {
	return string(kd.Spec.Traffic.Source)
}

func updateStatusReport(kd *kanaryv1alpha1.KanaryDeployment, status *kanaryv1alpha1.KanaryDeploymentStatus) {
	status.Report = kanaryv1alpha1.KanaryDeploymentStatusReport{
		Status:     getReportStatus(status),
		Validation: getValidation(kd),
		Scale:      getScale(kd),
		Traffic:    getTraffic(kd),
	}
}
