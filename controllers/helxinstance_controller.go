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

	// Fetch the HelxInstance custom resource
	helxInst := &helxv1.HelxInst{}
	if err := r.Get(ctx, req.NamespacedName, helxInst); err != nil {
		if errors.IsNotFound(err) {
			// Resource is already deleted, return without error
			logger.Info("HelxInstance deleted, nothing to reconcile", "NamespacedName", req.NamespacedName)
			return ctrl.Result{}, nil
		}
		logger.Error(err, "unable to fetch HelxInstance", "NamespacedName", req.NamespacedName)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Log the event and custom resource content
	logger.Info("Reconciling HelxInstance", "HelxInstance", fmt.Sprintf("%+v", helxInst))
	if err := helxapp_operations.CheckInit(ctx); err == nil {
		if artifacts, err := helxapp_operations.CreateDeploymentArtifacts(helxInst); err == nil {
			if artifacts != nil && artifacts.Deployment.Render != "" {
				logger.Info("generated Deployment YAML:")
				logger.Info(artifacts.Deployment.Render)
				if err = helxapp_operations.CreateDeploymentFromYAML(ctx, r.Client, r.Scheme, req, helxInst, artifacts.Deployment); err != nil {
					logger.Error(err, "unable to create deployment", "NamespacedName", req.NamespacedName)
				} else {
					for name, PVC := range artifacts.PVCs {
						if PVC.Render != "" {
							logger.Info("generated PVC YAML:")
							logger.Info(PVC.Render)
							if err = helxapp_operations.CreatePVCFromYAML(ctx, r.Client, r.Scheme, req, helxInst, PVC); err != nil {
								logger.Error(err, "unable to create pvc", "PVCName", name, "NamespacedName", req.NamespacedName)
							}
						}
					}
					for name, service := range artifacts.Services {
						if service.Render != "" {
							logger.Info("generated PVC YAML:")
							logger.Info(service.Render)
							if err = helxapp_operations.CreateServiceFromYAML(ctx, r.Client, r.Scheme, req, helxInst, service); err != nil {
								logger.Error(err, "unable to create service", "Service Name", name, "NamespacedName", req.NamespacedName)
							}
						}
					}
				}
			}
		}
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *HelxInstReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&helxv1.HelxInst{}).
		Complete(r)
}
