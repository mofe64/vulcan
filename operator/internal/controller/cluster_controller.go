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
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	platformv1alpha1 "github.com/mofe64/vulkan/operator/api/v1alpha1"
)

// ClusterReconciler reconciles a Cluster object
type ClusterReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=platform.platform.io,resources=clusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=platform.platform.io,resources=clusters/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=platform.platform.io,resources=clusters/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.21.0/pkg/reconcile
func (r *ClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	log.Info("Reconciling Cluster", "name", req.Name, "namespace", req.Namespace)
	var clu platformv1alpha1.Cluster
	if err := r.Get(ctx, req.NamespacedName, &clu); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	// Very naive health check – mark ready if kubeconfig secret exists
	var secret corev1.Secret
	err := r.Get(ctx, types.NamespacedName{Name: clu.Spec.KubeconfigSecret, Namespace: "default"}, &secret)
	if err != nil {
		clu.Status.Phase = "Error"
		clu.Status.Msg = "missing kubeconfig secret"
	} else {
		clu.Status.Phase = "Ready"
		clu.Status.Msg = ""
	}
	if updateErr := r.Status().Update(ctx, &clu); updateErr != nil {
		return ctrl.Result{}, updateErr
	}
	log.Info("Cluster reconciled", "id", clu.Name, "phase", clu.Status.Phase)
	return ctrl.Result{RequeueAfter: 10 * time.Minute}, nil

}

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&platformv1alpha1.Cluster{}).
		Named("cluster").
		Complete(r)
}
