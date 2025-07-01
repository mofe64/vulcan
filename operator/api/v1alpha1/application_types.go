package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ApplicationSpec struct {
	// Git repository to build & deploy.
	// +kubebuilder:validation:Format=uri
	RepoURL string `json:"repoURL"`

	// Build is either buildpack or dockerfile.
	Build BuildConfig `json:"build"`

	// Runtime environment variables (key=value)
	Env []EnvVar `json:"env,omitempty"`

	// Autoscaling policy (passed to HPA)
	Autoscaling HPAPolicy `json:"autoscaling,omitempty"`

	// ProjectRef is the reference to the project that the application belongs to.
	ProjectRef string `json:"projectRef"`

	// OrgRef is the reference to the name of the organization that the application belongs to.
	OrgRef string `json:"orgRef"`
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

type ApplicationStatus struct {
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
