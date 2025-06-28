package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ProjectClusterBindingSpec defines the desired state of ProjectClusterBinding.
type ProjectClusterBindingSpec struct {
	// ProjectRef is the reference to the project that the application belongs to.
	ProjectRef string `json:"projectRef"`

	// ClusterRef is the reference to the cluster that the application belongs to.
	ClusterRef string `json:"clusterRef"`
}

// ProjectClusterBindingStatus defines the observed state of ProjectClusterBinding.
type ProjectClusterBindingStatus struct {

	// +patchMergeKey=type
	// +patchStrategy=merge
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster

// ProjectClusterBinding is the Schema for the projectclusterbindings API.
type ProjectClusterBinding struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ProjectClusterBindingSpec   `json:"spec,omitempty"`
	Status ProjectClusterBindingStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ProjectClusterBindingList contains a list of ProjectClusterBinding.
type ProjectClusterBindingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ProjectClusterBinding `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ProjectClusterBinding{}, &ProjectClusterBindingList{})
}
