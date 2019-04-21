package anomalydetector

import kapiv1 "k8s.io/api/core/v1"

// ContainsString checks if the slice has the contains value in it.
func ContainsString(slice []string, contains string) bool {
	for _, value := range slice {
		if value == contains {
			return true
		}
	}
	return false
}

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
