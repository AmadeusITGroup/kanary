package v1alpha1

import (
	"k8s.io/api/apps/v1beta1"
	"k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func init() {
	SchemeBuilder.Register(&KanaryDeployment{}, &KanaryDeploymentList{})
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

// KanaryDeploymentSpec defines the desired state of KanaryDeployment
type KanaryDeploymentSpec struct {
	// DeploymentName is the name of the deployment that will be updated in case of a success
	// canary deployment testing
	// if DeploymentName is empty or not define. The KanaryDeployment will search for a Deployment with
	// same name than the KanaryDeployment. If the deployment not exist, the deployment will be created
	// with the deployment template present in the KanaryDeployment.
	DeploymentName string `json:"deploymentName,omitempty"`
	// serviceName is the name of the service that governs the associated Deployment.
	// This service can be empty of not defined, which means that some Kanary feature will not be
	// applied on the KanaryDeployment.
	ServiceName string `json:"serviceName,omitempty"`
	// Template  is the object that describes the deployment that will be created.
	Template DeploymentTemplate `json:"template,omitempty"`
	// Scale is the scaling configuration for the canary deployment
	Scale KanaryDeploymentSpecScale `json:"scale,omitempty"`
	// Traffic is the scaling configuration for the canary deployment
	Traffic KanaryDeploymentSpecTraffic `json:"traffic,omitempty"`
	// Validation is the scaling configuration for the canary deployment
	Validation KanaryDeploymentSpecValidation `json:"validation,omitempty"`
}

// KanaryDeploymentSpecScale defines the scale configuration for the canary deployment
type KanaryDeploymentSpecScale struct {
	Static *KanaryDeploymentSpecScaleStatic `json:"static,omitempty"`
}

// KanaryDeploymentSpecScaleStatic defines the static scale configuration for the canary deployment
type KanaryDeploymentSpecScaleStatic struct {
	// Number of desired pods. This is a pointer to distinguish between explicit
	// zero and not specified. Defaults to 1.
	// +optional
	Replicas *int32 `json:"replicas,omitempty"`
}

// KanaryDeploymentSpecTraffic defines the traffic configuration for the canary deployment
type KanaryDeploymentSpecTraffic struct {
	// Source defines the traffic source that targets the canary deployment pods
	Source KanaryDeploymentSpecTrafficSource `json:"source,omitempty"`
	// KanaryService is the name of the service that will be created to target specificaly
	// pods that serve the canary service version.
	// if kanaryService is empty or not define, a service name will be generated from the
	// serviceName provided in the KanaryDeploymentSpec.
	KanaryService string `json:"kanaryService,omitempty"`
	// Shadow
	Shadow *KanaryDeploymentSpecTrafficShadow `json:"shadow,omitempty"`
}

// KanaryDeploymentSpecTrafficSource defines the traffic source that targets the canary deployment pods
type KanaryDeploymentSpecTrafficSource string

const (
	// ServiceKanaryDeploymentSpecTrafficSource means that deployment service also target the canary deployment
	ServiceKanaryDeploymentSpecTrafficSource KanaryDeploymentSpecTrafficSource = "service"
	// KanaryServiceKanaryDeploymentSpecTrafficSource means that a dedicated service is created to target the canary deployment pods.
	KanaryServiceKanaryDeploymentSpecTrafficSource KanaryDeploymentSpecTrafficSource = "kanary-service"
	// BothKanaryDeploymentSpecTrafficSource means canary deployment pods are targetable thank the deployment service but also
	// with a the create kanary service.
	BothKanaryDeploymentSpecTrafficSource KanaryDeploymentSpecTrafficSource = "both"
	// NoneKanaryDeploymentSpecTrafficSource means the canary deployment pods are not accessible. it can be use when the application
	// don't define any service.
	NoneKanaryDeploymentSpecTrafficSource KanaryDeploymentSpecTrafficSource = "none"
	// ShadowKanaryDeploymentSpecTrafficSource means that the canary deployment pods are target by a shadow traffic.
	ShadowKanaryDeploymentSpecTrafficSource KanaryDeploymentSpecTrafficSource = "shadow"
)

// KanaryDeploymentSpecTrafficShadow define the activation of shadow traffic on canary pods
type KanaryDeploymentSpecTrafficShadow struct {
	Activate bool `json:"activate"`
}

// KanaryDeploymentSpecValidation defines the validation configuration for the canary deployment
type KanaryDeploymentSpecValidation struct {
	ValidationPeriod *metav1.Duration                          `json:"validationPeriod,omitempty"`
	Manual           *KanaryDeploymentSpecValidationManual     `json:"manual,omitempty"`
	LabelWatch       *KanaryDeploymentSpecValidationLabelWatch `json:"labelWatch,omitempty"`
	PromQL           *KanaryDeploymentSpecValidationPromQL     `json:"promQL,omitempty"`
}

// KanaryDeploymentSpecValidationManual defines the manual validation configuration
type KanaryDeploymentSpecValidationManual struct {
	Deadline KanaryDeploymentSpecValidationManualDeadine `json:"deadline,omitempty"`
	Status   KanaryDeploymentSpecValidationManualStatus  `json:"status,omitempty"`
}

// KanaryDeploymentSpecValidationManualDeadine defines the validation manual deadine mode
type KanaryDeploymentSpecValidationManualDeadine string

const (
	// NoneKanaryDeploymentSpecValidationManualDeadine means deadline is not activated.
	NoneKanaryDeploymentSpecValidationManualDeadine KanaryDeploymentSpecValidationManualDeadine = "none"
	// ValidKanaryDeploymentSpecValidationManualDeadine means that after the validation.ValidationPeriod
	// if the validation.manual.status is not set properly the KanaryDeployment will be considered as "valid"
	ValidKanaryDeploymentSpecValidationManualDeadine KanaryDeploymentSpecValidationManualDeadine = "valid"
	// InvalidKanaryDeploymentSpecValidationManualDeadine means that after the validation.ValidationPeriod
	// if the validation.manual.status is not set properly the KanaryDeployment will be considered as "invalid"
	InvalidKanaryDeploymentSpecValidationManualDeadine KanaryDeploymentSpecValidationManualDeadine = "invalid"
)

// KanaryDeploymentSpecValidationManualStatus defines the KanaryDeployment validation status in case of manual validation.
type KanaryDeploymentSpecValidationManualStatus string

const (
	// ValidKanaryDeploymentSpecValidationManualStatus means that the KanaryDeployment have been validated sucessfuly.
	ValidKanaryDeploymentSpecValidationManualStatus KanaryDeploymentSpecValidationManualStatus = "valid"
	// InvalidKanaryDeploymentSpecValidationManualStatus means that the KanaryDeployment have been invalidated.
	InvalidKanaryDeploymentSpecValidationManualStatus KanaryDeploymentSpecValidationManualStatus = "invalid"
)

// KanaryDeploymentSpecValidationLabelWatch defines the labelWatch validation configuration
type KanaryDeploymentSpecValidationLabelWatch struct {
	// PodSelector defines labels that should be present on the canary pods in order to validate
	// the canary deployment
	PodSelector *metav1.LabelSelector `json:"podSelector,omitempty"`
	// DeploymentSelector defines labels that should be present on the canary deployment in order to validate
	// the canary deployment
	DeploymentSelector *metav1.LabelSelector `json:"deploymentSelector,omitempty"`
}

// KanaryDeploymentSpecValidationPromQL defines the promQL validation configuration
type KanaryDeploymentSpecValidationPromQL struct {
	// Query defines the promQL query that will inform if the canary validation is successful or not.
	// the query should return "True" or "False"
	Query string `json:"query,omitempty"`
	// ServerURL defines the prometheus server URL
	ServerURL string `json:"serverURL,omitempty"`
}

// KanaryDeploymentStatus defines the observed state of KanaryDeployment
type KanaryDeploymentStatus struct {
	// CurrentHash represents the current MD5 spec deployment template hash
	CurrentHash string `json:"currentHash,omitempty"`
	// Represents the latest available observations of a kanarydeployment's current state.
	Conditions []KanaryDeploymentCondition `json:"conditions,omitempty"`
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
	// Activated means the KanaryDeployment strategy is activated
	ActivatedKanaryDeploymentConditionType KanaryDeploymentConditionType = "Activated"
	// Successeed means the KanaryDeployment strategy succeed,
	// the deployment rolling-update is in progress or already done.
	// it means also the deployment and the canary deployment have the same version.
	SucceededKanaryDeploymentConditionType KanaryDeploymentConditionType = "Successeed"
	// FailedKanaryDeploymentConditionType is added in a kanarydeployment when the canary deployment
	// process failed.
	FailedKanaryDeploymentConditionType KanaryDeploymentConditionType = "Failed"
	// ErroredKanaryDeploymentConditionType is added in a kanarydeployment when the canary deployment
	// process errored.
	ErroredKanaryDeploymentConditionType KanaryDeploymentConditionType = "Errored"
)

// KanaryDeploymentAnnotationKeyType corresponds to all possible Annotation Keys that can be added/updated by Kanary
type KanaryDeploymentAnnotationKeyType string

const (
	// MD5KanaryDeploymentAnnotationKey correspond to the annotation key for the deployment template md5 used to create the deployment.
	MD5KanaryDeploymentAnnotationKey KanaryDeploymentAnnotationKeyType = "kanary.k8s.io/md5"
)

const (
	// KanaryDeploymentIsKanaryLabelKey correspond to the label key used on a deployment to inform
	// that this instance is used in a canary deployment.
	KanaryDeploymentIsKanaryLabelKey = "kanary.k8s.io/iskanary"
	// KanaryDeploymentKanaryNameLabelKey correspond to the label key used on a deployment and pod to provide the KanaryDeployment name.
	KanaryDeploymentKanaryNameLabelKey = "kanary.k8s.io/name"
	// KanaryDeploymentActivateLabelKey correspond to the label key used on a pod to inform that this
	// Pod instance in a canary version of the application.
	KanaryDeploymentActivateLabelKey = "kanary.k8s.io/canary-pod"
	// KanaryDeploymentLabelValueTrue correspond to the label value True used with several Kanary label keys.
	KanaryDeploymentLabelValueTrue = "true"
	// KanaryDeploymentLabelValueFalse correspond to the label value False used with several Kanary label keys.
	KanaryDeploymentLabelValueFalse = "false"
)
