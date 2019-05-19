package strategies

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/go-logr/logr"

	appsv1beta1 "k8s.io/api/apps/v1beta1"
	corev1 "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	kanaryv1alpha1 "github.com/amadeusitgroup/kanary/pkg/apis/kanary/v1alpha1"
	"github.com/amadeusitgroup/kanary/pkg/config"
	"github.com/amadeusitgroup/kanary/pkg/controller/kanarydeployment/strategies/scale"
	"github.com/amadeusitgroup/kanary/pkg/controller/kanarydeployment/strategies/traffic"
	"github.com/amadeusitgroup/kanary/pkg/controller/kanarydeployment/strategies/validation"
	"github.com/amadeusitgroup/kanary/pkg/controller/kanarydeployment/utils"
)

// Interface represent the strategy interface
type Interface interface {
	Apply(kclient client.Client, reqLogger logr.Logger, kd *kanaryv1alpha1.KanaryDeployment, dep, canarydep *appsv1beta1.Deployment) (result reconcile.Result, err error)
}

// NewStrategy return new instance of the strategy
func NewStrategy(spec *kanaryv1alpha1.KanaryDeploymentSpec) (Interface, error) {
	scaleStatic := scale.NewStatic(spec.Scale.Static)
	scaleHPA := scale.NewHPA(spec.Scale.HPA)
	scaleImpls := map[scale.Interface]bool{
		scaleStatic: false,
		scaleHPA:    false,
	}
	if spec.Scale.HPA != nil {
		scaleImpls[scaleHPA] = true
	} else {
		scaleImpls[scaleStatic] = true
	}

	trafficKanaryService := traffic.NewKanaryService(&spec.Traffic)
	trafficMirror := traffic.NewMirror(&spec.Traffic)
	trafficImpls := map[traffic.Interface]bool{
		trafficKanaryService: false,
		trafficMirror:        false,
	}

	switch spec.Traffic.Source {
	case kanaryv1alpha1.ServiceKanaryDeploymentSpecTrafficSource, kanaryv1alpha1.KanaryServiceKanaryDeploymentSpecTrafficSource, kanaryv1alpha1.BothKanaryDeploymentSpecTrafficSource:
		trafficImpls[trafficKanaryService] = true
	case kanaryv1alpha1.MirrorKanaryDeploymentSpecTrafficSource:
		trafficImpls[trafficMirror] = true
	default:
	}

	var validationsImpls []validation.Interface
	for _, v := range spec.Validations.Items {
		if v.Manual != nil {
			validationsImpls = append(validationsImpls, validation.NewManual(&spec.Validations, &v))
		} else if v.LabelWatch != nil {
			validationsImpls = append(validationsImpls, validation.NewLabelWatch(&spec.Validations, &v))
		} else if v.PromQL != nil {
			validationsImpls = append(validationsImpls, validation.NewPromql(&spec.Validations, &v))
		}
	}

	return &strategy{
		scale:               scaleImpls,
		traffic:             trafficImpls,
		validations:         validationsImpls,
		subResourceDisabled: os.Getenv(config.KanaryStatusSubresourceDisabledEnvVar) == "1",
	}, nil
}

type strategy struct {
	scale               map[scale.Interface]bool
	traffic             map[traffic.Interface]bool
	validations         []validation.Interface
	subResourceDisabled bool
}

func (s *strategy) Apply(kclient client.Client, reqLogger logr.Logger, kd *kanaryv1alpha1.KanaryDeployment, dep, canarydep *appsv1beta1.Deployment) (result reconcile.Result, err error) {
	var newStatus *kanaryv1alpha1.KanaryDeploymentStatus
	newStatus, result, err = s.process(kclient, reqLogger, kd, dep, canarydep)
	utils.UpdateKanaryDeploymentStatusConditionsFailure(newStatus, metav1.Now(), err)
	return utils.UpdateKanaryDeploymentStatus(kclient, s.subResourceDisabled, reqLogger, kd, newStatus, result, err) //Try with plain resource
}

func (s *strategy) process(kclient client.Client, reqLogger logr.Logger, kd *kanaryv1alpha1.KanaryDeployment, dep, canarydep *appsv1beta1.Deployment) (*kanaryv1alpha1.KanaryDeploymentStatus, reconcile.Result, error) {

	reqLogger.Info("Cleanup scale")
	// First cleanup if needed
	for impl, activated := range s.scale {
		if !activated {
			status, result, err := impl.Clear(kclient, reqLogger, kd, canarydep)
			if err != nil {
				return status, result, fmt.Errorf("error during Clean processing, err: %v", err)
			}
			if needReturn(&result) {
				return status, result, err
			}
		}
	}

	reqLogger.Info("Cleanup traffic")
	// First process cleanup
	for impl, activated := range s.traffic {
		if !activated {
			status, result, err := impl.Cleanup(kclient, reqLogger, kd, canarydep)
			if err != nil {
				return status, result, fmt.Errorf("error during Traffic Cleanup processing, err: %v", err)
			}
			if needReturn(&result) {
				return status, result, err
			}
		}
	}

	reqLogger.Info("Implement scale")
	// then scale if need
	for impl, activated := range s.scale {
		if activated {
			status, result, err := impl.Scale(kclient, reqLogger, kd, canarydep)
			if err != nil {
				return status, result, fmt.Errorf("error during Scale processing, err: %v", err)
			}
			if needReturn(&result) {
				return status, result, err
			}
		}
	}

	reqLogger.Info("Implement traffic")
	// Then apply Traffic configuration
	for impl, activated := range s.traffic {
		if activated {
			status, result, err := impl.Traffic(kclient, reqLogger, kd, canarydep)
			if err != nil {
				return status, result, fmt.Errorf("error during Traffic processing, err: %v", err)
			}
			if needReturn(&result) {
				return status, result, err
			}

		}
	}

	//before going to validation step, let's check that initial delay period is completed
	if reaminingDelay, done := validation.IsInitialDelayDone(kd); !done {
		reqLogger.Info("Check Validation", "requeue-initial-delay", reaminingDelay)
		return &kd.Status, reconcile.Result{RequeueAfter: reaminingDelay}, nil
	}

	//In ase we are still running let's run the validation
	if !utils.IsKanaryDeploymentValidationCompleted(&kd.Status) {
		reqLogger.Info("Check Validation")

		if !utils.IsKanaryDeploymentValidationRunning(&kd.Status) {
			status := kd.Status.DeepCopy()
			utils.UpdateKanaryDeploymentStatusCondition(status, metav1.Now(), kanaryv1alpha1.RunningKanaryDeploymentConditionType, corev1.ConditionTrue, "Validation Started", false)
			reqLogger.Info("Validation Started")
			return status, reconcile.Result{Requeue: true}, nil
		}

		validationDeadlineDone := validation.IsDeadlinePeriodDone(kd)

		//Run validation for all strategies
		var results []*validation.Result
		var errs []error
		for _, validationItem := range s.validations {
			var result *validation.Result
			result, err := validationItem.Validation(kclient, reqLogger, kd, dep, canarydep)
			if err != nil {
				errs = append(errs, err)
			}
			results = append(results, result)
		}
		if len(errs) > 0 {
			return &kd.Status, reconcile.Result{Requeue: true}, utilerrors.NewAggregate(errs)
		}

		var forceSucceededNow bool
		var failMessages string
		failMessages, forceSucceededNow = computeStatus(results)
		failed := failMessages != ""

		// If any strategy fails, the kanary should fail
		if failed {
			status := kd.Status.DeepCopy()
			utils.UpdateKanaryDeploymentStatusCondition(status, metav1.Now(), kanaryv1alpha1.FailedKanaryDeploymentConditionType, corev1.ConditionTrue, fmt.Sprintf("KanaryDeployment failed, %s", failMessages), false)
			utils.UpdateKanaryDeploymentStatusCondition(status, metav1.Now(), kanaryv1alpha1.RunningKanaryDeploymentConditionType, corev1.ConditionFalse, "Validation ended with failure detected", false)
			reqLogger.Info("Check Validation", "in failed", failMessages, "updated status", fmt.Sprintf("%#v", status))
			return status, reconcile.Result{Requeue: true}, nil
		}

		// So there is no failure, does someone force for an early Success ?
		if forceSucceededNow {
			status := kd.Status.DeepCopy()
			utils.UpdateKanaryDeploymentStatusCondition(status, metav1.Now(), kanaryv1alpha1.SucceededKanaryDeploymentConditionType, corev1.ConditionTrue, "Forced Success", false)
			utils.UpdateKanaryDeploymentStatusCondition(status, metav1.Now(), kanaryv1alpha1.RunningKanaryDeploymentConditionType, corev1.ConditionFalse, "Validation ended with success forced", false)
			return status, reconcile.Result{Requeue: true}, nil
		}

		// No failure, so if we have not reached the validation deadline, let's requeue for next validation
		if !validationDeadlineDone && !failed {
			d := validation.GetNextValidationCheckDuration(kd)
			reqLogger.Info("Check Validation", "Periodic-Requeue", d)
			return &kd.Status, reconcile.Result{RequeueAfter: d}, nil
		}

		// Validation completed and everything is ok while we have reached the end of the validation period...

		//Particular case of the manual strategy with None as StatusAfterDeadline
		if validation.IsStatusAfterDeadlineNone(kd) {
			// No automation, no requeue, wait for manual input
			return &kd.Status, reconcile.Result{}, nil
		}

		//Looks like it is a success for the kanary!
		status := kd.Status.DeepCopy()
		utils.UpdateKanaryDeploymentStatusCondition(status, metav1.Now(), kanaryv1alpha1.SucceededKanaryDeploymentConditionType, corev1.ConditionTrue, "Validation ended with success", false)
		utils.UpdateKanaryDeploymentStatusCondition(status, metav1.Now(), kanaryv1alpha1.RunningKanaryDeploymentConditionType, corev1.ConditionFalse, "Validation ended with success", false)
		return status, reconcile.Result{Requeue: true}, nil
	}

	//In case of succeeded kanary, we may need to update the deployment
	if utils.IsKanaryDeploymentSucceeded(&kd.Status) {
		if kd.Spec.Validations.NoUpdate {
			return &kd.Status, reconcile.Result{}, nil // nothing else to do... the kanary succeeded, and we are in dry-run mode
		}

		var newDep *appsv1beta1.Deployment
		newDep, err := utils.UpdateDeploymentWithKanaryDeploymentTemplate(kd, dep)
		if err != nil {
			reqLogger.Error(err, "failed to update the Deployment artifact", "Namespace", newDep.Namespace, "Deployment", newDep.Name)
			return &kd.Status, reconcile.Result{}, err
		}
		err = kclient.Update(context.TODO(), newDep)
		if err != nil {
			reqLogger.Error(err, "failed to update the Deployment", "Namespace", newDep.Namespace, "Deployment", newDep.Name, "newDep", *newDep)
			return &kd.Status, reconcile.Result{}, err
		}
		status := kd.Status.DeepCopy()
		utils.UpdateKanaryDeploymentStatusCondition(status, metav1.Now(), kanaryv1alpha1.DeploymentUpdatedKanaryDeploymentConditionType, corev1.ConditionTrue, "Deployment updated successfully", false)
		return status, reconcile.Result{Requeue: true}, nil
	}
	return &kd.Status, reconcile.Result{}, nil
}

const (
	unknownFailureReason = "unknown failure reason"
)

func computeStatus(results []*validation.Result) (failMessages string, forceSuccessNow bool) {
	if len(results) == 0 {
		return "", forceSuccessNow
	}
	forceSuccessNow = true

	comments := []string{}
	for _, result := range results {

		if !result.ForceSuccessNow {
			forceSuccessNow = false
		}
		if result.IsFailed {
			if result.Comment != "" {
				comments = append(comments, result.Comment)
			} else {
				comments = append(comments, unknownFailureReason)
			}
		}
	}
	if len(comments) > 0 {
		failMessages = strings.Join(comments, ",")
	}
	return failMessages, forceSuccessNow
}

func needReturn(result *reconcile.Result) bool {
	if result.Requeue || int64(result.RequeueAfter) > int64(0) {
		return true
	}
	return false
}
