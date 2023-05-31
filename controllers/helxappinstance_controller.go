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
	"text/template"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/google/uuid"
	helxv1 "github.com/helxplatform/helxapp/api/v1"
	"github.com/helxplatform/helxapp/template_io"
)

// HelxAppInstanceReconciler reconciles a HelxAppInstance object
type HelxAppInstanceReconciler struct {
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
var xformer *template.Template
var clientset *kubernetes.Clientset

func Init() {
	var err error
	var config *rest.Config

	xformer, err = template_io.ParseTemplates("../templates")
	if err != nil {
		fmt.Print("failed to initialize xformer template: %v", err)
		return
	}
	config, err = rest.InClusterConfig()
	if err != nil {
		fmt.Print("failed to k8s client: %v", err)
		return
	}
	clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		fmt.Print("failed to k8s client: %v", err)
		return
	}
	initialized = true
}

func getDeploymentString(appname string, app helxv1.HelxApp, instance helxv1.HelxAppInstance) string {
	uuid := uuid.New()
	id := uuid.String()
	containers := []template_io.Container{}

	for i := 0; i < len(app.Spec.Services); i++ {
		ports := []template_io.Port{}

		for j := 0; j < len(app.Spec.Services[i].Ports); j++ {
			srcPort := app.Spec.Services[i].Ports[j]
			ports = append(ports, template_io.Port{ContainerPort: int(srcPort.ContainerPort), Protocol: "TCP"})
		}
		c := template_io.Container{
			Name:         app.Spec.Services[i].Name,
			Image:        app.Spec.Services[i].Image,
			Ports:        ports,
			Expose:       ports,
			VolumeMounts: []template_io.VolumeMount{},
		}
		containers = append(containers, c)
	}

	system := template_io.System{
		Name:           appname,
		Username:       "jeffw",
		SystemName:     appname,
		Host:           "",
		Identifier:     appname + "-" + id,
		AppID:          appname + "-" + id,
		CreateHomeDirs: false,
		DevPhase:       "test",
		Containers:     containers,
	}

	vars := make(map[string]interface{})
	vars["system"] = system

	// Call the function.
	result, err := template_io.RenderGoTemplate(xformer, "deployment", vars)
	if err != nil {
		fmt.Print("RenderGoTemplate() error = %v", err)
		return ""
	}
	return result
}

//+kubebuilder:rbac:groups=helx.renci.org,namespace=jeffw,resources=helxappinstances,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=helx.renci.org,namespace=jeffw,resources=helxappinstances/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=helx.renci.org,namespace=jeffw,resources=helxappinstances/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the HelxAppInstance object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.4/pkg/reconcile
func (r *HelxAppInstanceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Fetch the HelxAppInstance custom resource
	helxAppInstance := &helxv1.HelxAppInstance{}
	err := r.Get(ctx, req.NamespacedName, helxAppInstance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Resource is already deleted, return without error
			logger.Info("HelxAppInstance deleted, nothing to reconcile", "NamespacedName", req.NamespacedName)
			return ctrl.Result{}, nil
		}
		logger.Error(err, "unable to fetch HelxAppInstance", "NamespacedName", req.NamespacedName)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Log the event and custom resource content
	logger.Info("Reconciling HelxAppInstance", "HelxAppInstance", fmt.Sprintf("%+v", helxAppInstance))
	if !initialized {
		Init()
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *HelxAppInstanceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&helxv1.HelxAppInstance{}).
		Complete(r)
}
