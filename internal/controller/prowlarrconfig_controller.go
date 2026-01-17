/*
Copyright 2026.

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

package controller

import (
	"context"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	arrv1alpha1 "github.com/poiley/nebularr-operator/api/v1alpha1"
	"github.com/poiley/nebularr-operator/internal/adapters"
	_ "github.com/poiley/nebularr-operator/internal/adapters/prowlarr" // Register prowlarr adapter
	"github.com/poiley/nebularr-operator/internal/compiler"
	irv1 "github.com/poiley/nebularr-operator/internal/ir/v1"
)

const prowlarrFinalizer = "prowlarrconfig.arr.rinzler.cloud/finalizer"

// ProwlarrConfigReconciler reconciles a ProwlarrConfig object
type ProwlarrConfigReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Compiler *compiler.Compiler
	Helper   *ReconcileHelper
}

// +kubebuilder:rbac:groups=arr.rinzler.cloud,resources=prowlarrconfigs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=arr.rinzler.cloud,resources=prowlarrconfigs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=arr.rinzler.cloud,resources=prowlarrconfigs/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop
func (r *ProwlarrConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	// Fetch the ProwlarrConfig
	config := &arrv1alpha1.ProwlarrConfig{}
	if err := r.Get(ctx, req.NamespacedName, config); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("ProwlarrConfig resource not found, ignoring")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get ProwlarrConfig")
		return ctrl.Result{}, err
	}

	// Check if reconciliation is suspended
	if config.Spec.Reconciliation != nil && config.Spec.Reconciliation.Suspend {
		log.Info("Reconciliation is suspended")
		return ctrl.Result{}, nil
	}

	// Handle deletion
	if !config.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, config)
	}

	// Ensure finalizer
	if !controllerutil.ContainsFinalizer(config, prowlarrFinalizer) {
		controllerutil.AddFinalizer(config, prowlarrFinalizer)
		if err := r.Update(ctx, config); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// Reconcile the configuration
	return r.reconcileNormal(ctx, config)
}

// reconcileNormal handles the normal reconciliation flow
func (r *ProwlarrConfigReconciler) reconcileNormal(ctx context.Context, config *arrv1alpha1.ProwlarrConfig) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	log.Info("Reconciling ProwlarrConfig", "name", config.Name)

	statusWrapper := &ProwlarrStatusWrapper{Status: &config.Status}

	// Resolve secrets
	resolvedSecrets, err := r.Helper.ResolveConnectionSecrets(ctx, config.Namespace, &config.Spec.Connection)
	if err != nil {
		r.Helper.SetCondition(statusWrapper, config.Generation, ConditionTypeReady, metav1.ConditionFalse, "SecretResolutionFailed", err.Error())
		if statusErr := r.Status().Update(ctx, config); statusErr != nil {
			log.Error(statusErr, "Failed to update status")
		}
		return ctrl.Result{RequeueAfter: ErrorRequeueInterval}, err
	}

	// Resolve download client secrets
	if err := r.Helper.ResolveDownloadClientSecrets(ctx, config.Namespace, config.Spec.DownloadClients, resolvedSecrets); err != nil {
		r.Helper.SetCondition(statusWrapper, config.Generation, ConditionTypeReady, metav1.ConditionFalse, "SecretResolutionFailed", err.Error())
		if statusErr := r.Status().Update(ctx, config); statusErr != nil {
			log.Error(statusErr, "Failed to update status")
		}
		return ctrl.Result{RequeueAfter: ErrorRequeueInterval}, err
	}

	// Resolve indexer secrets
	if err := r.Helper.ResolveProwlarrIndexerSecrets(ctx, config.Namespace, config.Spec.Indexers, resolvedSecrets); err != nil {
		r.Helper.SetCondition(statusWrapper, config.Generation, ConditionTypeReady, metav1.ConditionFalse, "SecretResolutionFailed", err.Error())
		if statusErr := r.Status().Update(ctx, config); statusErr != nil {
			log.Error(statusErr, "Failed to update status")
		}
		return ctrl.Result{RequeueAfter: ErrorRequeueInterval}, err
	}

	// Resolve proxy secrets
	if err := r.Helper.ResolveProxySecrets(ctx, config.Namespace, config.Spec.Proxies, resolvedSecrets); err != nil {
		r.Helper.SetCondition(statusWrapper, config.Generation, ConditionTypeReady, metav1.ConditionFalse, "SecretResolutionFailed", err.Error())
		if statusErr := r.Status().Update(ctx, config); statusErr != nil {
			log.Error(statusErr, "Failed to update status")
		}
		return ctrl.Result{RequeueAfter: ErrorRequeueInterval}, err
	}

	// Resolve application secrets
	if err := r.Helper.ResolveApplicationSecrets(ctx, config.Namespace, config.Spec.Applications, resolvedSecrets); err != nil {
		r.Helper.SetCondition(statusWrapper, config.Generation, ConditionTypeReady, metav1.ConditionFalse, "SecretResolutionFailed", err.Error())
		if statusErr := r.Status().Update(ctx, config); statusErr != nil {
			log.Error(statusErr, "Failed to update status")
		}
		return ctrl.Result{RequeueAfter: ErrorRequeueInterval}, err
	}

	// Create connection IR
	connIR := &irv1.ConnectionIR{
		URL:    config.Spec.Connection.URL,
		APIKey: resolvedSecrets["apiKey"],
	}

	// Get capabilities for compilation
	adapter, ok := adapters.Get(adapters.AppProwlarr)
	if !ok {
		r.Helper.SetCondition(statusWrapper, config.Generation, ConditionTypeReady, metav1.ConditionFalse, "AdapterNotFound", "prowlarr adapter not registered")
		if statusErr := r.Status().Update(ctx, config); statusErr != nil {
			log.Error(statusErr, "Failed to update status")
		}
		return ctrl.Result{RequeueAfter: ErrorRequeueInterval}, nil
	}

	caps, err := adapter.Discover(ctx, connIR)
	if err != nil {
		r.Helper.SetCondition(statusWrapper, config.Generation, ConditionTypeReady, metav1.ConditionFalse, "DiscoveryFailed", err.Error())
		if statusErr := r.Status().Update(ctx, config); statusErr != nil {
			log.Error(statusErr, "Failed to update status")
		}
		return ctrl.Result{RequeueAfter: ErrorRequeueInterval}, err
	}

	// Compile CRD to IR
	desiredIR, err := r.Compiler.CompileProwlarrConfig(ctx, config, resolvedSecrets, caps)
	if err != nil {
		log.Error(err, "Failed to compile ProwlarrConfig to IR")
		r.Helper.SetCondition(statusWrapper, config.Generation, ConditionTypeReady, metav1.ConditionFalse, "CompilationFailed", err.Error())
		if statusErr := r.Status().Update(ctx, config); statusErr != nil {
			log.Error(statusErr, "Failed to update status")
		}
		return ctrl.Result{RequeueAfter: ErrorRequeueInterval}, err
	}

	// Reconcile using helper
	_, err = r.Helper.ReconcileConfig(ctx, adapters.AppProwlarr, connIR, desiredIR, statusWrapper, config.Generation)
	if err != nil {
		if statusErr := r.Status().Update(ctx, config); statusErr != nil {
			log.Error(statusErr, "Failed to update status")
		}
		return ctrl.Result{RequeueAfter: ErrorRequeueInterval}, err
	}

	// Update status
	if err := r.Status().Update(ctx, config); err != nil {
		log.Error(err, "Failed to update status")
		return ctrl.Result{}, err
	}

	// Determine requeue interval
	requeueAfter := DefaultRequeueInterval
	if config.Spec.Reconciliation != nil && config.Spec.Reconciliation.Interval != nil {
		requeueAfter = config.Spec.Reconciliation.Interval.Duration
	}

	return ctrl.Result{RequeueAfter: requeueAfter}, nil
}

// reconcileDelete handles deletion of the ProwlarrConfig
func (r *ProwlarrConfigReconciler) reconcileDelete(ctx context.Context, config *arrv1alpha1.ProwlarrConfig) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	log.Info("Handling deletion of ProwlarrConfig", "name", config.Name)

	// Try to resolve secrets for cleanup
	resolvedSecrets, err := r.Helper.ResolveConnectionSecrets(ctx, config.Namespace, &config.Spec.Connection)
	if err != nil {
		log.Error(err, "Failed to resolve secrets for cleanup, proceeding anyway")
	} else {
		connIR := &irv1.ConnectionIR{
			URL:    config.Spec.Connection.URL,
			APIKey: resolvedSecrets["apiKey"],
		}
		if err := r.Helper.CleanupManagedResources(ctx, adapters.AppProwlarr, connIR); err != nil {
			log.Error(err, "Failed to cleanup managed resources")
		}
	}

	// Remove finalizer
	controllerutil.RemoveFinalizer(config, prowlarrFinalizer)
	if err := r.Update(ctx, config); err != nil {
		return ctrl.Result{}, err
	}

	log.Info("Successfully deleted ProwlarrConfig", "name", config.Name)
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ProwlarrConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Initialize compiler if not set
	if r.Compiler == nil {
		r.Compiler = compiler.New()
	}
	// Initialize helper if not set
	if r.Helper == nil {
		r.Helper = NewReconcileHelper(r.Client)
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&arrv1alpha1.ProwlarrConfig{}).
		Named("prowlarrconfig").
		Complete(r)
}
