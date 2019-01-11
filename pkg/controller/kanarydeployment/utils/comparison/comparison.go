package comparison

import (
	"bytes"
	"crypto/md5" // #nosec
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"

	apps "k8s.io/api/apps/v1beta1"

	kanaryv1alpha1 "github.com/amadeusitgroup/kanary/pkg/apis/kanary/v1alpha1"
)

// IsUpToDateDeployment returns true if the Deployment is up to date with the KanaryDeployment deployment template.
func IsUpToDateDeployment(kd *kanaryv1alpha1.KanaryDeployment, dep *apps.Deployment) bool {

	hash, err := GenerateMD5DeploymentSpec(&kd.Spec.Template.Spec)
	if err != nil {
		return false
	}

	return CompareDeploymentMD5Hash(hash, dep)
}

// CompareDeploymentMD5Hash used to compare a md5 hash with the one setted in Deployment annotation
func CompareDeploymentMD5Hash(hash string, dep *apps.Deployment) bool {
	if val, ok := dep.Annotations[string(kanaryv1alpha1.MD5KanaryDeploymentAnnotationKey)]; ok && val == hash {
		return true
	}
	return false
}

// GenerateMD5DeploymentSpec used to generate the DeploymentSpec MD5 hash
func GenerateMD5DeploymentSpec(spec *apps.DeploymentSpec) (string, error) {
	b, err := json.Marshal(spec)
	if err != nil {
		return "", err
	}
	/* #nosec */
	hash := md5.New()
	_, err = io.Copy(hash, bytes.NewReader(b))
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

// SetMD5DeploymentSpecAnnotation used to set the md5 annotation key/value from the KanaryDeployement.Spec.Template.Spec
func SetMD5DeploymentSpecAnnotation(kd *kanaryv1alpha1.KanaryDeployment, dep *apps.Deployment) (string, error) {
	md5Spec, err := GenerateMD5DeploymentSpec(&kd.Spec.Template.Spec)
	if err != nil {
		return "", fmt.Errorf("unable to generates the JobSpec MD5, %v", err)
	}
	if dep.Annotations == nil {
		dep.SetAnnotations(map[string]string{})
	}
	dep.Annotations[string(kanaryv1alpha1.MD5KanaryDeploymentAnnotationKey)] = md5Spec
	return md5Spec, nil
}
