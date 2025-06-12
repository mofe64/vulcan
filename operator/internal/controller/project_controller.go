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
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	platformv1alpha1 "github.com/mofe64/vulkan/operator/api/v1alpha1"
	"github.com/mofe64/vulkan/operator/internal/util"
)

// ProjectReconciler reconciles a Project object
type ProjectReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=platform.platform.io,resources=projects,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=platform.platform.io,resources=projects/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=platform.platform.io,resources=projects/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.21.0/pkg/reconcile
func (r *ProjectReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	var proj platformv1alpha1.Project
	if err := r.Get(ctx, req.NamespacedName, &proj); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Ensure the project has a namespace in the *control-plane* cluster for CI‑build pods
	ns := fmt.Sprintf("cp-proj-%s", proj.Name)
	if err := util.EnsureNamespace(ctx, r.Client, ns, proj.Spec.DisplayName); err != nil {
		return ctrl.Result{}, err
	}

	if proj.Status.Phase == "" {
		proj.Status.Phase = "Ready"
		_ = r.Status().Update(ctx, &proj)
	}

	log.Info("Project reconciled", "id", proj.Name)
	return ctrl.Result{RequeueAfter: 15 * time.Minute}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ProjectReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&platformv1alpha1.Project{}).
		Named("project").
		Complete(r)
}
