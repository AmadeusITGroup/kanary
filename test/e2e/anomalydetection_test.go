package e2e

import (
	goctx "context"
	"fmt"
	"testing"
	"time"

	"github.com/amadeusitgroup/kanary/pkg/apis/kanary/v1alpha1"
	kanaryv1alpha1 "github.com/amadeusitgroup/kanary/pkg/apis/kanary/v1alpha1"
	utilskanary "github.com/amadeusitgroup/kanary/pkg/controller/kanarydeployment/utils"
	"github.com/amadeusitgroup/kanary/test/e2e/utils"
	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"
	"github.com/prometheus/client_golang/prometheus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func PromqlInvalidation(t *testing.T) {
	t.Parallel()

	//Preparation of environment
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

	//Run the prometheus server and gateway
	promHelper := utils.NewPromTestHelper(t, ctx, f, "Test_PromQlInvalidation", []prometheus.Collector{sampleHisto})
	promHelper.Run()
	defer promHelper.StopPushGateway()

	//Use to stop metric generation at the end of the test
	testTerminated := make(chan struct{})
	defer close(testTerminated)

	//Create the app Service
	newService := utils.NewService(namespace, serviceName, map[string]string{"app": name})
	err = f.Client.Create(goctx.TODO(), newService, &framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	if err != nil {
		t.Fatal(err)
	}

	//Create the app Deployment
	newDep := utils.NewDeployment(namespace, name, "busybox", "latest", commandV0, replicas)
	err = f.Client.Create(goctx.TODO(), newDep, &framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	if err != nil {
		t.Fatal(err)
	}

	//Wait for the app to be deployed
	err = e2eutil.WaitForDeployment(t, f.KubeClient, namespace, deploymentName, int(replicas), retryInterval, timeout)
	if err != nil {
		t.Fatal(err)
	}

	////Wait for the service to have endpoints
	err = utils.WaitForEndpointsCount(t, f.KubeClient, namespace, serviceName, int(replicas), retryInterval, timeout)
	if err != nil {
		t.Fatal(err)
	}

	//Generate good metrics
	t.Log("Start metric generation for OK pods of normal deployment")
	GenerateMetrics(t, f, testTerminated, namespace, metav1.ListOptions{LabelSelector: "app=" + name}, true, false)

	//Create the Kanary Deployment
	validationConfig := &kanaryv1alpha1.KanaryDeploymentSpecValidationList{
		ValidationPeriod: &metav1.Duration{Duration: 20 * time.Second},
		InitialDelay:     &metav1.Duration{Duration: 5 * time.Second},
		Items: []kanaryv1alpha1.KanaryDeploymentSpecValidation{
			{
				PromQL: &kanaryv1alpha1.KanaryDeploymentSpecValidationPromQL{
					PrometheusService: "prometheus",
					Query:             "(rate(mymetric_sum[10s])/rate(mymetric_count[10s]) and delta(mymetric_count[10s])>3)/scalar(sum(rate(mymetric_sum{kanary_k8s_operators_dev_canary_pod=\"false\"}[10s]))/sum(rate(mymetric_count{kanary_k8s_operators_dev_canary_pod=\"false\"}[10s])))",
					ContinuousValueDeviation: &kanaryv1alpha1.ContinuousValueDeviation{
						MaxDeviationPercent: v1alpha1.NewFloat64(33),
					},
				},
			},
		},
	}

	trafficConfig := &kanaryv1alpha1.KanaryDeploymentSpecTraffic{
		Source: kanaryv1alpha1.ServiceKanaryDeploymentSpecTrafficSource,
	}

	newKD := utils.NewKanaryDeployment(namespace, name, deploymentName, serviceName, "busybox", "latest", commandV1, replicas, nil, trafficConfig, validationConfig)
	err = f.Client.Create(goctx.TODO(), newKD, &framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	if err != nil {
		t.Fatal(err)
	}

	// canary deployment is replicas is setted to 1.
	err = e2eutil.WaitForDeployment(t, f.KubeClient, namespace, canaryName, 1, retryInterval, timeout)
	if err != nil {
		t.Fatal(err)
	}

	// check that kanary point is behind the service
	err = utils.WaitForEndpointsCount(t, f.KubeClient, namespace, serviceName, int(replicas)+1, retryInterval, timeout)
	if err != nil {
		t.Fatal(err)
	}

	//Generate bad metrics
	labelSlector := utils.LabelsAsSelectorString(utilskanary.GetLabelsForKanaryPod(name))
	t.Logf("Start metric generation for KO pods (%s)", labelSlector)
	GenerateMetrics(t, f, testTerminated, namespace, metav1.ListOptions{LabelSelector: labelSlector}, false, true)

	//Wait for the kanary status to change to Invalid
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

//Fake metric used for the test purpose
var sampleHisto = prometheus.NewHistogramVec(prometheus.HistogramOpts{
	Name: "mymetric",
	Help: "mymetric for e2e test",
}, []string{"pod", utils.SanitizeLabel(v1alpha1.KanaryDeploymentActivateLabelKey)})

//GenerateMetrics metrics generation for the pods.
func GenerateMetrics(t *testing.T, f *framework.Framework, done <-chan struct{}, namespace string, podListOptions metav1.ListOptions, okMetrics bool, isCanaryPod bool) {
	canaryLabel := v1alpha1.KanaryDeploymentLabelValueTrue
	if !isCanaryPod {
		canaryLabel = v1alpha1.KanaryDeploymentLabelValueFalse
	}
	failFatalf := t.Fatalf // to avoid problem in the golangci-lint
	ticker := time.NewTicker(time.Second)
	go func() {
		for {
			select {
			case <-ticker.C:
				{
					pods, _ := f.KubeClient.CoreV1().Pods(namespace).List(podListOptions)
					if pods == nil {
						failFatalf("Can't retrieve pods")
					}

					for _, pod := range pods.Items {
						if okMetrics {
							sampleHisto.WithLabelValues(pod.Name, canaryLabel).Observe(utils.RandValueIn(100, 10))
						} else {
							sampleHisto.WithLabelValues(pod.Name, canaryLabel).Observe(utils.RandValueIn(42, 5))
						}
					}
				}
			case <-done:
				{
					ticker.Stop()
					return
				}
			}
		}
	}()
}
