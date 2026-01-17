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

// MetadataProfileSpec defines which album types to allow
type MetadataProfileSpec struct {
	// PrimaryTypes: album, ep, single, broadcast, other
	// +optional
	PrimaryTypes []string `json:"primaryTypes,omitempty"`

	// SecondaryTypes: studio, compilation, soundtrack, live, remix, etc.
	// +optional
	SecondaryTypes []string `json:"secondaryTypes,omitempty"`

	// ReleaseStatuses: official, promotional, bootleg
	// +optional
	ReleaseStatuses []string `json:"releaseStatuses,omitempty"`
}

// LidarrRootFolder extends root folder with Lidarr requirements
type LidarrRootFolder struct {
	// Path is the root folder path.
	// +kubebuilder:validation:Required
	Path string `json:"path"`

	// Name is the display name for this root folder.
	// +optional
	Name string `json:"name,omitempty"`

	// DefaultMonitor: all, future, missing, existing, latest, first, none
	// +optional
	// +kubebuilder:validation:Enum=all;future;missing;existing;latest;first;none
	// +kubebuilder:default=all
	DefaultMonitor string `json:"defaultMonitor,omitempty"`
}

// LidarrNamingSpec for Lidarr naming
type LidarrNamingSpec struct {
	NamingSpec `json:",inline"`

	// ArtistFolderFormat for artist folders.
	// +optional
	ArtistFolderFormat string `json:"artistFolderFormat,omitempty"`

	// AlbumFolderFormat for album folders.
	// +optional
	AlbumFolderFormat string `json:"albumFolderFormat,omitempty"`
}

// LidarrConfigSpec defines the desired configuration for Lidarr
type LidarrConfigSpec struct {
	// Connection specifies how to connect to Lidarr.
	// Note: Lidarr uses API v1, not v3.
	// +kubebuilder:validation:Required
	Connection ConnectionSpec `json:"connection"`

	// Quality defines audio quality preferences.
	// +optional
	Quality *AudioQualitySpec `json:"quality,omitempty"`

	// Metadata configures which album types to include.
	// +optional
	Metadata *MetadataProfileSpec `json:"metadata,omitempty"`

	// DownloadClients configures download clients.
	// +optional
	DownloadClients []DownloadClientSpec `json:"downloadClients,omitempty"`

	// RemotePathMappings maps download client paths to local paths.
	// Required when download clients and Lidarr see files at different paths.
	// +optional
	RemotePathMappings []RemotePathMappingSpec `json:"remotePathMappings,omitempty"`

	// Indexers configures indexer sources.
	// +optional
	Indexers *IndexersSpec `json:"indexers,omitempty"`

	// Naming configures file/folder naming.
	// +optional
	Naming *LidarrNamingSpec `json:"naming,omitempty"`

	// RootFolders configures root folder paths.
	// Lidarr root folders require additional metadata.
	// +optional
	RootFolders []LidarrRootFolder `json:"rootFolders,omitempty"`

	// ImportLists configures automatic import lists (Spotify, Last.fm, etc.).
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

	// Reconciliation configures sync behavior.
	// +optional
	Reconciliation *ReconciliationSpec `json:"reconciliation,omitempty"`
}

// LidarrConfigStatus defines the observed state of LidarrConfig
type LidarrConfigStatus struct {
	// Conditions represent the latest observations of the LidarrConfig's state.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// Connected indicates whether Lidarr is reachable.
	// +optional
	Connected bool `json:"connected,omitempty"`

	// ServiceVersion is the Lidarr version.
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

// LidarrConfig is the all-in-one configuration for a Lidarr instance
type LidarrConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the desired configuration for Lidarr.
	// +kubebuilder:validation:Required
	Spec LidarrConfigSpec `json:"spec"`

	// Status defines the observed state of LidarrConfig.
	// +optional
	Status LidarrConfigStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// LidarrConfigList contains a list of LidarrConfig
type LidarrConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []LidarrConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&LidarrConfig{}, &LidarrConfigList{})
}
