package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// OrgSpec defines the desired state of Org.
type OrgSpec struct {

	// OrgID is a unique identifier for the organization
	// +kubebuilder:validation:Pattern=`^[0-9a-fA-F-]{36}$`
	// +kubebuilder:validation:Unique
	// +kubebuilder:validation:Required
	OrgID string `json:"orgID"`

	// DisplayName is a human-readable name for the organization
	// +kubebuilder:validation:MinLength=3
	// +kubebuilder:validation:MaxLength=100
	// +kubebuilder:validation:Required
	DisplayName string `json:"displayName"`

	// OwnerEmail receieves system notifications and is the primary contact for the organization
	// +kubebuilder:validation:Format=email
	// +kubebuilder:validation:Required
	OwnerEmail string `json:"ownerEmail"`

	// Quota defines the resource limits for the organization will be enforced by org controller
	OrgQuota OrgQuota `json:"quota,omitempty"`
}

type OrgQuota struct {
	// Clusters is the number of clusters that can be created in this organization
	// +kubebuilder:default=1
	Clusters int32 `json:"clusters,omitempty"`
	// Apps is the number of applications that can be created in this organization
	// +kubebuilder:default=100
	Apps int32 `json:"apps,omitempty"`
}

// OrgStatus defines the observed state of Org.
type OrgStatus struct {
	// Conditions represent the latest available observations
	// of the resource’s state.
	// +patchMergeKey=type
	// +patchStrategy=merge
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`

	Metrics OrgCounters `json:"metrics,omitempty"` // Metrics contains counters for the number of clusters, projects and apps in the organization
}

// OrgCounters holds counters for the number of clusters , projectsand applications in the organization.
// It is used to track the live usage of resources within the organization.
type OrgCounters struct {
	// Clusters is the number of clusters created in this organization
	Clusters int32 `json:"clusters,omitempty"`

	// Projects is the number of projects that can be created in this organization
	Projects int32 `json:"projects,omitempty"`

	// Apps is the number of applications created in this organization
	Apps int32 `json:"apps,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster

// Org is the Schema for the orgs API.
type Org struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OrgSpec   `json:"spec,omitempty"`
	Status OrgStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// OrgList contains a list of Org.
type OrgList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Org `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Org{}, &OrgList{})
}
