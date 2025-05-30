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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// OrgSpec defines the desired state of Org.
type OrgSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// DisplayName is a human-readable name for the organization
	// +kubebuilder:validation:MinLength=3
	// +kubebuilder:validation:MaxLength=100
	DisplayName string `json:"displayName"`

	// OwnerEmail receieves system notifications and is the primary contact for the organization
	// +kubebuilder:validation:Format=email
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
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Phase   string      `json:"phase,omitempty"`   // Phase indicates the current lifecycle phase of the organization (e.g., Pending, Ready, Failed)
	Metrics OrgCounters `json:"metrics,omitempty"` // Metrics contains counters for the number of clusters and apps in the organization
}

// OrgCounters holds counters for the number of clusters and applications in the organization.
// It is used to track the live usage of resources within the organization.
type OrgCounters struct {
	// Clusters is the number of clusters created in this organization
	Clusters int32 `json:"clusters,omitempty"`
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
