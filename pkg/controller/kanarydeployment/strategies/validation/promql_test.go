package validation

import (
	"reflect"
	"testing"
	"time"

	kanaryv1alpha1 "github.com/amadeusitgroup/kanary/pkg/apis/kanary/v1alpha1"
	kanaryv1alpha1test "github.com/amadeusitgroup/kanary/pkg/apis/kanary/v1alpha1/test"
	"github.com/amadeusitgroup/kanary/pkg/controller/kanarydeployment/anomalydetector"
	utilstest "github.com/amadeusitgroup/kanary/pkg/controller/kanarydeployment/utils/test"

	appsv1beta1 "k8s.io/api/apps/v1beta1"
	corev1 "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"k8s.io/client-go/kubernetes/scheme"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

func Test_promqlImpl_Validation(t *testing.T) {
	now := time.Now()
	creationTime := &metav1.Time{Time: now.Add(-2 * time.Minute)}
	logf.SetLogger(logf.ZapLogger(true))
	log := logf.Log.WithName("Test_promqlImpl_Validation")

	// Register operator types with the runtime scheme.
	s := scheme.Scheme
	s.AddKnownTypes(kanaryv1alpha1.SchemeGroupVersion, &kanaryv1alpha1.KanaryDeployment{})

	var (
		name = "foo"
		//	serviceName     = "foo"
		namespace       = "kanary"
		defaultReplicas = int32(5)
	)
	type fields struct {
		validationSpec         kanaryv1alpha1.KanaryDeploymentSpecValidationPromQL
		validationPeriod       time.Duration
		maxIntervalPeriod      time.Duration
		dryRun                 bool
		anomalydetector        anomalydetector.AnomalyDetector
		anomalydetectorFactory anomalydetector.Factory
	}
	type args struct {
		kclient   client.Client
		kd        *kanaryv1alpha1.KanaryDeployment
		dep       *appsv1beta1.Deployment
		canaryDep *appsv1beta1.Deployment
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *Result
		wantErr bool
	}{
		{
			name: "no detection",
			fields: fields{
				validationPeriod:       30 * time.Second,
				dryRun:                 false,
				validationSpec:         kanaryv1alpha1.KanaryDeploymentSpecValidationPromQL{},
				anomalydetectorFactory: anomalydetector.FakeFactory([]*corev1.Pod{}, nil),
			},
			args: args{
				kclient:   fake.NewFakeClient([]runtime.Object{utilstest.NewDeployment(name, namespace, defaultReplicas, nil), utilstest.NewPod(name, namespace, "hash", &utilstest.NewPodOptions{Labels: map[string]string{"foo": "bar"}})}...),
				kd:        kanaryv1alpha1test.NewKanaryDeployment(name, namespace, "", defaultReplicas, &kanaryv1alpha1test.NewKanaryDeploymentOptions{}),
				dep:       utilstest.NewDeployment(name, namespace, defaultReplicas, &utilstest.NewDeploymentOptions{CreationTime: creationTime, Labels: map[string]string{"foo": "bar"}, Selector: map[string]string{"foo": "bar"}}),
				canaryDep: utilstest.NewDeployment(name+"-kanary-"+name, namespace, 1, &utilstest.NewDeploymentOptions{CreationTime: creationTime, Labels: map[string]string{"foo": "bar", "foo-k": "bar-k"}, Selector: map[string]string{"foo-k": "bar-k"}}),
			},
			want: &Result{
				IsFailed: false,
			},
			wantErr: false,
		},
		{
			name: "detect-1",
			fields: fields{
				validationPeriod: 30 * time.Second,
				dryRun:           false,
				validationSpec:   kanaryv1alpha1.KanaryDeploymentSpecValidationPromQL{},
				anomalydetectorFactory: anomalydetector.FakeFactory(
					[]*corev1.Pod{utilstest.NewPod(name+"-kanary", namespace, "hash", &utilstest.NewPodOptions{Labels: map[string]string{"foo": "bar", "foo-k": "bar-k"}})},
					nil,
				),
			},
			args: args{
				kclient: fake.NewFakeClient(
					[]runtime.Object{
						utilstest.NewDeployment(name, namespace, defaultReplicas, nil),
						utilstest.NewPod(name, namespace, "hash", &utilstest.NewPodOptions{Labels: map[string]string{"foo": "bar"}}),
						utilstest.NewPod(name+"-kanary", namespace, "hash", &utilstest.NewPodOptions{Labels: map[string]string{"foo": "bar", "foo-k": "bar-k"}}),
					}...),
				kd:        kanaryv1alpha1test.NewKanaryDeployment(name, namespace, "", defaultReplicas, &kanaryv1alpha1test.NewKanaryDeploymentOptions{}),
				dep:       utilstest.NewDeployment(name, namespace, defaultReplicas, &utilstest.NewDeploymentOptions{CreationTime: creationTime, Labels: map[string]string{"foo": "bar"}, Selector: map[string]string{"foo": "bar"}}),
				canaryDep: utilstest.NewDeployment(name+"-kanary-"+name, namespace, 1, &utilstest.NewDeploymentOptions{CreationTime: creationTime, Labels: map[string]string{"foo": "bar", "foo-k": "bar-k"}, Selector: map[string]string{"foo-k": "bar-k"}}),
			},
			want: &Result{
				IsFailed: true,
				Comment:  "promQL query reported an issue with one of the kanary pod",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqLogger := log.WithValues("test:", tt.name)
			p := &promqlImpl{
				validationSpec:         tt.fields.validationSpec,
				validationPeriod:       tt.fields.validationPeriod,
				maxIntervalPeriod:      tt.fields.maxIntervalPeriod,
				dryRun:                 tt.fields.dryRun,
				anomalydetector:        tt.fields.anomalydetector,
				anomalydetectorFactory: tt.fields.anomalydetectorFactory,
			}
			got, err := p.Validation(tt.args.kclient, reqLogger, tt.args.kd, tt.args.dep, tt.args.canaryDep)
			if (err != nil) != tt.wantErr {
				t.Errorf("promqlImpl.Validation() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("promqlImpl.Validation() = %#v, want %#v", got, tt.want)
			}
		})
	}
}
