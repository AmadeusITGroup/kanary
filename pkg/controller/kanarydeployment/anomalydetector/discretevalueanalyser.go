package anomalydetector

import (
	"fmt"

	"github.com/amadeusitgroup/kanary/pkg/pod"
	"github.com/go-logr/logr"
	kapiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	kv1 "k8s.io/client-go/listers/core/v1"
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

//DiscreteValueOutOfListAnalyser anomalyDetector that check the ratio of good/bad value and return the pods that exceed a given threshold for that ratio
type DiscreteValueOutOfListAnalyser struct {
	TolerancePercent uint

	selector  labels.Selector
	analyser  discreteValueAnalyser
	podLister kv1.PodNamespaceLister
	logger    logr.Logger
}

//GetPodsOutOfBounds implements interface AnomalyDetector
func (d *DiscreteValueOutOfListAnalyser) GetPodsOutOfBounds() ([]*kapiv1.Pod, error) {
	listOfPods, err := d.podLister.List(d.selector)
	if err != nil {
		return nil, fmt.Errorf("can't list pods, error:%v", err)
	}

	listOfPods, err = pod.PurgeNotReadyPods(listOfPods)
	if err != nil {
		return nil, fmt.Errorf("can't purge not ready pods, error:%v", err)
	}
	podByName, podWithNoTraffic, err := PodByName(listOfPods, nil)
	if err != nil {
		return nil, err
	}
	result := []*kapiv1.Pod{}
	countersByPods, err := d.analyser.doAnalysis()
	if err != nil {
		return nil, err
	}

	for podName, counter := range countersByPods {
		_, found := podWithNoTraffic[podName]
		if found {
			continue
		}

		sum := counter.ok + counter.ko
		if sum >= 1 {
			ratio := counter.ko * 100 / sum
			if ratio > d.TolerancePercent {
				if p, ok := podByName[podName]; ok {
					// Only keeping known pod with ratio superior to Tolerance
					result = append(result, p)
				}
			}
		}
	}
	return result, nil
}
