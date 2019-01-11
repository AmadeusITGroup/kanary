package kanarydeployment

import (
	"context"
	"time"

	"github.com/go-logr/logr"

	appsv1beta1 "k8s.io/api/apps/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"

	kanaryv1alpha1 "github.com/amadeusitgroup/kanary/pkg/apis/kanary/v1alpha1"
	"github.com/amadeusitgroup/kanary/pkg/controller/kanarydeployment/strategies"
	"github.com/amadeusitgroup/kanary/pkg/controller/kanarydeployment/strategies/traffic"
	"github.com/amadeusitgroup/kanary/pkg/controller/kanarydeployment/utils"
)

var log = logf.Log.WithName("controller_kanarydeployment")

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
	return err
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
	reqLogger := log.WithValues("Namespace", request.Namespace, "KanaryDeployment", request.Name)
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

	if !kanaryv1alpha1.IsDefaultedKanaryDeployment(instance) {
		reqLogger.Info("Defaulting values")
		defaultedInstance := kanaryv1alpha1.DefaultKanaryDeployment(instance)
		err = r.client.Update(context.TODO(), defaultedInstance)
		if err != nil {
			reqLogger.Error(err, "failed to update KanaryDeployment")
			return reconcile.Result{}, err
		}
		// KanaryDeployment is now defaulted return and requeue
		return reconcile.Result{Requeue: true}, nil
	}

	// Check if the deployment already exists, if not create a new one
	deployment, needsReturn, result, err := r.manageDeploymentCreationFunc(reqLogger, instance, utils.GetDeploymentName(instance), utils.NewDeploymentFromKanaryDeploymentTemplate)
	if needsReturn {
		return updateKanaryDeploymentStatus(r.client, reqLogger, instance, metav1.Now(), result, err)
	}

	var canarydeployment *appsv1beta1.Deployment
	canarydeployment, needsReturn, result, err = r.manageCanaryDeploymentCreation(reqLogger, instance, utils.GetCanaryDeploymentName(instance))
	if needsReturn {
		return updateKanaryDeploymentStatus(r.client, reqLogger, instance, metav1.Now(), result, err)
	}

	strategy, err := strategies.NewStrategy(&instance.Spec)
	if err != nil {
		reqLogger.Error(err, "failed to instance the KanaryDeployment strategies")
		return reconcile.Result{}, err
	}
	if strategy == nil {
		return updateKanaryDeploymentStatus(r.client, reqLogger, instance, metav1.Now(), result, err)
	}

	return strategy.Apply(r.client, reqLogger, instance, deployment, canarydeployment)
}

func (r *ReconcileKanaryDeployment) manageCanaryDeploymentCreation(reqLogger logr.Logger, kd *kanaryv1alpha1.KanaryDeployment, name string) (*appsv1beta1.Deployment, bool, reconcile.Result, error) {
	// check that the deployment template was not updated since the creation
	currentHash, err := utils.GenerateMD5DeploymentSpec(&kd.Spec.Template.Spec)
	if err != nil {
		reqLogger.Error(err, "failed to generate Deployment template MD5")
		return nil, true, reconcile.Result{}, err
	}

	deployment := &appsv1beta1.Deployment{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: kd.Namespace}, deployment)
	if err != nil && errors.IsNotFound(err) {
		deployment, err = utils.NewCanaryDeploymentFromKanaryDeploymentTemplate(kd, r.scheme, false, traffic.NeedOverwriteSelector(kd))
		if err != nil {
			reqLogger.Error(err, "failed to create the Deployment artifact")
			return deployment, true, reconcile.Result{}, err
		}

		reqLogger.Info("Creating a new Deployment")
		err = r.client.Create(context.TODO(), deployment)
		if err != nil {
			reqLogger.Error(err, "failed to create new Deployment")
			return deployment, true, reconcile.Result{}, err
		}
		kd.Status.CurrentHash = currentHash
		// Deployment created successfully - return and requeue
		return deployment, true, reconcile.Result{Requeue: true}, nil
	} else if err != nil {
		reqLogger.Error(err, "failed to get Deployment")
		return deployment, true, reconcile.Result{}, err
	}

	if kd.Status.CurrentHash != "" && kd.Status.CurrentHash != currentHash {
		err = r.client.Delete(context.TODO(), deployment)
		if err != nil {
			reqLogger.Error(err, "failed to delete deprecated Deployment")
			return deployment, true, reconcile.Result{RequeueAfter: time.Second}, err
		}
	}

	return deployment, false, reconcile.Result{}, err
}

func (r *ReconcileKanaryDeployment) manageDeploymentCreationFunc(reqLogger logr.Logger, kd *kanaryv1alpha1.KanaryDeployment, name string, createFunc func(*kanaryv1alpha1.KanaryDeployment, *runtime.Scheme, bool) (*appsv1beta1.Deployment, error)) (*appsv1beta1.Deployment, bool, reconcile.Result, error) {
	deployment := &appsv1beta1.Deployment{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: kd.Namespace}, deployment)
	if err != nil && errors.IsNotFound(err) {
		deployment, err = createFunc(kd, r.scheme, false)
		if err != nil {
			reqLogger.Error(err, "failed to create the Deployment artifact")
			return deployment, true, reconcile.Result{}, err
		}

		reqLogger.Info("Creating a new Deployment")
		err = r.client.Create(context.TODO(), deployment)
		if err != nil {
			reqLogger.Error(err, "failed to create new Deployment")
			return deployment, true, reconcile.Result{}, err
		}
		// Deployment created successfully - return and requeue
		return deployment, true, reconcile.Result{Requeue: true}, nil
	} else if err != nil {
		reqLogger.Error(err, "failed to get Deployment")
		return deployment, true, reconcile.Result{}, err
	}

	return deployment, false, reconcile.Result{}, err
}

func updateKanaryDeploymentStatus(kclient client.StatusWriter, reqLogger logr.Logger, kd *kanaryv1alpha1.KanaryDeployment, now metav1.Time, result reconcile.Result, err error) (reconcile.Result, error) {
	newStatus := kd.Status.DeepCopy()
	utils.UpdateKanaryDeploymentStatusConditionsFailure(newStatus, now, err)
	return utils.UpdateKanaryDeploymentStatus(kclient, reqLogger, kd, newStatus, result, err)
}
