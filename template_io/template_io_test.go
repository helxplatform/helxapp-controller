package template_io

import (
	"testing"

	"github.com/CloudyKit/jet/v6"
)

func TestInitJetTemplate(t *testing.T) {
	//err := InitJetTemplate("../templates", "container-spec.jet")
	err := InitJetTemplate("../templates", "container-spec.jet")
	if err != nil {
		t.Errorf("failed to initialize Jet template: %v", err)
	}
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
	SecurityContext     *SecurityContext
	Containers          []Container
}

type SecurityContext struct {
	RunAsUser  int
	RunAsGroup int
	FsGroup    int
}

type Container struct {
	Name            string
	Image           string
	Command         []string
	Env             []EnvVar
	Ports           []Port
	Expose          []Port
	Resources       ResourceRequirements
	VolumeMounts    []VolumeMount
	SecurityContext *SecurityContext
	LivenessProbe   *Probe
	ReadinessProbe  *Probe
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

func TestRenderJetTemplate(t *testing.T) {
	// Initialize a VarMap with the necessary values.
	// Prepare the data for the template
	system := System{
		Name:                "test-system",
		Username:            "test-username",
		SystemName:          "test-system-name",
		Host:                "test-system-host",
		Identifier:          "test-identifier",
		AppID:               "test-app-id",
		EnableInitContainer: true,
		CreateHomeDirs:      true,
		DevPhase:            "test",
		SecurityContext: &SecurityContext{
			RunAsUser:  1000,
			RunAsGroup: 1000,
			FsGroup:    1000,
		},
		Containers: []Container{
			{
				Name:    "test-container",
				Image:   "test-image",
				Command: []string{"echo", "Hello, World!"},
				Env: []EnvVar{
					{
						Name:  "TEST_ENV_VAR",
						Value: "test",
					},
				},
				Ports: []Port{
					{
						ContainerPort: 8080,
						Protocol:      "TCP",
					},
				},
				Resources: ResourceRequirements{
					Limits: ResourceList{
						CPU:    "1",
						Memory: "1Gi",
						GPU:    "1",
					},
					Requests: ResourceList{
						CPU:    "0.5",
						Memory: "500Mi",
						GPU:    "1",
					},
				},
				VolumeMounts: []VolumeMount{
					{
						Name:      "test-volume",
						MountPath: "/test/path",
						SubPath:   "test/subpath",
						ReadOnly:  false,
					},
				},
				LivenessProbe: &Probe{
					Exec: &ExecAction{
						Command: []string{"echo", "liveness probe"},
					},
					InitialDelaySeconds: 10,
					PeriodSeconds:       20,
					FailureThreshold:    3,
				},
				ReadinessProbe: &Probe{
					HTTPGet: &HTTPGetAction{
						Path:   "readiness/probe",
						Port:   8080,
						Scheme: "HTTP",
					},
					InitialDelaySeconds: 5,
					PeriodSeconds:       10,
					FailureThreshold:    3,
				},
			},
		},
	}

	vars := make(jet.VarMap)
	vars.Set("system", system)

	// Call the function.
	result, err := RenderJetTemplate(vars)

	// Check for errors.
	if err != nil {
		t.Errorf("RenderJetTemplate() error = %v", err)
		return
	}
	t.Log(result)
}

func TestRenderNginx(t *testing.T) {
	// Initialize a VarMap with the necessary values.
	// Prepare the data for the template
	system := System{
		Name:                "nginx",
		Username:            "jeffw",
		SystemName:          "nginx",
		Host:                "host1",
		Identifier:          "nginx-1",
		AppID:               "nginx-app-id",
		EnableInitContainer: false,
		CreateHomeDirs:      false,
		DevPhase:            "test",
		SecurityContext: &SecurityContext{
			RunAsUser:  0,
			RunAsGroup: 0,
			FsGroup:    0,
		},
		Containers: []Container{
			{
				Name:    "nginx-test",
				Image:   "nginx:latest",
				Command: []string{},
				Env: []EnvVar{
					{
						Name:  "TEST_ENV_VAR",
						Value: "test",
					},
				},
				Ports: []Port{
					{
						ContainerPort: 80,
						Protocol:      "TCP",
					},
				},
				Expose: []Port{
					{
						ContainerPort: 80,
						Protocol:      "TCP",
					},
				},
				Resources: ResourceRequirements{
					Limits: ResourceList{
						CPU:    "1",
						Memory: "1Gi",
						GPU:    "0",
					},
					Requests: ResourceList{
						CPU:    "0.5",
						Memory: "500Mi",
						GPU:    "0",
					},
				},
				VolumeMounts: []VolumeMount{},
			},
		},
	}

	vars := make(jet.VarMap)
	vars.Set("system", system)

	// Call the function.
	result, err := RenderJetTemplate(vars)

	// Check for errors.
	if err != nil {
		t.Errorf("RenderJetTemplate() error = %v", err)
		return
	}
	t.Log("\n" + result)
}
