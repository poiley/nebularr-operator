package v1

// ProwlarrIR is the Prowlarr-specific intermediate representation.
// Prowlarr differs from other *arr apps - it manages indexers natively
// and syncs them to downstream applications (Radarr, Sonarr, Lidarr).
type ProwlarrIR struct {
	// Connection details for Prowlarr
	Connection *ConnectionIR `json:"connection,omitempty"`

	// Indexers configured in Prowlarr
	Indexers []ProwlarrIndexerIR `json:"indexers,omitempty"`

	// Proxies for indexer requests (FlareSolverr, HTTP, SOCKS)
	Proxies []IndexerProxyIR `json:"proxies,omitempty"`

	// Applications to sync indexers to (Radarr, Sonarr, Lidarr)
	Applications []ProwlarrApplicationIR `json:"applications,omitempty"`

	// DownloadClients configured in Prowlarr
	DownloadClients []DownloadClientIR `json:"downloadClients,omitempty"`

	// Unrealized tracks features that could not be compiled
	Unrealized []UnrealizedFeature `json:"unrealized,omitempty"`
}

// ProwlarrIndexerIR represents a native indexer in Prowlarr.
// Unlike IndexerIR used by other apps, this uses definitions (schemas).
type ProwlarrIndexerIR struct {
	// Name is the display name
	Name string `json:"name"`

	// Definition is the indexer definition ID (e.g., "1337x", "Nyaa", "IPTorrents")
	Definition string `json:"definition"`

	// Enable toggles the indexer
	Enable bool `json:"enable"`

	// Priority affects search order (1-50, lower = searched first)
	Priority int `json:"priority"`

	// BaseURL overrides the default URL for the indexer
	BaseURL string `json:"baseUrl,omitempty"`

	// APIKey for private trackers (resolved from K8s Secret)
	APIKey string `json:"apiKey,omitempty"`

	// Settings are definition-specific key-value settings
	Settings map[string]string `json:"settings,omitempty"`

	// Tags associate this indexer with proxies and applications
	Tags []string `json:"tags,omitempty"`
}

// IndexerProxyIR represents a proxy for indexer requests
type IndexerProxyIR struct {
	// Name is the display name
	Name string `json:"name"`

	// Type: flaresolverr, http, socks4, socks5
	Type string `json:"type"`

	// Host is the proxy URL or hostname
	// For FlareSolverr: full URL (http://flaresolverr:8191)
	// For HTTP/SOCKS: hostname only
	Host string `json:"host"`

	// Port for HTTP/SOCKS proxies (not used for FlareSolverr)
	Port int `json:"port,omitempty"`

	// Username for authenticated proxies
	Username string `json:"username,omitempty"`

	// Password for authenticated proxies (resolved from K8s Secret)
	Password string `json:"password,omitempty"`

	// RequestTimeout for FlareSolverr (seconds)
	RequestTimeout int `json:"requestTimeout,omitempty"`

	// Tags to associate with indexers that should use this proxy
	Tags []string `json:"tags,omitempty"`
}

// ProwlarrApplicationIR represents a downstream app to sync indexers to
type ProwlarrApplicationIR struct {
	// Name is the display name
	Name string `json:"name"`

	// Type: radarr, sonarr, lidarr
	Type string `json:"type"`

	// ProwlarrURL is the URL Prowlarr uses for itself (how apps reach Prowlarr)
	ProwlarrURL string `json:"prowlarrUrl,omitempty"`

	// URL is the application URL (how Prowlarr reaches the app)
	URL string `json:"url"`

	// APIKey for the application (resolved from K8s Secret or auto-discovered)
	APIKey string `json:"apiKey"`

	// SyncCategories to sync (numeric Newznab category IDs)
	// Defaults based on app type if not specified
	SyncCategories []int `json:"syncCategories,omitempty"`

	// SyncLevel: disabled, addOnly, fullSync
	SyncLevel string `json:"syncLevel"`

	// Tags to filter which indexers sync to this app
	Tags []string `json:"tags,omitempty"`
}

// Proxy type constants
const (
	ProxyTypeFlareSolverr = "flaresolverr"
	ProxyTypeHTTP         = "http"
	ProxyTypeSocks4       = "socks4"
	ProxyTypeSocks5       = "socks5"
)

// Application type constants
const (
	AppTypeRadarr  = "radarr"
	AppTypeSonarr  = "sonarr"
	AppTypeLidarr  = "lidarr"
	AppTypeReadarr = "readarr"
)

// Sync level constants
const (
	SyncLevelDisabled = "disabled"
	SyncLevelAddOnly  = "addOnly"
	SyncLevelFullSync = "fullSync"
)

// Default Newznab categories by app type
var DefaultSyncCategories = map[string][]int{
	AppTypeRadarr:  {2000, 2010, 2020, 2030, 2040, 2045, 2050, 2060}, // Movies
	AppTypeSonarr:  {5000, 5010, 5020, 5030, 5040, 5045, 5050},       // TV
	AppTypeLidarr:  {3000, 3010, 3020, 3030, 3040},                   // Audio
	AppTypeReadarr: {7000, 7010, 7020, 7030, 8000, 8010},             // Books, EBooks, Audiobooks
}
