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

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	helxv1 "github.com/helxplatform/helxapp/api/v1"
	"github.com/helxplatform/helxapp/helxapp_operations"
	"github.com/kr/pretty"
)

// HelxUserReconciler reconciles a HelxUser object
type HelxUserReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=helx.renci.org,resources=helxusers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=helx.renci.org,resources=helxusers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=helx.renci.org,resources=helxusers/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the HelxUser object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *HelxUserReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	userName := req.NamespacedName.String()

	// Fetch the HelxApp custom resource
	helxUser := &helxv1.HelxUser{}
	if err := r.Get(ctx, req.NamespacedName, helxUser); err != nil {
		if errors.IsNotFound(err) {
			// Resource is already deleted, return without error
			logger.Info("HelxUser deleted", "NamespacedName", req.NamespacedName)
			for _, inst := range helxapp_operations.DeleteUser(userName) {
				if err := helxapp_operations.DeleteDerivatives(&inst, r.Client, req, ctx); err != nil {
					logger.Error(err, "unable to delete derivatives")
				}
			}
			return ctrl.Result{}, nil
		}
		logger.Error(err, "unable to fetch HelxUser", "NamespacedName", req.NamespacedName)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Check if this reconciliation needs to process changes or if it's a resync
	if helxUser.Status.ObservedGeneration >= helxUser.Generation {
		// No changes since last observation
		logger.Info("No updates needed", "NamespacedName", req.NamespacedName)
		helxapp_operations.AddUser(helxUser)
		return ctrl.Result{}, nil
	}

	// Update observed generation after processing
	defer func() {
		helxUser.Status.ObservedGeneration = helxUser.Generation
		if err := r.Status().Update(ctx, helxUser); err != nil {
			logger.Error(err, "Failed to update HelxUser status", "NamespacedName", req.NamespacedName)
		}
	}()

	// Log the event and custom resource content
	logger.Info("Reconciling HelxUser")
	logger.V(1).Info(fmt.Sprintf("%# v\n", pretty.Formatter(helxUser)))
	if instList := helxapp_operations.AddUser(helxUser); len(instList) != 0 {
		for _, inst := range instList {
			if err := helxapp_operations.CreateDerivatives(&inst, r.Client, r.Scheme, req, ctx); err != nil {
				return ctrl.Result{}, err
			}
		}
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *HelxUserReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&helxv1.HelxUser{}).
		Complete(r)
}
