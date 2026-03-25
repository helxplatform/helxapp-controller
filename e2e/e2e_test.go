package e2e

import (
	"context"
	"fmt"
	"testing"
	"time"

	helxv1 "github.com/helxplatform/helxapp-controller/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ---------------------------------------------------------------------------
// 1-5: Core Lifecycle
// ---------------------------------------------------------------------------

func TestE2E_CreateTriple_DeploymentCreated(t *testing.T) {
	s := suffix()
	appName := "app-" + s
	userName := "user-" + s
	instName := "inst-" + s

	app := newApp(appName, []helxv1.Service{
		simpleService("main", "nginx:latest", 80, 8080, []string{"nginx", "-g", "daemon off;"}),
	})
	user := newUser(userName)
	inst := newInst(instName, appName, userName)

	registerCleanup(t, inst, app, user)
	createObj(t, app)
	createObj(t, user)
	createObj(t, inst)

	uuid := waitForInstUUID(t, instName)
	deploy := waitForDeployment(t, uuid)
	if deploy == nil {
		t.Fatal("expected deployment, got nil")
	}
	svc := waitForService(t, uuid)
	if svc == nil {
		t.Fatal("expected service, got nil")
	}
}

func TestE2E_OrderIndependence_InstFirst(t *testing.T) {
	s := suffix()
	appName := "app-" + s
	userName := "user-" + s
	instName := "inst-" + s

	app := newApp(appName, []helxv1.Service{
		simpleService("main", "nginx:latest", 80, 8080, []string{"nginx", "-g", "daemon off;"}),
	})
	user := newUser(userName)
	inst := newInst(instName, appName, userName)

	registerCleanup(t, inst, app, user)

	// Create Inst first — no deployment yet
	createObj(t, inst)
	uuid := waitForInstUUID(t, instName)

	// Create User — still no deployment (app missing)
	createObj(t, user)
	time.Sleep(3 * time.Second) // brief pause, deployment should NOT appear yet

	// Create App last — now the triple is complete
	createObj(t, app)
	deploy := waitForDeployment(t, uuid)
	if deploy == nil {
		t.Fatal("expected deployment after completing triple")
	}
}

func TestE2E_DeleteInst_WorkloadsRemoved(t *testing.T) {
	s := suffix()
	appName := "app-" + s
	userName := "user-" + s
	instName := "inst-" + s

	app := newApp(appName, []helxv1.Service{
		simpleService("main", "nginx:latest", 80, 8080, []string{"nginx", "-g", "daemon off;"}),
	})
	user := newUser(userName)
	inst := newInst(instName, appName, userName)

	registerCleanup(t, app, user) // inst will be explicitly deleted
	createObj(t, app)
	createObj(t, user)
	createObj(t, inst)

	uuid := waitForInstUUID(t, instName)
	waitForDeployment(t, uuid)

	// Delete HelxInst — workloads should be garbage-collected via ownerRef
	deleteObj(t, inst)
	waitForNoDeployment(t, uuid)
}

func TestE2E_DeleteApp_CascadesToInstances(t *testing.T) {
	s := suffix()
	appName := "app-" + s
	userName := "user-" + s
	instName := "inst-" + s

	app := newApp(appName, []helxv1.Service{
		simpleService("main", "nginx:latest", 80, 8080, []string{"nginx", "-g", "daemon off;"}),
	})
	user := newUser(userName)
	inst := newInst(instName, appName, userName)

	registerCleanup(t, inst, user) // app will be explicitly deleted
	createObj(t, app)
	createObj(t, user)
	createObj(t, inst)

	uuid := waitForInstUUID(t, instName)
	waitForDeployment(t, uuid)

	// Delete HelxApp — controller should call DeleteDerivatives for connected instances
	deleteObj(t, app)
	waitForNoDeployment(t, uuid)
}

func TestE2E_DeleteUser_CascadesToInstances(t *testing.T) {
	s := suffix()
	appName := "app-" + s
	userName := "user-" + s
	instName := "inst-" + s

	app := newApp(appName, []helxv1.Service{
		simpleService("main", "nginx:latest", 80, 8080, []string{"nginx", "-g", "daemon off;"}),
	})
	user := newUser(userName)
	inst := newInst(instName, appName, userName)

	registerCleanup(t, inst, app) // user will be explicitly deleted
	createObj(t, app)
	createObj(t, user)
	createObj(t, inst)

	uuid := waitForInstUUID(t, instName)
	waitForDeployment(t, uuid)

	// Delete HelxUser — controller should call DeleteDerivatives for connected instances
	deleteObj(t, user)
	waitForNoDeployment(t, uuid)
}

// ---------------------------------------------------------------------------
// 6-12: Workload Correctness
// ---------------------------------------------------------------------------

func TestE2E_DeploymentLabels(t *testing.T) {
	s := suffix()
	appName := "app-" + s
	userName := "user-" + s
	instName := "inst-" + s

	app := newApp(appName, []helxv1.Service{
		simpleService("main", "nginx:latest", 80, 8080, []string{"nginx", "-g", "daemon off;"}),
	})
	user := newUser(userName)
	inst := newInst(instName, appName, userName)

	registerCleanup(t, inst, app, user)
	createObj(t, app)
	createObj(t, user)
	createObj(t, inst)

	uuid := waitForInstUUID(t, instName)
	deploy := waitForDeployment(t, uuid)

	labels := deploy.Labels
	if labels["helx.renci.org/id"] != uuid {
		t.Errorf("expected label helx.renci.org/id=%s, got %s", uuid, labels["helx.renci.org/id"])
	}
	if labels["helx.renci.org/app-name"] != appName {
		t.Errorf("expected label helx.renci.org/app-name=%s, got %s", appName, labels["helx.renci.org/app-name"])
	}
	if labels["helx.renci.org/username"] != userName {
		t.Errorf("expected label helx.renci.org/username=%s, got %s", userName, labels["helx.renci.org/username"])
	}
}

func TestE2E_DeploymentContainerSpec(t *testing.T) {
	s := suffix()
	appName := "app-" + s
	userName := "user-" + s
	instName := "inst-" + s

	svc := helxv1.Service{
		Name:    "web",
		Image:   "nginx:1.25",
		Command: []string{"/bin/sh", "-c", "echo hello"},
		Ports:   []helxv1.PortMap{{ContainerPort: 80, Port: 8080}},
		Environment: map[string]string{
			"MY_VAR": "my_value",
		},
	}
	app := newApp(appName, []helxv1.Service{svc})
	user := newUser(userName)
	inst := newInst(instName, appName, userName)

	registerCleanup(t, inst, app, user)
	createObj(t, app)
	createObj(t, user)
	createObj(t, inst)

	uuid := waitForInstUUID(t, instName)
	deploy := waitForDeployment(t, uuid)

	containers := deploy.Spec.Template.Spec.Containers
	if len(containers) == 0 {
		t.Fatal("expected at least one container")
	}
	c := containers[0]
	if c.Image != "nginx:1.25" {
		t.Errorf("expected image nginx:1.25, got %s", c.Image)
	}
	if len(c.Command) < 3 || c.Command[2] != "echo hello" {
		t.Errorf("unexpected command: %v", c.Command)
	}
	foundEnv := false
	for _, e := range c.Env {
		if e.Name == "MY_VAR" && e.Value == "my_value" {
			foundEnv = true
		}
	}
	if !foundEnv {
		t.Error("expected env var MY_VAR=my_value in container")
	}
}

func TestE2E_ServiceCreatedWhenPortMapped(t *testing.T) {
	s := suffix()
	appName := "app-" + s
	userName := "user-" + s
	instName := "inst-" + s

	app := newApp(appName, []helxv1.Service{
		simpleService("main", "nginx:latest", 80, 8080, []string{"nginx", "-g", "daemon off;"}),
	})
	user := newUser(userName)
	inst := newInst(instName, appName, userName)

	registerCleanup(t, inst, app, user)
	createObj(t, app)
	createObj(t, user)
	createObj(t, inst)

	uuid := waitForInstUUID(t, instName)
	svc := waitForService(t, uuid)

	foundPort := false
	for _, p := range svc.Spec.Ports {
		if p.Port == 8080 {
			foundPort = true
		}
	}
	if !foundPort {
		t.Errorf("expected service port 8080, got ports %+v", svc.Spec.Ports)
	}
}

func TestE2E_NoServiceWhenPortZero(t *testing.T) {
	s := suffix()
	appName := "app-" + s
	userName := "user-" + s
	instName := "inst-" + s

	app := newApp(appName, []helxv1.Service{
		simpleService("main", "nginx:latest", 80, 0, []string{"nginx", "-g", "daemon off;"}),
	})
	user := newUser(userName)
	inst := newInst(instName, appName, userName)

	registerCleanup(t, inst, app, user)
	createObj(t, app)
	createObj(t, user)
	createObj(t, inst)

	uuid := waitForInstUUID(t, instName)
	// Wait for the deployment to ensure the controller has reconciled
	waitForDeployment(t, uuid)
	// Give the controller a moment, then check no service was created
	time.Sleep(5 * time.Second)
	if !noServiceExists(t, uuid) {
		t.Error("expected no service when Port=0")
	}
}

func TestE2E_PVCCreatedForPVCVolume(t *testing.T) {
	s := suffix()
	appName := "app-" + s
	userName := "user-" + s
	instName := "inst-" + s
	claimName := "data-" + s

	svc := helxv1.Service{
		Name:    "main",
		Image:   "nginx:latest",
		Command: []string{"nginx", "-g", "daemon off;"},
		Ports:   []helxv1.PortMap{{ContainerPort: 80, Port: 8080}},
		Volumes: map[string]string{
			"data": claimName + ":/data",
		},
	}
	app := newApp(appName, []helxv1.Service{svc})
	user := newUser(userName)
	inst := newInst(instName, appName, userName)

	// PVC cleanup (may have retain label or not)
	pvcObj := &corev1.PersistentVolumeClaim{}
	pvcObj.Name = claimName
	pvcObj.Namespace = testNS
	registerCleanup(t, inst, app, user, pvcObj)

	createObj(t, app)
	createObj(t, user)
	createObj(t, inst)

	waitForInstUUID(t, instName)
	pvc := waitForPVC(t, claimName)
	if pvc == nil {
		t.Fatal("expected PVC to be created")
	}
}

func TestE2E_NFSVolume_NoPVC(t *testing.T) {
	s := suffix()
	appName := "app-" + s
	userName := "user-" + s
	instName := "inst-" + s

	svc := helxv1.Service{
		Name:    "main",
		Image:   "nginx:latest",
		Command: []string{"nginx", "-g", "daemon off;"},
		Ports:   []helxv1.PortMap{{ContainerPort: 80, Port: 8080}},
		Volumes: map[string]string{
			"share": "nfs:///nfsserver/exports:/mnt",
		},
	}
	app := newApp(appName, []helxv1.Service{svc})
	user := newUser(userName)
	inst := newInst(instName, appName, userName)

	registerCleanup(t, inst, app, user)
	createObj(t, app)
	createObj(t, user)
	createObj(t, inst)

	uuid := waitForInstUUID(t, instName)
	deploy := waitForDeployment(t, uuid)

	// Verify NFS volume in deployment spec
	foundNFS := false
	for _, vol := range deploy.Spec.Template.Spec.Volumes {
		if vol.NFS != nil {
			foundNFS = true
		}
	}
	if !foundNFS {
		t.Error("expected NFS volume in deployment spec")
	}

	// Verify no PVC was created (NFS doesn't create PVCs)
	time.Sleep(5 * time.Second)
	ctx := context.Background()
	pvcList := &corev1.PersistentVolumeClaimList{}
	if err := k8sClient.List(ctx, pvcList, client.InNamespace(testNS),
		client.MatchingLabels{"helx.renci.org/id": uuid}); err != nil {
		t.Fatalf("failed to list PVCs: %v", err)
	}
	if len(pvcList.Items) != 0 {
		t.Errorf("expected 0 PVCs for NFS volume, got %d", len(pvcList.Items))
	}
}

func TestE2E_ResourceLimitsApplied(t *testing.T) {
	s := suffix()
	appName := "app-" + s
	userName := "user-" + s
	instName := "inst-" + s

	app := newApp(appName, []helxv1.Service{
		simpleService("main", "nginx:latest", 80, 8080, []string{"nginx", "-g", "daemon off;"}),
	})
	user := newUser(userName)
	inst := newInstWithResources(instName, appName, userName, map[string]helxv1.Resources{
		"main": {
			Limits:   map[string]string{"cpu": "2", "memory": "4G"},
			Requests: map[string]string{"cpu": "1", "memory": "2G"},
		},
	})

	registerCleanup(t, inst, app, user)
	createObj(t, app)
	createObj(t, user)
	createObj(t, inst)

	uuid := waitForInstUUID(t, instName)
	deploy := waitForDeployment(t, uuid)

	containers := deploy.Spec.Template.Spec.Containers
	if len(containers) == 0 {
		t.Fatal("expected at least one container")
	}
	res := containers[0].Resources

	cpuLimit := res.Limits[corev1.ResourceCPU]
	if cpuLimit.Cmp(resource.MustParse("2")) != 0 {
		t.Errorf("expected cpu limit=2, got %s", cpuLimit.String())
	}
	memReq := res.Requests[corev1.ResourceMemory]
	if memReq.Cmp(resource.MustParse("2G")) != 0 {
		t.Errorf("expected memory request=2G, got %s", memReq.String())
	}
}

// ---------------------------------------------------------------------------
// 13: Security Context
// ---------------------------------------------------------------------------

func TestE2E_SecurityContextFromInst(t *testing.T) {
	s := suffix()
	appName := "app-" + s
	userName := "user-" + s
	instName := "inst-" + s

	app := newApp(appName, []helxv1.Service{
		simpleService("main", "nginx:latest", 80, 8080, []string{"nginx", "-g", "daemon off;"}),
	})
	user := newUser(userName)
	uid := int64(1000)
	fsGroup := int64(2000)
	inst := newInstWithSC(instName, appName, userName, &helxv1.SecurityContext{
		RunAsUser: &uid,
		FSGroup:   &fsGroup,
	})

	registerCleanup(t, inst, app, user)
	createObj(t, app)
	createObj(t, user)
	createObj(t, inst)

	uuid := waitForInstUUID(t, instName)
	deploy := waitForDeployment(t, uuid)

	sc := deploy.Spec.Template.Spec.SecurityContext
	if sc == nil {
		t.Fatal("expected pod security context, got nil")
	}
	if sc.RunAsUser == nil || *sc.RunAsUser != 1000 {
		t.Errorf("expected runAsUser=1000, got %v", sc.RunAsUser)
	}
	if sc.FSGroup == nil || *sc.FSGroup != 2000 {
		t.Errorf("expected fsGroup=2000, got %v", sc.FSGroup)
	}
}

// ---------------------------------------------------------------------------
// 14-16: Volume DSL Options
// ---------------------------------------------------------------------------

func TestE2E_PVC_RWXAccessMode(t *testing.T) {
	s := suffix()
	appName := "app-" + s
	userName := "user-" + s
	instName := "inst-" + s
	claimName := "rwx-" + s

	svc := helxv1.Service{
		Name:    "main",
		Image:   "nginx:latest",
		Command: []string{"nginx", "-g", "daemon off;"},
		Ports:   []helxv1.PortMap{{ContainerPort: 80, Port: 8080}},
		Volumes: map[string]string{
			"data": claimName + ":/data,rwx",
		},
	}
	app := newApp(appName, []helxv1.Service{svc})
	user := newUser(userName)
	inst := newInst(instName, appName, userName)

	pvcObj := &corev1.PersistentVolumeClaim{}
	pvcObj.Name = claimName
	pvcObj.Namespace = testNS
	registerCleanup(t, inst, app, user, pvcObj)

	createObj(t, app)
	createObj(t, user)
	createObj(t, inst)

	waitForInstUUID(t, instName)
	pvc := waitForPVC(t, claimName)

	foundRWX := false
	for _, mode := range pvc.Spec.AccessModes {
		if mode == corev1.ReadWriteMany {
			foundRWX = true
		}
	}
	if !foundRWX {
		t.Errorf("expected ReadWriteMany access mode, got %v", pvc.Spec.AccessModes)
	}
}

func TestE2E_PVC_RetainLabel(t *testing.T) {
	s := suffix()
	appName := "app-" + s
	userName := "user-" + s
	instName := "inst-" + s
	claimName := "ret-" + s

	svc := helxv1.Service{
		Name:    "main",
		Image:   "nginx:latest",
		Command: []string{"nginx", "-g", "daemon off;"},
		Ports:   []helxv1.PortMap{{ContainerPort: 80, Port: 8080}},
		Volumes: map[string]string{
			"data": claimName + ":/data,retain",
		},
	}
	app := newApp(appName, []helxv1.Service{svc})
	user := newUser(userName)
	inst := newInst(instName, appName, userName)

	pvcObj := &corev1.PersistentVolumeClaim{}
	pvcObj.Name = claimName
	pvcObj.Namespace = testNS
	registerCleanup(t, inst, app, user, pvcObj)

	createObj(t, app)
	createObj(t, user)
	createObj(t, inst)

	waitForInstUUID(t, instName)
	pvc := waitForPVC(t, claimName)

	labels := pvc.Labels
	if labels["helx.renci.org/retain"] != "true" {
		t.Errorf("expected label helx.renci.org/retain=true, got %s", labels["helx.renci.org/retain"])
	}
}

func TestE2E_PVC_StorageSize(t *testing.T) {
	s := suffix()
	appName := "app-" + s
	userName := "user-" + s
	instName := "inst-" + s
	claimName := "sz-" + s

	svc := helxv1.Service{
		Name:    "main",
		Image:   "nginx:latest",
		Command: []string{"nginx", "-g", "daemon off;"},
		Ports:   []helxv1.PortMap{{ContainerPort: 80, Port: 8080}},
		Volumes: map[string]string{
			"data": claimName + ":/data,size=20Gi",
		},
	}
	app := newApp(appName, []helxv1.Service{svc})
	user := newUser(userName)
	inst := newInst(instName, appName, userName)

	pvcObj := &corev1.PersistentVolumeClaim{}
	pvcObj.Name = claimName
	pvcObj.Namespace = testNS
	registerCleanup(t, inst, app, user, pvcObj)

	createObj(t, app)
	createObj(t, user)
	createObj(t, inst)

	waitForInstUUID(t, instName)
	pvc := waitForPVC(t, claimName)

	storageReq := pvc.Spec.Resources.Requests[corev1.ResourceStorage]
	expected := resource.MustParse("20Gi")
	if storageReq.Cmp(expected) != 0 {
		t.Errorf("expected storage=20Gi, got %s", storageReq.String())
	}
}

// ---------------------------------------------------------------------------
// 17: Retain Behavior
// ---------------------------------------------------------------------------

func TestE2E_RetainPVC_SurvivesInstDeletion(t *testing.T) {
	s := suffix()
	appName := "app-" + s
	userName := "user-" + s
	instName := "inst-" + s
	claimName := "keep-" + s

	svc := helxv1.Service{
		Name:    "main",
		Image:   "nginx:latest",
		Command: []string{"nginx", "-g", "daemon off;"},
		Ports:   []helxv1.PortMap{{ContainerPort: 80, Port: 8080}},
		Volumes: map[string]string{
			"data": claimName + ":/data,retain",
		},
	}
	app := newApp(appName, []helxv1.Service{svc})
	user := newUser(userName)
	inst := newInst(instName, appName, userName)

	// Explicit PVC cleanup since retain means it won't be garbage-collected
	pvcObj := &corev1.PersistentVolumeClaim{}
	pvcObj.Name = claimName
	pvcObj.Namespace = testNS
	registerCleanup(t, app, user, pvcObj)

	createObj(t, app)
	createObj(t, user)
	createObj(t, inst)

	uuid := waitForInstUUID(t, instName)
	waitForDeployment(t, uuid)
	waitForPVC(t, claimName)

	// Delete the instance
	deleteObj(t, inst)
	waitForNoDeployment(t, uuid)

	// PVC should still exist because of retain label
	time.Sleep(5 * time.Second)
	if !pvcExists(t, claimName) {
		t.Error("expected PVC to survive instance deletion due to retain label")
	}
}

// ---------------------------------------------------------------------------
// 18-19: Update Handling
// ---------------------------------------------------------------------------

func TestE2E_UpdateApp_DeploymentUpdated(t *testing.T) {
	s := suffix()
	appName := "app-" + s
	userName := "user-" + s
	instName := "inst-" + s

	app := newApp(appName, []helxv1.Service{
		simpleService("main", "nginx:1.24", 80, 8080, []string{"nginx", "-g", "daemon off;"}),
	})
	user := newUser(userName)
	inst := newInst(instName, appName, userName)

	registerCleanup(t, inst, app, user)
	createObj(t, app)
	createObj(t, user)
	createObj(t, inst)

	uuid := waitForInstUUID(t, instName)
	deploy := waitForDeployment(t, uuid)
	if deploy.Spec.Template.Spec.Containers[0].Image != "nginx:1.24" {
		t.Fatalf("initial image should be nginx:1.24, got %s", deploy.Spec.Template.Spec.Containers[0].Image)
	}

	// Update the App image
	ctx := context.Background()
	freshApp := &helxv1.HelxApp{}
	if err := k8sClient.Get(ctx, types.NamespacedName{Name: appName, Namespace: testNS}, freshApp); err != nil {
		t.Fatalf("failed to re-fetch app: %v", err)
	}
	freshApp.Spec.Services[0].Image = "nginx:1.25"
	updateObj(t, freshApp)

	// Wait for the deployment to reflect the new image
	err := waitForCondition(t, waitTimeout, func() bool {
		d := &helxv1.HelxApp{}
		_ = k8sClient.Get(ctx, types.NamespacedName{Name: appName, Namespace: testNS}, d)
		// Re-check deployment
		deploy := waitForDeploymentQuiet(uuid)
		return deploy != nil && len(deploy.Spec.Template.Spec.Containers) > 0 &&
			deploy.Spec.Template.Spec.Containers[0].Image == "nginx:1.25"
	})
	if err != nil {
		t.Fatal("deployment image was not updated to nginx:1.25")
	}
}

func TestE2E_UpdateInst_ResourcesUpdated(t *testing.T) {
	s := suffix()
	appName := "app-" + s
	userName := "user-" + s
	instName := "inst-" + s

	app := newApp(appName, []helxv1.Service{
		simpleService("main", "nginx:latest", 80, 8080, []string{"nginx", "-g", "daemon off;"}),
	})
	user := newUser(userName)
	inst := newInstWithResources(instName, appName, userName, map[string]helxv1.Resources{
		"main": {
			Limits: map[string]string{"cpu": "1"},
		},
	})

	registerCleanup(t, inst, app, user)
	createObj(t, app)
	createObj(t, user)
	createObj(t, inst)

	uuid := waitForInstUUID(t, instName)
	waitForDeployment(t, uuid)

	// Update instance resources
	ctx := context.Background()
	freshInst := &helxv1.HelxInst{}
	if err := k8sClient.Get(ctx, types.NamespacedName{Name: instName, Namespace: testNS}, freshInst); err != nil {
		t.Fatalf("failed to re-fetch inst: %v", err)
	}
	freshInst.Spec.Resources = map[string]helxv1.Resources{
		"main": {
			Limits: map[string]string{"cpu": "4"},
		},
	}
	updateObj(t, freshInst)

	// Wait for deployment to reflect new resource limits
	err := waitForCondition(t, waitTimeout, func() bool {
		deploy := waitForDeploymentQuiet(uuid)
		if deploy == nil || len(deploy.Spec.Template.Spec.Containers) == 0 {
			return false
		}
		cpuLimit := deploy.Spec.Template.Spec.Containers[0].Resources.Limits[corev1.ResourceCPU]
		return cpuLimit.Cmp(resource.MustParse("4")) == 0
	})
	if err != nil {
		t.Fatal("deployment cpu limit was not updated to 4")
	}
}

// ---------------------------------------------------------------------------
// 20-21: Status Fields
// ---------------------------------------------------------------------------

func TestE2E_InstGetsUUID(t *testing.T) {
	s := suffix()
	appName := "app-" + s
	userName := "user-" + s
	instName := "inst-" + s

	app := newApp(appName, []helxv1.Service{
		simpleService("main", "nginx:latest", 80, 8080, []string{"nginx", "-g", "daemon off;"}),
	})
	user := newUser(userName)
	inst := newInst(instName, appName, userName)

	registerCleanup(t, inst, app, user)
	createObj(t, app)
	createObj(t, user)
	createObj(t, inst)

	uuid := waitForInstUUID(t, instName)
	if len(uuid) == 0 {
		t.Fatal("expected non-empty UUID")
	}
	// UUID should be a valid UUID format (36 chars with hyphens)
	if len(uuid) != 36 {
		t.Errorf("expected UUID length 36, got %d: %s", len(uuid), uuid)
	}
}

func TestE2E_ObservedGenerationUpdated(t *testing.T) {
	s := suffix()
	appName := "app-" + s

	app := newApp(appName, []helxv1.Service{
		simpleService("main", "nginx:latest", 80, 8080, []string{"nginx", "-g", "daemon off;"}),
	})

	registerCleanup(t, app)
	createObj(t, app)

	fetchedApp := &helxv1.HelxApp{}
	waitForObservedGeneration(t, appName, fetchedApp)

	if fetchedApp.Status.ObservedGeneration < fetchedApp.Generation {
		t.Errorf("expected ObservedGeneration >= Generation, got %d < %d",
			fetchedApp.Status.ObservedGeneration, fetchedApp.Generation)
	}
}

// ---------------------------------------------------------------------------
// 22: Multi-Instance
// ---------------------------------------------------------------------------

func TestE2E_MultipleInstances_SeparateDeployments(t *testing.T) {
	s := suffix()
	appName := "app-" + s
	userName := "user-" + s
	instName1 := "inst1-" + s
	instName2 := "inst2-" + s

	app := newApp(appName, []helxv1.Service{
		simpleService("main", "nginx:latest", 80, 8080, []string{"nginx", "-g", "daemon off;"}),
	})
	user := newUser(userName)
	inst1 := newInst(instName1, appName, userName)
	inst2 := newInst(instName2, appName, userName)

	registerCleanup(t, inst1, inst2, app, user)
	createObj(t, app)
	createObj(t, user)
	createObj(t, inst1)
	createObj(t, inst2)

	uuid1 := waitForInstUUID(t, instName1)
	uuid2 := waitForInstUUID(t, instName2)

	if uuid1 == uuid2 {
		t.Fatal("expected different UUIDs for different instances")
	}

	deploy1 := waitForDeployment(t, uuid1)
	deploy2 := waitForDeployment(t, uuid2)

	if deploy1.Name == deploy2.Name {
		t.Error("expected different deployment names for different instances")
	}
}

// ---------------------------------------------------------------------------
// Internal helpers for update tests
// ---------------------------------------------------------------------------

// waitForCondition polls until condFn returns true.
func waitForCondition(t *testing.T, timeout time.Duration, condFn func() bool) error {
	t.Helper()
	deadline := time.After(timeout)
	tick := time.NewTicker(pollInterval)
	defer tick.Stop()
	for {
		select {
		case <-deadline:
			return fmt.Errorf("condition not met within %s", timeout)
		case <-tick.C:
			if condFn() {
				return nil
			}
		}
	}
}

// waitForDeploymentQuiet returns the deployment or nil without failing the test.
func waitForDeploymentQuiet(uuid string) *appsv1.Deployment {
	ctx := context.Background()
	list := &appsv1.DeploymentList{}
	if err := k8sClient.List(ctx, list, client.InNamespace(testNS),
		client.MatchingLabels{"helx.renci.org/id": uuid}); err != nil {
		return nil
	}
	if len(list.Items) > 0 {
		return &list.Items[0]
	}
	return nil
}
