package validation

import (
	"context"
	"fmt"
	"time"

	"github.com/amadeusitgroup/kanary/pkg/apis/kanary/v1alpha1"
	kanaryv1alpha1 "github.com/amadeusitgroup/kanary/pkg/apis/kanary/v1alpha1"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

//GetNextValidationCheckDuration return the shortest duration between deadline-now and MaxIntervalPeriod
func GetNextValidationCheckDuration(kd *v1alpha1.KanaryDeployment) time.Duration {
	deadline := GetValidationDeadLine(kd)
	d := time.Until(deadline)
	if d < 0 {
		return time.Millisecond
	}

	if d < kd.Spec.Validations.MaxIntervalPeriod.Duration {
		return d
	}
	return kd.Spec.Validations.MaxIntervalPeriod.Duration
}

//GetValidationDeadLine return the timestamp for the end validation period
func GetValidationDeadLine(kd *v1alpha1.KanaryDeployment) time.Time {
	return kd.CreationTimestamp.Time.Add(kd.Spec.Validations.InitialDelay.Duration).Add(kd.Spec.Validations.ValidationPeriod.Duration)
}

// IsDeadlinePeriodDone returns true if the InitialDelay validation periode is over.
func IsDeadlinePeriodDone(kd *v1alpha1.KanaryDeployment) bool {
	return GetValidationDeadLine(kd).Before(time.Now())
}

// IsInitialDelayDone returns true if the InitialDelay validation periode is over.
func IsInitialDelayDone(kd *v1alpha1.KanaryDeployment) (time.Duration, bool) {
	now := time.Now()
	deadline := kd.CreationTimestamp.Time.Add(kd.Spec.Validations.InitialDelay.Duration)

	if now.After(deadline) {
		return deadline.Sub(now), true
	}

	return deadline.Sub(now), false
}

func getPods(kclient client.Client, reqLogger logr.Logger, KanaryDeploymentName, KanaryDeploymentNamespace string) ([]corev1.Pod, error) {
	pods := &corev1.PodList{}
	selector := labels.Set{
		kanaryv1alpha1.KanaryDeploymentKanaryNameLabelKey: KanaryDeploymentName,
	}
	listOptions := &client.ListOptions{
		LabelSelector: selector.AsSelector(),
		Namespace:     KanaryDeploymentNamespace,
	}
	err := kclient.List(context.TODO(), listOptions, pods)
	if err != nil {
		reqLogger.Error(err, "failed to list Pod from canary deployment")
		return nil, fmt.Errorf("failed to list pod from canary deployment, err:%v", err)
	}
	return pods.Items, nil
}
