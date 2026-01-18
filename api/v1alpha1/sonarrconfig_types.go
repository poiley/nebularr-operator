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

// SonarrNamingSpec extends NamingSpec with Sonarr-specific fields
type SonarrNamingSpec struct {
	NamingSpec `json:",inline"`

	// SeasonFolderFormat for season folders.
	// +optional
	SeasonFolderFormat string `json:"seasonFolderFormat,omitempty"`

	// DailyEpisodeFormat for daily shows.
	// +optional
	DailyEpisodeFormat string `json:"dailyEpisodeFormat,omitempty"`

	// AnimeEpisodeFormat for anime.
	// +optional
	AnimeEpisodeFormat string `json:"animeEpisodeFormat,omitempty"`
}

// SonarrConfigSpec defines the desired configuration for Sonarr
type SonarrConfigSpec struct {
	// Connection specifies how to connect to Sonarr.
	// +kubebuilder:validation:Required
	Connection ConnectionSpec `json:"connection"`

	// Quality defines TV quality preferences.
	// +optional
	Quality *VideoQualitySpec `json:"quality,omitempty"`

	// DownloadClients configures download clients.
	// +optional
	DownloadClients []DownloadClientSpec `json:"downloadClients,omitempty"`

	// RemotePathMappings maps download client paths to local paths.
	// Required when download clients and Sonarr see files at different paths.
	// +optional
	RemotePathMappings []RemotePathMappingSpec `json:"remotePathMappings,omitempty"`

	// Indexers configures indexer sources.
	// +optional
	Indexers *IndexersSpec `json:"indexers,omitempty"`

	// Naming configures file/folder naming.
	// +optional
	Naming *SonarrNamingSpec `json:"naming,omitempty"`

	// RootFolders configures root folder paths.
	// +optional
	RootFolders []string `json:"rootFolders,omitempty"`

	// ImportLists configures automatic import lists (Trakt, Plex, IMDb, etc.).
	// +optional
	ImportLists []ImportListSpec `json:"importLists,omitempty"`

	// MediaManagement configures media management settings.
	// +optional
	MediaManagement *MediaManagementSpec `json:"mediaManagement,omitempty"`

	// Authentication configures authentication settings.
	// +optional
	Authentication *AuthenticationSpec `json:"authentication,omitempty"`

	// Notifications configures notification connections (Discord, Slack, Email, etc.).
	// +optional
	Notifications []NotificationSpec `json:"notifications,omitempty"`

	// CustomFormats defines custom formats for fine-grained release quality control.
	// Custom formats allow matching releases based on title patterns, sources, resolutions, etc.
	// and assigning scores that affect quality profile decisions.
	// +optional
	CustomFormats []CustomFormatSpec `json:"customFormats,omitempty"`

	// DelayProfiles configures download delays for better release selection.
	// Delay profiles allow waiting for preferred releases before downloading,
	// with different delays for Usenet vs torrents and bypass conditions.
	// +optional
	DelayProfiles []DelayProfileSpec `json:"delayProfiles,omitempty"`

	// ReleaseProfiles configures release filtering and scoring.
	// Release profiles allow requiring/ignoring certain terms and scoring
	// releases based on preferred patterns. This is useful for filtering
	// out unwanted release groups or preferring specific qualities.
	// +optional
	ReleaseProfiles []ReleaseProfileSpec `json:"releaseProfiles,omitempty"`

	// Reconciliation configures sync behavior.
	// +optional
	Reconciliation *ReconciliationSpec `json:"reconciliation,omitempty"`
}

// SonarrConfigStatus defines the observed state of SonarrConfig
type SonarrConfigStatus struct {
	// Conditions represent the latest observations of the SonarrConfig's state.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// Connected indicates whether Sonarr is reachable.
	// +optional
	Connected bool `json:"connected,omitempty"`

	// ServiceVersion is the Sonarr version.
	// +optional
	ServiceVersion string `json:"serviceVersion,omitempty"`

	// LastReconcile is the timestamp of the last reconciliation.
	// +optional
	LastReconcile *metav1.Time `json:"lastReconcile,omitempty"`

	// ManagedResources lists resources created by this config.
	// +optional
	ManagedResources ManagedResources `json:"managedResources,omitempty"`

	// LastAppliedHash is the hash of the last applied spec.
	// +optional
	LastAppliedHash string `json:"lastAppliedHash,omitempty"`

	// ProwlarrRegistration tracks registration with Prowlarr (Pull Model).
	// +optional
	ProwlarrRegistration *ProwlarrRegistration `json:"prowlarrRegistration,omitempty"`

	// Health represents the app's health status from its internal health checks.
	// +optional
	Health *HealthStatus `json:"health,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="URL",type=string,JSONPath=`.spec.connection.url`
// +kubebuilder:printcolumn:name="Quality",type=string,JSONPath=`.spec.quality.preset`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// SonarrConfig is the all-in-one configuration for a Sonarr instance
type SonarrConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the desired configuration for Sonarr.
	// +kubebuilder:validation:Required
	Spec SonarrConfigSpec `json:"spec"`

	// Status defines the observed state of SonarrConfig.
	// +optional
	Status SonarrConfigStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SonarrConfigList contains a list of SonarrConfig
type SonarrConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SonarrConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SonarrConfig{}, &SonarrConfigList{})
}
