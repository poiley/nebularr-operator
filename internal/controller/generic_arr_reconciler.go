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
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	arrv1alpha1 "github.com/poiley/nebularr-operator/api/v1alpha1"
	"github.com/poiley/nebularr-operator/internal/adapters"
	"github.com/poiley/nebularr-operator/internal/compiler"
	irv1 "github.com/poiley/nebularr-operator/internal/ir/v1"
)

// ArrConfigObject defines the interface that all *arr config CRDs must implement.
// This enables generic reconciliation logic across Sonarr, Radarr, Lidarr, and Readarr.
type ArrConfigObject interface {
	// GetConnectionSpec returns the connection specification
	GetConnectionSpec() *arrv1alpha1.ConnectionSpec

	// GetReconciliationSpec returns the reconciliation configuration (may be nil)
	GetReconciliationSpec() *arrv1alpha1.ReconciliationSpec

	// GetDownloadClients returns the download client specs
	GetDownloadClients() []arrv1alpha1.DownloadClientSpec

	// GetIndexersSpec returns the indexers specification (may be nil)
	GetIndexersSpec() *arrv1alpha1.IndexersSpec

	// GetImportLists returns the import list specs
	GetImportLists() []arrv1alpha1.ImportListSpec

	// GetAuthenticationSpec returns the authentication specification (may be nil)
	GetAuthenticationSpec() *arrv1alpha1.AuthenticationSpec

	// GetStatusWrapper returns a ConfigStatus wrapper for updating status
	GetStatusWrapper() ConfigStatus

	// GetHealthStatusPtr returns a pointer to the Health field in the status
	GetHealthStatusPtr() **arrv1alpha1.HealthStatus

	// GetAppType returns the adapter type (e.g., adapters.AppSonarr)
	GetAppType() string

	// GetFinalizerName returns the finalizer name for this resource
	GetFinalizerName() string

	// ShouldRegisterWithProwlarr returns true if this app should auto-register with Prowlarr
	// (Lidarr uses ProwlarrCoordinator, so it returns false)
	ShouldRegisterWithProwlarr() bool

	// GetObject returns the underlying client.Object (the actual CRD)
	// This is needed because the k8s client needs the concrete type, not the adapter wrapper.
	GetObject() client.Object
}

// ArrConfigCompiler is a function type that compiles a config CRD to IR
type ArrConfigCompiler func(ctx context.Context, c *compiler.Compiler, config ArrConfigObject, secrets map[string]string, caps *adapters.Capabilities) (*irv1.IR, error)

// GenericArrReconciler provides shared reconciliation logic for all *arr config controllers.
// Each controller instantiates this with type-specific callbacks.
type GenericArrReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Compiler *compiler.Compiler
	Helper   *ReconcileHelper
	Recorder record.EventRecorder

	// CompileConfig is the type-specific compile function
	CompileConfig ArrConfigCompiler
}

// Reconcile handles the main reconciliation loop for any *arr config
func (r *GenericArrReconciler) Reconcile(ctx context.Context, config ArrConfigObject) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	obj := config.GetObject()

	// Check if reconciliation is suspended
	if spec := config.GetReconciliationSpec(); spec != nil && spec.Suspend {
		log.Info("Reconciliation is suspended")
		return ctrl.Result{}, nil
	}

	finalizerName := config.GetFinalizerName()

	// Handle deletion
	if !obj.GetDeletionTimestamp().IsZero() {
		return r.reconcileDelete(ctx, config, finalizerName)
	}

	// Ensure finalizer
	if !controllerutil.ContainsFinalizer(obj, finalizerName) {
		controllerutil.AddFinalizer(obj, finalizerName)
		if err := r.Update(ctx, obj); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// Reconcile the configuration
	return r.reconcileNormal(ctx, config)
}

// reconcileNormal handles the normal reconciliation flow
func (r *GenericArrReconciler) reconcileNormal(ctx context.Context, config ArrConfigObject) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	obj := config.GetObject()
	appType := config.GetAppType()
	log.Info(fmt.Sprintf("Reconciling %sConfig", appType), "name", obj.GetName())

	statusWrapper := config.GetStatusWrapper()
	generation := obj.GetGeneration()
	namespace := obj.GetNamespace()
	connSpec := config.GetConnectionSpec()

	// Resolve connection secrets
	resolvedSecrets, err := r.Helper.ResolveConnectionSecrets(ctx, namespace, connSpec)
	if err != nil {
		r.Helper.SetCondition(statusWrapper, generation, ConditionTypeReady, metav1.ConditionFalse, "SecretResolutionFailed", err.Error())
		if statusErr := r.updateStatus(ctx, config); statusErr != nil {
			log.Error(statusErr, "Failed to update status")
		}
		return ctrl.Result{RequeueAfter: ErrorRequeueInterval}, err
	}

	// Resolve download client secrets
	if err := r.Helper.ResolveDownloadClientSecrets(ctx, namespace, config.GetDownloadClients(), resolvedSecrets); err != nil {
		r.Helper.SetCondition(statusWrapper, generation, ConditionTypeReady, metav1.ConditionFalse, "SecretResolutionFailed", err.Error())
		if statusErr := r.updateStatus(ctx, config); statusErr != nil {
			log.Error(statusErr, "Failed to update status")
		}
		return ctrl.Result{RequeueAfter: ErrorRequeueInterval}, err
	}

	// Resolve indexer secrets
	if err := r.Helper.ResolveIndexerSecrets(ctx, namespace, config.GetIndexersSpec(), resolvedSecrets); err != nil {
		r.Helper.SetCondition(statusWrapper, generation, ConditionTypeReady, metav1.ConditionFalse, "SecretResolutionFailed", err.Error())
		if statusErr := r.updateStatus(ctx, config); statusErr != nil {
			log.Error(statusErr, "Failed to update status")
		}
		return ctrl.Result{RequeueAfter: ErrorRequeueInterval}, err
	}

	// Resolve import list secrets
	if err := r.Helper.ResolveImportListSecrets(ctx, namespace, config.GetImportLists(), resolvedSecrets); err != nil {
		r.Helper.SetCondition(statusWrapper, generation, ConditionTypeReady, metav1.ConditionFalse, "SecretResolutionFailed", err.Error())
		if statusErr := r.updateStatus(ctx, config); statusErr != nil {
			log.Error(statusErr, "Failed to update status")
		}
		return ctrl.Result{RequeueAfter: ErrorRequeueInterval}, err
	}

	// Resolve authentication secrets
	if err := r.Helper.ResolveAuthenticationSecrets(ctx, namespace, config.GetAuthenticationSpec(), resolvedSecrets); err != nil {
		r.Helper.SetCondition(statusWrapper, generation, ConditionTypeReady, metav1.ConditionFalse, "SecretResolutionFailed", err.Error())
		if statusErr := r.updateStatus(ctx, config); statusErr != nil {
			log.Error(statusErr, "Failed to update status")
		}
		return ctrl.Result{RequeueAfter: ErrorRequeueInterval}, err
	}

	// Create connection IR
	connIR := &irv1.ConnectionIR{
		URL:    connSpec.URL,
		APIKey: resolvedSecrets["apiKey"],
	}

	// Get adapter and capabilities
	adapter, ok := adapters.Get(appType)
	if !ok {
		r.Helper.SetCondition(statusWrapper, generation, ConditionTypeReady, metav1.ConditionFalse, "AdapterNotFound", fmt.Sprintf("%s adapter not registered", appType))
		if statusErr := r.updateStatus(ctx, config); statusErr != nil {
			log.Error(statusErr, "Failed to update status")
		}
		return ctrl.Result{RequeueAfter: ErrorRequeueInterval}, nil
	}

	caps, err := adapter.Discover(ctx, connIR)
	if err != nil {
		r.Helper.SetCondition(statusWrapper, generation, ConditionTypeReady, metav1.ConditionFalse, "DiscoveryFailed", err.Error())
		if statusErr := r.updateStatus(ctx, config); statusErr != nil {
			log.Error(statusErr, "Failed to update status")
		}
		return ctrl.Result{RequeueAfter: ErrorRequeueInterval}, err
	}

	// Compile CRD to IR using type-specific compiler
	desiredIR, err := r.CompileConfig(ctx, r.Compiler, config, resolvedSecrets, caps)
	if err != nil {
		log.Error(err, fmt.Sprintf("Failed to compile %sConfig to IR", appType))
		r.Helper.SetCondition(statusWrapper, generation, ConditionTypeReady, metav1.ConditionFalse, "CompilationFailed", err.Error())
		if statusErr := r.updateStatus(ctx, config); statusErr != nil {
			log.Error(statusErr, "Failed to update status")
		}
		return ctrl.Result{RequeueAfter: ErrorRequeueInterval}, err
	}

	// Reconcile using helper
	_, err = r.Helper.ReconcileConfig(ctx, appType, connIR, desiredIR, statusWrapper, generation)
	if err != nil {
		if statusErr := r.updateStatus(ctx, config); statusErr != nil {
			log.Error(statusErr, "Failed to update status")
		}
		return ctrl.Result{RequeueAfter: ErrorRequeueInterval}, err
	}

	// Apply direct configuration (import lists, media management, authentication)
	_, err = r.Helper.ApplyDirectConfig(ctx, appType, connIR, desiredIR, statusWrapper, generation)
	if err != nil {
		log.Error(err, "Failed to apply direct configuration (non-fatal)")
	}

	// Handle Prowlarr auto-registration if enabled for this type
	if config.ShouldRegisterWithProwlarr() {
		if indexersSpec := config.GetIndexersSpec(); indexersSpec != nil && indexersSpec.ProwlarrRef != nil {
			reg := ProwlarrAutoRegistration{
				ProwlarrRef: indexersSpec.ProwlarrRef,
				AppType:     appType,
				AppName:     fmt.Sprintf("nebularr-%s-%s", appType, obj.GetName()),
				AppURL:      connSpec.URL,
				AppAPIKey:   resolvedSecrets["apiKey"],
			}
			if err := r.Helper.HandleProwlarrRegistration(ctx, namespace, reg); err != nil {
				log.Error(err, "Failed to register with Prowlarr (non-fatal)")
			}
		}
	}

	// Check health and emit events
	healthStatus := r.Helper.CheckAndReportHealth(ctx, appType, connIR, obj, r.Recorder)
	if healthStatus != nil {
		if healthPtr := config.GetHealthStatusPtr(); healthPtr != nil {
			*healthPtr = healthStatus
		}
	}

	// Update status
	if err := r.updateStatus(ctx, config); err != nil {
		log.Error(err, "Failed to update status")
		return ctrl.Result{}, err
	}

	// Determine requeue interval
	requeueAfter := DefaultRequeueInterval
	if spec := config.GetReconciliationSpec(); spec != nil && spec.Interval != nil {
		requeueAfter = spec.Interval.Duration
	}

	return ctrl.Result{RequeueAfter: requeueAfter}, nil
}

// reconcileDelete handles deletion
func (r *GenericArrReconciler) reconcileDelete(ctx context.Context, config ArrConfigObject, finalizerName string) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	obj := config.GetObject()
	appType := config.GetAppType()
	log.Info(fmt.Sprintf("Handling deletion of %sConfig", appType), "name", obj.GetName())

	namespace := obj.GetNamespace()

	// Unregister from Prowlarr if prowlarrRef was set
	if indexersSpec := config.GetIndexersSpec(); indexersSpec != nil && indexersSpec.ProwlarrRef != nil {
		appName := fmt.Sprintf("nebularr-%s-%s", appType, obj.GetName())
		if err := r.Helper.HandleProwlarrUnregistration(ctx, namespace, indexersSpec.ProwlarrRef.Name, appName); err != nil {
			log.Error(err, "Failed to unregister from Prowlarr (non-fatal)")
		}
	}

	// Try to resolve secrets for cleanup
	connSpec := config.GetConnectionSpec()
	resolvedSecrets, err := r.Helper.ResolveConnectionSecrets(ctx, namespace, connSpec)
	if err != nil {
		log.Error(err, "Failed to resolve secrets for cleanup, proceeding anyway")
	} else {
		connIR := &irv1.ConnectionIR{
			URL:    connSpec.URL,
			APIKey: resolvedSecrets["apiKey"],
		}
		if err := r.Helper.CleanupManagedResources(ctx, appType, connIR); err != nil {
			log.Error(err, "Failed to cleanup managed resources")
		}
	}

	// Remove finalizer
	controllerutil.RemoveFinalizer(obj, finalizerName)
	if err := r.Update(ctx, obj); err != nil {
		return ctrl.Result{}, err
	}

	log.Info(fmt.Sprintf("Successfully deleted %sConfig", appType), "name", obj.GetName())
	return ctrl.Result{}, nil
}

// updateStatus updates the status subresource of the config
func (r *GenericArrReconciler) updateStatus(ctx context.Context, config ArrConfigObject) error {
	return r.Status().Update(ctx, config.GetObject())
}

// ConfigFetcher provides type-specific fetch and wrap functionality.
// This allows the generic reconciler to work with the concrete CRD types
// while keeping the reconciliation logic generic.
type ConfigFetcher interface {
	// NewEmpty returns a new empty CRD object to be populated by the k8s client
	NewEmpty() client.Object
	// Wrap wraps the fetched CRD object in an ArrConfigObject adapter
	Wrap(obj client.Object) ArrConfigObject
}

// FetchAndReconcile fetches the config object and runs reconciliation.
// This is a helper for the thin controller wrappers.
func (r *GenericArrReconciler) FetchAndReconcile(
	ctx context.Context,
	req ctrl.Request,
	fetcher ConfigFetcher,
) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	obj := fetcher.NewEmpty()
	if err := r.Get(ctx, req.NamespacedName, obj); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("Resource not found, ignoring")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get resource")
		return ctrl.Result{}, err
	}

	config := fetcher.Wrap(obj)
	return r.Reconcile(ctx, config)
}
