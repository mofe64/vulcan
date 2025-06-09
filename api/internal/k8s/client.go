package k8s

import (
	"context"

	platformv1 "github.com/mofe64/vulkan/operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// New returns a controller-runtime client that can talk to the Kubernetes API
// and understands all of our custom resource types (Org, Project, …).
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
	// build a *runtime.Scheme; this is a registry of all API types the
	//    client can encode/decode (core K8s + our CRDs).
	scheme := runtime.NewScheme()

	// add our CRD Go types to the scheme so the client can handle them.
	//    platformv1 is the package that Kubebuilder generated for:
	//    apiVersion: platform.io/v1alpha1
	_ = platformv1.AddToScheme(scheme)

	// finally create the typed client with the config + scheme.
	return client.New(cfg, client.Options{Scheme: scheme})
}
