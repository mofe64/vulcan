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

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	platformv1alpha1 "github.com/mofe64/vulkan/operator/api/v1alpha1"
	"github.com/mofe64/vulkan/operator/internal/metrics"
	"github.com/patrickmn/go-cache"
)

// OrgReconciler reconciles a Org object
type OrgReconciler struct {
	client.Client
	Scheme           *runtime.Scheme
	clusterCache     *cache.Cache
	applicationCache *cache.Cache

	Metrics *metrics.Metrics
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

	// quota Validation
	if err := r.validateQuota(ctx, &org); err != nil {
		return ctrl.Result{}, err
	}

	// update metrics
	if err := r.updateMetrics(ctx, &org); err != nil {
		return ctrl.Result{}, err
	}

	if org.Status.Phase == "" {
		org.Status.Phase = "Ready"
		if err := r.Status().Update(ctx, &org); err != nil {
			return ctrl.Result{}, err
		}
		log.Info("Org ready", "id", org.Name)
	}

	return ctrl.Result{}, nil

}

func (r *OrgReconciler) validateQuota(ctx context.Context, org *platformv1alpha1.Org) error {
	// Get counts with caching
	clusterCount, err := r.countClustersWithCache(ctx, org.Name)
	if err != nil {
		return fmt.Errorf("failed to count clusters: %w", err)
	}

	appCount, err := r.countApplicationsWithCache(ctx, org.Name)
	if err != nil {
		return fmt.Errorf("failed to count applications: %w", err)
	}

	// Validate against quota
	if clusterCount > org.Spec.OrgQuota.Clusters {
		return fmt.Errorf("cluster quota exceeded: %d/%d", clusterCount, org.Spec.OrgQuota.Clusters)
	}

	if appCount > org.Spec.OrgQuota.Apps {
		return fmt.Errorf("application quota exceeded: %d/%d", appCount, org.Spec.OrgQuota.Apps)
	}

	return nil
}

func (r *OrgReconciler) updateMetrics(ctx context.Context, org *platformv1alpha1.Org) error {
	clusterCount, err := r.countClustersWithCache(ctx, org.Name)
	if err != nil {
		return fmt.Errorf("failed to count clusters: %w", err)
	}

	appCount, err := r.countApplicationsWithCache(ctx, org.Name)
	if err != nil {
		return fmt.Errorf("failed to count applications: %w", err)
	}

	// Update org status with current counts
	org.Status.Metrics.Clusters = clusterCount
	org.Status.Metrics.Apps = appCount

	return nil

}

func (r *OrgReconciler) countClusters(ctx context.Context, orgName string) (int32, error) {

	// get all clusters
	var clusters platformv1alpha1.ClusterList
	if err := r.List(ctx, &clusters); err != nil {
		return 0, fmt.Errorf("failed to list clusters: %w", err)
	}

	// count clusters that belong to the org
	count := int32(0)
	for _, cluster := range clusters.Items {
		// check if the cluster belongs to the org
		if cluster.Spec.OrgRef == orgName {
			count++
		}
	}

	return count, nil
}

func (r *OrgReconciler) countApplications(ctx context.Context, orgName string) (int32, error) {

	// get all projects for this org
	var projects platformv1alpha1.ProjectList
	if err := r.List(ctx, &projects, client.MatchingLabels{
		"spec.orgRef": orgName,
	}); err != nil {
		return 0, fmt.Errorf("failed to list projects: %w", err)
	}

	// get all applications
	var applications platformv1alpha1.ApplicationList
	if err := r.List(ctx, &applications); err != nil {
		return 0, fmt.Errorf("failed to list applications: %w", err)
	}

	// count applications in the projects
	count := int32(0)
	projectNames := make(map[string]bool)

	// create a map of project names for quick lookup
	for _, project := range projects.Items {
		projectNames[project.Name] = true
	}

	// count applications in these projects
	for _, app := range applications.Items {
		if projectNames[app.Spec.ProjectRef] {
			count++
		}
	}

	return count, nil
}

// countClustersWithCache uses caching for better performance
func (r *OrgReconciler) countClustersWithCache(ctx context.Context, orgName string) (int32, error) {
	// try to get from cache first
	if count, ok := r.clusterCache.Get(orgName); ok {
		return count.(int32), nil
	}

	// if not in cache, count from API
	count, err := r.countClusters(ctx, orgName)
	if err != nil {
		return 0, err
	}

	// update cache
	r.clusterCache.Set(orgName, count, cache.DefaultExpiration)

	// update metrics
	r.Metrics.UpdateClusterCount(orgName, count)

	return count, nil
}

// countApplicationsWithCache uses caching for better performance
func (r *OrgReconciler) countApplicationsWithCache(ctx context.Context, orgName string) (int32, error) {
	// try to get from cache first
	if count, ok := r.applicationCache.Get(orgName); ok {
		return count.(int32), nil
	}

	// if not in cache, count from API
	count, err := r.countApplications(ctx, orgName)
	if err != nil {
		return 0, err
	}

	// update cache
	r.applicationCache.Set(orgName, count, cache.DefaultExpiration)

	// update metrics
	r.Metrics.UpdateApplicationCount(orgName, count)

	return count, nil
}

func (r *OrgReconciler) setupOwnerRBAC(ctx context.Context, org *platformv1alpha1.Org) error {
	// create role
	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("org-%s-owner", org.Name),
			Namespace: org.Namespace,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{"platform.io"},
				Resources: []string{"projects", "applications"},
				Verbs:     []string{"*"},
			},
		},
	}

	// create RoleBinding
	roleBinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("org-%s-owner-binding", org.Name),
			Namespace: org.Namespace,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind: "User",
				Name: org.Spec.OwnerEmail,
			},
		},
		RoleRef: rbacv1.RoleRef{
			Kind: "Role",
			Name: role.Name,
		},
	}

	return r.Create(ctx, roleBinding)
}

// SetupWithManager sets up the controller with the Manager.
func (r *OrgReconciler) SetupWithManager(mgr ctrl.Manager) error {

	// initialize caches
	r.clusterCache = cache.New(5*time.Minute, 10*time.Minute)
	r.applicationCache = cache.New(5*time.Minute, 10*time.Minute)
	return ctrl.NewControllerManagedBy(mgr).
		For(&platformv1alpha1.Org{}).
		Named("org").
		Complete(r)
}
