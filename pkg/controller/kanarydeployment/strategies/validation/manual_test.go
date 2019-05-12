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

		defaultValidationSpec = &kanaryv1alpha1.KanaryDeploymentSpecValidationList{
			Items: []kanaryv1alpha1.KanaryDeploymentSpecValidation{
				{
					Manual: &kanaryv1alpha1.KanaryDeploymentSpecValidationManual{},
				},
			},
		}
		validatedManualSpec = &kanaryv1alpha1.KanaryDeploymentSpecValidationList{
			Items: []kanaryv1alpha1.KanaryDeploymentSpecValidation{
				{
					Manual: &kanaryv1alpha1.KanaryDeploymentSpecValidationManual{
						Status: kanaryv1alpha1.ValidKanaryDeploymentSpecValidationManualStatus,
					},
				},
			},
			MaxIntervalPeriod: &metav1.Duration{Duration: 15 * time.Second},
			ValidationPeriod:  &metav1.Duration{Duration: 30 * time.Second},
		}
	)

	type fields struct {
		deadlineStatus         kanaryv1alpha1.KanaryDeploymentSpecValidationManualDeadineStatus
		validationManualStatus kanaryv1alpha1.KanaryDeploymentSpecValidationManualStatus
		dryRun                 bool
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
			args: args{
				kclient:   fake.NewFakeClient([]runtime.Object{}...),
				kd:        kanaryv1alpha1test.NewKanaryDeployment(name, namespace, "", defaultReplicas, &kanaryv1alpha1test.NewKanaryDeploymentOptions{Validations: defaultValidationSpec}),
				dep:       utilstest.NewDeployment(name, namespace, defaultReplicas, nil),
				canaryDep: utilstest.NewDeployment(name+"-kanary", namespace, 1, nil),
			},
			want: &Result{
				IsFailed: false,
			},
			wantErr: false,
		},
		{
			name: "validation manual validated",
			fields: fields{
				deadlineStatus:         kanaryv1alpha1.NoneKanaryDeploymentSpecValidationManualDeadineStatus,
				validationManualStatus: kanaryv1alpha1.ValidKanaryDeploymentSpecValidationManualStatus,
			},
			args: args{
				kd:        kanaryv1alpha1test.NewKanaryDeployment(name, namespace, "", defaultReplicas, &kanaryv1alpha1test.NewKanaryDeploymentOptions{Validations: validatedManualSpec}),
				dep:       utilstest.NewDeployment(name, namespace, defaultReplicas, nil),
				canaryDep: utilstest.NewDeployment(name+"-kanary", namespace, 1, nil),
				kclient: fake.NewFakeClient([]runtime.Object{
					utilstest.NewDeployment(name, namespace, defaultReplicas, nil),
					kanaryv1alpha1test.NewKanaryDeployment(name, namespace, "", defaultReplicas, &kanaryv1alpha1test.NewKanaryDeploymentOptions{Validations: validatedManualSpec}),
				}...),
			},
			want: &Result{
				IsFailed:        false,
				ForceSuccessNow: true,
			},
			wantErr: false,
		},
		{
			name: "validation manual invalidated",
			fields: fields{
				deadlineStatus:         kanaryv1alpha1.NoneKanaryDeploymentSpecValidationManualDeadineStatus,
				validationManualStatus: kanaryv1alpha1.InvalidKanaryDeploymentSpecValidationManualStatus,
			},
			args: args{
				kd:        kanaryv1alpha1test.NewKanaryDeployment(name, namespace, "", defaultReplicas, &kanaryv1alpha1test.NewKanaryDeploymentOptions{Validations: validatedManualSpec}),
				dep:       utilstest.NewDeployment(name, namespace, defaultReplicas, nil),
				canaryDep: utilstest.NewDeployment(name+"-kanary", namespace, 1, nil),
				kclient: fake.NewFakeClient([]runtime.Object{
					utilstest.NewDeployment(name, namespace, defaultReplicas, nil),
					kanaryv1alpha1test.NewKanaryDeployment(name, namespace, "", defaultReplicas, &kanaryv1alpha1test.NewKanaryDeploymentOptions{Validations: validatedManualSpec}),
				}...),
			},
			want: &Result{
				IsFailed: true,
				Comment:  "manual.status=invalid",
			},
			wantErr: false,
		},
		{
			name: "validation manual with deadline validated",
			fields: fields{
				deadlineStatus:         kanaryv1alpha1.ValidKanaryDeploymentSpecValidationManualDeadineStatus,
				validationManualStatus: "",
			},
			args: args{
				kd:        kanaryv1alpha1test.NewKanaryDeployment(name, namespace, "", defaultReplicas, &kanaryv1alpha1test.NewKanaryDeploymentOptions{StartTime: &creationTime, Validations: validatedManualSpec}),
				dep:       utilstest.NewDeployment(name, namespace, defaultReplicas, &utilstest.NewDeploymentOptions{CreationTime: &creationTime}),
				canaryDep: utilstest.NewDeployment(name+"-kanary", namespace, 1, &utilstest.NewDeploymentOptions{CreationTime: &creationTime}),
				kclient: fake.NewFakeClient([]runtime.Object{
					utilstest.NewDeployment(name, namespace, defaultReplicas, &utilstest.NewDeploymentOptions{CreationTime: &creationTime}),
					kanaryv1alpha1test.NewKanaryDeployment(name, namespace, "", defaultReplicas, &kanaryv1alpha1test.NewKanaryDeploymentOptions{Validations: validatedManualSpec}),
				}...),
			},
			want: &Result{
				IsFailed: false,
				Comment:  "deadline activated with 'valid' status",
			},
			wantErr: false,
		},
		{
			name: "validation manual with deadline invalidated",
			fields: fields{
				deadlineStatus:         kanaryv1alpha1.InvalidKanaryDeploymentSpecValidationManualDeadineStatus,
				validationManualStatus: "",
			},
			args: args{
				kd:        kanaryv1alpha1test.NewKanaryDeployment(name, namespace, "", defaultReplicas, &kanaryv1alpha1test.NewKanaryDeploymentOptions{StartTime: &creationTime, Validations: validatedManualSpec}),
				dep:       utilstest.NewDeployment(name, namespace, defaultReplicas, &utilstest.NewDeploymentOptions{CreationTime: &creationTime}),
				canaryDep: utilstest.NewDeployment(name+"-kanary", namespace, 1, &utilstest.NewDeploymentOptions{CreationTime: &creationTime}),
				kclient: fake.NewFakeClient([]runtime.Object{
					utilstest.NewDeployment(name, namespace, defaultReplicas, &utilstest.NewDeploymentOptions{CreationTime: &creationTime}),
					kanaryv1alpha1test.NewKanaryDeployment(name, namespace, "", defaultReplicas, &kanaryv1alpha1test.NewKanaryDeploymentOptions{Validations: validatedManualSpec}),
				}...),
			},
			want: &Result{
				IsFailed: true,
				Comment:  "deadline activated with 'invalid' status",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqLogger := log.WithValues("test:", tt.name)
			m := &manualImpl{
				deadlineStatus:         tt.fields.deadlineStatus,
				validationManualStatus: tt.fields.validationManualStatus,
				dryRun:                 tt.fields.dryRun,
			}
			got, err := m.Validation(tt.args.kclient, reqLogger, tt.args.kd, tt.args.dep, tt.args.canaryDep)
			if (err != nil) != tt.wantErr {
				t.Errorf("manualImpl.Validation() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("manualImpl.Validation() = %#v, want %#v", got, tt.want)
			}
		})
	}
}
