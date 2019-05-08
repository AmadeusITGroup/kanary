package validation

import (
	"time"

	"github.com/go-logr/logr"

	appsv1beta1 "k8s.io/api/apps/v1beta1"

	"sigs.k8s.io/controller-runtime/pkg/client"

	kanaryv1alpha1 "github.com/amadeusitgroup/kanary/pkg/apis/kanary/v1alpha1"
)

// NewManual returns new validation.Manual instance
func NewManual(list *kanaryv1alpha1.KanaryDeploymentSpecValidationList, s *kanaryv1alpha1.KanaryDeploymentSpecValidation) Interface {
	return &manualImpl{
		deadlineStatus:         s.Manual.StatusAfterDealine,
		validationManualStatus: s.Manual.Status,
		validationPeriod:       list.ValidationPeriod.Duration,
		maxIntervalPeriod:      list.MaxIntervalPeriod.Duration,
		dryRun:                 list.NoUpdate,
	}
}

type manualImpl struct {
	deadlineStatus         kanaryv1alpha1.KanaryDeploymentSpecValidationManualDeadineStatus
	validationManualStatus kanaryv1alpha1.KanaryDeploymentSpecValidationManualStatus
	validationPeriod       time.Duration
	maxIntervalPeriod      time.Duration
	dryRun                 bool
}

func (m *manualImpl) Validation(kclient client.Client, reqLogger logr.Logger, kd *kanaryv1alpha1.KanaryDeployment, dep, canaryDep *appsv1beta1.Deployment) (*Result, error) {
	var err error
	result := &Result{}

	if m.validationManualStatus == kanaryv1alpha1.ValidKanaryDeploymentSpecValidationManualStatus {
		result.NeedUpdateDeployment = true
	}

	var deadlineReached bool
	if canaryDep != nil {
		var requeueAfter time.Duration
		requeueAfter, deadlineReached = isDeadlinePeriodDone(m.validationPeriod, m.maxIntervalPeriod, canaryDep.CreationTimestamp.Time, time.Now())
		if !deadlineReached {
			result.RequeueAfter = requeueAfter
		} else if m.deadlineStatus == kanaryv1alpha1.ValidKanaryDeploymentSpecValidationManualDeadineStatus {
			result.NeedUpdateDeployment = true
		}
	}

	if m.validationManualStatus == kanaryv1alpha1.ValidKanaryDeploymentSpecValidationManualStatus {
	} else if m.validationManualStatus == kanaryv1alpha1.InvalidKanaryDeploymentSpecValidationManualStatus {
		result.IsFailed = true
		result.Comment = "manual.status=invalid"
	} else if deadlineReached && m.deadlineStatus == kanaryv1alpha1.InvalidKanaryDeploymentSpecValidationManualDeadineStatus {
		result.IsFailed = true
		result.Comment = "deadline activated with 'invalid' status"
	} else if deadlineReached && m.deadlineStatus == kanaryv1alpha1.ValidKanaryDeploymentSpecValidationManualDeadineStatus {
		result.Comment = "deadline activated with 'valid' status"
	}

	return result, err
}
