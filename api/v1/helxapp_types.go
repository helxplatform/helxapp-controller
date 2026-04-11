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
	AppClassName string    `json:"appClassName,omitempty"`
	SourceText   string    `json:"sourceText,omitempty"`
	Services     []Service `json:"services"`
}

// AmbassadorMapping configures an Ambassador routing annotation on the Service.
type AmbassadorMapping struct {
	// AmbassadorID restricts this mapping to a specific Ambassador instance.
	AmbassadorID string `json:"ambassadorId,omitempty"`
	// Prefix is the URL path prefix. Supports Go template expressions.
	// Default: /private/<AppClassName>/<UserName>/<UUID>/
	Prefix string `json:"prefix,omitempty"`
	// ProxyRewrite rewrites the upstream path. If empty, no rewrite is applied.
	ProxyRewrite string `json:"proxyRewrite,omitempty"`
}

// Service represents a single service in a HeLxApp
type Service struct {
	Name            string                      `json:"name"`
	Image           string                      `json:"image"`
	Command         []string                    `json:"command,omitempty"`
	Environment     map[string]string           `json:"environment,omitempty"`
	SecretsFrom     []string                    `json:"secretsFrom,omitempty"`
	ConfigMapsFrom  []string                    `json:"configMapsFrom,omitempty"`
	Init            bool                        `json:"init,omitempty"`
	Ports           []PortMap                   `json:"ports,omitempty"`
	ResourceBounds  map[string]ResourceBoundary `json:"resourceBounds,omitempty"`
	SecurityContext *SecurityContext            `json:"securityContext,omitempty"`
	Volumes         map[string]string           `json:"volumes,omitempty"`
	Ambassador      *AmbassadorMapping          `json:"ambassador,omitempty"`
	LivenessProbe   *Probe                      `json:"livenessProbe,omitempty"`
	ReadinessProbe  *Probe                      `json:"readinessProbe,omitempty"`
}

// Probe describes a health check to be performed against a container.
type Probe struct {
	// One of Exec, HTTPGet, or TCPSocket must be specified.
	Exec      *ExecAction      `json:"exec,omitempty"`
	HTTPGet   *HTTPGetAction   `json:"httpGet,omitempty"`
	TCPSocket *TCPSocketAction `json:"tcpSocket,omitempty"`
	// Timing fields.
	InitialDelaySeconds int32 `json:"initialDelaySeconds,omitempty"`
	PeriodSeconds       int32 `json:"periodSeconds,omitempty"`
	FailureThreshold    int32 `json:"failureThreshold,omitempty"`
}

// ExecAction describes a "run in container" action.
type ExecAction struct {
	Command []string `json:"command"`
}

// HTTPGetAction describes an HTTP GET request health check.
type HTTPGetAction struct {
	Path        string            `json:"path"`
	Port        int32             `json:"port"`
	Scheme      string            `json:"scheme,omitempty"`
	HTTPHeaders map[string]string `json:"httpHeaders,omitempty"`
}

// TCPSocketAction describes a TCP port health check.
type TCPSocketAction struct {
	Port int32 `json:"port"`
}

// ServicePort represents a single port for a service in a HeLxApp
type ResourceBoundary struct {
	Min string `json:"min,omitempty"`
	Max string `json:"max,omitempty"`
}

// ServicePort represents a single port for a service in a HeLxApp
type PortMap struct {
	ContainerPort int32 `json:"containerPort"`
	Port          int32 `json:"port,omitempty"`
}

type SecurityContext struct {
	RunAsUser          *int64  `json:"runAsUser,omitempty"`
	RunAsGroup         *int64  `json:"runAsGroup,omitempty"`
	FSGroup            *int64  `json:"fsGroup,omitempty"`
	SupplementalGroups []int64 `json:"supplementalGroups,omitempty"`
}

// HelxAppStatus defines the observed state of HelxApp
type HelxAppStatus struct {
	ObservedGeneration int64 `json:"observedGeneration"`
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
