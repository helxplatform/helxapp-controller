/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	helxv1 "github.com/helxplatform/helxapp/api/v1"
	"github.com/helxplatform/helxapp/helxapp_operations"
	"github.com/kr/pretty"
)

// HelxInstanceReconciler reconciles a HelxInstance object
type HelxInstReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=helx.renci.org,namespace=jeffw,resources=helxinsts,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=helx.renci.org,namespace=jeffw,resources=helxinsts/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=helx.renci.org,namespace=jeffw,resources=helxinsts/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the HelxInstance object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.4/pkg/reconcile
func (r *HelxInstReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	instName := req.NamespacedName.String()

	// Fetch the HelxInstance custom resource
	helxInst := &helxv1.HelxInst{}
	if err := r.Get(ctx, req.NamespacedName, helxInst); err != nil {
		if errors.IsNotFound(err) {
			// Resource is already deleted, return without error
			logger.Info("HelxInstance deleted", "NamespacedName", req.NamespacedName)
			helxapp_operations.DeleteInst(instName)
			return ctrl.Result{}, nil
		}
		logger.Error(err, "unable to fetch HelxInstance", "NamespacedName", req.NamespacedName)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Check if this reconciliation needs to process changes or if it's a resync
	if helxInst.Status.ObservedGeneration >= helxInst.Generation {
		// No changes since last observation
		logger.Info("No updates needed", "NamespacedName", req.NamespacedName)
		helxapp_operations.AddInst(helxInst)
		return ctrl.Result{}, nil
	}

	if helxInst.Status.UUID == "" {
		helxInst.Status.UUID = uuid.New().String()
	}
	// Update observed generation after processing
	defer func() {
		helxInst.Status.ObservedGeneration = helxInst.Generation
		if err := r.Status().Update(ctx, helxInst); err != nil {
			logger.Error(err, "Failed to update HelxInstance status", "NamespacedName", req.NamespacedName)
		}
	}()

	// Log the event and custom resource content
	logger.Info("Reconciling HelxInstance")
	logger.V(1).Info(fmt.Sprintf("%# v\n", pretty.Formatter(helxInst)))
	helxapp_operations.AddInst(helxInst)
	return ctrl.Result{}, helxapp_operations.CreateDerivatives(helxInst, r.Client, r.Scheme, req, ctx)
}

// SetupWithManager sets up the controller with the Manager.
func (r *HelxInstReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&helxv1.HelxInst{}).
		Complete(r)
}
