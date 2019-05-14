package validation

import (
	"context"
	"time"

	"github.com/go-logr/logr"

	appsv1beta1 "k8s.io/api/apps/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"

	"sigs.k8s.io/controller-runtime/pkg/client"

	kanaryv1alpha1 "github.com/amadeusitgroup/kanary/pkg/apis/kanary/v1alpha1"
	"github.com/amadeusitgroup/kanary/pkg/controller/kanarydeployment/anomalydetector"
)

// NewPromql returns new validation.Manual instance
func NewPromql(list *kanaryv1alpha1.KanaryDeploymentSpecValidationList, s *kanaryv1alpha1.KanaryDeploymentSpecValidation) Interface {

	return &promqlImpl{
		validationSpec:    *s.PromQL,
		validationPeriod:  list.ValidationPeriod.Duration,
		maxIntervalPeriod: list.MaxIntervalPeriod.Duration,
		dryRun:            list.NoUpdate,
	}
}

type promqlImpl struct {
	validationSpec    kanaryv1alpha1.KanaryDeploymentSpecValidationPromQL
	validationPeriod  time.Duration
	maxIntervalPeriod time.Duration
	dryRun            bool

	anomalydetector        anomalydetector.AnomalyDetector
	anomalydetectorFactory anomalydetector.Factory //for test purposes
}

type promqlPodLister struct {
	kclient   client.Client
	Namespace string
}

// List lists all Pods in the indexer.
func (pl *promqlPodLister) List(selector labels.Selector) (ret []*corev1.Pod, err error) {
	list := &corev1.PodList{}
	if err := pl.kclient.List(context.TODO(), &client.ListOptions{Namespace: pl.Namespace, LabelSelector: selector}, list); err != nil {
		return nil, err
	}
	result := []*corev1.Pod{}
	for _, p := range list.Items {
		result = append(result, &p)
	}
	return result, nil
}

// Pods returns an object that can list and get Pods.
func (pl *promqlPodLister) Get(name string) (*corev1.Pod, error) {
	pod := &corev1.Pod{}
	if err := pl.kclient.Get(context.TODO(), client.ObjectKey{Name: name, Namespace: pl.Namespace}, pod); err != nil {
		return nil, err
	}
	return pod, nil
}

func (p *promqlImpl) initAnomalyDetector(kclient client.Client, reqLogger logr.Logger, kd *kanaryv1alpha1.KanaryDeployment, labelSelector map[string]string) error {
	//config is kind of cloned but that allow decoupling between the CRD definition and the anomalydetector package
	anomalyDetectorConfig := anomalydetector.FactoryConfig{
		Config: anomalydetector.Config{
			Logger: reqLogger,
			PodLister: &promqlPodLister{
				kclient:   kclient,
				Namespace: kd.Namespace,
			},
			Selector: labels.SelectorFromSet(labelSelector),
		},
		PromConfig: &anomalydetector.ConfigPrometheusAnomalyDetector{
			PrometheusService: p.validationSpec.PrometheusService,
			PodNameKey:        p.validationSpec.PodNameKey,
			AllPodsQuery:      p.validationSpec.AllPodsQuery,
			Query:             p.validationSpec.Query,
		},
	}

	if p.validationSpec.ContinuousValueDeviation != nil {
		anomalyDetectorConfig.ContinuousValueDeviationConfig = &anomalydetector.ContinuousValueDeviationConfig{
			MaxDeviationPercent: *p.validationSpec.ContinuousValueDeviation.MaxDeviationPercent,
		}
	} else if p.validationSpec.ValueInRange != nil {
		anomalyDetectorConfig.ValueInRangeConfig = &anomalydetector.ValueInRangeConfig{
			Min: *p.validationSpec.ValueInRange.Min,
			Max: *p.validationSpec.ValueInRange.Max,
		}
	} else if p.validationSpec.DiscreteValueOutOfList != nil {
		anomalyDetectorConfig.DiscreteValueOutOfListConfig = &anomalydetector.DiscreteValueOutOfListConfig{
			BadValues:        p.validationSpec.DiscreteValueOutOfList.BadValues,
			GoodValues:       p.validationSpec.DiscreteValueOutOfList.GoodValues,
			Key:              p.validationSpec.DiscreteValueOutOfList.Key,
			TolerancePercent: *p.validationSpec.DiscreteValueOutOfList.TolerancePercent,
		}
	}

	if p.anomalydetectorFactory == nil {
		p.anomalydetectorFactory = anomalydetector.New
	}

	var err error
	if p.anomalydetector, err = p.anomalydetectorFactory(anomalyDetectorConfig); err != nil {
		return err
	}
	return nil
}

func (p *promqlImpl) Validation(kclient client.Client, reqLogger logr.Logger, kd *kanaryv1alpha1.KanaryDeployment, dep, canaryDep *appsv1beta1.Deployment) (*Result, error) {
	var err error
	result := &Result{}

	//re-init the anomaly detector at each validation in case some settings have changed in the kd
	if err = p.initAnomalyDetector(kclient, reqLogger, kd, canaryDep.Spec.Selector.MatchLabels); err != nil {
		return result, err
	}
	// By default a Deployement is valid until a Label is discovered on pod or deployment.
	pods, err := p.anomalydetector.GetPodsOutOfBounds()
	if err != nil {
		reqLogger.Error(err, "GetPodsOutOfBounds")
		return result, err
	}

	//Check if at least one kanary pod was detected by anomaly detector
	if len(pods) > 0 {
		result.IsFailed = true
		reqLogger.Info("GetPodsOutOfBounds", "detection", len(pods))
	}

	if result.IsFailed {
		result.Comment = "promQL query reported an issue with one of the kanary pod"
	}

	return result, err
}
