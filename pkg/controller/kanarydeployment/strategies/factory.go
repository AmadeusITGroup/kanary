package strategies

import (
	"fmt"

	"github.com/go-logr/logr"

	appsv1beta1 "k8s.io/api/apps/v1beta1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	kanaryv1alpha1 "github.com/amadeusitgroup/kanary/pkg/apis/kanary/v1alpha1"
	"github.com/amadeusitgroup/kanary/pkg/controller/kanarydeployment/strategies/scale"
	"github.com/amadeusitgroup/kanary/pkg/controller/kanarydeployment/strategies/traffic"
	"github.com/amadeusitgroup/kanary/pkg/controller/kanarydeployment/strategies/validation"
	"github.com/amadeusitgroup/kanary/pkg/controller/kanarydeployment/utils"
)

// Interface represent the strategy interface
type Interface interface {
	Manage(kclient client.Client, reqLogger logr.Logger, kd *kanaryv1alpha1.KanaryDeployment, dep, canarydep *appsv1beta1.Deployment) (result reconcile.Result, err error)
}

// NewStrategy return new instance of the strategy
func NewStrategy(spec *kanaryv1alpha1.KanaryDeploymentSpec) (Interface, error) {
	var scaleImpl scale.Interface
	if spec.Scale.Static != nil {
		scaleImpl = scale.NewStatic(spec.Scale.Static)
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
		// TODO implement NewLabelWatch()
	} else if spec.Validation.PromQL != nil {
		// TODO implement NewPromQL()
	}

	return &strategy{
		scale:      scaleImpl,
		traffic:    trafficImpls,
		validation: validationImpl,
	}, nil
}

type strategy struct {
	scale      scale.Interface
	traffic    map[traffic.Interface]bool
	validation validation.Interface
}

func (s *strategy) Manage(kclient client.Client, reqLogger logr.Logger, kd *kanaryv1alpha1.KanaryDeployment, dep, canarydep *appsv1beta1.Deployment) (result reconcile.Result, err error) {
	var newStatus *kanaryv1alpha1.KanaryDeploymentStatus
	newStatus, result, err = s.process(kclient, reqLogger, kd, dep, canarydep)

	utils.UpdateKanaryDeploymentStatusConditionsFailure(newStatus, metav1.Now(), err)
	return utils.UpdateKanaryDeploymentStatus(kclient, reqLogger, kd, newStatus, result, err)
}

func (s *strategy) process(kclient client.Client, reqLogger logr.Logger, kd *kanaryv1alpha1.KanaryDeployment, dep, canarydep *appsv1beta1.Deployment) (status *kanaryv1alpha1.KanaryDeploymentStatus, result reconcile.Result, err error) {
	status, result, err = s.scale.Scale(kclient, reqLogger, kd, canarydep)
	if err != nil {
		return status, result, fmt.Errorf("error during Scale process, err: %v", err)
	}
	if needReturn(&result) {
		return status, result, err
	}

	for impl, activated := range s.traffic {
		if activated {
			status, result, err = impl.Traffic(kclient, reqLogger, kd, canarydep)
		} else {
			status, result, err = impl.Cleanup(kclient, reqLogger, kd)
		}
		if err != nil {
			return status, result, fmt.Errorf("error during Traffic, err: %v", err)
		}
		if needReturn(&result) {
			return status, result, err
		}
	}

	status, result, err = s.validation.Validation(kclient, reqLogger, kd, dep, canarydep)
	if err != nil {
		return status, result, fmt.Errorf("error during Validation process, err: %v", err)
	}
	return status, result, err
}

func needReturn(result *reconcile.Result) bool {
	if result.Requeue || int64(result.RequeueAfter) > int64(0) {
		return true
	}
	return false
}
