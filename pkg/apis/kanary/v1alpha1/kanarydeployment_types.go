package v1alpha1

import (
	"k8s.io/api/apps/v1beta1"
	"k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// KanaryDeploymentSpec defines the desired state of KanaryDeployment
type KanaryDeploymentSpec struct {
	// Template  is the object that describes the deployment that will be created.
	Template DeploymentTemplate `json:"template,omitempty"`
	// Strategy indicates the strategy that the KanaryDeployment controller will use to perform
	// the canary deployment.
	Strategy KanaryDeploymentStrategy `json:"strategy,omitempty"`
	// serviceName is the name of the service that governs the associated Deployment.
	// This service can be empty of not defined, which means that some Kanary feature will not be
	// applied on the KanaryDeployment.
	ServiceName string `json:"serviceName,omitempty"`
}

// KanaryDeploymentStatus defines the observed state of KanaryDeployment
type KanaryDeploymentStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file

	// Represents the latest available observations of a kanarydeployment's current state.
	Conditions []KanaryDeploymentCondition `json:"conditions,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// KanaryDeployment is the Schema for the kanarydeployments API
// +k8s:openapi-gen=true
type KanaryDeployment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KanaryDeploymentSpec   `json:"spec,omitempty"`
	Status KanaryDeploymentStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// KanaryDeploymentList contains a list of KanaryDeployment
type KanaryDeploymentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KanaryDeployment `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KanaryDeployment{}, &KanaryDeploymentList{})
}

// DeploymentTemplate is the object that describes the deployment that will be created.
type DeploymentTemplate struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Specification of the desired behavior of the Deployment.
	// +optional
	Spec v1beta1.DeploymentSpec `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`
}

// KanaryDeploymentStrategy indicates the strategy that the KanaryDeployment controller will
// use to perform the canary deployment. It includes any additional parameters necessary to
// perform the canary deployment for the indicated strategy.
type KanaryDeploymentStrategy struct {
	// ServiceName is the name of the service that will be created to target specificaly
	// pods that serve the canary service version.
	// if kanaryServiceName is empty or not define, a service name will be generated from the
	// serviceName provided in the KanaryDeploymentSpec.
	ServiceName string `json:"serviceName,omitempty"`
}

// KanaryDeploymentCondition describes the state of a deployment at a certain point.
type KanaryDeploymentCondition struct {
	// Type of deployment condition.
	Type KanaryDeploymentConditionType `json:"type"`
	// Status of the condition, one of True, False, Unknown.
	Status v1.ConditionStatus `json:"status"`
	// The last time this condition was updated.
	LastUpdateTime metav1.Time `json:"lastUpdateTime,omitempty"`
	// Last time the condition transitioned from one status to another.
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
	// The reason for the condition's last transition.
	Reason string `json:"reason,omitempty"`
	// A human readable message indicating details about the transition.
	Message string `json:"message,omitempty"`
}

// KanaryDeploymentConditionType describes the state of a deployment at a certain point.
type KanaryDeploymentConditionType string

// These are valid conditions of a kanarydeployment.
const (
	// Available means the deployment is available, ie. at least the minimum available
	// replicas required are up and running for at least minReadySeconds.
	AvailableKanaryDeploymentConditionType KanaryDeploymentConditionType = "Available"
	// Progressing means the deployment is progressing. Progress for a deployment is
	// considered when a new canary deployment is created, and when new pods scale
	// up. Progress is not estimated for paused deployments or
	// when progressDeadlineSeconds is not specified.
	ProgressingKanaryDeploymentConditionType KanaryDeploymentConditionType = "Progressing"
	// FailureKanaryDeploymentConditionType is added in a kanarydeployment when the canary deployment
	// process failed.
	FailureKanaryDeploymentConditionType KanaryDeploymentConditionType = "Failure"
)
