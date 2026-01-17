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

// RadarrConfigSpec defines the desired configuration for Radarr
type RadarrConfigSpec struct {
	// Connection specifies how to connect to Radarr.
	// +kubebuilder:validation:Required
	Connection ConnectionSpec `json:"connection"`

	// Quality defines movie quality preferences.
	// Defaults to "balanced" preset if not specified.
	// +optional
	Quality *VideoQualitySpec `json:"quality,omitempty"`

	// DownloadClients configures download clients.
	// +optional
	DownloadClients []DownloadClientSpec `json:"downloadClients,omitempty"`

	// Indexers configures indexer sources.
	// +optional
	Indexers *IndexersSpec `json:"indexers,omitempty"`

	// Naming configures file/folder naming.
	// Defaults to "plex-friendly" preset if not specified.
	// +optional
	Naming *NamingSpec `json:"naming,omitempty"`

	// RootFolders configures root folder paths.
	// +optional
	RootFolders []string `json:"rootFolders,omitempty"`

	// ImportLists configures automatic import lists (IMDb, Trakt, Plex, etc.).
	// +optional
	ImportLists []ImportListSpec `json:"importLists,omitempty"`

	// MediaManagement configures media management settings.
	// +optional
	MediaManagement *MediaManagementSpec `json:"mediaManagement,omitempty"`

	// Authentication configures authentication settings.
	// +optional
	Authentication *AuthenticationSpec `json:"authentication,omitempty"`

	// Reconciliation configures sync behavior.
	// +optional
	Reconciliation *ReconciliationSpec `json:"reconciliation,omitempty"`
}

// RadarrConfigStatus defines the observed state of RadarrConfig
type RadarrConfigStatus struct {
	// Conditions represent the latest observations of the RadarrConfig's state.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// Connected indicates whether Radarr is reachable.
	// +optional
	Connected bool `json:"connected,omitempty"`

	// ServiceVersion is the Radarr version.
	// +optional
	ServiceVersion string `json:"serviceVersion,omitempty"`

	// LastReconcile is the timestamp of the last reconciliation.
	// +optional
	LastReconcile *metav1.Time `json:"lastReconcile,omitempty"`

	// ManagedResources lists resources created by this config.
	// +optional
	ManagedResources ManagedResources `json:"managedResources,omitempty"`

	// LastAppliedHash is the hash of the last applied spec.
	// Used for drift detection.
	// +optional
	LastAppliedHash string `json:"lastAppliedHash,omitempty"`

	// ProwlarrRegistration tracks registration with Prowlarr (Pull Model).
	// +optional
	ProwlarrRegistration *ProwlarrRegistration `json:"prowlarrRegistration,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="URL",type=string,JSONPath=`.spec.connection.url`
// +kubebuilder:printcolumn:name="Quality",type=string,JSONPath=`.spec.quality.preset`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// RadarrConfig is the all-in-one configuration for a Radarr instance
type RadarrConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the desired configuration for Radarr.
	// +kubebuilder:validation:Required
	Spec RadarrConfigSpec `json:"spec"`

	// Status defines the observed state of RadarrConfig.
	// +optional
	Status RadarrConfigStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// RadarrConfigList contains a list of RadarrConfig
type RadarrConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RadarrConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RadarrConfig{}, &RadarrConfigList{})
}
