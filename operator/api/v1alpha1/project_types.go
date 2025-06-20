package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ProjectSpec defines the desired state of Project.
type ProjectSpec struct {
	// OrgRef is the reference to the name of theorganization this project belongs to
	// +kubebuilder:validation:Pattern=`^[0-9a-fA-F-]{36}$`
	OrgRef string `json:"orgRef"`

	// DisplayName is a human-readable name for the project
	// +kubebuilder:validation:MinLength=3
	// +kubebuilder:validation:MaxLength=100
	DisplayName string `json:"displayName"`

	// ClusterSelector tells the platform where workloads
	// created in this Project should be deployed by default.
	ClusterSelector ClusterSelector `json:"clusterSelector"`

	// CIRepoDefault is an optional Git URL pre-filled in the UI.
	// +kubebuilder:validation:Format=uri
	CIRepoDefault string `json:"ciRepoDefault,omitempty"`
}

type ClusterSelector struct {
	//'attached', 'eks', 'aks', 'gke', etc.
	// +kubebuilder:validation:Enum=attached;eks;aks;gke
	Type string `json:"type"`

	// Required for cloud clusters like EKS, AKS, GKE
	Region string `json:"regions,omitempty"`
}

// ProjectStatus defines the observed state of Project.
type ProjectStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Namespace string `json:"namespace,omitempty"` // created by project controller
	Phase     string `json:"phase,omitempty"`     // Pending → Ready
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
