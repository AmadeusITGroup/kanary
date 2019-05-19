package e2e

import (
	goctx "context"
	"fmt"
	"testing"
	"time"

	kanaryv1alpha1 "github.com/amadeusitgroup/kanary/pkg/apis/kanary/v1alpha1"
	utilskd "github.com/amadeusitgroup/kanary/pkg/controller/kanarydeployment/utils"
	"github.com/amadeusitgroup/kanary/test/e2e/utils"
	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func ScheduledKanaryDeploymentIn30s(t *testing.T) {
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

	newService := utils.NewService(namespace, serviceName, map[string]string{"app": name})
	err = f.Client.Create(goctx.TODO(), newService, &framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	if err != nil {
		t.Fatal(err)
	}

	newDeployment := utils.NewDeployment(namespace, name, "busybox", "latest", commandV0, replicas)
	err = f.Client.Create(goctx.TODO(), newDeployment, &framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	if err != nil {
		t.Fatal(err)
	}

	err = e2eutil.WaitForDeployment(t, f.KubeClient, namespace, deploymentName, int(replicas), retryInterval, timeout)
	if err != nil {
		t.Fatal(err)
	}

	// Init KanaryDeployment with defaulted Strategy (desactiviated)
	validationSpec := &kanaryv1alpha1.KanaryDeploymentSpecValidationList{
		ValidationPeriod: &metav1.Duration{Duration: 20 * time.Second},
	}

	newKD := utils.NewKanaryDeployment(namespace, name, deploymentName, serviceName, "busybox", "latest", commandV1, replicas, nil, nil, validationSpec)
	schedTime := time.Now().Add(30 * time.Second)
	newKD.Spec.Schedule = schedTime.Format(time.RFC3339)
	err = f.Client.Create(goctx.TODO(), newKD, &framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	if err != nil {
		t.Fatal(err)
	}

	err = utils.WaitForFuncOnKanaryDeployment(t, f.Client, namespace, name, func(kd *kanaryv1alpha1.KanaryDeployment) (bool, error) {
		return utilskd.IsKanaryDeploymentScheduled(&kd.Status), nil
	}, 5*time.Second, time.Minute)
	if err != nil {
		t.Fatal(err)
	}

	err = utils.WaitForFuncOnKanaryDeployment(t, f.Client, namespace, name, func(kd *kanaryv1alpha1.KanaryDeployment) (bool, error) {
		return utilskd.IsKanaryDeploymentValidationRunning(&kd.Status), nil
	}, time.Second, time.Minute)
	if err != nil {
		t.Fatal(err)
	}

	if time.Now().Before(schedTime) {
		t.Fatal(fmt.Errorf("The kanary deployment was scheduled to early compare to request"))
	}
}
