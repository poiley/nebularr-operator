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

// TransmissionClientFactory creates Transmission clients.
// This allows for dependency injection in tests.
type TransmissionClientFactory func(url, username, password string) downloadstack.TransmissionClientInterface

// DownloadStackConfigReconciler reconciles a DownloadStackConfig object
type DownloadStackConfigReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Helper *ReconcileHelper

	// TransmissionClientFactory creates Transmission clients.
	// If nil, uses the default downloadstack.NewTransmissionClient.
	TransmissionClientFactory TransmissionClientFactory
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
	// PHASE 2: Download Client Configuration
	// =========================================================================

	// Validate at least one download client is configured
	if config.Spec.Transmission == nil && config.Spec.QBittorrent == nil && config.Spec.Deluge == nil && config.Spec.RTorrent == nil && config.Spec.SABnzbd == nil && config.Spec.NZBGet == nil {
		r.Helper.SetCondition(statusWrapper, config.Generation, ConditionTypeReady, metav1.ConditionFalse, "NoDownloadClient", "At least one download client (Transmission, qBittorrent, Deluge, rTorrent, SABnzbd, or NZBGet) must be configured")
		if statusErr := r.Status().Update(ctx, config); statusErr != nil {
			log.Error(statusErr, "Failed to update status")
		}
		return ctrl.Result{RequeueAfter: ErrorRequeueInterval}, fmt.Errorf("no download client configured")
	}

	// -------------------------------------------------------------------------
	// Transmission Configuration (if specified)
	// -------------------------------------------------------------------------
	if config.Spec.Transmission != nil {
		if err := r.reconcileTransmission(ctx, config, statusWrapper); err != nil {
			// Update status before returning error so conditions are persisted
			if statusErr := r.Status().Update(ctx, config); statusErr != nil {
				log.Error(statusErr, "Failed to update status after Transmission error")
			}
			return ctrl.Result{RequeueAfter: ErrorRequeueInterval}, err
		}
	}

	// -------------------------------------------------------------------------
	// qBittorrent Configuration (if specified)
	// -------------------------------------------------------------------------
	if config.Spec.QBittorrent != nil {
		if err := r.reconcileQBittorrent(ctx, config, statusWrapper); err != nil {
			// Update status before returning error so conditions are persisted
			if statusErr := r.Status().Update(ctx, config); statusErr != nil {
				log.Error(statusErr, "Failed to update status after qBittorrent error")
			}
			return ctrl.Result{RequeueAfter: ErrorRequeueInterval}, err
		}
	}

	// -------------------------------------------------------------------------
	// Deluge Configuration (if specified)
	// -------------------------------------------------------------------------
	if config.Spec.Deluge != nil {
		if err := r.reconcileDeluge(ctx, config, statusWrapper); err != nil {
			// Update status before returning error so conditions are persisted
			if statusErr := r.Status().Update(ctx, config); statusErr != nil {
				log.Error(statusErr, "Failed to update status after Deluge error")
			}
			return ctrl.Result{RequeueAfter: ErrorRequeueInterval}, err
		}
	}

	// -------------------------------------------------------------------------
	// rTorrent Configuration (if specified)
	// -------------------------------------------------------------------------
	if config.Spec.RTorrent != nil {
		if err := r.reconcileRTorrent(ctx, config, statusWrapper); err != nil {
			// Update status before returning error so conditions are persisted
			if statusErr := r.Status().Update(ctx, config); statusErr != nil {
				log.Error(statusErr, "Failed to update status after rTorrent error")
			}
			return ctrl.Result{RequeueAfter: ErrorRequeueInterval}, err
		}
	}

	// -------------------------------------------------------------------------
	// SABnzbd Configuration (if specified)
	// -------------------------------------------------------------------------
	if config.Spec.SABnzbd != nil {
		if err := r.reconcileSABnzbd(ctx, config, statusWrapper); err != nil {
			// Update status before returning error so conditions are persisted
			if statusErr := r.Status().Update(ctx, config); statusErr != nil {
				log.Error(statusErr, "Failed to update status after SABnzbd error")
			}
			return ctrl.Result{RequeueAfter: ErrorRequeueInterval}, err
		}
	}

	// -------------------------------------------------------------------------
	// NZBGet Configuration (if specified)
	// -------------------------------------------------------------------------
	if config.Spec.NZBGet != nil {
		if err := r.reconcileNZBGet(ctx, config, statusWrapper); err != nil {
			// Update status before returning error so conditions are persisted
			if statusErr := r.Status().Update(ctx, config); statusErr != nil {
				log.Error(statusErr, "Failed to update status after NZBGet error")
			}
			return ctrl.Result{RequeueAfter: ErrorRequeueInterval}, err
		}
	}

	// =========================================================================
	// Success
	// =========================================================================

	config.Status.LastReconcile = &now
	config.Status.ObservedGeneration = config.Generation
	r.Helper.SetCondition(statusWrapper, config.Generation, ConditionTypeReady, metav1.ConditionTrue, "Reconciled", "Configuration applied successfully")

	if err := r.Status().Update(ctx, config); err != nil {
		log.Error(err, "Failed to update final status")
		return ctrl.Result{}, err
	}

	// Determine requeue interval
	requeueAfter := DefaultRequeueInterval
	if config.Spec.Reconciliation != nil && config.Spec.Reconciliation.Interval != nil {
		requeueAfter = config.Spec.Reconciliation.Interval.Duration
	}

	log.Info("Successfully reconciled DownloadStackConfig", "name", config.Name)
	return ctrl.Result{RequeueAfter: requeueAfter}, nil
}

// reconcileTransmission handles Transmission configuration
func (r *DownloadStackConfigReconciler) reconcileTransmission(ctx context.Context, config *arrv1alpha1.DownloadStackConfig, statusWrapper *DownloadStackStatusWrapper) error {
	log := logf.FromContext(ctx)

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
			return err
		}

		transmissionPassword, err = r.Helper.ResolveSecretValue(ctx, config.Namespace, creds.Name, passwordKey)
		if err != nil {
			r.Helper.SetCondition(statusWrapper, config.Generation, ConditionTypeReady, metav1.ConditionFalse, "TransmissionCredentialsFailed", err.Error())
			return err
		}
	}

	// Create Transmission client and sync settings
	var transmissionClient downloadstack.TransmissionClientInterface
	if r.TransmissionClientFactory != nil {
		transmissionClient = r.TransmissionClientFactory(
			config.Spec.Transmission.Connection.URL,
			transmissionUsername,
			transmissionPassword,
		)
	} else {
		transmissionClient = downloadstack.NewTransmissionClient(
			config.Spec.Transmission.Connection.URL,
			transmissionUsername,
			transmissionPassword,
		)
	}

	// Test connection
	if err := transmissionClient.TestConnection(ctx); err != nil {
		log.Error(err, "Failed to connect to Transmission")
		config.Status.TransmissionConnected = false
		r.Helper.SetCondition(statusWrapper, config.Generation, ConditionTypeReady, metav1.ConditionFalse, "TransmissionConnectionFailed", err.Error())
		return err
	}

	config.Status.TransmissionConnected = true

	// Get Transmission version
	version, err := downloadstack.GetTransmissionVersion(ctx, transmissionClient)
	if err == nil {
		config.Status.TransmissionVersion = version
	}

	// Sync Transmission settings
	settingsInput := &downloadstack.TransmissionSettingsInput{
		Spec:     config.Spec.Transmission,
		Username: transmissionUsername,
		Password: transmissionPassword,
	}

	if err := downloadstack.SyncTransmissionSettings(ctx, transmissionClient, settingsInput); err != nil {
		log.Error(err, "Failed to sync Transmission settings")
		r.Helper.SetCondition(statusWrapper, config.Generation, ConditionTypeReady, metav1.ConditionFalse, "TransmissionSyncFailed", err.Error())
		return err
	}

	log.Info("Transmission configuration synced successfully")
	return nil
}

// reconcileQBittorrent handles qBittorrent configuration
func (r *DownloadStackConfigReconciler) reconcileQBittorrent(ctx context.Context, config *arrv1alpha1.DownloadStackConfig, statusWrapper *DownloadStackStatusWrapper) error {
	log := logf.FromContext(ctx)

	// Resolve qBittorrent credentials (optional)
	var qbtUsername, qbtPassword string
	if config.Spec.QBittorrent.Connection.CredentialsSecretRef != nil {
		creds := config.Spec.QBittorrent.Connection.CredentialsSecretRef
		usernameKey := creds.UsernameKey
		if usernameKey == "" {
			usernameKey = "username"
		}
		passwordKey := creds.PasswordKey
		if passwordKey == "" {
			passwordKey = "password"
		}

		var err error
		qbtUsername, err = r.Helper.ResolveSecretValue(ctx, config.Namespace, creds.Name, usernameKey)
		if err != nil {
			r.Helper.SetCondition(statusWrapper, config.Generation, ConditionTypeReady, metav1.ConditionFalse, "QBittorrentCredentialsFailed", err.Error())
			return err
		}

		qbtPassword, err = r.Helper.ResolveSecretValue(ctx, config.Namespace, creds.Name, passwordKey)
		if err != nil {
			r.Helper.SetCondition(statusWrapper, config.Generation, ConditionTypeReady, metav1.ConditionFalse, "QBittorrentCredentialsFailed", err.Error())
			return err
		}
	}

	// Create qBittorrent client
	qbtClient := downloadstack.NewQBittorrentClient(
		config.Spec.QBittorrent.Connection.URL,
		qbtUsername,
		qbtPassword,
	)

	// Test connection
	if err := qbtClient.TestConnection(ctx); err != nil {
		log.Error(err, "Failed to connect to qBittorrent")
		config.Status.QBittorrentConnected = false
		r.Helper.SetCondition(statusWrapper, config.Generation, ConditionTypeReady, metav1.ConditionFalse, "QBittorrentConnectionFailed", err.Error())
		return err
	}

	config.Status.QBittorrentConnected = true

	// Get qBittorrent version
	version, err := qbtClient.GetVersion(ctx)
	if err == nil {
		config.Status.QBittorrentVersion = version
	}

	// Sync qBittorrent settings
	if err := syncQBittorrentSettings(ctx, qbtClient, config.Spec.QBittorrent); err != nil {
		log.Error(err, "Failed to sync qBittorrent settings")
		r.Helper.SetCondition(statusWrapper, config.Generation, ConditionTypeReady, metav1.ConditionFalse, "QBittorrentSyncFailed", err.Error())
		return err
	}

	log.Info("qBittorrent configuration synced successfully")
	return nil
}

// syncQBittorrentSettings syncs qBittorrent preferences from spec
func syncQBittorrentSettings(ctx context.Context, client *downloadstack.QBittorrentClient, spec *arrv1alpha1.QBittorrentSpec) error {
	prefs := make(map[string]interface{})

	// Speed settings
	if spec.Speed != nil {
		if spec.Speed.DownloadLimit > 0 {
			prefs["dl_limit"] = spec.Speed.DownloadLimit * 1024 // KiB to bytes
		}
		if spec.Speed.UploadLimit > 0 {
			prefs["up_limit"] = spec.Speed.UploadLimit * 1024
		}
		if spec.Speed.GlobalDownloadSpeedLimit > 0 {
			prefs["dl_limit"] = spec.Speed.GlobalDownloadSpeedLimit * 1024
		}
		if spec.Speed.GlobalUploadSpeedLimit > 0 {
			prefs["up_limit"] = spec.Speed.GlobalUploadSpeedLimit * 1024
		}
	}

	// Alt-speed settings
	if spec.AltSpeed != nil {
		prefs["alt_dl_limit"] = spec.AltSpeed.DownloadLimit * 1024
		prefs["alt_up_limit"] = spec.AltSpeed.UploadLimit * 1024
		prefs["scheduler_enabled"] = spec.AltSpeed.SchedulerEnabled
		if spec.AltSpeed.SchedulerDays > 0 {
			prefs["scheduler_days"] = spec.AltSpeed.SchedulerDays
		}
		prefs["schedule_from_hour"] = spec.AltSpeed.ScheduleFromHour
		prefs["schedule_from_min"] = spec.AltSpeed.ScheduleFromMinute
		prefs["schedule_to_hour"] = spec.AltSpeed.ScheduleToHour
		prefs["schedule_to_min"] = spec.AltSpeed.ScheduleToMinute
	}

	// Directory settings
	if spec.Directories != nil {
		if spec.Directories.SavePath != "" {
			prefs["save_path"] = spec.Directories.SavePath
		}
		if spec.Directories.TempPath != "" {
			prefs["temp_path"] = spec.Directories.TempPath
		}
		prefs["temp_path_enabled"] = spec.Directories.TempPathEnabled
		if spec.Directories.CreateSubfolder != nil {
			prefs["create_subfolder_enabled"] = *spec.Directories.CreateSubfolder
		}
		if spec.Directories.AppendExtension != nil {
			prefs["incomplete_files_ext"] = *spec.Directories.AppendExtension
		}
	}

	// Seeding settings
	if spec.Seeding != nil {
		prefs["max_ratio_enabled"] = spec.Seeding.MaxRatioEnabled
		if spec.Seeding.MaxRatio != "" {
			// Parse ratio string to float
			var ratio float64
			if _, err := fmt.Sscanf(spec.Seeding.MaxRatio, "%f", &ratio); err == nil {
				prefs["max_ratio"] = ratio
			}
		}
		prefs["max_seeding_time_enabled"] = spec.Seeding.MaxSeedingTimeEnabled
		if spec.Seeding.MaxSeedingTime > 0 {
			prefs["max_seeding_time"] = spec.Seeding.MaxSeedingTime
		}
		if spec.Seeding.MaxRatioAction != nil {
			prefs["max_ratio_act"] = *spec.Seeding.MaxRatioAction
		}
	}

	// Queue settings
	if spec.Queue != nil {
		if spec.Queue.QueueingEnabled != nil {
			prefs["queueing_enabled"] = *spec.Queue.QueueingEnabled
		}
		if spec.Queue.MaxActiveDownloads > 0 {
			prefs["max_active_downloads"] = spec.Queue.MaxActiveDownloads
		}
		if spec.Queue.MaxActiveUploads > 0 {
			prefs["max_active_uploads"] = spec.Queue.MaxActiveUploads
		}
		if spec.Queue.MaxActiveTorrents > 0 {
			prefs["max_active_torrents"] = spec.Queue.MaxActiveTorrents
		}
	}

	// Connection settings
	if spec.Connections != nil {
		if spec.Connections.MaxConnections > 0 {
			prefs["max_connec"] = spec.Connections.MaxConnections
		}
		if spec.Connections.MaxConnectionsPerTorrent > 0 {
			prefs["max_connec_per_torrent"] = spec.Connections.MaxConnectionsPerTorrent
		}
		if spec.Connections.MaxUploads > 0 {
			prefs["max_uploads"] = spec.Connections.MaxUploads
		}
		if spec.Connections.MaxUploadsPerTorrent > 0 {
			prefs["max_uploads_per_torrent"] = spec.Connections.MaxUploadsPerTorrent
		}
		if spec.Connections.ListenPort > 0 {
			prefs["listen_port"] = spec.Connections.ListenPort
		}
		prefs["random_port"] = spec.Connections.RandomPort
		if spec.Connections.UPnPEnabled != nil {
			prefs["upnp"] = *spec.Connections.UPnPEnabled
		}
	}

	// BitTorrent protocol settings
	if spec.BitTorrent != nil {
		if spec.BitTorrent.DHT != nil {
			prefs["dht"] = *spec.BitTorrent.DHT
		}
		if spec.BitTorrent.PeX != nil {
			prefs["pex"] = *spec.BitTorrent.PeX
		}
		if spec.BitTorrent.LSD != nil {
			prefs["lsd"] = *spec.BitTorrent.LSD
		}
		if spec.BitTorrent.Encryption != nil {
			prefs["encryption"] = *spec.BitTorrent.Encryption
		}
		prefs["anonymous_mode"] = spec.BitTorrent.AnonymousMode
	}

	// Only set preferences if there are any
	if len(prefs) > 0 {
		return client.SetPreferences(ctx, prefs)
	}

	return nil
}

// reconcileDeluge handles Deluge configuration
func (r *DownloadStackConfigReconciler) reconcileDeluge(ctx context.Context, config *arrv1alpha1.DownloadStackConfig, statusWrapper *DownloadStackStatusWrapper) error {
	log := logf.FromContext(ctx)

	// Resolve Deluge password (optional, defaults to "deluge")
	var delugePassword string = "deluge"
	if config.Spec.Deluge.Connection.PasswordSecretRef != nil {
		keyRef := config.Spec.Deluge.Connection.PasswordSecretRef
		keyName := keyRef.Key
		if keyName == "" {
			keyName = "password"
		}

		var err error
		delugePassword, err = r.Helper.ResolveSecretValue(ctx, config.Namespace, keyRef.Name, keyName)
		if err != nil {
			r.Helper.SetCondition(statusWrapper, config.Generation, ConditionTypeReady, metav1.ConditionFalse, "DelugeCredentialsFailed", err.Error())
			return err
		}
	}

	// Create Deluge client
	delugeClient := downloadstack.NewDelugeClient(
		config.Spec.Deluge.Connection.URL,
		delugePassword,
	)

	// Test connection
	if err := delugeClient.TestConnection(ctx); err != nil {
		log.Error(err, "Failed to connect to Deluge")
		config.Status.DelugeConnected = false
		r.Helper.SetCondition(statusWrapper, config.Generation, ConditionTypeReady, metav1.ConditionFalse, "DelugeConnectionFailed", err.Error())
		return err
	}

	config.Status.DelugeConnected = true

	// Get Deluge version
	version, err := delugeClient.GetVersion(ctx)
	if err == nil {
		config.Status.DelugeVersion = version
	}

	// Sync Deluge settings
	if err := syncDelugeSettings(ctx, delugeClient, config.Spec.Deluge); err != nil {
		log.Error(err, "Failed to sync Deluge settings")
		r.Helper.SetCondition(statusWrapper, config.Generation, ConditionTypeReady, metav1.ConditionFalse, "DelugeSyncFailed", err.Error())
		return err
	}

	log.Info("Deluge configuration synced successfully")
	return nil
}

// syncDelugeSettings syncs Deluge configuration from spec
func syncDelugeSettings(ctx context.Context, client *downloadstack.DelugeClient, spec *arrv1alpha1.DelugeSpec) error {
	config := make(map[string]interface{})

	// Speed settings
	if spec.Speed != nil {
		if spec.Speed.MaxDownloadSpeed != 0 {
			config["max_download_speed"] = float64(spec.Speed.MaxDownloadSpeed)
		}
		if spec.Speed.MaxUploadSpeed != 0 {
			config["max_upload_speed"] = float64(spec.Speed.MaxUploadSpeed)
		}
		if spec.Speed.MaxDownloadSpeedPerTorrent != 0 {
			config["max_download_speed_per_torrent"] = float64(spec.Speed.MaxDownloadSpeedPerTorrent)
		}
		if spec.Speed.MaxUploadSpeedPerTorrent != 0 {
			config["max_upload_speed_per_torrent"] = float64(spec.Speed.MaxUploadSpeedPerTorrent)
		}
	}

	// Directory settings
	if spec.Directories != nil {
		if spec.Directories.DownloadLocation != "" {
			config["download_location"] = spec.Directories.DownloadLocation
		}
		config["move_completed"] = spec.Directories.MoveCompleted
		if spec.Directories.MoveCompletedPath != "" {
			config["move_completed_path"] = spec.Directories.MoveCompletedPath
		}
		config["copy_torrent_file"] = spec.Directories.CopyTorrentFile
		if spec.Directories.TorrentFilesLocation != "" {
			config["torrentfiles_location"] = spec.Directories.TorrentFilesLocation
		}
	}

	// Seeding settings
	if spec.Seeding != nil {
		config["stop_seed_at_ratio"] = spec.Seeding.StopSeedAtRatio
		if spec.Seeding.StopSeedRatio != "" {
			var ratio float64
			if _, err := fmt.Sscanf(spec.Seeding.StopSeedRatio, "%f", &ratio); err == nil {
				config["stop_seed_ratio"] = ratio
			}
		}
		config["remove_seed_at_ratio"] = spec.Seeding.RemoveAtRatio
		if spec.Seeding.ShareRatioLimit != "" {
			var ratio float64
			if _, err := fmt.Sscanf(spec.Seeding.ShareRatioLimit, "%f", &ratio); err == nil {
				config["share_ratio_limit"] = ratio
			}
		}
		if spec.Seeding.SeedTimeLimit != 0 {
			config["seed_time_limit"] = spec.Seeding.SeedTimeLimit
		}
	}

	// Queue settings
	if spec.Queue != nil {
		if spec.Queue.MaxActiveDownloading > 0 {
			config["max_active_downloading"] = spec.Queue.MaxActiveDownloading
		}
		if spec.Queue.MaxActiveSeeding > 0 {
			config["max_active_seeding"] = spec.Queue.MaxActiveSeeding
		}
		if spec.Queue.MaxActiveLimit > 0 {
			config["max_active_limit"] = spec.Queue.MaxActiveLimit
		}
		config["queue_new_to_top"] = spec.Queue.QueueNewToTop
	}

	// Connection settings
	if spec.Connections != nil {
		if spec.Connections.MaxConnections > 0 {
			config["max_connections_global"] = spec.Connections.MaxConnections
		}
		if spec.Connections.MaxConnectionsPerTorrent > 0 {
			config["max_connections_per_torrent"] = spec.Connections.MaxConnectionsPerTorrent
		}
		if spec.Connections.MaxUploadSlots > 0 {
			config["max_upload_slots_global"] = spec.Connections.MaxUploadSlots
		}
		if spec.Connections.MaxUploadSlotsPerTorrent > 0 {
			config["max_upload_slots_per_torrent"] = spec.Connections.MaxUploadSlotsPerTorrent
		}
		if len(spec.Connections.ListenPorts) == 2 {
			config["listen_ports"] = spec.Connections.ListenPorts
		}
		config["random_port"] = spec.Connections.RandomPort
	}

	// Protocol settings
	if spec.Protocol != nil {
		if spec.Protocol.DHT != nil {
			config["dht"] = *spec.Protocol.DHT
		}
		if spec.Protocol.UPnP != nil {
			config["upnp"] = *spec.Protocol.UPnP
		}
		if spec.Protocol.NATPMP != nil {
			config["natpmp"] = *spec.Protocol.NATPMP
		}
		if spec.Protocol.LSD != nil {
			config["lsd"] = *spec.Protocol.LSD
		}
		if spec.Protocol.ProtocolEncryption != nil {
			config["pe_enabled"] = *spec.Protocol.ProtocolEncryption
		}
		if spec.Protocol.EncryptionLevel != nil {
			config["enc_level"] = *spec.Protocol.EncryptionLevel
		}
	}

	// Only set config if there are any changes
	if len(config) > 0 {
		return client.SetConfig(ctx, config)
	}

	return nil
}

// reconcileRTorrent handles rTorrent configuration
func (r *DownloadStackConfigReconciler) reconcileRTorrent(ctx context.Context, config *arrv1alpha1.DownloadStackConfig, statusWrapper *DownloadStackStatusWrapper) error {
	log := logf.FromContext(ctx)

	// Resolve rTorrent credentials (optional - for HTTP basic auth)
	var rtUsername, rtPassword string
	if config.Spec.RTorrent.Connection.CredentialsSecretRef != nil {
		creds := config.Spec.RTorrent.Connection.CredentialsSecretRef
		usernameKey := creds.UsernameKey
		if usernameKey == "" {
			usernameKey = "username"
		}
		passwordKey := creds.PasswordKey
		if passwordKey == "" {
			passwordKey = "password"
		}

		var err error
		rtUsername, err = r.Helper.ResolveSecretValue(ctx, config.Namespace, creds.Name, usernameKey)
		if err != nil {
			r.Helper.SetCondition(statusWrapper, config.Generation, ConditionTypeReady, metav1.ConditionFalse, "RTorrentCredentialsFailed", err.Error())
			return err
		}

		rtPassword, err = r.Helper.ResolveSecretValue(ctx, config.Namespace, creds.Name, passwordKey)
		if err != nil {
			r.Helper.SetCondition(statusWrapper, config.Generation, ConditionTypeReady, metav1.ConditionFalse, "RTorrentCredentialsFailed", err.Error())
			return err
		}
	}

	// Create rTorrent client
	rtClient := downloadstack.NewRTorrentClient(
		config.Spec.RTorrent.Connection.URL,
		rtUsername,
		rtPassword,
	)

	// Test connection
	if err := rtClient.TestConnection(ctx); err != nil {
		log.Error(err, "Failed to connect to rTorrent")
		config.Status.RTorrentConnected = false
		r.Helper.SetCondition(statusWrapper, config.Generation, ConditionTypeReady, metav1.ConditionFalse, "RTorrentConnectionFailed", err.Error())
		return err
	}

	config.Status.RTorrentConnected = true

	// Get rTorrent version
	version, err := rtClient.GetVersion(ctx)
	if err == nil {
		config.Status.RTorrentVersion = version
	}

	// Sync rTorrent settings
	if err := syncRTorrentSettings(ctx, rtClient, config.Spec.RTorrent); err != nil {
		log.Error(err, "Failed to sync rTorrent settings")
		r.Helper.SetCondition(statusWrapper, config.Generation, ConditionTypeReady, metav1.ConditionFalse, "RTorrentSyncFailed", err.Error())
		return err
	}

	log.Info("rTorrent configuration synced successfully")
	return nil
}

// syncRTorrentSettings syncs rTorrent configuration from spec
func syncRTorrentSettings(ctx context.Context, client *downloadstack.RTorrentClient, spec *arrv1alpha1.RTorrentSpec) error {
	// Speed settings
	if spec.Speed != nil {
		if spec.Speed.DownloadRate > 0 {
			// Convert KiB/s to bytes/s
			if err := client.SetDownloadRate(ctx, int64(spec.Speed.DownloadRate)*1024); err != nil {
				return fmt.Errorf("failed to set download rate: %w", err)
			}
		}
		if spec.Speed.UploadRate > 0 {
			if err := client.SetUploadRate(ctx, int64(spec.Speed.UploadRate)*1024); err != nil {
				return fmt.Errorf("failed to set upload rate: %w", err)
			}
		}
	}

	// Directory settings
	if spec.Directories != nil {
		if spec.Directories.Directory != "" {
			if err := client.SetDirectory(ctx, spec.Directories.Directory); err != nil {
				return fmt.Errorf("failed to set directory: %w", err)
			}
		}
		// Session directory is typically set in config file, not via RPC
	}

	// Connection settings
	if spec.Connections != nil {
		if spec.Connections.MaxPeers > 0 {
			if err := client.SetMaxPeers(ctx, int64(spec.Connections.MaxPeers)); err != nil {
				return fmt.Errorf("failed to set max peers: %w", err)
			}
		}
		if spec.Connections.MaxUploads > 0 {
			if err := client.SetMaxUploads(ctx, int64(spec.Connections.MaxUploads)); err != nil {
				return fmt.Errorf("failed to set max uploads: %w", err)
			}
		}
	}

	// Protocol settings
	if spec.Protocol != nil {
		if spec.Protocol.DHT != nil {
			mode := "off"
			if *spec.Protocol.DHT {
				mode = "auto"
			}
			if err := client.SetDHTMode(ctx, mode); err != nil {
				return fmt.Errorf("failed to set DHT mode: %w", err)
			}
		}
		if spec.Protocol.Encryption != "" {
			if err := client.SetEncryptionMode(ctx, spec.Protocol.Encryption); err != nil {
				return fmt.Errorf("failed to set encryption mode: %w", err)
			}
		}
	}

	return nil
}

// reconcileSABnzbd handles SABnzbd configuration
func (r *DownloadStackConfigReconciler) reconcileSABnzbd(ctx context.Context, config *arrv1alpha1.DownloadStackConfig, statusWrapper *DownloadStackStatusWrapper) error {
	log := logf.FromContext(ctx)

	// Resolve SABnzbd API key (required)
	keyRef := &config.Spec.SABnzbd.Connection.APIKeySecretRef
	keyName := keyRef.Key
	if keyName == "" {
		keyName = "apiKey"
	}

	apiKey, err := r.Helper.ResolveSecretValue(ctx, config.Namespace, keyRef.Name, keyName)
	if err != nil {
		r.Helper.SetCondition(statusWrapper, config.Generation, ConditionTypeReady, metav1.ConditionFalse, "SABnzbdCredentialsFailed", err.Error())
		return err
	}

	// Create SABnzbd client
	sabClient := downloadstack.NewSABnzbdClient(
		config.Spec.SABnzbd.Connection.URL,
		apiKey,
	)

	// Test connection
	if err := sabClient.TestConnection(ctx); err != nil {
		log.Error(err, "Failed to connect to SABnzbd")
		config.Status.SABnzbdConnected = false
		r.Helper.SetCondition(statusWrapper, config.Generation, ConditionTypeReady, metav1.ConditionFalse, "SABnzbdConnectionFailed", err.Error())
		return err
	}

	config.Status.SABnzbdConnected = true

	// Get SABnzbd version
	version, err := sabClient.GetVersion(ctx)
	if err == nil {
		config.Status.SABnzbdVersion = version
	}

	// Sync SABnzbd settings
	if err := syncSABnzbdSettings(ctx, sabClient, config.Spec.SABnzbd); err != nil {
		log.Error(err, "Failed to sync SABnzbd settings")
		r.Helper.SetCondition(statusWrapper, config.Generation, ConditionTypeReady, metav1.ConditionFalse, "SABnzbdSyncFailed", err.Error())
		return err
	}

	log.Info("SABnzbd configuration synced successfully")
	return nil
}

// syncSABnzbdSettings syncs SABnzbd configuration from spec
func syncSABnzbdSettings(ctx context.Context, client *downloadstack.SABnzbdClient, spec *arrv1alpha1.SABnzbdSpec) error {
	// Speed settings
	if spec.Speed != nil {
		if spec.Speed.SpeedLimit > 0 {
			if err := client.SetSpeedLimit(ctx, spec.Speed.SpeedLimit); err != nil {
				return fmt.Errorf("failed to set speed limit: %w", err)
			}
		}
		if spec.Speed.PauseDownloads {
			if err := client.Pause(ctx); err != nil {
				return fmt.Errorf("failed to pause downloads: %w", err)
			}
		}
	}

	// Directory settings
	if spec.Directories != nil {
		if spec.Directories.DownloadDir != "" {
			if err := client.SetConfig(ctx, "misc", "download_dir", spec.Directories.DownloadDir); err != nil {
				return fmt.Errorf("failed to set download_dir: %w", err)
			}
		}
		if spec.Directories.CompleteDir != "" {
			if err := client.SetConfig(ctx, "misc", "complete_dir", spec.Directories.CompleteDir); err != nil {
				return fmt.Errorf("failed to set complete_dir: %w", err)
			}
		}
		if spec.Directories.IncompleteDir != "" {
			if err := client.SetConfig(ctx, "misc", "incomplete_dir", spec.Directories.IncompleteDir); err != nil {
				return fmt.Errorf("failed to set incomplete_dir: %w", err)
			}
		}
		if spec.Directories.ScriptDir != "" {
			if err := client.SetConfig(ctx, "misc", "script_dir", spec.Directories.ScriptDir); err != nil {
				return fmt.Errorf("failed to set script_dir: %w", err)
			}
		}
		if spec.Directories.NzbBackupDir != "" {
			if err := client.SetConfig(ctx, "misc", "nzb_backup_dir", spec.Directories.NzbBackupDir); err != nil {
				return fmt.Errorf("failed to set nzb_backup_dir: %w", err)
			}
		}
	}

	// Queue settings
	if spec.Queue != nil {
		if spec.Queue.PreCheck {
			if err := client.SetConfig(ctx, "misc", "pre_check", "1"); err != nil {
				return fmt.Errorf("failed to set pre_check: %w", err)
			}
		}
		if spec.Queue.MaxRetries > 0 {
			if err := client.SetConfig(ctx, "misc", "max_art_tries", fmt.Sprintf("%d", spec.Queue.MaxRetries)); err != nil {
				return fmt.Errorf("failed to set max_art_tries: %w", err)
			}
		}
	}

	// Post-processing settings
	if spec.PostProcessing != nil {
		if spec.PostProcessing.UnpackEnabled {
			if err := client.SetConfig(ctx, "misc", "unpack", "1"); err != nil {
				return fmt.Errorf("failed to set unpack: %w", err)
			}
		}
		if spec.PostProcessing.CleanupEnabled {
			if err := client.SetConfig(ctx, "misc", "cleanup_list", "1"); err != nil {
				return fmt.Errorf("failed to set cleanup_list: %w", err)
			}
		}
	}

	return nil
}

// reconcileNZBGet handles NZBGet configuration
func (r *DownloadStackConfigReconciler) reconcileNZBGet(ctx context.Context, config *arrv1alpha1.DownloadStackConfig, statusWrapper *DownloadStackStatusWrapper) error {
	log := logf.FromContext(ctx)

	// Resolve NZBGet credentials (optional - defaults to nzbget:tegbzn6789)
	var nzbgetUsername, nzbgetPassword string = "nzbget", "tegbzn6789"
	if config.Spec.NZBGet.Connection.CredentialsSecretRef != nil {
		creds := config.Spec.NZBGet.Connection.CredentialsSecretRef
		usernameKey := creds.UsernameKey
		if usernameKey == "" {
			usernameKey = "username"
		}
		passwordKey := creds.PasswordKey
		if passwordKey == "" {
			passwordKey = "password"
		}

		var err error
		nzbgetUsername, err = r.Helper.ResolveSecretValue(ctx, config.Namespace, creds.Name, usernameKey)
		if err != nil {
			r.Helper.SetCondition(statusWrapper, config.Generation, ConditionTypeReady, metav1.ConditionFalse, "NZBGetCredentialsFailed", err.Error())
			return err
		}

		nzbgetPassword, err = r.Helper.ResolveSecretValue(ctx, config.Namespace, creds.Name, passwordKey)
		if err != nil {
			r.Helper.SetCondition(statusWrapper, config.Generation, ConditionTypeReady, metav1.ConditionFalse, "NZBGetCredentialsFailed", err.Error())
			return err
		}
	}

	// Create NZBGet client
	nzbgetClient := downloadstack.NewNZBGetClient(
		config.Spec.NZBGet.Connection.URL,
		nzbgetUsername,
		nzbgetPassword,
	)

	// Test connection
	if err := nzbgetClient.TestConnection(ctx); err != nil {
		log.Error(err, "Failed to connect to NZBGet")
		config.Status.NZBGetConnected = false
		r.Helper.SetCondition(statusWrapper, config.Generation, ConditionTypeReady, metav1.ConditionFalse, "NZBGetConnectionFailed", err.Error())
		return err
	}

	config.Status.NZBGetConnected = true

	// Get NZBGet version
	version, err := nzbgetClient.GetVersion(ctx)
	if err == nil {
		config.Status.NZBGetVersion = version
	}

	// Sync NZBGet settings
	if err := syncNZBGetSettings(ctx, nzbgetClient, config.Spec.NZBGet); err != nil {
		log.Error(err, "Failed to sync NZBGet settings")
		r.Helper.SetCondition(statusWrapper, config.Generation, ConditionTypeReady, metav1.ConditionFalse, "NZBGetSyncFailed", err.Error())
		return err
	}

	log.Info("NZBGet configuration synced successfully")
	return nil
}

// syncNZBGetSettings syncs NZBGet configuration from spec
func syncNZBGetSettings(ctx context.Context, client *downloadstack.NZBGetClient, spec *arrv1alpha1.NZBGetSpec) error {
	// Speed settings
	if spec.Speed != nil {
		if spec.Speed.DownloadRate > 0 {
			if err := client.SetDownloadRate(ctx, spec.Speed.DownloadRate); err != nil {
				return fmt.Errorf("failed to set download rate: %w", err)
			}
		}
		if spec.Speed.ArticleTimeout > 0 {
			if err := client.SetConfig(ctx, "ArticleTimeout", fmt.Sprintf("%d", spec.Speed.ArticleTimeout)); err != nil {
				return fmt.Errorf("failed to set ArticleTimeout: %w", err)
			}
		}
		if spec.Speed.WriteBuffer > 0 {
			if err := client.SetConfig(ctx, "WriteBuffer", fmt.Sprintf("%d", spec.Speed.WriteBuffer)); err != nil {
				return fmt.Errorf("failed to set WriteBuffer: %w", err)
			}
		}
	}

	// Directory settings
	if spec.Directories != nil {
		if spec.Directories.MainDir != "" {
			if err := client.SetConfig(ctx, "MainDir", spec.Directories.MainDir); err != nil {
				return fmt.Errorf("failed to set MainDir: %w", err)
			}
		}
		if spec.Directories.DestDir != "" {
			if err := client.SetConfig(ctx, "DestDir", spec.Directories.DestDir); err != nil {
				return fmt.Errorf("failed to set DestDir: %w", err)
			}
		}
		if spec.Directories.InterDir != "" {
			if err := client.SetConfig(ctx, "InterDir", spec.Directories.InterDir); err != nil {
				return fmt.Errorf("failed to set InterDir: %w", err)
			}
		}
		if spec.Directories.NzbDir != "" {
			if err := client.SetConfig(ctx, "NzbDir", spec.Directories.NzbDir); err != nil {
				return fmt.Errorf("failed to set NzbDir: %w", err)
			}
		}
		if spec.Directories.TempDir != "" {
			if err := client.SetConfig(ctx, "TempDir", spec.Directories.TempDir); err != nil {
				return fmt.Errorf("failed to set TempDir: %w", err)
			}
		}
		if spec.Directories.ScriptDir != "" {
			if err := client.SetConfig(ctx, "ScriptDir", spec.Directories.ScriptDir); err != nil {
				return fmt.Errorf("failed to set ScriptDir: %w", err)
			}
		}
	}

	// Queue settings
	if spec.Queue != nil {
		if spec.Queue.DupeCheck {
			if err := client.SetConfig(ctx, "DupeCheck", "yes"); err != nil {
				return fmt.Errorf("failed to set DupeCheck: %w", err)
			}
		}
		if spec.Queue.PropagationDelay > 0 {
			if err := client.SetConfig(ctx, "PropagationDelay", fmt.Sprintf("%d", spec.Queue.PropagationDelay)); err != nil {
				return fmt.Errorf("failed to set PropagationDelay: %w", err)
			}
		}
		if spec.Queue.HealthCheck != "" {
			if err := client.SetConfig(ctx, "HealthCheck", spec.Queue.HealthCheck); err != nil {
				return fmt.Errorf("failed to set HealthCheck: %w", err)
			}
		}
	}

	// Post-processing settings
	if spec.PostProcessing != nil {
		if spec.PostProcessing.ParCheck != "" {
			if err := client.SetConfig(ctx, "ParCheck", spec.PostProcessing.ParCheck); err != nil {
				return fmt.Errorf("failed to set ParCheck: %w", err)
			}
		}
		if spec.PostProcessing.ParRepair != nil {
			value := "no"
			if *spec.PostProcessing.ParRepair {
				value = "yes"
			}
			if err := client.SetConfig(ctx, "ParRepair", value); err != nil {
				return fmt.Errorf("failed to set ParRepair: %w", err)
			}
		}
		if spec.PostProcessing.Unpack != nil {
			value := "no"
			if *spec.PostProcessing.Unpack {
				value = "yes"
			}
			if err := client.SetConfig(ctx, "Unpack", value); err != nil {
				return fmt.Errorf("failed to set Unpack: %w", err)
			}
		}
		if spec.PostProcessing.UnpackCleanupDisk != nil {
			value := "no"
			if *spec.PostProcessing.UnpackCleanupDisk {
				value = "yes"
			}
			if err := client.SetConfig(ctx, "UnpackCleanupDisk", value); err != nil {
				return fmt.Errorf("failed to set UnpackCleanupDisk: %w", err)
			}
		}
		if spec.PostProcessing.DirectUnpack != nil {
			value := "no"
			if *spec.PostProcessing.DirectUnpack {
				value = "yes"
			}
			if err := client.SetConfig(ctx, "DirectUnpack", value); err != nil {
				return fmt.Errorf("failed to set DirectUnpack: %w", err)
			}
		}
	}

	// Connection settings
	if spec.Connections != nil {
		if spec.Connections.ArticleConnections > 0 {
			if err := client.SetConfig(ctx, "ArticleConnections", fmt.Sprintf("%d", spec.Connections.ArticleConnections)); err != nil {
				return fmt.Errorf("failed to set ArticleConnections: %w", err)
			}
		}
		if spec.Connections.RetryInterval > 0 {
			if err := client.SetConfig(ctx, "RetryInterval", fmt.Sprintf("%d", spec.Connections.RetryInterval)); err != nil {
				return fmt.Errorf("failed to set RetryInterval: %w", err)
			}
		}
		if spec.Connections.TerminateTimeout > 0 {
			if err := client.SetConfig(ctx, "TerminateTimeout", fmt.Sprintf("%d", spec.Connections.TerminateTimeout)); err != nil {
				return fmt.Errorf("failed to set TerminateTimeout: %w", err)
			}
		}
		if spec.Connections.Decode != nil {
			value := "no"
			if *spec.Connections.Decode {
				value = "yes"
			}
			if err := client.SetConfig(ctx, "Decode", value); err != nil {
				return fmt.Errorf("failed to set Decode: %w", err)
			}
		}
	}

	return nil
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
