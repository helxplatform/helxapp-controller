package main

import (
	"flag"

	"github.com/helxplatform/helxapp-controller/pkg/controller"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func main() {
	var kubeconfig string
	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
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

	mgr, err := ctrl.NewManager(config, ctrl.Options{})
	if err != nil {
		panic(err)
	}

	if err := controller.NewHeLxAppReconciler(mgr, clientset).SetupWithManager(mgr); err != nil {
		panic(err)
	}

	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		panic(err)
	}
}

func getKubeConfig(kubeconfig string) (*rest.Config, error) {
	if kubeconfig != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}
	return rest.InClusterConfig()
}
