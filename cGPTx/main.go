package main

import (
	"flag"
	"net/http"
	"time"

	"github.com/helxplatform/helxapp-controller/pkg/controller"
	"github.com/helxplatform/helxapp-controller/pkg/helxapp"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

func startReadinessProbe() {
	http.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	if err := http.ListenAndServe(":8081", nil); err != nil {
		panic(err)
	}
}

func startLivenessProbe() {
	http.HandleFunc("/live", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	if err := http.ListenAndServe(":8082", nil); err != nil {
		panic(err)
	}
}

func getKubeConfig(kubeconfig string) (*rest.Config, error) {
	if kubeconfig != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}
	return rest.InClusterConfig()
}

func startManager(mgr manager.Manager) {
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		panic(err)
	}
}

func main() {
	var kubeconfig string

	go startLivenessProbe()

	flag.Parse()
	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))

	config, err := getKubeConfig(kubeconfig)
	if err != nil {
		panic(err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	mgr, err := ctrl.NewManager(config, ctrl.Options{SyncPeriod: pointer.Duration(60 * time.Second)})
	if err != nil {
		panic(err)
	}

	// Register your custom resource type with the scheme
	if err := helxapp.AddToScheme(mgr.GetScheme()); err != nil {
		panic(err)
	}

	if err := controller.NewHeLxAppReconciler(mgr, clientset).SetupWithManager(mgr); err != nil {
		panic(err)
	}

	go startManager(mgr)
	go startReadinessProbe()

	select {}
}
