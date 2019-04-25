package e2e

import (
	goctx "context"
	"fmt"
	"math/rand"
	"net"
	"net/url"
	"testing"
	"time"

	kanaryv1alpha1 "github.com/amadeusitgroup/kanary/pkg/apis/kanary/v1alpha1"
	utilskanary "github.com/amadeusitgroup/kanary/pkg/controller/kanarydeployment/utils"
	"github.com/amadeusitgroup/kanary/test/e2e/utils"
	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
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
	gauge     *prometheus.GaugeVec
}

func RandValueIn(base int, delta int) float64 {
	return float64(base+rand.Intn(delta)) - float64(delta/2)
}

func NewPromTestHelper(t *testing.T, ctx *framework.TestCtx, f *framework.Framework, namespace string) *PromTestHelper {
	url, _ := url.Parse(f.KubeConfig.Host)
	minikubeIP, _, _ := net.SplitHostPort(url.Host)
	var sampleGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "mymetric",
		Help: "mymetric for e2e test",
	}, []string{"pod"})

	return &PromTestHelper{
		t:         t,
		ctx:       ctx,
		f:         f,
		namespace: namespace,
		urlPush:   "http://" + minikubeIP + fmt.Sprintf(":%d", prometheusPushNodePort),
		gauge:     sampleGauge,
	}
}

var promConfig = `global:
  scrape_interval:     2s 
  evaluation_interval: 2s 
scrape_configs:  
  - job_name: 'pushgateway'
    honor_labels: true
    static_configs:
      - targets: ['localhost:9091']
`

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
	PromDeployment.Spec.Template.Spec.Containers[0].Ports = []corev1.ContainerPort{{ContainerPort: 9090}}
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
			{ContainerPort: 9091},
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
			Name:     "prom",
			Port:     9090,
			NodePort: prometheusNodePort,
		},
		{
			Name:     "push",
			Port:     9091,
			NodePort: prometheusPushNodePort,
		},
	}

	err = p.f.Client.Create(goctx.TODO(), promService, &framework.CleanupOptions{TestContext: p.ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	if err != nil {
		p.t.Fatal(err)
	}

	// wait that the canary pod is behind the service
	checkEndpoints := func(eps *corev1.Endpoints, wantedPod int) (bool, error) {
		nbPod := 0
		for _, sub := range eps.Subsets {
			nbPod += len(sub.Addresses)
		}
		if wantedPod != nbPod {
			return false, nil
		}
		return true, nil
	}

	check1Endpoints := func(eps *corev1.Endpoints) (bool, error) {
		return checkEndpoints(eps, 1)
	}

	err = utils.WaitForFuncOnEndpoints(p.t, p.f.KubeClient, p.namespace, "prometheus", check1Endpoints, retryInterval, timeout)
	if err != nil {
		p.t.Fatal(err)
	}
}

func (p *PromTestHelper) PushDataToProm() {
	completionTime := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "db_backup_last_completion_timestamp_seconds",
		Help: "The timestamp of the last successful completion of a DB backup.",
	})
	completionTime.SetToCurrentTime()
	if err := push.New(p.urlPush, "db_backup").
		Collector(completionTime).
		Grouping("db", "customers").
		Push(); err != nil {
		fmt.Println("Could not push completion time to Pushgateway:", err)
		p.t.FailNow()
	}
}

func (p *PromTestHelper) GenerateMetrics(pods corev1.PodList, okMetrics bool) {

	ticker := time.NewTicker(time.Second)
	go func() {
		for range ticker.C {
			for _, pod := range pods.Items {
				if okMetrics {
					p.gauge.WithLabelValues(pod.Name).Set(RandValueIn(100, 10))
				} else {
					p.gauge.WithLabelValues(pod.Name).Set(RandValueIn(42, 5))
				}
			}
			if err := push.New(p.urlPush, "db_backup").Collector(p.gauge).Push(); err != nil {
				fmt.Println("Could not push completion time to Pushgateway:", err)
				p.t.FailNow()
			}
		}
	}()
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

	promHelper := NewPromTestHelper(t, ctx, f, namespace)
	promHelper.DeployProm()

	newDep := newDeployment(namespace, name, "busybox", "latest", commandV0, replicas)
	err = f.Client.Create(goctx.TODO(), newDep, &framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	if err != nil {
		t.Fatal(err)
	}

	err = e2eutil.WaitForDeployment(t, f.KubeClient, namespace, deploymentName, int(replicas), retryInterval, timeout)
	if err != nil {
		t.Fatal(err)
	}

	//Generate good metrics
	{
		t.Log("Start metric generation for OK pods")
		pods, _ := f.KubeClient.CoreV1().Pods(namespace).List(metav1.ListOptions{LabelSelector: "app=" + name})
		if pods == nil {
			t.Fatalf("Can't retrieve pods")
		}
		if len(pods.Items) != int(replicas) {
			t.Fatalf("Bad Pod count")
		}
		promHelper.GenerateMetrics(*pods, true)
	}

	validationConfig := &kanaryv1alpha1.KanaryDeploymentSpecValidation{
		ValidationPeriod: &metav1.Duration{Duration: 20 * time.Second},
		PromQL: &kanaryv1alpha1.KanaryDeploymentSpecValidationPromQL{
			PrometheusService: "prometheus",
			// TODO write the QUERY
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

	//Generate bad metrics
	{
		labelSlector := LabelsAsSelectorString(utilskanary.GetLabelsForKanaryPod(name))
		t.Logf("Start metric generation for KO pods (%s)", labelSlector)
		pods, _ := f.KubeClient.CoreV1().Pods(namespace).List(metav1.ListOptions{LabelSelector: labelSlector})
		if pods == nil {
			t.Fatalf("Can't retrieve pods")
		}
		if len(pods.Items) != 1 {
			t.Fatalf("Bad Pod count expecting 1 got %d", len(pods.Items))
		}
		promHelper.GenerateMetrics(*pods, false)
	}
	time.Sleep(60 * time.Second)
	t.FailNow() //Test To be continued
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
