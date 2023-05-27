package template_io

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/CloudyKit/jet/v6"
	"github.com/jackc/pgx/v4"
)

type System struct {
	Name            string
	AMB             bool
	SystemEnv       []EnvVar
	Username        string
	SystemName      string
	Host            string
	Identifier      string
	AppID           string
	CreateHomeDirs  bool
	DevPhase        string
	SecurityContext *SecurityContext
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

var JetTemplate *jet.Template

// InitJetTemplate initializes the JetTemplate variable with a given directory and template name
func InitJetTemplate(dir string, templateName string) error {
	_, err := os.Stat(dir)
	if err != nil {
		return err
	}

	loader := jet.NewOSFileSystemLoader(dir)
	set := jet.NewSet(loader, jet.InDevelopmentMode())

	JetTemplate, err = set.GetTemplate(templateName)
	if err != nil {
		return err
	}

	return nil
}

func RenderJetTemplate(vars jet.VarMap) (string, error) {
	var buf bytes.Buffer
	if err := JetTemplate.Execute(&buf, vars, nil); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// InMemoryLoader holds templates in memory
type InMemoryLoader struct {
	templates map[string]string
}

func NewInMemoryLoader() *InMemoryLoader {
	return &InMemoryLoader{templates: make(map[string]string)}
}

func (loader *InMemoryLoader) Open(name string) (io.Reader, error) {
	if content, ok := loader.templates[name]; ok {
		return strings.NewReader(content), nil
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
		loader.templates[fullName] = content
	}
	return nil
}
