package e2e

import (
	goctx "context"
	"fmt"
	"math/rand"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1beta1"
	v2beta1 "k8s.io/api/autoscaling/v2beta1"
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

const (
	lineV0 = "while true; do echo 'v0'; sleep 5; done"
	lineV1 = "while true; do echo 'v1'; sleep 5; done"
)

var (
	commandV0 = []string{"/bin/sh", "-c", lineV0}
	commandV1 = []string{"/bin/sh", "-c", lineV1}
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
		t.Run("PromqlInvalidation", PromqlInvalidation)
		t.Run("Init-Kanary", InitKanaryDeploymentInstance)
		t.Run("Manual-Validation", ManualValidationAfterDeadline)
		t.Run("Manual-Invalidation", ManualInvalidationAfterDeadline)
		t.Run("DepLabelWatch-Invalid", InvalidationWithDeploymentLabels)
		t.Run("HPAcreation", HPAcreation)
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
	canaryName := name + "-kanary-" + name

	newService := newService(namespace, serviceName, map[string]string{"app": name})
	err = f.Client.Create(goctx.TODO(), newService, &framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	if err != nil {
		t.Fatal(err)
	}

	newDeployment := newDeployment(namespace, name, "busybox", "latest", commandV0, replicas)
	err = f.Client.Create(goctx.TODO(), newDeployment, &framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	if err != nil {
		t.Fatal(err)
	}

	err = e2eutil.WaitForDeployment(t, f.KubeClient, namespace, deploymentName, int(replicas), retryInterval, timeout)
	if err != nil {
		t.Fatal(err)
	}

	// Init KanaryDeployment with defaulted Strategy (desactiviated)
	newKD := newKanaryDeployment(namespace, name, deploymentName, serviceName, "busybox", "latest", commandV1, replicas, nil, nil, nil)
	err = f.Client.Create(goctx.TODO(), newKD, &framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	if err != nil {
		t.Fatal(err)
	}

	// wait for defaulting by the operator and then check kanary deployment is scaled to 1
	err = e2eutil.WaitForDeployment(t, f.KubeClient, namespace, canaryName, 1, retryInterval, timeout)
	if err != nil {
		t.Fatal(err)
	}

	// valid the kanaryDeployment
	err = utils.UpdateKanaryDeploymentFunc(f, namespace, name, func(k *kanaryv1alpha1.KanaryDeployment) {
		if k.Spec.Validation.Manual == nil {
			k.Spec.Validation.Manual = &kanaryv1alpha1.KanaryDeploymentSpecValidationManual{}
		}
		k.Spec.Validation.Manual.Status = kanaryv1alpha1.ValidKanaryDeploymentSpecValidationManualStatus
	}, retryInterval, timeout)
	if err != nil {
		t.Fatal(err)
	}

	// test if the canary deployment have been updated
	isUpdated := func(dep *appsv1.Deployment) (bool, error) {
		if !(len(dep.Spec.Template.Spec.Containers) > 0 && dep.Spec.Template.Spec.Containers[0].Command[2] == lineV1) {
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

func ManualValidationAfterDeadline(t *testing.T) {
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
	serviceName := ""
	canaryName := name + "-kanary-" + name

	newDeployment := newDeployment(namespace, name, "busybox", "latest", commandV0, replicas)
	err = f.Client.Create(goctx.TODO(), newDeployment, &framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	if err != nil {
		t.Fatal(err)
	}

	err = e2eutil.WaitForDeployment(t, f.KubeClient, namespace, deploymentName, int(replicas), retryInterval, timeout)
	if err != nil {
		t.Fatal(err)
	}

	validationConfig := &kanaryv1alpha1.KanaryDeploymentSpecValidation{
		ValidationPeriod: &metav1.Duration{Duration: 20 * time.Second},
		Manual: &kanaryv1alpha1.KanaryDeploymentSpecValidationManual{
			StatusAfterDealine: kanaryv1alpha1.ValidKanaryDeploymentSpecValidationManualDeadineStatus,
		},
	}
	newKD := newKanaryDeployment(namespace, name, deploymentName, serviceName, "busybox", "latest", commandV1, replicas, nil, nil, validationConfig)
	err = f.Client.Create(goctx.TODO(), newKD, &framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	if err != nil {
		t.Fatal(err)
	}

	// canary deployment is replicas is setted to 1.
	err = e2eutil.WaitForDeployment(t, f.KubeClient, namespace, canaryName, 1, retryInterval, timeout)
	if err != nil {
		t.Fatal(err)
	}

	// test if the deployment have been updated
	isUpdated := func(dep *appsv1.Deployment) (bool, error) {
		if !(len(dep.Spec.Template.Spec.Containers) > 0 && dep.Spec.Template.Spec.Containers[0].Command[2] == lineV1) {
			return false, nil
		}

		if dep.Status.UpdatedReplicas == dep.Status.AvailableReplicas {
			return true, nil
		}
		return false, nil
	}

	// check the update on the master deployment
	err = utils.WaitForFuncOnDeployment(t, f.KubeClient, namespace, name, isUpdated, retryInterval, 2*timeout)
	if err != nil {
		t.Fatal(err)
	}
}

func ManualInvalidationAfterDeadline(t *testing.T) {
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
	canaryName := name + "-kanary-" + name

	newService := newService(namespace, serviceName, map[string]string{"app": name})
	err = f.Client.Create(goctx.TODO(), newService, &framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	if err != nil {
		t.Fatal(err)
	}

	newDeployment := newDeployment(namespace, name, "busybox", "latest", commandV0, replicas)
	err = f.Client.Create(goctx.TODO(), newDeployment, &framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	if err != nil {
		t.Fatal(err)
	}

	err = e2eutil.WaitForDeployment(t, f.KubeClient, namespace, deploymentName, int(replicas), retryInterval, timeout)
	if err != nil {
		t.Fatal(err)
	}

	invalidationConfig := &kanaryv1alpha1.KanaryDeploymentSpecValidation{
		ValidationPeriod: &metav1.Duration{Duration: 50 * time.Second},
		Manual: &kanaryv1alpha1.KanaryDeploymentSpecValidationManual{
			StatusAfterDealine: kanaryv1alpha1.InvalidKanaryDeploymentSpecValidationManualDeadineStatus,
		},
	}
	trafficConfig := &kanaryv1alpha1.KanaryDeploymentSpecTraffic{
		Source: kanaryv1alpha1.ServiceKanaryDeploymentSpecTrafficSource,
	}
	newKD := newKanaryDeployment(namespace, name, deploymentName, serviceName, "busybox", "latest", commandV1, replicas, nil, trafficConfig, invalidationConfig)
	err = f.Client.Create(goctx.TODO(), newKD, &framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	if err != nil {
		t.Fatal(err)
	}

	// canary deployment is replicas is setted to 1.
	err = e2eutil.WaitForDeployment(t, f.KubeClient, namespace, canaryName, 1, retryInterval, timeout)
	if err != nil {
		t.Fatal(err)
	}

	// wait that the canary pod is behind the service
	err = utils.WaitForEndpointsCount(t, f.KubeClient, namespace, serviceName, 4, retryInterval, timeout)
	if err != nil {
		t.Fatal(err)
	}

	err = utils.WaitForInvalidStatusOnKanaryDeployment(t, f.Client, namespace, name, retryInterval, timeout)
	if err != nil {
		t.Fatal(err)
	}
	// check that pods are not anymore behind the service
	err = utils.WaitForEndpointsCount(t, f.KubeClient, namespace, serviceName, 3, retryInterval, timeout)
	if err != nil {
		t.Fatal(err)
	}

}

func InvalidationWithDeploymentLabels(t *testing.T) {
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
	canaryName := name + "-kanary-" + name

	newService := newService(namespace, serviceName, map[string]string{"app": name})
	err = f.Client.Create(goctx.TODO(), newService, &framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	if err != nil {
		t.Fatal(err)
	}

	newDeployment := newDeployment(namespace, name, "busybox", "latest", commandV0, replicas)
	err = f.Client.Create(goctx.TODO(), newDeployment, &framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	if err != nil {
		t.Fatal(err)
	}

	err = e2eutil.WaitForDeployment(t, f.KubeClient, namespace, deploymentName, int(replicas), retryInterval, timeout)
	if err != nil {
		t.Fatal(err)
	}

	mapFailed := map[string]string{"failed": "true"}

	invalidationConfig := &kanaryv1alpha1.KanaryDeploymentSpecValidation{
		ValidationPeriod: &metav1.Duration{Duration: 2 * time.Minute},
		LabelWatch: &kanaryv1alpha1.KanaryDeploymentSpecValidationLabelWatch{
			DeploymentInvalidationLabels: &metav1.LabelSelector{MatchLabels: mapFailed},
		},
	}
	trafficConfig := &kanaryv1alpha1.KanaryDeploymentSpecTraffic{
		Source: kanaryv1alpha1.BothKanaryDeploymentSpecTrafficSource,
	}
	newKD := newKanaryDeployment(namespace, name, deploymentName, serviceName, "busybox", "latest", commandV1, replicas, nil, trafficConfig, invalidationConfig)
	err = f.Client.Create(goctx.TODO(), newKD, &framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	if err != nil {
		t.Fatal(err)
	}

	// canary deployment is replicas is setted to 1.
	err = e2eutil.WaitForDeployment(t, f.KubeClient, namespace, canaryName, 1, retryInterval, timeout)
	if err != nil {
		t.Fatal(err)
	}

	err = utils.WaitForEndpointsCount(t, f.KubeClient, namespace, serviceName, 4, retryInterval, timeout)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("Update deployement kanary with failed label")
	addLabelsFunc := func(d *appsv1.Deployment) {
		if d.Labels == nil {
			d.Labels = map[string]string{}
		}
		for key, val := range mapFailed {
			d.Labels[key] = val
		}
	}
	err = updateDeploymentFunc(f, canaryName, namespace, addLabelsFunc)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Start checking KanaryDeployment status")

	err = utils.WaitForInvalidStatusOnKanaryDeployment(t, f.Client, namespace, name, retryInterval, 2*timeout)
	if err != nil {
		t.Fatal(err)
	}

	err = utils.WaitForEndpointsCount(t, f.KubeClient, namespace, serviceName, 3, retryInterval, timeout)
	if err != nil {
		t.Fatal(err)
	}
}

func HPAcreation(t *testing.T) {
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
	canaryName := name + "-kanary-" + name

	newService := newService(namespace, serviceName, map[string]string{"app": name})
	err = f.Client.Create(goctx.TODO(), newService, &framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	if err != nil {
		t.Fatal(err)
	}

	newDeployment := newDeployment(namespace, name, "busybox", "latest", commandV0, replicas)
	err = f.Client.Create(goctx.TODO(), newDeployment, &framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	if err != nil {
		t.Fatal(err)
	}

	err = e2eutil.WaitForDeployment(t, f.KubeClient, namespace, deploymentName, int(replicas), retryInterval, timeout)
	if err != nil {
		t.Fatal(err)
	}

	// Init KanaryDeployment with defaulted Strategy (desactiviated)
	scaleSpec := &kanaryv1alpha1.KanaryDeploymentSpecScale{
		HPA: &kanaryv1alpha1.HorizontalPodAutoscalerSpec{
			MinReplicas: kanaryv1alpha1.NewInt32(1),
			MaxReplicas: int32(3),
		},
	}

	newKD := newKanaryDeployment(namespace, name, deploymentName, serviceName, "busybox", "latest", commandV1, replicas, scaleSpec, nil, nil)
	err = f.Client.Create(goctx.TODO(), newKD, &framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	if err != nil {
		t.Fatal(err)
	}

	HPAValidationFunc := func(hpa *v2beta1.HorizontalPodAutoscaler) (bool, error) {
		if hpa.Status.CurrentReplicas != int32(1) {
			return false, nil
		}
		return true, nil
	}
	err = utils.WaitForFuncOnHPA(t, f.KubeClient, namespace, canaryName, HPAValidationFunc, retryInterval, timeout)
	if err != nil {
		t.Fatal(err)
	}

	// canary deployment is replicas is setted to 0 in deactivated mode.
	err = e2eutil.WaitForDeployment(t, f.KubeClient, namespace, canaryName, 1, retryInterval, timeout)
	if err != nil {
		t.Fatal(err)
	}
}

func updateDeploymentFunc(f *framework.Framework, name, namespace string, updateFunc func(kd *appsv1.Deployment)) error {
	kd := &appsv1.Deployment{}
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

func newDeployment(namespace, name, image, tag string, command []string, replicas int32) *appsv1.Deployment {
	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: appsv1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: newDeploymentSpec(name, image, tag, command, replicas),
	}
}

func newDeploymentSpec(name, image, tag string, command []string, replicas int32) appsv1.DeploymentSpec {
	return appsv1.DeploymentSpec{
		Replicas: &replicas,
		Selector: &metav1.LabelSelector{
			MatchLabels: map[string]string{"app": name, "version": tag},
		},
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{"app": name, "version": tag},
			},

			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  "container1",
						Image: fmt.Sprintf("%s:%s", image, tag),
						Ports: []corev1.ContainerPort{
							{ContainerPort: 80},
						},
						Command:         command,
						ImagePullPolicy: corev1.PullIfNotPresent,
					},
				},
			},
		},
	}
}

func newKanaryDeployment(namespace, name, deploymentName, serviceName, image, tag string, command []string, replicas int32, scale *kanaryv1alpha1.KanaryDeploymentSpecScale, traffic *kanaryv1alpha1.KanaryDeploymentSpecTraffic, validation *kanaryv1alpha1.KanaryDeploymentSpecValidation) *kanaryv1alpha1.KanaryDeployment {
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
				Spec: newDeploymentSpec(name, image, tag, command, replicas),
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
