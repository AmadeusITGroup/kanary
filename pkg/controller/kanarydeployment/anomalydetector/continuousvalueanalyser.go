package anomalydetector

import (
	"fmt"
	"math"

	"github.com/amadeusitgroup/kanary/pkg/pod"
	"github.com/go-logr/logr"
	kapiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	kv1 "k8s.io/client-go/listers/core/v1"
)

var _ AnomalyDetector = &ContinuousValueDeviationAnalyser{}

//deviationByPodName float64: 1=no deviation at all, 0.2=80% deviation down, 1.7=70% deviation up
type deviationByPodName map[string]float64
type continuousValueAnalyser interface {
	doAnalysis() (deviationByPodName, error)
}

//ContinuousValueDeviationAnalyser anomalyDetector that check the deviation of a continous value compare to average
type ContinuousValueDeviationAnalyser struct {
	MaxDeviationPercent float64

	selector      labels.Selector
	analyser      continuousValueAnalyser
	podLister     kv1.PodNamespaceLister
	logger        logr.Logger
	exclusionFunc func(*kapiv1.Pod) (bool, error)
}

// func excludePodFromComparisonKubervisor(p *kapiv1.Pod) (bool, error) {
// 	traffic, _, err2 := labeling.IsPodTrafficLabelOkOrPause(p)
// 	if err2 != nil {
// 		return nil, err2
// 	}
// 	return !traffic, nil
// }

//PodByName return 2 maps of pods
// all pods indexed by their names
// all pods to be excluded from comparison indexed by their names
func PodByName(listOfPods []*kapiv1.Pod, exclusionFunc func(*kapiv1.Pod) (bool, error)) (allPods, excludeFromComparison map[string]*kapiv1.Pod, err error) {
	podByName := map[string]*kapiv1.Pod{}
	podWithNoTraffic := map[string]*kapiv1.Pod{}

	for _, p := range listOfPods {
		podByName[p.Name] = p
		if exclusionFunc != nil {
			exclude, err := exclusionFunc(p)
			if err != nil {
				return nil, nil, err
			}
			if exclude {
				podWithNoTraffic[p.Name] = p
			}
		}
	}
	return podByName, podWithNoTraffic, nil
}

//GetPodsOutOfBounds implements interface AnomalyDetector
func (d *ContinuousValueDeviationAnalyser) GetPodsOutOfBounds() ([]*kapiv1.Pod, error) {
	listOfPods, err := d.podLister.List(d.selector)
	if err != nil {
		return nil, fmt.Errorf("can't list pods, error:%v", err)
	}
	listOfPods, err = pod.PurgeNotReadyPods(listOfPods)
	if err != nil {
		return nil, fmt.Errorf("can't purge not ready pods, error:%v", err)
	}
	podByName, podWithNoTraffic, err := PodByName(listOfPods, d.exclusionFunc)
	if err != nil {
		return nil, err
	}
	result := []*kapiv1.Pod{}

	deviationByPods, err := d.analyser.doAnalysis()
	if err != nil {
		return nil, err
	}

	if len(deviationByPods) == 0 {
		return result, nil
	}

	maxDeviation := d.MaxDeviationPercent / 100.0
	if maxDeviation == 0.0 {
		zeroErr := fmt.Errorf("maxDeviation=0 for continuous value analysis")
		d.logger.Error(zeroErr, "")
		return nil, zeroErr
	}

	for podName, deviation := range deviationByPods {
		_, found := podWithNoTraffic[podName]
		if found {
			continue
		}

		if math.Abs(1-deviation) > maxDeviation {
			if p, ok := podByName[podName]; ok {
				// Only keeping known pod with too big deviation
				result = append(result, p)
			}
		}
	}
	return result, nil
}
