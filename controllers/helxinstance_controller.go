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

package controllers

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	helxv1 "github.com/helxplatform/helxapp/api/v1"
	"github.com/helxplatform/helxapp/helxapp_operations"
)

// HelxInstanceReconciler reconciles a HelxInstance object
type HelxInstanceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

type System struct {
	Name                string
	AMB                 bool
	SystemEnv           []EnvVar
	Username            string
	SystemName          string
	Host                string
	Identifier          string
	AppID               string
	EnableInitContainer bool
	CreateHomeDirs      bool
	DevPhase            string
	SecurityContext     SecurityContext
	Containers          []Container
}

type SecurityContext struct {
	RunAsUser  int
	RunAsGroup int
	FsGroup    int
}

type Container struct {
	Name           string
	Image          string
	Command        []string
	Env            []EnvVar
	Ports          []Port
	Expose         []Port
	Resources      ResourceRequirements
	VolumeMounts   []VolumeMount
	LivenessProbe  *Probe
	ReadinessProbe *Probe
}

type EnvVar struct {
	Name  string
	Value string
}

type Port struct {
	ContainerPort int
	Protocol      string
}

type ResourceRequirements struct {
	Limits   ResourceList
	Requests ResourceList
}

type ResourceList struct {
	CPU    string
	Memory string
	GPU    string
}

type VolumeMount struct {
	Name      string
	MountPath string
	SubPath   string
	ReadOnly  bool
}

type Probe struct {
	Exec                *ExecAction
	HTTPGet             *HTTPGetAction
	TCPSocket           *TCPSocketAction
	InitialDelaySeconds int32
	PeriodSeconds       int32
	FailureThreshold    int32
}

type ExecAction struct {
	Command []string
}

type HTTPGetAction struct {
	Path        string
	Port        int32
	Scheme      string
	HttpHeaders map[string]string
}

type TCPSocketAction struct {
	Port int32
}

var initialized bool = false

//+kubebuilder:rbac:groups=helx.renci.org,namespace=jeffw,resources=helxinstances,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=helx.renci.org,namespace=jeffw,resources=helxinstances/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=helx.renci.org,namespace=jeffw,resources=helxinstances/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the HelxInstance object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.4/pkg/reconcile
func (r *HelxInstanceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Fetch the HelxInstance custom resource
	helxInstance := &helxv1.HelxInstance{}
	if err := r.Get(ctx, req.NamespacedName, helxInstance); err != nil {
		if errors.IsNotFound(err) {
			// Resource is already deleted, return without error
			logger.Info("HelxInstance deleted, nothing to reconcile", "NamespacedName", req.NamespacedName)
			return ctrl.Result{}, nil
		}
		logger.Error(err, "unable to fetch HelxInstance", "NamespacedName", req.NamespacedName)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Log the event and custom resource content
	logger.Info("Reconciling HelxInstance", "HelxInstance", fmt.Sprintf("%+v", helxInstance))
	if err := helxapp_operations.CheckInit(ctx); err == nil {
		deploymentYAML := helxapp_operations.CreateDeploymentString(&helxInstance.Spec)
		if deploymentYAML != "" {
			logger.Info("generated YAML:")
			logger.Info(deploymentYAML)
			if err = helxapp_operations.CreateDeploymentFromYAML(ctx, r.Client, r.Scheme, req, helxInstance, deploymentYAML); err != nil {
				logger.Error(err, "unable to create deployment", "NamespacedName", req.NamespacedName)
			}
		}
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *HelxInstanceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&helxv1.HelxInstance{}).
		Complete(r)
}
