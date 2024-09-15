package helxapp_operations

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"text/template"

	"github.com/go-logr/logr"
	helxv1 "github.com/helxplatform/helxapp-controller/api/v1"
	"github.com/helxplatform/helxapp-controller/template_io"
	"gomodules.xyz/jsonpatch/v2"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/yaml"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type RenderArtifact struct {
	Render string
	Attr   map[string]string
}

type Artifacts struct {
	Deployment RenderArtifact
	PVCs       map[string]RenderArtifact
	Services   map[string]RenderArtifact
}

type InstTableElement struct {
	Inst helxv1.HelxInst
}

type TableElement[T any] struct {
	Obj     *T
	InstSet map[string]bool
}

var appTable = make(map[string]TableElement[helxv1.HelxApp])
var userTable = make(map[string]TableElement[helxv1.HelxUser])
var instanceTable = make(map[string]InstTableElement)
var xformer *template.Template
var storage map[string][]string
var simpleDebugLogger func(string)
var simpleInfoLogger func(string)
var simpleErrorLogger func(error, string)

func newSimpleDebugLogger(logger logr.Logger) func(message string) {
	return func(message string) {
		logger.V(1).Info(message)
	}
}
func newSimpleInfoLogger(logger logr.Logger) func(message string) {
	return func(message string) {
		logger.Info(message)
	}
}

func newSimpleErrorLogger(logger logr.Logger) func(err error, message string) {
	return func(err error, message string) {
		logger.Error(err, message)
	}
}

func Initalize(logger logr.Logger) error {
	var err error

	simpleDebugLogger = newSimpleDebugLogger(logger)
	simpleInfoLogger = newSimpleInfoLogger(logger)
	simpleErrorLogger = newSimpleErrorLogger(logger)
	xformer, storage, err = template_io.ParseTemplates("templates", simpleDebugLogger)
	if err != nil {
		simpleErrorLogger(err, "failed to initialize xformer template")
		return err
	} else {
		simpleInfoLogger("helxapp_operations initialized")
		return nil
	}
}

func clearStorage() {
	for k := range storage {
		delete(storage, k)
	}
}

func GetNamespacedName[T client.Object](obj T) string {
	return obj.GetNamespace() + "/" + obj.GetName()
}

func GetAppNameFromInst(inst *helxv1.HelxInst) string {
	if inst == nil || inst.Spec.AppName == "" {
		return ""
	} else {
		if strings.Contains(inst.Spec.AppName, "/") {
			return inst.Spec.AppName
		} else {
			return inst.Namespace + "/" + inst.Spec.AppName
		}
	}
}

func GetUserNameFromInst(inst *helxv1.HelxInst) string {
	if inst == nil || inst.Spec.UserName == "" {
		return ""
	} else {
		if strings.Contains(inst.Spec.UserName, "/") {
			return inst.Spec.UserName
		} else {
			return inst.Namespace + "/" + inst.Spec.UserName
		}
	}
}

func addInstToMap(m map[string]InstTableElement, instName string, inst *helxv1.HelxInst) {
	simpleDebugLogger(fmt.Sprintf("storing Inst using name '%s'", instName))
	m[instName] = InstTableElement{Inst: *inst}
}

// AddToMap adds the given HelxAppSpec instance to the map
func addObjToMap[T any](m map[string]TableElement[T], objName string, obj *T) []helxv1.HelxInst {
	var instNames []string
	var instList []helxv1.HelxInst

	simpleDebugLogger(fmt.Sprintf("storing Object using name '%s'", objName))
	if element, found := m[objName]; !found {
		element = TableElement[T]{Obj: obj, InstSet: make(map[string]bool)}
		m[objName] = element
	} else {
		element.Obj = obj
		m[objName] = element
		for k := range element.InstSet {
			instNames = append(instNames, k)
		}
	}

	for _, instName := range instNames {
		if element, found := instanceTable[instName]; found {
			instList = append(instList, element.Inst)
		}
	}
	return instList
}

// AddToMap adds the given HelxAppSpec instance to the map
func connectObj[T any](m map[string]TableElement[T], objName string, instName string) {
	simpleDebugLogger(fmt.Sprintf("associating '%s' and '%s'", objName, instName))
	if element, found := m[objName]; !found {
		element = TableElement[T]{Obj: nil, InstSet: make(map[string]bool)}
		element.InstSet[instName] = true
		m[objName] = element
	} else {
		element.InstSet[instName] = true
	}
}

func AddApp(app *helxv1.HelxApp) []helxv1.HelxInst {
	appName := GetNamespacedName(app)

	return addObjToMap[helxv1.HelxApp](appTable, appName, app)
}

func AddInst(inst *helxv1.HelxInst) {
	appName := GetAppNameFromInst(inst)
	instName := GetNamespacedName(inst)
	userName := GetUserNameFromInst(inst)

	addInstToMap(instanceTable, instName, inst)
	connectObj(appTable, appName, instName)
	connectObj(userTable, userName, instName)
}

func AddUser(user *helxv1.HelxUser) []helxv1.HelxInst {
	userName := GetNamespacedName(user)

	return addObjToMap[helxv1.HelxUser](userTable, userName, user)
}

func GetInstFromMap(m map[string]InstTableElement, appName string) (helxv1.HelxInst, bool) {
	if value, found := m[appName]; found {
		return value.Inst, found
	} else {
		return helxv1.HelxInst{}, false
	}
}

func GetObjFromMap[T any](m map[string]TableElement[T], objName string) *T {
	if value, found := m[objName]; found {
		return value.Obj
	} else {
		return nil
	}
}

func GetApp(appName string) *helxv1.HelxApp {
	return GetObjFromMap[helxv1.HelxApp](appTable, appName)
}

func GetInst(instName string) (helxv1.HelxInst, bool) {
	return GetInstFromMap(instanceTable, instName)
}

func GetUser(userName string) *helxv1.HelxUser {
	return GetObjFromMap[helxv1.HelxUser](userTable, userName)
}

func DeleteObjFromMap[T any](m map[string]TableElement[T], objName string) []helxv1.HelxInst {
	instList := []helxv1.HelxInst{}

	if element, found := m[objName]; found {
		for k := range element.InstSet {
			if inst, found := GetInst(k); found {
				instList = append(instList, inst)
			}
		}
	}
	return instList
}

func DeleteApp(appName string) []helxv1.HelxInst {
	return DeleteObjFromMap[helxv1.HelxApp](appTable, appName)
}

func DeleteInst(instName string) {
	if instElement, found := instanceTable[instName]; found {
		appName := instElement.Inst.Spec.AppName
		if appElement, found := appTable[appName]; found {
			delete(appElement.InstSet, appName)
		}
		userName := instElement.Inst.Spec.UserName
		if userElement, found := userTable[userName]; found {
			delete(userElement.InstSet, userName)
		}
		delete(instanceTable, instName)
	}
}

func DeleteUser(userName string) []helxv1.HelxInst {
	return DeleteObjFromMap[helxv1.HelxUser](userTable, userName)
}

/*
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
*/

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
	if options != "" {
		optionPairs := regexp.MustCompile(`([^:#,=]+)(?:=([^:#,=]+))?`)
		for _, pair := range optionPairs.FindAllStringSubmatch(options, -1) {
			key := pair[1]
			value := "true" // Default to "true" if no explicit value
			if len(pair) > 2 && pair[2] != "" {
				value = pair[2]
			}
			attr[key] = value
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

	if ro, found := attr["ro"]; found && ro == "true" {
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

// processPorts creates port mappings and checks if any service is available.
func transformPorts(srcPorts []helxv1.PortMap) ([]template_io.PortMap, bool) {
	var dstPorts []template_io.PortMap
	hasService := false

	for _, srcMap := range srcPorts {
		dstMap := template_io.PortMap{
			ContainerPort: int(srcMap.ContainerPort),
			Protocol:      "TCP",
			Port:          int(srcMap.Port),
		}
		dstPorts = append(dstPorts, dstMap)
		if dstMap.Port != 0 {
			hasService = true
		}
	}

	return dstPorts, hasService
}

func transformVolumes(service helxv1.Service, volumeSourceMap map[string]*template_io.Volume) ([]*template_io.VolumeMount, error) {
	var details []*template_io.VolumeMount

	for volumeName, volume := range service.Volumes {
		templateVolume, templateVolumeMount, err := processVolume(volumeName, volume)
		if err != nil {
			return nil, err
		}
		if _, found := volumeSourceMap[volumeName]; !found {
			volumeSourceMap[volumeName] = templateVolume
		}
		details = append(details, templateVolumeMount)
	}

	return details, nil
}

// getResourceLimits returns the resource limits and requests if available.
func transformResources(serviceName string, resources map[string]helxv1.Resources) template_io.Resources {
	if resource, found := resources[serviceName]; found {
		return template_io.Resources{
			Limits:   resource.Limits,
			Requests: resource.Requests,
		}
	}

	return template_io.Resources{}
}

// ServiceProcessor processes the services from the application spec and returns containers.
func transformApp(instance *helxv1.HelxInst, app helxv1.HelxApp) ([]template_io.Container, map[string]*template_io.Volume, error) {
	containers := []template_io.Container{}
	sourceMap := make(map[string]*template_io.Volume)

	for _, service := range app.Spec.Services {
		ports, hasService := transformPorts(service.Ports)
		volumeList, err := transformVolumes(service, sourceMap)
		if err != nil {
			simpleErrorLogger(err, "parse volume sources failed")
			continue
		}

		resources := transformResources(service.Name, instance.Spec.Resources)
		container := template_io.Container{
			Name:         service.Name,
			Command:      service.Command[:],
			Environment:  service.Environment,
			HasService:   hasService,
			Image:        service.Image,
			Ports:        ports,
			Resources:    resources,
			VolumeMounts: volumeList,
		}

		containers = append(containers, container)
	}

	return containers, sourceMap, nil
}

// stabilizeRender performs re-renders until the output stabilizes.
func renderObject(system template_io.System, templateName string, objID string, obj interface{}, postRender func(string)) error {
	vars := make(map[string]interface{})

	vars["system"] = system
	if obj != nil {
		vars[objID] = obj
	}

	if initialRender, err := template_io.RenderGoTemplate(xformer, templateName, vars); err != nil {
		simpleErrorLogger(err, "RenderGoTemplate failed")
		return err
	} else {
		current := initialRender
		previous := initialRender

		for {
			var err error
			current, err = template_io.ReRender(previous, vars)
			if err != nil {
				simpleErrorLogger(err, "ReRender failed")
				return err
			}
			if current == previous {
				break
			}
			previous = current
		}

		postRender(current)
		return nil
	}
}

func GenerateArtifacts(instance *helxv1.HelxInst) (*Artifacts, error) {
	appName := GetAppNameFromInst(instance)
	userName := GetUserNameFromInst(instance)

	simpleDebugLogger(fmt.Sprintf("retrieving (app,user) via (%s,%s)", appName, userName))

	app := GetApp(appName)
	user := GetUser(userName)

	if app != nil && user != nil {
		containers, volumeSourceMap, error := transformApp(instance, *app)
		if error == nil && len(containers) >= 1 {
			volumes := make(map[string]template_io.Volume)

			for name, value := range volumeSourceMap {
				volumes[name] = *value
			}

			systemEnv := make(map[string]string)
			systemEnv["GUID"] = instance.Status.UUID
			systemEnv["USER"] = instance.Spec.UserName
			systemEnv["HOST"] = ""
			systemEnv["APP_CLASS_NAME"] = app.Spec.AppClassName
			systemEnv["APP_NAME"] = instance.Spec.AppName
			systemEnv["INSTANCE_NAME"] = instance.GetNamespace() + "/" + instance.GetName()

			system := template_io.System{
				AppClassName: app.Spec.AppClassName,
				AppName:      instance.Spec.AppName,
				InstanceName: instance.Name,
				Containers:   containers,
				Environment:  systemEnv,
				Host:         "",
				UUID:         instance.Status.UUID,
				UserHandle:   user.Spec.UserHandle,
				UserName:     instance.Spec.UserName,
				Volumes:      volumes,
			}

			simpleInfoLogger("applying templates")
			clearStorage()

			artifacts := Artifacts{}

			if err := renderObject(system, "deployment", "", nil, func(render string) {
				artifacts.Deployment = RenderArtifact{Render: render, Attr: make(map[string]string)}
			}); err != nil {
				return nil, err
			}

			for _, volume := range system.Volumes {
				if volume.Scheme == "pvc" {
					if err := renderObject(system, "pvc", "volume", volume, func(render string) {
						if artifacts.PVCs == nil {
							artifacts.PVCs = make(map[string]RenderArtifact)
						}
						artifacts.PVCs[volume.Attr["claim"]] = RenderArtifact{Render: render, Attr: make(map[string]string)}
						if retain, found := volume.Attr["retain"]; found {
							artifacts.PVCs[volume.Attr["claim"]].Attr["retain"] = retain
						}
					}); err != nil {
						return nil, err
					}
				}
			}

			for _, container := range system.Containers {
				if err := renderObject(system, "service", "container", container, func(render string) {
					if artifacts.Services == nil {
						artifacts.Services = make(map[string]RenderArtifact)
					}
					simpleDebugLogger(fmt.Sprintf("processing service %s", container.Name))
					simpleDebugLogger(fmt.Sprintf("render = %s", render))
					artifacts.Services[container.Name] = RenderArtifact{Render: render, Attr: make(map[string]string)}
				}); err != nil {
					return nil, err
				}
			}

			return &artifacts, nil
		}
	}
	return nil, nil
}

func CreateOrUpdateResource[T client.Object](
	ctx context.Context,
	c client.Client,
	scheme *runtime.Scheme,
	req ctrl.Request,
	instance *helxv1.HelxInst,
	src string,
	getTarget func() (T, error),
	acceptablePatchOp func(jsonpatch.JsonPatchOperation) bool) error {

	target, err := getTarget()

	if err != nil {
		return err
	}
	// Set the Namespace and Name for the Resource if it's not set
	if target.GetNamespace() == "" {
		target.SetNamespace(req.NamespacedName.Namespace)
	}
	if target.GetName() == "" {
		target.SetName(req.NamespacedName.Name)
	}

	// Check if the resource already exists
	existing := target.DeepCopyObject().(T)

	err = c.Get(ctx, types.NamespacedName{Name: target.GetName(), Namespace: target.GetNamespace()}, existing)
	if err != nil && errors.IsNotFound(err) {
		// Resource does not exist, create it
		simpleInfoLogger(fmt.Sprintf("creating resource %s", GetNamespacedName(target)))

		// Check for the retain label before setting the controller reference
		labels := target.GetLabels()
		if retain, exists := labels["helx.renci.org/retain"]; !exists || retain != "true" {
			// Set the controller reference so that the Resource will be deleted when the HelxInst is deleted
			if err := ctrl.SetControllerReference(instance, target, scheme); err != nil {
				return err
			}
		} else {
			// Optionally, log or handle the case where the retain label is set to true
			simpleInfoLogger(fmt.Sprintf("Skipping setting controller reference for %s as it has the retain label set to true", GetNamespacedName(target)))
		}
		if err := c.Create(ctx, target); err != nil {
			return err
		}
	} else if err != nil {
		return err
	} else {
		simpleInfoLogger(fmt.Sprintf("patching resource %s", GetNamespacedName(existing)))

		// Marshal both objects to JSON
		existingJSON, err := json.Marshal(existing)
		if err != nil {
			return fmt.Errorf("failed to marshal existing object: %v", err)
		}
		simpleDebugLogger(fmt.Sprintf("existingJSON:\n%s", existingJSON))
		targetJSON, err := json.Marshal(target)
		if err != nil {
			return fmt.Errorf("failed to marshal target object: %v", err)
		}
		simpleDebugLogger(fmt.Sprintf("targetJSON:\n%s", targetJSON))

		// Compute the JSON Patch
		patch, err := jsonpatch.CreatePatch(existingJSON, targetJSON)
		if err != nil {
			return fmt.Errorf("failed to create JSON patch: %v", err)
		}

		var filteredPatch []jsonpatch.JsonPatchOperation

		for _, op := range patch {
			if found := acceptablePatchOp(op); found {
				filteredPatch = append(filteredPatch, op)
			}
		}

		// Apply the JSON Patch
		patchBytes, err := json.Marshal(filteredPatch)
		if err != nil {
			return fmt.Errorf("failed to marshal JSON patch: %v", err)
		}
		simpleDebugLogger(fmt.Sprintf("patch:\n%s", patchBytes))
		if err := c.Patch(ctx, existing, client.RawPatch(types.JSONPatchType, patchBytes)); err != nil {
			return fmt.Errorf("failed to apply patch: %v", err)
		}

		simpleInfoLogger("Resource updated successfully")
		return nil
	}

	return nil
}

func DeleteDeployments(ctx context.Context, c client.Client, instance *helxv1.HelxInst) error {
	var deployments *appsv1.DeploymentList = new(appsv1.DeploymentList)

	listOpts := []client.ListOption{
		client.InNamespace(instance.ObjectMeta.Namespace),
		client.MatchingLabels{"helx.renci.org/id": instance.Status.UUID},
	}

	if err := c.List(ctx, deployments, listOpts...); err != nil {
		return fmt.Errorf("failed to get deployment list: %v", err)
	}

	for _, deployment := range deployments.Items {
		if retain, found := deployment.ObjectMeta.Labels["helx.renci.org/retain"]; !found || retain != "true" {
			if err := c.Delete(ctx, &deployment, client.PropagationPolicy(metav1.DeletePropagationForeground)); err != nil {
				return fmt.Errorf("failed to delete deployment: %v", err)
			}
		}
	}
	return nil
}

func DeletePVCs(ctx context.Context, c client.Client, instance *helxv1.HelxInst) error {
	var pvcs *corev1.PersistentVolumeClaimList = new(corev1.PersistentVolumeClaimList)

	listOpts := []client.ListOption{
		client.InNamespace(instance.ObjectMeta.Namespace),
		client.MatchingLabels{"helx.renci.org/id": instance.Status.UUID},
	}

	if err := c.List(ctx, pvcs, listOpts...); err != nil {
		return fmt.Errorf("failed to get pvc list: %v", err)
	}

	for _, pvc := range pvcs.Items {
		if retain, found := pvc.ObjectMeta.Labels["helx.renci.org/retain"]; !found || retain != "true" {
			if err := c.Delete(ctx, &pvc, client.PropagationPolicy(metav1.DeletePropagationForeground)); err != nil {
				return fmt.Errorf("failed to delete pvc: %v", err)
			}
		}
	}
	return nil
}

func DeleteServices(ctx context.Context, c client.Client, instance *helxv1.HelxInst) error {
	var services *corev1.ServiceList = new(corev1.ServiceList)

	listOpts := []client.ListOption{
		client.InNamespace(instance.ObjectMeta.Namespace),
		client.MatchingLabels{"helx.renci.org/id": instance.Status.UUID},
	}

	if err := c.List(ctx, services, listOpts...); err != nil {
		return fmt.Errorf("failed to get service list: %v", err)
	}

	for _, service := range services.Items {
		if retain, found := service.ObjectMeta.Labels["helx.renci.org/retain"]; !found || retain != "true" {
			if err := c.Delete(ctx, &service, client.PropagationPolicy(metav1.DeletePropagationForeground)); err != nil {
				return fmt.Errorf("failed to delete service: %v", err)
			}
		}
	}
	return nil
}

func DeploymentFromYAML(ctx context.Context, c client.Client, scheme *runtime.Scheme, req ctrl.Request, instance *helxv1.HelxInst, artifact RenderArtifact) error {
	return CreateOrUpdateResource(ctx, c, scheme, req, instance, artifact.Render,
		func() (*appsv1.Deployment, error) {
			decode := yaml.NewYAMLOrJSONDecoder(strings.NewReader(artifact.Render), 100)
			var deployment appsv1.Deployment

			simpleInfoLogger("creating deployment from string")
			if err := decode.Decode(&deployment); err != nil {
				return nil, err
			}
			return &deployment, nil
		},
		func(op jsonpatch.JsonPatchOperation) bool {
			return true
		})
}

func PVCFromYAML(ctx context.Context, c client.Client, scheme *runtime.Scheme, req ctrl.Request, instance *helxv1.HelxInst, artifact RenderArtifact) error {
	return CreateOrUpdateResource(ctx, c, scheme, req, instance, artifact.Render,
		func() (*corev1.PersistentVolumeClaim, error) {
			decode := yaml.NewYAMLOrJSONDecoder(strings.NewReader(artifact.Render), 100)
			var pvc corev1.PersistentVolumeClaim

			simpleInfoLogger("creating pvc from string")
			if err := decode.Decode(&pvc); err != nil {
				return nil, err
			}
			return &pvc, nil
		},
		func(op jsonpatch.JsonPatchOperation) bool {
			if op.Operation == "remove" {
				return false
			} else {
				return true
			}
		})
}

func ServiceFromYAML(ctx context.Context, c client.Client, scheme *runtime.Scheme, req ctrl.Request, instance *helxv1.HelxInst, artifact RenderArtifact) error {
	return CreateOrUpdateResource(ctx, c, scheme, req, instance, artifact.Render,
		func() (*corev1.Service, error) {
			decode := yaml.NewYAMLOrJSONDecoder(strings.NewReader(artifact.Render), 100)
			var service corev1.Service

			simpleInfoLogger("creating service from string")
			if err := decode.Decode(&service); err != nil {
				return nil, err
			}
			return &service, nil
		},
		func(op jsonpatch.JsonPatchOperation) bool {
			return true
		})
}

func CreateDerivatives(instance *helxv1.HelxInst, c client.Client, scheme *runtime.Scheme, req ctrl.Request, ctx context.Context) error {
	if artifacts, err := GenerateArtifacts(instance); err == nil {
		if artifacts != nil && artifacts.Deployment.Render != "" {
			simpleInfoLogger("generated Deployment YAML")
			simpleDebugLogger(artifacts.Deployment.Render)
			if err = DeploymentFromYAML(ctx, c, scheme, req, instance, artifacts.Deployment); err != nil {
				simpleErrorLogger(err, fmt.Sprintf("unable to create or update deployment NamespacedName: %s", req.NamespacedName))
			} else {
				for name, PVC := range artifacts.PVCs {
					if PVC.Render != "" {
						simpleInfoLogger("generated PVC YAML")
						simpleDebugLogger(PVC.Render)
						if err = PVCFromYAML(ctx, c, scheme, req, instance, PVC); err != nil {
							simpleErrorLogger(err, fmt.Sprintf("unable to create or update pvc PVCName: %s NamespacedName: %s ", name, req.NamespacedName))
						}
					}
				}
				for name, service := range artifacts.Services {
					if service.Render != "" {
						simpleInfoLogger("generated Service YAML:")
						simpleDebugLogger(service.Render)
						if err = ServiceFromYAML(ctx, c, scheme, req, instance, service); err != nil {
							simpleErrorLogger(err, fmt.Sprintf("unable to create or update service Service Name: %s NamespacedName: %s", name, req.NamespacedName))
						}
					}
				}
			}
		}
		return err
	} else {
		return err
	}
}

func DeleteDerivatives(instance *helxv1.HelxInst, c client.Client, req ctrl.Request, ctx context.Context) error {
	if err := DeleteDeployments(ctx, c, instance); err != nil {
		simpleErrorLogger(err, fmt.Sprintf("unable to delete deployments NamespacedName: %s", req.NamespacedName))
		return err
	}
	if err := DeletePVCs(ctx, c, instance); err != nil {
		simpleErrorLogger(err, fmt.Sprintf("unable to delete pvcs NamespacedName: %s", req.NamespacedName))
		return err
	}
	if err := DeleteServices(ctx, c, instance); err != nil {
		simpleErrorLogger(err, fmt.Sprintf("unable to delete services NamespacedName: %s", req.NamespacedName))
		return err
	}
	return nil
}
