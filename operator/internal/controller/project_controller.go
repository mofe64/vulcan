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

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	platformv1alpha1 "github.com/mofe64/vulkan/operator/api/v1alpha1"
	"github.com/mofe64/vulkan/operator/internal/utils"
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
	// ensure namespace exists in target cluster
	// namespace should ideally have been created in the projectclusterbinding controller
	// this is to reconfirm
	ns := fmt.Sprintf("proj-%s", proj.Name)
	if err := utils.EnsureNamespace(ctx, r.Client, ns, proj.Spec.DisplayName); err != nil {
		return ctrl.Result{}, err
	}

	// add labels to namespace
	if err := utils.AddLabelsToNamespace(ctx, r.Client, ns, map[string]string{
		"vulkan.io/project": proj.Name,
		"vulkan.io/org":     proj.Spec.OrgRef,
	}); err != nil {
		return ctrl.Result{}, err
	}

	// set resource quota
	quota := &corev1.ResourceQuota{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "project-quota",
			Namespace: ns,
		},
		// TODO:
		Spec: corev1.ResourceQuotaSpec{
			Hard: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("10"),
				corev1.ResourceMemory: resource.MustParse("20Gi"),
			},
		},
	}

	// create quota in namespace
	if err := r.Client.Create(ctx, quota); err != nil {
		return ctrl.Result{}, err
	}

	// create network policy
	networkPolicy := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "default-deny",
			Namespace: ns,
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{},
			PolicyTypes: []networkingv1.PolicyType{
				networkingv1.PolicyTypeIngress,
				networkingv1.PolicyTypeEgress,
			},
		},
	}

	// create network policy in namespace
	if err := r.Client.Create(ctx, networkPolicy); err != nil {
		return ctrl.Result{}, err
	}

	if proj.Status.Phase == "" {
		proj.Status.Phase = "Ready"
		_ = r.Status().Update(ctx, &proj)
	}

	log.Info("Project reconciled", "id", proj.Name)
	return ctrl.Result{}, nil
}

// func (r *ProjectReconciler) validateClusterSelection(ctx context.Context, project *v1alpha1.Project) error {
//     // 1. Get available clusters
//     var clusters v1alpha1.ClusterList
//     if err := r.List(ctx, &clusters, client.MatchingLabels{
//         "type": project.Spec.ClusterSelector.Type,
//     }); err != nil {
//         return err
//     }

//     // 2. Validate cluster health
//     for _, cluster := range clusters.Items {
//         if cluster.Status.Phase != "Ready" {
//             continue
//         }

//         // Check region if specified
//         if project.Spec.ClusterSelector.Region != "" {
//             if cluster.Spec.Region != project.Spec.ClusterSelector.Region {
//                 continue
//             }
//         }

//         // Check capacity
//         if err := r.checkClusterCapacity(ctx, &cluster); err != nil {
//             continue
//         }

//         return nil
//     }

//     return fmt.Errorf("no suitable cluster found")
// }

// SetupWithManager sets up the controller with the Manager.
func (r *ProjectReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&platformv1alpha1.Project{}).
		Named("project").
		Complete(r)
}
