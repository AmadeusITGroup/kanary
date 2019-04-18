package utils

import (
	"testing"

	"k8s.io/apimachinery/pkg/api/equality"

	corev1 "k8s.io/api/core/v1"

	kanaryv1alpha1 "github.com/amadeusitgroup/kanary/pkg/apis/kanary/v1alpha1"
	kanaryv1alpha1test "github.com/amadeusitgroup/kanary/pkg/apis/kanary/v1alpha1/test"
	utilstest "github.com/amadeusitgroup/kanary/pkg/controller/kanarydeployment/utils/test"
)

func TestNewCanaryServiceForKanaryDeployment(t *testing.T) {
	namespace := "kanary"
	name := "foo"
	dummyKD := kanaryv1alpha1test.NewKanaryDeployment(name, namespace, name, 3, nil)

	type args struct {
		kd             *kanaryv1alpha1.KanaryDeployment
		service        *corev1.Service
		overwriteLabel bool
	}
	tests := []struct {
		name string
		args args
		want *corev1.Service
	}{
		{
			name: "nodePort service",
			args: args{
				kd:             dummyKD,
				service:        utilstest.NewService(name, namespace, nil, &utilstest.NewServiceOptions{Type: corev1.ServiceTypeNodePort, Ports: []corev1.ServicePort{{Port: 8080, NodePort: 3010}}}),
				overwriteLabel: false,
			},
			want: utilstest.NewService(name+"-kanary-"+name, namespace, map[string]string{kanaryv1alpha1.KanaryDeploymentActivateLabelKey: kanaryv1alpha1.KanaryDeploymentLabelValueTrue, kanaryv1alpha1.KanaryDeploymentKanaryNameLabelKey: name}, &utilstest.NewServiceOptions{Type: corev1.ServiceTypeClusterIP, Ports: []corev1.ServicePort{{Port: 8080}}}),
		},
		{
			name: "loadbalancer service",
			args: args{
				kd:             dummyKD,
				service:        utilstest.NewService(name, namespace, nil, &utilstest.NewServiceOptions{Type: corev1.ServiceTypeLoadBalancer}),
				overwriteLabel: false,
			},
			want: utilstest.NewService(name+"-kanary-"+name, namespace, map[string]string{kanaryv1alpha1.KanaryDeploymentActivateLabelKey: kanaryv1alpha1.KanaryDeploymentLabelValueTrue, kanaryv1alpha1.KanaryDeploymentKanaryNameLabelKey: name}, &utilstest.NewServiceOptions{Type: corev1.ServiceTypeClusterIP}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got, _ := NewCanaryServiceForKanaryDeployment(tt.args.kd, tt.args.service, tt.args.overwriteLabel, PrepareSchemeForOwnerRef(), false); !equality.Semantic.DeepEqual(got, tt.want) {
				t.Errorf("NewCanaryServiceForKanaryDeployment() = %v, want %v", got, tt.want)
			}
		})
	}
}
