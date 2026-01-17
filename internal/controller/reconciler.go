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

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	arrv1alpha1 "github.com/poiley/nebularr-operator/api/v1alpha1"
	"github.com/poiley/nebularr-operator/internal/adapters"
	irv1 "github.com/poiley/nebularr-operator/internal/ir/v1"
	"github.com/poiley/nebularr-operator/internal/metrics"
	"github.com/poiley/nebularr-operator/internal/prowlarr"
)

const (
	// Condition types
	ConditionTypeReady     = "Ready"
	ConditionTypeConnected = "Connected"
	ConditionTypeSynced    = "Synced"

	// Default requeue intervals
	DefaultRequeueInterval = 5 * time.Minute
	ErrorRequeueInterval   = 30 * time.Second
)

// ConfigStatus is an interface for updating status on *arr config resources
type ConfigStatus interface {
	GetConditions() []metav1.Condition
	SetConditions(conditions []metav1.Condition)
	SetConnected(connected bool)
	SetServiceVersion(version string)
	SetLastReconcile(t *metav1.Time)
	SetLastAppliedHash(hash string)
}

// ReconcileHelper provides shared reconciliation logic for all *arr controllers
type ReconcileHelper struct {
	Client client.Client
}

// NewReconcileHelper creates a new ReconcileHelper
func NewReconcileHelper(c client.Client) *ReconcileHelper {
	return &ReconcileHelper{Client: c}
}

// ReconcileConfig performs the common reconciliation flow for any *arr config
func (h *ReconcileHelper) ReconcileConfig(
	ctx context.Context,
	appType string,
	connIR *irv1.ConnectionIR,
	desiredIR *irv1.IR,
	status ConfigStatus,
	generation int64,
) (*adapters.ApplyResult, error) {
	log := logf.FromContext(ctx)
	startTime := time.Now()

	// Get the adapter
	adapter, ok := adapters.Get(appType)
	if !ok {
		err := fmt.Errorf("%s adapter not registered", appType)
		h.SetCondition(status, generation, ConditionTypeReady, metav1.ConditionFalse, "AdapterNotFound", err.Error())
		metrics.RecordSyncFailure(appType, "adapter_not_found", time.Since(startTime).Seconds())
		return nil, err
	}

	// Test connectivity and get service info
	serviceInfo, err := adapter.Connect(ctx, connIR)
	if err != nil {
		log.Error(err, "Failed to connect to service", "app", appType)
		status.SetConnected(false)
		h.SetCondition(status, generation, ConditionTypeConnected, metav1.ConditionFalse, "ConnectionFailed", err.Error())
		h.SetCondition(status, generation, ConditionTypeReady, metav1.ConditionFalse, "ConnectionFailed", fmt.Sprintf("Cannot connect to %s", appType))
		metrics.RecordConnectionStatus(appType, connIR.URL, false)
		metrics.RecordSyncFailure(appType, "connection_failed", time.Since(startTime).Seconds())
		return nil, err
	}

	status.SetConnected(true)
	status.SetServiceVersion(serviceInfo.Version)
	h.SetCondition(status, generation, ConditionTypeConnected, metav1.ConditionTrue, "Connected", fmt.Sprintf("Connected to %s %s", appType, serviceInfo.Version))

	// Record connection success and service version
	metrics.RecordConnectionStatus(appType, connIR.URL, true)
	metrics.RecordServiceVersion(appType, connIR.URL, serviceInfo.Version)

	// Discover capabilities
	caps, err := adapter.Discover(ctx, connIR)
	if err != nil {
		log.Error(err, "Failed to discover capabilities", "app", appType)
		h.SetCondition(status, generation, ConditionTypeReady, metav1.ConditionFalse, "DiscoveryFailed", err.Error())
		return nil, err
	}

	// Get current state
	currentIR, err := adapter.CurrentState(ctx, connIR)
	if err != nil {
		log.Error(err, "Failed to get current state", "app", appType)
		h.SetCondition(status, generation, ConditionTypeReady, metav1.ConditionFalse, "StateFetchFailed", err.Error())
		return nil, err
	}

	// Compute diff
	changes, err := adapter.Diff(currentIR, desiredIR, caps)
	if err != nil {
		log.Error(err, "Failed to compute diff", "app", appType)
		h.SetCondition(status, generation, ConditionTypeReady, metav1.ConditionFalse, "DiffFailed", err.Error())
		return nil, err
	}

	// Apply changes if needed
	var result *adapters.ApplyResult
	if !changes.IsEmpty() {
		log.Info("Applying changes", "creates", len(changes.Creates), "updates", len(changes.Updates), "deletes", len(changes.Deletes))

		// Record drift detection
		for _, change := range changes.Creates {
			metrics.RecordConfigDrift(appType, change.ResourceType)
		}
		for _, change := range changes.Updates {
			metrics.RecordConfigDrift(appType, change.ResourceType)
		}

		result, err = adapter.Apply(ctx, connIR, changes)
		if err != nil {
			log.Error(err, "Failed to apply changes")
			h.SetCondition(status, generation, ConditionTypeSynced, metav1.ConditionFalse, "ApplyFailed", err.Error())
			h.SetCondition(status, generation, ConditionTypeReady, metav1.ConditionFalse, "ApplyFailed",
				fmt.Sprintf("Applied %d/%d changes", result.Applied, changes.TotalChanges()))
			metrics.RecordSyncFailure(appType, "apply_failed", time.Since(startTime).Seconds())
			return result, err
		}

		// Record applied changes
		for _, change := range changes.Creates {
			metrics.RecordApplyChange(appType, "create", change.ResourceType)
		}
		for _, change := range changes.Updates {
			metrics.RecordApplyChange(appType, "update", change.ResourceType)
		}
		for _, change := range changes.Deletes {
			metrics.RecordApplyChange(appType, "delete", change.ResourceType)
		}

		if !result.Success() {
			log.Info("Some changes failed to apply", "applied", result.Applied, "failed", result.Failed)
			// Log each individual error for debugging
			for _, applyErr := range result.Errors {
				log.Error(applyErr.Error, "Failed to apply change",
					"resourceType", applyErr.Change.ResourceType,
					"resourceName", applyErr.Change.Name)
			}
			h.SetCondition(status, generation, ConditionTypeSynced, metav1.ConditionFalse, "PartiallyApplied",
				fmt.Sprintf("Applied %d changes, %d failed", result.Applied, result.Failed))
		} else {
			log.Info("All changes applied successfully", "applied", result.Applied)
			h.SetCondition(status, generation, ConditionTypeSynced, metav1.ConditionTrue, "Synced",
				fmt.Sprintf("Applied %d changes", result.Applied))
		}
	} else {
		log.Info("No changes to apply, state is in sync")
		h.SetCondition(status, generation, ConditionTypeSynced, metav1.ConditionTrue, "InSync", "Configuration is in sync")
		result = &adapters.ApplyResult{Applied: 0}
	}

	// Update timestamps and hash
	now := metav1.Now()
	status.SetLastReconcile(&now)
	status.SetLastAppliedHash(desiredIR.SourceHash)
	h.SetCondition(status, generation, ConditionTypeReady, metav1.ConditionTrue, "Ready", "Configuration reconciled successfully")

	// Record successful sync
	metrics.RecordSyncSuccess(appType, time.Since(startTime).Seconds())

	return result, nil
}

// CleanupManagedResources removes all managed resources from the service
func (h *ReconcileHelper) CleanupManagedResources(ctx context.Context, appType string, connIR *irv1.ConnectionIR) error {
	log := logf.FromContext(ctx)

	adapter, ok := adapters.Get(appType)
	if !ok {
		return fmt.Errorf("%s adapter not registered", appType)
	}

	// Get current managed state
	currentIR, err := adapter.CurrentState(ctx, connIR)
	if err != nil {
		log.Error(err, "Failed to get current state for cleanup")
		return err
	}

	if currentIR == nil {
		return nil
	}

	// Create a changeset to delete all managed resources
	caps, _ := adapter.Discover(ctx, connIR)
	emptyIR := &irv1.IR{App: appType}
	changes, err := adapter.Diff(currentIR, emptyIR, caps)
	if err != nil {
		log.Error(err, "Failed to compute deletion diff")
		return err
	}

	if !changes.IsEmpty() {
		result, err := adapter.Apply(ctx, connIR, changes)
		if err != nil {
			log.Error(err, "Failed to cleanup managed resources")
			return err
		}
		log.Info("Cleaned up managed resources", "deleted", result.Applied)
	}

	return nil
}

// SetCondition sets a condition on the config status
func (h *ReconcileHelper) SetCondition(status ConfigStatus, generation int64, condType string, condStatus metav1.ConditionStatus, reason, message string) {
	conditions := status.GetConditions()
	meta.SetStatusCondition(&conditions, metav1.Condition{
		Type:               condType,
		Status:             condStatus,
		ObservedGeneration: generation,
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            message,
	})
	status.SetConditions(conditions)
}

// ResolveSecretValue retrieves a value from a Kubernetes Secret
func (h *ReconcileHelper) ResolveSecretValue(ctx context.Context, namespace, name, key string) (string, error) {
	secret := &corev1.Secret{}
	if err := h.Client.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, secret); err != nil {
		return "", fmt.Errorf("failed to get secret %s/%s: %w", namespace, name, err)
	}

	value, ok := secret.Data[key]
	if !ok {
		return "", fmt.Errorf("key %q not found in secret %s/%s", key, namespace, name)
	}

	return string(value), nil
}

// ResolveConnectionSecrets resolves secrets for a ConnectionSpec
func (h *ReconcileHelper) ResolveConnectionSecrets(ctx context.Context, namespace string, conn *arrv1alpha1.ConnectionSpec) (map[string]string, error) {
	resolved := make(map[string]string)

	if conn.APIKeySecretRef != nil {
		key := conn.APIKeySecretRef.Key
		if key == "" {
			key = "apiKey"
		}
		apiKey, err := h.ResolveSecretValue(ctx, namespace, conn.APIKeySecretRef.Name, key)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve API key secret: %w", err)
		}
		resolved["apiKey"] = apiKey
	}

	return resolved, nil
}

// ResolveDownloadClientSecrets resolves credentials for download clients
func (h *ReconcileHelper) ResolveDownloadClientSecrets(ctx context.Context, namespace string, clients []arrv1alpha1.DownloadClientSpec, resolved map[string]string) error {
	for _, dc := range clients {
		if dc.CredentialsSecretRef != nil {
			secretName := dc.CredentialsSecretRef.Name
			usernameKey := dc.CredentialsSecretRef.UsernameKey
			if usernameKey == "" {
				usernameKey = "username"
			}
			passwordKey := dc.CredentialsSecretRef.PasswordKey
			if passwordKey == "" {
				passwordKey = "password"
			}

			username, err := h.ResolveSecretValue(ctx, namespace, secretName, usernameKey)
			if err != nil {
				return fmt.Errorf("failed to resolve download client credentials: %w", err)
			}
			resolved[secretName+"/"+usernameKey] = username

			password, err := h.ResolveSecretValue(ctx, namespace, secretName, passwordKey)
			if err != nil {
				return fmt.Errorf("failed to resolve download client credentials: %w", err)
			}
			resolved[secretName+"/"+passwordKey] = password
		}
	}
	return nil
}

// ResolveIndexerSecrets resolves API keys for direct indexers
func (h *ReconcileHelper) ResolveIndexerSecrets(ctx context.Context, namespace string, indexers *arrv1alpha1.IndexersSpec, resolved map[string]string) error {
	if indexers == nil {
		return nil
	}

	for _, idx := range indexers.Direct {
		if idx.APIKeySecretRef != nil {
			keyName := idx.APIKeySecretRef.Key
			if keyName == "" {
				keyName = "apiKey"
			}
			apiKey, err := h.ResolveSecretValue(ctx, namespace, idx.APIKeySecretRef.Name, keyName)
			if err != nil {
				return fmt.Errorf("failed to resolve indexer API key: %w", err)
			}
			resolved[idx.APIKeySecretRef.Name+"/"+keyName] = apiKey
		}
	}
	return nil
}

// ResolveProwlarrIndexerSecrets resolves API keys for Prowlarr indexers
func (h *ReconcileHelper) ResolveProwlarrIndexerSecrets(ctx context.Context, namespace string, indexers []arrv1alpha1.ProwlarrIndexer, resolved map[string]string) error {
	for _, idx := range indexers {
		if idx.APIKeySecretRef != nil {
			keyName := idx.APIKeySecretRef.Key
			if keyName == "" {
				keyName = "apiKey"
			}
			apiKey, err := h.ResolveSecretValue(ctx, namespace, idx.APIKeySecretRef.Name, keyName)
			if err != nil {
				return fmt.Errorf("failed to resolve Prowlarr indexer API key: %w", err)
			}
			resolved[idx.APIKeySecretRef.Name+"/"+keyName] = apiKey
		}
	}
	return nil
}

// ResolveProxySecrets resolves credentials for indexer proxies
func (h *ReconcileHelper) ResolveProxySecrets(ctx context.Context, namespace string, proxies []arrv1alpha1.IndexerProxy, resolved map[string]string) error {
	for _, proxy := range proxies {
		if proxy.CredentialsSecretRef != nil {
			secretName := proxy.CredentialsSecretRef.Name
			usernameKey := proxy.CredentialsSecretRef.UsernameKey
			if usernameKey == "" {
				usernameKey = "username"
			}
			passwordKey := proxy.CredentialsSecretRef.PasswordKey
			if passwordKey == "" {
				passwordKey = "password"
			}

			username, err := h.ResolveSecretValue(ctx, namespace, secretName, usernameKey)
			if err != nil {
				return fmt.Errorf("failed to resolve proxy credentials: %w", err)
			}
			resolved[secretName+"/"+usernameKey] = username

			password, err := h.ResolveSecretValue(ctx, namespace, secretName, passwordKey)
			if err != nil {
				return fmt.Errorf("failed to resolve proxy credentials: %w", err)
			}
			resolved[secretName+"/"+passwordKey] = password
		}
	}
	return nil
}

// ResolveApplicationSecrets resolves API keys for Prowlarr applications
func (h *ReconcileHelper) ResolveApplicationSecrets(ctx context.Context, namespace string, apps []arrv1alpha1.ProwlarrApplication, resolved map[string]string) error {
	for _, app := range apps {
		if app.APIKeySecretRef != nil {
			keyName := app.APIKeySecretRef.Key
			if keyName == "" {
				keyName = "apiKey"
			}
			apiKey, err := h.ResolveSecretValue(ctx, namespace, app.APIKeySecretRef.Name, keyName)
			if err != nil {
				return fmt.Errorf("failed to resolve application API key: %w", err)
			}
			resolved[app.APIKeySecretRef.Name+"/"+keyName] = apiKey
		}
	}
	return nil
}

// ResolveImportListSecrets resolves secrets for import lists
// Each import list may have a SettingsSecretRef that contains sensitive settings.
// All keys from the referenced secret are loaded with format "secretName/key".
func (h *ReconcileHelper) ResolveImportListSecrets(ctx context.Context, namespace string, lists []arrv1alpha1.ImportListSpec, resolved map[string]string) error {
	for _, list := range lists {
		if list.SettingsSecretRef != nil {
			secret := &corev1.Secret{}
			if err := h.Client.Get(ctx, client.ObjectKey{Namespace: namespace, Name: list.SettingsSecretRef.Name}, secret); err != nil {
				return fmt.Errorf("failed to get import list secret %s: %w", list.SettingsSecretRef.Name, err)
			}
			// Add all keys from the secret with prefix "secretName/"
			for key, value := range secret.Data {
				resolved[list.SettingsSecretRef.Name+"/"+key] = string(value)
			}
		}
	}
	return nil
}

// ResolveAuthenticationSecrets resolves password secrets for authentication
func (h *ReconcileHelper) ResolveAuthenticationSecrets(ctx context.Context, namespace string, auth *arrv1alpha1.AuthenticationSpec, resolved map[string]string) error {
	if auth == nil || auth.PasswordSecretRef == nil {
		return nil
	}

	keyName := auth.PasswordSecretRef.Key
	if keyName == "" {
		keyName = "password"
	}

	password, err := h.ResolveSecretValue(ctx, namespace, auth.PasswordSecretRef.Name, keyName)
	if err != nil {
		return fmt.Errorf("failed to resolve authentication password: %w", err)
	}
	resolved[auth.PasswordSecretRef.Name+"/"+keyName] = password
	return nil
}

// ProwlarrAutoRegistration holds info for auto-registering with Prowlarr
type ProwlarrAutoRegistration struct {
	// ProwlarrRef is the reference to the ProwlarrConfig
	ProwlarrRef *arrv1alpha1.ProwlarrRef

	// AppType is the app type: radarr, sonarr, lidarr
	AppType string

	// AppName is the unique name for this app in Prowlarr (e.g., "nebularr-radarr-myconfig")
	AppName string

	// AppURL is the app's URL
	AppURL string

	// AppAPIKey is the app's API key
	AppAPIKey string
}

// HandleProwlarrRegistration handles auto-registration with Prowlarr if prowlarrRef is set
func (h *ReconcileHelper) HandleProwlarrRegistration(ctx context.Context, namespace string, reg ProwlarrAutoRegistration) error {
	log := logf.FromContext(ctx)

	if reg.ProwlarrRef == nil {
		return nil
	}

	// Check if auto-register is enabled (default true)
	autoRegister := true
	if reg.ProwlarrRef.AutoRegister != nil {
		autoRegister = *reg.ProwlarrRef.AutoRegister
	}

	if !autoRegister {
		log.V(1).Info("Auto-registration disabled for prowlarrRef")
		return nil
	}

	// Look up the ProwlarrConfig
	prowlarrConfig := &arrv1alpha1.ProwlarrConfig{}
	if err := h.Client.Get(ctx, types.NamespacedName{
		Name:      reg.ProwlarrRef.Name,
		Namespace: namespace,
	}, prowlarrConfig); err != nil {
		return fmt.Errorf("failed to get ProwlarrConfig %s: %w", reg.ProwlarrRef.Name, err)
	}

	// Resolve Prowlarr's API key
	prowlarrSecrets, err := h.ResolveConnectionSecrets(ctx, namespace, &prowlarrConfig.Spec.Connection)
	if err != nil {
		return fmt.Errorf("failed to resolve Prowlarr secrets: %w", err)
	}

	prowlarrConn := prowlarr.ProwlarrConnection{
		URL:    prowlarrConfig.Spec.Connection.URL,
		APIKey: prowlarrSecrets["apiKey"],
	}

	// Register this app with Prowlarr
	regService := prowlarr.NewRegistrationService()
	appReg := prowlarr.AppRegistration{
		Name:      reg.AppName,
		Type:      reg.AppType,
		URL:       reg.AppURL,
		APIKey:    reg.AppAPIKey,
		SyncLevel: irv1.SyncLevelFullSync,
	}

	log.Info("Auto-registering with Prowlarr", "prowlarr", reg.ProwlarrRef.Name, "app", reg.AppName)
	if err := regService.Register(ctx, prowlarrConn, appReg); err != nil {
		return fmt.Errorf("failed to register with Prowlarr: %w", err)
	}

	log.Info("Successfully registered with Prowlarr", "prowlarr", reg.ProwlarrRef.Name, "app", reg.AppName)
	return nil
}

// ApplyDirectConfig applies configuration directly using ApplyDirect if the adapter supports it.
// This handles resources like import lists, media management, and authentication that use
// a different sync pattern than the diff-based approach.
func (h *ReconcileHelper) ApplyDirectConfig(
	ctx context.Context,
	appType string,
	connIR *irv1.ConnectionIR,
	desiredIR *irv1.IR,
	status ConfigStatus,
	generation int64,
) (*adapters.ApplyResult, error) {
	log := logf.FromContext(ctx)

	// Get the adapter
	adapter, ok := adapters.Get(appType)
	if !ok {
		return nil, fmt.Errorf("%s adapter not registered", appType)
	}

	// Check if adapter supports DirectApplier interface
	directApplier, ok := adapter.(adapters.DirectApplier)
	if !ok {
		log.V(1).Info("Adapter does not support DirectApplier, skipping", "app", appType)
		return &adapters.ApplyResult{}, nil
	}

	// Check if there's anything to apply directly
	hasDirectApplyWork := len(desiredIR.ImportLists) > 0 ||
		desiredIR.MediaManagement != nil ||
		desiredIR.Authentication != nil

	if !hasDirectApplyWork {
		log.V(1).Info("No direct apply work to do", "app", appType)
		return &adapters.ApplyResult{}, nil
	}

	log.Info("Applying direct configuration",
		"app", appType,
		"importLists", len(desiredIR.ImportLists),
		"hasMediaManagement", desiredIR.MediaManagement != nil,
		"hasAuthentication", desiredIR.Authentication != nil)

	result, err := directApplier.ApplyDirect(ctx, connIR, desiredIR)
	if err != nil {
		log.Error(err, "Failed to apply direct configuration", "app", appType)
		return result, err
	}

	if result != nil && !result.Success() {
		log.Info("Some direct configuration changes failed",
			"app", appType,
			"applied", result.Applied,
			"failed", result.Failed,
			"skipped", result.Skipped)
	} else if result != nil && result.Applied > 0 {
		log.Info("Direct configuration applied successfully",
			"app", appType,
			"applied", result.Applied)
	}

	return result, nil
}

// HandleProwlarrUnregistration removes this app from Prowlarr
func (h *ReconcileHelper) HandleProwlarrUnregistration(ctx context.Context, namespace string, prowlarrRefName, appName string) error {
	log := logf.FromContext(ctx)

	if prowlarrRefName == "" {
		return nil
	}

	// Look up the ProwlarrConfig
	prowlarrConfig := &arrv1alpha1.ProwlarrConfig{}
	if err := h.Client.Get(ctx, types.NamespacedName{
		Name:      prowlarrRefName,
		Namespace: namespace,
	}, prowlarrConfig); err != nil {
		// ProwlarrConfig might be deleted already, log and continue
		log.V(1).Info("ProwlarrConfig not found for unregistration, skipping", "name", prowlarrRefName)
		return nil
	}

	// Resolve Prowlarr's API key
	prowlarrSecrets, err := h.ResolveConnectionSecrets(ctx, namespace, &prowlarrConfig.Spec.Connection)
	if err != nil {
		log.Error(err, "Failed to resolve Prowlarr secrets for unregistration")
		return nil // Don't block deletion
	}

	prowlarrConn := prowlarr.ProwlarrConnection{
		URL:    prowlarrConfig.Spec.Connection.URL,
		APIKey: prowlarrSecrets["apiKey"],
	}

	// Unregister this app from Prowlarr
	regService := prowlarr.NewRegistrationService()
	log.Info("Unregistering from Prowlarr", "prowlarr", prowlarrRefName, "app", appName)
	if err := regService.Unregister(ctx, prowlarrConn, appName); err != nil {
		log.Error(err, "Failed to unregister from Prowlarr")
		return nil // Don't block deletion
	}

	log.Info("Successfully unregistered from Prowlarr", "prowlarr", prowlarrRefName, "app", appName)
	return nil
}

// HealthStatusSetter is an interface for setting health status on config resources
type HealthStatusSetter interface {
	SetHealthStatus(health *arrv1alpha1.HealthStatus)
	GetHealthStatus() *arrv1alpha1.HealthStatus
}

// CheckAndReportHealth checks the health of the app and reports events for issues.
// It returns the health status that should be stored in the CRD status.
// The recorder parameter is the standard Kubernetes EventRecorder from controller-runtime.
func (h *ReconcileHelper) CheckAndReportHealth(
	ctx context.Context,
	appType string,
	connIR *irv1.ConnectionIR,
	obj runtime.Object,
	recorder record.EventRecorder,
) *arrv1alpha1.HealthStatus {
	log := logf.FromContext(ctx)

	// Get the adapter
	adapter, ok := adapters.Get(appType)
	if !ok {
		log.V(1).Info("Adapter not found for health check", "app", appType)
		return nil
	}

	// Check if adapter supports HealthChecker interface
	healthChecker, ok := adapter.(adapters.HealthChecker)
	if !ok {
		log.V(1).Info("Adapter does not support HealthChecker", "app", appType)
		return nil
	}

	// Fetch health from the app
	healthIR, err := healthChecker.GetHealth(ctx, connIR)
	if err != nil {
		log.Error(err, "Failed to fetch health from app", "app", appType)
		return nil
	}

	// Convert IR health to API status
	now := metav1.Now()
	healthStatus := &arrv1alpha1.HealthStatus{
		Healthy:    healthIR.Healthy,
		IssueCount: len(healthIR.Issues),
		LastCheck:  &now,
		Issues:     make([]arrv1alpha1.HealthIssueStatus, 0, len(healthIR.Issues)),
	}

	for _, issue := range healthIR.Issues {
		healthStatus.Issues = append(healthStatus.Issues, arrv1alpha1.HealthIssueStatus{
			Source:  issue.Source,
			Type:    string(issue.Type),
			Message: issue.Message,
			WikiURL: issue.WikiURL,
		})

		// Count by type
		switch issue.Type {
		case irv1.HealthIssueTypeError:
			healthStatus.ErrorCount++
		case irv1.HealthIssueTypeWarning:
			healthStatus.WarningCount++
		}
	}

	// Emit K8s events for health issues
	if recorder != nil && obj != nil {
		for _, issue := range healthIR.Issues {
			eventType := corev1.EventTypeWarning
			reason := "HealthWarning"

			if issue.Type == irv1.HealthIssueTypeError {
				reason = "HealthError"
			} else if issue.Type == irv1.HealthIssueTypeNotice {
				eventType = corev1.EventTypeNormal
				reason = "HealthNotice"
			}

			// Only emit events for errors and warnings (not notices)
			if issue.Type != irv1.HealthIssueTypeNotice {
				recorder.Event(obj, eventType, reason, fmt.Sprintf("[%s] %s", issue.Source, issue.Message))
			}
		}
	}

	log.V(1).Info("Health check completed",
		"app", appType,
		"healthy", healthIR.Healthy,
		"issues", len(healthIR.Issues),
		"errors", healthStatus.ErrorCount,
		"warnings", healthStatus.WarningCount)

	return healthStatus
}
