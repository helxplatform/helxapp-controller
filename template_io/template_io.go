package template_io

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig"
	"github.com/jackc/pgx/v4"
)

type System struct {
	Name            string
	AMB             bool
	SystemEnv       []EnvVar
	Username        string
	AppName         string
	Host            string
	Identifier      string
	CreateHomeDirs  bool
	DevPhase        string
	SecurityContext SecurityContext
	Containers      []Container
	InitContainers  []Container
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
	SecurityContext SecurityContext
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
	Limits   *ResourceList
	Requests *ResourceList
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

func RenderTemplateToString(tmpl *template.Template, name string, data interface{}) string {
	buf := new(bytes.Buffer)
	err := tmpl.ExecuteTemplate(buf, name, data)
	if err != nil {
		panic(err)
	}
	return buf.String()
}

func HasGPU(containers []Container) bool {
	for _, container := range containers {
		limits := container.Resources.Limits
		requests := container.Resources.Requests
		if limits != nil && limits.GPU != "0" || requests != nil && requests.GPU != "0" {
			return true
		}
	}
	return false
}

func ParseTemplates(dir string) (*template.Template, error) {
	// Get a list of all .tmpl files in the directory
	files, err := filepath.Glob(filepath.Join(dir, "*.tmpl"))
	if err != nil {
		return nil, err
	}

	// No templates in the directory
	if len(files) == 0 {
		return nil, nil
	}

	var tmpl *template.Template

	funcMap := sprig.TxtFuncMap()

	funcMap["templateToString"] = func(name string, data interface{}) string {
		return RenderTemplateToString(tmpl, name, data)
	}

	funcMap["hasGPU"] = HasGPU

	tmpl = template.New("").Funcs(funcMap)

	// Parse all .tmpl files in the directory
	tmpl, err = tmpl.ParseFiles(files...)
	if err != nil {
		return nil, err
	}

	return tmpl, nil
}

func RenderGoTemplate(tmpl *template.Template, templateName string, context map[string]interface{}) (string, error) {
	var output bytes.Buffer

	if err := tmpl.ExecuteTemplate(&output, templateName, context); err != nil {
		return "", err
	}

	return output.String(), nil
}

type InMemoryLoader struct {
	templates map[string]*template.Template
}

func NewInMemoryLoader() *InMemoryLoader {
	return &InMemoryLoader{templates: make(map[string]*template.Template)}
}

func (loader *InMemoryLoader) Open(name string) (io.Reader, error) {
	if tmpl, ok := loader.templates[name]; ok {
		var buf bytes.Buffer
		tmpl.Execute(&buf, nil)
		return strings.NewReader(buf.String()), nil
	}
	return nil, fmt.Errorf("template %s not found", name)
}

func (loader *InMemoryLoader) Exists(name string) bool {
	_, exists := loader.templates[name]
	return exists
}

func BuildConnectionString(host string, user string, password string, dbname string) string {
	return fmt.Sprintf("host=%s user=%s password=%s dbname=%s sslmode=disable", host, user, password, dbname)
}

func (loader *InMemoryLoader) LoadTemplatesFromDB(connStr string, tableName string) error {
	conn, err := pgx.Connect(context.Background(), connStr)
	if err != nil {
		return err
	}
	defer conn.Close(context.Background())

	rows, err := conn.Query(context.Background(), fmt.Sprintf("SELECT set_name, name, content FROM %s", tableName))
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var setName, name, content string
		err = rows.Scan(&setName, &name, &content)
		if err != nil {
			return err
		}
		fullName := setName + ":" + name
		tmpl, err := template.New(fullName).Parse(content)
		if err != nil {
			return err
		}
		loader.templates[fullName] = tmpl
	}
	return nil
}
