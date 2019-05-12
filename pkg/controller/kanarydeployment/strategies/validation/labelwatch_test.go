package validation

import (
	"reflect"
	"testing"
	"time"

	kanaryv1alpha1 "github.com/amadeusitgroup/kanary/pkg/apis/kanary/v1alpha1"
	kanaryv1alpha1test "github.com/amadeusitgroup/kanary/pkg/apis/kanary/v1alpha1/test"
	utilstest "github.com/amadeusitgroup/kanary/pkg/controller/kanarydeployment/utils/test"

	appsv1beta1 "k8s.io/api/apps/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

func Test_labelWatchImpl_Validation(t *testing.T) {
	now := time.Now()
	past := now.Add(-10 * time.Hour)
	creationTime := &metav1.Time{Time: now.Add(-2 * time.Minute)}
	logf.SetLogger(logf.ZapLogger(true))
	log := logf.Log.WithName("Test_labelWatchImpl_Validation")

	// Register operator types with the runtime scheme.
	s := scheme.Scheme
	s.AddKnownTypes(kanaryv1alpha1.SchemeGroupVersion, &kanaryv1alpha1.KanaryDeployment{})

	var (
		name = "foo"
		//	serviceName     = "foo"
		namespace       = "kanary"
		defaultReplicas = int32(5)

		validatedLabelWatchPodSpec = &kanaryv1alpha1.KanaryDeploymentSpecValidationList{
			Items: []kanaryv1alpha1.KanaryDeploymentSpecValidation{
				{
					LabelWatch: &kanaryv1alpha1.KanaryDeploymentSpecValidationLabelWatch{
						PodInvalidationLabels: &metav1.LabelSelector{MatchLabels: map[string]string{"kanary": "fail"}},
					},
				},
			},
		}
		validatedLabelWatchPodSpecNotFinished = &kanaryv1alpha1.KanaryDeploymentSpecValidationList{
			Items: []kanaryv1alpha1.KanaryDeploymentSpecValidation{
				{
					LabelWatch: &kanaryv1alpha1.KanaryDeploymentSpecValidationLabelWatch{
						PodInvalidationLabels: &metav1.LabelSelector{MatchLabels: map[string]string{"kanary": "fail"}},
					},
				},
			},
			ValidationPeriod: &metav1.Duration{Duration: 4 * time.Minute},
		}

		mapFailed = map[string]string{"failed": "true"}
	)

	type fields struct {
		dryRun bool
		config *kanaryv1alpha1.KanaryDeploymentSpecValidationLabelWatch
	}
	type args struct {
		kclient   client.Client
		kd        *kanaryv1alpha1.KanaryDeployment
		dep       *appsv1beta1.Deployment
		canaryDep *appsv1beta1.Deployment
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *Result
		wantErr bool
	}{
		{
			name: "default manual validation spec",
			fields: fields{
				dryRun: false,
				config: &kanaryv1alpha1.KanaryDeploymentSpecValidationLabelWatch{
					DeploymentInvalidationLabels: &metav1.LabelSelector{MatchLabels: mapFailed},
				},
			},
			args: args{
				kclient:   fake.NewFakeClient([]runtime.Object{utilstest.NewDeployment(name, namespace, defaultReplicas, nil)}...),
				kd:        kanaryv1alpha1test.NewKanaryDeployment(name, namespace, "", defaultReplicas, &kanaryv1alpha1test.NewKanaryDeploymentOptions{StartTime: &metav1.Time{Time: past}, Validations: validatedLabelWatchPodSpec}),
				dep:       utilstest.NewDeployment(name, namespace, defaultReplicas, &utilstest.NewDeploymentOptions{CreationTime: creationTime}),
				canaryDep: utilstest.NewDeployment(name+"-kanary", namespace, 1, &utilstest.NewDeploymentOptions{CreationTime: creationTime}),
			},
			want: &Result{
				IsFailed: false,
			},
			wantErr: false,
		},
		{
			name: "default manual validation spec, dryRun",
			fields: fields{
				dryRun: true,
				config: &kanaryv1alpha1.KanaryDeploymentSpecValidationLabelWatch{
					DeploymentInvalidationLabels: &metav1.LabelSelector{MatchLabels: mapFailed},
				},
			},
			args: args{
				kclient:   fake.NewFakeClient([]runtime.Object{utilstest.NewDeployment(name, namespace, defaultReplicas, nil)}...),
				kd:        kanaryv1alpha1test.NewKanaryDeployment(name, namespace, "", defaultReplicas, &kanaryv1alpha1test.NewKanaryDeploymentOptions{StartTime: &metav1.Time{Time: past}, Validations: validatedLabelWatchPodSpec}),
				dep:       utilstest.NewDeployment(name, namespace, defaultReplicas, &utilstest.NewDeploymentOptions{CreationTime: creationTime}),
				canaryDep: utilstest.NewDeployment(name+"-kanary", namespace, 1, &utilstest.NewDeploymentOptions{CreationTime: creationTime}),
			},
			want: &Result{
				IsFailed: false,
			},
			wantErr: false,
		},
		{
			name: "validation period not finished",
			fields: fields{
				dryRun: false,
				config: &kanaryv1alpha1.KanaryDeploymentSpecValidationLabelWatch{
					DeploymentInvalidationLabels: &metav1.LabelSelector{MatchLabels: mapFailed},
				},
			},
			args: args{
				kclient:   fake.NewFakeClient([]runtime.Object{utilstest.NewDeployment(name, namespace, defaultReplicas, nil)}...),
				kd:        kanaryv1alpha1test.NewKanaryDeployment(name, namespace, "", defaultReplicas, &kanaryv1alpha1test.NewKanaryDeploymentOptions{StartTime: &metav1.Time{Time: now}, Validations: validatedLabelWatchPodSpecNotFinished}),
				dep:       utilstest.NewDeployment(name, namespace, defaultReplicas, &utilstest.NewDeploymentOptions{CreationTime: creationTime}),
				canaryDep: utilstest.NewDeployment(name+"-kanary", namespace, 1, &utilstest.NewDeploymentOptions{CreationTime: creationTime}),
			},
			want: &Result{
				IsFailed: false,
			},
			wantErr: false,
		},
		{
			name: "pod selector: validation success",
			fields: fields{
				dryRun: false,
				config: &kanaryv1alpha1.KanaryDeploymentSpecValidationLabelWatch{
					PodInvalidationLabels: &metav1.LabelSelector{MatchLabels: mapFailed},
				},
			},
			args: args{
				kclient:   fake.NewFakeClient([]runtime.Object{utilstest.NewDeployment(name, namespace, defaultReplicas, nil)}...),
				kd:        kanaryv1alpha1test.NewKanaryDeployment(name, namespace, "", defaultReplicas, &kanaryv1alpha1test.NewKanaryDeploymentOptions{StartTime: &metav1.Time{Time: past}, Validations: validatedLabelWatchPodSpec}),
				dep:       utilstest.NewDeployment(name, namespace, defaultReplicas, &utilstest.NewDeploymentOptions{CreationTime: creationTime}),
				canaryDep: utilstest.NewDeployment(name+"-kanary", namespace, 1, &utilstest.NewDeploymentOptions{CreationTime: creationTime}),
			},
			want: &Result{
				IsFailed: false,
			},
			wantErr: false,
		},
		{
			name: "deployment Selector: validation period not finished",
			fields: fields{
				dryRun: false,
				config: &kanaryv1alpha1.KanaryDeploymentSpecValidationLabelWatch{
					DeploymentInvalidationLabels: &metav1.LabelSelector{MatchLabels: mapFailed},
				},
			},
			args: args{
				kclient:   fake.NewFakeClient([]runtime.Object{utilstest.NewDeployment(name, namespace, defaultReplicas, nil)}...),
				kd:        kanaryv1alpha1test.NewKanaryDeployment(name, namespace, "", defaultReplicas, &kanaryv1alpha1test.NewKanaryDeploymentOptions{StartTime: &metav1.Time{Time: now}, Validations: validatedLabelWatchPodSpec}),
				dep:       utilstest.NewDeployment(name, namespace, defaultReplicas, &utilstest.NewDeploymentOptions{CreationTime: creationTime}),
				canaryDep: utilstest.NewDeployment(name+"-kanary", namespace, 1, &utilstest.NewDeploymentOptions{CreationTime: creationTime}),
			},
			want: &Result{
				IsFailed: false,
			},
			wantErr: false,
		},
		{
			name: "Deployment selector: validation period not finished, label failed present",
			fields: fields{
				dryRun: false,
				config: &kanaryv1alpha1.KanaryDeploymentSpecValidationLabelWatch{
					DeploymentInvalidationLabels: &metav1.LabelSelector{MatchLabels: mapFailed},
				},
			},
			args: args{
				kclient:   fake.NewFakeClient([]runtime.Object{utilstest.NewDeployment(name, namespace, defaultReplicas, nil)}...),
				kd:        kanaryv1alpha1test.NewKanaryDeployment(name, namespace, "", defaultReplicas, &kanaryv1alpha1test.NewKanaryDeploymentOptions{StartTime: &metav1.Time{Time: now}, Validations: validatedLabelWatchPodSpec}),
				dep:       utilstest.NewDeployment(name, namespace, defaultReplicas, &utilstest.NewDeploymentOptions{CreationTime: creationTime}),
				canaryDep: utilstest.NewDeployment(name+"-kanary", namespace, 1, &utilstest.NewDeploymentOptions{CreationTime: creationTime, Labels: mapFailed}),
			},
			want: &Result{
				IsFailed: true,
				Comment:  "labelWatch has detected invalidation labels",
			},
			wantErr: false,
		},
		{
			name: "Pod selector: validation period not finished, label failed present",
			fields: fields{
				dryRun: false,
				config: &kanaryv1alpha1.KanaryDeploymentSpecValidationLabelWatch{
					PodInvalidationLabels: &metav1.LabelSelector{MatchLabels: mapFailed},
				},
			},
			args: args{
				kclient:   fake.NewFakeClient([]runtime.Object{utilstest.NewPod(name, namespace, "hash", &utilstest.NewPodOptions{Labels: mapFailed})}...),
				kd:        kanaryv1alpha1test.NewKanaryDeployment(name, namespace, "", defaultReplicas, &kanaryv1alpha1test.NewKanaryDeploymentOptions{StartTime: &metav1.Time{Time: now}, Validations: validatedLabelWatchPodSpec}),
				dep:       utilstest.NewDeployment(name, namespace, defaultReplicas, &utilstest.NewDeploymentOptions{CreationTime: creationTime}),
				canaryDep: utilstest.NewDeployment(name+"-kanary", namespace, 1, &utilstest.NewDeploymentOptions{CreationTime: creationTime, Labels: mapFailed}),
			},
			want: &Result{
				IsFailed: true,
				Comment:  "labelWatch has detected invalidation labels",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqLogger := log.WithValues("test:", tt.name)
			l := &labelWatchImpl{
				dryRun: tt.fields.dryRun,
				config: tt.fields.config,
			}
			got, err := l.Validation(tt.args.kclient, reqLogger, tt.args.kd, tt.args.dep, tt.args.canaryDep)
			if (err != nil) != tt.wantErr {
				t.Errorf("labelWatchImpl.Validation() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("labelWatchImpl.Validation() = %#v, want %#v", got, tt.want)
			}
		})
	}
}
