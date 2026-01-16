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

// ProwlarrIndexer defines a native indexer in Prowlarr
type ProwlarrIndexer struct {
	// Name is the display name.
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Definition is the indexer definition (e.g., "1337x", "Nyaa", "IPTorrents").
	// +kubebuilder:validation:Required
	Definition string `json:"definition"`

	// BaseURL overrides the default URL for the indexer.
	// +optional
	BaseURL string `json:"baseUrl,omitempty"`

	// Settings are definition-specific settings.
	// +optional
	Settings map[string]string `json:"settings,omitempty"`

	// APIKeySecretRef for private indexers.
	// +optional
	APIKeySecretRef *SecretKeySelector `json:"apiKeySecretRef,omitempty"`

	// Tags associate this indexer with proxies.
	// +optional
	Tags []string `json:"tags,omitempty"`

	// Priority (1-50).
	// +optional
	// +kubebuilder:default=25
	Priority int `json:"priority,omitempty"`

	// Enabled enables/disables this indexer.
	// +optional
	// +kubebuilder:default=true
	Enabled *bool `json:"enabled,omitempty"`
}

// IndexerProxy defines a proxy for indexer requests
type IndexerProxy struct {
	// Name is the display name.
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Type: flaresolverr, http, socks4, socks5
	// +kubebuilder:validation:Enum=flaresolverr;http;socks4;socks5
	// +kubebuilder:validation:Required
	Type string `json:"type"`

	// Host is the proxy URL or hostname.
	// For FlareSolverr: full URL (http://flaresolverr:8191)
	// For HTTP/SOCKS: hostname only
	// +kubebuilder:validation:Required
	Host string `json:"host"`

	// Port for HTTP/SOCKS proxies.
	// +optional
	Port int `json:"port,omitempty"`

	// CredentialsSecretRef for authenticated proxies.
	// +optional
	CredentialsSecretRef *CredentialsSecretRef `json:"credentialsSecretRef,omitempty"`

	// RequestTimeout for FlareSolverr (seconds).
	// +optional
	// +kubebuilder:default=60
	RequestTimeout int `json:"requestTimeout,omitempty"`

	// Tags to associate with indexers that should use this proxy.
	// +optional
	Tags []string `json:"tags,omitempty"`
}

// ProwlarrApplication defines sync to a downstream app
type ProwlarrApplication struct {
	// Name is the display name.
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Type: radarr, sonarr, lidarr
	// +kubebuilder:validation:Enum=radarr;sonarr;lidarr
	// +kubebuilder:validation:Required
	Type string `json:"type"`

	// URL is the application URL.
	// +kubebuilder:validation:Required
	URL string `json:"url"`

	// APIKeySecretRef for the application.
	// If not specified, auto-discovery is attempted.
	// +optional
	APIKeySecretRef *SecretKeySelector `json:"apiKeySecretRef,omitempty"`

	// ConfigPath for API key auto-discovery.
	// +optional
	ConfigPath string `json:"configPath,omitempty"`

	// SyncCategories to sync (human-readable or numeric).
	// Defaults based on app type if not specified.
	// +optional
	SyncCategories []string `json:"syncCategories,omitempty"`

	// SyncLevel: disabled, addOnly, fullSync
	// +optional
	// +kubebuilder:validation:Enum=disabled;addOnly;fullSync
	// +kubebuilder:default=fullSync
	SyncLevel string `json:"syncLevel,omitempty"`
}

// ProwlarrConfigSpec defines the desired configuration for Prowlarr
type ProwlarrConfigSpec struct {
	// Connection specifies how to connect to Prowlarr.
	// Note: Prowlarr uses API v1, not v3.
	// +kubebuilder:validation:Required
	Connection ConnectionSpec `json:"connection"`

	// Indexers configures native indexers in Prowlarr.
	// +optional
	Indexers []ProwlarrIndexer `json:"indexers,omitempty"`

	// Proxies configures indexer proxies (e.g., FlareSolverr).
	// +optional
	Proxies []IndexerProxy `json:"proxies,omitempty"`

	// Applications configures sync to Radarr/Sonarr/Lidarr.
	// Usually not needed if those apps use prowlarrRef with autoRegister.
	// +optional
	Applications []ProwlarrApplication `json:"applications,omitempty"`

	// DownloadClients configures download clients in Prowlarr.
	// +optional
	DownloadClients []DownloadClientSpec `json:"downloadClients,omitempty"`

	// Reconciliation configures sync behavior.
	// +optional
	Reconciliation *ReconciliationSpec `json:"reconciliation,omitempty"`
}

// ProwlarrConfigStatus defines the observed state of ProwlarrConfig
type ProwlarrConfigStatus struct {
	// Conditions represent the latest observations of the ProwlarrConfig's state.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// Connected indicates whether Prowlarr is reachable.
	// +optional
	Connected bool `json:"connected,omitempty"`

	// ServiceVersion is the Prowlarr version.
	// +optional
	ServiceVersion string `json:"serviceVersion,omitempty"`

	// LastReconcile is the timestamp of the last reconciliation.
	// +optional
	LastReconcile *metav1.Time `json:"lastReconcile,omitempty"`

	// ManagedIndexers lists managed indexer IDs.
	// +optional
	ManagedIndexers []int `json:"managedIndexers,omitempty"`

	// ManagedProxies lists managed proxy IDs.
	// +optional
	ManagedProxies []int `json:"managedProxies,omitempty"`

	// ManagedApplications lists managed application IDs.
	// +optional
	ManagedApplications []int `json:"managedApplications,omitempty"`

	// LastAppliedHash is the hash of the last applied spec.
	// +optional
	LastAppliedHash string `json:"lastAppliedHash,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="URL",type=string,JSONPath=`.spec.connection.url`
// +kubebuilder:printcolumn:name="Indexers",type=integer,JSONPath=`.status.managedIndexers`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// ProwlarrConfig is the all-in-one configuration for a Prowlarr instance
type ProwlarrConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the desired configuration for Prowlarr.
	// +kubebuilder:validation:Required
	Spec ProwlarrConfigSpec `json:"spec"`

	// Status defines the observed state of ProwlarrConfig.
	// +optional
	Status ProwlarrConfigStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ProwlarrConfigList contains a list of ProwlarrConfig
type ProwlarrConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ProwlarrConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ProwlarrConfig{}, &ProwlarrConfigList{})
}
