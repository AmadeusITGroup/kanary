package validation

import (
	"fmt"

	"github.com/go-logr/logr"

	appsv1beta1 "k8s.io/api/apps/v1beta1"
	corev1 "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	"sigs.k8s.io/controller-runtime/pkg/client"

	kanaryv1alpha1 "github.com/amadeusitgroup/kanary/pkg/apis/kanary/v1alpha1"
)

// NewLabelWatch returns new validation.LabelWatch instance
func NewLabelWatch(list *kanaryv1alpha1.KanaryDeploymentSpecValidationList, s *kanaryv1alpha1.KanaryDeploymentSpecValidation) Interface {
	return &labelWatchImpl{
		dryRun: list.NoUpdate,
		config: s.LabelWatch,
	}
}

type labelWatchImpl struct {
	dryRun bool
	config *kanaryv1alpha1.KanaryDeploymentSpecValidationLabelWatch
}

func (l *labelWatchImpl) Validation(kclient client.Client, reqLogger logr.Logger, kd *kanaryv1alpha1.KanaryDeployment, dep, canaryDep *appsv1beta1.Deployment) (*Result, error) {
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
		deadlineReached = IsDeadlinePeriodDone(kd)
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
