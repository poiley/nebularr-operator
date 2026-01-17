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
// Bazarr-specific Types
// =============================================================================

// BazarrLanguage defines a language for subtitle downloads
type BazarrLanguage struct {
	// Code is the ISO 639-1 language code (e.g., "en", "es", "fr").
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^[a-z]{2,3}$`
	Code string `json:"code"`

	// Forced indicates if forced subtitles should be used.
	// +optional
	Forced bool `json:"forced,omitempty"`

	// HearingImpaired indicates if hearing impaired subtitles should be used.
	// +optional
	HearingImpaired bool `json:"hearingImpaired,omitempty"`
}

// BazarrLanguageProfile defines a language profile for Bazarr
type BazarrLanguageProfile struct {
	// Name is the profile name.
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Languages is the list of languages in order of preference.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	Languages []BazarrLanguage `json:"languages"`

	// DefaultForSeries makes this the default profile for series.
	// +optional
	DefaultForSeries bool `json:"defaultForSeries,omitempty"`

	// DefaultForMovies makes this the default profile for movies.
	// +optional
	DefaultForMovies bool `json:"defaultForMovies,omitempty"`
}

// BazarrProvider defines a subtitle provider configuration
type BazarrProvider struct {
	// Name is the provider name (e.g., "opensubtitles", "subscene", "podnapisi").
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Username for providers requiring authentication.
	// +optional
	Username string `json:"username,omitempty"`

	// PasswordSecretRef references the password Secret.
	// +optional
	PasswordSecretRef *SecretKeySelector `json:"passwordSecretRef,omitempty"`

	// APIKeySecretRef for providers using API key authentication.
	// +optional
	APIKeySecretRef *SecretKeySelector `json:"apiKeySecretRef,omitempty"`
}

// BazarrConnectionSpec defines connection to Sonarr/Radarr for Bazarr
type BazarrConnectionSpec struct {
	// URL is the base URL (e.g., http://sonarr:8989).
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^https?://`
	URL string `json:"url"`

	// APIKeySecretRef references a Secret containing the API key.
	// If not specified, auto-discovery from ConfigPath is attempted.
	// +optional
	APIKeySecretRef *SecretKeySelector `json:"apiKeySecretRef,omitempty"`

	// ConfigPath is the path to config.xml for API key auto-discovery.
	// Defaults to /{app}-config/config.xml
	// +optional
	ConfigPath string `json:"configPath,omitempty"`
}

// BazarrConfigSpec defines the desired configuration for Bazarr
type BazarrConfigSpec struct {
	// Sonarr connection configuration.
	// +kubebuilder:validation:Required
	Sonarr BazarrConnectionSpec `json:"sonarr"`

	// Radarr connection configuration.
	// +kubebuilder:validation:Required
	Radarr BazarrConnectionSpec `json:"radarr"`

	// LanguageProfiles defines language profiles for subtitle downloads.
	// +optional
	LanguageProfiles []BazarrLanguageProfile `json:"languageProfiles,omitempty"`

	// Providers configures subtitle providers.
	// +optional
	Providers []BazarrProvider `json:"providers,omitempty"`

	// Authentication configures Bazarr authentication.
	// +optional
	Authentication *AuthenticationSpec `json:"authentication,omitempty"`

	// OutputPath is where to write Bazarr's config.yaml.
	// Used for init-container config generation.
	// +optional
	// +kubebuilder:default="/config/config/config.yaml"
	OutputPath string `json:"outputPath,omitempty"`

	// ConfigMapRef references a ConfigMap to store the generated config.
	// If specified, generates config to ConfigMap instead of file.
	// +optional
	ConfigMapRef *LocalObjectReference `json:"configMapRef,omitempty"`

	// Reconciliation configures sync behavior.
	// +optional
	Reconciliation *ReconciliationSpec `json:"reconciliation,omitempty"`
}

// BazarrConfigStatus defines the observed state of BazarrConfig
type BazarrConfigStatus struct {
	// Conditions represent the latest observations of the BazarrConfig's state.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// SonarrConnected indicates whether Sonarr is reachable.
	// +optional
	SonarrConnected bool `json:"sonarrConnected,omitempty"`

	// RadarrConnected indicates whether Radarr is reachable.
	// +optional
	RadarrConnected bool `json:"radarrConnected,omitempty"`

	// ConfigGenerated indicates if the config.yaml was generated.
	// +optional
	ConfigGenerated bool `json:"configGenerated,omitempty"`

	// LastReconcile is the timestamp of the last reconciliation.
	// +optional
	LastReconcile *metav1.Time `json:"lastReconcile,omitempty"`

	// LastAppliedHash is the hash of the last applied spec.
	// +optional
	LastAppliedHash string `json:"lastAppliedHash,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Sonarr",type=string,JSONPath=`.spec.sonarr.url`
// +kubebuilder:printcolumn:name="Radarr",type=string,JSONPath=`.spec.radarr.url`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// BazarrConfig is the configuration for Bazarr subtitle management
type BazarrConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the desired configuration for Bazarr.
	// +kubebuilder:validation:Required
	Spec BazarrConfigSpec `json:"spec"`

	// Status defines the observed state of BazarrConfig.
	// +optional
	Status BazarrConfigStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// BazarrConfigList contains a list of BazarrConfig
type BazarrConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BazarrConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&BazarrConfig{}, &BazarrConfigList{})
}
