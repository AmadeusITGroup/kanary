package v1alpha1

import (
	"k8s.io/api/apps/v1beta1"
	"k8s.io/api/autoscaling/v2beta1"
	v1 "k8s.io/api/core/v1"

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
	// Validations is the scaling configuration for the canary deployment
	Validations KanaryDeploymentSpecValidationList `json:"validations,omitempty"`
	// Schedule helps you to define when that canary deployment should start. RFC3339 = "2006-01-02T15:04:05Z07:00" "2006-01-02T15:04:05Z"
	Schedule string `json:"schedule,omiempty"`
}

// KanaryDeploymentSpecScale defines the scale configuration for the canary deployment
type KanaryDeploymentSpecScale struct {
	Static *KanaryDeploymentSpecScaleStatic `json:"static,omitempty"`
	HPA    *HorizontalPodAutoscalerSpec     `json:"hpa,omitempty"`
}

// KanaryDeploymentSpecScaleStatic defines the static scale configuration for the canary deployment
type KanaryDeploymentSpecScaleStatic struct {
	// Number of desired pods. This is a pointer to distinguish between explicit
	// zero and not specified. Defaults to 1.
	// +optional
	Replicas *int32 `json:"replicas,omitempty"`
}

// HorizontalPodAutoscalerSpec describes the desired functionality of the HorizontalPodAutoscaler.
type HorizontalPodAutoscalerSpec struct {
	// minReplicas is the lower limit for the number of replicas to which the autoscaler can scale down.
	// It defaults to 1 pod.
	// +optional
	MinReplicas *int32 `json:"minReplicas,omitempty" protobuf:"varint,2,opt,name=minReplicas"`
	// maxReplicas is the upper limit for the number of replicas to which the autoscaler can scale up.
	// It cannot be less that minReplicas.
	MaxReplicas int32 `json:"maxReplicas" protobuf:"varint,3,opt,name=maxReplicas"`
	// metrics contains the specifications for which to use to calculate the
	// desired replica count (the maximum replica count across all metrics will
	// be used).  The desired replica count is calculated multiplying the
	// ratio between the target value and the current value by the current
	// number of pods.  Ergo, metrics used must decrease as the pod count is
	// increased, and vice-versa.  See the individual metric source types for
	// more information about how each type of metric must respond.
	// +optional
	Metrics []v2beta1.MetricSpec `json:"metrics,omitempty" protobuf:"bytes,4,rep,name=metrics"`
}

// KanaryDeploymentSpecTraffic defines the traffic configuration for the canary deployment
type KanaryDeploymentSpecTraffic struct {
	// Source defines the traffic source that targets the canary deployment pods
	Source KanaryDeploymentSpecTrafficSource `json:"source,omitempty"`
	// KanaryService is the name of the service that will be created to target specifically
	// pods that serve the canary service version.
	// if kanaryService is empty or not define, a service name will be generated from the
	// serviceName provided in the KanaryDeploymentSpec.
	KanaryService string `json:"kanaryService,omitempty"`
	// Mirror
	Mirror *KanaryDeploymentSpecTrafficMirror `json:"mirror,omitempty"`
}

// KanaryDeploymentSpecTrafficSource defines the traffic source that targets the canary deployment pods
type KanaryDeploymentSpecTrafficSource string

const (
	// ServiceKanaryDeploymentSpecTrafficSource means that deployment service also target the canary deployment. Normal service discovery and loadbalacing done by kubernetes will be applied.
	ServiceKanaryDeploymentSpecTrafficSource KanaryDeploymentSpecTrafficSource = "service"
	// KanaryServiceKanaryDeploymentSpecTrafficSource means that a dedicated service is created to target the canary deployment pods. The canary pods do not receive traffic from the classic service.
	KanaryServiceKanaryDeploymentSpecTrafficSource KanaryDeploymentSpecTrafficSource = "kanary-service"
	// BothKanaryDeploymentSpecTrafficSource means canary deployment pods are targetable thank the deployment service but also
	// with a the create kanary service.
	BothKanaryDeploymentSpecTrafficSource KanaryDeploymentSpecTrafficSource = "both"
	// NoneKanaryDeploymentSpecTrafficSource means the canary deployment pods are not accessible. it can be use when the application
	// don't define any service.
	NoneKanaryDeploymentSpecTrafficSource KanaryDeploymentSpecTrafficSource = "none"
	// MirrorKanaryDeploymentSpecTrafficSource means that the canary deployment pods are target by a mirror traffic. This can be done only if istio is installed.
	MirrorKanaryDeploymentSpecTrafficSource KanaryDeploymentSpecTrafficSource = "mirror"
)

// KanaryDeploymentSpecTrafficMirror define the activation of mirror traffic on canary pods
type KanaryDeploymentSpecTrafficMirror struct {
	Activate bool `json:"activate"`
}

// KanaryDeploymentSpecValidationList define list of KanaryDeploymentSpecValidation
type KanaryDeploymentSpecValidationList struct {
	// InitialDelay duration after the KanaryDeployment has started before validation checks is started.
	InitialDelay *metav1.Duration `json:"initialDelay,omitempty"`
	// ValidationPeriod validation checks duration.
	ValidationPeriod *metav1.Duration `json:"validationPeriod,omitempty"`
	// MaxIntervalPeriod max interval duration between two validation tentative
	MaxIntervalPeriod *metav1.Duration `json:"maxIntervalPeriod,omitempty"`
	// NoUpdate if set to true, the Deployment will no be updated after a succeed validation period.
	NoUpdate bool `json:"noUpdate,omitempty"`
	// Items list of KanaryDeploymentSpecValidation
	Items []KanaryDeploymentSpecValidation `json:"items,omitempty"`
}

// KanaryDeploymentSpecValidation defines the validation configuration for the canary deployment
type KanaryDeploymentSpecValidation struct {
	Manual     *KanaryDeploymentSpecValidationManual     `json:"manual,omitempty"`
	LabelWatch *KanaryDeploymentSpecValidationLabelWatch `json:"labelWatch,omitempty"`
	PromQL     *KanaryDeploymentSpecValidationPromQL     `json:"promQL,omitempty"`
}

// KanaryDeploymentSpecValidationManual defines the manual validation configuration
type KanaryDeploymentSpecValidationManual struct {
	StatusAfterDealine KanaryDeploymentSpecValidationManualDeadineStatus `json:"statusAfterDeadline,omitempty"`
	Status             KanaryDeploymentSpecValidationManualStatus        `json:"status,omitempty"`
}

// KanaryDeploymentSpecValidationManualDeadineStatus defines the validation manual deadine mode
type KanaryDeploymentSpecValidationManualDeadineStatus string

const (
	// NoneKanaryDeploymentSpecValidationManualDeadineStatus means deadline is not activated.
	NoneKanaryDeploymentSpecValidationManualDeadineStatus KanaryDeploymentSpecValidationManualDeadineStatus = "none"
	// ValidKanaryDeploymentSpecValidationManualDeadineStatus means that after the validation.ValidationPeriod
	// if the validation.manual.status is not set properly the KanaryDeployment will be considered as "valid"
	ValidKanaryDeploymentSpecValidationManualDeadineStatus KanaryDeploymentSpecValidationManualDeadineStatus = "valid"
	// InvalidKanaryDeploymentSpecValidationManualDeadineStatus means that after the validation.ValidationPeriod
	// if the validation.manual.status is not set properly the KanaryDeployment will be considered as "invalid"
	InvalidKanaryDeploymentSpecValidationManualDeadineStatus KanaryDeploymentSpecValidationManualDeadineStatus = "invalid"
)

// KanaryDeploymentSpecValidationManualStatus defines the KanaryDeployment validation status in case of manual validation.
type KanaryDeploymentSpecValidationManualStatus string

const (
	// ValidKanaryDeploymentSpecValidationManualStatus means that the KanaryDeployment have been validated successfully.
	ValidKanaryDeploymentSpecValidationManualStatus KanaryDeploymentSpecValidationManualStatus = "valid"
	// InvalidKanaryDeploymentSpecValidationManualStatus means that the KanaryDeployment have been invalidated.
	InvalidKanaryDeploymentSpecValidationManualStatus KanaryDeploymentSpecValidationManualStatus = "invalid"
)

// KanaryDeploymentSpecValidationLabelWatch defines the labelWatch validation configuration
type KanaryDeploymentSpecValidationLabelWatch struct {
	// PodInvalidationLabels defines labels that should be present on the canary pods in order to invalidate
	// the canary deployment
	PodInvalidationLabels *metav1.LabelSelector `json:"podInvalidationLabels,omitempty"`
	// DeploymentInvalidationLabels defines labels that should be present on the canary deployment in order to invalidate
	// the canary deployment
	DeploymentInvalidationLabels *metav1.LabelSelector `json:"deploymentInvalidationLabels,omitempty"`
}

// KanaryDeploymentSpecValidationPromQL defines the promQL validation configuration
type KanaryDeploymentSpecValidationPromQL struct {
	PrometheusService string `json:"prometheusService"`
	Query             string `json:"query"` //The promQL query
	// note the AND close that prevent to return record when there is less that 70 records over the floating time window of 1m
	PodNameKey               string                    `json:"podNamekey"`   // Key to access the podName
	AllPodsQuery             bool                      `json:"allPodsQuery"` // This indicate that the query will return a result that is applicable to all pods. The pod dimension and so the PodNameKey is not taken into account. Default value is false.
	ValueInRange             *ValueInRange             `json:"valueInRange,omitempty"`
	DiscreteValueOutOfList   *DiscreteValueOutOfList   `json:"discreteValueOutOfList,omitempty"`
	ContinuousValueDeviation *ContinuousValueDeviation `json:"continuousValueDeviation,omitempty"`
}

// ValueInRange detect anomaly when the value returned is not inside the defined range
type ValueInRange struct {
	Min *float64 `json:"min"` // Min , the lower bound of the range. Default value is 0.0
	Max *float64 `json:"max"` // Max , the upper bound of the range. Default value is 1.0
}

// ContinuousValueDeviation detect anomaly when the average value for a pod is deviating from the average for the fleet of pods. If a pods does not register enough event it should not be returned by the PromQL
// The promQL should return value that are grouped by:
// 1- the podname
type ContinuousValueDeviation struct {
	//PromQL example, deviation compare to global average: (rate(solution_price_sum[1m])/rate(solution_price_count[1m]) and delta(solution_price_count[1m])>70) / scalar(sum(rate(solution_price_sum[1m]))/sum(rate(solution_price_count[1m])))
	MaxDeviationPercent *float64 `json:"maxDeviationPercent"` // MaxDeviationPercent maxDeviation computation based on % of the mean
}

// DiscreteValueOutOfList detect anomaly when the a value is not in the list with a ratio that exceed the tolerance
// The promQL should return counter that are grouped by:
// 1-the key of the value to monitor
// 2-the podname
type DiscreteValueOutOfList struct {
	//PromQL example: sum(delta(ms_rpc_count{job=\"kubernetes-pods\",run=\"foo\"}[10s])) by (code,kubernetes_pod_name)
	Key              string   `json:"key"`                  // Key for the metrics. For the previous example it will be "code"
	GoodValues       []string `json:"goodValues,omitempty"` // Good Values ["200","201"]. If empty means that BadValues should be used to do exclusion instead of inclusion.
	BadValues        []string `json:"badValues,omitempty"`  // Bad Values ["500","404"].
	TolerancePercent *uint    `json:"tolerance"`            // % of Bad values tolerated until the pod is considered out of SLA
}

// KanaryDeploymentStatus defines the observed state of KanaryDeployment
type KanaryDeploymentStatus struct {
	// CurrentHash represents the current MD5 spec deployment template hash
	CurrentHash string `json:"currentHash,omitempty"`
	// Represents the latest available observations of a kanarydeployment's current state.
	Conditions []KanaryDeploymentCondition `json:"conditions,omitempty"`
	// Report
	Report KanaryDeploymentStatusReport `json:"report,omitempty"`
}

type KanaryDeploymentStatusReport struct {
	Status     string `json:"status,omitempty"`
	Validation string `json:"validation,omitempty"`
	Scale      string `json:"scale,omitempty"`
	Traffic    string `json:"traffic,omitempty"`
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
	ScheduledKanaryDeploymentConditionType KanaryDeploymentConditionType = "Scheduled"
	// Activated means the KanaryDeployment strategy is activated
	ActivatedKanaryDeploymentConditionType KanaryDeploymentConditionType = "Activated"
	// Succeeded means the KanaryDeployment strategy succeed,
	// the deployment rolling-update is in progress or already done.
	// it means also the deployment and the canary deployment have the same version.
	SucceededKanaryDeploymentConditionType KanaryDeploymentConditionType = "Succeeded"
	// FailedKanaryDeploymentConditionType is added in a kanarydeployment when the canary deployment
	// process failed.
	FailedKanaryDeploymentConditionType KanaryDeploymentConditionType = "Failed"
	// RunningKanaryDeploymentConditionType is added in a kanarydeployment when the canary is still under validation.
	RunningKanaryDeploymentConditionType KanaryDeploymentConditionType = "Running"
	// DeploymentUpdated is added in a kanarydeployment when the canary succeded and that the deployment was updated
	DeploymentUpdatedKanaryDeploymentConditionType KanaryDeploymentConditionType = "DeploymentUpdated"

	// ErroredKanaryDeploymentConditionType is added in a kanarydeployment when the canary deployment
	// process errored.
	ErroredKanaryDeploymentConditionType KanaryDeploymentConditionType = "Errored"
	// TrafficServiceKanaryDeploymentConditionType means the KanaryDeployment Traffic strategy is activated
	TrafficKanaryDeploymentConditionType KanaryDeploymentConditionType = "Traffic"
)

// KanaryDeploymentAnnotationKeyType corresponds to all possible Annotation Keys that can be added/updated by Kanary
type KanaryDeploymentAnnotationKeyType string

const (
	// MD5KanaryDeploymentAnnotationKey correspond to the annotation key for the deployment template md5 used to create the deployment.
	MD5KanaryDeploymentAnnotationKey KanaryDeploymentAnnotationKeyType = "kanary.k8s-operators.dev/md5"
)

const (
	// KanaryDeploymentIsKanaryLabelKey correspond to the label key used on a deployment to inform
	// that this instance is used in a canary deployment.
	KanaryDeploymentIsKanaryLabelKey = "kanary.k8s-operators.dev/iskanary"
	// KanaryDeploymentKanaryNameLabelKey correspond to the label key used on a deployment and pod to provide the KanaryDeployment name.
	KanaryDeploymentKanaryNameLabelKey = "kanary.k8s-operators.dev/name"
	// KanaryDeploymentActivateLabelKey correspond to the label key used on a pod to inform that this
	// Pod instance in a canary version of the application.
	KanaryDeploymentActivateLabelKey = "kanary.k8s-operators.dev/canary-pod"
	// KanaryDeploymentLabelValueTrue correspond to the label value True used with several Kanary label keys.
	KanaryDeploymentLabelValueTrue = "true"
	// KanaryDeploymentLabelValueFalse correspond to the label value False used with several Kanary label keys.
	KanaryDeploymentLabelValueFalse = "false"
)
