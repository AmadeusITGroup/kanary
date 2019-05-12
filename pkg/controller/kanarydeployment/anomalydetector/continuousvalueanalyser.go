package anomalydetector

import (
	"fmt"
	"math"

	"github.com/amadeusitgroup/kanary/pkg/pod"
	kapiv1 "k8s.io/api/core/v1"
)

var _ AnomalyDetector = &ContinuousValueDeviationAnalyser{}

//deviationByPodName float64: 1=no deviation at all, 0.2=80% deviation down, 1.7=70% deviation up
type deviationByPodName map[string]float64
type continuousValueAnalyser interface {
	doAnalysis() (deviationByPodName, error)
}

//ContinuousValueDeviationConfig Configuration for ContinuousValueDeviationAnalyser
type ContinuousValueDeviationConfig struct {
	MaxDeviationPercent float64
}

//ContinuousValueDeviationAnalyser anomalyDetector that check the deviation of a continous value compare to average
type ContinuousValueDeviationAnalyser struct {
	ConfigSpecific ContinuousValueDeviationConfig
	ConfigAnalyser Config

	analyser continuousValueAnalyser
}

//GetPodsOutOfBounds implements interface AnomalyDetector
func (d *ContinuousValueDeviationAnalyser) GetPodsOutOfBounds() ([]*kapiv1.Pod, error) {
	listOfPods, err := d.ConfigAnalyser.PodLister.List(d.ConfigAnalyser.Selector)
	if err != nil {
		return nil, fmt.Errorf("can't list pods, error:%v", err)
	}
	listOfPods, err = pod.PurgeNotReadyPods(listOfPods)
	if err != nil {
		return nil, fmt.Errorf("can't purge not ready pods, error:%v", err)
	}
	podByName, podWithNoTraffic, err := PodByName(listOfPods, d.ConfigAnalyser.ExclusionFunc)
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

	maxDeviation := d.ConfigSpecific.MaxDeviationPercent / 100.0
	if maxDeviation == 0.0 {
		zeroErr := fmt.Errorf("maxDeviation=0 for continuous value analysis")
		d.ConfigAnalyser.Logger.Error(zeroErr, "")
		return nil, zeroErr
	}

	//check if the key is GlobalKeyQuery that means that the result is applicable to all pods
	if len(deviationByPods) == 1 {
		if v, ok := deviationByPods[GlobalQueryKey]; ok {
			for _, pod := range podByName {
				deviationByPods[pod.Name] = v
			}
			delete(deviationByPods, GlobalQueryKey)
		}
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
