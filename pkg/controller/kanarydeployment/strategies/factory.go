package strategies

import (
	"fmt"
	"os"

	"github.com/go-logr/logr"

	appsv1beta1 "k8s.io/api/apps/v1beta1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

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

	var validationImpl validation.Interface
	if spec.Validation.Manual != nil {
		validationImpl = validation.NewManual(&spec.Validation)
	} else if spec.Validation.LabelWatch != nil {
		validationImpl = validation.NewLabelWatch(&spec.Validation)
	} /* else if spec.Validation.PromQL != nil {
		// TODO implement NewPromQL()
	}*/

	return &strategy{
		scale:               scaleImpls,
		traffic:             trafficImpls,
		validation:          validationImpl,
		subResourceDisabled: os.Getenv(config.KanaryStatusSubresourceDisabledEnvVar) == "1",
	}, nil
}

type strategy struct {
	scale               map[scale.Interface]bool
	traffic             map[traffic.Interface]bool
	validation          validation.Interface
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
	reqLogger.Info("Check Validation")
	if validation.IsValidationDelayPeriodDone(kd) {
		status, result, err = s.validation.Validation(kclient, reqLogger, kd, dep, canarydep)
		if err != nil {
			return status, result, fmt.Errorf("error during Validation processing, err: %v", err)
		}
	}
	return status, result, err
}

func needReturn(result *reconcile.Result) bool {
	if result.Requeue || int64(result.RequeueAfter) > int64(0) {
		return true
	}
	return false
}
