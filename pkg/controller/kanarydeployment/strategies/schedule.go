package strategies

import (
	"fmt"
	"time"

	kanaryv1alpha1 "github.com/amadeusitgroup/kanary/pkg/apis/kanary/v1alpha1"
	"github.com/amadeusitgroup/kanary/pkg/controller/kanarydeployment/utils"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	schedOnTheFlyMessage = "On the fly"
)

func getScheduleTime(strSchedule string) (time.Time, error) {
	//Check RFC3339 = "2006-01-02T15:04:05Z07:00"
	t, err := time.Parse(time.RFC3339, strSchedule)
	if err != nil {
		return time.Time{}, fmt.Errorf("can't parse Schedule field: %s", err.Error())
	}

	if t.Before(time.Now().Add(-time.Minute)) {
		return time.Time{}, fmt.Errorf("Scheduling time in the past by more than one minute: %s", time.Since(t).String())
	}
	return t, nil
}

//apply the scheduling,
// status: the modified status. Modified with the scheduling activities
// reconcile result: should the item be requeued (and maybe status applied or error handled)
// if status and result are nil that mean that everyting is ok an next step in the reconcile sequence should be engaged
func ApplyScheduling(reqLogger logr.Logger, kd *kanaryv1alpha1.KanaryDeployment) (*kanaryv1alpha1.KanaryDeploymentStatus, *reconcile.Result) {
	if !utils.IsKanaryDeploymentScheduled(&kd.Status) {
		status := kd.Status.DeepCopy()
		if kd.Spec.Schedule == "" {
			utils.UpdateKanaryDeploymentStatusCondition(status, metav1.Now(), kanaryv1alpha1.ScheduledKanaryDeploymentConditionType, corev1.ConditionTrue, schedOnTheFlyMessage, false)
			return status, &reconcile.Result{Requeue: true} // Scheduled!
		}

		sched, err := getScheduleTime(kd.Spec.Schedule)
		if err != nil {
			utils.UpdateKanaryDeploymentStatusCondition(status, metav1.Now(), kanaryv1alpha1.ScheduledKanaryDeploymentConditionType, corev1.ConditionFalse, err.Error(), true)
			reqLogger.Info("Schedule", "Not Scheduled", err.Error())
			return status, &reconcile.Result{} // don't requeue it is over for this kanary
		}
		utils.UpdateKanaryDeploymentStatusCondition(status, metav1.Now(), kanaryv1alpha1.ScheduledKanaryDeploymentConditionType, corev1.ConditionTrue, sched.Format(time.RFC3339), false)
		return status, &reconcile.Result{RequeueAfter: time.Until(sched)} // Scheduled!
	}

	if kd.Spec.Schedule != "" {
		status := kd.Status.DeepCopy()
		sched, err := getScheduleTime(kd.Spec.Schedule)
		if err != nil {
			utils.UpdateKanaryDeploymentStatusCondition(status, metav1.Now(), kanaryv1alpha1.ScheduledKanaryDeploymentConditionType, corev1.ConditionFalse, err.Error(), true)
			reqLogger.Info("Schedule", "Not Scheduled", err.Error())
			return status, &reconcile.Result{} // don't requeue it is over for this kanary
		}
		if sched.After(time.Now()) {
			utils.UpdateKanaryDeploymentStatusCondition(status, metav1.Now(), kanaryv1alpha1.ScheduledKanaryDeploymentConditionType, corev1.ConditionTrue, sched.Format(time.RFC3339), false)
			return status, &reconcile.Result{RequeueAfter: time.Until(sched)} // Scheduled!
		}
	}
	return nil, nil
}
