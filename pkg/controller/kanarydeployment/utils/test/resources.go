package utils_test

import (
	appsv1beta1 "k8s.io/api/apps/v1beta1"
	corev1 "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kanaryv1alpha1 "github.com/amadeusitgroup/kanary/pkg/apis/kanary/v1alpha1"
	"github.com/amadeusitgroup/kanary/pkg/controller/kanarydeployment/utils"
)

// NewDeploymentOptions used to provide Deployment creation options
type NewDeploymentOptions struct {
	CreationTime *metav1.Time
}

// NewDeployment returns new Deployment instance for testing purpose
func NewDeployment(name, namespace string, replicas int32, options *NewDeploymentOptions) *appsv1beta1.Deployment {
	spec := &appsv1beta1.DeploymentSpec{
		Replicas: &replicas,
	}
	md5, _ := utils.GenerateMD5DeploymentSpec(spec)
	newDep := &appsv1beta1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: appsv1beta1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Annotations: map[string]string{string(kanaryv1alpha1.MD5KanaryDeploymentAnnotationKey): md5},
		},
		Spec: *spec,
	}

	if options != nil {
		if options.CreationTime != nil {
			newDep.CreationTimestamp = *options.CreationTime
		}
	}

	return newDep
}

func NewService(name, namespace string, labelsSelector map[string]string) *corev1.Service {
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
		},
	}
}
