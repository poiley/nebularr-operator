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
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	arrv1alpha1 "github.com/poiley/nebularr-operator/api/v1alpha1"
	"github.com/poiley/nebularr-operator/internal/adapters"
	_ "github.com/poiley/nebularr-operator/internal/adapters/lidarr" // Register lidarr adapter
	"github.com/poiley/nebularr-operator/internal/compiler"
	irv1 "github.com/poiley/nebularr-operator/internal/ir/v1"
)

const lidarrFinalizer = "lidarrconfig.arr.rinzler.cloud/finalizer"

// LidarrConfigReconciler reconciles a LidarrConfig object
type LidarrConfigReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Compiler *compiler.Compiler
	Helper   *ReconcileHelper
}

// +kubebuilder:rbac:groups=arr.rinzler.cloud,resources=lidarrconfigs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=arr.rinzler.cloud,resources=lidarrconfigs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=arr.rinzler.cloud,resources=lidarrconfigs/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop
func (r *LidarrConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	// Fetch the LidarrConfig
	config := &arrv1alpha1.LidarrConfig{}
	if err := r.Get(ctx, req.NamespacedName, config); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("LidarrConfig resource not found, ignoring")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get LidarrConfig")
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
	if !controllerutil.ContainsFinalizer(config, lidarrFinalizer) {
		controllerutil.AddFinalizer(config, lidarrFinalizer)
		if err := r.Update(ctx, config); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// Reconcile the configuration
	return r.reconcileNormal(ctx, config)
}

// reconcileNormal handles the normal reconciliation flow
func (r *LidarrConfigReconciler) reconcileNormal(ctx context.Context, config *arrv1alpha1.LidarrConfig) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	log.Info("Reconciling LidarrConfig", "name", config.Name)

	statusWrapper := &LidarrStatusWrapper{Status: &config.Status}

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
	if err := r.Helper.ResolveIndexerSecrets(ctx, config.Namespace, config.Spec.Indexers, resolvedSecrets); err != nil {
		r.Helper.SetCondition(statusWrapper, config.Generation, ConditionTypeReady, metav1.ConditionFalse, "SecretResolutionFailed", err.Error())
		if statusErr := r.Status().Update(ctx, config); statusErr != nil {
			log.Error(statusErr, "Failed to update status")
		}
		return ctrl.Result{RequeueAfter: ErrorRequeueInterval}, err
	}

	// Resolve import list secrets
	if err := r.Helper.ResolveImportListSecrets(ctx, config.Namespace, config.Spec.ImportLists, resolvedSecrets); err != nil {
		r.Helper.SetCondition(statusWrapper, config.Generation, ConditionTypeReady, metav1.ConditionFalse, "SecretResolutionFailed", err.Error())
		if statusErr := r.Status().Update(ctx, config); statusErr != nil {
			log.Error(statusErr, "Failed to update status")
		}
		return ctrl.Result{RequeueAfter: ErrorRequeueInterval}, err
	}

	// Resolve authentication secrets
	if err := r.Helper.ResolveAuthenticationSecrets(ctx, config.Namespace, config.Spec.Authentication, resolvedSecrets); err != nil {
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
	adapter, ok := adapters.Get(adapters.AppLidarr)
	if !ok {
		r.Helper.SetCondition(statusWrapper, config.Generation, ConditionTypeReady, metav1.ConditionFalse, "AdapterNotFound", "lidarr adapter not registered")
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
	desiredIR, err := r.Compiler.CompileLidarrConfig(ctx, config, resolvedSecrets, caps)
	if err != nil {
		log.Error(err, "Failed to compile LidarrConfig to IR")
		r.Helper.SetCondition(statusWrapper, config.Generation, ConditionTypeReady, metav1.ConditionFalse, "CompilationFailed", err.Error())
		if statusErr := r.Status().Update(ctx, config); statusErr != nil {
			log.Error(statusErr, "Failed to update status")
		}
		return ctrl.Result{RequeueAfter: ErrorRequeueInterval}, err
	}

	// Reconcile using helper (handles quality profiles, download clients, indexers, naming, root folders)
	_, err = r.Helper.ReconcileConfig(ctx, adapters.AppLidarr, connIR, desiredIR, statusWrapper, config.Generation)
	if err != nil {
		if statusErr := r.Status().Update(ctx, config); statusErr != nil {
			log.Error(statusErr, "Failed to update status")
		}
		return ctrl.Result{RequeueAfter: ErrorRequeueInterval}, err
	}

	// Apply direct configuration (import lists, media management, authentication)
	_, err = r.Helper.ApplyDirectConfig(ctx, adapters.AppLidarr, connIR, desiredIR, statusWrapper, config.Generation)
	if err != nil {
		log.Error(err, "Failed to apply direct configuration (non-fatal)")
		// Don't fail reconciliation for direct config errors - they're supplementary
	}

	// Handle Prowlarr auto-registration if prowlarrRef is set
	if config.Spec.Indexers != nil && config.Spec.Indexers.ProwlarrRef != nil {
		reg := ProwlarrAutoRegistration{
			ProwlarrRef: config.Spec.Indexers.ProwlarrRef,
			AppType:     adapters.AppLidarr,
			AppName:     fmt.Sprintf("nebularr-%s-%s", adapters.AppLidarr, config.Name),
			AppURL:      config.Spec.Connection.URL,
			AppAPIKey:   resolvedSecrets["apiKey"],
		}
		if err := r.Helper.HandleProwlarrRegistration(ctx, config.Namespace, reg); err != nil {
			log.Error(err, "Failed to register with Prowlarr (non-fatal)")
		}
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

// reconcileDelete handles deletion of the LidarrConfig
func (r *LidarrConfigReconciler) reconcileDelete(ctx context.Context, config *arrv1alpha1.LidarrConfig) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	log.Info("Handling deletion of LidarrConfig", "name", config.Name)

	// Unregister from Prowlarr if prowlarrRef was set
	if config.Spec.Indexers != nil && config.Spec.Indexers.ProwlarrRef != nil {
		appName := fmt.Sprintf("nebularr-%s-%s", adapters.AppLidarr, config.Name)
		if err := r.Helper.HandleProwlarrUnregistration(ctx, config.Namespace, config.Spec.Indexers.ProwlarrRef.Name, appName); err != nil {
			log.Error(err, "Failed to unregister from Prowlarr (non-fatal)")
		}
	}

	// Try to resolve secrets for cleanup
	resolvedSecrets, err := r.Helper.ResolveConnectionSecrets(ctx, config.Namespace, &config.Spec.Connection)
	if err != nil {
		log.Error(err, "Failed to resolve secrets for cleanup, proceeding anyway")
	} else {
		connIR := &irv1.ConnectionIR{
			URL:    config.Spec.Connection.URL,
			APIKey: resolvedSecrets["apiKey"],
		}
		if err := r.Helper.CleanupManagedResources(ctx, adapters.AppLidarr, connIR); err != nil {
			log.Error(err, "Failed to cleanup managed resources")
		}
	}

	// Remove finalizer
	controllerutil.RemoveFinalizer(config, lidarrFinalizer)
	if err := r.Update(ctx, config); err != nil {
		return ctrl.Result{}, err
	}

	log.Info("Successfully deleted LidarrConfig", "name", config.Name)
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *LidarrConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Initialize compiler if not set
	if r.Compiler == nil {
		r.Compiler = compiler.New()
	}
	// Initialize helper if not set
	if r.Helper == nil {
		r.Helper = NewReconcileHelper(r.Client)
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&arrv1alpha1.LidarrConfig{}).
		Named("lidarrconfig").
		Complete(r)
}
