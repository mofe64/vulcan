package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ProjectSpec defines the desired state of Project.
type ProjectSpec struct {
	// OrgRef is the reference to the name of the org cr that the project belongs to.
	// +kubebuilder:validation:Required
	OrgRef string `json:"orgRef"`

	// ProjectID is a unique identifier for the project
	// +kubebuilder:validation:Pattern=`^[0-9a-fA-F-]{36}$`
	// +kubebuilder:validation:Unique
	// +kubebuilder:validation:Required
	ProjectID string `json:"projectID"`

	// DisplayName is a human-readable name for the project
	// +kubebuilder:validation:MinLength=3
	// +kubebuilder:validation:MaxLength=100
	// +kubebuilder:validation:Unique
	// +kubebuilder:validation:Required
	DisplayName string `json:"displayName"`

	// ProjectMaxCores is the maximum number of cores that can be used by the project
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=100
	// +kubebuilder:default=100
	ProjectMaxCores int `json:"projectMaxCores"`

	// ProjectMaxMemory is the maximum amount of memory that can be used by the project
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=100
	// +kubebuilder:default=10
	ProjectMaxMemory int `json:"projectMaxMemoryInGigabytes"`

	// ProjectMaxStorage is the maximum amount of storage that can be used by the project
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=100
	// +kubebuilder:default=20
	ProjectMaxStorage int `json:"projectMaxEphemeralStorageInGigabytes"`

	// ProjectNamespace is the namespace that the project will be deployed to
	// if not provided, a namespace will be created with the name of the project
	// +kubebuilder:validation:Optional
	ProjectNamespace string `json:"projectNamespace,omitempty"`
}

// ProjectStatus defines the observed state of Project.
type ProjectStatus struct {
	// +patchMergeKey=type
	// +patchStrategy=merge
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster

// Project is the Schema for the projects API.
type Project struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ProjectSpec   `json:"spec,omitempty"`
	Status ProjectStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ProjectList contains a list of Project.
type ProjectList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Project `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Project{}, &ProjectList{})
}
