package controller

import (
	"context"
	"time"

	"github.com/mofe64/vulkan/operator/internal/metrics"
	"github.com/mofe64/vulkan/operator/internal/utils"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	platformv1alpha1 "github.com/mofe64/vulkan/operator/api/v1alpha1"
)

// ProjectReconciler reconciles a Project object
type ProjectReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=platform.platform.io,resources=projects,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=platform.platform.io,resources=projects/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=platform.platform.io,resources=projects/finalizers,verbs=update

func (r *ProjectReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	var proj platformv1alpha1.Project
	if err := r.Get(ctx, req.NamespacedName, &proj); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	if !proj.DeletionTimestamp.IsZero() {
		log.Info("Project is being deleted", "id", proj.Name, "org", proj.Spec.OrgRef, "deletionTimestamp", proj.DeletionTimestamp)
		// project is being deleted
		if utils.ContainsString(proj.ObjectMeta.Finalizers, platformv1alpha1.ProjectFinalizer) {
			log.Info("Finalizing Project", "id", proj.Name, "org", proj.Spec.OrgRef)

			// delete all cluster bindings for this project
			var clusterBindings platformv1alpha1.ProjectClusterBindingList
			if err := r.List(ctx, &clusterBindings); err != nil {
				log.Error(err, "Failed to list cluster bindings", "id", proj.Name, "org", proj.Spec.OrgRef)
				return ctrl.Result{}, err
			}

			var projectClusterBindings platformv1alpha1.ProjectClusterBindingList
			for _, cb := range clusterBindings.Items {
				if cb.Spec.ProjectRef == proj.Spec.ProjectID {
					projectClusterBindings.Items = append(projectClusterBindings.Items, cb)
				}
			}

			for _, clusterBinding := range projectClusterBindings.Items {
				if err := r.Delete(ctx, &clusterBinding); err != nil {
					log.Error(err, "Failed to delete cluster binding", "projectName",
						proj.Spec.DisplayName, "clusterRef", clusterBinding.Spec.ClusterRef)
					apimeta.SetStatusCondition(&proj.Status.Conditions, metav1.Condition{
						Type:    platformv1alpha1.Error,
						Status:  metav1.ConditionTrue,
						Reason:  "ClusterBindingDeletionError",
						Message: err.Error(),
					})
					if err := r.Status().Update(ctx, &proj); err != nil {
						log.Error(err, "Failed to update project status", "projectName", proj.Spec.DisplayName)
						return ctrl.Result{}, err
					}
					return ctrl.Result{
						RequeueAfter: time.Minute * 5,
					}, err
				}

			}

			// remove the finalizer
			proj.ObjectMeta.Finalizers = utils.RemoveString(proj.ObjectMeta.Finalizers, platformv1alpha1.ProjectFinalizer)
			if err := r.Update(ctx, &proj); err != nil {
				log.Error(err, "Failed to update project", "projectName", proj.Spec.DisplayName)
				return ctrl.Result{}, err
			}

			// decrement the project metrics
			metrics.DecProjects(proj.Spec.OrgRef)

		}

		return ctrl.Result{}, nil
	}

	apimeta.SetStatusCondition(&proj.Status.Conditions, metav1.Condition{
		Type:    platformv1alpha1.Ready,
		Status:  metav1.ConditionTrue,
		Reason:  "Reconciled",
		Message: "Project is healthy",
	})
	if err := r.Status().Update(ctx, &proj); err != nil {
		log.Error(err, "Failed to update project status", "projectName", proj.Spec.DisplayName)
		return ctrl.Result{}, err
	}

	// add the finalizer
	if !utils.ContainsString(proj.ObjectMeta.Finalizers, platformv1alpha1.ProjectFinalizer) {
		proj.ObjectMeta.Finalizers = append(proj.ObjectMeta.Finalizers, platformv1alpha1.ProjectFinalizer)
		if err := r.Update(ctx, &proj); err != nil {
			log.Error(err, "Error updating project")
			return ctrl.Result{}, err
		}
	}

	metrics.IncProjects(proj.Spec.OrgRef)

	log.Info("Project reconciled", "id", proj.Spec.ProjectID)
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ProjectReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&platformv1alpha1.Project{}).
		Named("project").
		Complete(r)
}
