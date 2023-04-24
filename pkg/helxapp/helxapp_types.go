package helxapp

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Service represents a service to be created within the HeLxApp
type Service struct {
	Name        string            `json:"name"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// HeLxAppSpec defines the desired state of HeLxApp
type HeLxAppSpec struct {
	Replicas int32     `json:"replicas"`
	Services []Service `json:"services"`
	Version  string    `json:"version"`
}

// HeLxAppStatus defines the observed state of HeLxApp
type HeLxAppStatus struct {
	// Add your fields here, e.g., current service configurations.
}

// HeLxApp is the Schema for the helxapps API
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:object:generate=true
type HeLxApp struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   HeLxAppSpec   `json:"spec,omitempty"`
	Status HeLxAppStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// HeLxAppList contains a list of HeLxApp
type HeLxAppList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []HeLxApp `json:"items"`
}
