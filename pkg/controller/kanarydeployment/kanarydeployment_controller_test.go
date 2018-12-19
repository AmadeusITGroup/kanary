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
)

func TestReconcileKanaryDeployment_Reconcile(t *testing.T) {
	logf.SetLogger(logf.ZapLogger(true))

	var (
		name            = "foo"
		serviceName     = "foo"
		namespace       = "kanary"
		defaultReplicas = int32(5)
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
			name: "[INIT] Deployment creation",

			request: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      name,
					Namespace: namespace,
				},
			},
			fields: fields{
				scheme: s,
				client: fake.NewFakeClient([]runtime.Object{
					newKanaryDeployment(name, namespace, "", defaultReplicas),
				}...),
			},
			want: reconcile.Result{
				Requeue: true,
			},
			wantFunc: func(r *ReconcileKanaryDeployment) error {
				deployment := &appsv1beta1.Deployment{}
				err := r.client.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: namespace}, deployment)
				if err != nil && errors.IsNotFound(err) {
					return fmt.Errorf("unable to get the created deployment, %v", err)
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
					newKanaryDeployment(name, namespace, "", defaultReplicas),
					newDeployment(name, namespace),
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
				} else if *deployment.Spec.Replicas != int32(0) {
					return fmt.Errorf("replicas should be equal to 0, current value %d", *deployment.Spec.Replicas)
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
					newKanaryDeployment(name, namespace, "", defaultReplicas),
					newDeployment(name, namespace),
					newDeployment(name+"-kanary", namespace),
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
					newKanaryDeployment(name, namespace, serviceName, defaultReplicas),
					newDeployment(name, namespace),
					newDeployment(name+"-kanary", namespace),
				}...),
			},
			want: reconcile.Result{
				Requeue:      true,
				RequeueAfter: time.Duration(time.Second),
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
					newKanaryDeployment(name, namespace, serviceName, defaultReplicas),
					newDeployment(name, namespace),
					newDeployment(name+"-kanary", namespace),
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
					if key == KanaryDeploymentActivateLabelKey && val == KanaryDeploymentLabelValueTrue {
						labelFound = true
						break
					}
				}

				if !labelFound {
					return fmt.Errorf("unable to found the label key: %s in the service.Spec.Selector map", KanaryDeploymentActivateLabelKey)
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

func newKanaryDeployment(name, namespace, serviceName string, replicas int32) *kanaryv1alpha1.KanaryDeployment {
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

	kd.Spec.Template.Spec.Replicas = newInt32(replicas)
	kd.Spec.ServiceName = serviceName

	return kd
}

func newDeployment(name, namespace string) *appsv1beta1.Deployment {
	return &appsv1beta1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: appsv1beta1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
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
