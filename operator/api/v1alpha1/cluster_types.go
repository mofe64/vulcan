package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Important: Run "make" to regenerate code after modifying this file

// ClusterSpec defines the desired state of Cluster.
type ClusterSpec struct {

	// OrgRef is the reference to the org that the cluster belongs to.
	OrgRef string `json:"orgRef"`

	// +kubebuilder:validation:Enum=attached;eks
	Type string `json:"type"`

	// Region is mandatory for managed clouds.
	Region string `json:"region,omitempty"`

	// NodePools for managed clusters.
	NodePools []NodePool `json:"nodePools,omitempty"`

	// When Type==attached, hold secret name that contains kubeconfig.
	KubeconfigSecret string `json:"kubeconfigSecret,omitempty"`
}

// NodePool describes ONE group of worker nodes that share the same
// size, scaling rules and scheduling hints.
//
// Why we need it
// --------------
// • A single Kubernetes cluster often mixes machine types:
//   - small on-demand nodes for web traffic
//   - larger spot nodes for background jobs
//   - GPU nodes for ML workloads
//     Each mix is expressed as a separate NodePool.
//   - When the Cluster reconciler talks to AWS/GKE/AKS it turns every
//     NodePool into a managed node-group, honouring min/max scaling.
//
// Fields explained
// ----------------
// Name – logical pool name (“default”, “gpu”, “spot”).
// InstanceType  – cloud SKU (t3.medium, n1-standard-4, …).
// MinSize/MaxSize – auto-scaler bounds.  If Desired is nil the cloud auto-scaler can pick any value inside this range.
// Desired – optional fixed replica count.  Overrides auto-scale.
// Labels – key/value tags added to every node so Deployments can target the pool with nodeSelector or topology spread.
// Taints – stop unrelated Pods from landing here unless they have a matching toleration; useful for GPU-only nodes.
type NodePool struct {
	// Logical name for users & dashboards.
	// e.g. "default", "gpu", "spot"
	Name string `json:"name"`

	// Cloud machine SKU.
	// AWS: "t3.medium"  GCP: "e2-standard-4"
	InstanceType string `json:"instanceType"`

	// Autoscaler lower & upper bounds.
	MinSize int32 `json:"minSize"`
	MaxSize int32 `json:"maxSize"`

	// Optional fixed size; when set we bypass autoscaler.
	// +kubebuilder:validation:Optional
	Desired *int32 `json:"desired,omitempty"`

	// Node labels copied to every node in this pool.
	// Apps can select the pool via nodeSelector.
	// +kubebuilder:validation:Optional
	Labels map[string]string `json:"labels,omitempty"`

	// Node taints copied to every node in this pool.
	// Forces only tolerating Pods to schedule here
	// (e.g. isolate GPU workloads).
	// +kubebuilder:validation:Optional
	Taints []corev1.Taint `json:"taints,omitempty"`
}

// ClusterStatus defines the observed state of Cluster.
type ClusterStatus struct {
	// Conditions represent the latest available observations
	// of the resource’s state.
	// +patchMergeKey=type
	// +patchStrategy=merge
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`
	// Endpoint is useful for CLI ‘kubeconfig’ command.
	Endpoint string `json:"endpoint,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster

// Cluster is the Schema for the clusters API.
type Cluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterSpec   `json:"spec,omitempty"`
	Status ClusterStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ClusterList contains a list of Cluster.
type ClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Cluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Cluster{}, &ClusterList{})
}
