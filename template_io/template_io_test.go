package template_io

import (
	"fmt"
	"strings"
	"testing"
	"text/template"

	helxv1 "github.com/helxplatform/helxapp-controller/api/v1"
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
		UserName:     "test-username",
		Host:         "test-system-host",
		UUID:         "xxx-xxx-xxx-xxx",
		SecurityContext: &SecurityContext{
			RunAsUser:  "1000",
			RunAsGroup: "1000",
			FSGroup:    "1000",
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
		UserName:     "jeffw",
		Host:         "host1",
		UUID:         "xxx-xxx-xxx-xxx",
		SecurityContext: &SecurityContext{
			RunAsUser:  "0",
			RunAsGroup: "0",
			FSGroup:    "0",
		},
		Containers: []Container{
			{
				Name:    "nginx-test",
				Image:   Image{ImageName: "nginx:latest", Attr: map[string]string{}},
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

func ensureTemplates(t *testing.T) {
	t.Helper()
	if testTemplate == nil {
		var err error
		testTemplate, storage, err = ParseTemplates("../templates", nil)
		if err != nil {
			t.Fatalf("failed to parse templates: %v", err)
		}
	}
}

// 53. TestExtractSCFromCR_Full - All fields populated
func TestExtractSCFromCR_Full(t *testing.T) {
	user := int64(1000)
	group := int64(1000)
	fs := int64(2000)
	sc := &helxv1.SecurityContext{
		RunAsUser:          &user,
		RunAsGroup:         &group,
		FSGroup:            &fs,
		SupplementalGroups: []int64{100, 200},
	}
	result := ExtractSCFromCR(sc)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.RunAsUser != "1000" {
		t.Errorf("RunAsUser = %q, want %q", result.RunAsUser, "1000")
	}
	if result.RunAsGroup != "1000" {
		t.Errorf("RunAsGroup = %q, want %q", result.RunAsGroup, "1000")
	}
	if result.FSGroup != "2000" {
		t.Errorf("FSGroup = %q, want %q", result.FSGroup, "2000")
	}
	if len(result.SupplementalGroups) != 2 {
		t.Fatalf("SupplementalGroups length = %d, want 2", len(result.SupplementalGroups))
	}
	if result.SupplementalGroups[0] != "100" || result.SupplementalGroups[1] != "200" {
		t.Errorf("SupplementalGroups = %v, want [100 200]", result.SupplementalGroups)
	}
}

// 54. TestExtractSCFromCR_Partial - Only RunAsUser set
func TestExtractSCFromCR_Partial(t *testing.T) {
	user := int64(500)
	sc := &helxv1.SecurityContext{
		RunAsUser: &user,
	}
	result := ExtractSCFromCR(sc)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.RunAsUser != "500" {
		t.Errorf("RunAsUser = %q, want %q", result.RunAsUser, "500")
	}
	if result.RunAsGroup != "" {
		t.Errorf("RunAsGroup = %q, want empty", result.RunAsGroup)
	}
	if result.FSGroup != "" {
		t.Errorf("FSGroup = %q, want empty", result.FSGroup)
	}
	if len(result.SupplementalGroups) != 0 {
		t.Errorf("SupplementalGroups = %v, want empty", result.SupplementalGroups)
	}
}

// 55. TestExtractSCFromCR_Nil - nil input returns nil
func TestExtractSCFromCR_Nil(t *testing.T) {
	result := ExtractSCFromCR(nil)
	if result != nil {
		t.Errorf("expected nil, got %+v", result)
	}
}

// 56. TestExtractSCFromCR_Empty - empty struct returns nil
func TestExtractSCFromCR_Empty(t *testing.T) {
	sc := &helxv1.SecurityContext{}
	result := ExtractSCFromCR(sc)
	if result != nil {
		t.Errorf("expected nil for empty SecurityContext, got %+v", result)
	}
}

// 57. TestExtractSCFromMap_Full - map with all fields
func TestExtractSCFromMap_Full(t *testing.T) {
	data := map[string]interface{}{
		"runAsUser":          "1000",
		"runAsGroup":         "1000",
		"fsGroup":            "2000",
		"supplementalGroups": []interface{}{"100", "200"},
	}
	result := ExtractSCFromMap(data)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.RunAsUser != "1000" {
		t.Errorf("RunAsUser = %q, want %q", result.RunAsUser, "1000")
	}
	if result.RunAsGroup != "1000" {
		t.Errorf("RunAsGroup = %q, want %q", result.RunAsGroup, "1000")
	}
	if result.FSGroup != "2000" {
		t.Errorf("FSGroup = %q, want %q", result.FSGroup, "2000")
	}
	if len(result.SupplementalGroups) != 2 {
		t.Fatalf("SupplementalGroups length = %d, want 2", len(result.SupplementalGroups))
	}
	if result.SupplementalGroups[0] != "100" || result.SupplementalGroups[1] != "200" {
		t.Errorf("SupplementalGroups = %v, want [100 200]", result.SupplementalGroups)
	}
}

// 58. TestExtractSCFromMap_Empty - empty map returns nil
func TestExtractSCFromMap_Empty(t *testing.T) {
	data := map[string]interface{}{}
	result := ExtractSCFromMap(data)
	if result != nil {
		t.Errorf("expected nil for empty map, got %+v", result)
	}
}

// 59. TestReRender_Stable - plain text without template expressions
func TestReRender_Stable(t *testing.T) {
	input := "hello world, no templates here"
	ctx := map[string]interface{}{}
	output, err := ReRender(input, ctx)
	if err != nil {
		t.Fatalf("ReRender error: %v", err)
	}
	if output != input {
		t.Errorf("output = %q, want %q", output, input)
	}
}

// 60. TestReRender_WithVariables - text with template variables
func TestReRender_WithVariables(t *testing.T) {
	input := "Hello {{.system.UserName}}!"
	ctx := map[string]interface{}{
		"system": map[string]interface{}{
			"UserName": "testuser",
		},
	}
	output, err := ReRender(input, ctx)
	if err != nil {
		t.Fatalf("ReRender error: %v", err)
	}
	expected := "Hello testuser!"
	if output != expected {
		t.Errorf("output = %q, want %q", output, expected)
	}
}

// 61. TestRenderPVCTemplate - Render PVC template with various access modes
func TestRenderPVCTemplate(t *testing.T) {
	ensureTemplates(t)

	system := System{
		UUID: "test-uuid-pvc",
	}

	cases := []struct {
		name       string
		attr       map[string]string
		expectMode string
	}{
		{"rwx", map[string]string{"claim": "test-claim", "rwx": "true"}, "ReadWriteMany"},
		{"rox", map[string]string{"claim": "test-claim", "rox": "true"}, "ReadOnlyMany"},
		{"rwop", map[string]string{"claim": "test-claim", "rwop": "true"}, "ReadWriteOncePod"},
		{"default", map[string]string{"claim": "test-claim"}, "ReadWriteOnce"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			vol := Volume{
				Name:   "test-vol",
				Scheme: "pvc",
				Attr:   tc.attr,
			}
			vars := map[string]interface{}{
				"system": system,
				"volume": vol,
			}
			result, err := RenderGoTemplate(testTemplate, "pvc", vars)
			if err != nil {
				t.Fatalf("RenderGoTemplate error: %v", err)
			}
			if !strings.Contains(result, tc.expectMode) {
				t.Errorf("expected output to contain %q, got:\n%s", tc.expectMode, result)
			}
		})
	}
}

// 62. TestRenderServiceTemplate - Render service template with HasService=true
func TestRenderServiceTemplate(t *testing.T) {
	ensureTemplates(t)

	system := System{
		AppName:      "test-app",
		InstanceName: "test-instance",
		UUID:         "test-uuid-svc",
	}
	container := Container{
		HasService: true,
		Ports: []PortMap{
			{ContainerPort: 8080, Port: 80, Protocol: "TCP"},
		},
	}
	vars := map[string]interface{}{
		"system":    system,
		"container": container,
	}
	result, err := RenderGoTemplate(testTemplate, "service", vars)
	if err != nil {
		t.Fatalf("RenderGoTemplate error: %v", err)
	}
	if !strings.Contains(result, "port: 80") {
		t.Errorf("expected output to contain port mapping, got:\n%s", result)
	}
	if !strings.Contains(result, "targetPort: 8080") {
		t.Errorf("expected output to contain targetPort mapping, got:\n%s", result)
	}
}

// 63. TestRenderServiceTemplate_NoService - Render service template with HasService=false
func TestRenderServiceTemplate_NoService(t *testing.T) {
	ensureTemplates(t)

	system := System{
		AppName:      "test-app",
		InstanceName: "test-instance",
		UUID:         "test-uuid-nosvc",
	}
	container := Container{
		HasService: false,
	}
	vars := map[string]interface{}{
		"system":    system,
		"container": container,
	}
	result, err := RenderGoTemplate(testTemplate, "service", vars)
	if err != nil {
		t.Fatalf("RenderGoTemplate error: %v", err)
	}
	trimmed := strings.TrimSpace(result)
	if strings.Contains(trimmed, "kind: Service") {
		t.Errorf("expected no Service output when HasService=false, got:\n%s", result)
	}
}

// 64. TestRenderDeployment_WithInitContainers - System with InitContainers populated
func TestRenderDeployment_WithInitContainers(t *testing.T) {
	ensureTemplates(t)

	system := System{
		AppName:      "test-app",
		AppClassName: "test-class",
		InstanceName: "test-instance",
		UserName:     "testuser",
		UUID:         "test-uuid-init",
		Host:         "test-host",
		SecurityContext: &SecurityContext{
			RunAsUser:  "1000",
			RunAsGroup: "1000",
			FSGroup:    "1000",
		},
		Containers: []Container{
			{
				Name:  "main",
				Image: Image{ImageName: "nginx:latest", Attr: map[string]string{}},
			},
		},
		InitContainers: []Container{
			{
				Name:  "init-setup",
				Image: Image{ImageName: "busybox:latest", Attr: map[string]string{}},
			},
		},
		Volumes: map[string]Volume{},
	}

	vars := map[string]interface{}{
		"system": system,
	}
	result, err := RenderGoTemplate(testTemplate, "deployment", vars)
	if err != nil {
		t.Fatalf("RenderGoTemplate error: %v", err)
	}
	if !strings.Contains(result, "initContainers") {
		t.Errorf("expected output to contain 'initContainers', got:\n%s", result)
	}
}

// 65. TestRenderDeployment_Labels - Render deployment, check helx.renci.org labels
func TestRenderDeployment_Labels(t *testing.T) {
	ensureTemplates(t)

	system := System{
		AppName:      "label-app",
		AppClassName: "label-class",
		InstanceName: "label-instance",
		UserName:     "labeluser",
		UUID:         "test-uuid-labels",
		Host:         "test-host",
		SecurityContext: &SecurityContext{
			RunAsUser:  "1000",
			RunAsGroup: "1000",
			FSGroup:    "1000",
		},
		Containers: []Container{},
		Volumes:    map[string]Volume{},
	}

	vars := map[string]interface{}{
		"system": system,
	}
	result, err := RenderGoTemplate(testTemplate, "deployment", vars)
	if err != nil {
		t.Fatalf("RenderGoTemplate error: %v", err)
	}
	if !strings.Contains(result, "helx.renci.org/id") {
		t.Errorf("expected output to contain 'helx.renci.org/id' label, got:\n%s", result)
	}
	if !strings.Contains(result, "helx.renci.org/app-name") {
		t.Errorf("expected output to contain 'helx.renci.org/app-name' label, got:\n%s", result)
	}
	if !strings.Contains(result, "helx.renci.org/username") {
		t.Errorf("expected output to contain 'helx.renci.org/username' label, got:\n%s", result)
	}
}
