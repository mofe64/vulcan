package controller

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/jackc/pgx/v5/pgxpool"
	platformv1alpha1 "github.com/mofe64/vulkan/operator/api/v1alpha1"
	"github.com/mofe64/vulkan/operator/internal/utils"
)

// ProjectClusterBindingReconciler reconciles a ProjectClusterBinding object
type ProjectClusterBindingReconciler struct {
	client.Client
	Scheme        *runtime.Scheme
	TargetFactory utils.TargetClientFactory // helper to create a client for a Cluster CRD
	DB            *pgxpool.Pool
}

// +kubebuilder:rbac:groups=platform.platform.io,resources=projectclusterbindings,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=platform.platform.io,resources=projectclusterbindings/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=platform.platform.io,resources=projectclusterbindings/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ProjectClusterBinding object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.21.0/pkg/reconcile
func (r *ProjectClusterBindingReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	log.Info("Reconciling ProjectClusterBinding", "name", req.Name, "namespace", req.Namespace)

	var binding platformv1alpha1.ProjectClusterBinding
	if err := r.Get(ctx, req.NamespacedName, &binding); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Fetch referenced Project & Cluster
	var proj platformv1alpha1.Project
	if err := r.Get(ctx, types.NamespacedName{Name: binding.Spec.ProjectID}, &proj); err != nil {
		return r.fail(ctx, &binding, "project lookup", err)
	}
	var clu platformv1alpha1.Cluster
	if err := r.Get(ctx, types.NamespacedName{Name: binding.Spec.ClusterID}, &clu); err != nil {
		return r.fail(ctx, &binding, "cluster lookup", err)
	}
	// TODO:
	// should check if we are using default (current)cluster, if yes,
	// then just use the request client, else
	// build a client to the *target* cluster using kubeconfig secret
	tgt, err := r.TargetFactory.ClientFor(ctx, &clu)
	if err != nil {
		return r.fail(ctx, &binding, "kubeconfig", err)
	}

	ns := fmt.Sprintf("proj-%s", proj.Name) // namespace pattern

	if err := utils.EnsureNamespace(ctx, tgt, ns, proj.Spec.DisplayName); err != nil {
		return r.fail(ctx, &binding, "namespace", err)
	}

	// todo: fetch project members for this project and assign them roles in the target cluster based on their roles in the project

	//todo: validate capacity

	// Update status → Ready
	if binding.Status.Phase != "Ready" {
		binding.Status.Phase = "Ready"
		binding.Status.Message = ""
		_ = r.Status().Update(ctx, &binding)
	}
	log.Info("Binding ready", "binding", binding.Name)
	return ctrl.Result{}, nil
}

func (r *ProjectClusterBindingReconciler) fail(ctx context.Context, b *platformv1alpha1.ProjectClusterBinding, msg string, err error) (ctrl.Result, error) {
	b.Status.Phase = "Error"
	b.Status.Message = fmt.Sprintf("%s: %v", msg, err)
	_ = r.Status().Update(ctx, b)
	return ctrl.Result{}, err
}

// func (r *ProjectClusterBindingReconciler) validateCapacity(ctx context.Context, cluster *v1alpha1.Cluster, project *v1alpha1.Project) error {
// 	// Get current resource usage
// 	var pods corev1.PodList
// 	if err := r.List(ctx, &pods, client.InNamespace(project.Status.Namespace)); err != nil {
// 		return err
// 	}

// 	// Calculate required resources
// 	var requiredCPU resource.Quantity
// 	var requiredMemory resource.Quantity

// 	for _, pod := range pods.Items {
// 		for _, container := range pod.Spec.Containers {
// 			requiredCPU.Add(*container.Resources.Requests.Cpu())
// 			requiredMemory.Add(*container.Resources.Requests.Memory())
// 		}
// 	}

// 	// Check against cluster capacity
// 	if requiredCPU.Cmp(cluster.Status.AvailableCPU) > 0 {
// 		return fmt.Errorf("insufficient CPU capacity")
// 	}

// 	if requiredMemory.Cmp(cluster.Status.AvailableMemory) > 0 {
// 		return fmt.Errorf("insufficient memory capacity")
// 	}

// 	return nil
// }

// SetupWithManager sets up the controller with the Manager.
func (r *ProjectClusterBindingReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&platformv1alpha1.ProjectClusterBinding{}).
		Named("projectclusterbinding").
		Complete(r)
}
