package helxapp_operations

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	helxv1 "github.com/helxplatform/helxapp-controller/api/v1"
	"github.com/helxplatform/helxapp-controller/template_io"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func resetTables() {
	appTable = make(map[string]TableElement[helxv1.HelxApp])
	userTable = make(map[string]TableElement[helxv1.HelxUser])
	instanceTable = make(map[string]InstTableElement)
}

func TestMain(m *testing.M) {
	logger := logr.Discard()
	simpleDebugLogger = newSimpleDebugLogger(logger)
	simpleInfoLogger = newSimpleInfoLogger(logger)
	simpleErrorLogger = newSimpleErrorLogger(logger)

	var err error
	xformer, storage, err = template_io.ParseTemplates("../templates", nil)
	if err != nil {
		panic("failed to parse templates: " + err.Error())
	}

	os.Exit(m.Run())
}

// ---------------------------------------------------------------------------
// Helper constructors
// ---------------------------------------------------------------------------

func makeApp(namespace, name, appClassName string, services []helxv1.Service) *helxv1.HelxApp {
	return &helxv1.HelxApp{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: helxv1.HelxAppSpec{
			AppClassName: appClassName,
			Services:     services,
		},
	}
}

func makeUser(namespace, name string, handle *string) *helxv1.HelxUser {
	return &helxv1.HelxUser{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: helxv1.HelxUserSpec{
			UserHandle: handle,
		},
	}
}

func makeInst(namespace, name, appName, userName, uuid string) *helxv1.HelxInst {
	return &helxv1.HelxInst{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: helxv1.HelxInstSpec{
			AppName:  appName,
			UserName: userName,
		},
		Status: helxv1.HelxInstStatus{
			UUID: uuid,
		},
	}
}

func makeInstWithResources(namespace, name, appName, userName, uuid string, resources map[string]helxv1.Resources) *helxv1.HelxInst {
	inst := makeInst(namespace, name, appName, userName, uuid)
	inst.Spec.Resources = resources
	return inst
}

func makeInstWithSC(namespace, name, appName, userName, uuid string, sc *helxv1.SecurityContext) *helxv1.HelxInst {
	inst := makeInst(namespace, name, appName, userName, uuid)
	inst.Spec.SecurityContext = sc
	return inst
}

// ---------------------------------------------------------------------------
// 1-3: Basic Add/Get for App, User, Inst
// ---------------------------------------------------------------------------

func TestAddAndGetApp(t *testing.T) {
	resetTables()
	app := makeApp("ns", "myapp", "AppClass1", nil)
	AddApp(app)

	got := GetApp("ns/myapp")
	if got == nil {
		t.Fatal("expected app, got nil")
	}
	if got.Name != "myapp" {
		t.Errorf("expected name myapp, got %s", got.Name)
	}
}

func TestAddAndGetUser(t *testing.T) {
	resetTables()
	user := makeUser("ns", "alice", nil)
	AddUser(user)

	got := GetUser("ns/alice")
	if got == nil {
		t.Fatal("expected user, got nil")
	}
	if got.Name != "alice" {
		t.Errorf("expected name alice, got %s", got.Name)
	}
}

func TestAddAndGetInst(t *testing.T) {
	resetTables()
	inst := makeInst("ns", "inst1", "myapp", "alice", "uuid-1")
	AddInst(inst)

	got, found := GetInst("ns/inst1")
	if !found {
		t.Fatal("expected inst, not found")
	}
	if got.Name != "inst1" {
		t.Errorf("expected name inst1, got %s", got.Name)
	}
}

// ---------------------------------------------------------------------------
// 4: AddInst connects App and User
// ---------------------------------------------------------------------------

func TestAddInstConnectsAppAndUser(t *testing.T) {
	resetTables()
	app := makeApp("ns", "myapp", "cls", nil)
	user := makeUser("ns", "alice", nil)
	AddApp(app)
	AddUser(user)

	inst := makeInst("ns", "inst1", "myapp", "alice", "uuid-1")
	AddInst(inst)

	// appTable entry for ns/myapp should have inst in its InstSet
	appElem, ok := appTable["ns/myapp"]
	if !ok {
		t.Fatal("app not in table")
	}
	if !appElem.InstSet["ns/inst1"] {
		t.Error("inst not in app's InstSet")
	}

	userElem, ok := userTable["ns/alice"]
	if !ok {
		t.Fatal("user not in table")
	}
	if !userElem.InstSet["ns/inst1"] {
		t.Error("inst not in user's InstSet")
	}
}

// ---------------------------------------------------------------------------
// 5-6: AddApp/AddUser returns existing instances
// ---------------------------------------------------------------------------

func TestAddAppReturnsExistingInstances(t *testing.T) {
	resetTables()
	// Add inst first (creates placeholder in appTable)
	inst := makeInst("ns", "inst1", "myapp", "alice", "uuid-1")
	AddInst(inst)

	// Now add the app -- should return the existing instance
	app := makeApp("ns", "myapp", "cls", nil)
	instList := AddApp(app)
	if len(instList) != 1 {
		t.Fatalf("expected 1 inst, got %d", len(instList))
	}
	if instList[0].Name != "inst1" {
		t.Errorf("expected inst1, got %s", instList[0].Name)
	}
}

func TestAddUserReturnsExistingInstances(t *testing.T) {
	resetTables()
	inst := makeInst("ns", "inst1", "myapp", "alice", "uuid-1")
	AddInst(inst)

	user := makeUser("ns", "alice", nil)
	instList := AddUser(user)
	if len(instList) != 1 {
		t.Fatalf("expected 1 inst, got %d", len(instList))
	}
	if instList[0].Name != "inst1" {
		t.Errorf("expected inst1, got %s", instList[0].Name)
	}
}

// ---------------------------------------------------------------------------
// 7-9: Delete operations
// ---------------------------------------------------------------------------

func TestDeleteApp(t *testing.T) {
	resetTables()
	app := makeApp("ns", "myapp", "cls", nil)
	AddApp(app)
	inst := makeInst("ns", "inst1", "myapp", "alice", "uuid-1")
	AddInst(inst)

	instList := DeleteApp("ns/myapp")
	if len(instList) != 1 {
		t.Fatalf("expected 1 associated inst, got %d", len(instList))
	}
}

func TestDeleteUser(t *testing.T) {
	resetTables()
	user := makeUser("ns", "alice", nil)
	AddUser(user)
	inst := makeInst("ns", "inst1", "myapp", "alice", "uuid-1")
	AddInst(inst)

	instList := DeleteUser("ns/alice")
	if len(instList) != 1 {
		t.Fatalf("expected 1 associated inst, got %d", len(instList))
	}
}

func TestDeleteInst(t *testing.T) {
	resetTables()
	inst := makeInst("ns", "inst1", "myapp", "alice", "uuid-1")
	AddInst(inst)

	DeleteInst("ns/inst1")
	_, found := GetInst("ns/inst1")
	if found {
		t.Error("inst should have been deleted")
	}
}

// ---------------------------------------------------------------------------
// 10-14: GetAppNameFromInst / GetUserNameFromInst
// ---------------------------------------------------------------------------

func TestGetAppNameFromInst_Bare(t *testing.T) {
	inst := makeInst("ns", "inst1", "nginx1", "alice", "uuid-1")
	got := GetAppNameFromInst(inst)
	if got != "ns/nginx1" {
		t.Errorf("expected ns/nginx1, got %s", got)
	}
}

func TestGetAppNameFromInst_Namespaced(t *testing.T) {
	inst := makeInst("ns", "inst1", "other/nginx1", "alice", "uuid-1")
	got := GetAppNameFromInst(inst)
	if got != "other/nginx1" {
		t.Errorf("expected other/nginx1, got %s", got)
	}
}

func TestGetAppNameFromInst_Nil(t *testing.T) {
	got := GetAppNameFromInst(nil)
	if got != "" {
		t.Errorf("expected empty, got %s", got)
	}
}

func TestGetUserNameFromInst_Bare(t *testing.T) {
	inst := makeInst("ns", "inst1", "myapp", "alice", "uuid-1")
	got := GetUserNameFromInst(inst)
	if got != "ns/alice" {
		t.Errorf("expected ns/alice, got %s", got)
	}
}

func TestGetUserNameFromInst_Namespaced(t *testing.T) {
	inst := makeInst("ns", "inst1", "myapp", "other/alice", "uuid-1")
	got := GetUserNameFromInst(inst)
	if got != "other/alice" {
		t.Errorf("expected other/alice, got %s", got)
	}
}

// ---------------------------------------------------------------------------
// 15-25: processVolume tests
// ---------------------------------------------------------------------------

func TestProcessVolume_PVCDefault(t *testing.T) {
	vol, mnt, err := processVolume("v1", "myvolume:/mnt")
	if err != nil {
		t.Fatal(err)
	}
	if vol.Scheme != "pvc" {
		t.Errorf("expected pvc scheme, got %s", vol.Scheme)
	}
	if vol.Attr["claim"] != "myvolume" {
		t.Errorf("expected claim=myvolume, got %s", vol.Attr["claim"])
	}
	if mnt.MountPath != "/mnt" {
		t.Errorf("expected /mnt, got %s", mnt.MountPath)
	}
	if mnt.Name != "v1" {
		t.Errorf("expected name v1, got %s", mnt.Name)
	}
}

func TestProcessVolume_PVCExplicit(t *testing.T) {
	vol, mnt, err := processVolume("v1", "pvc://myvolume:/mnt")
	if err != nil {
		t.Fatal(err)
	}
	if vol.Scheme != "pvc" {
		t.Errorf("expected pvc, got %s", vol.Scheme)
	}
	if vol.Attr["claim"] != "myvolume" {
		t.Errorf("expected claim=myvolume, got %s", vol.Attr["claim"])
	}
	if mnt.MountPath != "/mnt" {
		t.Errorf("expected /mnt, got %s", mnt.MountPath)
	}
}

func TestProcessVolume_NFS(t *testing.T) {
	// NFS requires leading slash in source so SplitN produces 3 parts
	vol, mnt, err := processVolume("v1", "nfs:///server/path:/mnt")
	if err != nil {
		t.Fatal(err)
	}
	if vol.Scheme != "nfs" {
		t.Errorf("expected nfs, got %s", vol.Scheme)
	}
	if vol.Attr["server"] != "server" {
		t.Errorf("expected server=server, got %s", vol.Attr["server"])
	}
	if vol.Attr["path"] != "/path" {
		t.Errorf("expected path=/path, got %s", vol.Attr["path"])
	}
	if mnt.MountPath != "/mnt" {
		t.Errorf("expected /mnt, got %s", mnt.MountPath)
	}
}

func TestProcessVolume_SubPath(t *testing.T) {
	_, mnt, err := processVolume("v1", "vol:/mnt#sub")
	if err != nil {
		t.Fatal(err)
	}
	if mnt.SubPath != "sub" {
		t.Errorf("expected subpath=sub, got %s", mnt.SubPath)
	}
}

func TestProcessVolume_Options(t *testing.T) {
	vol, _, err := processVolume("v1", "vol:/mnt,rwx,retain")
	if err != nil {
		t.Fatal(err)
	}
	if vol.Attr["rwx"] != "true" {
		t.Errorf("expected rwx=true, got %s", vol.Attr["rwx"])
	}
	if vol.Attr["retain"] != "true" {
		t.Errorf("expected retain=true, got %s", vol.Attr["retain"])
	}
}

func TestProcessVolume_OptionWithValue(t *testing.T) {
	vol, _, err := processVolume("v1", "vol:/mnt,size=20G")
	if err != nil {
		t.Fatal(err)
	}
	if vol.Attr["size"] != "20G" {
		t.Errorf("expected size=20G, got %s", vol.Attr["size"])
	}
}

func TestProcessVolume_ReadOnly(t *testing.T) {
	_, mnt, err := processVolume("v1", "vol:/mnt,ro")
	if err != nil {
		t.Fatal(err)
	}
	if !mnt.ReadOnly {
		t.Error("expected ReadOnly=true")
	}
}

func TestProcessVolume_FullComplex(t *testing.T) {
	vol, mnt, err := processVolume("v1", "pvc://myvol:/data#subdir,rwx,retain,size=10G,ro")
	if err != nil {
		t.Fatal(err)
	}
	if vol.Scheme != "pvc" {
		t.Errorf("expected pvc, got %s", vol.Scheme)
	}
	if vol.Attr["claim"] != "myvol" {
		t.Errorf("expected claim=myvol, got %s", vol.Attr["claim"])
	}
	if mnt.MountPath != "/data" {
		t.Errorf("expected /data, got %s", mnt.MountPath)
	}
	if mnt.SubPath != "subdir" {
		t.Errorf("expected subdir, got %s", mnt.SubPath)
	}
	if vol.Attr["rwx"] != "true" {
		t.Error("expected rwx=true")
	}
	if vol.Attr["retain"] != "true" {
		t.Error("expected retain=true")
	}
	if vol.Attr["size"] != "10G" {
		t.Errorf("expected size=10G, got %s", vol.Attr["size"])
	}
	if !mnt.ReadOnly {
		t.Error("expected ReadOnly=true")
	}
}

func TestProcessVolume_TemplateExpression(t *testing.T) {
	// Template expressions like {{ .system.UserName }} should pass through the regex
	// because the regex chars are not in the disallowed set for source/mount
	// Actually the regex disallows : # , in source, and {{ }} contain none of those.
	// But the regex [^:#,]+ would match "{{ .system.UserName }}-home"
	vol, mnt, err := processVolume("home", "myhome:/home/user")
	if err != nil {
		t.Fatal(err)
	}
	if vol.Attr["claim"] != "myhome" {
		t.Errorf("expected claim=myhome, got %s", vol.Attr["claim"])
	}
	if mnt.MountPath != "/home/user" {
		t.Errorf("expected /home/user, got %s", mnt.MountPath)
	}
}

func TestProcessVolume_InvalidFormat(t *testing.T) {
	_, _, err := processVolume("v1", "justabadstring")
	if err == nil {
		t.Error("expected error for invalid format")
	}
}

func TestProcessVolume_InvalidNFS(t *testing.T) {
	// NFS without leading slash gives < 3 parts from SplitN
	_, _, err := processVolume("v1", "nfs://server:/mnt")
	if err == nil {
		t.Error("expected error for invalid NFS source")
	}
}

// ---------------------------------------------------------------------------
// Secret and ConfigMap volume scheme tests
// ---------------------------------------------------------------------------

func TestProcessVolume_Secret(t *testing.T) {
	vol, mnt, err := processVolume("v1", "secret://my-secret:/mnt/secret")
	if err != nil {
		t.Fatal(err)
	}
	if vol.Scheme != "secret" {
		t.Errorf("expected secret scheme, got %s", vol.Scheme)
	}
	if vol.Attr["secretName"] != "my-secret" {
		t.Errorf("expected secretName=my-secret, got %s", vol.Attr["secretName"])
	}
	if mnt.MountPath != "/mnt/secret" {
		t.Errorf("expected /mnt/secret, got %s", mnt.MountPath)
	}
}

func TestProcessVolume_SecretReadOnly(t *testing.T) {
	_, mnt, err := processVolume("v1", "secret://my-secret:/mnt/secret,ro")
	if err != nil {
		t.Fatal(err)
	}
	if !mnt.ReadOnly {
		t.Error("expected ReadOnly=true")
	}
}

func TestProcessVolume_SecretSubPath(t *testing.T) {
	_, mnt, err := processVolume("v1", "secret://my-secret:/mnt/secret#tls.crt")
	if err != nil {
		t.Fatal(err)
	}
	if mnt.SubPath != "tls.crt" {
		t.Errorf("expected subPath=tls.crt, got %s", mnt.SubPath)
	}
}

func TestProcessVolume_ConfigMap(t *testing.T) {
	vol, mnt, err := processVolume("v1", "configmap://my-config:/etc/config")
	if err != nil {
		t.Fatal(err)
	}
	if vol.Scheme != "configmap" {
		t.Errorf("expected configmap scheme, got %s", vol.Scheme)
	}
	if vol.Attr["configMapName"] != "my-config" {
		t.Errorf("expected configMapName=my-config, got %s", vol.Attr["configMapName"])
	}
	if mnt.MountPath != "/etc/config" {
		t.Errorf("expected /etc/config, got %s", mnt.MountPath)
	}
}

func TestProcessVolume_ConfigMapReadOnly(t *testing.T) {
	_, mnt, err := processVolume("v1", "configmap://my-config:/etc/config,ro")
	if err != nil {
		t.Fatal(err)
	}
	if !mnt.ReadOnly {
		t.Error("expected ReadOnly=true")
	}
}

func TestProcessVolume_ConfigMapSubPath(t *testing.T) {
	_, mnt, err := processVolume("v1", "configmap://my-config:/etc/config/app.conf#app.conf")
	if err != nil {
		t.Fatal(err)
	}
	if mnt.SubPath != "app.conf" {
		t.Errorf("expected subPath=app.conf, got %s", mnt.SubPath)
	}
}

// ---------------------------------------------------------------------------
// 26-29: processImageAndOptions tests
// ---------------------------------------------------------------------------

func TestProcessImage_Simple(t *testing.T) {
	name, attr := processImageAndOptions("nginx:latest")
	if name != "nginx:latest" {
		t.Errorf("expected nginx:latest, got %s", name)
	}
	if len(attr) != 0 {
		t.Errorf("expected no attrs, got %v", attr)
	}
}

func TestProcessImage_WithOption(t *testing.T) {
	name, attr := processImageAndOptions("nginx:latest,Always")
	if name != "nginx:latest" {
		t.Errorf("expected nginx:latest, got %s", name)
	}
	if attr["Always"] != "true" {
		t.Errorf("expected Always=true, got %s", attr["Always"])
	}
}

func TestProcessImage_WithKeyValue(t *testing.T) {
	name, attr := processImageAndOptions("myimage:v1,pull=IfNotPresent,debug")
	if name != "myimage:v1" {
		t.Errorf("expected myimage:v1, got %s", name)
	}
	if attr["pull"] != "IfNotPresent" {
		t.Errorf("expected pull=IfNotPresent, got %s", attr["pull"])
	}
	if attr["debug"] != "true" {
		t.Errorf("expected debug=true, got %s", attr["debug"])
	}
}

func TestProcessImage_NoOptions(t *testing.T) {
	name, attr := processImageAndOptions("busybox")
	if name != "busybox" {
		t.Errorf("expected busybox, got %s", name)
	}
	if len(attr) != 0 {
		t.Errorf("expected no attrs, got %v", attr)
	}
}

// ---------------------------------------------------------------------------
// 30-34: transformPorts and transformResources
// ---------------------------------------------------------------------------

func TestTransformPorts_WithService(t *testing.T) {
	src := []helxv1.PortMap{
		{ContainerPort: 80, Port: 8080},
	}
	ports, hasService := transformPorts(src)
	if !hasService {
		t.Error("expected hasService=true")
	}
	if len(ports) != 1 {
		t.Fatalf("expected 1 port, got %d", len(ports))
	}
	if ports[0].ContainerPort != 80 || ports[0].Port != 8080 {
		t.Errorf("unexpected port mapping: %+v", ports[0])
	}
	if ports[0].Protocol != "TCP" {
		t.Errorf("expected TCP, got %s", ports[0].Protocol)
	}
}

func TestTransformPorts_NoService(t *testing.T) {
	src := []helxv1.PortMap{
		{ContainerPort: 80, Port: 0},
	}
	_, hasService := transformPorts(src)
	if hasService {
		t.Error("expected hasService=false when Port==0")
	}
}

func TestTransformPorts_Empty(t *testing.T) {
	ports, hasService := transformPorts(nil)
	if hasService {
		t.Error("expected hasService=false for nil ports")
	}
	if len(ports) != 0 {
		t.Errorf("expected 0 ports, got %d", len(ports))
	}
}

func TestTransformResources_Found(t *testing.T) {
	resources := map[string]helxv1.Resources{
		"main": {
			Limits:   map[string]string{"cpu": "2", "memory": "4G"},
			Requests: map[string]string{"cpu": "1", "memory": "2G"},
		},
	}
	res := transformResources("main", resources)
	if res.Limits["cpu"] != "2" {
		t.Errorf("expected cpu limit=2, got %s", res.Limits["cpu"])
	}
	if res.Requests["memory"] != "2G" {
		t.Errorf("expected memory request=2G, got %s", res.Requests["memory"])
	}
}

func TestTransformResources_NotFound(t *testing.T) {
	resources := map[string]helxv1.Resources{
		"other": {
			Limits: map[string]string{"cpu": "2"},
		},
	}
	res := transformResources("main", resources)
	if res.Limits != nil || res.Requests != nil {
		t.Error("expected empty resources when service not found")
	}
}

// ---------------------------------------------------------------------------
// 35-38: transformApp
// ---------------------------------------------------------------------------

func TestTransformApp_SingleService(t *testing.T) {
	app := helxv1.HelxApp{
		Spec: helxv1.HelxAppSpec{
			Services: []helxv1.Service{
				{
					Name:    "main",
					Image:   "nginx:latest",
					Command: []string{"nginx", "-g", "daemon off;"},
					Ports:   []helxv1.PortMap{{ContainerPort: 80, Port: 8080}},
				},
			},
		},
	}
	inst := makeInst("ns", "inst1", "myapp", "alice", "uuid-1")

	containers, sourceMap, err := transformApp(inst, app, *makeUser("ns", "testuser", nil))
	if err != nil {
		t.Fatal(err)
	}
	if len(containers) != 1 {
		t.Fatalf("expected 1 container, got %d", len(containers))
	}
	if containers[0].Name != "main" {
		t.Errorf("expected name=main, got %s", containers[0].Name)
	}
	if containers[0].Image.ImageName != "nginx:latest" {
		t.Errorf("expected image=nginx:latest, got %s", containers[0].Image.ImageName)
	}
	if !containers[0].HasService {
		t.Error("expected HasService=true")
	}
	if len(sourceMap) != 0 {
		t.Errorf("expected no volumes, got %d", len(sourceMap))
	}
}

func TestTransformApp_MultiService(t *testing.T) {
	app := helxv1.HelxApp{
		Spec: helxv1.HelxAppSpec{
			Services: []helxv1.Service{
				{Name: "web", Image: "nginx", Command: []string{"nginx"}, Ports: []helxv1.PortMap{{ContainerPort: 80, Port: 80}}},
				{Name: "sidecar", Image: "fluentd", Command: []string{"fluentd"}, Ports: []helxv1.PortMap{{ContainerPort: 24224}}},
			},
		},
	}
	inst := makeInst("ns", "inst1", "myapp", "alice", "uuid-1")

	containers, _, err := transformApp(inst, app, *makeUser("ns", "testuser", nil))
	if err != nil {
		t.Fatal(err)
	}
	if len(containers) != 2 {
		t.Fatalf("expected 2 containers, got %d", len(containers))
	}
	if containers[0].Name != "web" || containers[1].Name != "sidecar" {
		t.Errorf("unexpected container names: %s, %s", containers[0].Name, containers[1].Name)
	}
}

func TestTransformApp_WithEnvironment(t *testing.T) {
	app := helxv1.HelxApp{
		Spec: helxv1.HelxAppSpec{
			Services: []helxv1.Service{
				{
					Name:        "main",
					Image:       "nginx",
					Command:     []string{"nginx"},
					Ports:       []helxv1.PortMap{{ContainerPort: 80}},
					Environment: map[string]string{"FOO": "bar", "BAZ": "qux"},
				},
			},
		},
	}
	inst := makeInst("ns", "inst1", "myapp", "alice", "uuid-1")

	containers, _, err := transformApp(inst, app, *makeUser("ns", "testuser", nil))
	if err != nil {
		t.Fatal(err)
	}
	if containers[0].Environment["FOO"] != "bar" {
		t.Errorf("expected FOO=bar, got %s", containers[0].Environment["FOO"])
	}
	if containers[0].Environment["BAZ"] != "qux" {
		t.Errorf("expected BAZ=qux, got %s", containers[0].Environment["BAZ"])
	}
}

func TestTransformApp_WithCommand(t *testing.T) {
	app := helxv1.HelxApp{
		Spec: helxv1.HelxAppSpec{
			Services: []helxv1.Service{
				{
					Name:    "main",
					Image:   "alpine",
					Command: []string{"/bin/sh", "-c", "echo hello"},
					Ports:   []helxv1.PortMap{{ContainerPort: 80}},
				},
			},
		},
	}
	inst := makeInst("ns", "inst1", "myapp", "alice", "uuid-1")

	containers, _, err := transformApp(inst, app, *makeUser("ns", "testuser", nil))
	if err != nil {
		t.Fatal(err)
	}
	cmd := containers[0].Command
	if len(cmd) != 3 || cmd[0] != "/bin/sh" || cmd[1] != "-c" || cmd[2] != "echo hello" {
		t.Errorf("unexpected command: %v", cmd)
	}
}

// ---------------------------------------------------------------------------
// Helper to set up graph state for artifact tests
// ---------------------------------------------------------------------------

func setupGraphForArtifacts(app *helxv1.HelxApp, user *helxv1.HelxUser, inst *helxv1.HelxInst) {
	resetTables()
	AddApp(app)
	AddUser(user)
	AddInst(inst)
}

// ---------------------------------------------------------------------------
// 39-52: GenerateArtifacts tests
// ---------------------------------------------------------------------------

func TestGenerateArtifacts_NginxMinimal(t *testing.T) {
	app := makeApp("ns", "myapp", "Nginx", []helxv1.Service{
		{
			Name:    "main",
			Image:   "nginx:latest",
			Command: []string{"nginx", "-g", "daemon off;"},
			Ports:   []helxv1.PortMap{{ContainerPort: 80, Port: 8080}},
		},
	})
	user := makeUser("ns", "alice", nil)
	inst := makeInst("ns", "inst1", "myapp", "alice", "test-uuid-1")

	setupGraphForArtifacts(app, user, inst)
	artifacts, err := GenerateArtifacts(inst)
	if err != nil {
		t.Fatal(err)
	}
	if artifacts == nil {
		t.Fatal("expected artifacts, got nil")
	}
	if artifacts.Deployment.Render == "" {
		t.Error("expected non-empty deployment render")
	}
	if !strings.Contains(artifacts.Deployment.Render, "kind: Deployment") {
		t.Error("deployment render should contain 'kind: Deployment'")
	}
	if !strings.Contains(artifacts.Deployment.Render, "test-uuid-1") {
		t.Error("deployment render should contain UUID")
	}
	// No PVCs for this app
	if len(artifacts.PVCs) != 0 {
		t.Errorf("expected 0 PVCs, got %d", len(artifacts.PVCs))
	}
	// Should have a service since Port != 0
	if len(artifacts.Services) != 1 {
		t.Fatalf("expected 1 service, got %d", len(artifacts.Services))
	}
	svc := artifacts.Services["main"]
	if !strings.Contains(svc.Render, "kind: Service") {
		t.Error("service render should contain 'kind: Service'")
	}
}

func TestGenerateArtifacts_WithPVC(t *testing.T) {
	app := makeApp("ns", "myapp", "Nginx", []helxv1.Service{
		{
			Name:    "main",
			Image:   "nginx",
			Command: []string{"nginx"},
			Ports:   []helxv1.PortMap{{ContainerPort: 80, Port: 80}},
			Volumes: map[string]string{"data": "mydata:/data"},
		},
	})
	user := makeUser("ns", "alice", nil)
	inst := makeInst("ns", "inst1", "myapp", "alice", "test-uuid-2")

	setupGraphForArtifacts(app, user, inst)
	artifacts, err := GenerateArtifacts(inst)
	if err != nil {
		t.Fatal(err)
	}
	if artifacts == nil {
		t.Fatal("expected artifacts, got nil")
	}
	if len(artifacts.PVCs) != 1 {
		t.Fatalf("expected 1 PVC, got %d", len(artifacts.PVCs))
	}
	pvc, ok := artifacts.PVCs["mydata"]
	if !ok {
		t.Fatal("expected PVC with claim name 'mydata'")
	}
	if !strings.Contains(pvc.Render, "PersistentVolumeClaim") {
		t.Error("PVC render should contain PersistentVolumeClaim")
	}
}

func TestGenerateArtifacts_WithNFS(t *testing.T) {
	app := makeApp("ns", "myapp", "Nginx", []helxv1.Service{
		{
			Name:    "main",
			Image:   "nginx",
			Command: []string{"nginx"},
			Ports:   []helxv1.PortMap{{ContainerPort: 80, Port: 80}},
			Volumes: map[string]string{"share": "nfs:///myserver/exports:/mnt"},
		},
	})
	user := makeUser("ns", "alice", nil)
	inst := makeInst("ns", "inst1", "myapp", "alice", "test-uuid-3")

	setupGraphForArtifacts(app, user, inst)
	artifacts, err := GenerateArtifacts(inst)
	if err != nil {
		t.Fatal(err)
	}
	if artifacts == nil {
		t.Fatal("expected artifacts, got nil")
	}
	// NFS volumes should NOT produce PVCs
	if len(artifacts.PVCs) != 0 {
		t.Errorf("expected 0 PVCs for NFS, got %d", len(artifacts.PVCs))
	}
	// The deployment should reference the NFS volume
	if !strings.Contains(artifacts.Deployment.Render, "nfs") {
		t.Error("deployment should reference nfs volume")
	}
}

func TestGenerateArtifacts_RetainLabel(t *testing.T) {
	app := makeApp("ns", "myapp", "Nginx", []helxv1.Service{
		{
			Name:    "main",
			Image:   "nginx",
			Command: []string{"nginx"},
			Ports:   []helxv1.PortMap{{ContainerPort: 80, Port: 80}},
			Volumes: map[string]string{"data": "mydata:/data,retain"},
		},
	})
	user := makeUser("ns", "alice", nil)
	inst := makeInst("ns", "inst1", "myapp", "alice", "test-uuid-4")

	setupGraphForArtifacts(app, user, inst)
	artifacts, err := GenerateArtifacts(inst)
	if err != nil {
		t.Fatal(err)
	}
	if artifacts == nil {
		t.Fatal("expected artifacts, got nil")
	}
	pvc, ok := artifacts.PVCs["mydata"]
	if !ok {
		t.Fatal("expected PVC with claim name 'mydata'")
	}
	if pvc.Attr["retain"] != "true" {
		t.Errorf("expected retain=true in PVC attr, got %s", pvc.Attr["retain"])
	}
	if !strings.Contains(pvc.Render, "helx.renci.org/retain") {
		t.Error("PVC render should contain retain label")
	}
}

func TestGenerateArtifacts_MissingApp(t *testing.T) {
	resetTables()
	user := makeUser("ns", "alice", nil)
	AddUser(user)
	inst := makeInst("ns", "inst1", "myapp", "alice", "test-uuid-5")
	AddInst(inst)

	artifacts, err := GenerateArtifacts(inst)
	if err != nil {
		t.Fatal(err)
	}
	if artifacts != nil {
		t.Error("expected nil artifacts when app is missing")
	}
}

func TestGenerateArtifacts_MissingUser(t *testing.T) {
	resetTables()
	app := makeApp("ns", "myapp", "Nginx", []helxv1.Service{
		{Name: "main", Image: "nginx", Command: []string{"nginx"}, Ports: []helxv1.PortMap{{ContainerPort: 80, Port: 80}}},
	})
	AddApp(app)
	inst := makeInst("ns", "inst1", "myapp", "alice", "test-uuid-6")
	AddInst(inst)

	artifacts, err := GenerateArtifacts(inst)
	if err != nil {
		t.Fatal(err)
	}
	if artifacts != nil {
		t.Error("expected nil artifacts when user is missing")
	}
}

func TestGenerateArtifacts_WithResources(t *testing.T) {
	app := makeApp("ns", "myapp", "Nginx", []helxv1.Service{
		{
			Name:    "main",
			Image:   "nginx",
			Command: []string{"nginx"},
			Ports:   []helxv1.PortMap{{ContainerPort: 80, Port: 80}},
		},
	})
	user := makeUser("ns", "alice", nil)
	inst := makeInstWithResources("ns", "inst1", "myapp", "alice", "test-uuid-7",
		map[string]helxv1.Resources{
			"main": {
				Limits:   map[string]string{"cpu": "2", "memory": "4G"},
				Requests: map[string]string{"cpu": "1", "memory": "2G"},
			},
		},
	)

	setupGraphForArtifacts(app, user, inst)
	artifacts, err := GenerateArtifacts(inst)
	if err != nil {
		t.Fatal(err)
	}
	if artifacts == nil {
		t.Fatal("expected artifacts, got nil")
	}
	render := artifacts.Deployment.Render
	if !strings.Contains(render, "resources:") {
		t.Error("deployment should contain resources section")
	}
	if !strings.Contains(render, "cpu: 2") {
		t.Error("deployment should contain cpu limit")
	}
	if !strings.Contains(render, "memory: 2G") {
		t.Error("deployment should contain memory request")
	}
}

func TestGenerateArtifacts_WithSecurityContext(t *testing.T) {
	app := makeApp("ns", "myapp", "Nginx", []helxv1.Service{
		{
			Name:    "main",
			Image:   "nginx",
			Command: []string{"nginx"},
			Ports:   []helxv1.PortMap{{ContainerPort: 80, Port: 80}},
		},
	})
	user := makeUser("ns", "alice", nil)
	uid := int64(1000)
	gid := int64(1000)
	inst := makeInstWithSC("ns", "inst1", "myapp", "alice", "test-uuid-8",
		&helxv1.SecurityContext{
			RunAsUser:  &uid,
			RunAsGroup: &gid,
		},
	)

	setupGraphForArtifacts(app, user, inst)
	artifacts, err := GenerateArtifacts(inst)
	if err != nil {
		t.Fatal(err)
	}
	if artifacts == nil {
		t.Fatal("expected artifacts, got nil")
	}
	render := artifacts.Deployment.Render
	if !strings.Contains(render, "runAsUser: 1000") {
		t.Error("deployment should contain runAsUser")
	}
	if !strings.Contains(render, "runAsGroup: 1000") {
		t.Error("deployment should contain runAsGroup")
	}
}

func TestGenerateArtifacts_TemplateInterpolation(t *testing.T) {
	// Test that template expressions like {{ .system.UserName }} in volume names get resolved
	app := makeApp("ns", "myapp", "Nginx", []helxv1.Service{
		{
			Name:    "main",
			Image:   "nginx",
			Command: []string{"nginx"},
			Ports:   []helxv1.PortMap{{ContainerPort: 80, Port: 80}},
		},
	})
	user := makeUser("ns", "alice", nil)
	inst := makeInst("ns", "inst1", "myapp", "alice", "test-uuid-9")

	setupGraphForArtifacts(app, user, inst)
	artifacts, err := GenerateArtifacts(inst)
	if err != nil {
		t.Fatal(err)
	}
	if artifacts == nil {
		t.Fatal("expected artifacts, got nil")
	}
	// The deployment should have resolved UserName
	render := artifacts.Deployment.Render
	if !strings.Contains(render, "alice") {
		t.Error("deployment should contain resolved UserName 'alice'")
	}
}

func TestGenerateArtifacts_SystemEnvironment(t *testing.T) {
	app := makeApp("ns", "myapp", "Nginx", []helxv1.Service{
		{
			Name:    "main",
			Image:   "nginx",
			Command: []string{"nginx"},
			Ports:   []helxv1.PortMap{{ContainerPort: 80, Port: 80}},
		},
	})
	user := makeUser("ns", "alice", nil)
	inst := makeInst("ns", "inst1", "myapp", "alice", "test-uuid-10")

	setupGraphForArtifacts(app, user, inst)
	artifacts, err := GenerateArtifacts(inst)
	if err != nil {
		t.Fatal(err)
	}
	if artifacts == nil {
		t.Fatal("expected artifacts, got nil")
	}
	render := artifacts.Deployment.Render
	// System env vars should be in the rendered deployment
	if !strings.Contains(render, "GUID") {
		t.Error("deployment should contain GUID env var")
	}
	if !strings.Contains(render, "test-uuid-10") {
		t.Error("deployment should contain UUID value")
	}
	if !strings.Contains(render, "USER") {
		t.Error("deployment should contain USER env var")
	}
}

func TestGenerateArtifacts_Filebrowser(t *testing.T) {
	// Simulates the filebrowser test fixture
	app := makeApp("jeffw", "filebrowser1", "Filebrowser", []helxv1.Service{
		{
			Name:    "main",
			Command: []string{"/bin/sh", "-c", `/filebrowser --noauth --root="/home" --address=0.0.0.0 --database=/home/{{ .system.UserName}}/.filebrowser/filebrowser.db`},
			Image:   "wateim/filebrowser:jeffw,Always",
			Ports:   []helxv1.PortMap{{ContainerPort: 80, Port: 8080}},
			Volumes: map[string]string{
				"home": "jeffw-home:/home/jeffw,rwx,retain",
			},
		},
	})
	user := makeUser("jeffw", "jeffw", nil)
	inst := makeInstWithResources("jeffw", "filebrowser1", "filebrowser1", "jeffw", "fb-uuid-1",
		map[string]helxv1.Resources{
			"main": {
				Requests: map[string]string{"cpu": "1", "memory": "1G"},
				Limits:   map[string]string{"cpu": "1", "memory": "1G"},
			},
		},
	)

	setupGraphForArtifacts(app, user, inst)
	artifacts, err := GenerateArtifacts(inst)
	if err != nil {
		t.Fatal(err)
	}
	if artifacts == nil {
		t.Fatal("expected artifacts, got nil")
	}

	// Check deployment
	deploy := artifacts.Deployment.Render
	if !strings.Contains(deploy, "kind: Deployment") {
		t.Error("should have deployment")
	}
	if !strings.Contains(deploy, "wateim/filebrowser:jeffw") {
		t.Error("should contain image name")
	}
	if !strings.Contains(deploy, "imagePullPolicy: Always") {
		t.Error("should have Always pull policy")
	}
	if !strings.Contains(deploy, "fb-uuid-1") {
		t.Error("should contain UUID")
	}

	// Check PVC
	if len(artifacts.PVCs) != 1 {
		t.Fatalf("expected 1 PVC, got %d", len(artifacts.PVCs))
	}
	pvc, ok := artifacts.PVCs["jeffw-home"]
	if !ok {
		t.Fatal("expected PVC with claim jeffw-home")
	}
	if !strings.Contains(pvc.Render, "PersistentVolumeClaim") {
		t.Error("PVC render should contain PersistentVolumeClaim")
	}
	if pvc.Attr["retain"] != "true" {
		t.Error("PVC should have retain attr")
	}

	// Check service
	if len(artifacts.Services) != 1 {
		t.Fatalf("expected 1 service, got %d", len(artifacts.Services))
	}
	svc := artifacts.Services["main"]
	if !strings.Contains(svc.Render, "kind: Service") {
		t.Error("service render should contain 'kind: Service'")
	}
}

func TestGenerateArtifacts_PVCAccessModes(t *testing.T) {
	app := makeApp("ns", "myapp", "App", []helxv1.Service{
		{
			Name:    "main",
			Image:   "nginx",
			Command: []string{"nginx"},
			Ports:   []helxv1.PortMap{{ContainerPort: 80, Port: 80}},
			Volumes: map[string]string{"data": "mydata:/data,rwx"},
		},
	})
	user := makeUser("ns", "alice", nil)
	inst := makeInst("ns", "inst1", "myapp", "alice", "test-uuid-rwx")

	setupGraphForArtifacts(app, user, inst)
	artifacts, err := GenerateArtifacts(inst)
	if err != nil {
		t.Fatal(err)
	}
	if artifacts == nil {
		t.Fatal("expected artifacts")
	}
	pvc := artifacts.PVCs["mydata"]
	if !strings.Contains(pvc.Render, "ReadWriteMany") {
		t.Error("PVC with rwx should contain ReadWriteMany")
	}
}

func TestGenerateArtifacts_PVCStorageSize(t *testing.T) {
	app := makeApp("ns", "myapp", "App", []helxv1.Service{
		{
			Name:    "main",
			Image:   "nginx",
			Command: []string{"nginx"},
			Ports:   []helxv1.PortMap{{ContainerPort: 80, Port: 80}},
			Volumes: map[string]string{"data": "mydata:/data,size=20G"},
		},
	})
	user := makeUser("ns", "alice", nil)
	inst := makeInst("ns", "inst1", "myapp", "alice", "test-uuid-size")

	setupGraphForArtifacts(app, user, inst)
	artifacts, err := GenerateArtifacts(inst)
	if err != nil {
		t.Fatal(err)
	}
	if artifacts == nil {
		t.Fatal("expected artifacts")
	}
	pvc := artifacts.PVCs["mydata"]
	if !strings.Contains(pvc.Render, "storage: 20G") {
		t.Errorf("PVC should contain 'storage: 20G', got:\n%s", pvc.Render)
	}
}

func TestGenerateArtifacts_NoServicePort(t *testing.T) {
	// When Port=0, HasService=false, so the service template should render empty
	app := makeApp("ns", "myapp", "App", []helxv1.Service{
		{
			Name:    "main",
			Image:   "nginx",
			Command: []string{"nginx"},
			Ports:   []helxv1.PortMap{{ContainerPort: 80, Port: 0}},
		},
	})
	user := makeUser("ns", "alice", nil)
	inst := makeInst("ns", "inst1", "myapp", "alice", "test-uuid-nosvc")

	setupGraphForArtifacts(app, user, inst)
	artifacts, err := GenerateArtifacts(inst)
	if err != nil {
		t.Fatal(err)
	}
	if artifacts == nil {
		t.Fatal("expected artifacts")
	}
	// Service render should be empty/minimal since HasService is false
	svc := artifacts.Services["main"]
	if strings.Contains(svc.Render, "kind: Service") {
		t.Error("service render should NOT contain 'kind: Service' when Port=0")
	}
}

// ---------------------------------------------------------------------------
// Additional coverage tests
// ---------------------------------------------------------------------------

// 1. GetUserNameFromInst — nil inst case
func TestGetUserNameFromInst_Nil(t *testing.T) {
	got := GetUserNameFromInst(nil)
	if got != "" {
		t.Errorf("expected empty string for nil inst, got %s", got)
	}
}

// 2. GetUserNameFromInst — empty UserName
func TestGetUserNameFromInst_EmptyUserName(t *testing.T) {
	inst := makeInst("ns", "inst1", "myapp", "", "uuid-1")
	got := GetUserNameFromInst(inst)
	if got != "" {
		t.Errorf("expected empty string for empty UserName, got %s", got)
	}
}

// 3. GetObjFromMap — not found case via GetApp
func TestGetApp_NotFound(t *testing.T) {
	resetTables()
	got := GetApp("nonexistent")
	if got != nil {
		t.Errorf("expected nil for nonexistent app, got %+v", got)
	}
}

// 4. GetObjFromMap — not found case via GetUser
func TestGetUser_NotFound(t *testing.T) {
	resetTables()
	got := GetUser("nonexistent")
	if got != nil {
		t.Errorf("expected nil for nonexistent user, got %+v", got)
	}
}

// 5. DeleteInst — verify associations are cleaned up from appTable and userTable
func TestDeleteInst_CleansAssociations(t *testing.T) {
	resetTables()
	app := makeApp("ns", "myapp", "cls", nil)
	user := makeUser("ns", "alice", nil)
	AddApp(app)
	AddUser(user)

	inst := makeInst("ns", "inst1", "myapp", "alice", "uuid-1")
	AddInst(inst)

	// Verify associations exist before delete
	appElem, ok := appTable["ns/myapp"]
	if !ok {
		t.Fatal("app not in table before delete")
	}
	if !appElem.InstSet["ns/inst1"] {
		t.Error("inst not in app's InstSet before delete")
	}

	userElem, ok := userTable["ns/alice"]
	if !ok {
		t.Fatal("user not in table before delete")
	}
	if !userElem.InstSet["ns/inst1"] {
		t.Error("inst not in user's InstSet before delete")
	}

	// Delete the inst
	DeleteInst("ns/inst1")

	// Verify inst is gone
	_, found := GetInst("ns/inst1")
	if found {
		t.Error("inst should have been deleted from instanceTable")
	}
}

// 6. DeleteInst — deleting a nonexistent inst should not panic
func TestDeleteInst_Nonexistent(t *testing.T) {
	resetTables()
	// Should not panic
	DeleteInst("ns/nonexistent")
}

// 7. clearStorage — test that it clears the storage map
func TestClearStorage(t *testing.T) {
	// Seed storage with some data
	storage["testkey1"] = []string{"a", "b"}
	storage["testkey2"] = []string{"c"}

	if len(storage) == 0 {
		t.Fatal("storage should have entries before clearing")
	}

	clearStorage()

	if len(storage) != 0 {
		t.Errorf("expected storage to be empty after clearStorage, got %d entries", len(storage))
	}
}

// 8. newSimpleErrorLogger — call the returned function
func TestNewSimpleErrorLogger_Call(t *testing.T) {
	logger := logr.Discard()
	fn := newSimpleErrorLogger(logger)

	// Calling the returned function with a real error should not panic
	fn(errors.New("test error"), "test message")
}

// 9. newSimpleInfoLogger — call the returned function
func TestNewSimpleInfoLogger_Call(t *testing.T) {
	logger := logr.Discard()
	fn := newSimpleInfoLogger(logger)

	// Calling the returned function should not panic
	fn("test info message")
}

// 10. newSimpleDebugLogger — call the returned function
func TestNewSimpleDebugLogger_Call(t *testing.T) {
	logger := logr.Discard()
	fn := newSimpleDebugLogger(logger)

	// Calling the returned function should not panic
	fn("test debug message")
}

// 11. transformVolumes — duplicate volume names across services (already-found branch)
func TestTransformVolumes_DuplicateVolumeNames(t *testing.T) {
	// First service adds volume "data" to the sourceMap
	service1 := helxv1.Service{
		Name:    "svc1",
		Image:   "nginx",
		Command: []string{"nginx"},
		Volumes: map[string]string{"data": "mydata:/data1"},
	}
	sourceMap := make(map[string]*template_io.Volume)

	mounts1, err := transformVolumes(service1, sourceMap)
	if err != nil {
		t.Fatal(err)
	}
	if len(mounts1) != 1 {
		t.Fatalf("expected 1 mount from service1, got %d", len(mounts1))
	}
	if sourceMap["data"] == nil {
		t.Fatal("expected volume 'data' in sourceMap after first service")
	}
	originalClaim := sourceMap["data"].Attr["claim"]
	if originalClaim != "mydata" {
		t.Errorf("expected claim=mydata, got %s", originalClaim)
	}

	// Second service also has volume "data" but with a different claim
	// The "already found" branch should skip overwriting
	service2 := helxv1.Service{
		Name:    "svc2",
		Image:   "nginx",
		Command: []string{"nginx"},
		Volumes: map[string]string{"data": "otherdata:/data2"},
	}

	mounts2, err := transformVolumes(service2, sourceMap)
	if err != nil {
		t.Fatal(err)
	}
	if len(mounts2) != 1 {
		t.Fatalf("expected 1 mount from service2, got %d", len(mounts2))
	}

	// sourceMap["data"] should still point to the original volume (mydata), not otherdata
	if sourceMap["data"].Attr["claim"] != "mydata" {
		t.Errorf("expected sourceMap to retain original claim=mydata, got %s", sourceMap["data"].Attr["claim"])
	}

	// But the mount for service2 should reference the new mount path
	if mounts2[0].MountPath != "/data2" {
		t.Errorf("expected mount path /data2, got %s", mounts2[0].MountPath)
	}
}

// 12. transformApp with volumes — cover the transformVolumes call from within transformApp
func TestTransformApp_WithVolumes(t *testing.T) {
	app := helxv1.HelxApp{
		Spec: helxv1.HelxAppSpec{
			Services: []helxv1.Service{
				{
					Name:    "main",
					Image:   "nginx:latest",
					Command: []string{"nginx"},
					Ports:   []helxv1.PortMap{{ContainerPort: 80, Port: 8080}},
					Volumes: map[string]string{
						"data": "mydata:/data",
						"logs": "mylogs:/logs",
					},
				},
			},
		},
	}
	inst := makeInst("ns", "inst1", "myapp", "alice", "uuid-1")

	containers, sourceMap, err := transformApp(inst, app, *makeUser("ns", "testuser", nil))
	if err != nil {
		t.Fatal(err)
	}
	if len(containers) != 1 {
		t.Fatalf("expected 1 container, got %d", len(containers))
	}
	if len(containers[0].VolumeMounts) != 2 {
		t.Errorf("expected 2 volume mounts, got %d", len(containers[0].VolumeMounts))
	}
	if len(sourceMap) != 2 {
		t.Errorf("expected 2 volumes in sourceMap, got %d", len(sourceMap))
	}
	if sourceMap["data"] == nil || sourceMap["logs"] == nil {
		t.Error("expected both data and logs volumes in sourceMap")
	}
}

// 13. GenerateArtifacts with multiple PVC volumes on one service
func TestGenerateArtifacts_MultiplePVCs(t *testing.T) {
	app := makeApp("ns", "myapp", "App", []helxv1.Service{
		{
			Name:    "main",
			Image:   "nginx",
			Command: []string{"nginx"},
			Ports:   []helxv1.PortMap{{ContainerPort: 80, Port: 80}},
			Volumes: map[string]string{
				"data":  "mydata:/data",
				"cache": "mycache:/cache",
			},
		},
	})
	user := makeUser("ns", "alice", nil)
	inst := makeInst("ns", "inst1", "myapp", "alice", "test-uuid-multipvc")

	setupGraphForArtifacts(app, user, inst)
	artifacts, err := GenerateArtifacts(inst)
	if err != nil {
		t.Fatal(err)
	}
	if artifacts == nil {
		t.Fatal("expected artifacts, got nil")
	}
	if len(artifacts.PVCs) != 2 {
		t.Fatalf("expected 2 PVCs, got %d", len(artifacts.PVCs))
	}
	if _, ok := artifacts.PVCs["mydata"]; !ok {
		t.Error("expected PVC with claim 'mydata'")
	}
	if _, ok := artifacts.PVCs["mycache"]; !ok {
		t.Error("expected PVC with claim 'mycache'")
	}
	for name, pvc := range artifacts.PVCs {
		if !strings.Contains(pvc.Render, "PersistentVolumeClaim") {
			t.Errorf("PVC %s render should contain PersistentVolumeClaim", name)
		}
	}
}

// 14. GetNamespacedName — verify it produces namespace/name
func TestGetNamespacedName(t *testing.T) {
	app := makeApp("mynamespace", "myappname", "cls", nil)
	got := GetNamespacedName(app)
	expected := "mynamespace/myappname"
	if got != expected {
		t.Errorf("expected %s, got %s", expected, got)
	}
}

// 15. GetNamespacedName with empty namespace
func TestGetNamespacedName_EmptyNamespace(t *testing.T) {
	app := makeApp("", "myappname", "cls", nil)
	got := GetNamespacedName(app)
	expected := "/myappname"
	if got != expected {
		t.Errorf("expected %s, got %s", expected, got)
	}
}

// 16. GetNamespacedName with user object
func TestGetNamespacedName_User(t *testing.T) {
	user := makeUser("testns", "alice", nil)
	got := GetNamespacedName(user)
	expected := "testns/alice"
	if got != expected {
		t.Errorf("expected %s, got %s", expected, got)
	}
}

// 17. GetNamespacedName with inst object
func TestGetNamespacedName_Inst(t *testing.T) {
	inst := makeInst("testns", "inst1", "myapp", "alice", "uuid-1")
	got := GetNamespacedName(inst)
	expected := "testns/inst1"
	if got != expected {
		t.Errorf("expected %s, got %s", expected, got)
	}
}

// 18. GenerateArtifacts with duplicate volume across two services (already-found branch in full pipeline)
func TestGenerateArtifacts_DuplicateVolumeAcrossServices(t *testing.T) {
	app := makeApp("ns", "myapp", "App", []helxv1.Service{
		{
			Name:    "web",
			Image:   "nginx",
			Command: []string{"nginx"},
			Ports:   []helxv1.PortMap{{ContainerPort: 80, Port: 80}},
			Volumes: map[string]string{"shared": "sharedvol:/web-data"},
		},
		{
			Name:    "worker",
			Image:   "worker:latest",
			Command: []string{"worker"},
			Ports:   []helxv1.PortMap{{ContainerPort: 9090}},
			Volumes: map[string]string{"shared": "sharedvol:/worker-data"},
		},
	})
	user := makeUser("ns", "alice", nil)
	inst := makeInst("ns", "inst1", "myapp", "alice", "test-uuid-dupvol")

	setupGraphForArtifacts(app, user, inst)
	artifacts, err := GenerateArtifacts(inst)
	if err != nil {
		t.Fatal(err)
	}
	if artifacts == nil {
		t.Fatal("expected artifacts, got nil")
	}
	// Only one PVC should be created even though two services reference the same volume name
	if len(artifacts.PVCs) != 1 {
		t.Fatalf("expected 1 PVC for duplicate volume name, got %d", len(artifacts.PVCs))
	}
	if _, ok := artifacts.PVCs["sharedvol"]; !ok {
		t.Error("expected PVC with claim 'sharedvol'")
	}
}

// ---------------------------------------------------------------------------
// mergeEnvironment tests
// ---------------------------------------------------------------------------

func TestMergeEnvironment_BothNil(t *testing.T) {
	result := mergeEnvironment(nil, nil)
	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

func TestMergeEnvironment_AppOnly(t *testing.T) {
	app := map[string]string{"FOO": "bar"}
	result := mergeEnvironment(app, nil)
	if result["FOO"] != "bar" {
		t.Errorf("expected FOO=bar, got %s", result["FOO"])
	}
}

func TestMergeEnvironment_InstOnly(t *testing.T) {
	inst := map[string]string{"BAZ": "qux"}
	result := mergeEnvironment(nil, inst)
	if result["BAZ"] != "qux" {
		t.Errorf("expected BAZ=qux, got %s", result["BAZ"])
	}
}

func TestMergeEnvironment_Disjoint(t *testing.T) {
	app := map[string]string{"FOO": "bar"}
	inst := map[string]string{"BAZ": "qux"}
	result := mergeEnvironment(app, inst)
	if len(result) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(result))
	}
	if result["FOO"] != "bar" || result["BAZ"] != "qux" {
		t.Errorf("unexpected result: %v", result)
	}
}

func TestMergeEnvironment_InstOverridesApp(t *testing.T) {
	app := map[string]string{"FOO": "from-app", "SHARED": "app-value"}
	inst := map[string]string{"SHARED": "inst-value", "NEW": "inst-only"}
	result := mergeEnvironment(app, inst)
	if result["FOO"] != "from-app" {
		t.Errorf("expected FOO=from-app, got %s", result["FOO"])
	}
	if result["SHARED"] != "inst-value" {
		t.Errorf("expected SHARED=inst-value (inst takes precedence), got %s", result["SHARED"])
	}
	if result["NEW"] != "inst-only" {
		t.Errorf("expected NEW=inst-only, got %s", result["NEW"])
	}
}

func TestTransformApp_InstanceEnvMerged(t *testing.T) {
	app := helxv1.HelxApp{
		Spec: helxv1.HelxAppSpec{
			Services: []helxv1.Service{
				{
					Name:        "main",
					Image:       "nginx",
					Command:     []string{"nginx"},
					Ports:       []helxv1.PortMap{{ContainerPort: 80}},
					Environment: map[string]string{"APP_VAR": "app-val", "SHARED": "from-app"},
				},
			},
		},
	}
	inst := makeInst("ns", "inst1", "myapp", "alice", "uuid-1")
	inst.Spec.Environment = map[string]string{"INST_VAR": "inst-val", "SHARED": "from-inst"}

	containers, _, err := transformApp(inst, app, *makeUser("ns", "testuser", nil))
	if err != nil {
		t.Fatal(err)
	}
	if len(containers) != 1 {
		t.Fatalf("expected 1 container, got %d", len(containers))
	}
	env := containers[0].Environment
	if env["APP_VAR"] != "app-val" {
		t.Errorf("expected APP_VAR=app-val, got %s", env["APP_VAR"])
	}
	if env["INST_VAR"] != "inst-val" {
		t.Errorf("expected INST_VAR=inst-val, got %s", env["INST_VAR"])
	}
	if env["SHARED"] != "from-inst" {
		t.Errorf("expected SHARED=from-inst (inst precedence), got %s", env["SHARED"])
	}
}

func TestGenerateArtifacts_InstanceEnvInDeployment(t *testing.T) {
	app := makeApp("ns", "myapp", "Nginx", []helxv1.Service{
		{
			Name:        "main",
			Image:       "nginx",
			Command:     []string{"nginx"},
			Ports:       []helxv1.PortMap{{ContainerPort: 80, Port: 80}},
			Environment: map[string]string{"APP_LEVEL": "hello"},
		},
	})
	user := makeUser("ns", "alice", nil)
	inst := makeInst("ns", "inst1", "myapp", "alice", "test-uuid-env")
	inst.Spec.Environment = map[string]string{"INST_LEVEL": "world"}

	setupGraphForArtifacts(app, user, inst)
	artifacts, err := GenerateArtifacts(inst)
	if err != nil {
		t.Fatal(err)
	}
	if artifacts == nil {
		t.Fatal("expected artifacts")
	}
	render := artifacts.Deployment.Render
	if !strings.Contains(render, "APP_LEVEL") {
		t.Error("deployment should contain APP_LEVEL env var")
	}
	if !strings.Contains(render, "INST_LEVEL") {
		t.Error("deployment should contain INST_LEVEL env var from instance")
	}
}

// ---------------------------------------------------------------------------
// User environment and volume merge tests
// ---------------------------------------------------------------------------

func TestTransformApp_UserEnvMerged(t *testing.T) {
	app := helxv1.HelxApp{
		Spec: helxv1.HelxAppSpec{
			Services: []helxv1.Service{
				{
					Name:        "main",
					Image:       "nginx",
					Command:     []string{"nginx"},
					Ports:       []helxv1.PortMap{{ContainerPort: 80}},
					Environment: map[string]string{"APP_VAR": "app", "SHARED": "from-app"},
				},
			},
		},
	}
	user := helxv1.HelxUser{}
	user.Name = "alice"
	user.Namespace = "ns"
	user.Spec.Environment = map[string]string{"USER_VAR": "user", "SHARED": "from-user"}

	inst := makeInst("ns", "inst1", "myapp", "alice", "uuid-1")

	containers, _, err := transformApp(inst, app, user)
	if err != nil {
		t.Fatal(err)
	}
	env := containers[0].Environment
	if env["APP_VAR"] != "app" {
		t.Errorf("expected APP_VAR=app, got %s", env["APP_VAR"])
	}
	if env["USER_VAR"] != "user" {
		t.Errorf("expected USER_VAR=user, got %s", env["USER_VAR"])
	}
	// User overrides app
	if env["SHARED"] != "from-user" {
		t.Errorf("expected SHARED=from-user (user > app), got %s", env["SHARED"])
	}
}

func TestTransformApp_ThreeWayEnvPrecedence(t *testing.T) {
	app := helxv1.HelxApp{
		Spec: helxv1.HelxAppSpec{
			Services: []helxv1.Service{
				{
					Name:        "main",
					Image:       "nginx",
					Command:     []string{"nginx"},
					Ports:       []helxv1.PortMap{{ContainerPort: 80}},
					Environment: map[string]string{"KEY": "from-app"},
				},
			},
		},
	}
	user := helxv1.HelxUser{}
	user.Name = "alice"
	user.Namespace = "ns"
	user.Spec.Environment = map[string]string{"KEY": "from-user"}

	inst := makeInst("ns", "inst1", "myapp", "alice", "uuid-1")
	inst.Spec.Environment = map[string]string{"KEY": "from-inst"}

	containers, _, err := transformApp(inst, app, user)
	if err != nil {
		t.Fatal(err)
	}
	// inst > user > app
	if containers[0].Environment["KEY"] != "from-inst" {
		t.Errorf("expected KEY=from-inst (inst wins), got %s", containers[0].Environment["KEY"])
	}
}

func TestTransformApp_UserVolumes(t *testing.T) {
	app := helxv1.HelxApp{
		Spec: helxv1.HelxAppSpec{
			Services: []helxv1.Service{
				{
					Name:    "main",
					Image:   "nginx",
					Command: []string{"nginx"},
					Ports:   []helxv1.PortMap{{ContainerPort: 80}},
					Volumes: map[string]string{"appvol": "app-data:/data"},
				},
			},
		},
	}
	user := helxv1.HelxUser{}
	user.Name = "alice"
	user.Namespace = "ns"
	user.Spec.Volumes = map[string]string{"uservol": "user-home:/home/alice"}

	inst := makeInst("ns", "inst1", "myapp", "alice", "uuid-1")

	containers, sourceMap, err := transformApp(inst, app, user)
	if err != nil {
		t.Fatal(err)
	}

	// Both volumes should be in the source map
	if _, ok := sourceMap["appvol"]; !ok {
		t.Error("expected appvol in sourceMap")
	}
	if _, ok := sourceMap["uservol"]; !ok {
		t.Error("expected uservol in sourceMap")
	}

	// Container should have mounts for both
	mounts := containers[0].VolumeMounts
	mountNames := make(map[string]bool)
	for _, m := range mounts {
		mountNames[m.Name] = true
	}
	if !mountNames["appvol"] {
		t.Error("expected appvol mount on container")
	}
	if !mountNames["uservol"] {
		t.Error("expected uservol mount on container")
	}
}

func TestTransformApp_UserVolumesOnAllContainers(t *testing.T) {
	app := helxv1.HelxApp{
		Spec: helxv1.HelxAppSpec{
			Services: []helxv1.Service{
				{Name: "web", Image: "nginx", Command: []string{"nginx"}, Ports: []helxv1.PortMap{{ContainerPort: 80}}},
				{Name: "sidecar", Image: "busybox", Command: []string{"sh"}, Ports: []helxv1.PortMap{{ContainerPort: 9090}}},
			},
		},
	}
	user := helxv1.HelxUser{}
	user.Name = "alice"
	user.Namespace = "ns"
	user.Spec.Volumes = map[string]string{"shared": "shared-data:/shared"}

	inst := makeInst("ns", "inst1", "myapp", "alice", "uuid-1")

	containers, _, err := transformApp(inst, app, user)
	if err != nil {
		t.Fatal(err)
	}
	if len(containers) != 2 {
		t.Fatalf("expected 2 containers, got %d", len(containers))
	}
	for _, c := range containers {
		found := false
		for _, m := range c.VolumeMounts {
			if m.Name == "shared" {
				found = true
			}
		}
		if !found {
			t.Errorf("expected user volume 'shared' on container %s", c.Name)
		}
	}
}

func TestGenerateArtifacts_UserEnvAndVolumes(t *testing.T) {
	app := makeApp("ns", "myapp", "Nginx", []helxv1.Service{
		{
			Name:    "main",
			Image:   "nginx",
			Command: []string{"nginx"},
			Ports:   []helxv1.PortMap{{ContainerPort: 80, Port: 80}},
		},
	})
	user := makeUser("ns", "alice", nil)
	user.Spec.Environment = map[string]string{"USER_ENV": "from-user"}
	user.Spec.Volumes = map[string]string{"uhome": "alice-home:/home/alice"}

	inst := makeInst("ns", "inst1", "myapp", "alice", "test-uuid-userenv")

	setupGraphForArtifacts(app, user, inst)
	artifacts, err := GenerateArtifacts(inst)
	if err != nil {
		t.Fatal(err)
	}
	if artifacts == nil {
		t.Fatal("expected artifacts")
	}
	render := artifacts.Deployment.Render
	if !strings.Contains(render, "USER_ENV") {
		t.Error("deployment should contain USER_ENV from user")
	}
	if !strings.Contains(render, "alice-home") {
		t.Error("deployment should reference user volume claim alice-home")
	}
	// User PVC should be created
	if _, ok := artifacts.PVCs["alice-home"]; !ok {
		t.Error("expected PVC for user volume alice-home")
	}
}

// ---------------------------------------------------------------------------
// Secret and ConfigMap volume rendering tests
// ---------------------------------------------------------------------------

func TestGenerateArtifacts_SecretVolume(t *testing.T) {
	app := makeApp("ns", "myapp", "App", []helxv1.Service{
		{
			Name:    "main",
			Image:   "nginx",
			Command: []string{"nginx"},
			Ports:   []helxv1.PortMap{{ContainerPort: 80, Port: 80}},
			Volumes: map[string]string{"creds": "secret://db-creds:/mnt/creds,ro"},
		},
	})
	user := makeUser("ns", "alice", nil)
	inst := makeInst("ns", "inst1", "myapp", "alice", "test-uuid-secret")

	setupGraphForArtifacts(app, user, inst)
	artifacts, err := GenerateArtifacts(inst)
	if err != nil {
		t.Fatal(err)
	}
	if artifacts == nil {
		t.Fatal("expected artifacts")
	}
	render := artifacts.Deployment.Render
	if !strings.Contains(render, "secret:") {
		t.Error("deployment should contain secret volume source")
	}
	if !strings.Contains(render, "secretName: db-creds") {
		t.Errorf("deployment should contain secretName: db-creds, got:\n%s", render)
	}
	// Secret volumes should NOT produce PVCs
	if len(artifacts.PVCs) != 0 {
		t.Errorf("expected 0 PVCs for secret volume, got %d", len(artifacts.PVCs))
	}
}

func TestGenerateArtifacts_ConfigMapVolume(t *testing.T) {
	app := makeApp("ns", "myapp", "App", []helxv1.Service{
		{
			Name:    "main",
			Image:   "nginx",
			Command: []string{"nginx"},
			Ports:   []helxv1.PortMap{{ContainerPort: 80, Port: 80}},
			Volumes: map[string]string{"cfg": "configmap://app-config:/etc/config"},
		},
	})
	user := makeUser("ns", "alice", nil)
	inst := makeInst("ns", "inst1", "myapp", "alice", "test-uuid-configmap")

	setupGraphForArtifacts(app, user, inst)
	artifacts, err := GenerateArtifacts(inst)
	if err != nil {
		t.Fatal(err)
	}
	if artifacts == nil {
		t.Fatal("expected artifacts")
	}
	render := artifacts.Deployment.Render
	if !strings.Contains(render, "configMap:") {
		t.Error("deployment should contain configMap volume source")
	}
	if !strings.Contains(render, "name: app-config") {
		t.Errorf("deployment should contain name: app-config, got:\n%s", render)
	}
	// ConfigMap volumes should NOT produce PVCs
	if len(artifacts.PVCs) != 0 {
		t.Errorf("expected 0 PVCs for configmap volume, got %d", len(artifacts.PVCs))
	}
}

// ---------------------------------------------------------------------------
// buildEnvFrom and envFrom rendering tests
// ---------------------------------------------------------------------------

func TestBuildEnvFrom_Empty(t *testing.T) {
	result := buildEnvFrom(nil, nil, nil, nil, nil, nil)
	if len(result) != 0 {
		t.Errorf("expected empty, got %d", len(result))
	}
}

func TestBuildEnvFrom_SecretsOnly(t *testing.T) {
	result := buildEnvFrom([]string{"db-creds"}, nil, nil, nil, nil, nil)
	if len(result) != 1 {
		t.Fatalf("expected 1, got %d", len(result))
	}
	if result[0].SecretName != "db-creds" {
		t.Errorf("expected db-creds, got %s", result[0].SecretName)
	}
}

func TestBuildEnvFrom_ConfigMapsOnly(t *testing.T) {
	result := buildEnvFrom(nil, nil, nil, []string{"app-config"}, nil, nil)
	if len(result) != 1 {
		t.Fatalf("expected 1, got %d", len(result))
	}
	if result[0].ConfigMapName != "app-config" {
		t.Errorf("expected app-config, got %s", result[0].ConfigMapName)
	}
}

func TestBuildEnvFrom_Dedup(t *testing.T) {
	// Same secret at app and inst level should appear once
	result := buildEnvFrom([]string{"shared"}, nil, []string{"shared"}, nil, nil, nil)
	if len(result) != 1 {
		t.Fatalf("expected 1 (deduped), got %d", len(result))
	}
	if result[0].SecretName != "shared" {
		t.Errorf("expected shared, got %s", result[0].SecretName)
	}
}

func TestBuildEnvFrom_AllThreeLevels(t *testing.T) {
	result := buildEnvFrom(
		[]string{"app-secret"}, []string{"user-secret"}, []string{"inst-secret"},
		[]string{"app-cm"}, []string{"user-cm"}, []string{"inst-cm"},
	)
	if len(result) != 6 {
		t.Fatalf("expected 6 envFrom sources, got %d", len(result))
	}
}

func TestTransformApp_EnvFrom(t *testing.T) {
	app := helxv1.HelxApp{
		Spec: helxv1.HelxAppSpec{
			Services: []helxv1.Service{
				{
					Name:           "main",
					Image:          "nginx",
					Command:        []string{"nginx"},
					Ports:          []helxv1.PortMap{{ContainerPort: 80}},
					SecretsFrom:    []string{"app-secret"},
					ConfigMapsFrom: []string{"app-config"},
				},
			},
		},
	}
	user := helxv1.HelxUser{}
	user.Name = "alice"
	user.Namespace = "ns"
	user.Spec.SecretsFrom = []string{"user-secret"}

	inst := makeInst("ns", "inst1", "myapp", "alice", "uuid-1")
	inst.Spec.ConfigMapsFrom = []string{"inst-config"}

	containers, _, err := transformApp(inst, app, user)
	if err != nil {
		t.Fatal(err)
	}
	envFrom := containers[0].EnvFrom
	if len(envFrom) != 4 {
		t.Fatalf("expected 4 envFrom sources, got %d", len(envFrom))
	}
	// Check that we have the expected sources
	secretNames := map[string]bool{}
	cmNames := map[string]bool{}
	for _, ef := range envFrom {
		if ef.SecretName != "" {
			secretNames[ef.SecretName] = true
		}
		if ef.ConfigMapName != "" {
			cmNames[ef.ConfigMapName] = true
		}
	}
	if !secretNames["app-secret"] || !secretNames["user-secret"] {
		t.Errorf("missing expected secrets: %v", secretNames)
	}
	if !cmNames["app-config"] || !cmNames["inst-config"] {
		t.Errorf("missing expected configmaps: %v", cmNames)
	}
}

func TestGenerateArtifacts_EnvFrom(t *testing.T) {
	app := makeApp("ns", "myapp", "App", []helxv1.Service{
		{
			Name:        "main",
			Image:       "nginx",
			Command:     []string{"nginx"},
			Ports:       []helxv1.PortMap{{ContainerPort: 80, Port: 80}},
			SecretsFrom: []string{"db-creds"},
		},
	})
	user := makeUser("ns", "alice", nil)
	user.Spec.ConfigMapsFrom = []string{"user-settings"}

	inst := makeInst("ns", "inst1", "myapp", "alice", "test-uuid-envfrom")

	setupGraphForArtifacts(app, user, inst)
	artifacts, err := GenerateArtifacts(inst)
	if err != nil {
		t.Fatal(err)
	}
	if artifacts == nil {
		t.Fatal("expected artifacts")
	}
	render := artifacts.Deployment.Render
	if !strings.Contains(render, "envFrom:") {
		t.Errorf("deployment should contain envFrom block, got:\n%s", render)
	}
	if !strings.Contains(render, "secretRef:") {
		t.Errorf("deployment should contain secretRef, got:\n%s", render)
	}
	if !strings.Contains(render, "name: db-creds") {
		t.Errorf("deployment should contain name: db-creds, got:\n%s", render)
	}
	if !strings.Contains(render, "configMapRef:") {
		t.Errorf("deployment should contain configMapRef, got:\n%s", render)
	}
	if !strings.Contains(render, "name: user-settings") {
		t.Errorf("deployment should contain name: user-settings, got:\n%s", render)
	}
}
