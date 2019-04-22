package utils_test

// Functions in this package are for unittest usage only
// FOR TEST PURPOSE ONLY

//folder name "test" and package "utils_test" are different on purpose: avoid automatic import by go_import and force import alias.

import (
	"sync"
	"testing"
	"time"

	kapiv1 "k8s.io/api/core/v1"
	kv1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
)

//ValidateTestSequence is a test helper to validate the sequence of steps
// FOR TEST PURPOSE ONLY
func ValidateTestSequence(wg *sync.WaitGroup, t *testing.T, duration time.Duration, sequenceTitle string, closingChannels []chan struct{}) {
	wg.Add(1)
	go func() {
		sequenceCompleted := make(chan struct{})
		go func() {
			defer wg.Done()
			timeout := time.After(duration)
			select {
			case <-timeout:
				t.Errorf("The sequence %s did not complete in %f s", sequenceTitle, duration.Seconds())
			case <-sequenceCompleted:
				return
			}
		}()

		for _, c := range closingChannels {
			<-c
		}
		close(sequenceCompleted)
	}()
}

type testStep struct {
	sync.Mutex
	c        chan struct{}
	doneOnce bool
}

func (ts *testStep) getChan() <-chan struct{} {
	ts.Lock()
	defer ts.Unlock()
	return ts.c
}

//TestStepSequence sequence of test steps
type TestStepSequence struct {
	name     string
	t        *testing.T
	steps    []*testStep
	duration time.Duration
}

//Len return the number of steps in the sequence
func (es *TestStepSequence) Len() int {
	return len(es.steps)
}

//PassOnlyOnce to be called when a step in the sequence is considered as passed and can't be passed a second time else (t.Fatal)
func (es *TestStepSequence) PassOnlyOnce(step int) {
	defer func() {
		if r := recover(); r != nil {
			es.t.Fatalf("Recovered in Passing step: %s", r)
			return
		}
	}()

	if step > len(es.steps) {
		es.t.Fatalf("Step out of bound")
		return
	}

	es.steps[step].Lock()
	defer es.steps[step].Unlock()
	if es.steps[step].doneOnce {
		es.t.Fatalf("Step pass 2 times, expecting once only")
		return
	}

	close(es.steps[step].c)
	es.steps[step].c = nil
	es.steps[step].doneOnce = true
}

//PassAtLeastOnce to be called when a step in the sequence is considered as passed qnd can't be passed a second time
func (es *TestStepSequence) PassAtLeastOnce(step int) {
	defer func() {
		if r := recover(); r != nil {
			es.t.Fatalf("Recovered in Passing step: %s", r)
			return
		}
	}()

	if step > len(es.steps) {
		es.t.Fatalf("Step out of bound")
		return
	}

	es.steps[step].Lock()
	defer es.steps[step].Unlock()

	if es.steps[step].doneOnce {
		return
	}
	close(es.steps[step].c)
	es.steps[step].c = nil
	es.steps[step].doneOnce = true
}

//Completed check that all step of the sequence have been completed.
func (es *TestStepSequence) Completed() bool {
	for _, s := range es.steps {
		if !s.doneOnce {
			return false
		}
	}
	return true
}

//ValidateTestSequence validate that the sequence is completed in order in the given time
func (es *TestStepSequence) ValidateTestSequence(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		sequenceCompleted := make(chan struct{})
		go func() {
			defer wg.Done()
			timeout := time.After(es.duration)
			select {
			case <-timeout:
				es.t.Errorf("The sequence %s did not complete in %f s", es.name, es.duration.Seconds())
			case <-sequenceCompleted:
				return
			}
		}()

		for _, step := range es.steps {
			<-step.getChan()
		}
		close(sequenceCompleted)
	}()
}

//ValidateTestSequenceNoOrder validate that the sequence is completed in order in the given time
func (es *TestStepSequence) ValidateTestSequenceNoOrder(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		sequenceCompleted := make(chan struct{})
		go func() {
			defer wg.Done()
			timeout := time.After(es.duration)
			select {
			case <-timeout:
				es.t.Errorf("The sequence %s did not complete in %f s", es.name, es.duration.Seconds())
			case <-sequenceCompleted:
				return
			}
		}()
		var wgIn sync.WaitGroup
		for _, step := range es.steps {
			wgIn.Add(1)
			go func(ts *testStep) {
				defer wgIn.Done()
				<-ts.getChan()
			}(step)
		}
		wgIn.Wait()
		close(sequenceCompleted)
	}()
}

//NewTestSequence Create a test sequence
func NewTestSequence(t *testing.T, registrationName string, count int, duration time.Duration) *TestStepSequence {
	s := &TestStepSequence{
		name:     registrationName,
		steps:    make([]*testStep, count),
		t:        t,
		duration: duration,
	}
	for i := range s.steps {
		s.steps[i] = &testStep{
			c: make(chan struct{}),
		}
	}

	if _, ok := MapOfSequences[s.name]; ok {
		t.Fatalf("Multiple definition of sequence %s", s.name)
		return nil
	}
	MapOfSequences[s.name] = s
	return s
}

//GetTestSequence retrieve test sequence
//Should be called
func GetTestSequence(t *testing.T, registrationName string) *TestStepSequence {
	//don't use t.Name because the name change depending if the testcase is running or not.
	s, ok := MapOfSequences[registrationName]
	if !ok {
		t.Fatalf("Undefined test sequence %s", registrationName)
	}
	return s
}

//MapOfSequences represent a TestStepSequence map
var MapOfSequences = map[string]*TestStepSequence{}

//NewTestPodLister create a new PodLister.
// FOR TEST PURPOSE ONLY
func NewTestPodLister(pods []*kapiv1.Pod) kv1.PodLister {
	index := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{})
	for _, p := range pods {
		if err := index.Add(p); err != nil {
			continue
		}
	}
	return kv1.NewPodLister(index)
}

//NewTestPodNamespaceLister create a new PodNamespaceLister.
// FOR TEST PURPOSE ONLY
func NewTestPodNamespaceLister(pods []*kapiv1.Pod, namespace string) kv1.PodNamespaceLister {
	return NewTestPodLister(pods).Pods(namespace)
}

//PodGen generate a pod with some label and status
// FOR TEST PURPOSE ONLY
func PodGen(name, namespace string, labels, annotations map[string]string, running, ready bool) *kapiv1.Pod {
	p := kapiv1.Pod{}
	p.Name = name
	p.Namespace = namespace
	p.SetLabels(labels)
	p.SetAnnotations(annotations)

	if running {
		p.Status = kapiv1.PodStatus{Phase: kapiv1.PodRunning}
		if ready {
			p.Status.Conditions = []kapiv1.PodCondition{{Type: kapiv1.PodReady, Status: kapiv1.ConditionTrue}}
		} else {
			p.Status.Conditions = []kapiv1.PodCondition{{Type: kapiv1.PodReady, Status: kapiv1.ConditionFalse}}
		}
	} else {
		p.Status = kapiv1.PodStatus{Phase: kapiv1.PodUnknown}
	}
	return &p
}

//TestPodControl test mock for podcontro
type TestPodControl struct {
	T                                        *testing.T
	Case                                     string
	FailOnUndefinedFunc                      bool
	InitBreakerAnnotationAndLabelFunc        func(name string, p *kapiv1.Pod) (*kapiv1.Pod, error)
	UpdateBreakerAnnotationAndLabelFunc      func(name string, strategy string, p *kapiv1.Pod) (*kapiv1.Pod, error)
	UpdateActivationLabelsAndAnnotationsFunc func(name string, p *kapiv1.Pod) (*kapiv1.Pod, error)
	UpdatePauseLabelsAndAnnotationsFunc      func(name string, p *kapiv1.Pod) (*kapiv1.Pod, error)
	RemoveBreakerAnnotationAndLabelFunc      func(p *kapiv1.Pod) (*kapiv1.Pod, error)
	KillPodFunc                              func(name string, p *kapiv1.Pod) error
}

//InitBreakerAnnotationAndLabel fake implementation for podcontrol
func (t *TestPodControl) InitBreakerAnnotationAndLabel(name string, p *kapiv1.Pod) (*kapiv1.Pod, error) {
	if t.InitBreakerAnnotationAndLabelFunc != nil {
		return t.InitBreakerAnnotationAndLabelFunc(name, p)
	}
	if t.FailOnUndefinedFunc {
		t.T.Errorf("UpdateBreakerAnnotationAndLabel should not be called in %s/%s", t.T.Name(), t.Case)
	}
	return nil, nil
}

//UpdateBreakerAnnotationAndLabel fake implementation for podcontrol
func (t *TestPodControl) UpdateBreakerAnnotationAndLabel(name string, strategy string, p *kapiv1.Pod) (*kapiv1.Pod, error) {
	if t.UpdateBreakerAnnotationAndLabelFunc != nil {
		return t.UpdateBreakerAnnotationAndLabelFunc(name, strategy, p)
	}
	if t.FailOnUndefinedFunc {
		t.T.Errorf("UpdateBreakerAnnotationAndLabel should not be called in %s/%s", t.T.Name(), t.Case)
	}
	return nil, nil
}

//UpdateActivationLabelsAndAnnotations fake implementation for podcontrol
func (t *TestPodControl) UpdateActivationLabelsAndAnnotations(name string, p *kapiv1.Pod) (*kapiv1.Pod, error) {
	if t.UpdateActivationLabelsAndAnnotationsFunc != nil {
		return t.UpdateActivationLabelsAndAnnotationsFunc(name, p)
	}
	if t.FailOnUndefinedFunc {
		t.T.Errorf("UpdateActivationLabelsAndAnnotationsFunc should not be called in %s/%s", t.T.Name(), t.Case)
	}
	return nil, nil
}

//UpdatePauseLabelsAndAnnotations fake implementation for podcontrol
func (t *TestPodControl) UpdatePauseLabelsAndAnnotations(name string, p *kapiv1.Pod) (*kapiv1.Pod, error) {
	if t.UpdatePauseLabelsAndAnnotationsFunc != nil {
		return t.UpdatePauseLabelsAndAnnotationsFunc(name, p)
	}
	if t.FailOnUndefinedFunc {
		t.T.Errorf("UpdatePauseLabelsAndAnnotationsFunc should not be called in %s/%s", t.T.Name(), t.Case)
	}

	return nil, nil
}

// RemoveBreakerAnnotationAndLabel fake implementatin for podcontrol
func (t *TestPodControl) RemoveBreakerAnnotationAndLabel(p *kapiv1.Pod) (*kapiv1.Pod, error) {
	if t.RemoveBreakerAnnotationAndLabelFunc != nil {
		return t.RemoveBreakerAnnotationAndLabelFunc(p)
	}
	if t.FailOnUndefinedFunc {
		t.T.Errorf("RemoveBreakerAnnotationAndLabelFunc should not be called in %s/%s", t.T.Name(), t.Case)
	}

	return nil, nil
}

//KillPod fake implementation for podcontrol
func (t *TestPodControl) KillPod(name string, p *kapiv1.Pod) error {
	if t.KillPodFunc != nil {
		return t.KillPodFunc(name, p)
	}
	if t.FailOnUndefinedFunc {
		t.T.Errorf("KillPod should not be called in %s/%s", t.T.Name(), t.Case)
	}
	return nil
}
