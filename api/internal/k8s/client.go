package k8s

import (
	"context"
	"fmt"

	platformv1 "github.com/mofe64/vulkan/operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// *runtime.Scheme; this is a registry of all API types the
//
//	client can encode/decode (core K8s + our CRDs).
var scheme *runtime.Scheme

func init() {
	scheme = runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = platformv1.AddToScheme(scheme)
}

// NewControlPlane builds a client for the **cluster Vulkan is running in**.
// inCluster should be true when code executes in a Pod, false when running on
// a developer laptop.
func New(ctx context.Context, inCluster bool) (client.Client, error) {
	var cfg *rest.Config
	var err error
	if inCluster {
		// if Running INSIDE a pod → use the service-account token and the
		//    env vars KUBERNETES_SERVICE_HOST / _PORT injected by Kubernetes
		cfg, err = rest.InClusterConfig()
	} else {
		// if running outside pod (eg developer laptop) → fall back to ~/.kube/config
		cfg, err = clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
	}
	// propagate error if we could not build a rest.Config
	if err != nil {
		return nil, err
	}

	// add our CRD Go types to the scheme so the client can handle them.
	//    platformv1 is the package that Kubebuilder generated for:
	//    apiVersion: platform.io/v1alpha1
	_ = platformv1.AddToScheme(scheme)

	// finally create the typed client with the config + scheme.
	return client.New(cfg, client.Options{Scheme: scheme})
}

// NewRemoteFromSecret builds a client for a remote workload cluster whose
// kubeconfig YAML is stored in a Secret *inside the control‑plane cluster*.
//
//	cpClient      – already‑initialised client for the control‑plane
//	secretNS/Name – location of the Secret that holds key "kubeconfig" (bytes)
func NewRemoteFromSecret(ctx context.Context, cpClient client.Client, secretNS, secretName string) (client.Client, error) {
	var sec corev1.Secret
	if err := cpClient.Get(ctx, types.NamespacedName{Name: secretName, Namespace: secretNS}, &sec); err != nil {
		return nil, fmt.Errorf("load kubeconfig secret: %w", err)
	}

	kubeBytes, ok := sec.Data["kubeconfig"]
	if !ok {
		return nil, fmt.Errorf("secret %s/%s missing key 'kubeconfig'", secretNS, secretName)
	}

	restCfg, err := clientcmd.RESTConfigFromKubeConfig(kubeBytes)
	if err != nil {
		return nil, fmt.Errorf("parse kubeconfig: %w", err)
	}

	return client.New(restCfg, client.Options{Scheme: scheme})
}

// BuildFromBytes is exported in case a reconciler already has the raw kubeconfig
// (e.g., from an ExternalSecret) and wants a client directly.
func BuildFromBytes(kubeconfig []byte) (client.Client, error) {
	restCfg, err := clientcmd.RESTConfigFromKubeConfig(kubeconfig)
	if err != nil {
		return nil, err
	}
	return client.New(restCfg, client.Options{Scheme: scheme})
}
