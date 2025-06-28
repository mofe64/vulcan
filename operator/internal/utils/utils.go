package utils

import (
	"context"
	"fmt"
	"slices"

	platformv1alpha1 "github.com/mofe64/vulkan/operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// TargetClientFactory converts a Cluster CRD into a controller-runtime client
// that talks to *that* cluster.

type TargetClientFactory interface {
	ClientFor(ctx context.Context, clu *platformv1alpha1.Cluster) (client.Client, error)
}

type targetClientFactory struct {
	CP client.Client
}

func NewTargetClientFactory(cp client.Client) TargetClientFactory {
	return &targetClientFactory{CP: cp}
}

// ClientFor reads clu.Spec.KubeconfigSecret, builds rest.Config, returns a client.
func (f *targetClientFactory) ClientFor(ctx context.Context, clu *platformv1alpha1.Cluster) (client.Client, error) {
	// load cluster Secret that holds kubeconfig YAML
	var sec corev1.Secret
	err := f.CP.Get(ctx,
		client.ObjectKey{Namespace: clu.Spec.KubeconfigSecretNamespace, Name: clu.Spec.KubeconfigSecretName},
		&sec)
	if err != nil {
		return nil, err
	}

	// build rest.Config from kubeconfig bytes
	kubeYAML := sec.Data["kubeconfig"]
	cfg, err := clientcmd.RESTConfigFromKubeConfig(kubeYAML)
	if err != nil {
		return nil, err
	}
	cfg.QPS, cfg.Burst = 200, 400 // optional tuning

	// ceate a typed client with the same scheme
	return client.New(cfg, client.Options{Scheme: f.CP.Scheme()})
}

func EnsureNamespace(ctx context.Context, c client.Client, name, display string) error {
	var ns corev1.Namespace
	err := c.Get(ctx, types.NamespacedName{Name: name}, &ns)
	if client.IgnoreNotFound(err) != nil {
		return err // real error
	}
	if err == nil {
		// Namespace already exists – patch friendly label if needed
		if ns.Labels == nil {
			ns.Labels = map[string]string{}
		}
		if ns.Labels["vulkan.io/displayName"] != display {
			ns.Labels["vulkan.io/displayName"] = display
			return c.Update(ctx, &ns)
		}
		return nil // up‑to‑date
	}

	// Not found – create
	ns = corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"vulkan.io/displayName": display,
			},
		},
	}
	return c.Create(ctx, &ns)
}

func AddLabelsToNamespace(ctx context.Context, c client.Client, name string, labels map[string]string) error {
	var ns corev1.Namespace
	err := c.Get(ctx, types.NamespacedName{Name: name}, &ns)
	if client.IgnoreNotFound(err) != nil {
		return err
	}
	ns.Labels = labels
	return c.Update(ctx, &ns)
}

// ensureRoleBinding ensures the subject has desired role inside the namespace.
// ensureRoleBinding makes sure a RoleBinding exists that grants **one subject**
// (a user’s email) a standard permission level (admin / edit / view) **inside a
// single namespace**.
//
// Why do we reference a *ClusterRole* instead of creating a namespaced Role?
//  1. **Built‑ins already exist** – Kubernetes ships with the ClusterRoles
//     `admin`, `edit`, and `view`.  Re‑using them means we don’t have to
//     create or maintain a custom Role in *every* project namespace.
//  2. **Namespaced limitation still applies** – Although the RoleRef points
//     to a *cluster‑scoped* role, a RoleBinding that lives *inside* a
//     namespace automatically fences the permissions to *that* namespace
//     only.  So `ClusterRole/view` bound in `proj‑123` can list Pods only in
//     `proj‑123`.
//  3. **Consistent behaviour** – The same three ClusterRoles exist in every
//     Kubernetes distribution (on‑prem, EKS, GKE).  Using them keeps RBAC
//     uniform across all attached clusters.
//  4. **Less clutter** – Deleting the namespace instantly removes the
//     RoleBinding; no orphaned Role objects remain because we never created
//     any.
//
// If you later need a tighter permission set, simply change the RoleRef kind
// to "Role" and create a bespoke Role in each namespace – the rest of this
// helper stays the same.
func EnsureRoleBinding(ctx context.Context, c client.Client, ns, subject, role string) error {
	rbName := fmt.Sprintf("rb-%s-%s", role, subject)
	var rb rbacv1.RoleBinding
	err := c.Get(ctx, client.ObjectKey{Namespace: ns, Name: rbName}, &rb)
	if client.IgnoreNotFound(err) != nil {
		return err
	}
	if err == nil {
		return nil // already present
	}

	rb = rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: rbName},
		Subjects:   []rbacv1.Subject{{Kind: rbacv1.UserKind, Name: subject}},
		RoleRef:    rbacv1.RoleRef{Kind: "ClusterRole", Name: role, APIGroup: "rbac.authorization.k8s.io"},
	}
	return c.Create(ctx, &rb)
}

func ContainsString(slice []string, str string) bool {
	return slices.Contains(slice, str)
}

func RemoveString(slice []string, str string) []string {
	for i, s := range slice {
		if s == str {
			return append(slice[:i], slice[i+1:]...)
		}
	}
	return slice
}

// UpdateClusterStatusWithRetry persists the *Status* of a Cluster CR and
// automatically retries when the apiserver returns a 409 Conflict.
//
// Callers should:
//
//  1. Fetch (or already hold) a Cluster object.
//  2. Mutate **only** its .Status fields / conditions.
//  3. Pass that object to this function.
//
// If all retries are exhausted the last error is returned so that Reconcile
// can surface it and the request is re-queued.
func UpdateClusterStatusWithRetry(
	ctx context.Context,
	c client.Client,
	desired *platformv1alpha1.Cluster,
) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		// Always re-get in case another writer won the race.
		var current platformv1alpha1.Cluster
		if err := c.Get(ctx, client.ObjectKeyFromObject(desired), &current); err != nil {
			return err
		}

		// Copy the desired Status onto the fresh object.
		current.Status = desired.Status

		// Try to write it back.
		return c.Status().Update(ctx, &current)
	})
}
