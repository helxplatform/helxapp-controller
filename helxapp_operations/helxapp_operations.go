package helxapp_operations

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"text/template"

	"github.com/google/uuid"
	helxv1 "github.com/helxplatform/helxapp/api/v1"
	"github.com/helxplatform/helxapp/template_io"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/yaml"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var apps = make(map[string]helxv1.HelxApp)
var xformer *template.Template
var simpleInfoLogger func(string)
var simpleErrorLogger func(error, string)
var initialized bool = false
var initLock sync.Mutex

func newSimpleInfoLogger(ctx context.Context) func(message string) {
	return func(message string) {
		logger := log.FromContext(ctx)
		logger.Info(message)
	}
}

func newSimpleErrorLogger(ctx context.Context) func(err error, message string) {
	return func(err error, message string) {
		logger := log.FromContext(ctx)
		logger.Error(err, message)
	}
}

// AddToMap adds the given HelxAppSpec instance to the map
func addAppToMap(m map[string]helxv1.HelxApp, app *helxv1.HelxApp) {
	simpleInfoLogger(fmt.Sprintf("storing App using name '%s'", app.Name))
	m[app.Name] = *app
}

func AddApp(app *helxv1.HelxApp) {
	addAppToMap(apps, app)
}

func GetAppFromMap(m map[string]helxv1.HelxApp, appName string) (helxv1.HelxApp, bool) {
	value, found := m[appName]
	return value, found
}

func GetApp(appName string) (helxv1.HelxApp, bool) {
	return GetAppFromMap(apps, appName)
}

func CheckInit(ctx context.Context) error {
	var err error

	initLock.Lock()
	defer initLock.Unlock()
	if !initialized {
		simpleInfoLogger = newSimpleInfoLogger(ctx)
		simpleErrorLogger = newSimpleErrorLogger(ctx)
		xformer, err = template_io.ParseTemplates("templates")
		if err != nil {
			simpleErrorLogger(err, "failed to initialize xformer template")
			return err
		} else {
			simpleInfoLogger("helxapp_operations initialized")
			initialized = true
		}
	}
	return nil
}

func CreateDeploymentString(instance *helxv1.HelxAppInstanceSpec) string {
	uuid := uuid.New()
	id := uuid.String()
	containers := []template_io.Container{}

	if app, found := GetAppFromMap(apps, instance.AppName); found {
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
			Name:           instance.AppName,
			Username:       "jeffw",
			AppName:        instance.AppName,
			Host:           "",
			Identifier:     instance.AppName + "-" + id,
			CreateHomeDirs: false,
			DevPhase:       "test",
			Containers:     containers,
		}

		vars := make(map[string]interface{})
		vars["system"] = system

		simpleInfoLogger(fmt.Sprintf("applying template to %v+", system))
		// Call the function.
		if result, err := template_io.RenderGoTemplate(xformer, "deployment", vars); err != nil {
			simpleErrorLogger(err, "RenderGoTemplate failed")
			return ""
		} else {
			return result
		}
	}
	return ""
}

func CreateDeploymentFromYAML(ctx context.Context, c client.Client, scheme *runtime.Scheme, req ctrl.Request, deploymentYAML string) error {
	// Convert YAML string to a Deployment object
	decode := yaml.NewYAMLOrJSONDecoder(strings.NewReader(deploymentYAML), 100)
	var deployment appsv1.Deployment
	if err := decode.Decode(&deployment); err != nil {
		return err
	}

	// Set the Namespace and Name for the Deployment if it's not set
	if deployment.Namespace == "" {
		deployment.Namespace = req.NamespacedName.Namespace
	}
	if deployment.Name == "" {
		deployment.Name = req.NamespacedName.Name
	}

	// Set the controller reference so that the Deployment will be deleted when the HelxApp is deleted
	if err := ctrl.SetControllerReference(&deployment, &deployment, scheme); err != nil {
		return err
	}

	// Create the Deployment
	if err := c.Create(ctx, &deployment); err != nil {
		return err
	}

	return nil
}
