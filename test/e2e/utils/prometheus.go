package utils

import (
	goctx "context"
	"fmt"
	"math/rand"
	"regexp"
	"sync"
	"testing"
	"time"

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

var (
	retryInterval        = time.Second * 5
	timeout              = time.Second * 60
	cleanupRetryInterval = time.Second * 1
	cleanupTimeout       = time.Second * 5
)

//PromTestHelper struct helps to create a Prometheus pod that rune a prom server and a push gateway
type PromTestHelper struct {
	t          *testing.T
	ctx        *framework.TestCtx
	f          *framework.Framework
	namespace  string
	urlPush    string
	collectors []prometheus.Collector
	jobName    string
	done       chan struct{}
	wg         sync.WaitGroup
}

//RandValueIn return a random value +/- delta around the base
func RandValueIn(base int, delta int) float64 {
	return float64(base+rand.Intn(delta)) - float64(delta/2)
}

//NewPromTestHelper contructor for PromTestHelper
func NewPromTestHelper(t *testing.T, ctx *framework.TestCtx, f *framework.Framework, jobName string, collectors []prometheus.Collector) *PromTestHelper {
	namespace, _ := ctx.GetNamespace()
	return &PromTestHelper{
		t:          t,
		ctx:        ctx,
		f:          f,
		namespace:  namespace,
		urlPush:    "http://127.0.0.1:8001/api/v1/namespaces/" + namespace + "/services/http:prometheus:" + fmt.Sprintf("%d", prometheusPushPort) + "/proxy",
		jobName:    jobName,
		done:       make(chan struct{}),
		collectors: collectors,
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

//StopPushGateway terminates on going metric push
func (p *PromTestHelper) StopPushGateway() {
	close(p.done)
	p.wg.Wait()
}

func (p *PromTestHelper) createPromConfigMap() {
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

//Run deploy the prometheus pod (server+pushgateway) and associated service
func (p *PromTestHelper) Run() {
	p.createPromConfigMap()
	PromDeployment := NewDeployment(p.namespace, "prometheus", "prom/prometheus", "v2.9.2", nil, 1)
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

	promService := NewService(p.namespace, "prometheus", map[string]string{"app": "prometheus"})
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

	err = WaitForEndpointsCount(p.t, p.f.KubeClient, p.namespace, "prometheus", 1, retryInterval, timeout)
	if err != nil {
		p.t.Fatal(err)
	}

	p.runPushGateway()
}

func (p *PromTestHelper) runPushGateway() {
	failNow := p.t.FailNow // to avoid problem in the golangci-lint
	ticker := time.NewTicker(time.Second)
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		for {
			select {
			case <-ticker.C:
				{
					for _, collector := range p.collectors {
						if err := push.New(p.urlPush, p.jobName).Collector(collector).Push(); err != nil {
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
}

//SanitizeLabel sanitize a kubernetes label to be used as prometheus label
func SanitizeLabel(label string) string {
	r := regexp.MustCompile("[^a-zA-Z0-9_]")
	return r.ReplaceAllString(label, "_")
}
