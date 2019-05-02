package e2e

import (
	goctx "context"
	"fmt"
	"math/rand"
	"regexp"
	"sync"
	"testing"
	"time"

	"github.com/amadeusitgroup/kanary/pkg/apis/kanary/v1alpha1"
	kanaryv1alpha1 "github.com/amadeusitgroup/kanary/pkg/apis/kanary/v1alpha1"
	utilskanary "github.com/amadeusitgroup/kanary/pkg/controller/kanarydeployment/utils"
	"github.com/amadeusitgroup/kanary/test/e2e/utils"
	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	prometheusPushPort      = 9091
	prometheusServerPort    = 9090
	prometheusPushNodePort  = 30991
	prometheusNodePort      = 30990
	prometheusConfigMapName = "promconfig"
)

type PromTestHelper struct {
	t         *testing.T
	ctx       *framework.TestCtx
	f         *framework.Framework
	namespace string
	urlPush   string
	histogram *prometheus.HistogramVec
	jobName   string
	done      chan struct{}
	wg        sync.WaitGroup
}

func RandValueIn(base int, delta int) float64 {
	return float64(base+rand.Intn(delta)) - float64(delta/2)
}

func NewPromTestHelper(t *testing.T, ctx *framework.TestCtx, f *framework.Framework, namespace string, jobName string) *PromTestHelper {
	var sampleHisto = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: "mymetric",
		Help: "mymetric for e2e test",
	}, []string{"pod", sanitizeLabel(v1alpha1.KanaryDeploymentActivateLabelKey)})

	return &PromTestHelper{
		t:         t,
		ctx:       ctx,
		f:         f,
		namespace: namespace,
		urlPush:   "http://127.0.0.1:8001/api/v1/namespaces/" + namespace + "/services/http:prometheus:" + fmt.Sprintf("%d", prometheusPushPort) + "/proxy",
		histogram: sampleHisto,
		jobName:   jobName,
		done:      make(chan struct{}),
	}
}

var promConfig = `global:
  scrape_interval:     2s 
  evaluation_interval: 2s 
scrape_configs:  
  - job_name: 'pushgateway'
    honor_labels: true
    static_configs:
      - targets: ['localhost:` + fmt.Sprintf("%d", prometheusPushPort) + `']
`

func (p *PromTestHelper) Stop() {
	close(p.done)
	p.wg.Wait()
}

func (p *PromTestHelper) CreatePromConfigMap() {
	cm := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: corev1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      prometheusConfigMapName,
			Namespace: p.namespace,
		},
		Data: map[string]string{
			"prometheus.yml": promConfig,
		},
	}
	err := p.f.Client.Create(goctx.TODO(), cm, &framework.CleanupOptions{TestContext: p.ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	if err != nil {
		p.t.Fatal(err)
	}
}

func (p *PromTestHelper) DeployProm() {
	p.CreatePromConfigMap()
	PromDeployment := newDeployment(p.namespace, "prometheus", "prom/prometheus", "v2.9.2", nil, 1)
	PromDeployment.Spec.Template.Spec.Containers[0].Ports = []corev1.ContainerPort{{ContainerPort: prometheusServerPort}}
	PromDeployment.Spec.Template.Spec.Containers[0].Args = []string{"--config.file=/etc/config/prometheus.yml"}
	PromDeployment.Spec.Template.Spec.Containers[0].VolumeMounts = []corev1.VolumeMount{
		{
			Name:      "config-volume",
			MountPath: "/etc/config",
		},
	}
	PromDeployment.Spec.Template.Spec.Volumes = []corev1.Volume{
		{
			Name: "config-volume",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: prometheusConfigMapName,
					},
				},
			},
		},
	}
	pushContainer := corev1.Container{
		Name:  "pushgateway",
		Image: "prom/pushgateway:v0.8.0",
		Ports: []corev1.ContainerPort{
			{ContainerPort: prometheusPushPort},
		},
	}
	PromDeployment.Spec.Template.Spec.Containers = append(PromDeployment.Spec.Template.Spec.Containers, pushContainer)
	err := p.f.Client.Create(goctx.TODO(), PromDeployment, &framework.CleanupOptions{TestContext: p.ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	if err != nil {
		p.t.Fatal(err)
	}

	err = e2eutil.WaitForDeployment(p.t, p.f.KubeClient, p.namespace, "prometheus", 1, retryInterval, 2*timeout)
	if err != nil {
		p.t.Fatal(err)
	}

	promService := newService(p.namespace, "prometheus", map[string]string{"app": "prometheus"})
	promService.Spec.Type = corev1.ServiceTypeNodePort
	promService.Spec.Ports = []corev1.ServicePort{
		{
			Name: "prom",
			Port: 80,
			TargetPort: intstr.IntOrString{
				Type:   intstr.Int,
				IntVal: prometheusServerPort,
			},
			NodePort: prometheusNodePort,
		},
		{
			Name:     "push",
			Port:     prometheusPushPort,
			NodePort: prometheusPushNodePort,
		},
	}

	err = p.f.Client.Create(goctx.TODO(), promService, &framework.CleanupOptions{TestContext: p.ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	if err != nil {
		p.t.Fatal(err)
	}

	err = utils.WaitForEndpointsCount(p.t, p.f.KubeClient, p.namespace, "prometheus", 1, retryInterval, timeout)
	if err != nil {
		p.t.Fatal(err)
	}
}

func (p *PromTestHelper) GenerateMetrics(podListOptions metav1.ListOptions, okMetrics bool, isCanaryPod bool) (podsCount int) {
	pods, _ := p.f.KubeClient.CoreV1().Pods(p.namespace).List(podListOptions)
	if pods == nil {
		p.t.Fatalf("Can't retrieve pods")
	}
	canaryLabel := v1alpha1.KanaryDeploymentLabelValueTrue
	if !isCanaryPod {
		canaryLabel = v1alpha1.KanaryDeploymentLabelValueFalse
	}

	failNow := p.t.FailNow // to avoid problem in the golangci-lint

	ticker := time.NewTicker(time.Second)
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		for {
			select {
			case <-ticker.C:
				{
					{
						for _, pod := range pods.Items {
							if okMetrics {
								p.histogram.WithLabelValues(pod.Name, canaryLabel).Observe(RandValueIn(100, 10))
							} else {
								p.histogram.WithLabelValues(pod.Name, canaryLabel).Observe(RandValueIn(42, 5))
							}
						}
						if err := push.New(p.urlPush, p.jobName).Collector(p.histogram).Push(); err != nil {
							fmt.Println("Could not push completion time to Pushgateway:", err)
							failNow()
						}
					}
				}
			case <-p.done:
				{
					ticker.Stop()
					return
				}
			}
		}
	}()
	return len(pods.Items)
}

func sanitizeLabel(label string) string {
	r := regexp.MustCompile("[^a-zA-Z0-9_]")
	return r.ReplaceAllString(label, "_")
}

func PromqlInvalidation(t *testing.T) {
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

	promHelper := NewPromTestHelper(t, ctx, f, namespace, "Test_PromQlInvalidation")
	promHelper.DeployProm()
	defer promHelper.Stop()

	newService := newService(namespace, serviceName, map[string]string{"app": name})
	err = f.Client.Create(goctx.TODO(), newService, &framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	if err != nil {
		t.Fatal(err)
	}

	newDep := newDeployment(namespace, name, "busybox", "latest", commandV0, replicas)
	err = f.Client.Create(goctx.TODO(), newDep, &framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	if err != nil {
		t.Fatal(err)
	}

	err = e2eutil.WaitForDeployment(t, f.KubeClient, namespace, deploymentName, int(replicas), retryInterval, timeout)
	if err != nil {
		t.Fatal(err)
	}

	// check that pods are behind the service
	err = utils.WaitForEndpointsCount(t, f.KubeClient, namespace, serviceName, int(replicas), retryInterval, timeout)
	if err != nil {
		t.Fatal(err)
	}

	//Generate good metrics
	t.Log("Start metric generation for OK pods of normal deployment")
	if podsCount := promHelper.GenerateMetrics(metav1.ListOptions{LabelSelector: "app=" + name}, true, false); podsCount != int(replicas) {
		t.Fatalf("Bad Pod count")
	}

	validationConfig := &kanaryv1alpha1.KanaryDeploymentSpecValidation{
		ValidationPeriod: &metav1.Duration{Duration: 20 * time.Second},
		InitialDelay:     &metav1.Duration{Duration: 5 * time.Second},
		PromQL: &kanaryv1alpha1.KanaryDeploymentSpecValidationPromQL{
			PrometheusService: "prometheus",
			Query:             "(rate(mymetric_sum[10s])/rate(mymetric_count[10s]) and delta(mymetric_count[10s])>3)/scalar(sum(rate(mymetric_sum{kanary_k8s_io_canary_pod=\"false\"}[10s]))/sum(rate(mymetric_count{kanary_k8s_io_canary_pod=\"false\"}[10s])))",
			ContinuousValueDeviation: &kanaryv1alpha1.ContinuousValueDeviation{
				MaxDeviationPercent: v1alpha1.NewFloat64(33),
			},
		},
	}

	trafficConfig := &kanaryv1alpha1.KanaryDeploymentSpecTraffic{
		Source: kanaryv1alpha1.ServiceKanaryDeploymentSpecTrafficSource,
	}

	newKD := newKanaryDeployment(namespace, name, deploymentName, serviceName, "busybox", "latest", commandV1, replicas, nil, trafficConfig, validationConfig)
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
	labelSlector := LabelsAsSelectorString(utilskanary.GetLabelsForKanaryPod(name))
	t.Logf("Start metric generation for KO pods (%s)", labelSlector)
	if podsCount := promHelper.GenerateMetrics(metav1.ListOptions{LabelSelector: labelSlector}, false, true); podsCount != 1 {
		t.Fatalf("Bad Pod count")
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

func LabelsAsSelectorString(lbs map[string]string) string {
	str := ""
	for k, v := range lbs {
		str += k + "=" + v + ","
	}
	if len(str) > 0 {
		str = str[:len(str)-1]
	}
	return str
}
