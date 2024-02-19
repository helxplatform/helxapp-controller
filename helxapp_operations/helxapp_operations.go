package helxapp_operations

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"text/template"

	"github.com/google/uuid"
	helxv1 "github.com/helxplatform/helxapp/api/v1"
	"github.com/helxplatform/helxapp/template_io"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/yaml"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type RenderArtifacts struct {
	DeploymentString string
	PVCStrings       map[string]string
}

var apps = make(map[string]helxv1.HelxApp)
var xformer *template.Template
var storage map[string][]string
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

func clearStorage() {
	for k := range storage {
		delete(storage, k)
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
		xformer, storage, err = template_io.ParseTemplates("templates", simpleInfoLogger)
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

/*
ProcessVolume parses a volume string according to the following BNF specification:

<volume-source> ::= [<scheme> "://"] <source>
<source>        ::= <src> ":" <mntpoint> ["#" <subpath>] ["," <optionlist>]
<optionlist>    ::= <option> ["," <optionlist>]
<option>        ::= <key> ["=" <value>]
<key>           ::= <string>
<value>         ::= <string>

Where:
  - <scheme> can be "pvc" or "nfs". If omitted, defaults to "pvc".
  - <src> is the source of the volume.
  - <mntpoint> is the mount point for the volume.
  - <subpath> (optional) is a subpath within the volume.
  - <optionlist> (optional) is a comma-separated list of options. Each option can
    be a key-value pair (<key>=<value>) or a single key, in which case the value
    defaults to "true".
  - <string> represents a sequence of characters where ":", "#", ",", and "=" are
    disallowed, except as delimiters within the structure.

The pattern used to parse the volume string accommodates these rules, allowing
for flexible volume specification with optional default values for omitted parts.

Example volume strings:
- "pvc://myvolume:/mnt"
- "nfs://server/path:/mnt#subpath,opt1=val1,opt2"
- "myvolume:/mnt,opt1,opt2=val2"

This function returns a pointer to a Volume and VolumeMount object populated
based on the parsed input, or an error if the input does not match the expected
format.
*/
func processVolume(volumeId, volumeStr string) (*template_io.Volume, *template_io.VolumeMount, error) {
	attr := make(map[string]string)
	pattern := `^(?:(pvc|nfs)(:\/\/))?([^:#,]+):([^:#,]+)(?:#([^:#,]*))?(?:,(([^:#,=]+(?:=[^:#,=]+)?)(?:,([^:#,=]+(?:=[^:#,=]+)?))*))?$`
	re := regexp.MustCompile(pattern)

	matches := re.FindStringSubmatch(volumeStr)
	if matches == nil {
		return nil, nil, fmt.Errorf("volume spec does not match the expected format")
	}

	// Extracting the components
	scheme := matches[1] // Scheme might be empty
	if scheme == "" {
		scheme = "pvc" // Default to "pvc" if empty
	}
	src := matches[3]
	mountPath := matches[4]
	subPath := matches[5]
	options := matches[6]

	// Parsing options into a map
	optionMap := make(map[string]string)
	if options != "" {
		optionPairs := regexp.MustCompile(`([^:#,=]+)(?:=([^:#,=]+))?`)
		for _, pair := range optionPairs.FindAllStringSubmatch(options, -1) {
			key := pair[1]
			value := "true" // Default to "true" if no explicit value
			if len(pair) > 2 && pair[2] != "" {
				value = pair[2]
			}
			optionMap[key] = value
		}
	}

	if scheme == "pvc" {
		attr["claim"] = src
	} else if scheme == "nfs" {
		// Split src into components
		parts := strings.SplitN(src, "/", 3) // Split into 3 parts to separate server from path
		if len(parts) < 3 {
			return nil, nil, fmt.Errorf("invalid NFS source format")
		}
		// The first part is empty due to the leading '/', so parts[1] is the server and parts[2] is the path
		attr["server"] = parts[1]
		attr["path"] = "/" + parts[2] // Prepend '/' to the path to maintain its absolute format
	} else {
		return nil, nil, fmt.Errorf("unknown scheme")
	}

	readOnly := false

	if ro, found := optionMap["ro"]; found && ro == "true" {
		readOnly = true
	}

	templateVolume := template_io.Volume{
		Name:   volumeId,
		Scheme: scheme,
		Attr:   attr,
	}

	templateVolumeMount := template_io.VolumeMount{
		Name:      volumeId,
		MountPath: mountPath,
		SubPath:   subPath,
		ReadOnly:  readOnly,
	}

	return &templateVolume, &templateVolumeMount, nil
}

func parseServiceVolume(service helxv1.Service, sourceMap map[string]*template_io.Volume) ([]*template_io.VolumeMount, error) {
	var details []*template_io.VolumeMount

	for volumeName, volume := range service.Volumes {
		templateVolume, templateVolumeMount, err := processVolume(volumeName, volume)
		if err != nil {
			return nil, err
		}
		if _, found := sourceMap[volumeName]; !found {
			sourceMap[volumeName] = templateVolume
		}
		details = append(details, templateVolumeMount)
	}

	return details, nil
}

func CreateDeploymentArtifacts(instance *helxv1.HelxInstanceSpec) (*RenderArtifacts, error) {
	uuid := uuid.New()
	id := uuid.String()
	containers := []template_io.Container{}
	sourceMap := make(map[string]*template_io.Volume)

	if app, found := GetAppFromMap(apps, instance.AppName); found {
		for i := 0; i < len(app.Spec.Services); i++ {
			ports := []template_io.Port{}
			name := app.Spec.Services[i].Name

			for j := 0; j < len(app.Spec.Services[i].Ports); j++ {
				srcPort := app.Spec.Services[i].Ports[j]
				ports = append(ports, template_io.Port{ContainerPort: int(srcPort.ContainerPort), Protocol: "TCP"})
			}

			if volumeList, err := parseServiceVolume(app.Spec.Services[i], sourceMap); err == nil {
				resources := template_io.Resources{}

				if resource, found := instance.Resources[name]; found {
					resources.Limits = resource.Limits
					resources.Requests = resource.Requests
				}

				c := template_io.Container{
					Name:         name,
					Image:        app.Spec.Services[i].Image,
					Ports:        ports,
					Expose:       ports,
					Resources:    resources,
					VolumeMounts: volumeList,
				}
				containers = append(containers, c)
			} else {
				simpleErrorLogger(err, "parse sources failed")
			}
		}
		if len(containers) >= 1 {
			volumes := make(map[string]template_io.Volume)

			for name, value := range sourceMap {
				volumes[name] = *value
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
				Volumes:        volumes,
			}

			vars := make(map[string]interface{})
			vars["system"] = system

			simpleInfoLogger(fmt.Sprintf("applying template to %v+", system))
			clearStorage()
			if deploymentYAML, err := template_io.RenderGoTemplate(xformer, "deployment", vars); err != nil {
				simpleErrorLogger(err, "RenderGoTemplate failed")
				return nil, err
			} else {
				artifacts := RenderArtifacts{}
				artifacts.DeploymentString = deploymentYAML
				for _, volume := range system.Volumes {
					if volume.Scheme == "pvc" {
						vars := make(map[string]interface{})
						vars["volume"] = volume
						if pvcYAML, err := template_io.RenderGoTemplate(xformer, "pvc", vars); err != nil {
							simpleErrorLogger(err, "RenderGoTemplate failed")
							return nil, err
						} else {
							if artifacts.PVCStrings == nil {
								artifacts.PVCStrings = make(map[string]string)
							}
							artifacts.PVCStrings[volume.Attr["claim"]] = pvcYAML
						}
					}
				}
				return &artifacts, nil
			}
		}
	}
	return nil, nil
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

func CreatePVCFromYAML(ctx context.Context, c client.Client, scheme *runtime.Scheme, req ctrl.Request, instance *helxv1.HelxInstance, pvcYAML string) error {
	// Convert YAML string to a PVC object
	decode := yaml.NewYAMLOrJSONDecoder(strings.NewReader(pvcYAML), 100)
	var pvc corev1.PersistentVolumeClaim
	if err := decode.Decode(&pvc); err != nil {
		return err
	}

	// Set the Namespace and Name for the PVC if it's not set
	if pvc.Namespace == "" {
		pvc.Namespace = req.NamespacedName.Namespace
	}
	if pvc.Name == "" {
		pvc.Name = req.NamespacedName.Name
	}

	// Set the controller reference so that the PVC will be deleted when the HelxApp is deleted
	if err := ctrl.SetControllerReference(instance, &pvc, scheme); err != nil {
		return err
	}

	// Check if the PVC already exists
	var existingPVC corev1.PersistentVolumeClaim
	err := c.Get(ctx, client.ObjectKey{
		Namespace: pvc.Namespace,
		Name:      pvc.Name,
	}, &existingPVC)

	// Proceed with creation only if PVC does not exist
	if err != nil && errors.IsNotFound(err) {
		// PVC does not exist, create it
		if err := c.Create(ctx, &pvc); err != nil {
			return err
		}
	} else if err != nil {
		// An error occurred that isn't related to the non-existence of the PVC
		return err
	}

	return nil
}
