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

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	arrv1alpha1 "github.com/poiley/nebularr-operator/api/v1alpha1"
	"github.com/poiley/nebularr-operator/internal/adapters/downloadstack"
)

const (
	downloadStackFinalizer = "downloadstackconfig.arr.rinzler.cloud/finalizer"

	// Annotation keys for Deployment restart
	restartAnnotationKey    = "downloadstack.arr.rinzler.cloud/restartedAt"
	configHashAnnotationKey = "downloadstack.arr.rinzler.cloud/gluetun-hash"
)

// DownloadStackConfigReconciler reconciles a DownloadStackConfig object
type DownloadStackConfigReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Helper *ReconcileHelper
}

// +kubebuilder:rbac:groups=arr.rinzler.cloud,resources=downloadstackconfigs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=arr.rinzler.cloud,resources=downloadstackconfigs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=arr.rinzler.cloud,resources=downloadstackconfigs/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;update;patch

// Reconcile is part of the main kubernetes reconciliation loop
func (r *DownloadStackConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	// Fetch the DownloadStackConfig
	config := &arrv1alpha1.DownloadStackConfig{}
	if err := r.Get(ctx, req.NamespacedName, config); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("DownloadStackConfig resource not found, ignoring")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get DownloadStackConfig")
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
	if !controllerutil.ContainsFinalizer(config, downloadStackFinalizer) {
		controllerutil.AddFinalizer(config, downloadStackFinalizer)
		if err := r.Update(ctx, config); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// Reconcile the configuration
	return r.reconcileNormal(ctx, config)
}

// reconcileNormal handles the normal reconciliation flow
func (r *DownloadStackConfigReconciler) reconcileNormal(ctx context.Context, config *arrv1alpha1.DownloadStackConfig) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	log.Info("Reconciling DownloadStackConfig", "name", config.Name)

	statusWrapper := &DownloadStackStatusWrapper{Status: &config.Status}
	now := metav1.Now()

	// =========================================================================
	// PHASE 1: Gluetun Configuration
	// =========================================================================

	// Resolve Gluetun credentials
	gluetunInput := &downloadstack.GluetunEnvInput{
		Spec: &config.Spec.Gluetun,
	}

	if config.Spec.Gluetun.Provider.CredentialsSecretRef != nil {
		creds := config.Spec.Gluetun.Provider.CredentialsSecretRef
		usernameKey := creds.UsernameKey
		if usernameKey == "" {
			usernameKey = "username"
		}
		passwordKey := creds.PasswordKey
		if passwordKey == "" {
			passwordKey = "password"
		}

		username, err := r.Helper.ResolveSecretValue(ctx, config.Namespace, creds.Name, usernameKey)
		if err != nil {
			r.Helper.SetCondition(statusWrapper, config.Generation, ConditionTypeReady, metav1.ConditionFalse, "GluetunCredentialsFailed", err.Error())
			if statusErr := r.Status().Update(ctx, config); statusErr != nil {
				log.Error(statusErr, "Failed to update status")
			}
			return ctrl.Result{RequeueAfter: ErrorRequeueInterval}, err
		}
		gluetunInput.Username = username

		password, err := r.Helper.ResolveSecretValue(ctx, config.Namespace, creds.Name, passwordKey)
		if err != nil {
			r.Helper.SetCondition(statusWrapper, config.Generation, ConditionTypeReady, metav1.ConditionFalse, "GluetunCredentialsFailed", err.Error())
			if statusErr := r.Status().Update(ctx, config); statusErr != nil {
				log.Error(statusErr, "Failed to update status")
			}
			return ctrl.Result{RequeueAfter: ErrorRequeueInterval}, err
		}
		gluetunInput.Password = password
	}

	// For WireGuard private key
	if config.Spec.Gluetun.Provider.PrivateKeySecretRef != nil {
		keyRef := config.Spec.Gluetun.Provider.PrivateKeySecretRef
		keyName := keyRef.Key
		if keyName == "" {
			keyName = "privateKey"
		}
		privateKey, err := r.Helper.ResolveSecretValue(ctx, config.Namespace, keyRef.Name, keyName)
		if err != nil {
			r.Helper.SetCondition(statusWrapper, config.Generation, ConditionTypeReady, metav1.ConditionFalse, "GluetunPrivateKeyFailed", err.Error())
			if statusErr := r.Status().Update(ctx, config); statusErr != nil {
				log.Error(statusErr, "Failed to update status")
			}
			return ctrl.Result{RequeueAfter: ErrorRequeueInterval}, err
		}
		gluetunInput.PrivateKey = privateKey
	}

	// Generate Gluetun env vars
	gluetunEnv := downloadstack.GenerateGluetunEnv(gluetunInput)
	newHash := downloadstack.HashGluetunEnv(gluetunEnv)

	// Create/update Gluetun env Secret
	secretName := config.Name + "-gluetun-env"
	gluetunSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: config.Namespace,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, gluetunSecret, func() error {
		// Set owner reference
		if err := controllerutil.SetControllerReference(config, gluetunSecret, r.Scheme); err != nil {
			return err
		}

		// Convert env map to StringData
		gluetunSecret.StringData = gluetunEnv
		return nil
	})
	if err != nil {
		r.Helper.SetCondition(statusWrapper, config.Generation, ConditionTypeReady, metav1.ConditionFalse, "GluetunSecretFailed", err.Error())
		if statusErr := r.Status().Update(ctx, config); statusErr != nil {
			log.Error(statusErr, "Failed to update status")
		}
		return ctrl.Result{RequeueAfter: ErrorRequeueInterval}, err
	}

	config.Status.GluetunSecretGenerated = true

	// Check if Gluetun config changed and needs restart
	configChanged := newHash != config.Status.GluetunConfigHash
	config.Status.GluetunConfigHash = newHash

	// Trigger Deployment restart if config changed
	if configChanged && config.Spec.RestartOnGluetunChange {
		if err := r.restartDeployment(ctx, config); err != nil {
			log.Error(err, "Failed to trigger Deployment restart", "deployment", config.Spec.DeploymentRef.Name)
			// Don't fail reconciliation for this
		} else {
			log.Info("Triggered Deployment restart due to Gluetun config change", "deployment", config.Spec.DeploymentRef.Name)
		}
	}

	// =========================================================================
	// PHASE 2: Transmission Configuration
	// =========================================================================

	// Resolve Transmission credentials (optional)
	var transmissionUsername, transmissionPassword string
	if config.Spec.Transmission.Connection.CredentialsSecretRef != nil {
		creds := config.Spec.Transmission.Connection.CredentialsSecretRef
		usernameKey := creds.UsernameKey
		if usernameKey == "" {
			usernameKey = "username"
		}
		passwordKey := creds.PasswordKey
		if passwordKey == "" {
			passwordKey = "password"
		}

		var err error
		transmissionUsername, err = r.Helper.ResolveSecretValue(ctx, config.Namespace, creds.Name, usernameKey)
		if err != nil {
			r.Helper.SetCondition(statusWrapper, config.Generation, ConditionTypeReady, metav1.ConditionFalse, "TransmissionCredentialsFailed", err.Error())
			if statusErr := r.Status().Update(ctx, config); statusErr != nil {
				log.Error(statusErr, "Failed to update status")
			}
			return ctrl.Result{RequeueAfter: ErrorRequeueInterval}, err
		}

		transmissionPassword, err = r.Helper.ResolveSecretValue(ctx, config.Namespace, creds.Name, passwordKey)
		if err != nil {
			r.Helper.SetCondition(statusWrapper, config.Generation, ConditionTypeReady, metav1.ConditionFalse, "TransmissionCredentialsFailed", err.Error())
			if statusErr := r.Status().Update(ctx, config); statusErr != nil {
				log.Error(statusErr, "Failed to update status")
			}
			return ctrl.Result{RequeueAfter: ErrorRequeueInterval}, err
		}
	}

	// Create Transmission client and sync settings
	transmissionClient := downloadstack.NewTransmissionClient(
		config.Spec.Transmission.Connection.URL,
		transmissionUsername,
		transmissionPassword,
	)

	// Test connection
	if err := transmissionClient.TestConnection(ctx); err != nil {
		log.Error(err, "Failed to connect to Transmission")
		config.Status.TransmissionConnected = false
		r.Helper.SetCondition(statusWrapper, config.Generation, ConditionTypeReady, metav1.ConditionFalse, "TransmissionConnectionFailed", err.Error())
		if statusErr := r.Status().Update(ctx, config); statusErr != nil {
			log.Error(statusErr, "Failed to update status")
		}
		return ctrl.Result{RequeueAfter: ErrorRequeueInterval}, err
	}

	config.Status.TransmissionConnected = true

	// Get Transmission version
	version, err := downloadstack.GetTransmissionVersion(ctx, transmissionClient)
	if err == nil {
		config.Status.TransmissionVersion = version
	}

	// Sync Transmission settings
	settingsInput := &downloadstack.TransmissionSettingsInput{
		Spec:     &config.Spec.Transmission,
		Username: transmissionUsername,
		Password: transmissionPassword,
	}

	if err := downloadstack.SyncTransmissionSettings(ctx, transmissionClient, settingsInput); err != nil {
		log.Error(err, "Failed to sync Transmission settings")
		r.Helper.SetCondition(statusWrapper, config.Generation, ConditionTypeReady, metav1.ConditionFalse, "TransmissionSyncFailed", err.Error())
		if statusErr := r.Status().Update(ctx, config); statusErr != nil {
			log.Error(statusErr, "Failed to update status")
		}
		return ctrl.Result{RequeueAfter: ErrorRequeueInterval}, err
	}

	// =========================================================================
	// Success
	// =========================================================================

	config.Status.LastReconcile = &now
	config.Status.ObservedGeneration = config.Generation
	r.Helper.SetCondition(statusWrapper, config.Generation, ConditionTypeReady, metav1.ConditionTrue, "Reconciled", "Configuration applied successfully")

	if err := r.Status().Update(ctx, config); err != nil {
		log.Error(err, "Failed to update status")
		return ctrl.Result{}, err
	}

	// Determine requeue interval
	requeueAfter := DefaultRequeueInterval
	if config.Spec.Reconciliation != nil && config.Spec.Reconciliation.Interval != nil {
		requeueAfter = config.Spec.Reconciliation.Interval.Duration
	}

	log.Info("Successfully reconciled DownloadStackConfig", "requeueAfter", requeueAfter)
	return ctrl.Result{RequeueAfter: requeueAfter}, nil
}

// reconcileDelete handles cleanup when the resource is being deleted
func (r *DownloadStackConfigReconciler) reconcileDelete(ctx context.Context, config *arrv1alpha1.DownloadStackConfig) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	log.Info("Reconciling deletion of DownloadStackConfig", "name", config.Name)

	// The Gluetun Secret will be garbage collected due to owner reference

	// Remove finalizer
	controllerutil.RemoveFinalizer(config, downloadStackFinalizer)
	if err := r.Update(ctx, config); err != nil {
		return ctrl.Result{}, err
	}

	log.Info("Successfully removed finalizer from DownloadStackConfig")
	return ctrl.Result{}, nil
}

// restartDeployment annotates the Deployment to trigger a restart
func (r *DownloadStackConfigReconciler) restartDeployment(ctx context.Context, config *arrv1alpha1.DownloadStackConfig) error {
	deployment := &appsv1.Deployment{}
	if err := r.Get(ctx, client.ObjectKey{
		Namespace: config.Namespace,
		Name:      config.Spec.DeploymentRef.Name,
	}, deployment); err != nil {
		return fmt.Errorf("failed to get deployment: %w", err)
	}

	// Add/update restart annotations
	if deployment.Spec.Template.Annotations == nil {
		deployment.Spec.Template.Annotations = make(map[string]string)
	}
	deployment.Spec.Template.Annotations[restartAnnotationKey] = time.Now().Format(time.RFC3339)
	deployment.Spec.Template.Annotations[configHashAnnotationKey] = config.Status.GluetunConfigHash

	if err := r.Update(ctx, deployment); err != nil {
		return fmt.Errorf("failed to update deployment: %w", err)
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager
func (r *DownloadStackConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Initialize helper if not set
	if r.Helper == nil {
		r.Helper = NewReconcileHelper(r.Client)
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&arrv1alpha1.DownloadStackConfig{}).
		Owns(&corev1.Secret{}).
		Complete(r)
}
