package utils

import (
	"fmt"

	appsv1beta1 "k8s.io/api/apps/v1beta1"
	corev1 "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	kanaryv1alpha1 "github.com/amadeusitgroup/kanary/pkg/apis/kanary/v1alpha1"
)

// NewCanaryServiceForKanaryDeployment returns a Service object
func NewCanaryServiceForKanaryDeployment(kd *kanaryv1alpha1.KanaryDeployment, service *corev1.Service, overwriteLabel bool) *corev1.Service {
	kanaryServiceName := GetCanaryServiceName(kd)

	labelSelector := map[string]string{}
	for key, val := range service.Spec.Selector {
		if overwriteLabel {
			labelSelector[key] = fmt.Sprintf("%s-%s", val, "kanary")
		} else {
			labelSelector[key] = val
		}

	}
	labelSelector[kanaryv1alpha1.KanaryDeploymentActivateLabelKey] = kanaryv1alpha1.KanaryDeploymentLabelValueTrue

	newService := service.DeepCopy()
	newService.ObjectMeta = metav1.ObjectMeta{
		Name:      kanaryServiceName,
		Namespace: kd.Namespace,
	}
	newService.Spec.Selector = labelSelector
	if newService.Spec.Type == corev1.ServiceTypeNodePort || newService.Spec.Type == corev1.ServiceTypeLoadBalancer {
		// this is to remove Port collision
		newService.Spec.Type = corev1.ServiceTypeClusterIP
	}
	newService.Spec.ClusterIP = ""
	newService.Status = corev1.ServiceStatus{}

	return newService
}

// GetCanaryServiceName returns the canary service name depending of the spec
func GetCanaryServiceName(kd *kanaryv1alpha1.KanaryDeployment) string {
	kanaryServiceName := kd.Spec.Traffic.KanaryService
	if kanaryServiceName == "" {
		kanaryServiceName = fmt.Sprintf("%s-kanary", kd.Spec.ServiceName)
	}
	return kanaryServiceName
}

// NewDeploymentFromKanaryDeploymentTemplate returns a Deployment object
func NewDeploymentFromKanaryDeploymentTemplate(kd *kanaryv1alpha1.KanaryDeployment, scheme *runtime.Scheme, setOwnerRef bool) (*appsv1beta1.Deployment, error) {
	ls := GetLabelsForKanaryDeploymentd(kd.Name)

	dep := &appsv1beta1.Deployment{
		TypeMeta:   kd.Spec.Template.TypeMeta,
		ObjectMeta: kd.Spec.Template.ObjectMeta,
		Spec:       kd.Spec.Template.Spec,
	}

	if dep.Labels == nil {
		dep.Labels = map[string]string{}
	}
	for key, val := range ls {
		dep.Labels[key] = val
	}

	dep.Name = GetDeploymentName(kd)
	if dep.Namespace == "" {
		dep.Namespace = kd.Namespace
	}

	if _, err := SetMD5DeploymentSpecAnnotation(kd, dep); err != nil {
		return nil, fmt.Errorf("unable to set the md5 annotation, %v", err)
	}

	if setOwnerRef {
		// Set KanaryDeployment instance as the owner and controller
		if err := controllerutil.SetControllerReference(kd, dep, scheme); err != nil {
			return dep, err
		}
	}
	return dep, nil
}

// NewCanaryDeploymentFromKanaryDeploymentTemplate returns a Deployment object
func NewCanaryDeploymentFromKanaryDeploymentTemplate(kd *kanaryv1alpha1.KanaryDeployment, scheme *runtime.Scheme, setOwnerRef bool, overwriteLabel bool) (*appsv1beta1.Deployment, error) {
	dep, err := NewDeploymentFromKanaryDeploymentTemplate(kd, scheme, true)
	if err != nil {
		return nil, err
	}
	dep.Name = GetCanaryDeploymentName(kd)
	if dep.Spec.Template.Labels == nil {
		dep.Spec.Template.Labels = map[string]string{}
	}
	if dep.Spec.Selector.MatchLabels == nil {
		dep.Spec.Selector.MatchLabels = map[string]string{}
	}

	if overwriteLabel {
		for key, value := range dep.Spec.Template.Labels {
			dep.Spec.Template.Labels[key] = fmt.Sprintf("%s-%s", value, "kanary")
		}

		for key, value := range dep.Spec.Selector.MatchLabels {
			dep.Spec.Selector.MatchLabels[key] = fmt.Sprintf("%s-%s", value, "kanary")
		}
	}

	dep.Spec.Template.Labels[kanaryv1alpha1.KanaryDeploymentActivateLabelKey] = kanaryv1alpha1.KanaryDeploymentLabelValueTrue
	dep.Spec.Selector.MatchLabels[kanaryv1alpha1.KanaryDeploymentActivateLabelKey] = kanaryv1alpha1.KanaryDeploymentLabelValueTrue

	dep.Spec.Replicas = GetCanaryReplicasValue(kd)

	return dep, nil
}

// UpdateDeploymentWithKanaryDeploymentTemplate returns a Deployment object updated
func UpdateDeploymentWithKanaryDeploymentTemplate(kd *kanaryv1alpha1.KanaryDeployment, oldDep *appsv1beta1.Deployment) (*appsv1beta1.Deployment, error) {
	newDep := oldDep.DeepCopy()
	{
		newDep.Labels = kd.Spec.Template.Labels
		newDep.Annotations = kd.Spec.Template.Annotations
		newDep.Spec = kd.Spec.Template.Spec
	}

	if _, err := SetMD5DeploymentSpecAnnotation(kd, newDep); err != nil {
		return nil, fmt.Errorf("unable to set the md5 annotation, %v", err)
	}

	return newDep, nil
}

// GetDeploymentName returns the Deployment name from the KanaryDeployment instance
func GetDeploymentName(kd *kanaryv1alpha1.KanaryDeployment) string {
	name := kd.Spec.Template.ObjectMeta.Name
	if kd.Spec.DeploymentName != "" {
		name = kd.Spec.DeploymentName
	} else if name == "" {
		name = kd.Name
	}
	return name
}

// GetCanaryDeploymentName returns the Canary Deployment name from the KanaryDeployment instance
func GetCanaryDeploymentName(kd *kanaryv1alpha1.KanaryDeployment) string {
	return fmt.Sprintf("%s-kanary", GetDeploymentName(kd))
}

// GetLabelsForKanaryDeploymentd return labels belonging to the given KanaryDeployment CR name.
func GetLabelsForKanaryDeploymentd(name string) map[string]string {
	return map[string]string{
		kanaryv1alpha1.KanaryDeploymentIsKanaryLabelKey:   kanaryv1alpha1.KanaryDeploymentLabelValueTrue,
		kanaryv1alpha1.KanaryDeploymentKanaryNameLabelKey: name,
	}
}

// GetCanaryReplicasValue returns the replicas value of the Canary Deployment
func GetCanaryReplicasValue(kd *kanaryv1alpha1.KanaryDeployment) *int32 {
	var value *int32
	if kd.Spec.Scale.Static != nil {
		value = kd.Spec.Scale.Static.Replicas
	}
	return value
}
