package helxapp

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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

// HeLxAppSpec defines the desired state of HeLxApp
// +k8s:deepcopy-gen=true
type HeLxAppSpec struct {
	Services []Service `json:"services"`
}

// Service represents a single service in a HeLxApp
// +k8s:deepcopy-gen=true
type Service struct {
	Name        string            `json:"name"`
	Image       string            `json:"image"`
	Ports       []ServicePort     `json:"ports"`
	Environment map[string]string `json:"environment"`
	Volumes     []VolumeMount     `json:"volumes"`
	Replicas    int32             `json:"replicas"`
}

// ServicePort represents a single port for a service in a HeLxApp
// +k8s:deepcopy-gen=true
type ServicePort struct {
	ContainerPort int32 `json:"containerPort"`
	HostPort      int32 `json:"hostPort,omitempty"`
}

// VolumeMount represents a single volume mount for a service in a HeLxApp
// +k8s:deepcopy-gen=true
type VolumeMount struct {
	MountPath string `json:"mountPath"`
	HostPath  string `json:"hostPath"`
}

// HeLxAppStatus defines the observed state of HeLxApp
// +k8s:deepcopy-gen=true
type HeLxAppStatus struct {
	// Add your fields here, e.g., current service configurations.
}

// HeLxAppList contains a list of HeLxApp
// +kubebuilder:object:root=true
// +kubebuilder:object:generate=true
type HeLxAppList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []HeLxApp `json:"items"`
}
