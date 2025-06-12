/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

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
	"github.com/mofe64/vulkan/operator/internal/util"
)

// ProjectClusterBindingReconciler reconciles a ProjectClusterBinding object
type ProjectClusterBindingReconciler struct {
	client.Client
	Scheme        *runtime.Scheme
	TargetFactory *util.TargetClientFactory // helper to create a client for a Cluster CRD
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
	// Build a client to the *target* cluster using kubeconfig secret
	tgt, err := r.TargetFactory.ClientFor(ctx, &clu)
	if err != nil {
		return r.fail(ctx, &binding, "kubeconfig", err)
	}

	ns := fmt.Sprintf("proj-%s", proj.Name) // namespace pattern

	if err := util.EnsureNamespace(ctx, tgt, ns, proj.Spec.DisplayName); err != nil {
		return r.fail(ctx, &binding, "namespace", err)
	}

	// todo: fetch project members for this project and assign them roles in the target cluster based on their roles in the project

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

// SetupWithManager sets up the controller with the Manager.
func (r *ProjectClusterBindingReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&platformv1alpha1.ProjectClusterBinding{}).
		Named("projectclusterbinding").
		Complete(r)
}
