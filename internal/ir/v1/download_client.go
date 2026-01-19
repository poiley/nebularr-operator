package v1

// DownloadClientIR represents a download client configuration
type DownloadClientIR struct {
	// ID is the service-side ID (populated from CurrentState, used for updates/deletes)
	ID int `json:"id,omitempty"`

	// Name is the client name (generated: "nebularr-{name}")
	Name string `json:"name"`

	// Protocol is "torrent" or "usenet"
	Protocol string `json:"protocol"`

	// Implementation is the client type (qbittorrent, transmission, etc.)
	Implementation string `json:"implementation"`

	// Enable toggles the client
	Enable bool `json:"enable"`

	// Priority affects selection order (higher = preferred)
	Priority int `json:"priority"`

	// RemoveCompletedDownloads after import
	RemoveCompletedDownloads bool `json:"removeCompletedDownloads,omitempty"`

	// RemoveFailedDownloads on failure
	RemoveFailedDownloads bool `json:"removeFailedDownloads,omitempty"`

	// Connection details
	Host     string `json:"host"`
	Port     int    `json:"port"`
	UseTLS   bool   `json:"useTls,omitempty"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"` // Resolved from K8s Secret

	// Category for downloads (app-specific field names at adapter level)
	// For Radarr: movieCategory, for Sonarr: tvCategory, for Lidarr: musicCategory
	Category string `json:"category,omitempty"`

	// Directory override
	Directory string `json:"directory,omitempty"`
}

// Protocol constants
const (
	ProtocolTorrent = "torrent"
	ProtocolUsenet  = "usenet"
)

// Implementation constants for download clients
const (
	ImplementationQBittorrent  = "qbittorrent"
	ImplementationTransmission = "transmission"
	ImplementationDeluge       = "deluge"
	ImplementationRTorrent     = "rtorrent"
	ImplementationSABnzbd      = "sabnzbd"
	ImplementationNZBGet       = "nzbget"
)
