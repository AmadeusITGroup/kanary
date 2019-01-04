package kanarydeployment

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	appsv1beta1 "k8s.io/api/apps/v1beta1"
	corev1 "k8s.io/api/core/v1"

	"k8s.io/client-go/kubernetes/scheme"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"

	kanaryv1alpha1 "github.com/amadeusitgroup/kanary/pkg/apis/kanary/v1alpha1"
	"github.com/amadeusitgroup/kanary/pkg/controller/kanarydeployment/utils"
)

func TestReconcileKanaryDeployment_Reconcile(t *testing.T) {
	logf.SetLogger(logf.ZapLogger(true))

	var (
		name            = "foo"
		serviceName     = "foo"
		namespace       = "kanary"
		defaultReplicas = int32(5)

		kanaryServiceTraffic = &kanaryv1alpha1.KanaryDeploymentSpecTraffic{
			Source: kanaryv1alpha1.KanaryServiceKanaryDeploymentSpecTrafficSource,
		}
	)

	// Register operator types with the runtime scheme.
	s := scheme.Scheme
	s.AddKnownTypes(kanaryv1alpha1.SchemeGroupVersion, &kanaryv1alpha1.KanaryDeployment{})

	type fields struct {
		client client.Client
		scheme *runtime.Scheme
	}
	type args struct {
		request reconcile.Request
	}
	tests := []struct {
		name     string
		fields   fields
		request  reconcile.Request
		want     reconcile.Result
		wantErr  bool
		wantFunc func(*ReconcileKanaryDeployment) error
	}{
		{
			name: "[INIT] KanaryDeployment dont exist",

			request: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      name,
					Namespace: namespace,
				},
			},
			fields: fields{
				scheme: s,
				client: fake.NewFakeClient([]runtime.Object{}...),
			},
			want: reconcile.Result{
				Requeue: false,
			},
		},

		{
			name: "[INIT] KanaryDeployment Not defaulted",

			request: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      name,
					Namespace: namespace,
				},
			},
			fields: fields{
				scheme: s,
				client: fake.NewFakeClient([]runtime.Object{
					newKanaryDeployment(name, namespace, "", defaultReplicas, nil, nil, nil),
				}...),
			},
			want: reconcile.Result{
				Requeue: true,
			},
			wantFunc: func(r *ReconcileKanaryDeployment) error {
				kd := &kanaryv1alpha1.KanaryDeployment{}
				err := r.client.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: namespace}, kd)
				if err != nil && errors.IsNotFound(err) {
					return fmt.Errorf("unable to get the created deployment, %v", err)
				}

				if kd.Spec.Scale.Static == nil || kd.Spec.Scale.Static.Replicas == nil {
					return fmt.Errorf("kd.Spec.Scale.Static.Replicas should be defaulted")
				}

				return err
			},
		},

		{
			name: "[INIT] canary Deployment creation",

			request: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      name,
					Namespace: namespace,
				},
			},
			fields: fields{
				scheme: s,
				client: fake.NewFakeClient([]runtime.Object{
					newKanaryDeployment(name, namespace, "", defaultReplicas, nil, nil, nil),
					newDeployment(name, namespace, defaultReplicas),
				}...),
			},
			want: reconcile.Result{
				Requeue: true,
			},
			wantFunc: func(r *ReconcileKanaryDeployment) error {
				deployment := &appsv1beta1.Deployment{}
				err := r.client.Get(context.TODO(), types.NamespacedName{Name: name + "-kanary", Namespace: namespace}, deployment)
				if err != nil && errors.IsNotFound(err) {
					return fmt.Errorf("unable to get the created canary deployment, %v", err)
				}
				if err != nil {
					return err
				}
				// check if replicas is equal to 0
				if deployment.Spec.Replicas == nil {
					return fmt.Errorf("replicas should not be nil")
				} else if *deployment.Spec.Replicas != int32(1) {
					return fmt.Errorf("replicas should be equal to 1, current value %d", *deployment.Spec.Replicas)
				}

				return nil
			},
		},

		{
			name: "[INIT] service is not defined",

			request: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      name,
					Namespace: namespace,
				},
			},
			fields: fields{
				scheme: s,
				client: fake.NewFakeClient([]runtime.Object{
					newKanaryDeployment(name, namespace, "", defaultReplicas, nil, nil, nil),
					newDeployment(name, namespace, defaultReplicas),
					newDeployment(name+"-kanary", namespace, 1),
				}...),
			},
			want: reconcile.Result{
				Requeue: false,
			},
			wantFunc: func(r *ReconcileKanaryDeployment) error {
				service := &corev1.Service{}
				err := r.client.Get(context.TODO(), types.NamespacedName{Name: name + "-kanary", Namespace: namespace}, service)
				if err != nil && errors.IsNotFound(err) {
					return nil
				}
				return err
			},
		},

		{
			name: "[INIT] service is define but dont exist",

			request: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      name,
					Namespace: namespace,
				},
			},
			fields: fields{
				scheme: s,
				client: fake.NewFakeClient([]runtime.Object{
					newKanaryDeployment(name, namespace, serviceName, defaultReplicas, nil, kanaryServiceTraffic, nil),
					newDeployment(name, namespace, defaultReplicas),
					newDeployment(name+"-kanary", namespace, 1),
				}...),
			},
			want: reconcile.Result{
				Requeue:      true,
				RequeueAfter: time.Duration(1 * time.Second),
			},
			wantErr: true,
			wantFunc: func(r *ReconcileKanaryDeployment) error {
				service := &corev1.Service{}
				err := r.client.Get(context.TODO(), types.NamespacedName{Name: name + "-kanary", Namespace: namespace}, service)
				if err != nil && errors.IsNotFound(err) {
					return nil
				}
				return err
			},
		},

		{
			name: "[INIT] service is define, test kanary service creation",

			request: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      name,
					Namespace: namespace,
				},
			},
			fields: fields{
				scheme: s,
				client: fake.NewFakeClient([]runtime.Object{
					newKanaryDeployment(name, namespace, serviceName, defaultReplicas, nil, kanaryServiceTraffic, nil),
					newDeployment(name, namespace, defaultReplicas),
					newDeployment(name+"-kanary", namespace, 1),
					newService(serviceName, namespace, nil),
				}...),
			},
			want: reconcile.Result{
				Requeue: true,
			},
			wantFunc: func(r *ReconcileKanaryDeployment) error {
				service := &corev1.Service{}
				err := r.client.Get(context.TODO(), types.NamespacedName{Name: name + "-kanary", Namespace: namespace}, service)
				if err != nil && errors.IsNotFound(err) {
					return fmt.Errorf("unable to get the created canary service, %v", err)
				}
				labelFound := false
				for key, val := range service.Spec.Selector {
					if key == kanaryv1alpha1.KanaryDeploymentActivateLabelKey && val == kanaryv1alpha1.KanaryDeploymentLabelValueTrue {
						labelFound = true
						break
					}
				}

				if !labelFound {
					return fmt.Errorf("unable to found the label key: %s in the service.Spec.Selector map", kanaryv1alpha1.KanaryDeploymentActivateLabelKey)
				}
				return err
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &ReconcileKanaryDeployment{
				client: tt.fields.client,
				scheme: tt.fields.scheme,
			}
			got, err := r.Reconcile(tt.request)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReconcileKanaryDeployment.Reconcile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ReconcileKanaryDeployment.Reconcile() = %v, want %v", got, tt.want)
			}
			if tt.wantFunc != nil {
				if err := tt.wantFunc(r); err != nil {
					t.Errorf("ReconcileKanaryDeployment.Reconcile() not properly validated, %v", err)
				}
			}
		})
	}
}

func newKanaryDeployment(name, namespace, serviceName string, replicas int32, scale *kanaryv1alpha1.KanaryDeploymentSpecScale, traffic *kanaryv1alpha1.KanaryDeploymentSpecTraffic, validation *kanaryv1alpha1.KanaryDeploymentSpecValidation) *kanaryv1alpha1.KanaryDeployment {
	kd := &kanaryv1alpha1.KanaryDeployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "KanaryDeployment",
			APIVersion: kanaryv1alpha1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	kd.Spec.Template.Spec.Selector = &metav1.LabelSelector{}
	kd.Spec.Template.Spec.Replicas = kanaryv1alpha1.NewInt32(replicas)
	kd.Spec.ServiceName = serviceName

	if scale != nil {
		kd.Spec.Scale = *scale
	}
	if traffic != nil {
		kd.Spec.Traffic = *traffic
	}
	if validation != nil {
		kd.Spec.Validation = *validation
	}
	kd = kanaryv1alpha1.DefaultKanaryDeployment(kd)
	kd.Spec.ServiceName = serviceName

	return kd
}

func newDeployment(name, namespace string, replicas int32) *appsv1beta1.Deployment {
	spec := &appsv1beta1.DeploymentSpec{
		Replicas: &replicas,
	}
	md5, _ := utils.GenerateMD5DeploymentSpec(spec)
	return &appsv1beta1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: appsv1beta1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Annotations: map[string]string{string(kanaryv1alpha1.MD5KanaryDeploymentAnnotationKey): md5},
		},
		Spec: *spec,
	}
}

func newService(name, namespace string, labelsSelector map[string]string) *corev1.Service {
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: corev1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: labelsSelector,
		},
	}
}
