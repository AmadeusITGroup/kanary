package utils

import (
	"fmt"

	"github.com/amadeusitgroup/kanary/pkg/apis/kanary/v1alpha1"
)

// ValidateKanaryDeployment used to validate a KanaryDeployment
// return a list of errors in case of unvalid fields.
func ValidateKanaryDeployment(kd *v1alpha1.KanaryDeployment) []error {
	var errs []error
	errs = append(errs, validateKanaryDeploymentSpec(&kd.Spec)...)
	return errs
}

// validateKanaryDeploymentSpec used to validate a KanaryDeploymentSpec
// return a list of errors in case of unvalid Spec.
func validateKanaryDeploymentSpec(spec *v1alpha1.KanaryDeploymentSpec) []error {
	var errs []error
	errs = append(errs, validateKanaryDeploymentSpecScale(&spec.Scale)...)
	errs = append(errs, validateKanaryDeploymentSpecTraffic(&spec.Traffic)...)
	errs = append(errs, validateKanaryDeploymentSpecValidation(&spec.Validation)...)
	return errs
}

func validateKanaryDeploymentSpecScale(s *v1alpha1.KanaryDeploymentSpecScale) []error {
	var errs []error
	if s.Static == nil {
		errs = append(errs, fmt.Errorf("spec.scale.static not defined: %v", s))
	}
	if s.Static != nil {
		// For the moment nothing todo
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
		t.Source == v1alpha1.ShadowKanaryDeploymentSpecTrafficSource) {
		errs = append(errs, fmt.Errorf("spec.traffic.source bad value, current value:%s", t.Source))
	}

	if t.Source != v1alpha1.ShadowKanaryDeploymentSpecTrafficSource && t.Shadow != nil {
		errs = append(errs, fmt.Errorf("spec.traffic bad configuration, 'shadow' configuration provived, but 'source'=%s", t.Source))
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
