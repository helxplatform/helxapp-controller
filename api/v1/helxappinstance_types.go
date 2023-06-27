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

// HelxAppInstanceSpec defines the desired state of HelxAppInstance
type HelxAppInstanceSpec struct {
	Identifier               string `json:"identifier,omitempty"`
	AppName                  string `json:"appName,omitempty"`
	Name                     string `json:"name,omitempty"`
	SourceText               string `json:"sourceText,omitempty"`
	AmbassadorId             string `json:"ambassadorId,omitempty"`
	Username                 string `json:"username,omitempty"`
	Host                     string `json:"host,omitempty"`
	Namespace                string `json:"namespace,omitempty"`
	Serviceaccount           string `json:"serviceaccount,omitempty"`
	ConnString               string `json:"connString,omitempty"`
	GiteaIntegration         bool   `json:"giteaIntegration,omitempty"`
	GiteaHost                string `json:"giteaHost,omitempty"`
	GiteaUser                string `json:"giteaUser,omitempty"`
	GiteaServiceName         string `json:"giteaServiceName,omitempty"`
	RunLevel                 int    `json:"runLevel,omitempty"`
	RunAsUser                int    `json:"runAsUser,omitempty"`
	RunAsGroup               int    `json:"runAsGroup,omitempty"`
	FsGroup                  int    `json:"fsGroup,omitempty"`
	SupplementalGroups       []int  `json:"supplementalGroups,omitempty"`
	Privileged               bool   `json:"privileged,omitempty"`
	AllowPrivilegeEscalation bool   `json:"allowPrivilegeEscalation,omitempty"`
}

// HelxAppInstanceStatus defines the observed state of HelxAppInstance
type HelxAppInstanceStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// HelxAppInstance is the Schema for the helxappinstances API
type HelxAppInstance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   HelxAppInstanceSpec   `json:"spec,omitempty"`
	Status HelxAppInstanceStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// HelxAppInstanceList contains a list of HelxAppInstance
type HelxAppInstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []HelxAppInstance `json:"items"`
}

func init() {
	SchemeBuilder.Register(&HelxAppInstance{}, &HelxAppInstanceList{})
}
