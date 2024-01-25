package helxapp_operations

import (
	"context"
	"errors"
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
		xformer, err = template_io.ParseTemplates("templates", simpleInfoLogger)
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

func processSingleVolume(volumeMount helxv1.VolumeMount) (template_io.Volume, template_io.VolumeMount, string, error) {
	var scheme, source, path string

	if strings.Contains(volumeMount.SourcePath, "://") {
		parts := strings.SplitN(volumeMount.SourcePath, "://", 2)
		scheme = parts[0]
		if scheme != "pvc" && scheme != "home" {
			return template_io.Volume{}, template_io.VolumeMount{}, "", errors.New("invalid scheme detected")
		}
		split := strings.SplitN(parts[1], "/", 2)
		source = split[0]
		if len(split) > 1 {
			path = split[1]
		}
	} else {
		scheme = "pvc"
		split := strings.SplitN(volumeMount.SourcePath, "/", 2)
		source = split[0]
		if len(split) > 1 {
			path = split[1]
		}
	}

	sourceKey := scheme + ":" + source
	templateVolume := template_io.Volume{
		Scheme: scheme,
		Source: source,
		Path:   path,
	}

	templateVolumeMount := template_io.VolumeMount{
		Name:      sourceKey,
		MountPath: path,
		SubPath:   "",
		ReadOnly:  false,
	}

	return templateVolume, templateVolumeMount, sourceKey, nil
}

func parseVolumeSourcePath(service helxv1.Service, sourceMap map[string]template_io.Volume) ([]template_io.VolumeMount, error) {
	var details []template_io.VolumeMount

	for _, volumeMount := range service.Volumes {
		templateVolume, templateVolumeMount, sourceKey, err := processSingleVolume(volumeMount)
		if err != nil {
			return nil, err
		}
		if _, found := sourceMap[sourceKey]; !found {
			sourceMap[sourceKey] = templateVolume
		}
		details = append(details, templateVolumeMount)
	}

	return details, nil
}

func CreateDeploymentString(instance *helxv1.HelxInstanceSpec) string {
	uuid := uuid.New()
	id := uuid.String()
	containers := []template_io.Container{}
	var sourceMap map[string]template_io.Volume

	if app, found := GetAppFromMap(apps, instance.AppName); found {
		for i := 0; i < len(app.Spec.Services); i++ {
			ports := []template_io.Port{}

			for j := 0; j < len(app.Spec.Services[i].Ports); j++ {
				srcPort := app.Spec.Services[i].Ports[j]
				ports = append(ports, template_io.Port{ContainerPort: int(srcPort.ContainerPort), Protocol: "TCP"})
			}

			if volumeMap, err := parseVolumeSourcePath(app.Spec.Services[i], sourceMap); err == nil {
				c := template_io.Container{
					Name:         app.Spec.Services[i].Name,
					Image:        app.Spec.Services[i].Image,
					Ports:        ports,
					Expose:       ports,
					VolumeMounts: volumeMap,
				}
				containers = append(containers, c)
			} else {
				simpleErrorLogger(err, "parse sources failed")
			}
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
		if result, err := template_io.RenderGoTemplate(xformer, "deployment", vars); err != nil {
			simpleErrorLogger(err, "RenderGoTemplate failed")
			return ""
		} else {
			return result
		}
	}
	return ""
}

func CreateDeploymentFromYAML(ctx context.Context, c client.Client, scheme *runtime.Scheme, req ctrl.Request, instance *helxv1.HelxInstance, deploymentYAML string) error {
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
	if err := ctrl.SetControllerReference(instance, &deployment, scheme); err != nil {
		return err
	}

	// Create the Deployment
	if err := c.Create(ctx, &deployment); err != nil {
		return err
	}

	return nil
}
