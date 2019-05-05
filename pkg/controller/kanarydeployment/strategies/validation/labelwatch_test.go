package validation

import (
	"fmt"
	"testing"
	"time"

	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"

	"k8s.io/client-go/kubernetes/scheme"

	appsv1beta1 "k8s.io/api/apps/v1beta1"
	corev1 "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	kanaryv1alpha1 "github.com/amadeusitgroup/kanary/pkg/apis/kanary/v1alpha1"
	kanaryv1alpha1test "github.com/amadeusitgroup/kanary/pkg/apis/kanary/v1alpha1/test"
	utilstest "github.com/amadeusitgroup/kanary/pkg/controller/kanarydeployment/utils/test"
)

func Test_labelWatchImpl_Validation(t *testing.T) {
	now := time.Now()
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

		validatedLabelWatchPodSpec = &kanaryv1alpha1.KanaryDeploymentSpecValidation{
			LabelWatch: &kanaryv1alpha1.KanaryDeploymentSpecValidationLabelWatch{
				PodInvalidationLabels: &metav1.LabelSelector{MatchLabels: map[string]string{"kanary": "fail"}},
			},
		}

		mapFailed = map[string]string{"failed": "true"}
	)

	type fields struct {
		validationPeriod  *metav1.Duration
		maxIntervalPeriod *metav1.Duration
		dryRun            bool
		config            *kanaryv1alpha1.KanaryDeploymentSpecValidationLabelWatch
	}
	type args struct {
		kclient   client.Client
		kd        *kanaryv1alpha1.KanaryDeployment
		dep       *appsv1beta1.Deployment
		canaryDep *appsv1beta1.Deployment
	}
	tests := []struct {
		name              string
		fields            fields
		args              args
		wantStatusSucceed bool
		wantStatusInvalid bool
		wantResult        reconcile.Result
		wantErr           bool
	}{
		{
			name: "default manual validation spec",
			fields: fields{
				validationPeriod:  &metav1.Duration{Duration: 30 * time.Second},
				maxIntervalPeriod: &metav1.Duration{Duration: 15 * time.Second},
				dryRun:            false,
				config: &kanaryv1alpha1.KanaryDeploymentSpecValidationLabelWatch{
					DeploymentInvalidationLabels: &metav1.LabelSelector{MatchLabels: mapFailed},
				},
			},
			args: args{
				kclient:   fake.NewFakeClient([]runtime.Object{utilstest.NewDeployment(name, namespace, defaultReplicas, nil)}...),
				kd:        kanaryv1alpha1test.NewKanaryDeployment(name, namespace, "", defaultReplicas, &kanaryv1alpha1test.NewKanaryDeploymentOptions{Validation: validatedLabelWatchPodSpec}),
				dep:       utilstest.NewDeployment(name, namespace, defaultReplicas, &utilstest.NewDeploymentOptions{CreationTime: creationTime}),
				canaryDep: utilstest.NewDeployment(name+"-kanary", namespace, 1, &utilstest.NewDeploymentOptions{CreationTime: creationTime}),
			},
			wantStatusSucceed: true,
			wantStatusInvalid: false,
			wantResult:        reconcile.Result{},
			wantErr:           false,
		},
		{
			name: "validation period not finished",
			fields: fields{
				validationPeriod:  &metav1.Duration{Duration: 4 * time.Minute},
				maxIntervalPeriod: &metav1.Duration{Duration: 15 * time.Second},
				dryRun:            false,
				config: &kanaryv1alpha1.KanaryDeploymentSpecValidationLabelWatch{
					DeploymentInvalidationLabels: &metav1.LabelSelector{MatchLabels: mapFailed},
				},
			},
			args: args{
				kclient:   fake.NewFakeClient([]runtime.Object{utilstest.NewDeployment(name, namespace, defaultReplicas, nil)}...),
				kd:        kanaryv1alpha1test.NewKanaryDeployment(name, namespace, "", defaultReplicas, &kanaryv1alpha1test.NewKanaryDeploymentOptions{Validation: validatedLabelWatchPodSpec}),
				dep:       utilstest.NewDeployment(name, namespace, defaultReplicas, &utilstest.NewDeploymentOptions{CreationTime: creationTime}),
				canaryDep: utilstest.NewDeployment(name+"-kanary", namespace, 1, &utilstest.NewDeploymentOptions{CreationTime: creationTime}),
			},
			wantStatusSucceed: false,
			wantStatusInvalid: false,
			wantResult:        reconcile.Result{},
			wantErr:           false,
		},
		{
			name: "dryrun, no deployment update",
			fields: fields{
				validationPeriod:  &metav1.Duration{Duration: 30 * time.Second},
				maxIntervalPeriod: &metav1.Duration{Duration: 15 * time.Second},
				dryRun:            true,
				config: &kanaryv1alpha1.KanaryDeploymentSpecValidationLabelWatch{
					DeploymentInvalidationLabels: &metav1.LabelSelector{MatchLabels: mapFailed},
				},
			},
			args: args{
				kclient:   fake.NewFakeClient([]runtime.Object{}...),
				kd:        kanaryv1alpha1test.NewKanaryDeployment(name, namespace, "", defaultReplicas, &kanaryv1alpha1test.NewKanaryDeploymentOptions{Validation: validatedLabelWatchPodSpec}),
				dep:       utilstest.NewDeployment(name, namespace, defaultReplicas, &utilstest.NewDeploymentOptions{CreationTime: creationTime}),
				canaryDep: utilstest.NewDeployment(name+"-kanary", namespace, 1, &utilstest.NewDeploymentOptions{CreationTime: creationTime}),
			},
			wantStatusSucceed: true,
			wantStatusInvalid: false,
			wantResult:        reconcile.Result{},
			wantErr:           false,
		},
		{
			name: "pod selector: validation success",
			fields: fields{
				validationPeriod:  &metav1.Duration{Duration: 30 * time.Second},
				maxIntervalPeriod: &metav1.Duration{Duration: 15 * time.Second},
				dryRun:            false,
				config: &kanaryv1alpha1.KanaryDeploymentSpecValidationLabelWatch{
					PodInvalidationLabels: &metav1.LabelSelector{MatchLabels: mapFailed},
				},
			},
			args: args{
				kclient:   fake.NewFakeClient([]runtime.Object{utilstest.NewDeployment(name, namespace, defaultReplicas, nil)}...),
				kd:        kanaryv1alpha1test.NewKanaryDeployment(name, namespace, "", defaultReplicas, &kanaryv1alpha1test.NewKanaryDeploymentOptions{Validation: validatedLabelWatchPodSpec}),
				dep:       utilstest.NewDeployment(name, namespace, defaultReplicas, &utilstest.NewDeploymentOptions{CreationTime: creationTime}),
				canaryDep: utilstest.NewDeployment(name+"-kanary", namespace, 1, &utilstest.NewDeploymentOptions{CreationTime: creationTime}),
			},
			wantStatusSucceed: true,
			wantStatusInvalid: false,
			wantResult:        reconcile.Result{},
			wantErr:           false,
		},
		{
			name: "deployment Selector: validation period not finished",
			fields: fields{
				validationPeriod:  &metav1.Duration{Duration: 4 * time.Minute},
				maxIntervalPeriod: &metav1.Duration{Duration: 15 * time.Second},
				dryRun:            false,
				config: &kanaryv1alpha1.KanaryDeploymentSpecValidationLabelWatch{
					DeploymentInvalidationLabels: &metav1.LabelSelector{MatchLabels: mapFailed},
				},
			},
			args: args{
				kclient:   fake.NewFakeClient([]runtime.Object{utilstest.NewDeployment(name, namespace, defaultReplicas, nil)}...),
				kd:        kanaryv1alpha1test.NewKanaryDeployment(name, namespace, "", defaultReplicas, &kanaryv1alpha1test.NewKanaryDeploymentOptions{Validation: validatedLabelWatchPodSpec}),
				dep:       utilstest.NewDeployment(name, namespace, defaultReplicas, &utilstest.NewDeploymentOptions{CreationTime: creationTime}),
				canaryDep: utilstest.NewDeployment(name+"-kanary", namespace, 1, &utilstest.NewDeploymentOptions{CreationTime: creationTime}),
			},
			wantStatusSucceed: false,
			wantStatusInvalid: false,
			wantResult:        reconcile.Result{},
			wantErr:           false,
		},
		{
			name: "Deployment selector: validation period not finished, label failed present",
			fields: fields{
				validationPeriod:  &metav1.Duration{Duration: 4 * time.Minute},
				maxIntervalPeriod: &metav1.Duration{Duration: 15 * time.Second},
				dryRun:            false,
				config: &kanaryv1alpha1.KanaryDeploymentSpecValidationLabelWatch{
					DeploymentInvalidationLabels: &metav1.LabelSelector{MatchLabels: mapFailed},
				},
			},
			args: args{
				kclient:   fake.NewFakeClient([]runtime.Object{utilstest.NewDeployment(name, namespace, defaultReplicas, nil)}...),
				kd:        kanaryv1alpha1test.NewKanaryDeployment(name, namespace, "", defaultReplicas, &kanaryv1alpha1test.NewKanaryDeploymentOptions{Validation: validatedLabelWatchPodSpec}),
				dep:       utilstest.NewDeployment(name, namespace, defaultReplicas, &utilstest.NewDeploymentOptions{CreationTime: creationTime}),
				canaryDep: utilstest.NewDeployment(name+"-kanary", namespace, 1, &utilstest.NewDeploymentOptions{CreationTime: creationTime, Labels: mapFailed}),
			},
			wantStatusSucceed: false,
			wantStatusInvalid: true,
			wantResult:        reconcile.Result{},
			wantErr:           false,
		},
		{
			name: "Pod selector: validation period not finished, label failed present",
			fields: fields{
				validationPeriod:  &metav1.Duration{Duration: 4 * time.Minute},
				maxIntervalPeriod: &metav1.Duration{Duration: 15 * time.Second},
				dryRun:            false,
				config: &kanaryv1alpha1.KanaryDeploymentSpecValidationLabelWatch{
					PodInvalidationLabels: &metav1.LabelSelector{MatchLabels: mapFailed},
				},
			},
			args: args{
				kclient:   fake.NewFakeClient([]runtime.Object{utilstest.NewPod(name, namespace, "hash", &utilstest.NewPodOptions{Labels: mapFailed})}...),
				kd:        kanaryv1alpha1test.NewKanaryDeployment(name, namespace, "", defaultReplicas, &kanaryv1alpha1test.NewKanaryDeploymentOptions{Validation: validatedLabelWatchPodSpec}),
				dep:       utilstest.NewDeployment(name, namespace, defaultReplicas, &utilstest.NewDeploymentOptions{CreationTime: creationTime}),
				canaryDep: utilstest.NewDeployment(name+"-kanary", namespace, 1, &utilstest.NewDeploymentOptions{CreationTime: creationTime, Labels: mapFailed}),
			},
			wantStatusSucceed: false,
			wantStatusInvalid: true,
			wantResult:        reconcile.Result{},
			wantErr:           false,
		},
	}
	for _, tt := range tests {
		reqLogger := log.WithValues("test:", tt.name)
		t.Run(tt.name, func(t *testing.T) {
			l := &labelWatchImpl{
				validationPeriod:  tt.fields.validationPeriod,
				maxIntervalPeriod: tt.fields.maxIntervalPeriod,
				dryRun:            tt.fields.dryRun,
				config:            tt.fields.config,
			}
			gotStatus, _, err := l.Validation(tt.args.kclient, reqLogger, tt.args.kd, tt.args.dep, tt.args.canaryDep)
			if (err != nil) != tt.wantErr {
				t.Errorf("labelWatchImpl.Validation() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			reqLogger.Info(fmt.Sprintf("err:%v", err))
			var gotSucceed bool
			var gotInvalid bool
			for _, cond := range gotStatus.Conditions {
				if cond.Type == kanaryv1alpha1.SucceededKanaryDeploymentConditionType && cond.Status == corev1.ConditionTrue {
					gotSucceed = true
				}
				if cond.Type == kanaryv1alpha1.FailedKanaryDeploymentConditionType && cond.Status == corev1.ConditionTrue {
					gotInvalid = true
				}
			}

			if gotSucceed != tt.wantStatusSucceed {
				t.Errorf("manualImpl.Validation() gotSucceed = %v, want %v", gotSucceed, tt.wantStatusSucceed)
			}

			if gotInvalid != tt.wantStatusInvalid {
				t.Errorf("manualImpl.Validation() gotInvalid = %v, want %v", gotInvalid, tt.wantStatusInvalid)
			}
		})
	}
}
