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
		Name:           "test-system",
		Username:       "test-username",
		AppName:        "test-system-name",
		Host:           "test-system-host",
		Identifier:     "test-identifier",
		CreateHomeDirs: true,
		DevPhase:       "test",
		SecurityContext: SecurityContext{
			RunAsUser:  1000,
			RunAsGroup: 1000,
			FsGroup:    1000,
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
		Name:           "nginx",
		Username:       "jeffw",
		AppName:        "nginx",
		Host:           "host1",
		Identifier:     "nginx-1",
		CreateHomeDirs: false,
		DevPhase:       "test",
		SecurityContext: SecurityContext{
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
