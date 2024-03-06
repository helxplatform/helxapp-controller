package template_io

import (
	"fmt"
	"testing"
	"text/template"
)

var testTemplate *template.Template
var storage map[string][]string

func TestInitGoTemplate(t *testing.T) {
	var err error

	testTemplate, storage, err = ParseTemplates("../templates", nil)
	if err != nil {
		t.Errorf("failed to initialize Go template: %v", err)
	}
}

func TestRenderGoTemplate(t *testing.T) {
	system := System{
		AppName:      "test-app",
		AppClassName: "test-system",
		InstanceName: "test-instance",
		Username:     "test-username",
		Host:         "test-system-host",
		GUID:         "xxx-xxx-xxx-xxx",
		SecurityContext: &SecurityContext{
			RunAsUser:  "1000",
			RunAsGroup: "1000",
			FsGroup:    "1000",
		},
		Containers: []Container{
			// similar to previous test case
		},
	}

	vars := make(map[string]interface{})
	vars["system"] = system

	result, err := RenderGoTemplate(testTemplate, "deployment", vars)

	if err != nil {
		t.Errorf("RenderGoTemplate() error = %v", err)
		return
	}
	t.Log(result)
}

func TestRenderNginx(t *testing.T) {
	system := System{
		AppClassName: "nginx",
		AppName:      "nginx",
		InstanceName: "inst1",
		Username:     "jeffw",
		Host:         "host1",
		GUID:         "xxx-xxx-xxx-xxx",
		SecurityContext: &SecurityContext{
			RunAsUser:  "0",
			RunAsGroup: "0",
			FsGroup:    "0",
		},
		Containers: []Container{
			{
				Name:    "nginx-test",
				Image:   "nginx:latest",
				Command: []string{},
				Environment: map[string]string{
					"TEST_ENV_VAR": "test",
				},
				Ports: []PortMap{
					{
						ContainerPort: 80,
						Port:          80,
						Protocol:      "TCP",
					},
				},
				Resources: Resources{
					Limits: map[string]string{
						"cpu":    "1",
						"memory": "1Gi",
					},
					Requests: map[string]string{
						"cpu":    "0.5",
						"memory": "500Mi",
					},
				},
				VolumeMounts: []*VolumeMount{
					{
						Name:      "v1",
						MountPath: "/mnt/v1",
						ReadOnly:  false,
					},
					{
						Name:      "v2",
						MountPath: "/mnt/v2",
						ReadOnly:  false,
					},
				},
			},
		},
		Volumes: map[string]Volume{
			"v1": {
				Name:   "v1",
				Scheme: "pvc",
				Attr: map[string]string{
					"claim": "pvcsrc",
				},
			},
			"v2": {
				Name:   "v2",
				Scheme: "nfs",
				Attr: map[string]string{
					"server": "s",
					"path":   "/x/y",
				},
			},
		},
	}

	vars := make(map[string]interface{})
	vars["system"] = system

	result, err := RenderGoTemplate(testTemplate, "deployment", vars)

	if err != nil {
		t.Errorf("RenderGoTemplate() error = %v", err)
		return
	}
	t.Log("\n" + result)
	fmt.Printf("%s", result)
}
