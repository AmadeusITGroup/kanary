package utils

import (
	"fmt"

	"github.com/amadeusitgroup/kanary/pkg/apis/kanary/v1alpha1"
)

// ValidateKanaryDeployment used to validate a KanaryDeployment
// return a list of errors in case of unvalid fields.
func ValidateKanaryDeployment(kd *v1alpha1.KanaryDeployment) []error {
	var errs []error
	errs = append(errs, validateKanaryDeploymentSpecScale(&kd.Spec.Scale)...)
	errs = append(errs, validateKanaryDeploymentSpecTraffic(&kd.Spec.Traffic)...)
	errs = append(errs, validateKanaryDeploymentSpecValidationList(&kd.Spec.Validations)...)
	return errs
}

func validateKanaryDeploymentSpecScale(s *v1alpha1.KanaryDeploymentSpecScale) []error {
	var errs []error
	if s.Static == nil {
		errs = append(errs, fmt.Errorf("spec.scale.static not defined: %v", s))
	}
	return errs
}

func validateKanaryDeploymentSpecTraffic(t *v1alpha1.KanaryDeploymentSpecTraffic) []error {
	var errs []error
	if !(t.Source == "" ||
		t.Source == v1alpha1.NoneKanaryDeploymentSpecTrafficSource ||
		t.Source == v1alpha1.ServiceKanaryDeploymentSpecTrafficSource ||
		t.Source == v1alpha1.KanaryServiceKanaryDeploymentSpecTrafficSource ||
		t.Source == v1alpha1.BothKanaryDeploymentSpecTrafficSource ||
		t.Source == v1alpha1.MirrorKanaryDeploymentSpecTrafficSource) {
		errs = append(errs, fmt.Errorf("spec.traffic.source bad value, current value:%s", t.Source))
	}

	if t.Source != v1alpha1.MirrorKanaryDeploymentSpecTrafficSource && t.Mirror != nil {
		errs = append(errs, fmt.Errorf("spec.traffic bad configuration, 'mirror' configuration provived, but 'source'=%s", t.Source))
	}

	return errs
}

func validateKanaryDeploymentSpecValidationList(list *v1alpha1.KanaryDeploymentSpecValidationList) []error {
	var errs []error
	if len(list.Items) == 0 {
		return []error{fmt.Errorf("validation list is not set")}
	}
	for _, v := range list.Items {
		errs = append(errs, validateKanaryDeploymentSpecValidation(&v)...)
	}
	return errs
}

func validateKanaryDeploymentSpecValidation(v *v1alpha1.KanaryDeploymentSpecValidation) []error {
	var errs []error
	if v.Manual == nil && v.LabelWatch == nil && v.PromQL == nil {
		errs = append(errs, fmt.Errorf("spec.validation not defined: %v", v))
	}

	return errs
}
