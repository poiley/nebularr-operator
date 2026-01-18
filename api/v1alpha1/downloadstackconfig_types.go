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
// qBittorrent Types
// =============================================================================

// QBittorrentSpec defines qBittorrent torrent client configuration
type QBittorrentSpec struct {
	// Connection settings
	// +kubebuilder:validation:Required
	Connection QBittorrentConnectionSpec `json:"connection"`

	// Speed limits
	// +optional
	Speed *QBittorrentSpeedSpec `json:"speed,omitempty"`

	// AltSpeed (scheduled limits)
	// +optional
	AltSpeed *QBittorrentAltSpeedSpec `json:"altSpeed,omitempty"`

	// Directories configuration
	// +optional
	Directories *QBittorrentDirectoriesSpec `json:"directories,omitempty"`

	// Seeding limits
	// +optional
	Seeding *QBittorrentSeedingSpec `json:"seeding,omitempty"`

	// Queue settings
	// +optional
	Queue *QBittorrentQueueSpec `json:"queue,omitempty"`

	// Connection settings (peers, etc.)
	// +optional
	Connections *QBittorrentConnectionsSpec `json:"connections,omitempty"`

	// BitTorrent protocol settings
	// +optional
	BitTorrent *QBittorrentBitTorrentSpec `json:"bittorrent,omitempty"`
}

// QBittorrentConnectionSpec defines how to connect to qBittorrent
type QBittorrentConnectionSpec struct {
	// URL to qBittorrent WebUI (e.g., http://localhost:8080)
	// +kubebuilder:validation:Required
	// +kubebuilder:default="http://localhost:8080"
	URL string `json:"url"`

	// CredentialsSecretRef for authentication
	// +optional
	CredentialsSecretRef *CredentialsSecretRef `json:"credentialsSecretRef,omitempty"`
}

// QBittorrentSpeedSpec defines speed limit settings
type QBittorrentSpeedSpec struct {
	// DownloadLimit in KiB/s (0 = unlimited)
	// +optional
	DownloadLimit int `json:"downloadLimit,omitempty"`

	// UploadLimit in KiB/s (0 = unlimited)
	// +optional
	UploadLimit int `json:"uploadLimit,omitempty"`

	// GlobalDownloadSpeedLimit in KiB/s (0 = unlimited)
	// +optional
	GlobalDownloadSpeedLimit int `json:"globalDownloadSpeedLimit,omitempty"`

	// GlobalUploadSpeedLimit in KiB/s (0 = unlimited)
	// +optional
	GlobalUploadSpeedLimit int `json:"globalUploadSpeedLimit,omitempty"`
}

// QBittorrentAltSpeedSpec defines alternate speed (scheduled) settings
type QBittorrentAltSpeedSpec struct {
	// Enabled enables alt-speed limits
	// +optional
	Enabled bool `json:"enabled,omitempty"`

	// DownloadLimit in KiB/s
	// +optional
	DownloadLimit int `json:"downloadLimit,omitempty"`

	// UploadLimit in KiB/s
	// +optional
	UploadLimit int `json:"uploadLimit,omitempty"`

	// SchedulerEnabled enables scheduled alt-speed
	// +optional
	SchedulerEnabled bool `json:"schedulerEnabled,omitempty"`

	// SchedulerDays is a bitmask (1=Mon, 2=Tue, 4=Wed, 8=Thu, 16=Fri, 32=Sat, 64=Sun, 127=All)
	// +optional
	SchedulerDays int `json:"schedulerDays,omitempty"`

	// ScheduleFromHour is the start hour (0-23)
	// +optional
	ScheduleFromHour int `json:"scheduleFromHour,omitempty"`

	// ScheduleFromMinute is the start minute (0-59)
	// +optional
	ScheduleFromMinute int `json:"scheduleFromMinute,omitempty"`

	// ScheduleToHour is the end hour (0-23)
	// +optional
	ScheduleToHour int `json:"scheduleToHour,omitempty"`

	// ScheduleToMinute is the end minute (0-59)
	// +optional
	ScheduleToMinute int `json:"scheduleToMinute,omitempty"`
}

// QBittorrentDirectoriesSpec defines directory settings
type QBittorrentDirectoriesSpec struct {
	// SavePath is the default save path for downloads
	// +optional
	SavePath string `json:"savePath,omitempty"`

	// TempPath is the temporary download path
	// +optional
	TempPath string `json:"tempPath,omitempty"`

	// TempPathEnabled enables use of temporary path
	// +optional
	TempPathEnabled bool `json:"tempPathEnabled,omitempty"`

	// CreateSubfolder creates subfolder for multi-file torrents
	// +optional
	CreateSubfolder *bool `json:"createSubfolder,omitempty"`

	// AppendExtension adds .!qB extension to incomplete files
	// +optional
	AppendExtension *bool `json:"appendExtension,omitempty"`
}

// QBittorrentSeedingSpec defines seeding limit settings
type QBittorrentSeedingSpec struct {
	// MaxRatio is the max seeding ratio (e.g., 2.0)
	// +optional
	MaxRatio string `json:"maxRatio,omitempty"`

	// MaxRatioEnabled enables ratio limit
	// +optional
	MaxRatioEnabled bool `json:"maxRatioEnabled,omitempty"`

	// MaxSeedingTime is max seeding time in minutes
	// +optional
	MaxSeedingTime int `json:"maxSeedingTime,omitempty"`

	// MaxSeedingTimeEnabled enables time limit
	// +optional
	MaxSeedingTimeEnabled bool `json:"maxSeedingTimeEnabled,omitempty"`

	// MaxRatioAction: pause (0), remove (1), remove_and_delete (3), enable_super_seeding (2)
	// +optional
	// +kubebuilder:validation:Enum=0;1;2;3
	MaxRatioAction *int `json:"maxRatioAction,omitempty"`
}

// QBittorrentQueueSpec defines queue settings
type QBittorrentQueueSpec struct {
	// QueueingEnabled enables download queueing
	// +optional
	QueueingEnabled *bool `json:"queueingEnabled,omitempty"`

	// MaxActiveDownloads is the max concurrent downloads
	// +optional
	MaxActiveDownloads int `json:"maxActiveDownloads,omitempty"`

	// MaxActiveUploads is the max concurrent uploads
	// +optional
	MaxActiveUploads int `json:"maxActiveUploads,omitempty"`

	// MaxActiveTorrents is the max total active torrents
	// +optional
	MaxActiveTorrents int `json:"maxActiveTorrents,omitempty"`
}

// QBittorrentConnectionsSpec defines connection/peer settings
type QBittorrentConnectionsSpec struct {
	// MaxConnections is the global max connections
	// +optional
	MaxConnections int `json:"maxConnections,omitempty"`

	// MaxConnectionsPerTorrent is the per-torrent max connections
	// +optional
	MaxConnectionsPerTorrent int `json:"maxConnectionsPerTorrent,omitempty"`

	// MaxUploads is the global max upload slots
	// +optional
	MaxUploads int `json:"maxUploads,omitempty"`

	// MaxUploadsPerTorrent is the per-torrent max upload slots
	// +optional
	MaxUploadsPerTorrent int `json:"maxUploadsPerTorrent,omitempty"`

	// ListenPort is the listening port for incoming connections
	// +optional
	ListenPort int `json:"listenPort,omitempty"`

	// RandomPort uses random port on startup
	// +optional
	RandomPort bool `json:"randomPort,omitempty"`

	// UPnPEnabled enables UPnP/NAT-PMP port forwarding
	// +optional
	UPnPEnabled *bool `json:"upnpEnabled,omitempty"`
}

// QBittorrentBitTorrentSpec defines BitTorrent protocol settings
type QBittorrentBitTorrentSpec struct {
	// DHT enables Distributed Hash Table
	// +optional
	DHT *bool `json:"dht,omitempty"`

	// PeX enables Peer Exchange
	// +optional
	PeX *bool `json:"pex,omitempty"`

	// LSD enables Local Service Discovery
	// +optional
	LSD *bool `json:"lsd,omitempty"`

	// Encryption: 0=prefer, 1=force_on, 2=force_off
	// +optional
	// +kubebuilder:validation:Enum=0;1;2
	Encryption *int `json:"encryption,omitempty"`

	// AnonymousMode hides client identity
	// +optional
	AnonymousMode bool `json:"anonymousMode,omitempty"`
}

// =============================================================================
// Deluge Types
// =============================================================================

// DelugeSpec defines Deluge torrent client configuration
type DelugeSpec struct {
	// Connection settings
	// +kubebuilder:validation:Required
	Connection DelugeConnectionSpec `json:"connection"`

	// Speed limits
	// +optional
	Speed *DelugeSpeedSpec `json:"speed,omitempty"`

	// Directories configuration
	// +optional
	Directories *DelugeDirectoriesSpec `json:"directories,omitempty"`

	// Seeding limits
	// +optional
	Seeding *DelugeSeedingSpec `json:"seeding,omitempty"`

	// Queue settings
	// +optional
	Queue *DelugeQueueSpec `json:"queue,omitempty"`

	// Connection settings (peers, etc.)
	// +optional
	Connections *DelugeConnectionsSpec `json:"connections,omitempty"`

	// Protocol settings (DHT, encryption, etc.)
	// +optional
	Protocol *DelugeProtocolSpec `json:"protocol,omitempty"`
}

// DelugeConnectionSpec defines how to connect to Deluge
type DelugeConnectionSpec struct {
	// URL to Deluge Web UI (e.g., http://localhost:8112)
	// +kubebuilder:validation:Required
	// +kubebuilder:default="http://localhost:8112"
	URL string `json:"url"`

	// PasswordSecretRef references the password Secret for Deluge Web UI.
	// Deluge Web UI uses a single password for authentication (default: "deluge").
	// +optional
	PasswordSecretRef *SecretKeySelector `json:"passwordSecretRef,omitempty"`
}

// DelugeSpeedSpec defines speed limit settings
type DelugeSpeedSpec struct {
	// MaxDownloadSpeed in KiB/s (-1 = unlimited)
	// +optional
	// +kubebuilder:default=-1
	MaxDownloadSpeed int `json:"maxDownloadSpeed,omitempty"`

	// MaxUploadSpeed in KiB/s (-1 = unlimited)
	// +optional
	// +kubebuilder:default=-1
	MaxUploadSpeed int `json:"maxUploadSpeed,omitempty"`

	// MaxDownloadSpeedPerTorrent in KiB/s (-1 = unlimited)
	// +optional
	// +kubebuilder:default=-1
	MaxDownloadSpeedPerTorrent int `json:"maxDownloadSpeedPerTorrent,omitempty"`

	// MaxUploadSpeedPerTorrent in KiB/s (-1 = unlimited)
	// +optional
	// +kubebuilder:default=-1
	MaxUploadSpeedPerTorrent int `json:"maxUploadSpeedPerTorrent,omitempty"`
}

// DelugeDirectoriesSpec defines directory settings
type DelugeDirectoriesSpec struct {
	// DownloadLocation is the default download directory
	// +optional
	DownloadLocation string `json:"downloadLocation,omitempty"`

	// MoveCompleted enables moving completed downloads
	// +optional
	MoveCompleted bool `json:"moveCompleted,omitempty"`

	// MoveCompletedPath is the path to move completed downloads to
	// +optional
	MoveCompletedPath string `json:"moveCompletedPath,omitempty"`

	// CopyTorrentFile copies .torrent files to a location
	// +optional
	CopyTorrentFile bool `json:"copyTorrentFile,omitempty"`

	// TorrentFilesLocation is where to copy .torrent files
	// +optional
	TorrentFilesLocation string `json:"torrentFilesLocation,omitempty"`
}

// DelugeSeedingSpec defines seeding limit settings
type DelugeSeedingSpec struct {
	// StopSeedAtRatio enables stopping seeding at a ratio
	// +optional
	StopSeedAtRatio bool `json:"stopSeedAtRatio,omitempty"`

	// StopSeedRatio is the ratio to stop seeding at (e.g., 2.0)
	// +optional
	StopSeedRatio string `json:"stopSeedRatio,omitempty"`

	// RemoveAtRatio removes the torrent when ratio is reached
	// +optional
	RemoveAtRatio bool `json:"removeAtRatio,omitempty"`

	// ShareRatioLimit is the share ratio limit
	// +optional
	ShareRatioLimit string `json:"shareRatioLimit,omitempty"`

	// SeedTimeLimit is the max seeding time in seconds (-1 = unlimited)
	// +optional
	SeedTimeLimit int `json:"seedTimeLimit,omitempty"`
}

// DelugeQueueSpec defines queue settings
type DelugeQueueSpec struct {
	// MaxActiveDownloading is the max concurrent downloads
	// +optional
	MaxActiveDownloading int `json:"maxActiveDownloading,omitempty"`

	// MaxActiveSeeding is the max concurrent seeding torrents
	// +optional
	MaxActiveSeeding int `json:"maxActiveSeeding,omitempty"`

	// MaxActiveLimit is the total max active torrents
	// +optional
	MaxActiveLimit int `json:"maxActiveLimit,omitempty"`

	// QueueNewToTop adds new torrents to the top of the queue
	// +optional
	QueueNewToTop bool `json:"queueNewToTop,omitempty"`
}

// DelugeConnectionsSpec defines connection/peer settings
type DelugeConnectionsSpec struct {
	// MaxConnections is the global max connections
	// +optional
	MaxConnections int `json:"maxConnections,omitempty"`

	// MaxConnectionsPerTorrent is the per-torrent max connections
	// +optional
	MaxConnectionsPerTorrent int `json:"maxConnectionsPerTorrent,omitempty"`

	// MaxUploadSlots is the global max upload slots
	// +optional
	MaxUploadSlots int `json:"maxUploadSlots,omitempty"`

	// MaxUploadSlotsPerTorrent is the per-torrent max upload slots
	// +optional
	MaxUploadSlotsPerTorrent int `json:"maxUploadSlotsPerTorrent,omitempty"`

	// ListenPorts is the range of ports to listen on [start, end]
	// +optional
	ListenPorts []int `json:"listenPorts,omitempty"`

	// RandomPort enables random port selection
	// +optional
	RandomPort bool `json:"randomPort,omitempty"`
}

// DelugeProtocolSpec defines protocol settings
type DelugeProtocolSpec struct {
	// DHT enables Distributed Hash Table
	// +optional
	DHT *bool `json:"dht,omitempty"`

	// UPnP enables UPnP port forwarding
	// +optional
	UPnP *bool `json:"upnp,omitempty"`

	// NATPMP enables NAT-PMP port forwarding
	// +optional
	NATPMP *bool `json:"natpmp,omitempty"`

	// LSD enables Local Service Discovery
	// +optional
	LSD *bool `json:"lsd,omitempty"`

	// ProtocolEncryption enables protocol encryption
	// +optional
	ProtocolEncryption *bool `json:"protocolEncryption,omitempty"`

	// EncryptionLevel: 0=handshake, 1=full, 2=either
	// +optional
	// +kubebuilder:validation:Enum=0;1;2
	EncryptionLevel *int `json:"encryptionLevel,omitempty"`
}

// =============================================================================
// rTorrent Types
// =============================================================================

// RTorrentSpec defines rTorrent client configuration
type RTorrentSpec struct {
	// Connection settings
	// +kubebuilder:validation:Required
	Connection RTorrentConnectionSpec `json:"connection"`

	// Speed limits
	// +optional
	Speed *RTorrentSpeedSpec `json:"speed,omitempty"`

	// Directories configuration
	// +optional
	Directories *RTorrentDirectoriesSpec `json:"directories,omitempty"`

	// Seeding limits
	// +optional
	Seeding *RTorrentSeedingSpec `json:"seeding,omitempty"`

	// Connection settings (peers, etc.)
	// +optional
	Connections *RTorrentConnectionsSpec `json:"connections,omitempty"`

	// Protocol settings
	// +optional
	Protocol *RTorrentProtocolSpec `json:"protocol,omitempty"`
}

// RTorrentConnectionSpec defines how to connect to rTorrent
type RTorrentConnectionSpec struct {
	// URL to rTorrent XML-RPC interface (e.g., http://localhost:8080/RPC2)
	// Can also be a Unix socket path (e.g., /path/to/.local/share/rtorrent/rtorrent.sock)
	// +kubebuilder:validation:Required
	URL string `json:"url"`

	// CredentialsSecretRef for HTTP Basic authentication (if using a web server proxy)
	// +optional
	CredentialsSecretRef *CredentialsSecretRef `json:"credentialsSecretRef,omitempty"`
}

// RTorrentSpeedSpec defines speed limit settings
type RTorrentSpeedSpec struct {
	// DownloadRate in KiB/s (0 = unlimited)
	// +optional
	DownloadRate int `json:"downloadRate,omitempty"`

	// UploadRate in KiB/s (0 = unlimited)
	// +optional
	UploadRate int `json:"uploadRate,omitempty"`
}

// RTorrentDirectoriesSpec defines directory settings
type RTorrentDirectoriesSpec struct {
	// Directory is the default download directory
	// +optional
	Directory string `json:"directory,omitempty"`

	// SessionDirectory is the session data directory
	// +optional
	SessionDirectory string `json:"sessionDirectory,omitempty"`
}

// RTorrentSeedingSpec defines seeding limit settings
type RTorrentSeedingSpec struct {
	// MinSeedRatio is the minimum ratio to maintain (-1 = disabled)
	// +optional
	MinSeedRatio string `json:"minSeedRatio,omitempty"`

	// MaxSeedRatio is the maximum ratio before stopping (-1 = disabled)
	// +optional
	MaxSeedRatio string `json:"maxSeedRatio,omitempty"`

	// MinSeedTime is minimum seeding time in seconds
	// +optional
	MinSeedTime int `json:"minSeedTime,omitempty"`

	// MaxSeedTime is maximum seeding time in seconds (-1 = disabled)
	// +optional
	MaxSeedTime int `json:"maxSeedTime,omitempty"`
}

// RTorrentConnectionsSpec defines connection/peer settings
type RTorrentConnectionsSpec struct {
	// MaxPeers is the global max peers
	// +optional
	MaxPeers int `json:"maxPeers,omitempty"`

	// MaxPeersPerTorrent is the per-torrent max peers
	// +optional
	MaxPeersPerTorrent int `json:"maxPeersPerTorrent,omitempty"`

	// MaxUploads is the global max upload slots
	// +optional
	MaxUploads int `json:"maxUploads,omitempty"`

	// MaxUploadsPerTorrent is the per-torrent max upload slots
	// +optional
	MaxUploadsPerTorrent int `json:"maxUploadsPerTorrent,omitempty"`

	// Port is the listening port (0 = random)
	// +optional
	Port int `json:"port,omitempty"`

	// PortRange is the port range (e.g., "6881-6889")
	// +optional
	PortRange string `json:"portRange,omitempty"`

	// PortRandomize randomizes the port within the range
	// +optional
	PortRandomize bool `json:"portRandomize,omitempty"`
}

// RTorrentProtocolSpec defines protocol settings
type RTorrentProtocolSpec struct {
	// DHT enables Distributed Hash Table
	// +optional
	DHT *bool `json:"dht,omitempty"`

	// PEX enables Peer Exchange
	// +optional
	PEX *bool `json:"pex,omitempty"`

	// Encryption mode: none, allow_incoming, try_outgoing, require, require_RC4, require_RC4_strong
	// +optional
	Encryption string `json:"encryption,omitempty"`
}

// =============================================================================
// SABnzbd Types
// =============================================================================

// SABnzbdSpec defines SABnzbd usenet client configuration
type SABnzbdSpec struct {
	// Connection settings
	// +kubebuilder:validation:Required
	Connection SABnzbdConnectionSpec `json:"connection"`

	// Speed limits
	// +optional
	Speed *SABnzbdSpeedSpec `json:"speed,omitempty"`

	// Directories configuration
	// +optional
	Directories *SABnzbdDirectoriesSpec `json:"directories,omitempty"`

	// Categories configuration
	// +optional
	Categories []SABnzbdCategorySpec `json:"categories,omitempty"`

	// Queue settings
	// +optional
	Queue *SABnzbdQueueSpec `json:"queue,omitempty"`

	// Post-processing settings
	// +optional
	PostProcessing *SABnzbdPostProcessingSpec `json:"postProcessing,omitempty"`
}

// SABnzbdConnectionSpec defines how to connect to SABnzbd
type SABnzbdConnectionSpec struct {
	// URL to SABnzbd API (e.g., http://localhost:8080)
	// +kubebuilder:validation:Required
	// +kubebuilder:default="http://localhost:8080"
	URL string `json:"url"`

	// APIKeySecretRef references the API key Secret for SABnzbd.
	// +kubebuilder:validation:Required
	APIKeySecretRef SecretKeySelector `json:"apiKeySecretRef"`
}

// SABnzbdSpeedSpec defines speed limit settings
type SABnzbdSpeedSpec struct {
	// SpeedLimit in KiB/s (0 = unlimited)
	// +optional
	SpeedLimit int `json:"speedLimit,omitempty"`

	// SpeedLimitPercentage is the percentage of bandwidth to use (0-100)
	// +optional
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=100
	SpeedLimitPercentage int `json:"speedLimitPercentage,omitempty"`

	// PauseDownloads pauses all downloads
	// +optional
	PauseDownloads bool `json:"pauseDownloads,omitempty"`
}

// SABnzbdDirectoriesSpec defines directory settings
type SABnzbdDirectoriesSpec struct {
	// DownloadDir is the temporary download directory
	// +optional
	DownloadDir string `json:"downloadDir,omitempty"`

	// CompleteDir is the completed downloads directory
	// +optional
	CompleteDir string `json:"completeDir,omitempty"`

	// IncompleteDir is the incomplete downloads directory
	// +optional
	IncompleteDir string `json:"incompleteDir,omitempty"`

	// ScriptDir is the post-processing scripts directory
	// +optional
	ScriptDir string `json:"scriptDir,omitempty"`

	// NzbBackupDir is the NZB backup directory
	// +optional
	NzbBackupDir string `json:"nzbBackupDir,omitempty"`
}

// SABnzbdCategorySpec defines a download category
type SABnzbdCategorySpec struct {
	// Name is the category name
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Dir is the directory for this category
	// +optional
	Dir string `json:"dir,omitempty"`

	// Priority: -100 (default), -2 (paused), -1 (low), 0 (normal), 1 (high), 2 (force)
	// +optional
	// +kubebuilder:validation:Enum=-100;-2;-1;0;1;2
	Priority int `json:"priority,omitempty"`

	// Script is the post-processing script for this category
	// +optional
	Script string `json:"script,omitempty"`
}

// SABnzbdQueueSpec defines queue settings
type SABnzbdQueueSpec struct {
	// PreCheck enables pre-download check
	// +optional
	PreCheck bool `json:"preCheck,omitempty"`

	// MaxRetries is the max number of retries per server
	// +optional
	MaxRetries int `json:"maxRetries,omitempty"`

	// Connections is the total number of connections
	// +optional
	Connections int `json:"connections,omitempty"`
}

// SABnzbdPostProcessingSpec defines post-processing settings
type SABnzbdPostProcessingSpec struct {
	// Enabled enables post-processing
	// +optional
	Enabled bool `json:"enabled,omitempty"`

	// QuickCheck enables quick verification
	// +optional
	QuickCheck bool `json:"quickCheck,omitempty"`

	// UnpackEnabled enables automatic unpacking
	// +optional
	UnpackEnabled bool `json:"unpackEnabled,omitempty"`

	// CleanupEnabled cleans up files after unpacking
	// +optional
	CleanupEnabled bool `json:"cleanupEnabled,omitempty"`

	// ScriptEnabled enables post-processing scripts
	// +optional
	ScriptEnabled bool `json:"scriptEnabled,omitempty"`
}

// =============================================================================
// NZBGet Types
// =============================================================================

// NZBGetSpec defines NZBGet usenet client configuration
type NZBGetSpec struct {
	// Connection settings
	// +kubebuilder:validation:Required
	Connection NZBGetConnectionSpec `json:"connection"`

	// Speed limits
	// +optional
	Speed *NZBGetSpeedSpec `json:"speed,omitempty"`

	// Directories configuration
	// +optional
	Directories *NZBGetDirectoriesSpec `json:"directories,omitempty"`

	// Categories configuration
	// +optional
	Categories []NZBGetCategorySpec `json:"categories,omitempty"`

	// Queue settings
	// +optional
	Queue *NZBGetQueueSpec `json:"queue,omitempty"`

	// Post-processing settings
	// +optional
	PostProcessing *NZBGetPostProcessingSpec `json:"postProcessing,omitempty"`

	// Connections settings
	// +optional
	Connections *NZBGetConnectionsSpec `json:"connections,omitempty"`
}

// NZBGetConnectionSpec defines how to connect to NZBGet
type NZBGetConnectionSpec struct {
	// URL to NZBGet JSON-RPC API (e.g., http://localhost:6789)
	// +kubebuilder:validation:Required
	// +kubebuilder:default="http://localhost:6789"
	URL string `json:"url"`

	// CredentialsSecretRef for authentication (username/password)
	// +optional
	CredentialsSecretRef *CredentialsSecretRef `json:"credentialsSecretRef,omitempty"`
}

// NZBGetSpeedSpec defines speed limit settings
type NZBGetSpeedSpec struct {
	// DownloadRate in KiB/s (0 = unlimited)
	// +optional
	DownloadRate int `json:"downloadRate,omitempty"`

	// ArticleTimeout is the timeout for fetching an article in seconds
	// +optional
	ArticleTimeout int `json:"articleTimeout,omitempty"`

	// WriteBuffer is the disk write buffer size in bytes
	// +optional
	WriteBuffer int `json:"writeBuffer,omitempty"`
}

// NZBGetDirectoriesSpec defines directory settings
type NZBGetDirectoriesSpec struct {
	// MainDir is the main working directory
	// +optional
	MainDir string `json:"mainDir,omitempty"`

	// DestDir is the destination directory for completed downloads
	// +optional
	DestDir string `json:"destDir,omitempty"`

	// InterDir is the intermediate directory during download
	// +optional
	InterDir string `json:"interDir,omitempty"`

	// NzbDir is the directory to monitor for NZB files
	// +optional
	NzbDir string `json:"nzbDir,omitempty"`

	// TempDir is the directory for temporary files
	// +optional
	TempDir string `json:"tempDir,omitempty"`

	// ScriptDir is the directory containing post-processing scripts
	// +optional
	ScriptDir string `json:"scriptDir,omitempty"`
}

// NZBGetCategorySpec defines a download category
type NZBGetCategorySpec struct {
	// Name is the category name
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// DestDir is the destination directory for this category
	// +optional
	DestDir string `json:"destDir,omitempty"`

	// Unpack enables unpacking for this category
	// +optional
	Unpack *bool `json:"unpack,omitempty"`

	// Aliases are alternative names for this category
	// +optional
	Aliases []string `json:"aliases,omitempty"`
}

// NZBGetQueueSpec defines queue settings
type NZBGetQueueSpec struct {
	// FlushQueue writes queue to disk immediately
	// +optional
	FlushQueue bool `json:"flushQueue,omitempty"`

	// DupeCheck enables duplicate checking
	// +optional
	DupeCheck bool `json:"dupeCheck,omitempty"`

	// PropagationDelay is the delay before downloading in seconds
	// +optional
	PropagationDelay int `json:"propagationDelay,omitempty"`

	// HealthCheck: none, park, delete, pause
	// +optional
	// +kubebuilder:validation:Enum=none;park;delete;pause
	HealthCheck string `json:"healthCheck,omitempty"`
}

// NZBGetPostProcessingSpec defines post-processing settings
type NZBGetPostProcessingSpec struct {
	// ParCheck: auto, always, force, manual
	// +optional
	// +kubebuilder:validation:Enum=auto;always;force;manual
	ParCheck string `json:"parCheck,omitempty"`

	// ParRepair enables automatic repair
	// +optional
	ParRepair *bool `json:"parRepair,omitempty"`

	// Unpack enables automatic unpacking
	// +optional
	Unpack *bool `json:"unpack,omitempty"`

	// UnpackCleanupDisk removes archive files after unpacking
	// +optional
	UnpackCleanupDisk *bool `json:"unpackCleanupDisk,omitempty"`

	// DirectUnpack enables unpacking while downloading
	// +optional
	DirectUnpack *bool `json:"directUnpack,omitempty"`

	// ScriptOrder is the order of post-processing scripts
	// +optional
	ScriptOrder []string `json:"scriptOrder,omitempty"`
}

// NZBGetConnectionsSpec defines connection/server settings
type NZBGetConnectionsSpec struct {
	// ArticleConnections is connections per news server
	// +optional
	ArticleConnections int `json:"articleConnections,omitempty"`

	// RetryInterval is seconds between retries
	// +optional
	RetryInterval int `json:"retryInterval,omitempty"`

	// TerminateTimeout is timeout for graceful termination in seconds
	// +optional
	TerminateTimeout int `json:"terminateTimeout,omitempty"`

	// Decode enables article decoding (should typically be enabled)
	// +optional
	Decode *bool `json:"decode,omitempty"`
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
	// At least one download client must be specified
	// +optional
	Transmission *TransmissionSpec `json:"transmission,omitempty"`

	// QBittorrent configuration (applied via WebUI API)
	// At least one download client must be specified
	// +optional
	QBittorrent *QBittorrentSpec `json:"qbittorrent,omitempty"`

	// Deluge configuration (applied via JSON-RPC API)
	// At least one download client must be specified
	// +optional
	Deluge *DelugeSpec `json:"deluge,omitempty"`

	// RTorrent configuration (applied via XML-RPC API)
	// At least one download client must be specified
	// +optional
	RTorrent *RTorrentSpec `json:"rtorrent,omitempty"`

	// SABnzbd configuration (applied via REST API)
	// Usenet download client
	// +optional
	SABnzbd *SABnzbdSpec `json:"sabnzbd,omitempty"`

	// NZBGet configuration (applied via JSON-RPC API)
	// Usenet download client
	// +optional
	NZBGet *NZBGetSpec `json:"nzbget,omitempty"`

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

	// QBittorrentConnected indicates if qBittorrent WebUI is reachable
	// +optional
	QBittorrentConnected bool `json:"qbittorrentConnected,omitempty"`

	// QBittorrentVersion is the qBittorrent version
	// +optional
	QBittorrentVersion string `json:"qbittorrentVersion,omitempty"`

	// DelugeConnected indicates if Deluge Web UI is reachable
	// +optional
	DelugeConnected bool `json:"delugeConnected,omitempty"`

	// DelugeVersion is the Deluge version
	// +optional
	DelugeVersion string `json:"delugeVersion,omitempty"`

	// RTorrentConnected indicates if rTorrent XML-RPC is reachable
	// +optional
	RTorrentConnected bool `json:"rtorrentConnected,omitempty"`

	// RTorrentVersion is the rTorrent version
	// +optional
	RTorrentVersion string `json:"rtorrentVersion,omitempty"`

	// SABnzbdConnected indicates if SABnzbd API is reachable
	// +optional
	SABnzbdConnected bool `json:"sabnzbdConnected,omitempty"`

	// SABnzbdVersion is the SABnzbd version
	// +optional
	SABnzbdVersion string `json:"sabnzbdVersion,omitempty"`

	// NZBGetConnected indicates if NZBGet JSON-RPC is reachable
	// +optional
	NZBGetConnected bool `json:"nzbgetConnected,omitempty"`

	// NZBGetVersion is the NZBGet version
	// +optional
	NZBGetVersion string `json:"nzbgetVersion,omitempty"`

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
