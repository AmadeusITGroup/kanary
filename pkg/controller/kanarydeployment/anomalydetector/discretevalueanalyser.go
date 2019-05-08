package anomalydetector

import (
	"fmt"

	"github.com/amadeusitgroup/kanary/pkg/pod"
	kapiv1 "k8s.io/api/core/v1"
)

type okkoCount struct {
	ok uint
	ko uint
}

type okkoByPodName map[string]okkoCount
type discreteValueAnalyser interface {
	doAnalysis() (okkoByPodName, error)
}

var _ AnomalyDetector = &DiscreteValueOutOfListAnalyser{}

//DiscreteValueOutOfListConfig configuration for DiscreteValueOutOfListAnalyser
type DiscreteValueOutOfListConfig struct {
	Key              string   // Key for the metrics. For the previous example it will be "code"
	GoodValues       []string // Good Values ["200","201"]. If empty means that BadValues should be used to do exclusion instead of inclusion.
	BadValues        []string // Bad Values ["500","404"].
	TolerancePercent uint
	valueCheckerFunc func(value string) (ok bool)
}

//DiscreteValueOutOfListAnalyser anomalyDetector that check the ratio of good/bad value and return the pods that exceed a given threshold for that ratio
type DiscreteValueOutOfListAnalyser struct {
	ConfigSpecific DiscreteValueOutOfListConfig
	ConfigAnalyser Config

	analyser discreteValueAnalyser
}

//GetPodsOutOfBounds implements interface AnomalyDetector
func (d *DiscreteValueOutOfListAnalyser) GetPodsOutOfBounds() ([]*kapiv1.Pod, error) {
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
	countersByPods, err := d.analyser.doAnalysis()
	if err != nil {
		return nil, err
	}

	//check if the key is GlobalKeyQuery that means that the result is applicable to all pods
	if len(countersByPods) == 1 {
		if v, ok := countersByPods[GlobalQueryKey]; ok {
			for _, pod := range podByName {
				countersByPods[pod.Name] = v
			}
			delete(countersByPods, GlobalQueryKey)
		}
	}

	for podName, counter := range countersByPods {
		_, found := podWithNoTraffic[podName]
		if found {
			continue
		}

		sum := counter.ok + counter.ko
		if sum >= 1 {
			ratio := counter.ko * 100 / sum
			if ratio > d.ConfigSpecific.TolerancePercent {
				if p, ok := podByName[podName]; ok {
					// Only keeping known pod with ratio superior to Tolerance
					result = append(result, p)
				}
			}
		}
	}
	return result, nil
}
