package anomalydetector

import (
	"fmt"

	"github.com/amadeusitgroup/kanary/pkg/pod"
	kapiv1 "k8s.io/api/core/v1"
)

var _ AnomalyDetector = &ValueInRangeAnalyser{}

//inRangeByPodName true means in range
type inRangeByPodName map[string]bool
type valueInRangeAnalyser interface {
	doAnalysis() (inRangeByPodName, error)
}

//ValueInRangeConfig Configuration for ValueInRangeAnalyser
type ValueInRangeConfig struct {
	Min float64
	Max float64
}

//ValueInRangeAnalyser anomalyDetector that check the deviation of a continous value compare to average
type ValueInRangeAnalyser struct {
	ConfigSpecific ValueInRangeConfig
	ConfigAnalyser Config

	analyser valueInRangeAnalyser
}

//GetPodsOutOfBounds implements interface AnomalyDetector
func (d *ValueInRangeAnalyser) GetPodsOutOfBounds() ([]*kapiv1.Pod, error) {
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

	inRangeByPods, err := d.analyser.doAnalysis()
	if err != nil {
		return nil, err
	}

	//check if the key is GlobalKeyQuery that means that the result is applicable to all pods
	if len(inRangeByPods) == 1 {
		if v, ok := inRangeByPods[GlobalQueryKey]; ok {
			for _, pod := range podByName {
				inRangeByPods[pod.Name] = v
			}
			delete(inRangeByPods, GlobalQueryKey)
		}
	}

	for podName, inRange := range inRangeByPods {
		_, found := podWithNoTraffic[podName]
		if found {
			continue
		}
		if !inRange {
			if p, ok := podByName[podName]; ok {
				result = append(result, p)
			}
		}
	}
	return result, nil
}
