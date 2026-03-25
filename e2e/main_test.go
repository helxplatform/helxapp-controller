package e2e

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"testing"
	"time"

	helxv1 "github.com/helxplatform/helxapp-controller/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	pollInterval = 2 * time.Second
	waitTimeout  = 60 * time.Second
)

var (
	k8sClient client.Client
	testNS    string
)

// ---------------------------------------------------------------------------
// Driver
// ---------------------------------------------------------------------------

func TestMain(m *testing.M) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		loadingRules, &clientcmd.ConfigOverrides{})

	ns, _, err := kubeConfig.Namespace()
	if err != nil {
		fmt.Fprintf(os.Stderr, "e2e: failed to get namespace from kubeconfig: %v\n", err)
		os.Exit(1)
	}
	testNS = ns

	cfg, err := kubeConfig.ClientConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "e2e: failed to build REST config: %v\n", err)
		os.Exit(1)
	}

	if err := helxv1.AddToScheme(scheme.Scheme); err != nil {
		fmt.Fprintf(os.Stderr, "e2e: failed to register CRD scheme: %v\n", err)
		os.Exit(1)
	}

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	if err != nil {
		fmt.Fprintf(os.Stderr, "e2e: failed to create k8s client: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()

	// Preflight: verify CRDs are installed
	for _, list := range []client.ObjectList{
		&helxv1.HelxAppList{},
		&helxv1.HelxUserList{},
		&helxv1.HelxInstList{},
	} {
		if err := k8sClient.List(ctx, list, client.InNamespace(testNS)); err != nil {
			fmt.Fprintf(os.Stderr, "e2e: CRD not available (%T): %v\n", list, err)
			os.Exit(1)
		}
	}

	// Preflight: verify controller pod is running in namespace
	podList := &corev1.PodList{}
	if err := k8sClient.List(ctx, podList, client.InNamespace(testNS),
		client.MatchingLabels{"app.kubernetes.io/name": "helxapp-controller"}); err != nil {
		fmt.Fprintf(os.Stderr, "e2e: cannot list controller pods: %v\n", err)
		os.Exit(1)
	}
	running := false
	for _, pod := range podList.Items {
		if pod.Status.Phase == corev1.PodRunning {
			running = true
			break
		}
	}
	if !running {
		fmt.Fprintf(os.Stderr, "e2e: no running controller pod (label app.kubernetes.io/name=helxapp-controller) in namespace %s\n", testNS)
		os.Exit(1)
	}

	fmt.Printf("e2e: namespace=%s — running tests\n", testNS)
	os.Exit(m.Run())
}

// ---------------------------------------------------------------------------
// Helpers — resource constructors
// ---------------------------------------------------------------------------

func suffix() string {
	b := make([]byte, 3)
	rand.Read(b)
	return hex.EncodeToString(b) // 6 hex chars
}

func newApp(name string, services []helxv1.Service) *helxv1.HelxApp {
	return &helxv1.HelxApp{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: testNS},
		Spec:       helxv1.HelxAppSpec{AppClassName: name, Services: services},
	}
}

func newUser(name string) *helxv1.HelxUser {
	return &helxv1.HelxUser{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: testNS},
		Spec:       helxv1.HelxUserSpec{},
	}
}

func newInst(name, appName, userName string) *helxv1.HelxInst {
	return &helxv1.HelxInst{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: testNS},
		Spec:       helxv1.HelxInstSpec{AppName: appName, UserName: userName},
	}
}

func newInstWithSC(name, appName, userName string, sc *helxv1.SecurityContext) *helxv1.HelxInst {
	return &helxv1.HelxInst{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: testNS},
		Spec:       helxv1.HelxInstSpec{AppName: appName, UserName: userName, SecurityContext: sc},
	}
}

func newInstWithResources(name, appName, userName string, res map[string]helxv1.Resources) *helxv1.HelxInst {
	return &helxv1.HelxInst{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: testNS},
		Spec:       helxv1.HelxInstSpec{AppName: appName, UserName: userName, Resources: res},
	}
}

// ---------------------------------------------------------------------------
// Helpers — CRUD
// ---------------------------------------------------------------------------

func createObj(t *testing.T, obj client.Object) {
	t.Helper()
	ctx := context.Background()
	if err := k8sClient.Create(ctx, obj); err != nil {
		t.Fatalf("create %T %s: %v", obj, obj.GetName(), err)
	}
}

func deleteObj(t *testing.T, obj client.Object) {
	t.Helper()
	ctx := context.Background()
	if err := k8sClient.Delete(ctx, obj); err != nil && !apierrors.IsNotFound(err) {
		t.Logf("cleanup delete %T %s: %v", obj, obj.GetName(), err)
	}
}

func updateObj(t *testing.T, obj client.Object) {
	t.Helper()
	ctx := context.Background()
	if err := k8sClient.Update(ctx, obj); err != nil {
		t.Fatalf("update %T %s: %v", obj, obj.GetName(), err)
	}
}

// registerCleanup schedules deletion of all objects at test end.
func registerCleanup(t *testing.T, objs ...client.Object) {
	t.Helper()
	t.Cleanup(func() {
		for _, obj := range objs {
			deleteObj(t, obj)
		}
	})
}

// ---------------------------------------------------------------------------
// Helpers — wait functions
// ---------------------------------------------------------------------------

// waitForInstUUID polls until the HelxInst has a non-empty Status.UUID.
func waitForInstUUID(t *testing.T, name string) string {
	t.Helper()
	var uuid string
	ctx := context.Background()
	err := wait.PollImmediate(pollInterval, waitTimeout, func() (bool, error) {
		inst := &helxv1.HelxInst{}
		if err := k8sClient.Get(ctx, types.NamespacedName{Name: name, Namespace: testNS}, inst); err != nil {
			return false, nil
		}
		if inst.Status.UUID != "" {
			uuid = inst.Status.UUID
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		t.Fatalf("timed out waiting for UUID on HelxInst %s", name)
	}
	return uuid
}

// waitForDeployment polls until a Deployment with label helx.renci.org/id=<uuid> exists.
func waitForDeployment(t *testing.T, uuid string) *appsv1.Deployment {
	t.Helper()
	var found *appsv1.Deployment
	ctx := context.Background()
	err := wait.PollImmediate(pollInterval, waitTimeout, func() (bool, error) {
		list := &appsv1.DeploymentList{}
		if err := k8sClient.List(ctx, list, client.InNamespace(testNS),
			client.MatchingLabels{"helx.renci.org/id": uuid}); err != nil {
			return false, nil
		}
		if len(list.Items) > 0 {
			found = &list.Items[0]
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		t.Fatalf("timed out waiting for Deployment with helx.renci.org/id=%s", uuid)
	}
	return found
}

// waitForService polls until a Service with label helx.renci.org/id=<uuid> exists.
func waitForService(t *testing.T, uuid string) *corev1.Service {
	t.Helper()
	var found *corev1.Service
	ctx := context.Background()
	err := wait.PollImmediate(pollInterval, waitTimeout, func() (bool, error) {
		list := &corev1.ServiceList{}
		if err := k8sClient.List(ctx, list, client.InNamespace(testNS),
			client.MatchingLabels{"helx.renci.org/id": uuid}); err != nil {
			return false, nil
		}
		if len(list.Items) > 0 {
			found = &list.Items[0]
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		t.Fatalf("timed out waiting for Service with helx.renci.org/id=%s", uuid)
	}
	return found
}

// waitForPVC polls until a PVC with the given name exists.
func waitForPVC(t *testing.T, name string) *corev1.PersistentVolumeClaim {
	t.Helper()
	var found *corev1.PersistentVolumeClaim
	ctx := context.Background()
	err := wait.PollImmediate(pollInterval, waitTimeout, func() (bool, error) {
		pvc := &corev1.PersistentVolumeClaim{}
		if err := k8sClient.Get(ctx, types.NamespacedName{Name: name, Namespace: testNS}, pvc); err != nil {
			return false, nil
		}
		found = pvc
		return true, nil
	})
	if err != nil {
		t.Fatalf("timed out waiting for PVC %s", name)
	}
	return found
}

// waitForNoDeployment polls until no Deployment with label helx.renci.org/id=<uuid> exists.
func waitForNoDeployment(t *testing.T, uuid string) {
	t.Helper()
	ctx := context.Background()
	err := wait.PollImmediate(pollInterval, waitTimeout, func() (bool, error) {
		list := &appsv1.DeploymentList{}
		if err := k8sClient.List(ctx, list, client.InNamespace(testNS),
			client.MatchingLabels{"helx.renci.org/id": uuid}); err != nil {
			return false, nil
		}
		return len(list.Items) == 0, nil
	})
	if err != nil {
		t.Fatalf("timed out waiting for Deployment with helx.renci.org/id=%s to be deleted", uuid)
	}
}

// waitForNoService polls until no Service with label helx.renci.org/id=<uuid> exists.
func waitForNoService(t *testing.T, uuid string) {
	t.Helper()
	ctx := context.Background()
	err := wait.PollImmediate(pollInterval, waitTimeout, func() (bool, error) {
		list := &corev1.ServiceList{}
		if err := k8sClient.List(ctx, list, client.InNamespace(testNS),
			client.MatchingLabels{"helx.renci.org/id": uuid}); err != nil {
			return false, nil
		}
		return len(list.Items) == 0, nil
	})
	if err != nil {
		t.Fatalf("timed out waiting for Service with helx.renci.org/id=%s to be deleted", uuid)
	}
}

// noServiceExists returns true if no Service with the given UUID label exists right now.
func noServiceExists(t *testing.T, uuid string) bool {
	t.Helper()
	ctx := context.Background()
	list := &corev1.ServiceList{}
	if err := k8sClient.List(ctx, list, client.InNamespace(testNS),
		client.MatchingLabels{"helx.renci.org/id": uuid}); err != nil {
		return false
	}
	return len(list.Items) == 0
}

// pvcExists returns true if a PVC with the given name exists right now.
func pvcExists(t *testing.T, name string) bool {
	t.Helper()
	ctx := context.Background()
	pvc := &corev1.PersistentVolumeClaim{}
	err := k8sClient.Get(ctx, types.NamespacedName{Name: name, Namespace: testNS}, pvc)
	return err == nil
}

// waitForObservedGeneration polls until the object's ObservedGeneration >= Generation.
func waitForObservedGeneration(t *testing.T, name string, obj client.Object) {
	t.Helper()
	ctx := context.Background()
	err := wait.PollImmediate(pollInterval, waitTimeout, func() (bool, error) {
		if err := k8sClient.Get(ctx, types.NamespacedName{Name: name, Namespace: testNS}, obj); err != nil {
			return false, nil
		}
		switch o := obj.(type) {
		case *helxv1.HelxApp:
			return o.Status.ObservedGeneration >= o.Generation, nil
		case *helxv1.HelxUser:
			return o.Status.ObservedGeneration >= o.Generation, nil
		case *helxv1.HelxInst:
			return o.Status.ObservedGeneration >= o.Generation, nil
		}
		return false, fmt.Errorf("unsupported type %T", obj)
	})
	if err != nil {
		t.Fatalf("timed out waiting for ObservedGeneration on %T %s", obj, name)
	}
}

// simpleService returns a minimal service spec for test Apps.
func simpleService(name, image string, containerPort, port int32, command []string) helxv1.Service {
	return helxv1.Service{
		Name:    name,
		Image:   image,
		Command: command,
		Ports:   []helxv1.PortMap{{ContainerPort: containerPort, Port: port}},
	}
}
