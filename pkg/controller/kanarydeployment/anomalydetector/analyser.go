package anomalydetector

import (
	"github.com/go-logr/logr"
	kapiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	kv1 "k8s.io/client-go/listers/core/v1"
)

//AnomalyDetector returns the list of pods that do not behave correctly according to the configuration
type AnomalyDetector interface {
	GetPodsOutOfBounds() ([]*kapiv1.Pod, error)
}

//Config generic part of the configuration for anomalyDetector
type Config struct {
	Selector      labels.Selector
	PodLister     kv1.PodNamespaceLister
	Logger        logr.Logger
	ExclusionFunc func(*kapiv1.Pod) (bool, error)
}

var _ AnomalyDetector = &Fake{}

//Fake should be used in test to mock an AnomalyDetector
type Fake struct {
	Pods []*kapiv1.Pod
	Err  error
}

//GetPodsOutOfBounds implements AnomalyDetector
func (f *Fake) GetPodsOutOfBounds() ([]*kapiv1.Pod, error) {
	return f.Pods, f.Err
}

//FakeFactory create a fake anomaly detector factory that return a Fake anomaly detector
func FakeFactory(p []*kapiv1.Pod, e error) Factory {
	return func(cfg FactoryConfig) (AnomalyDetector, error) {
		return &Fake{
			Pods: p, Err: e,
		}, nil
	}
}
