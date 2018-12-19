package kanarydeployment

import (
	"context"
	"fmt"
	"time"

	kanaryv1alpha1 "github.com/amadeusitgroup/kanary/pkg/apis/kanary/v1alpha1"

	appsv1beta1 "k8s.io/api/apps/v1beta1"
	corev1 "k8s.io/api/core/v1"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_kanarydeployment")

const (
	KanaryDeploymentIsKanaryLabelKey   = "kanary.k8s.io/iskanary"
	KanaryDeploymentKanaryNameLabelKey = "kanary.k8s.io/name"
	KanaryDeploymentActivateLabelKey   = "kanary.k8s.io/canary-pod"
	KanaryDeploymentLabelValueTrue     = "true"
	KanaryDeploymentLabelValueFalse    = "false"
)

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new KanaryDeployment Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileKanaryDeployment{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("kanarydeployment-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource KanaryDeployment
	err = c.Watch(&source.Kind{Type: &kanaryv1alpha1.KanaryDeployment{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch for changes to secondary resource Deployment and requeue the owner KanaryDeployment
	err = c.Watch(&source.Kind{Type: &appsv1beta1.Deployment{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &kanaryv1alpha1.KanaryDeployment{},
	})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileKanaryDeployment{}

// ReconcileKanaryDeployment reconciles a KanaryDeployment object
type ReconcileKanaryDeployment struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a KanaryDeployment object and makes changes based on the state read
// and what is in the KanaryDeployment.Spec
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileKanaryDeployment) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling KanaryDeployment")

	// Fetch the KanaryDeployment instance
	instance := &kanaryv1alpha1.KanaryDeployment{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// Check if the deployment already exists, if not create a new one
	deployment := &appsv1beta1.Deployment{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: getDeploymentName(instance), Namespace: instance.Namespace}, deployment)
	if err != nil && errors.IsNotFound(err) {
		deployment = r.deploymentForKanaryDeployment(instance)
		reqLogger.Info("Creating a new Deployment", "Deployment.Namespace", deployment.Namespace, "Deployment.Name", deployment.Name)
		err = r.client.Create(context.TODO(), deployment)
		if err != nil {
			reqLogger.Error(err, "failed to create new Deployment", "Deployment.Namespace", deployment.Namespace, "Deployment.Name", deployment.Name)
			return reconcile.Result{}, err
		}
		// Deployment created successfully - return and requeue
		return reconcile.Result{Requeue: true}, nil
	} else if err != nil {
		reqLogger.Error(err, "failed to get Deployment")
		return reconcile.Result{}, err
	}

	canarydeployment := &appsv1beta1.Deployment{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: getCanaryDeploymentName(instance), Namespace: instance.Namespace}, canarydeployment)
	if err != nil && errors.IsNotFound(err) {
		canarydeployment = r.canaryDeploymentForKanaryDeployment(instance)
		// set by default canary deployment to replicas=0
		canarydeployment.Spec.Replicas = newInt32(0)

		reqLogger.Info("Creating a new Canary Deployment", "Deployment.Namespace", canarydeployment.Namespace, "Deployment.Name", canarydeployment.Name)
		err = r.client.Create(context.TODO(), canarydeployment)
		if err != nil {
			reqLogger.Error(err, "failed to create new Deployment", "Deployment.Namespace", canarydeployment.Namespace, "Deployment.Name", canarydeployment.Name)
			return reconcile.Result{}, err
		}
		// Deployment created successfully - return and requeue
		return reconcile.Result{Requeue: true}, nil
	}

	// Retrieve service if defined
	var service *corev1.Service
	if instance.Spec.ServiceName != "" {
		service = &corev1.Service{}
		err = r.client.Get(context.TODO(), types.NamespacedName{Name: instance.Spec.ServiceName, Namespace: instance.Namespace}, service)
		if err != nil && errors.IsNotFound(err) {
			// TODO update status, to say that the service didn't exist

			return reconcile.Result{Requeue: true, RequeueAfter: time.Duration(time.Second)}, err
		} else if err != nil {
			reqLogger.Error(err, "failed to get Deployment")
			return reconcile.Result{}, err
		}
	}

	if service != nil {
		kanaryService := r.canaryServiceForKanaryDeployment(instance, service)
		err = r.client.Get(context.TODO(), types.NamespacedName{Name: kanaryService.Name, Namespace: kanaryService.Namespace}, kanaryService)
		if err != nil && errors.IsNotFound(err) {
			reqLogger.Info("Creating a new Canary Service", "Service.Namespace", kanaryService.Namespace, "Service.Name", kanaryService.Name)
			err = r.client.Create(context.TODO(), kanaryService)
			if err != nil {
				reqLogger.Error(err, "failed to create new CanaryService", "Service.Namespace", kanaryService.Namespace, "Service.Name", kanaryService.Name)
				return reconcile.Result{}, err
			}
			// Deployment created successfully - return and requeue
			return reconcile.Result{Requeue: true}, nil
		} else if err != nil {
			reqLogger.Error(err, "failed to get Service")
			return reconcile.Result{}, err
		}
	}

	// TODO implement this function

	return reconcile.Result{}, nil
}

// deploymentForKanaryDeployment returns a Deployment object
func (r *ReconcileKanaryDeployment) deploymentForKanaryDeployment(kd *kanaryv1alpha1.KanaryDeployment) *appsv1beta1.Deployment {
	ls := labelsForKanaryDeploymentd(kd.Name)

	dep := &appsv1beta1.Deployment{
		TypeMeta:   kd.Spec.Template.TypeMeta,
		ObjectMeta: kd.Spec.Template.ObjectMeta,
		Spec:       kd.Spec.Template.Spec,
	}

	if dep.Labels == nil {
		dep.Labels = map[string]string{}
	}
	for key, val := range ls {
		dep.Labels[key] = val
	}

	dep.Name = getDeploymentName(kd)
	if dep.Namespace == "" {
		dep.Namespace = kd.Namespace
	}

	// Set KanaryDeployment instance as the owner and controller
	controllerutil.SetControllerReference(kd, dep, r.scheme)
	return dep
}

// canaryDeploymentForKanaryDeployment returns a Deployment object
func (r *ReconcileKanaryDeployment) canaryDeploymentForKanaryDeployment(kd *kanaryv1alpha1.KanaryDeployment) *appsv1beta1.Deployment {
	dep := r.deploymentForKanaryDeployment(kd)
	dep.Name = getCanaryDeploymentName(kd)
	if dep.Spec.Template.Labels == nil {
		dep.Spec.Template.Labels = map[string]string{}
	}
	dep.Spec.Template.Labels[KanaryDeploymentActivateLabelKey] = KanaryDeploymentLabelValueTrue

	return dep
}

// canaryServiceForKanaryDeployment returns a Service object
func (r *ReconcileKanaryDeployment) canaryServiceForKanaryDeployment(kd *kanaryv1alpha1.KanaryDeployment, service *corev1.Service) *corev1.Service {
	kanaryServiceName := kd.Spec.Strategy.ServiceName
	if kanaryServiceName == "" {
		kanaryServiceName = fmt.Sprintf("%s-kanary", service.Name)
	}

	labelSelector := map[string]string{}
	for key, val := range service.Spec.Selector {
		labelSelector[key] = val
	}
	labelSelector[KanaryDeploymentActivateLabelKey] = KanaryDeploymentLabelValueTrue

	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: corev1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      kanaryServiceName,
			Namespace: kd.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: labelSelector,
		},
	}
}

func getDeploymentName(kd *kanaryv1alpha1.KanaryDeployment) string {
	name := kd.Spec.Template.ObjectMeta.Name
	if name == "" {
		name = kd.Name
	}
	return name
}

func getCanaryDeploymentName(kd *kanaryv1alpha1.KanaryDeployment) string {
	return fmt.Sprintf("%s-kanary", getDeploymentName(kd))
}

// belonging to the given KanaryDeployment CR name.
func labelsForKanaryDeploymentd(name string) map[string]string {
	return map[string]string{
		KanaryDeploymentIsKanaryLabelKey:   KanaryDeploymentLabelValueTrue,
		KanaryDeploymentKanaryNameLabelKey: name,
	}
}

func newInt32(i int32) *int32 {
	return &i
}
