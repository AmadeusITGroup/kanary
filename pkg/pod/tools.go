package pod

import (
	kapiv1 "k8s.io/api/core/v1"
)

//FilterOut remove from the slice pods for which exclude function return true
func FilterOut(slice []*kapiv1.Pod, exclude func(*kapiv1.Pod) (bool, error)) ([]*kapiv1.Pod, error) {
	b := []*kapiv1.Pod{}
	for _, x := range slice {
		ok, err := exclude(x)
		if err != nil {
			return b, err
		}
		if !ok {
			b = append(b, x)
		}
	}
	return b, nil
}

//PurgeNotReadyPods keep only pods that are ready inside the slice
func PurgeNotReadyPods(pods []*kapiv1.Pod) ([]*kapiv1.Pod, error) {
	return FilterOut(pods, func(a *kapiv1.Pod) (bool, error) { return !IsReady(a), nil })
}

//IsReady check if the pod is Ready
func IsReady(p *kapiv1.Pod) bool {
	if p.Status.Phase == kapiv1.PodRunning {
		for _, c := range p.Status.Conditions {
			if c.Type == kapiv1.PodReady {
				return c.Status == kapiv1.ConditionTrue
			}
		}
	}
	return false
}

//KeepRunningPods check if the pod is Ready
func KeepRunningPods(pods []*kapiv1.Pod) ([]*kapiv1.Pod, error) {
	return FilterOut(pods, func(a *kapiv1.Pod) (bool, error) { return a.Status.Phase != kapiv1.PodRunning, nil })
}

// ExcludeFromSlice return a slice of Pod that is not present in another slice
func ExcludeFromSlice(fromSlice, inSlice []*kapiv1.Pod) []*kapiv1.Pod {
	output := []*kapiv1.Pod{}
	for _, from := range fromSlice {
		found := false
		for _, in := range inSlice {
			if from.Name == in.Name && from.Namespace == in.Namespace {
				found = true
				break
			}
		}
		if !found {
			output = append(output, from)
		}
	}
	return output
}
