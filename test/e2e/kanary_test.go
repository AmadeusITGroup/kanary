package e2e

import (
	goctx "context"
	"fmt"
	"math/rand"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1beta1"
	corev1 "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"

	apis "github.com/amadeusitgroup/kanary/pkg/apis"
	kanaryv1alpha1 "github.com/amadeusitgroup/kanary/pkg/apis/kanary/v1alpha1"
	"github.com/amadeusitgroup/kanary/test/e2e/utils"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyz")

func RandStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

var (
	retryInterval        = time.Second * 5
	timeout              = time.Second * 60
	cleanupRetryInterval = time.Second * 1
	cleanupTimeout       = time.Second * 5
)

func TestKanary(t *testing.T) {
	kanaryList := &kanaryv1alpha1.KanaryDeploymentList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "KanaryDeployment",
			APIVersion: "kanary.k8s.io/v1alpha1",
		},
	}

	if err := framework.AddToFrameworkScheme(apis.AddToScheme, kanaryList); err != nil {
		t.Fatalf("failed to add custom resource scheme to framework: %v", err)
	}

	// run subtests
	t.Run("kanary-group", func(t *testing.T) {
		t.Run("Init-Kanary", InitKanaryDeploymentInstance)
	})
}

func InitKanaryDeploymentInstance(t *testing.T) {
	t.Parallel()
	f, ctx, err := InitKanaryOperator(t)
	defer ctx.Cleanup()
	if err != nil {
		t.Fatal(err)
	}

	namespace, err := ctx.GetNamespace()
	if err != nil {
		t.Fatal(fmt.Errorf("could not get namespace: %v", err))
	}
	name := RandStringRunes(6)
	replicas := int32(3)
	deploymentName := name
	serviceName := name
	canaryName := name + "-kanary"

	newService := newService(namespace, serviceName, map[string]string{"app": name})
	err = f.Client.Create(goctx.TODO(), newService, &framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	if err != nil {
		t.Fatal(err)
	}

	newDeployment := newDeployment(namespace, name, "nginx:1.15.4", replicas)
	err = f.Client.Create(goctx.TODO(), newDeployment, &framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	if err != nil {
		t.Fatal(err)
	}

	err = e2eutil.WaitForDeployment(t, f.KubeClient, namespace, deploymentName, int(replicas), retryInterval, timeout)
	if err != nil {
		t.Fatal(err)
	}

	// Init KanaryDeployment with defaulted Strategy (desactiviated)
	newKD := newKanaryDeployment(namespace, name, deploymentName, serviceName, "nginx:latest", replicas, nil, nil, nil)
	err = f.Client.Create(goctx.TODO(), newKD, &framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	if err != nil {
		t.Fatal(err)
	}

	// canary deployment is replicas is setted to 0 in deactivated mode.
	err = e2eutil.WaitForDeployment(t, f.KubeClient, namespace, canaryName, 1, retryInterval, timeout)
	if err != nil {
		t.Fatal(err)
	}

	// valid the kanaryDeployment
	utils.UpdateKanaryDeploymentFunc(f, namespace, name, func(k *kanaryv1alpha1.KanaryDeployment) {
		if k.Spec.Validation.Manual == nil {
			k.Spec.Validation.Manual = &kanaryv1alpha1.KanaryDeploymentSpecValidationManual{}
		}
		k.Spec.Validation.Manual.Status = kanaryv1alpha1.ValidKanaryDeploymentSpecValidationManualStatus
	}, retryInterval, timeout)

	// wait that the canary deployment scale to 1
	err = e2eutil.WaitForDeployment(t, f.KubeClient, namespace, canaryName, 1, retryInterval, timeout)
	if err != nil {
		t.Fatal(err)
	}

	// test if the canary deployment have been updated
	isUpdated := func(dep *appsv1.Deployment) (bool, error) {
		if !(len(dep.Spec.Template.Spec.Containers) > 0 && dep.Spec.Template.Spec.Containers[0].Image == "nginx:latest") {
			return false, nil
		}

		if dep.Status.UpdatedReplicas == dep.Status.AvailableReplicas {
			return true, nil
		}
		return false, nil
	}
	// check the update on the master deployment
	err = utils.WaitForFuncOnDeployment(t, f.KubeClient, namespace, name, isUpdated, retryInterval, timeout)
	if err != nil {
		t.Fatal(err)
	}
}

func updateDeploymentFunc(f *framework.Framework, name, namespace string, updateFunc func(kd *kanaryv1alpha1.KanaryDeployment)) error {
	kd := &kanaryv1alpha1.KanaryDeployment{}
	err := f.Client.Get(goctx.TODO(), types.NamespacedName{Name: name, Namespace: namespace}, kd)
	if err != nil {
		return err
	}

	updateKD := kd.DeepCopy()
	updateFunc(updateKD)
	err = f.Client.Update(goctx.TODO(), updateKD)
	if err != nil {
		return err
	}

	return nil
}

func InitKanaryOperator(t *testing.T) (*framework.Framework, *framework.TestCtx, error) {
	ctx := framework.NewTestCtx(t)
	err := ctx.InitializeClusterResources(&framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	if err != nil {
		t.Fatalf("failed to initialize cluster resources: %v", err)
	}
	t.Log("Initialized cluster resources")
	namespace, err := ctx.GetNamespace()
	if err != nil {
		t.Fatal(err)
	}
	// get global framework variables
	f := framework.Global
	// wait for memcached-operator to be ready
	err = e2eutil.WaitForDeployment(t, f.KubeClient, namespace, "kanary", 1, retryInterval, timeout)
	return f, ctx, err
}

func newDeployment(namespace, name, image string, replicas int32) *appsv1.Deployment {
	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: appsv1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: newDeploymentSpec(name, image, replicas),
	}
}

func newDeploymentSpec(name, image string, replicas int32) appsv1.DeploymentSpec {
	return appsv1.DeploymentSpec{
		Replicas: &replicas,
		Selector: &metav1.LabelSelector{
			MatchLabels: map[string]string{"app": name},
		},
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{"app": name},
			},

			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  "nginx",
						Image: image,
						Ports: []corev1.ContainerPort{
							{ContainerPort: 80},
						},
					},
				},
			},
		},
	}
}

func newKanaryDeployment(namespace, name, deploymentName, serviceName, image string, replicas int32, scale *kanaryv1alpha1.KanaryDeploymentSpecScale, traffic *kanaryv1alpha1.KanaryDeploymentSpecTraffic, validation *kanaryv1alpha1.KanaryDeploymentSpecValidation) *kanaryv1alpha1.KanaryDeployment {
	kd := &kanaryv1alpha1.KanaryDeployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "KanaryDeployment",
			APIVersion: kanaryv1alpha1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: kanaryv1alpha1.KanaryDeploymentSpec{
			ServiceName:    serviceName,
			DeploymentName: deploymentName,
			Template: kanaryv1alpha1.DeploymentTemplate{
				Spec: newDeploymentSpec(name, image, replicas),
			},
		},
	}

	if scale != nil {
		kd.Spec.Scale = *scale
	}
	if traffic != nil {
		kd.Spec.Traffic = *traffic
	}
	if validation != nil {
		kd.Spec.Validation = *validation
	}

	return kd
}

func newService(namespace, name string, labelsSelector map[string]string) *corev1.Service {
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: corev1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: labelsSelector,
			Ports: []corev1.ServicePort{
				{Port: 80},
			},
		},
	}
}
