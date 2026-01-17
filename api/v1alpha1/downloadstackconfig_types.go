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
// Gluetun Types
// =============================================================================

// GluetunSpec defines VPN configuration for Gluetun
type GluetunSpec struct {
	// Provider configuration
	// +kubebuilder:validation:Required
	Provider GluetunProviderSpec `json:"provider"`

	// VPNType: openvpn or wireguard
	// +kubebuilder:validation:Enum=openvpn;wireguard
	// +kubebuilder:default=openvpn
	VPNType string `json:"vpnType,omitempty"`

	// Server selection
	// +optional
	Server *GluetunServerSpec `json:"server,omitempty"`

	// Firewall settings
	// +optional
	Firewall *GluetunFirewallSpec `json:"firewall,omitempty"`

	// KillSwitch blocks traffic if VPN drops
	// +optional
	KillSwitch *GluetunKillSwitchSpec `json:"killSwitch,omitempty"`

	// DNS settings
	// +optional
	DNS *GluetunDNSSpec `json:"dns,omitempty"`

	// IPv6 settings
	// +optional
	IPv6 *GluetunIPv6Spec `json:"ipv6,omitempty"`

	// Logging settings
	// +optional
	Logging *GluetunLoggingSpec `json:"logging,omitempty"`
}

// GluetunProviderSpec defines the VPN provider configuration
type GluetunProviderSpec struct {
	// Name is the VPN provider: nordvpn, mullvad, expressvpn, pia, surfshark, etc.
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// CredentialsSecretRef for OpenVPN username/password authentication
	// +optional
	CredentialsSecretRef *CredentialsSecretRef `json:"credentialsSecretRef,omitempty"`

	// PrivateKeySecretRef for WireGuard private key
	// +optional
	PrivateKeySecretRef *SecretKeySelector `json:"privateKeySecretRef,omitempty"`
}

// GluetunServerSpec defines server selection options
type GluetunServerSpec struct {
	// Regions to connect to (e.g., ["Netherlands", "Germany"])
	// +optional
	Regions []string `json:"regions,omitempty"`

	// Countries to connect to
	// +optional
	Countries []string `json:"countries,omitempty"`

	// Cities to connect to
	// +optional
	Cities []string `json:"cities,omitempty"`

	// Hostnames of specific servers
	// +optional
	Hostnames []string `json:"hostnames,omitempty"`
}

// GluetunFirewallSpec defines firewall settings
type GluetunFirewallSpec struct {
	// VPNInputPorts are ports to allow inbound on VPN interface
	// +optional
	VPNInputPorts []int `json:"vpnInputPorts,omitempty"`

	// InputPorts are ports to allow inbound on all interfaces
	// +optional
	InputPorts []int `json:"inputPorts,omitempty"`

	// OutboundSubnets are subnets to allow outbound (local network access)
	// +optional
	OutboundSubnets []string `json:"outboundSubnets,omitempty"`

	// Debug enables firewall debug logging
	// +optional
	Debug bool `json:"debug,omitempty"`
}

// GluetunKillSwitchSpec defines kill switch settings
type GluetunKillSwitchSpec struct {
	// Enabled blocks traffic if VPN connection drops
	// +kubebuilder:default=true
	Enabled bool `json:"enabled,omitempty"`
}

// GluetunDNSSpec defines DNS settings
type GluetunDNSSpec struct {
	// OverTLS enables DNS over TLS (DoT)
	// +optional
	OverTLS bool `json:"overTls,omitempty"`

	// PlaintextAddress is the plaintext DNS server
	// +kubebuilder:default="1.1.1.1"
	PlaintextAddress string `json:"plaintextAddress,omitempty"`

	// KeepNameserver keeps the existing nameserver
	// +optional
	KeepNameserver bool `json:"keepNameserver,omitempty"`
}

// GluetunIPv6Spec defines IPv6 settings
type GluetunIPv6Spec struct {
	// Enabled enables IPv6 (usually disabled for VPN)
	// +kubebuilder:default=false
	Enabled bool `json:"enabled,omitempty"`
}

// GluetunLoggingSpec defines logging settings
type GluetunLoggingSpec struct {
	// Level: debug, info, warning, error
	// +kubebuilder:validation:Enum=debug;info;warning;error
	// +kubebuilder:default=info
	Level string `json:"level,omitempty"`
}

// =============================================================================
// Transmission Types
// =============================================================================

// TransmissionSpec defines Transmission torrent client configuration
type TransmissionSpec struct {
	// Connection settings
	// +kubebuilder:validation:Required
	Connection TransmissionConnectionSpec `json:"connection"`

	// Speed limits
	// +optional
	Speed *TransmissionSpeedSpec `json:"speed,omitempty"`

	// AltSpeed (turtle mode / scheduled limits)
	// +optional
	AltSpeed *TransmissionAltSpeedSpec `json:"altSpeed,omitempty"`

	// Directories configuration
	// +optional
	Directories *TransmissionDirectoriesSpec `json:"directories,omitempty"`

	// Seeding limits
	// +optional
	Seeding *TransmissionSeedingSpec `json:"seeding,omitempty"`

	// Queue settings
	// +optional
	Queue *TransmissionQueueSpec `json:"queue,omitempty"`

	// Peers settings
	// +optional
	Peers *TransmissionPeersSpec `json:"peers,omitempty"`

	// Security/protocol settings
	// +optional
	Security *TransmissionSecuritySpec `json:"security,omitempty"`

	// Blocklist settings
	// +optional
	Blocklist *TransmissionBlocklistSpec `json:"blocklist,omitempty"`
}

// TransmissionConnectionSpec defines how to connect to Transmission
type TransmissionConnectionSpec struct {
	// URL to Transmission RPC (e.g., http://localhost:9091)
	// +kubebuilder:validation:Required
	// +kubebuilder:default="http://localhost:9091"
	URL string `json:"url"`

	// CredentialsSecretRef for authentication (optional if no auth)
	// +optional
	CredentialsSecretRef *CredentialsSecretRef `json:"credentialsSecretRef,omitempty"`
}

// TransmissionSpeedSpec defines speed limit settings
type TransmissionSpeedSpec struct {
	// DownloadLimit in KB/s (0 = unlimited)
	// +optional
	DownloadLimit int `json:"downloadLimit,omitempty"`

	// DownloadLimitEnabled enables download limit
	// +optional
	DownloadLimitEnabled bool `json:"downloadLimitEnabled,omitempty"`

	// UploadLimit in KB/s (0 = unlimited)
	// +optional
	UploadLimit int `json:"uploadLimit,omitempty"`

	// UploadLimitEnabled enables upload limit
	// +optional
	UploadLimitEnabled bool `json:"uploadLimitEnabled,omitempty"`
}

// TransmissionAltSpeedSpec defines alt-speed (turtle mode) settings
type TransmissionAltSpeedSpec struct {
	// Enabled enables alt-speed mode
	// +optional
	Enabled bool `json:"enabled,omitempty"`

	// Down is the alt-speed download limit in KB/s
	// +optional
	Down int `json:"down,omitempty"`

	// Up is the alt-speed upload limit in KB/s
	// +optional
	Up int `json:"up,omitempty"`

	// TimeEnabled enables scheduled alt-speed
	// +optional
	TimeEnabled bool `json:"timeEnabled,omitempty"`

	// TimeBegin is minutes from midnight for schedule start
	// +optional
	TimeBegin int `json:"timeBegin,omitempty"`

	// TimeEnd is minutes from midnight for schedule end
	// +optional
	TimeEnd int `json:"timeEnd,omitempty"`

	// TimeDays are days to enable alt-speed (1=Mon, 7=Sun)
	// +optional
	TimeDays []int `json:"timeDays,omitempty"`
}

// TransmissionDirectoriesSpec defines directory settings
type TransmissionDirectoriesSpec struct {
	// Download is the completed downloads directory
	// +optional
	Download string `json:"download,omitempty"`

	// Incomplete is the incomplete downloads directory
	// +optional
	Incomplete string `json:"incomplete,omitempty"`

	// IncompleteEnabled enables incomplete directory
	// +optional
	IncompleteEnabled bool `json:"incompleteEnabled,omitempty"`
}

// TransmissionSeedingSpec defines seeding limit settings
type TransmissionSeedingSpec struct {
	// RatioLimit is the seed ratio to stop at
	// +optional
	RatioLimit string `json:"ratioLimit,omitempty"`

	// RatioLimited enables ratio limit
	// +optional
	RatioLimited bool `json:"ratioLimited,omitempty"`

	// IdleLimit is minutes of idle before stopping
	// +optional
	IdleLimit int `json:"idleLimit,omitempty"`

	// IdleLimitEnabled enables idle limit
	// +optional
	IdleLimitEnabled bool `json:"idleLimitEnabled,omitempty"`
}

// TransmissionQueueSpec defines queue settings
type TransmissionQueueSpec struct {
	// DownloadSize is max concurrent downloads
	// +optional
	DownloadSize int `json:"downloadSize,omitempty"`

	// DownloadEnabled enables download queue
	// +optional
	DownloadEnabled bool `json:"downloadEnabled,omitempty"`

	// SeedSize is max concurrent seeds
	// +optional
	SeedSize int `json:"seedSize,omitempty"`

	// SeedEnabled enables seed queue
	// +optional
	SeedEnabled bool `json:"seedEnabled,omitempty"`

	// StalledEnabled enables stalled torrent handling
	// +optional
	StalledEnabled bool `json:"stalledEnabled,omitempty"`

	// StalledMinutes is time before a torrent is considered stalled
	// +optional
	StalledMinutes int `json:"stalledMinutes,omitempty"`
}

// TransmissionPeersSpec defines peer settings
type TransmissionPeersSpec struct {
	// LimitGlobal is the global peer limit
	// +optional
	LimitGlobal int `json:"limitGlobal,omitempty"`

	// LimitPerTorrent is the per-torrent peer limit
	// +optional
	LimitPerTorrent int `json:"limitPerTorrent,omitempty"`

	// Port is the peer port
	// +optional
	Port int `json:"port,omitempty"`

	// RandomPort enables random port selection
	// +optional
	RandomPort bool `json:"randomPort,omitempty"`

	// PortForwardingEnabled enables port forwarding
	// +optional
	PortForwardingEnabled bool `json:"portForwardingEnabled,omitempty"`
}

// TransmissionSecuritySpec defines security/protocol settings
type TransmissionSecuritySpec struct {
	// Encryption: required, preferred, tolerated
	// +kubebuilder:validation:Enum=required;preferred;tolerated
	// +kubebuilder:default=preferred
	Encryption string `json:"encryption,omitempty"`

	// PEXEnabled enables Peer Exchange
	// +optional
	PEXEnabled *bool `json:"pexEnabled,omitempty"`

	// DHTEnabled enables Distributed Hash Table
	// +optional
	DHTEnabled *bool `json:"dhtEnabled,omitempty"`

	// LPDEnabled enables Local Peer Discovery
	// +optional
	LPDEnabled *bool `json:"lpdEnabled,omitempty"`

	// UTPEnabled enables Micro Transport Protocol
	// +optional
	UTPEnabled *bool `json:"utpEnabled,omitempty"`
}

// TransmissionBlocklistSpec defines blocklist settings
type TransmissionBlocklistSpec struct {
	// Enabled enables blocklist
	// +optional
	Enabled bool `json:"enabled,omitempty"`

	// URL is the blocklist URL
	// +optional
	URL string `json:"url,omitempty"`
}

// =============================================================================
// DownloadStackConfig
// =============================================================================

// DownloadStackConfigSpec defines the desired configuration for the download stack
type DownloadStackConfigSpec struct {
	// DeploymentRef references the Deployment to manage
	// +kubebuilder:validation:Required
	DeploymentRef LocalObjectReference `json:"deploymentRef"`

	// Gluetun VPN configuration (generates env Secret)
	// +kubebuilder:validation:Required
	Gluetun GluetunSpec `json:"gluetun"`

	// Transmission configuration (applied via RPC)
	// +kubebuilder:validation:Required
	Transmission TransmissionSpec `json:"transmission"`

	// RestartOnGluetunChange triggers Deployment restart when Gluetun config changes
	// +kubebuilder:default=true
	RestartOnGluetunChange bool `json:"restartOnGluetunChange,omitempty"`

	// Reconciliation configures sync behavior
	// +optional
	Reconciliation *ReconciliationSpec `json:"reconciliation,omitempty"`
}

// DownloadStackConfigStatus defines the observed state of DownloadStackConfig
type DownloadStackConfigStatus struct {
	// Conditions represent the latest observations
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// GluetunConfigHash is the hash of the generated Gluetun config
	// +optional
	GluetunConfigHash string `json:"gluetunConfigHash,omitempty"`

	// GluetunSecretGenerated indicates if the Gluetun env Secret was created
	// +optional
	GluetunSecretGenerated bool `json:"gluetunSecretGenerated,omitempty"`

	// TransmissionConnected indicates if Transmission RPC is reachable
	// +optional
	TransmissionConnected bool `json:"transmissionConnected,omitempty"`

	// TransmissionVersion is the Transmission version
	// +optional
	TransmissionVersion string `json:"transmissionVersion,omitempty"`

	// LastReconcile is the timestamp of the last reconciliation
	// +optional
	LastReconcile *metav1.Time `json:"lastReconcile,omitempty"`

	// ObservedGeneration is the last observed generation
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Deployment",type=string,JSONPath=`.spec.deploymentRef.name`
// +kubebuilder:printcolumn:name="VPN",type=string,JSONPath=`.spec.gluetun.provider.name`
// +kubebuilder:printcolumn:name="Transmission",type=string,JSONPath=`.status.transmissionConnected`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// DownloadStackConfig manages Gluetun VPN and Transmission configuration
type DownloadStackConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the desired configuration
	// +kubebuilder:validation:Required
	Spec DownloadStackConfigSpec `json:"spec"`

	// Status defines the observed state
	// +optional
	Status DownloadStackConfigStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// DownloadStackConfigList contains a list of DownloadStackConfig
type DownloadStackConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DownloadStackConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DownloadStackConfig{}, &DownloadStackConfigList{})
}
