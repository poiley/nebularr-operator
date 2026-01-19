package v1

// IndexersIR represents indexer configuration
type IndexersIR struct {
	// ProwlarrRef delegates indexer management to Prowlarr
	// Mutually exclusive with Direct
	ProwlarrRef *ProwlarrRefIR `json:"prowlarrRef,omitempty"`

	// Direct configures indexers directly
	Direct []IndexerIR `json:"direct,omitempty"`
}

// ProwlarrRefIR references a Prowlarr instance
type ProwlarrRefIR struct {
	// ConfigName is the ProwlarrConfig name
	ConfigName string `json:"configName"`

	// AutoRegister this app with Prowlarr
	AutoRegister bool `json:"autoRegister"`

	// Include filters which indexers to sync (empty = all)
	Include []string `json:"include,omitempty"`

	// Exclude filters out specific indexers
	Exclude []string `json:"exclude,omitempty"`
}

// IndexerIR represents a direct indexer configuration
type IndexerIR struct {
	// ID is the service-side ID (populated from CurrentState, used for updates/deletes)
	ID int `json:"id,omitempty"`

	// Name is the indexer name (generated: "nebularr-{name}")
	Name string `json:"name"`

	// Protocol is "torrent" or "usenet"
	Protocol string `json:"protocol"`

	// Implementation is the indexer protocol (Torznab, Newznab)
	Implementation string `json:"implementation"`

	// Enable toggles the indexer
	Enable bool `json:"enable"`

	// Priority affects search order (lower = searched first in most apps)
	Priority int `json:"priority"`

	// URL is the indexer base URL
	URL string `json:"url"`

	// APIKey for authentication (resolved from K8s Secret)
	APIKey string `json:"apiKey,omitempty"`

	// Categories to search (numeric IDs)
	Categories []int `json:"categories,omitempty"`

	// MinimumSeeders for torrents
	MinimumSeeders int `json:"minimumSeeders,omitempty"`

	// SeedRatio target
	SeedRatio float64 `json:"seedRatio,omitempty"`

	// SeedTimeMinutes minimum seed time
	SeedTimeMinutes int `json:"seedTimeMinutes,omitempty"`

	// RSS/Search toggles
	EnableRss               bool `json:"enableRss"`
	EnableAutomaticSearch   bool `json:"enableAutomaticSearch"`
	EnableInteractiveSearch bool `json:"enableInteractiveSearch"`
}

// Indexer implementation constants
const (
	ImplementationTorznab = "Torznab"
	ImplementationNewznab = "Newznab"
)
