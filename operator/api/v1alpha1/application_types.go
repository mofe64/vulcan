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

// ApplicationSpec defines the desired state of Application.
type ApplicationSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Git repository to build & deploy.
	// +kubebuilder:validation:Format=uri
	RepoURL string `json:"repoURL"`

	// Build is either buildpack or dockerfile.
	Build BuildConfig `json:"build"`

	// Runtime environment variables (key=value)
	Env []EnvVar `json:"env,omitempty"`

	// Autoscaling policy (passed to HPA)
	Autoscaling HPAPolicy `json:"autoscaling,omitempty"`
}

type BuildConfig struct {
	// +kubebuilder:validation:Enum=buildpack;dockerfile
	Strategy string `json:"strategy"`
	// Branch or tag (defaults to main)
	Ref string `json:"ref,omitempty"`
	// Optional Dockerfile path, relevant only for dockerfile strategy
	Dockerfile string `json:"dockerfile,omitempty"`
}

type EnvVar struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type HPAPolicy struct {
	Min int32 `json:"minReplicas"`
	Max int32 `json:"maxReplicas"`
}

// ApplicationStatus defines the observed state of Application.
type ApplicationStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Latest image pushed by Tekton build.
	Image string `json:"image,omitempty"`
	// git SHA deployed
	Revision string `json:"revision,omitempty"`
	// Healthy, Progressing, Error
	Health string `json:"health,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Application is the Schema for the applications API.
type Application struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ApplicationSpec   `json:"spec,omitempty"`
	Status ApplicationStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ApplicationList contains a list of Application.
type ApplicationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Application `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Application{}, &ApplicationList{})
}
