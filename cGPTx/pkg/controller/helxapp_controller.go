package controller

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/helxplatform/helxapp-controller/pkg/helxapp"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type HeLxAppReconciler struct {
	client.Client
	clientset *kubernetes.Clientset
	Log       logr.Logger
	Scheme    *runtime.Scheme
}

func NewHeLxAppReconciler(mgr ctrl.Manager, clientset *kubernetes.Clientset) *HeLxAppReconciler {
	return &HeLxAppReconciler{
		Client:    mgr.GetClient(),
		clientset: clientset,
		Log:       ctrl.Log.WithName("controllers").WithName("HeLxApp"),
		Scheme:    mgr.GetScheme(),
	}
}

func (r *HeLxAppReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&helxapp.HeLxApp{}). // Replace with the actual type of your custom resource.
		Complete(r)
}
func (r *HeLxAppReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("helxapp", req.NamespacedName)

	// Fetch the HeLxApp instance
	instance := &helxapp.HeLxApp{}
	err := r.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("HeLxApp not found. Ignoring since object must be deleted.")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}

	// Your reconciliation logic here...

	return ctrl.Result{}, nil
}
