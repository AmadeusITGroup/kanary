package utils

import (
	"testing"
	"time"

	appsv1beta1 "k8s.io/api/apps/v1beta1"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

func WaitForFuncOnDeployment(t *testing.T, kubeclient kubernetes.Interface, namespace, name string, f func(dep *appsv1beta1.Deployment) (bool, error), retryInterval, timeout time.Duration) error {
	err := wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		deployment, err := kubeclient.AppsV1beta1().Deployments(namespace).Get(name, metav1.GetOptions{IncludeUninitialized: true})
		if err != nil {
			if apierrors.IsNotFound(err) {
				t.Logf("Waiting for availability of %s deployment\n", name)
				return false, nil
			}
			return false, err
		}

		ok, err := f(deployment)
		t.Logf("Waiting for condition function to be true ok for %s deployment (%t/%v)\n", name, ok, err)
		return ok, err
	})
	return err
}
