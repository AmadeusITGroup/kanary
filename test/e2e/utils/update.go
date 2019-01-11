package utils

import (
	goctx "context"
	"time"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"

	framework "github.com/operator-framework/operator-sdk/pkg/test"

	kanaryv1alpha1 "github.com/amadeusitgroup/kanary/pkg/apis/kanary/v1alpha1"
)

// UpdateKanaryDeploymentFunc used to update a KanaryDeployment with retry and timeout policy
func UpdateKanaryDeploymentFunc(f *framework.Framework, namespace, name string, updateFunc func(kd *kanaryv1alpha1.KanaryDeployment), retryInterval, timeout time.Duration) error {
	return wait.Poll(retryInterval, timeout, func() (bool, error) {
		kd := &kanaryv1alpha1.KanaryDeployment{}
		if err := f.Client.Get(goctx.TODO(), types.NamespacedName{Name: name, Namespace: namespace}, kd); err != nil {
			return false, nil
		}

		updateKD := kd.DeepCopy()
		updateFunc(updateKD)
		if err := f.Client.Update(goctx.TODO(), updateKD); err != nil {
			return false, nil
		}
		return true, nil
	})

}
