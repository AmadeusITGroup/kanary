package utils

import (
	"context"
	"fmt"

	appsv1beta1 "k8s.io/api/apps/v1beta1"
	corev1 "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/amadeusitgroup/kanary/pkg/apis"
	kanaryv1alpha1 "github.com/amadeusitgroup/kanary/pkg/apis/kanary/v1alpha1"
	"github.com/amadeusitgroup/kanary/pkg/controller/kanarydeployment/utils/comparison"
)

//PrepareSchemeForOwnerRef return the scheme required to write the kanary ownerreference
func PrepareSchemeForOwnerRef() *runtime.Scheme {
	scheme := runtime.NewScheme()
	if err := apis.AddToScheme(scheme); err != nil {
		panic(err.Error())
	}
	return scheme
}

// NewCanaryServiceForKanaryDeployment returns a Service object
func NewCanaryServiceForKanaryDeployment(kd *kanaryv1alpha1.KanaryDeployment, service *corev1.Service, overwriteLabel bool, scheme *runtime.Scheme, setOwnerRef bool) (*corev1.Service, error) {
	kanaryServiceName := GetCanaryServiceName(kd)

	labelSelector := map[string]string{}
	labelSelector[kanaryv1alpha1.KanaryDeploymentKanaryNameLabelKey] = kd.Name
	labelSelector[kanaryv1alpha1.KanaryDeploymentActivateLabelKey] = kanaryv1alpha1.KanaryDeploymentLabelValueTrue

	newService := service.DeepCopy()
	newService.ObjectMeta = metav1.ObjectMeta{
		Name:      kanaryServiceName,
		Namespace: kd.Namespace,
	}
	newService.Spec.Selector = labelSelector
	if newService.Spec.Type == corev1.ServiceTypeNodePort || newService.Spec.Type == corev1.ServiceTypeLoadBalancer {
		// this is to remove Port collision
		if newService.Spec.Type == corev1.ServiceTypeNodePort {
			for i := range newService.Spec.Ports {
				newService.Spec.Ports[i].NodePort = 0
			}
		}
		if newService.Spec.Type == corev1.ServiceTypeLoadBalancer {
			newService.Spec.LoadBalancerSourceRanges = nil
		}

		newService.Spec.Type = corev1.ServiceTypeClusterIP
	}
	newService.Spec.ClusterIP = ""
	newService.Status = corev1.ServiceStatus{}

	if setOwnerRef {
		// Set KanaryDeployment instance as the owner and controller
		if err := controllerutil.SetControllerReference(kd, newService, scheme); err != nil {
			return nil, err
		}
	}
	return newService, nil
}

// GetCanaryServiceName returns the canary service name depending of the spec
func GetCanaryServiceName(kd *kanaryv1alpha1.KanaryDeployment) string {
	kanaryServiceName := kd.Spec.Traffic.KanaryService
	if kanaryServiceName == "" {
		kanaryServiceName = fmt.Sprintf("%s-kanary-%s", kd.Spec.ServiceName, kd.Name)
	}
	return kanaryServiceName
}

// NewDeploymentFromKanaryDeploymentTemplate returns a Deployment object
func NewDeploymentFromKanaryDeploymentTemplate(kdold *kanaryv1alpha1.KanaryDeployment, scheme *runtime.Scheme, setOwnerRef bool) (*appsv1beta1.Deployment, error) {
	kd := kdold.DeepCopy()
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

	if _, err := comparison.SetMD5DeploymentSpecAnnotation(kd, dep); err != nil {
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
func NewCanaryDeploymentFromKanaryDeploymentTemplate(kclient client.Client, kd *kanaryv1alpha1.KanaryDeployment, scheme *runtime.Scheme, setOwnerRef bool) (*appsv1beta1.Deployment, error) {
	dep, err := NewDeploymentFromKanaryDeploymentTemplate(kd, scheme, true)
	if err != nil {
		return nil, err
	}
	dep.Name = GetCanaryDeploymentName(kd)
	// Overwrite the Pods labels and the Deployment spec selector
	dep.Spec.Template.Labels = map[string]string{
		kanaryv1alpha1.KanaryDeploymentKanaryNameLabelKey: kd.Name,
		kanaryv1alpha1.KanaryDeploymentActivateLabelKey:   kanaryv1alpha1.KanaryDeploymentLabelValueTrue,
	}
	dep.Spec.Selector.MatchLabels = dep.Spec.Template.Labels

	//Here add the labels that are not part of the service selector
	service := &corev1.Service{}
	err = kclient.Get(context.TODO(), types.NamespacedName{Name: kd.Spec.ServiceName, Namespace: kd.Namespace}, service)
	serviceSelector := service.Spec.Selector
	if err == nil {
		for k, v := range kd.Spec.Template.Spec.Template.ObjectMeta.Labels {
			if _, ok := serviceSelector[k]; ok {
				continue // don't add this label that is used by service discovery. The traffic strategy will add it if needed
			}
			dep.Spec.Template.Labels[k] = v //typically add labels like "version" that are used for pod management but not for service discovery
		}
	}

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

	if _, err := comparison.SetMD5DeploymentSpecAnnotation(kd, newDep); err != nil {
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
	return fmt.Sprintf("%s-kanary-%s", GetDeploymentName(kd), kd.Name)
}

// GetLabelsForKanaryDeploymentd return labels belonging to the given KanaryDeployment CR name.
func GetLabelsForKanaryDeploymentd(name string) map[string]string {
	return map[string]string{
		kanaryv1alpha1.KanaryDeploymentIsKanaryLabelKey:   kanaryv1alpha1.KanaryDeploymentLabelValueTrue,
		kanaryv1alpha1.KanaryDeploymentKanaryNameLabelKey: name,
	}
}

// GetLabelsForKanaryPod return labels of a canary pod associated to a kanarydeployment.
func GetLabelsForKanaryPod(kdname string) map[string]string {
	return map[string]string{
		kanaryv1alpha1.KanaryDeploymentKanaryNameLabelKey: kdname,
		kanaryv1alpha1.KanaryDeploymentActivateLabelKey:   kanaryv1alpha1.KanaryDeploymentLabelValueTrue,
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
