/*
Copyright 2023.

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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// HelxAppSpec defines the desired state of HelxApp
type HelxAppSpec struct {
	Name     string    `json:"name"`
	Services []Service `json:"services"`
}

// Service represents a single service in a HeLxApp
type Service struct {
	Name        string            `json:"name"`
	Image       string            `json:"image"`
	Ports       []ServicePort     `json:"ports,omitempty"`
	Environment map[string]string `json:"environment,omitempty"`
	Volumes     []VolumeMount     `json:"volumes,omitempty"`
	Replicas    int32             `json:"replicas,omitempty"`
}

// ServicePort represents a single port for a service in a HeLxApp
type ServicePort struct {
	ContainerPort int32 `json:"containerPort"`
	HostPort      int32 `json:"hostPort,omitempty"`
}

// VolumeMount represents a single volume mount for a service in a HeLxApp
type VolumeMount struct {
	MountPath string `json:"mountPath"`
	HostPath  string `json:"hostPath"`
}

// HelxAppStatus defines the observed state of HelxApp
type HelxAppStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// HelxApp is the Schema for the helxapps API
type HelxApp struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   HelxAppSpec   `json:"spec,omitempty"`
	Status HelxAppStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// HelxAppList contains a list of HelxApp
type HelxAppList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []HelxApp `json:"items"`
}

func init() {
	SchemeBuilder.Register(&HelxApp{}, &HelxAppList{})
}
