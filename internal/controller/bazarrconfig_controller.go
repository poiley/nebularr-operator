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

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	arrv1alpha1 "github.com/poiley/nebularr-operator/api/v1alpha1"
	"github.com/poiley/nebularr-operator/internal/adapters/bazarr"
)

const bazarrFinalizer = "bazarrconfig.arr.rinzler.cloud/finalizer"

// BazarrConfigReconciler reconciles a BazarrConfig object
type BazarrConfigReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Helper *ReconcileHelper
}

// +kubebuilder:rbac:groups=arr.rinzler.cloud,resources=bazarrconfigs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=arr.rinzler.cloud,resources=bazarrconfigs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=arr.rinzler.cloud,resources=bazarrconfigs/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop
func (r *BazarrConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	// Fetch the BazarrConfig
	config := &arrv1alpha1.BazarrConfig{}
	if err := r.Get(ctx, req.NamespacedName, config); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("BazarrConfig resource not found, ignoring")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get BazarrConfig")
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
	if !controllerutil.ContainsFinalizer(config, bazarrFinalizer) {
		controllerutil.AddFinalizer(config, bazarrFinalizer)
		if err := r.Update(ctx, config); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// Reconcile the configuration
	return r.reconcileNormal(ctx, config)
}

// reconcileNormal handles the normal reconciliation flow
func (r *BazarrConfigReconciler) reconcileNormal(ctx context.Context, config *arrv1alpha1.BazarrConfig) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	log.Info("Reconciling BazarrConfig", "name", config.Name)

	statusWrapper := &BazarrStatusWrapper{Status: &config.Status}

	// Resolve Sonarr API key
	sonarrAPIKey, err := r.resolveConnectionAPIKey(ctx, config.Namespace, &config.Spec.Sonarr)
	if err != nil {
		r.Helper.SetCondition(statusWrapper, config.Generation, ConditionTypeReady, metav1.ConditionFalse, "SonarrSecretResolutionFailed", err.Error())
		config.Status.SonarrConnected = false
		if statusErr := r.Status().Update(ctx, config); statusErr != nil {
			log.Error(statusErr, "Failed to update status")
		}
		return ctrl.Result{RequeueAfter: ErrorRequeueInterval}, err
	}
	config.Status.SonarrConnected = true

	// Resolve Radarr API key
	radarrAPIKey, err := r.resolveConnectionAPIKey(ctx, config.Namespace, &config.Spec.Radarr)
	if err != nil {
		r.Helper.SetCondition(statusWrapper, config.Generation, ConditionTypeReady, metav1.ConditionFalse, "RadarrSecretResolutionFailed", err.Error())
		config.Status.RadarrConnected = false
		if statusErr := r.Status().Update(ctx, config); statusErr != nil {
			log.Error(statusErr, "Failed to update status")
		}
		return ctrl.Result{RequeueAfter: ErrorRequeueInterval}, err
	}
	config.Status.RadarrConnected = true

	// Resolve provider secrets
	providerSecrets, err := r.resolveProviderSecrets(ctx, config.Namespace, config.Spec.Providers)
	if err != nil {
		r.Helper.SetCondition(statusWrapper, config.Generation, ConditionTypeReady, metav1.ConditionFalse, "ProviderSecretResolutionFailed", err.Error())
		if statusErr := r.Status().Update(ctx, config); statusErr != nil {
			log.Error(statusErr, "Failed to update status")
		}
		return ctrl.Result{RequeueAfter: ErrorRequeueInterval}, err
	}

	// Resolve auth password if needed
	authPassword := ""
	if config.Spec.Authentication != nil && config.Spec.Authentication.PasswordSecretRef != nil {
		key := config.Spec.Authentication.PasswordSecretRef.Key
		if key == "" {
			key = "password"
		}
		authPassword, err = r.Helper.ResolveSecretValue(ctx, config.Namespace, config.Spec.Authentication.PasswordSecretRef.Name, key)
		if err != nil {
			r.Helper.SetCondition(statusWrapper, config.Generation, ConditionTypeReady, metav1.ConditionFalse, "AuthSecretResolutionFailed", err.Error())
			if statusErr := r.Status().Update(ctx, config); statusErr != nil {
				log.Error(statusErr, "Failed to update status")
			}
			return ctrl.Result{RequeueAfter: ErrorRequeueInterval}, err
		}
	}

	// Generate Bazarr config
	input := &bazarr.GeneratorInput{
		Spec:                    &config.Spec,
		SonarrAPIKey:            sonarrAPIKey,
		RadarrAPIKey:            radarrAPIKey,
		ResolvedProviderSecrets: providerSecrets,
		ResolvedAuthPassword:    authPassword,
	}

	generatedConfig, err := bazarr.Generate(input)
	if err != nil {
		log.Error(err, "Failed to generate Bazarr config")
		r.Helper.SetCondition(statusWrapper, config.Generation, ConditionTypeReady, metav1.ConditionFalse, "ConfigGenerationFailed", err.Error())
		if statusErr := r.Status().Update(ctx, config); statusErr != nil {
			log.Error(statusErr, "Failed to update status")
		}
		return ctrl.Result{RequeueAfter: ErrorRequeueInterval}, err
	}

	// Convert to YAML
	configYAML, err := bazarr.GenerateYAML(generatedConfig)
	if err != nil {
		log.Error(err, "Failed to serialize Bazarr config to YAML")
		r.Helper.SetCondition(statusWrapper, config.Generation, ConditionTypeReady, metav1.ConditionFalse, "YAMLSerializationFailed", err.Error())
		if statusErr := r.Status().Update(ctx, config); statusErr != nil {
			log.Error(statusErr, "Failed to update status")
		}
		return ctrl.Result{RequeueAfter: ErrorRequeueInterval}, err
	}

	// Write to ConfigMap if specified
	if config.Spec.ConfigMapRef != nil {
		if err := r.ensureConfigMap(ctx, config, configYAML); err != nil {
			log.Error(err, "Failed to update ConfigMap")
			r.Helper.SetCondition(statusWrapper, config.Generation, ConditionTypeReady, metav1.ConditionFalse, "ConfigMapUpdateFailed", err.Error())
			if statusErr := r.Status().Update(ctx, config); statusErr != nil {
				log.Error(statusErr, "Failed to update status")
			}
			return ctrl.Result{RequeueAfter: ErrorRequeueInterval}, err
		}
	}

	// Update status
	config.Status.ConfigGenerated = true
	now := metav1.Now()
	config.Status.LastReconcile = &now
	r.Helper.SetCondition(statusWrapper, config.Generation, ConditionTypeReady, metav1.ConditionTrue, "Ready", "Bazarr configuration generated successfully")

	if err := r.Status().Update(ctx, config); err != nil {
		log.Error(err, "Failed to update status")
		return ctrl.Result{}, err
	}

	// Determine requeue interval
	requeueAfter := DefaultRequeueInterval
	if config.Spec.Reconciliation != nil && config.Spec.Reconciliation.Interval != nil {
		requeueAfter = config.Spec.Reconciliation.Interval.Duration
	}

	log.Info("Successfully reconciled BazarrConfig", "name", config.Name)
	return ctrl.Result{RequeueAfter: requeueAfter}, nil
}

// reconcileDelete handles deletion of the BazarrConfig
func (r *BazarrConfigReconciler) reconcileDelete(ctx context.Context, config *arrv1alpha1.BazarrConfig) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	log.Info("Handling deletion of BazarrConfig", "name", config.Name)

	// Clean up ConfigMap if we created one
	if config.Spec.ConfigMapRef != nil {
		cm := &corev1.ConfigMap{}
		err := r.Get(ctx, client.ObjectKey{
			Namespace: config.Namespace,
			Name:      config.Spec.ConfigMapRef.Name,
		}, cm)
		if err == nil {
			// Check if we own this ConfigMap
			for _, ref := range cm.OwnerReferences {
				if ref.UID == config.UID {
					if err := r.Delete(ctx, cm); err != nil && !apierrors.IsNotFound(err) {
						log.Error(err, "Failed to delete ConfigMap")
					}
					break
				}
			}
		}
	}

	// Remove finalizer
	controllerutil.RemoveFinalizer(config, bazarrFinalizer)
	if err := r.Update(ctx, config); err != nil {
		return ctrl.Result{}, err
	}

	log.Info("Successfully deleted BazarrConfig", "name", config.Name)
	return ctrl.Result{}, nil
}

// resolveConnectionAPIKey resolves the API key for a Bazarr connection spec
func (r *BazarrConfigReconciler) resolveConnectionAPIKey(ctx context.Context, namespace string, conn *arrv1alpha1.BazarrConnectionSpec) (string, error) {
	if conn.APIKeySecretRef == nil {
		return "", nil // API key will be auto-discovered from config.xml
	}

	key := conn.APIKeySecretRef.Key
	if key == "" {
		key = "apiKey"
	}

	return r.Helper.ResolveSecretValue(ctx, namespace, conn.APIKeySecretRef.Name, key)
}

// resolveProviderSecrets resolves secrets for all providers
func (r *BazarrConfigReconciler) resolveProviderSecrets(ctx context.Context, namespace string, providers []arrv1alpha1.BazarrProvider) (map[string]bazarr.ProviderSecrets, error) {
	result := make(map[string]bazarr.ProviderSecrets)

	for _, provider := range providers {
		secrets := bazarr.ProviderSecrets{}

		// Resolve password if specified
		if provider.PasswordSecretRef != nil {
			key := provider.PasswordSecretRef.Key
			if key == "" {
				key = "password"
			}
			password, err := r.Helper.ResolveSecretValue(ctx, namespace, provider.PasswordSecretRef.Name, key)
			if err != nil {
				return nil, err
			}
			secrets.Password = password
		}

		// Resolve API key if specified
		if provider.APIKeySecretRef != nil {
			key := provider.APIKeySecretRef.Key
			if key == "" {
				key = "apiKey"
			}
			apiKey, err := r.Helper.ResolveSecretValue(ctx, namespace, provider.APIKeySecretRef.Name, key)
			if err != nil {
				return nil, err
			}
			secrets.APIKey = apiKey
		}

		result[provider.Name] = secrets
	}

	return result, nil
}

// ensureConfigMap creates or updates the ConfigMap with generated config
func (r *BazarrConfigReconciler) ensureConfigMap(ctx context.Context, config *arrv1alpha1.BazarrConfig, configData []byte) error {
	cm := &corev1.ConfigMap{}
	cmKey := client.ObjectKey{
		Namespace: config.Namespace,
		Name:      config.Spec.ConfigMapRef.Name,
	}

	err := r.Get(ctx, cmKey, cm)
	if apierrors.IsNotFound(err) {
		// Create new ConfigMap
		cm = &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      config.Spec.ConfigMapRef.Name,
				Namespace: config.Namespace,
			},
			Data: map[string]string{
				"config.yaml": string(configData),
			},
		}

		// Set owner reference
		if err := controllerutil.SetControllerReference(config, cm, r.Scheme); err != nil {
			return err
		}

		return r.Create(ctx, cm)
	}
	if err != nil {
		return err
	}

	// Update existing ConfigMap
	if cm.Data == nil {
		cm.Data = make(map[string]string)
	}
	cm.Data["config.yaml"] = string(configData)

	return r.Update(ctx, cm)
}

// SetupWithManager sets up the controller with the Manager.
func (r *BazarrConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Initialize helper if not set
	if r.Helper == nil {
		r.Helper = NewReconcileHelper(r.Client)
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&arrv1alpha1.BazarrConfig{}).
		Owns(&corev1.ConfigMap{}).
		Complete(r)
}
