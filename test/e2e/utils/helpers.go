package utils

import (
	goctx "context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1beta1"
	corev1 "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	framework "github.com/operator-framework/operator-sdk/pkg/test"

	kanaryv1alpha1 "github.com/amadeusitgroup/kanary/pkg/apis/kanary/v1alpha1"
)

//UpdateDeploymentFunc update the deployment using the updateFunc
func UpdateDeploymentFunc(f *framework.Framework, name, namespace string, updateFunc func(kd *appsv1.Deployment)) error {
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

//NewDeployment create a new deployment
func NewDeployment(namespace, name, image, tag string, command []string, replicas int32) *appsv1.Deployment {
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

//NewKanaryDeployment create a new Kanary Deployments CR
func NewKanaryDeployment(namespace, name, deploymentName, serviceName, image, tag string, command []string, replicas int32, scale *kanaryv1alpha1.KanaryDeploymentSpecScale, traffic *kanaryv1alpha1.KanaryDeploymentSpecTraffic, validation *kanaryv1alpha1.KanaryDeploymentSpecValidationList) *kanaryv1alpha1.KanaryDeployment {
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
		kd.Spec.Validations = *validation
	}

	return kd
}

//NewService create a new service
func NewService(namespace, name string, labelsSelector map[string]string) *corev1.Service {
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

//LabelsAsSelectorString albel map to label string
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
