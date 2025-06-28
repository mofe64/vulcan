package controller

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/jackc/pgx/v5/pgxpool"
	platformv1alpha1 "github.com/mofe64/vulkan/operator/api/v1alpha1"
	"github.com/mofe64/vulkan/operator/internal/model"
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

func (r *ProjectClusterBindingReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	log.Info("Reconciling ProjectClusterBinding", "name", req.Name, "namespace", req.Namespace)

	var binding platformv1alpha1.ProjectClusterBinding
	if err := r.Get(ctx, req.NamespacedName, &binding); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Fetch referenced Project & Cluster
	var proj platformv1alpha1.Project
	if err := r.Get(ctx, types.NamespacedName{Name: binding.Spec.ProjectRef}, &proj); err != nil {
		// if err is not not found
		// set unknown condition
		// and requeue after 5 minutes
		if !errors.IsNotFound(err) {
			apimeta.SetStatusCondition(&binding.Status.Conditions, metav1.Condition{
				Type:    platformv1alpha1.Unknown,
				Status:  metav1.ConditionTrue,
				Reason:  "ProjectLookupFailed",
				Message: err.Error(),
			})
			_ = r.Status().Update(ctx, &binding)
			return ctrl.Result{RequeueAfter: time.Minute * 5}, err
		} else {
			apimeta.SetStatusCondition(&binding.Status.Conditions, metav1.Condition{
				Type:    platformv1alpha1.Error,
				Status:  metav1.ConditionTrue,
				Reason:  "ProjectLookupError",
				Message: err.Error(),
			})
			_ = r.Status().Update(ctx, &binding)
			return ctrl.Result{}, err
		}

	}
	var clu platformv1alpha1.Cluster
	if err := r.Get(ctx, types.NamespacedName{Name: binding.Spec.ClusterRef}, &clu); err != nil {
		// if err is not not found
		// set unknown condition
		// and requeue after 5 minutes
		if !errors.IsNotFound(err) {
			apimeta.SetStatusCondition(&binding.Status.Conditions, metav1.Condition{
				Type:    platformv1alpha1.Unknown,
				Status:  metav1.ConditionTrue,
				Reason:  "ClusterLookupFailed",
				Message: err.Error(),
			})
			_ = r.Status().Update(ctx, &binding)
			return ctrl.Result{RequeueAfter: time.Minute * 5}, err
		} else {
			apimeta.SetStatusCondition(&binding.Status.Conditions, metav1.Condition{
				Type:    platformv1alpha1.Error,
				Status:  metav1.ConditionTrue,
				Reason:  "ClusterLookupError",
				Message: err.Error(),
			})
			_ = r.Status().Update(ctx, &binding)
			return ctrl.Result{}, err
		}
	}
	var k8sClient client.Client
	var err error
	if clu.Spec.Type != "attached" {
		k8sClient, err = r.TargetFactory.ClientFor(ctx, &clu)
		if err != nil {
			apimeta.SetStatusCondition(&binding.Status.Conditions, metav1.Condition{
				Type:    platformv1alpha1.Error,
				Status:  metav1.ConditionTrue,
				Reason:  "ClusterTargetGenError",
				Message: err.Error(),
			})
			_ = r.Status().Update(ctx, &binding)
			return ctrl.Result{}, err
		}
	} else {
		k8sClient = r.Client
	}

	var ns string
	if proj.Spec.ProjectNamespace != "" {
		ns = proj.Spec.ProjectNamespace
	} else {
		ns = fmt.Sprintf("proj-%s-%s-%s", proj.Spec.OrgRef, proj.Name, proj.Spec.ProjectID)
	}

	if err := utils.EnsureNamespace(ctx, k8sClient, ns, proj.Spec.DisplayName); err != nil {
		apimeta.SetStatusCondition(&binding.Status.Conditions, metav1.Condition{
			Type:    platformv1alpha1.Error,
			Status:  metav1.ConditionTrue,
			Reason:  "NamespaceCreationError",
			Message: err.Error(),
		})
		apimeta.SetStatusCondition(&binding.Status.Conditions, metav1.Condition{
			Type:    platformv1alpha1.Ready,
			Status:  metav1.ConditionFalse,
			Reason:  "NamespaceCreationError",
			Message: err.Error(),
		})
		_ = r.Status().Update(ctx, &binding)
		return ctrl.Result{}, err
	}

	if err := utils.AddLabelsToNamespace(ctx, r.Client, ns, map[string]string{
		"vulkan.io/project":     proj.Name,
		"vulkan.io/projectID":   proj.Spec.ProjectID,
		"vulkan.io/displayName": proj.Spec.DisplayName,
		"vulkan.io/org":         proj.Spec.OrgRef,
	}); err != nil {
		apimeta.SetStatusCondition(&binding.Status.Conditions, metav1.Condition{
			Type:    platformv1alpha1.Error,
			Status:  metav1.ConditionTrue,
			Reason:  "NamespaceLabelingError",
			Message: err.Error(),
		})
		apimeta.SetStatusCondition(&binding.Status.Conditions, metav1.Condition{
			Type:    platformv1alpha1.Ready,
			Status:  metav1.ConditionFalse,
			Reason:  "NamespaceLabelingError",
			Message: err.Error(),
		})
		_ = r.Status().Update(ctx, &binding)
		return ctrl.Result{}, err
	}

	cpuLimitString := fmt.Sprintf("%d", proj.Spec.ProjectMaxCores)
	memoryLimitString := fmt.Sprintf("%dGi", proj.Spec.ProjectMaxMemory)
	storageLimitString := fmt.Sprintf("%dGi", proj.Spec.ProjectMaxStorage)

	// create resource quota
	quota := &corev1.ResourceQuota{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("quota-%s", proj.Name),
			Namespace: ns,
		},
		Spec: corev1.ResourceQuotaSpec{
			Hard: corev1.ResourceList{
				corev1.ResourceCPU:              resource.MustParse(cpuLimitString),
				corev1.ResourceMemory:           resource.MustParse(memoryLimitString),
				corev1.ResourceEphemeralStorage: resource.MustParse(storageLimitString),
			},
		},
	}

	if err := r.Client.Create(ctx, quota); err != nil {
		apimeta.SetStatusCondition(&binding.Status.Conditions, metav1.Condition{
			Type:    platformv1alpha1.Error,
			Status:  metav1.ConditionTrue,
			Reason:  "QuotaCreationError",
			Message: err.Error(),
		})
		apimeta.SetStatusCondition(&binding.Status.Conditions, metav1.Condition{
			Type:    platformv1alpha1.Ready,
			Status:  metav1.ConditionFalse,
			Reason:  "QuotaCreationError",
			Message: err.Error(),
		})
		_ = r.Status().Update(ctx, &binding)
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

	if err := r.Client.Create(ctx, networkPolicy); err != nil {
		apimeta.SetStatusCondition(&binding.Status.Conditions, metav1.Condition{
			Type:    platformv1alpha1.Error,
			Status:  metav1.ConditionTrue,
			Reason:  "NetworkPolicyCreationError",
			Message: err.Error(),
		})
		apimeta.SetStatusCondition(&binding.Status.Conditions, metav1.Condition{
			Type:    platformv1alpha1.Ready,
			Status:  metav1.ConditionFalse,
			Reason:  "NetworkPolicyCreationError",
			Message: err.Error(),
		})
		_ = r.Status().Update(ctx, &binding)
		return ctrl.Result{}, err
	}

	// Fetch project members for this project and assign them roles in the target cluster based on their roles in the project
	rows, err := r.DB.Query(ctx, `
		SELECT pm.project_id, pm.user_id, pm.role, u.email 
		FROM project_members pm 
		JOIN users u ON pm.user_id = u.id 
		WHERE pm.project_id = $1
	`, proj.Spec.ProjectID)
	if err != nil {
		apimeta.SetStatusCondition(&binding.Status.Conditions, metav1.Condition{
			Type:    platformv1alpha1.Error,
			Status:  metav1.ConditionTrue,
			Reason:  "ProjectMemberLookupError",
			Message: err.Error(),
		})
		_ = r.Status().Update(ctx, &binding)
		return ctrl.Result{}, err
	}
	defer rows.Close()
	projectMembers := []model.ProjectMember{}
	for rows.Next() {
		var member model.ProjectMember
		err := rows.Scan(&member.ProjectID, &member.UserID, &member.Role, &member.Email)
		if err != nil {
			return ctrl.Result{}, err
		}
		projectMembers = append(projectMembers, member)
	}

	// Create role bindings for each project member in the target cluster
	for _, member := range projectMembers {
		// Map project roles to Kubernetes roles
		var k8sRole string
		switch member.Role {
		case "admin":
			k8sRole = "admin"
		case "maintainer":
			k8sRole = "edit"
		case "viewer":
			k8sRole = "view"
		default:
			log.Info("Unknown role, skipping", "user", member.Email, "role", member.Role)
			continue
		}

		// Create role binding in the project namespace
		if err := utils.EnsureRoleBinding(ctx, k8sClient, ns, member.Email, k8sRole); err != nil {
			log.Error(err, "Failed to create role binding", "user", member.Email, "role", k8sRole, "namespace", ns)
			apimeta.SetStatusCondition(&binding.Status.Conditions, metav1.Condition{
				Type:    platformv1alpha1.Error,
				Status:  metav1.ConditionTrue,
				Reason:  "RoleBindingCreationError",
				Message: fmt.Sprintf("Failed to create role binding for user %s: %v", member.Email, err),
			})
			_ = r.Status().Update(ctx, &binding)
			return ctrl.Result{}, err
		}
		log.Info("Created role binding", "user", member.Email, "role", k8sRole, "namespace", ns)
	}

	// Set binding as ready
	apimeta.SetStatusCondition(&binding.Status.Conditions, metav1.Condition{
		Type:    platformv1alpha1.Ready,
		Status:  metav1.ConditionTrue,
		Reason:  "BindingReady",
		Message: fmt.Sprintf("Successfully created role bindings for %d project members in namespace %s", len(projectMembers), ns),
	})
	_ = r.Status().Update(ctx, &binding)

	log.Info("Binding ready", "binding", binding.Name)
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ProjectClusterBindingReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&platformv1alpha1.ProjectClusterBinding{}).
		Named("projectclusterbinding").
		Complete(r)
}
