package controller

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	platformv1alpha1 "github.com/mofe64/vulkan/operator/api/v1alpha1"
	"github.com/mofe64/vulkan/operator/internal/utils"
)

// ClusterReconciler reconciles a Cluster object
type ClusterReconciler struct {
	client.Client
	Scheme        *runtime.Scheme
	TargetFactory utils.TargetClientFactory
}

// +kubebuilder:rbac:groups=platform.platform.io,resources=clusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=platform.platform.io,resources=clusters/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=platform.platform.io,resources=clusters/finalizers,verbs=update
func (r *ClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	log.Info("Reconciling Cluster", "name", req.Name, "namespace", req.Namespace)
	var clu platformv1alpha1.Cluster
	if err := r.Get(ctx, req.NamespacedName, &clu); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// set the cluster ready condition to unknown
	// since we haven't done validation yet
	apimeta.SetStatusCondition(&clu.Status.Conditions, metav1.Condition{
		Type:               platformv1alpha1.Ready,
		Status:             metav1.ConditionUnknown,
		Reason:             "Reconciling",
		Message:            "Waiting for controller checks",
		ObservedGeneration: clu.GetGeneration(),
	})

	// check kubeconfig secret exists
	var secret corev1.Secret
	err := r.Get(ctx, types.NamespacedName{Name: clu.Spec.KubeconfigSecret, Namespace: "default"}, &secret)
	if err != nil {
		// set the cluster ready condition to false
		apimeta.SetStatusCondition(&clu.Status.Conditions, metav1.Condition{
			Type:               platformv1alpha1.Ready,
			Status:             metav1.ConditionFalse,
			Reason:             "KubeconfigSecretMissing",
			Message:            "Secret " + clu.Spec.KubeconfigSecret + " not found",
			ObservedGeneration: clu.GetGeneration(),
		})
		// set the cluster error condition to true
		apimeta.SetStatusCondition(&clu.Status.Conditions, metav1.Condition{
			Type:               platformv1alpha1.Error,
			Status:             metav1.ConditionTrue,
			Reason:             "KubeconfigSecretMissing",
			Message:            "Cluster cannot be contacted without kubeconfig",
			ObservedGeneration: clu.GetGeneration(),
		})

		// update the cluster status
		_ = r.Status().Update(ctx, &clu)

		// requeue after a short delay
		return ctrl.Result{RequeueAfter: time.Minute * 5}, nil
	}

	// check the cluster health
	isHealthy, msg, err := r.checkClusterHealth(ctx, &clu)
	if err != nil || !isHealthy {
		log.Error(err, "Cluster health check failed", "message", msg)
		apimeta.SetStatusCondition(&clu.Status.Conditions, metav1.Condition{
			Type:               platformv1alpha1.Ready,
			Status:             metav1.ConditionFalse,
			Reason:             "HealthCheckFailed",
			Message:            "Health probe failed: " + msg,
			ObservedGeneration: clu.GetGeneration(),
		})
		apimeta.SetStatusCondition(&clu.Status.Conditions, metav1.Condition{
			Type:               platformv1alpha1.Error,
			Status:             metav1.ConditionTrue,
			Reason:             "HealthCheckFailed",
			Message:            msg,
			ObservedGeneration: clu.GetGeneration(),
		})
		_ = r.Status().Update(ctx, &clu)
		return ctrl.Result{RequeueAfter: time.Minute * 2}, nil
	}

	// set the cluster ready condition to true
	apimeta.SetStatusCondition(&clu.Status.Conditions, metav1.Condition{
		Type:               platformv1alpha1.Ready,
		Status:             metav1.ConditionTrue,
		Reason:             "Reconciled",
		Message:            "Cluster is healthy",
		ObservedGeneration: clu.GetGeneration(),
	})
	// clear the Error flag if it was set previously
	apimeta.SetStatusCondition(&clu.Status.Conditions, metav1.Condition{
		Type:               platformv1alpha1.Error,
		Status:             metav1.ConditionFalse,
		Reason:             "NoError",
		Message:            "No outstanding errors",
		ObservedGeneration: clu.GetGeneration(),
	})
	_ = r.Status().Update(ctx, &clu)
	log.Info("Cluster reconciled", "id", clu.Name, "phase", clu.Status.Conditions)
	return ctrl.Result{}, nil

}

func (r *ClusterReconciler) checkClusterHealth(ctx context.Context, clu *platformv1alpha1.Cluster) (bool, string, error) {
	tgt, err := r.TargetFactory.ClientFor(ctx, clu)
	log := logf.FromContext(ctx)
	if err != nil {
		return false, "Failed to create client for cluster", err
	}

	var nodes corev1.NodeList
	if err := tgt.List(ctx, &nodes); err != nil {
		return false, "unable to list nodes", fmt.Errorf("listing nodes: %w", err)
	}

	if len(nodes.Items) == 0 {
		return false, "No nodes found in cluster", nil
	}
	for _, node := range nodes.Items {
		for _, condition := range node.Status.Conditions {
			if condition.Type == corev1.NodeReady && condition.Status != corev1.ConditionTrue {
				return false, "Node " + node.Name + " is not ready", nil
			}
		}
	}

	for _, n := range nodes.Items {
		for _, c := range n.Status.Conditions {
			if c.Type == corev1.NodeReady && c.Status != corev1.ConditionTrue {
				return false,
					fmt.Sprintf("node %s is not Ready", n.Name),
					nil
			}
		}
	}
	log.Info("Cluster is healthy", "nodes", len(nodes.Items))

	return true, "Cluster is healthy", nil

}

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&platformv1alpha1.Cluster{}).
		Named("cluster").
		Complete(r)
}
