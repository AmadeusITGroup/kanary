package utils

import (
	"context"
	"testing"
	"time"

	appsv1beta1 "k8s.io/api/apps/v1beta1"
	"k8s.io/api/autoscaling/v2beta1"
	corev1 "k8s.io/api/core/v1"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"

	framework "github.com/operator-framework/operator-sdk/pkg/test"

	dynclient "sigs.k8s.io/controller-runtime/pkg/client"

	kanaryv1alpha1 "github.com/amadeusitgroup/kanary/pkg/apis/kanary/v1alpha1"
	utilsctrl "github.com/amadeusitgroup/kanary/pkg/controller/kanarydeployment/utils"
)

// WaitForFuncOnDeployment used to wait a valid condition on a Deployment
func WaitForFuncOnDeployment(t *testing.T, kubeclient kubernetes.Interface, namespace, name string, f func(dep *appsv1beta1.Deployment) (bool, error), retryInterval, timeout time.Duration) error {
	return wait.Poll(retryInterval, timeout, func() (bool, error) {
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
}

//EndpointCheckFunc function to perform checks on endpoint object
type EndpointCheckFunc func(*corev1.Endpoints) (bool, error)

// WaitForFuncOnEndpoints used to wait a valid condition on Endpoints
func WaitForFuncOnEndpoints(t *testing.T, kubeclient kubernetes.Interface, namespace, name string, f EndpointCheckFunc, retryInterval, timeout time.Duration) error {
	return wait.Poll(retryInterval, timeout, func() (bool, error) {
		eps, err := kubeclient.CoreV1().Endpoints(namespace).Get(name, metav1.GetOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) {
				t.Logf("Waiting for availability of %s endpoint\n", name)
				return false, nil
			}
			return false, err
		}

		ok, err := f(eps)
		t.Logf("Waiting for condition function to be true ok for %s endpoint (%t/%v)\n", name, ok, err)
		return ok, err
	})
}

// WaitForFuncOnKanaryDeployment used to wait a valid condition on a KanaryDeployment
func WaitForFuncOnKanaryDeployment(t *testing.T, client framework.FrameworkClient, namespace, name string, f func(kd *kanaryv1alpha1.KanaryDeployment) (bool, error), retryInterval, timeout time.Duration) error {
	return wait.Poll(retryInterval, timeout, func() (bool, error) {
		objKey := dynclient.ObjectKey{
			Namespace: namespace,
			Name:      name,
		}
		kanaryDeployment := &kanaryv1alpha1.KanaryDeployment{}
		err := client.Get(context.TODO(), objKey, kanaryDeployment)
		if err != nil {
			if apierrors.IsNotFound(err) {
				t.Logf("Waiting for availability of %s kanaryDeployment\n", name)
				return false, nil
			}
			return false, err
		}

		ok, err := f(kanaryDeployment)
		t.Logf("Waiting for condition function to be true ok for %s kanaryDeployment (%t/%v)\n", name, ok, err)
		return ok, err
	})
}

func checkInvalidStatus(kd *kanaryv1alpha1.KanaryDeployment) (bool, error) {
	if utilsctrl.IsKanaryDeploymentFailed(&kd.Status) {
		return true, nil
	}
	return false, nil
}

// WaitForInvalidStatusOnKanaryDeployment used to wait an invalidated status on a KanaryDeployment
func WaitForInvalidStatusOnKanaryDeployment(t *testing.T, client framework.FrameworkClient, namespace, name string, retryInterval, timeout time.Duration) error {
	return WaitForFuncOnKanaryDeployment(t, client, namespace, name, checkInvalidStatus, retryInterval, timeout)
}

// WaitForFuncOnHPA used to wait a valid condition on a HPA
func WaitForFuncOnHPA(t *testing.T, kubeclient kubernetes.Interface, namespace, name string, f func(hpa *v2beta1.HorizontalPodAutoscaler) (bool, error), retryInterval, timeout time.Duration) error {
	return wait.Poll(retryInterval, timeout, func() (bool, error) {
		hpa, err := kubeclient.AutoscalingV2beta1().HorizontalPodAutoscalers(namespace).Get(name, metav1.GetOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) {
				t.Logf("Waiting for availability of %s HorizontalPodAutoscaler\n", name)
				return false, nil
			}
			return false, err
		}

		ok, err := f(hpa)
		t.Logf("Waiting for condition function to be true ok for %s HorizontalPodAutoscaler (%t/%v)\n", name, ok, err)
		return ok, err
	})
}

//CheckEndpoints check if the count of endpoint is the expexted one
func CheckEndpoints(t *testing.T, eps *corev1.Endpoints, wantedPod int) (bool, error) {
	nbPod := 0
	for _, sub := range eps.Subsets {
		nbPod += len(sub.Addresses)
	}
	if wantedPod != nbPod {
		t.Logf("checkEndpoints %d-%d", wantedPod, nbPod)
		return false, nil
	}
	return true, nil
}

//WaitForEndpointsCount wait for the endpoint to reach a given count
func WaitForEndpointsCount(t *testing.T, kubeclient kubernetes.Interface, namespace, name string, endpointCount int, retryInterval, timeout time.Duration) error {
	f := func(eps *corev1.Endpoints) (bool, error) {
		return CheckEndpoints(t, eps, endpointCount)
	}
	return WaitForFuncOnEndpoints(t, kubeclient, namespace, name, f, retryInterval, timeout)
}
