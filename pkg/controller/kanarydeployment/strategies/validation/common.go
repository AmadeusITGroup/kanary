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

func isDeadlinePeriodDone(validationPeriod, maxRequeueDuration time.Duration, startTime, now time.Time) (time.Duration, bool) {
	if startTime.Add(validationPeriod).Before(now) {
		return time.Duration(0), true
	}
	d := startTime.Add(validationPeriod).Sub(now)
	if d-maxRequeueDuration > 0 {
		return maxRequeueDuration, false
	}
	return d, false
}

// IsValidationDelayPeriodDone returns true if the InitialDelay validation periode is over.
func IsValidationDelayPeriodDone(kd *v1alpha1.KanaryDeployment) (time.Duration, bool) {
	now := time.Now()
	return isDeadlinePeriodDone(kd.Spec.Validation.InitialDelay.Duration, kd.Spec.Validation.MaxIntervalPeriod.Duration, kd.CreationTimestamp.Time, now)
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
