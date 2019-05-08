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
	if s.subResourceDisabled {
		//use plain resource
		return utils.UpdateKanaryDeploymentStatus(kclient, reqLogger, kd, newStatus, result, err) //Try with plain resource
	}
	//use subresource
	return utils.UpdateKanaryDeploymentStatus(kclient.Status(), reqLogger, kd, newStatus, result, err) //Updating StatusSubresource may depends on Kubernetes version! https://book.kubebuilder.io/basics/status_subresource.html
}

func (s *strategy) process(kclient client.Client, reqLogger logr.Logger, kd *kanaryv1alpha1.KanaryDeployment, dep, canarydep *appsv1beta1.Deployment) (status *kanaryv1alpha1.KanaryDeploymentStatus, result reconcile.Result, err error) {
	reqLogger.Info("Cleanup scale")
	// First cleanup if needed
	for impl, activated := range s.scale {
		if !activated {
			status, result, err = impl.Clear(kclient, reqLogger, kd, canarydep)
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
			status, result, err = impl.Cleanup(kclient, reqLogger, kd, canarydep)
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
			status, result, err = impl.Scale(kclient, reqLogger, kd, canarydep)
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
			status, result, err = impl.Traffic(kclient, reqLogger, kd, canarydep)
			if err != nil {
				return status, result, fmt.Errorf("error during Traffic processing, err: %v", err)
			}
			if needReturn(&result) {
				return status, result, err
			}

		}
	}

	if reaminingDelay, done := validation.IsValidationDelayPeriodDone(kd); done {
		reqLogger.Info("Check Validation")
		var results []*validation.Result
		var errs []error
		for _, validationItem := range s.validations {
			var result *validation.Result
			result, err = validationItem.Validation(kclient, reqLogger, kd, dep, canarydep)
			if err != nil {
				errs = append(errs, err)
			}
			results = append(results, result)
		}
		if len(errs) > 0 {
			return &kd.Status, reconcile.Result{Requeue: true}, utilerrors.NewAggregate(errs)
		}
		var needUpdateDeployment bool
		var failed bool
		status, result, failed, needUpdateDeployment = computeStatus(results, status)
		if needReturn(&result) {
			return status, result, nil
		}
		if !failed && needUpdateDeployment && !kd.Spec.Validations.NoUpdate {
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
		if needUpdateDeployment && !failed {
			utils.UpdateKanaryDeploymentStatusCondition(status, metav1.Now(), kanaryv1alpha1.SucceededKanaryDeploymentConditionType, corev1.ConditionTrue, "Deployment updated successfully")
		}
	} else {
		reqLogger.Info("Check Validation", "requeue-initial-delay", reaminingDelay)
		result.RequeueAfter = reaminingDelay
		return status, result, err
	}

	return status, result, err
}

func computeStatus(results []*validation.Result, status *kanaryv1alpha1.KanaryDeploymentStatus) (*kanaryv1alpha1.KanaryDeploymentStatus, reconcile.Result, bool, bool) {
	newStatus := status.DeepCopy()
	newResult := reconcile.Result{}
	isFailed := false
	needUpdateDeployment := true
	if len(results) == 0 {
		return newStatus, newResult, isFailed, needUpdateDeployment
	}

	comments := []string{}
	for _, result := range results {
		if result.IsFailed {
			isFailed = true
		}
		if !result.NeedUpdateDeployment {
			needUpdateDeployment = false
		}
		if result.Comment == "" {
			comments = append(comments, result.Comment)
		}
		if result.Requeue {
			newResult.Requeue = true
		}
		if newResult.RequeueAfter == 0 {
			newResult.RequeueAfter = result.RequeueAfter
		} else if newResult.RequeueAfter != 0 && result.RequeueAfter < newResult.RequeueAfter {
			newResult.RequeueAfter = result.RequeueAfter
		}
	}
	if isFailed {
		utils.UpdateKanaryDeploymentStatusCondition(newStatus, metav1.Now(), kanaryv1alpha1.FailedKanaryDeploymentConditionType, corev1.ConditionTrue, fmt.Sprintf("KanaryDeployment failed, %s", strings.Join(comments, ",")))
	}
	return newStatus, newResult, isFailed, needUpdateDeployment
}

func needReturn(result *reconcile.Result) bool {
	if result.Requeue || int64(result.RequeueAfter) > int64(0) {
		return true
	}
	return false
}
