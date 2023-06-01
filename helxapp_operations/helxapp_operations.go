package helxapp_operations

import (
	"context"
	"fmt"
	"strings"
	"text/template"

	"github.com/google/uuid"
	helxv1 "github.com/helxplatform/helxapp/api/v1"
	"github.com/helxplatform/helxapp/template_io"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/yaml"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var apps *map[string]helxv1.HelxAppSpec
var xformer *template.Template

// AddToMap adds the given HelxAppSpec instance to the map
func addAppToMap(m *map[string]helxv1.HelxAppSpec, app *helxv1.HelxAppSpec) {
	(*m)[app.Name] = *app
}

func AddApp(app *helxv1.HelxAppSpec) {
	addAppToMap(apps, app)
}

func GetAppFromMap(m *map[string]helxv1.HelxAppSpec, appName string) helxv1.HelxAppSpec {
	return (*m)[appName]
}

func GetApp(appName string) helxv1.HelxAppSpec {
	return GetAppFromMap(apps, appName)
}

func Init() error {
	var err error

	xformer, err = template_io.ParseTemplates("../templates")
	if err != nil {
		fmt.Printf("failed to initialize xformer template: %v", err)
	}
	return err
}

func CreateDeploymentString(instance *helxv1.HelxAppInstanceSpec) string {
	uuid := uuid.New()
	id := uuid.String()
	containers := []template_io.Container{}
	appSpec := GetAppFromMap(apps, instance.AppName)

	for i := 0; i < len(appSpec.Services); i++ {
		ports := []template_io.Port{}

		for j := 0; j < len(appSpec.Services[i].Ports); j++ {
			srcPort := appSpec.Services[i].Ports[j]
			ports = append(ports, template_io.Port{ContainerPort: int(srcPort.ContainerPort), Protocol: "TCP"})
		}
		c := template_io.Container{
			Name:         appSpec.Services[i].Name,
			Image:        appSpec.Services[i].Image,
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

	// Call the function.
	result, err := template_io.RenderGoTemplate(xformer, "deployment", vars)
	if err != nil {
		fmt.Printf("RenderGoTemplate() error = %v", err)
		return ""
	}
	return result
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
