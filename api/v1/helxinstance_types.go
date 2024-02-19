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

// HelxInstanceSpec defines the desired state of HelxInstance
type HelxInstanceSpec struct {
	Identifier               string               `json:"identifier,omitempty"`
	AppName                  string               `json:"appName,omitempty"`
	AmbassadorId             string               `json:"ambassadorId,omitempty"`
	Resources                map[string]Resources `json:"resources,omitempty"`
	Username                 string               `json:"username,omitempty"`
	Namespace                string               `json:"namespace,omitempty"`
	Serviceaccount           string               `json:"serviceaccount,omitempty"`
	RunAsUser                int                  `json:"runAsUser,omitempty"`
	RunAsGroup               int                  `json:"runAsGroup,omitempty"`
	FsGroup                  int                  `json:"fsGroup,omitempty"`
	SupplementalGroups       []int                `json:"supplementalGroups,omitempty"`
	Privileged               bool                 `json:"privileged,omitempty"`
	AllowPrivilegeEscalation bool                 `json:"allowPrivilegeEscalation,omitempty"`
}

// ServicePort represents a single port for a service in a HeLxApp
type Resources struct {
	Requests map[string]string `json:"request,omitempty"`
	Limits   map[string]string `json:"limit,omitempty"`
}

// HelxInstanceStatus defines the observed state of HelxInstance
type HelxInstanceStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// HelxInstance is the Schema for the helxinstances API
type HelxInstance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   HelxInstanceSpec   `json:"spec,omitempty"`
	Status HelxInstanceStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// HelxInstanceList contains a list of HelxInstance
type HelxInstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []HelxInstance `json:"items"`
}

func init() {
	SchemeBuilder.Register(&HelxInstance{}, &HelxInstanceList{})
}
