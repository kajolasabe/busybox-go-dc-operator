package busybox

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	busyboxv1alpha1 "persistent.com/busybox/busybox-go-dc-operator/pkg/apis/busybox/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
	
	"reflect"
)

var log = logf.Log.WithName("controller_busybox")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new Busybox Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileBusybox{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("busybox-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Busybox
	err = c.Watch(&source.Kind{Type: &busyboxv1alpha1.Busybox{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner Busybox
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &busyboxv1alpha1.Busybox{},
	})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileBusybox implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileBusybox{}

// ReconcileBusybox reconciles a Busybox object
type ReconcileBusybox struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a Busybox object and makes changes based on the state read
// and what is in the Busybox.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileBusybox) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling Busybox")

	// Fetch the Busybox instance
	instance := &busyboxv1alpha1.Busybox{}
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

	// Check if the DeploymentConfig already exists, if not create a new one
	deployConfig := &appsv1.DeploymentConfig{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: instance.Name, Namespace: instance.Namespace}, deployConfig)
	if err != nil && errors.IsNotFound(err) {
		// Define a new DeploymentConfig
		dc := r.deployConfigForBusybox(instance)
		reqLogger.Info("Creating a new Deployment Config.", "DeploymentConfig.Namespace", dc.Namespace, "DeploymentConfig.Name", dc.Name)
		err = r.client.Create(context.TODO(), dc)
		if err != nil {
			reqLogger.Error(err, "Failed to create new Deployment Config.", "DeploymentConfig.Namespace", dc.Namespace, "DeploymentConfig.Name", dc.Name)
			return reconcile.Result{}, err
		}
		// DeploymentConfig created successfully - return and requeue
		// NOTE: that the requeue is made with the purpose to provide the deploymentconfig object for the next step to ensure the deploymentconfig size is the same as the spec.
		// Also, you could GET the deploymentconfig object again instead of requeue if you wish. See more over it here: https://godoc.org/sigs.k8s.io/controller-runtime/pkg/reconcile#Reconciler
		return reconcile.Result{Requeue: true}, nil
	} else if err != nil {
		reqLogger.Error(err, "Failed to get Deployment Config.")
		return reconcile.Result{}, err
	}

	// deployment;config created successfully - don't requeue
	currentStatus="DeploymentConfig created"
	if !reflect.DeepEqual(currentStatus, instance.Status.Status) {
		instance.Status.Status=currentStatus
		if err := r.client.Status().Update(context.TODO(), instance); err != nil {
			return reconcile.Result{},err
		}
	}
	// Ensure the deploymentconfig size is the same as the spec
	size := instance.Spec.Size
	if *deployConfig.Spec.Replicas != size {
		deployConfig.Spec.Replicas = &size
		err = r.client.Update(context.TODO(), deployConfig)
		if err != nil {
			reqLogger.Error(err, "Failed to update Deployment Config.", "DeploymentConfig.Namespace", deployConfig.Namespace, "DeploymentDonfig.Name", deployConfig.Name)
			return reconcile.Result{}, err
		}
	}
	return reconcile.Result{}, nil
	
	/*// Define a new Pod object
	pod := newPodForCR(instance)

	// Set Busybox instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, pod, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Check if this Pod already exists
	found := &corev1.Pod{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: pod.Name, Namespace: pod.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a new Pod", "Pod.Namespace", pod.Namespace, "Pod.Name", pod.Name)
		err = r.client.Create(context.TODO(), pod)
		if err != nil {
			return reconcile.Result{}, err
		}

		// Pod created successfully - don't requeue
		return reconcile.Result{}, nil
	} else if err != nil {
		return reconcile.Result{}, err
	}

	// Pod already exists - don't requeue
	reqLogger.Info("Skip reconcile: Pod already exists", "Pod.Namespace", found.Namespace, "Pod.Name", found.Name)
	return reconcile.Result{}, nil*/
}

// newPodForCR returns a busybox pod with the same name/namespace as the cr
/*func newPodForCR(cr *busyboxv1alpha1.Busybox) *corev1.Pod {
	labels := map[string]string{
		"app": cr.Name,
	}
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name + "-pod",
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:    "busybox",
					Image:   "busybox",
					Command: []string{"sleep", "3600"},
				},
			},
		},
	}
}*/

func (r *ReconcileBusybox) deploymentConfigForBusybox(m *busyboxv1alpha1.Busybox) *appsv1.DeploymentConfig {
	ls := map[string]string{
		"app": m.Name,
	}
	replicas := m.Spec.Size
	dep := &appsv1.DeploymentConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.Name,
			Namespace: m.Namespace,
		},
		Spec: appsv1.DeploymentConfigSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: ls,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: ls,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: "busybox-operator",
					Containers: []corev1.Container{{
                                        	Name: "busybox",
						Image: "busybox",
						ImagePullPolicy: corev1.PullAlways,
			                        Command: []string{"sleep", "3600"},
					}},	
				},
			},
		},
	}
	// Set instance as the owner of the Deployment.
	controllerutil.SetControllerReference(m, dep, r.scheme)
	return dep
}
