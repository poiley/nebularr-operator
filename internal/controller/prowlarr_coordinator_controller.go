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
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	arrv1alpha1 "github.com/poiley/nebularr-operator/api/v1alpha1"
	irv1 "github.com/poiley/nebularr-operator/internal/ir/v1"
	"github.com/poiley/nebularr-operator/internal/prowlarr"
)

// ProwlarrCoordinatorReconciler coordinates Pull Model registration between
// *arr apps and Prowlarr. It watches for apps with spec.indexers.prowlarrRef
// and automatically registers them with the referenced ProwlarrConfig.
type ProwlarrCoordinatorReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Helper *ReconcileHelper
}

// +kubebuilder:rbac:groups=arr.rinzler.cloud,resources=prowlarrconfigs,verbs=get;list;watch
// +kubebuilder:rbac:groups=arr.rinzler.cloud,resources=radarrconfigs,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=arr.rinzler.cloud,resources=radarrconfigs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=arr.rinzler.cloud,resources=sonarrconfigs,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=arr.rinzler.cloud,resources=sonarrconfigs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=arr.rinzler.cloud,resources=lidarrconfigs,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=arr.rinzler.cloud,resources=lidarrconfigs/status,verbs=get;update;patch

// Reconcile handles coordination between ProwlarrConfig and *arr configs
func (r *ProwlarrCoordinatorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	log.Info("Reconciling Prowlarr coordination", "prowlarrConfig", req.NamespacedName)

	// Get the ProwlarrConfig that triggered this reconcile
	prowlarrConfig := &arrv1alpha1.ProwlarrConfig{}
	if err := r.Get(ctx, req.NamespacedName, prowlarrConfig); err != nil {
		if client.IgnoreNotFound(err) != nil {
			return ctrl.Result{}, err
		}
		// ProwlarrConfig deleted - handle cleanup in the watcher
		return ctrl.Result{}, nil
	}

	// Skip if ProwlarrConfig is being deleted
	if !prowlarrConfig.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, nil
	}

	// Skip if not connected
	if !prowlarrConfig.Status.Connected {
		log.Info("ProwlarrConfig not connected, skipping coordination")
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	// Get Prowlarr connection info
	prowlarrSecrets, err := r.Helper.ResolveConnectionSecrets(ctx, prowlarrConfig.Namespace, &prowlarrConfig.Spec.Connection)
	if err != nil {
		log.Error(err, "Failed to resolve Prowlarr secrets")
		return ctrl.Result{RequeueAfter: ErrorRequeueInterval}, err
	}

	prowlarrConn := prowlarr.ProwlarrConnection{
		URL:    prowlarrConfig.Spec.Connection.URL,
		APIKey: prowlarrSecrets["apiKey"],
	}

	// Build map of apps defined in Push Model (spec.applications[])
	pushModelApps := make(map[string]bool)
	for _, app := range prowlarrConfig.Spec.Applications {
		// Key: namespace/appType/appName (e.g., "arr-stack/radarr/radarr")
		key := fmt.Sprintf("%s/%s", app.Type, app.Name)
		pushModelApps[key] = true
	}

	// Find all *arr configs that reference this ProwlarrConfig
	var errs []error

	// Process RadarrConfigs
	radarrList := &arrv1alpha1.RadarrConfigList{}
	if err := r.List(ctx, radarrList, client.InNamespace(req.Namespace)); err != nil {
		log.Error(err, "Failed to list RadarrConfigs")
	} else {
		for i := range radarrList.Items {
			config := &radarrList.Items[i]
			if err := r.processAppConfig(ctx, prowlarrConfig, prowlarrConn, pushModelApps,
				config.Name, config.Namespace, irv1.AppTypeRadarr,
				config.Spec.Indexers, config.Spec.Connection.URL,
				&config.Status.ProwlarrRegistration); err != nil {
				errs = append(errs, err)
			}
			// Always update status (even on error, the registration field may have been set with error details)
			if err := r.Status().Update(ctx, config); err != nil {
				log.Error(err, "Failed to update RadarrConfig status", "name", config.Name)
			}
		}
	}

	// Process SonarrConfigs
	sonarrList := &arrv1alpha1.SonarrConfigList{}
	if err := r.List(ctx, sonarrList, client.InNamespace(req.Namespace)); err != nil {
		log.Error(err, "Failed to list SonarrConfigs")
	} else {
		for i := range sonarrList.Items {
			config := &sonarrList.Items[i]
			if err := r.processAppConfig(ctx, prowlarrConfig, prowlarrConn, pushModelApps,
				config.Name, config.Namespace, irv1.AppTypeSonarr,
				config.Spec.Indexers, config.Spec.Connection.URL,
				&config.Status.ProwlarrRegistration); err != nil {
				errs = append(errs, err)
			}
			// Always update status (even on error, the registration field may have been set with error details)
			if err := r.Status().Update(ctx, config); err != nil {
				log.Error(err, "Failed to update SonarrConfig status", "name", config.Name)
			}
		}
	}

	// Process LidarrConfigs
	lidarrList := &arrv1alpha1.LidarrConfigList{}
	if err := r.List(ctx, lidarrList, client.InNamespace(req.Namespace)); err != nil {
		log.Error(err, "Failed to list LidarrConfigs")
	} else {
		for i := range lidarrList.Items {
			config := &lidarrList.Items[i]
			if err := r.processAppConfig(ctx, prowlarrConfig, prowlarrConn, pushModelApps,
				config.Name, config.Namespace, irv1.AppTypeLidarr,
				config.Spec.Indexers, config.Spec.Connection.URL,
				&config.Status.ProwlarrRegistration); err != nil {
				errs = append(errs, err)
			}
			// Always update status (even on error, the registration field may have been set with error details)
			if err := r.Status().Update(ctx, config); err != nil {
				log.Error(err, "Failed to update LidarrConfig status", "name", config.Name)
			}
		}
	}

	if len(errs) > 0 {
		return ctrl.Result{RequeueAfter: ErrorRequeueInterval}, fmt.Errorf("encountered %d errors during coordination", len(errs))
	}

	return ctrl.Result{RequeueAfter: DefaultRequeueInterval}, nil
}

// processAppConfig handles registration for a single app config
func (r *ProwlarrCoordinatorReconciler) processAppConfig(
	ctx context.Context,
	prowlarrConfig *arrv1alpha1.ProwlarrConfig,
	prowlarrConn prowlarr.ProwlarrConnection,
	pushModelApps map[string]bool,
	configName, namespace, appType string,
	indexers *arrv1alpha1.IndexersSpec,
	appURL string,
	registration **arrv1alpha1.ProwlarrRegistration,
) error {
	log := logf.FromContext(ctx)

	// Check if this app has a prowlarrRef pointing to the current ProwlarrConfig
	if indexers == nil || indexers.ProwlarrRef == nil {
		// No prowlarrRef - if previously registered, mark as unregistered
		if *registration != nil && (*registration).Registered && (*registration).ProwlarrName == prowlarrConfig.Name {
			*registration = &arrv1alpha1.ProwlarrRegistration{
				Registered:   false,
				ProwlarrName: "",
				Message:      "ProwlarrRef removed from config",
			}
		}
		return nil
	}

	prowlarrRef := indexers.ProwlarrRef
	if prowlarrRef.Name != prowlarrConfig.Name {
		// References a different ProwlarrConfig
		return nil
	}

	// Check for conflict: app defined in both Push and Pull model
	pushKey := fmt.Sprintf("%s/%s", appType, configName)
	if pushModelApps[pushKey] {
		log.Info("Conflict detected: app defined in both Push and Pull model",
			"app", configName, "type", appType, "prowlarr", prowlarrConfig.Name)
		*registration = &arrv1alpha1.ProwlarrRegistration{
			Registered:   false,
			ProwlarrName: prowlarrConfig.Name,
			Message:      "Conflict: app is defined in both ProwlarrConfig.applications[] (Push) and has prowlarrRef (Pull). Remove from one.",
		}
		return fmt.Errorf("conflict: %s/%s defined in both Push and Pull model", appType, configName)
	}

	// Check if auto-register is enabled (default true)
	autoRegister := true
	if prowlarrRef.AutoRegister != nil {
		autoRegister = *prowlarrRef.AutoRegister
	}

	if !autoRegister {
		log.V(1).Info("Auto-registration disabled", "app", configName, "type", appType)
		*registration = &arrv1alpha1.ProwlarrRegistration{
			Registered:   false,
			ProwlarrName: prowlarrConfig.Name,
			Message:      "Auto-registration disabled (autoRegister: false)",
		}
		return nil
	}

	// Resolve app's API key
	// Look for the secret in the same namespace
	appSecrets := make(map[string]string)
	// We need to get the app's connection spec - for now, we'll get it from the parent
	// This is a simplified approach; in production you'd want to look up the actual secret

	appAPIKey := ""
	// Try to find the API key secret for this app type
	secretName := fmt.Sprintf("%s-api-key", appType)
	if apiKey, err := r.Helper.ResolveSecretValue(ctx, namespace, secretName, "apiKey"); err == nil {
		appAPIKey = apiKey
	} else {
		log.V(1).Info("Could not resolve app API key, will use connection from config", "app", configName, "error", err)
		// Try the config-specific secret name pattern
		secretName = fmt.Sprintf("%s-api-key", configName)
		if apiKey, err := r.Helper.ResolveSecretValue(ctx, namespace, secretName, "apiKey"); err == nil {
			appAPIKey = apiKey
		}
	}

	if appAPIKey == "" {
		*registration = &arrv1alpha1.ProwlarrRegistration{
			Registered:   false,
			ProwlarrName: prowlarrConfig.Name,
			Message:      fmt.Sprintf("Could not resolve API key for %s", configName),
		}
		return fmt.Errorf("could not resolve API key for %s/%s", appType, configName)
	}

	// Build app registration
	appName := fmt.Sprintf("nebularr-pull-%s-%s", configName, appType)
	appReg := prowlarr.AppRegistration{
		Name:        appName,
		Type:        appType,
		URL:         appURL,
		APIKey:      appAPIKey,
		ProwlarrURL: prowlarrConfig.Spec.Connection.URL,
		SyncLevel:   irv1.SyncLevelFullSync,
	}

	// Apply category filter from prowlarrRef.include/exclude
	// For now, use defaults; category filtering would need indexer definitions

	// Register with Prowlarr
	regService := prowlarr.NewRegistrationService()
	log.Info("Registering app with Prowlarr (Pull Model)",
		"app", configName, "type", appType, "prowlarr", prowlarrConfig.Name, "appName", appName)

	if err := regService.Register(ctx, prowlarrConn, appReg); err != nil {
		log.Error(err, "Failed to register app with Prowlarr", "app", configName)
		*registration = &arrv1alpha1.ProwlarrRegistration{
			Registered:   false,
			ProwlarrName: prowlarrConfig.Name,
			Message:      fmt.Sprintf("Registration failed: %v", err),
		}
		return err
	}

	// Success - update status
	now := metav1.Now()
	*registration = &arrv1alpha1.ProwlarrRegistration{
		Registered:   true,
		ProwlarrName: prowlarrConfig.Name,
		LastSync:     &now,
		Message:      fmt.Sprintf("Registered as %s", appName),
	}

	log.Info("Successfully registered app with Prowlarr (Pull Model)",
		"app", configName, "type", appType, "prowlarr", prowlarrConfig.Name)

	_ = appSecrets // silence unused variable
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ProwlarrCoordinatorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if r.Helper == nil {
		r.Helper = NewReconcileHelper(r.Client)
	}

	// Map function to trigger reconcile of ProwlarrConfig when an app config changes
	mapAppConfigToProwlarr := func(ctx context.Context, obj client.Object) []reconcile.Request {
		var requests []reconcile.Request

		// Get prowlarrRef from the object
		var prowlarrRefName string

		switch config := obj.(type) {
		case *arrv1alpha1.RadarrConfig:
			if config.Spec.Indexers != nil && config.Spec.Indexers.ProwlarrRef != nil {
				prowlarrRefName = config.Spec.Indexers.ProwlarrRef.Name
			}
		case *arrv1alpha1.SonarrConfig:
			if config.Spec.Indexers != nil && config.Spec.Indexers.ProwlarrRef != nil {
				prowlarrRefName = config.Spec.Indexers.ProwlarrRef.Name
			}
		case *arrv1alpha1.LidarrConfig:
			if config.Spec.Indexers != nil && config.Spec.Indexers.ProwlarrRef != nil {
				prowlarrRefName = config.Spec.Indexers.ProwlarrRef.Name
			}
		}

		if prowlarrRefName != "" {
			requests = append(requests, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      prowlarrRefName,
					Namespace: obj.GetNamespace(),
				},
			})
		}

		return requests
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&arrv1alpha1.ProwlarrConfig{}).
		Watches(
			&arrv1alpha1.RadarrConfig{},
			handler.EnqueueRequestsFromMapFunc(mapAppConfigToProwlarr),
		).
		Watches(
			&arrv1alpha1.SonarrConfig{},
			handler.EnqueueRequestsFromMapFunc(mapAppConfigToProwlarr),
		).
		Watches(
			&arrv1alpha1.LidarrConfig{},
			handler.EnqueueRequestsFromMapFunc(mapAppConfigToProwlarr),
		).
		Named("prowlarrcoordinator").
		Complete(r)
}
