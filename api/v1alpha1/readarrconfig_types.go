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

// ReadarrMetadataProfileSpec defines metadata profile preferences for Readarr.
// Metadata profiles control how book metadata is matched and handled.
type ReadarrMetadataProfileSpec struct {
	// Name is the profile name.
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// MinPopularity is the minimum Goodreads popularity for books.
	// +optional
	// +kubebuilder:validation:Minimum=0
	MinPopularity int `json:"minPopularity,omitempty"`

	// SkipMissingDate skips books without a release date.
	// +optional
	// +kubebuilder:default=true
	SkipMissingDate *bool `json:"skipMissingDate,omitempty"`

	// SkipMissingIsbn skips books without an ISBN.
	// +optional
	// +kubebuilder:default=false
	SkipMissingIsbn *bool `json:"skipMissingIsbn,omitempty"`

	// SkipPartsAndSets skips book parts and box sets.
	// +optional
	// +kubebuilder:default=false
	SkipPartsAndSets *bool `json:"skipPartsAndSets,omitempty"`

	// SkipSeriesSecondary skips secondary series entries.
	// +optional
	// +kubebuilder:default=false
	SkipSeriesSecondary *bool `json:"skipSeriesSecondary,omitempty"`

	// AllowedLanguages are the languages to allow.
	// If empty, all languages are allowed.
	// +optional
	AllowedLanguages []string `json:"allowedLanguages,omitempty"`
}

// ReadarrQualitySpec defines book quality preferences.
type ReadarrQualitySpec struct {
	// Preset selects a built-in quality configuration.
	// Options: "standard", "high-quality", "audiobook-focus"
	// +optional
	// +kubebuilder:validation:Enum=standard;high-quality;audiobook-focus
	Preset string `json:"preset,omitempty"`

	// AllowedFormats are the book formats to allow.
	// +optional
	AllowedFormats []ReadarrFormatSpec `json:"allowedFormats,omitempty"`

	// UpgradeAllowed permits upgrading to higher quality formats.
	// +optional
	// +kubebuilder:default=true
	UpgradeAllowed *bool `json:"upgradeAllowed,omitempty"`

	// Cutoff is the format quality that stops upgrades.
	// +optional
	Cutoff string `json:"cutoff,omitempty"`
}

// ReadarrFormatSpec defines an allowed book format.
type ReadarrFormatSpec struct {
	// Format is the book format type.
	// Options: EPUB, MOBI, AZW3, PDF, FLAC, MP3, M4B, Unknown
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=EPUB;MOBI;AZW3;PDF;FLAC;MP3;M4B;Unknown
	Format string `json:"format"`

	// Allowed indicates if this format is allowed.
	// +kubebuilder:default=true
	Allowed bool `json:"allowed,omitempty"`
}

// ReadarrNamingSpec extends NamingSpec with Readarr-specific fields
type ReadarrNamingSpec struct {
	NamingSpec `json:",inline"`

	// AuthorFolderFormat for author folders.
	// +optional
	AuthorFolderFormat string `json:"authorFolderFormat,omitempty"`

	// StandardBookFormat for book files.
	// +optional
	StandardBookFormat string `json:"standardBookFormat,omitempty"`

	// ColonReplacementFormat replaces colons in filenames.
	// +optional
	// +kubebuilder:validation:Enum=delete;dash;spaceDash;spaceDashSpace
	ColonReplacementFormat string `json:"colonReplacementFormat,omitempty"`
}

// ReadarrConfigSpec defines the desired configuration for Readarr
type ReadarrConfigSpec struct {
	// Connection specifies how to connect to Readarr.
	// +kubebuilder:validation:Required
	Connection ConnectionSpec `json:"connection"`

	// Quality defines book quality preferences.
	// +optional
	Quality *ReadarrQualitySpec `json:"quality,omitempty"`

	// MetadataProfile defines metadata profile preferences.
	// Controls how book metadata is matched from Goodreads/other sources.
	// +optional
	MetadataProfile *ReadarrMetadataProfileSpec `json:"metadataProfile,omitempty"`

	// DownloadClients configures download clients.
	// +optional
	DownloadClients []DownloadClientSpec `json:"downloadClients,omitempty"`

	// RemotePathMappings maps download client paths to local paths.
	// +optional
	RemotePathMappings []RemotePathMappingSpec `json:"remotePathMappings,omitempty"`

	// Indexers configures indexer sources.
	// +optional
	Indexers *IndexersSpec `json:"indexers,omitempty"`

	// Naming configures file/folder naming.
	// +optional
	Naming *ReadarrNamingSpec `json:"naming,omitempty"`

	// RootFolders configures root folder paths for book storage.
	// +optional
	RootFolders []string `json:"rootFolders,omitempty"`

	// ImportLists configures automatic import lists (Goodreads, LazyLibrarian, etc.).
	// +optional
	ImportLists []ImportListSpec `json:"importLists,omitempty"`

	// MediaManagement configures media management settings.
	// +optional
	MediaManagement *MediaManagementSpec `json:"mediaManagement,omitempty"`

	// Authentication configures authentication settings.
	// +optional
	Authentication *AuthenticationSpec `json:"authentication,omitempty"`

	// Notifications configures notification connections.
	// +optional
	Notifications []NotificationSpec `json:"notifications,omitempty"`

	// Reconciliation configures sync behavior.
	// +optional
	Reconciliation *ReconciliationSpec `json:"reconciliation,omitempty"`
}

// ReadarrConfigStatus defines the observed state of ReadarrConfig
type ReadarrConfigStatus struct {
	// Conditions represent the latest observations of the ReadarrConfig's state.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// Connected indicates whether Readarr is reachable.
	// +optional
	Connected bool `json:"connected,omitempty"`

	// ServiceVersion is the Readarr version.
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

// ReadarrConfig is the all-in-one configuration for a Readarr instance
type ReadarrConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the desired configuration for Readarr.
	// +kubebuilder:validation:Required
	Spec ReadarrConfigSpec `json:"spec"`

	// Status defines the observed state of ReadarrConfig.
	// +optional
	Status ReadarrConfigStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ReadarrConfigList contains a list of ReadarrConfig
type ReadarrConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ReadarrConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ReadarrConfig{}, &ReadarrConfigList{})
}
