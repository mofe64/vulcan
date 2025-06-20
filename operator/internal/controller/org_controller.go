package controller

import (
	"context"

	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	platformv1alpha1 "github.com/mofe64/vulkan/operator/api/v1alpha1"
)

// OrgReconciler reconciles a Org object
type OrgReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=platform.platform.io,resources=orgs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=platform.platform.io,resources=orgs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=platform.platform.io,resources=orgs/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *OrgReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	var org platformv1alpha1.Org
	if err := r.Get(ctx, req.NamespacedName, &org); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// set the org ready condition to true
	apimeta.SetStatusCondition(&org.Status.Conditions, metav1.Condition{
		Type:               platformv1alpha1.Ready,
		Status:             metav1.ConditionTrue,
		Reason:             "Reconciled",
		Message:            "Org is ready",
		ObservedGeneration: org.GetGeneration(),
	})
	_ = r.Status().Update(ctx, &org)
	log.Info("Org ready", "id", org.Name)
	return ctrl.Result{}, nil

}

// SetupWithManager sets up the controller with the Manager.
func (r *OrgReconciler) SetupWithManager(mgr ctrl.Manager) error {

	return ctrl.NewControllerManagedBy(mgr).
		For(&platformv1alpha1.Org{}).
		Named("org").
		Complete(r)
}
