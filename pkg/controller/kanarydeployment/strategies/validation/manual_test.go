package validation

import (
	"reflect"
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

func Test_manualImpl_Validation(t *testing.T) {
	now := time.Now()
	creationTime := metav1.Time{Time: now.Add(-20 * time.Minute)}
	logf.SetLogger(logf.ZapLogger(true))
	log := logf.Log.WithName("Test_manualImpl_Validation")

	// Register operator types with the runtime scheme.
	s := scheme.Scheme
	s.AddKnownTypes(kanaryv1alpha1.SchemeGroupVersion, &kanaryv1alpha1.KanaryDeployment{})

	var (
		name = "foo"
		//	serviceName     = "foo"
		namespace       = "kanary"
		defaultReplicas = int32(5)

		defaultValidationSpec = &kanaryv1alpha1.KanaryDeploymentSpecValidation{
			Manual: &kanaryv1alpha1.KanaryDeploymentSpecValidationManual{},
		}

		validatedManualSpec = &kanaryv1alpha1.KanaryDeploymentSpecValidation{
			Manual: &kanaryv1alpha1.KanaryDeploymentSpecValidationManual{
				Status: kanaryv1alpha1.ValidKanaryDeploymentSpecValidationManualStatus,
			},
		}
	)

	type fields struct {
		deadline         kanaryv1alpha1.KanaryDeploymentSpecValidationManualDeadineStatus
		status           kanaryv1alpha1.KanaryDeploymentSpecValidationManualStatus
		validationPeriod time.Duration
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
			args: args{
				kclient:   fake.NewFakeClient([]runtime.Object{}...),
				kd:        kanaryv1alpha1test.NewKanaryDeployment(name, namespace, "", defaultReplicas, &kanaryv1alpha1test.NewKanaryDeploymentOptions{Validation: defaultValidationSpec}),
				dep:       utilstest.NewDeployment(name, namespace, defaultReplicas, nil),
				canaryDep: utilstest.NewDeployment(name+"-kanary", namespace, 1, nil),
			},
			wantStatusSucceed: false,
			wantResult:        reconcile.Result{},
			wantErr:           false,
		},

		{
			name: "validation manual validated",
			fields: fields{
				deadline: kanaryv1alpha1.NoneKanaryDeploymentSpecValidationManualDeadineStatus,
				status:   kanaryv1alpha1.ValidKanaryDeploymentSpecValidationManualStatus,
			},
			args: args{
				kd:        kanaryv1alpha1test.NewKanaryDeployment(name, namespace, "", defaultReplicas, &kanaryv1alpha1test.NewKanaryDeploymentOptions{Validation: validatedManualSpec}),
				dep:       utilstest.NewDeployment(name, namespace, defaultReplicas, nil),
				canaryDep: utilstest.NewDeployment(name+"-kanary", namespace, 1, nil),
				kclient: fake.NewFakeClient([]runtime.Object{
					utilstest.NewDeployment(name, namespace, defaultReplicas, nil),
					kanaryv1alpha1test.NewKanaryDeployment(name, namespace, "", defaultReplicas, &kanaryv1alpha1test.NewKanaryDeploymentOptions{Validation: validatedManualSpec}),
				}...),
			},
			wantStatusSucceed: true,
			wantResult:        reconcile.Result{},
			wantErr:           false,
		},
		{
			name: "validation manual invalidated",
			fields: fields{
				deadline: kanaryv1alpha1.NoneKanaryDeploymentSpecValidationManualDeadineStatus,
				status:   kanaryv1alpha1.InvalidKanaryDeploymentSpecValidationManualStatus,
			},
			args: args{
				kd:        kanaryv1alpha1test.NewKanaryDeployment(name, namespace, "", defaultReplicas, &kanaryv1alpha1test.NewKanaryDeploymentOptions{Validation: validatedManualSpec}),
				dep:       utilstest.NewDeployment(name, namespace, defaultReplicas, nil),
				canaryDep: utilstest.NewDeployment(name+"-kanary", namespace, 1, nil),
				kclient: fake.NewFakeClient([]runtime.Object{
					utilstest.NewDeployment(name, namespace, defaultReplicas, nil),
					kanaryv1alpha1test.NewKanaryDeployment(name, namespace, "", defaultReplicas, &kanaryv1alpha1test.NewKanaryDeploymentOptions{Validation: validatedManualSpec}),
				}...),
			},
			wantStatusSucceed: false,
			wantStatusInvalid: true,
			wantResult:        reconcile.Result{},
			wantErr:           false,
		},
		{
			name: "validation manual with deadline validated",
			fields: fields{
				deadline:         kanaryv1alpha1.ValidKanaryDeploymentSpecValidationManualDeadineStatus,
				status:           "",
				validationPeriod: 15 * time.Minute,
			},
			args: args{
				kd:        kanaryv1alpha1test.NewKanaryDeployment(name, namespace, "", defaultReplicas, &kanaryv1alpha1test.NewKanaryDeploymentOptions{StartTime: &creationTime, Validation: validatedManualSpec}),
				dep:       utilstest.NewDeployment(name, namespace, defaultReplicas, &utilstest.NewDeploymentOptions{CreationTime: &creationTime}),
				canaryDep: utilstest.NewDeployment(name+"-kanary", namespace, 1, &utilstest.NewDeploymentOptions{CreationTime: &creationTime}),
				kclient: fake.NewFakeClient([]runtime.Object{
					utilstest.NewDeployment(name, namespace, defaultReplicas, &utilstest.NewDeploymentOptions{CreationTime: &creationTime}),
					kanaryv1alpha1test.NewKanaryDeployment(name, namespace, "", defaultReplicas, &kanaryv1alpha1test.NewKanaryDeploymentOptions{Validation: validatedManualSpec}),
				}...),
			},
			wantStatusSucceed: true,
			wantResult:        reconcile.Result{},
			wantErr:           false,
		},
		{
			name: "validation manual with deadline invalidated",
			fields: fields{
				deadline:         kanaryv1alpha1.InvalidKanaryDeploymentSpecValidationManualDeadineStatus,
				status:           "",
				validationPeriod: 15 * time.Minute,
			},
			args: args{
				kd:        kanaryv1alpha1test.NewKanaryDeployment(name, namespace, "", defaultReplicas, &kanaryv1alpha1test.NewKanaryDeploymentOptions{StartTime: &creationTime, Validation: validatedManualSpec}),
				dep:       utilstest.NewDeployment(name, namespace, defaultReplicas, &utilstest.NewDeploymentOptions{CreationTime: &creationTime}),
				canaryDep: utilstest.NewDeployment(name+"-kanary", namespace, 1, &utilstest.NewDeploymentOptions{CreationTime: &creationTime}),
				kclient: fake.NewFakeClient([]runtime.Object{
					utilstest.NewDeployment(name, namespace, defaultReplicas, &utilstest.NewDeploymentOptions{CreationTime: &creationTime}),
					kanaryv1alpha1test.NewKanaryDeployment(name, namespace, "", defaultReplicas, &kanaryv1alpha1test.NewKanaryDeploymentOptions{Validation: validatedManualSpec}),
				}...),
			},
			wantStatusInvalid: true,
			wantResult:        reconcile.Result{},
			wantErr:           false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqLogger := log.WithValues("test:", tt.name)
			m := &manualImpl{
				deadlineStatus:         tt.fields.deadline,
				validationManualStatus: tt.fields.status,
				validationPeriod:       tt.fields.validationPeriod,
			}

			gotStatus, gotResult, err := m.Validation(tt.args.kclient, reqLogger, tt.args.kd, tt.args.dep, tt.args.canaryDep)
			if (err != nil) != tt.wantErr {
				t.Errorf("manualImpl.Validation() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
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

			if !reflect.DeepEqual(gotResult, tt.wantResult) {
				t.Errorf("manualImpl.Validation() gotResult = %v, want %v", gotResult, tt.wantResult)
			}
		})
	}
}
