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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// =============================================================================
// Connection Types
// =============================================================================

// ConnectionSpec defines how to connect to an *arr service
type ConnectionSpec struct {
	// URL is the base URL of the service (e.g., http://radarr:7878)
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^https?://`
	URL string `json:"url"`

	// APIKeySecretRef references a Secret containing the API key.
	// If not specified, auto-discovery is attempted.
	// +optional
	APIKeySecretRef *SecretKeySelector `json:"apiKeySecretRef,omitempty"`

	// ConfigPath is the path to config.xml for API key auto-discovery.
	// Only used if APIKeySecretRef is not specified.
	// Defaults to /{app}-config/config.xml
	// +optional
	ConfigPath string `json:"configPath,omitempty"`

	// InsecureSkipVerify disables TLS certificate verification.
	// +optional
	InsecureSkipVerify bool `json:"insecureSkipVerify,omitempty"`

	// Timeout specifies the connection timeout.
	// +optional
	// +kubebuilder:default="30s"
	Timeout *metav1.Duration `json:"timeout,omitempty"`
}

// SecretKeySelector selects a key from a Kubernetes Secret
type SecretKeySelector struct {
	// Name is the name of the Secret in the same namespace.
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Key is the key within the Secret.
	// +optional
	// +kubebuilder:default="apiKey"
	Key string `json:"key,omitempty"`
}

// CredentialsSecretRef references username/password from a Secret
type CredentialsSecretRef struct {
	// Name is the name of the Secret.
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// UsernameKey is the key for the username.
	// +optional
	// +kubebuilder:default="username"
	UsernameKey string `json:"usernameKey,omitempty"`

	// PasswordKey is the key for the password.
	// +optional
	// +kubebuilder:default="password"
	PasswordKey string `json:"passwordKey,omitempty"`
}

// LocalObjectReference references an object in the same namespace
type LocalObjectReference struct {
	// Name is the name of the referenced object.
	// +kubebuilder:validation:Required
	Name string `json:"name"`
}

// =============================================================================
// Quality Types
// =============================================================================

// VideoQualitySpec defines video quality preferences
type VideoQualitySpec struct {
	// Preset is a built-in quality configuration.
	// See PRESETS.md for available presets.
	// Valid values: uhd-hdr, uhd-sdr, fhd-quality, fhd-streaming, hd, balanced, any, storage-optimized
	// If not specified, defaults to "balanced".
	// +optional
	Preset string `json:"preset,omitempty"`

	// TemplateRef references a QualityTemplate for custom presets.
	// Mutually exclusive with Preset.
	// +optional
	TemplateRef *LocalObjectReference `json:"templateRef,omitempty"`

	// Exclude removes formats/features from the preset.
	// +optional
	Exclude []string `json:"exclude,omitempty"`

	// PreferAdditional adds formats to the preferred list.
	// +optional
	PreferAdditional []string `json:"preferAdditional,omitempty"`

	// RejectAdditional adds formats to the rejected list.
	// +optional
	RejectAdditional []string `json:"rejectAdditional,omitempty"`

	// --- Full manual control (overrides preset entirely if specified) ---

	// Tiers defines quality tiers in order of preference.
	// If specified, preset is ignored.
	// +optional
	Tiers []VideoQualityTier `json:"tiers,omitempty"`

	// UpgradeUntil defines the quality to upgrade until.
	// +optional
	UpgradeUntil *VideoQualityTier `json:"upgradeUntil,omitempty"`

	// PreferredFormats lists formats with positive scoring.
	// +optional
	PreferredFormats []string `json:"preferredFormats,omitempty"`

	// RejectedFormats lists formats to reject.
	// +optional
	RejectedFormats []string `json:"rejectedFormats,omitempty"`
}

// VideoQualityTier represents a resolution + source combination
type VideoQualityTier struct {
	// Resolution: 2160, 1080, 720, 480 (without 'p' suffix)
	// +kubebuilder:validation:Enum=2160;1080;720;480
	Resolution string `json:"resolution"`

	// Sources: bluray, remux, webdl, webrip, hdtv, dvd
	// +optional
	Sources []string `json:"sources,omitempty"`
}

// AudioQualitySpec defines audio quality preferences (for Lidarr)
type AudioQualitySpec struct {
	// Preset is a built-in quality configuration.
	// See PRESETS.md for available presets.
	// +optional
	// +kubebuilder:validation:Enum=lossless-hires;lossless;high-quality;balanced;portable;any
	Preset string `json:"preset,omitempty"`

	// TemplateRef references a QualityTemplate.
	// +optional
	TemplateRef *LocalObjectReference `json:"templateRef,omitempty"`

	// Exclude removes tiers/formats from the preset.
	// +optional
	Exclude []string `json:"exclude,omitempty"`

	// PreferAdditional adds formats to preferred list.
	// +optional
	PreferAdditional []string `json:"preferAdditional,omitempty"`

	// --- Full manual control ---

	// Tiers defines quality tiers: lossless-hires, lossless, lossy-high, lossy-mid, lossy-low
	// +optional
	Tiers []string `json:"tiers,omitempty"`

	// UpgradeUntil defines the tier to upgrade until.
	// +optional
	UpgradeUntil string `json:"upgradeUntil,omitempty"`

	// PreferredFormats: flac, alac, mp3-320, aac-320, etc.
	// +optional
	PreferredFormats []string `json:"preferredFormats,omitempty"`
}

// =============================================================================
// Download Client Types
// =============================================================================

// DownloadClientSpec defines a download client
type DownloadClientSpec struct {
	// Name is the display name for this client.
	// Also used for type inference if Type is not specified.
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// URL is the client URL (e.g., http://qbittorrent:8080)
	// +kubebuilder:validation:Required
	URL string `json:"url"`

	// Type is the client type. If not specified, inferred from Name.
	// +optional
	// +kubebuilder:validation:Enum=qbittorrent;transmission;deluge;rtorrent;nzbget;sabnzbd
	Type string `json:"type,omitempty"`

	// CredentialsSecretRef references username/password.
	// +optional
	CredentialsSecretRef *CredentialsSecretRef `json:"credentialsSecretRef,omitempty"`

	// Category for downloads. Defaults to app name (e.g., "radarr").
	// +optional
	Category string `json:"category,omitempty"`

	// Priority affects client selection (higher = preferred).
	// +optional
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=100
	// +kubebuilder:default=50
	Priority int `json:"priority,omitempty"`

	// Enabled enables/disables this client.
	// +optional
	// +kubebuilder:default=true
	Enabled *bool `json:"enabled,omitempty"`
}

// =============================================================================
// Indexer Types
// =============================================================================

// IndexersSpec defines indexer configuration
type IndexersSpec struct {
	// ProwlarrRef delegates indexer management to Prowlarr.
	// Mutually exclusive with Direct.
	// +optional
	ProwlarrRef *ProwlarrRef `json:"prowlarrRef,omitempty"`

	// Direct configures indexers directly (no Prowlarr).
	// Mutually exclusive with ProwlarrRef.
	// +optional
	Direct []DirectIndexer `json:"direct,omitempty"`
}

// ProwlarrRef references a Prowlarr instance for indexer management
type ProwlarrRef struct {
	// Name is the name of a ProwlarrConfig in the same namespace.
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// AutoRegister automatically registers this app with Prowlarr.
	// +optional
	// +kubebuilder:default=true
	AutoRegister *bool `json:"autoRegister,omitempty"`

	// Include filters which Prowlarr indexers to sync.
	// If empty, all indexers are synced.
	// +optional
	Include []string `json:"include,omitempty"`

	// Exclude filters out specific Prowlarr indexers.
	// +optional
	Exclude []string `json:"exclude,omitempty"`
}

// DirectIndexer defines an indexer configured directly
type DirectIndexer struct {
	// Name is the display name.
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// URL is the indexer URL.
	// +kubebuilder:validation:Required
	URL string `json:"url"`

	// Type: torrent or usenet
	// +kubebuilder:validation:Enum=torrent;usenet
	// +kubebuilder:default=torrent
	Type string `json:"type,omitempty"`

	// APIKeySecretRef for indexer authentication.
	// +optional
	APIKeySecretRef *SecretKeySelector `json:"apiKeySecretRef,omitempty"`

	// Categories to search. Human-readable (e.g., "movies-hd") or numeric IDs.
	// +optional
	Categories []string `json:"categories,omitempty"`

	// Priority (1-50, lower = higher priority).
	// +optional
	// +kubebuilder:default=25
	Priority int `json:"priority,omitempty"`

	// Enabled enables/disables this indexer.
	// +optional
	// +kubebuilder:default=true
	Enabled *bool `json:"enabled,omitempty"`
}

// =============================================================================
// Naming Types
// =============================================================================

// NamingSpec defines file/folder naming configuration
type NamingSpec struct {
	// Preset is a built-in naming configuration.
	// +optional
	// +kubebuilder:validation:Enum=plex-friendly;jellyfin-friendly;kodi-friendly;detailed;minimal;scene
	// +kubebuilder:default=plex-friendly
	Preset string `json:"preset,omitempty"`

	// --- Full manual control (overrides preset) ---

	// RenameMedia enables renaming (movies/episodes/tracks).
	// +optional
	RenameMedia *bool `json:"renameMedia,omitempty"`

	// StandardFormat is the format string for standard files.
	// +optional
	StandardFormat string `json:"standardFormat,omitempty"`

	// FolderFormat is the format string for folders.
	// +optional
	FolderFormat string `json:"folderFormat,omitempty"`
}

// =============================================================================
// Reconciliation Types
// =============================================================================

// ReconciliationSpec configures reconciliation behavior
type ReconciliationSpec struct {
	// Interval between reconciliations.
	// +optional
	// +kubebuilder:default="5m"
	Interval *metav1.Duration `json:"interval,omitempty"`

	// Suspend pauses reconciliation.
	// +optional
	Suspend bool `json:"suspend,omitempty"`
}

// =============================================================================
// Status Types
// =============================================================================

// ProwlarrRegistration tracks registration state with Prowlarr
type ProwlarrRegistration struct {
	// Registered indicates whether this app is registered with Prowlarr.
	// +optional
	Registered bool `json:"registered,omitempty"`

	// ProwlarrName is the name of the ProwlarrConfig this app is registered with.
	// +optional
	ProwlarrName string `json:"prowlarrName,omitempty"`

	// ApplicationID is the ID of this app in Prowlarr.
	// +optional
	ApplicationID *int `json:"applicationId,omitempty"`

	// LastSync is the timestamp of the last successful sync.
	// +optional
	LastSync *metav1.Time `json:"lastSync,omitempty"`

	// SyncedIndexers lists indexers synced from Prowlarr.
	// +optional
	SyncedIndexers []string `json:"syncedIndexers,omitempty"`

	// Message provides status details or error information.
	// +optional
	Message string `json:"message,omitempty"`
}

// ManagedResources tracks created resources
type ManagedResources struct {
	// QualityProfileID is the managed quality profile ID.
	// +optional
	QualityProfileID *int `json:"qualityProfileId,omitempty"`

	// CustomFormatIDs are the managed custom format IDs.
	// +optional
	CustomFormatIDs []int `json:"customFormatIds,omitempty"`

	// DownloadClientIDs are the managed download client IDs.
	// +optional
	DownloadClientIDs []int `json:"downloadClientIds,omitempty"`

	// IndexerIDs are the managed indexer IDs.
	// +optional
	IndexerIDs []int `json:"indexerIds,omitempty"`

	// RootFolderIDs are the managed root folder IDs.
	// +optional
	RootFolderIDs []int `json:"rootFolderIds,omitempty"`
}

// PolicyStatus is common status for all policies
type PolicyStatus struct {
	// Conditions represent the latest observations.
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// Realized indicates whether the policy has been applied.
	// +optional
	Realized bool `json:"realized,omitempty"`

	// Message provides additional status information.
	// +optional
	Message string `json:"message,omitempty"`
}

// HealthStatus represents the health state of an *arr app
type HealthStatus struct {
	// Healthy indicates whether the app has no error-level issues.
	// +optional
	Healthy bool `json:"healthy,omitempty"`

	// IssueCount is the total number of health issues.
	// +optional
	IssueCount int `json:"issueCount,omitempty"`

	// ErrorCount is the number of error-level issues.
	// +optional
	ErrorCount int `json:"errorCount,omitempty"`

	// WarningCount is the number of warning-level issues.
	// +optional
	WarningCount int `json:"warningCount,omitempty"`

	// LastCheck is the timestamp of the last health check.
	// +optional
	LastCheck *metav1.Time `json:"lastCheck,omitempty"`

	// Issues lists the current health issues.
	// +optional
	Issues []HealthIssueStatus `json:"issues,omitempty"`
}

// HealthIssueStatus represents a single health issue
type HealthIssueStatus struct {
	// Source identifies the check that produced this issue.
	// +optional
	Source string `json:"source,omitempty"`

	// Type is the severity: error, warning, notice.
	// +optional
	Type string `json:"type,omitempty"`

	// Message is the human-readable description.
	// +optional
	Message string `json:"message,omitempty"`

	// WikiURL is a link to documentation about this issue.
	// +optional
	WikiURL string `json:"wikiUrl,omitempty"`
}

// =============================================================================
// Import List Types
// =============================================================================

// ImportListSpec defines an import list configuration for Radarr/Sonarr/Lidarr
type ImportListSpec struct {
	// Name is the display name for this import list.
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Type is the import list implementation type.
	// For Radarr: IMDbListImport, TraktListImport, TraktPopularImport, TraktUserImport,
	//             PlexImport, RadarrImport, TMDbListImport, TMDbPopularImport, etc.
	// For Sonarr: SonarrImport, TraktListImport, TraktPopularImport, PlexImport,
	//             ImdbImport, etc.
	// For Lidarr: SpotifyFollowedArtists, SpotifyPlaylist, LastFmUser, etc.
	// +kubebuilder:validation:Required
	Type string `json:"type"`

	// Enabled enables/disables this import list.
	// +optional
	// +kubebuilder:default=true
	Enabled *bool `json:"enabled,omitempty"`

	// EnableAuto automatically adds items from this list.
	// +optional
	// +kubebuilder:default=true
	EnableAuto *bool `json:"enableAuto,omitempty"`

	// SearchOnAdd searches for items when added from this list.
	// +optional
	// +kubebuilder:default=true
	SearchOnAdd *bool `json:"searchOnAdd,omitempty"`

	// QualityProfile is the name of the quality profile to use.
	// +kubebuilder:validation:Required
	QualityProfile string `json:"qualityProfile"`

	// RootFolder is the root folder path for items from this list.
	// +kubebuilder:validation:Required
	RootFolder string `json:"rootFolder"`

	// --- Radarr-specific fields ---

	// Monitor specifies what to monitor. Radarr only.
	// +optional
	// +kubebuilder:validation:Enum=movieOnly;movieAndCollection;none
	// +kubebuilder:default=movieOnly
	Monitor string `json:"monitor,omitempty"`

	// MinimumAvailability specifies when the movie is considered available. Radarr only.
	// +optional
	// +kubebuilder:validation:Enum=tba;announced;inCinemas;released
	// +kubebuilder:default=announced
	MinimumAvailability string `json:"minimumAvailability,omitempty"`

	// --- Sonarr-specific fields ---

	// SeriesType specifies the series type. Sonarr only.
	// +optional
	// +kubebuilder:validation:Enum=standard;daily;anime
	// +kubebuilder:default=standard
	SeriesType string `json:"seriesType,omitempty"`

	// SeasonFolder enables season folders. Sonarr only.
	// +optional
	// +kubebuilder:default=true
	SeasonFolder *bool `json:"seasonFolder,omitempty"`

	// ShouldMonitor specifies what to monitor. Sonarr only.
	// +optional
	// +kubebuilder:validation:Enum=all;future;missing;existing;firstSeason;latestSeason;pilot;none
	// +kubebuilder:default=all
	ShouldMonitor string `json:"shouldMonitor,omitempty"`

	// --- Type-specific settings ---

	// Settings contains type-specific configuration.
	// Keys are camelCase API field names.
	// Examples:
	//   IMDb: listId (e.g., "ls123456", "top250")
	//   Trakt: username, listname, accessToken, refreshToken
	//   Plex: accessToken, serverUrl
	// +optional
	Settings map[string]string `json:"settings,omitempty"`

	// SettingsSecretRef references a Secret containing sensitive settings.
	// Secret keys should match the settings keys (e.g., accessToken).
	// Values from this secret override Settings.
	// +optional
	SettingsSecretRef *SecretKeySelector `json:"settingsSecretRef,omitempty"`
}

// =============================================================================
// Media Management Types
// =============================================================================

// MediaManagementSpec defines media management configuration
type MediaManagementSpec struct {
	// RecycleBin is the path to the recycle bin folder.
	// If empty, recycle bin is disabled.
	// +optional
	RecycleBin string `json:"recycleBin,omitempty"`

	// RecycleBinCleanupDays is the number of days before items are removed from recycle bin.
	// +optional
	// +kubebuilder:default=7
	// +kubebuilder:validation:Minimum=0
	RecycleBinCleanupDays *int `json:"recycleBinCleanupDays,omitempty"`

	// SetPermissions enables setting file permissions on Linux.
	// +optional
	// +kubebuilder:default=false
	SetPermissions *bool `json:"setPermissions,omitempty"`

	// ChmodFolder is the folder permission mode (e.g., "755").
	// +optional
	// +kubebuilder:default="755"
	ChmodFolder string `json:"chmodFolder,omitempty"`

	// ChownGroup is the group to set for files (Linux only).
	// +optional
	ChownGroup string `json:"chownGroup,omitempty"`

	// DeleteEmptyFolders removes empty folders after moving/deleting files.
	// +optional
	// +kubebuilder:default=false
	DeleteEmptyFolders *bool `json:"deleteEmptyFolders,omitempty"`

	// CreateEmptyFolders creates folders for artists/movies/series even when empty.
	// +optional
	// +kubebuilder:default=false
	CreateEmptyFolders *bool `json:"createEmptyFolders,omitempty"`

	// UseHardlinks uses hardlinks instead of copy when possible.
	// +optional
	// +kubebuilder:default=true
	UseHardlinks *bool `json:"useHardlinks,omitempty"`

	// --- Lidarr-specific fields ---

	// WatchLibraryForChanges monitors the library folder for changes. Lidarr only.
	// +optional
	WatchLibraryForChanges *bool `json:"watchLibraryForChanges,omitempty"`

	// AllowFingerprinting enables audio fingerprinting. Lidarr only.
	// +optional
	// +kubebuilder:validation:Enum=never;newFiles;always
	AllowFingerprinting string `json:"allowFingerprinting,omitempty"`
}

// =============================================================================
// Authentication Types
// =============================================================================

// AuthenticationSpec defines authentication configuration
type AuthenticationSpec struct {
	// Method is the authentication method.
	// +kubebuilder:validation:Enum=none;forms;external
	// +kubebuilder:default=none
	Method string `json:"method,omitempty"`

	// Username for forms authentication.
	// +optional
	Username string `json:"username,omitempty"`

	// PasswordSecretRef references the password Secret for forms authentication.
	// +optional
	PasswordSecretRef *SecretKeySelector `json:"passwordSecretRef,omitempty"`

	// AuthenticationRequired specifies when authentication is required.
	// +optional
	// +kubebuilder:validation:Enum=enabled;disabledForLocalAddresses
	// +kubebuilder:default=enabled
	AuthenticationRequired string `json:"authenticationRequired,omitempty"`
}
