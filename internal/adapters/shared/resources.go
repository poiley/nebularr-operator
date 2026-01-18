package shared

// BaseDownloadClientResource contains common download client fields
// shared across Sonarr, Lidarr, Readarr, and Prowlarr.
// App-specific adapters can embed this and add additional fields.
type BaseDownloadClientResource struct {
	ID             int     `json:"id,omitempty"`
	Name           string  `json:"name"`
	Implementation string  `json:"implementation"`
	ConfigContract string  `json:"configContract"`
	Protocol       string  `json:"protocol"`
	Enable         bool    `json:"enable"`
	Priority       int     `json:"priority"`
	Tags           []int   `json:"tags"`
	Fields         []Field `json:"fields"`
}

// BaseIndexerResource contains common indexer fields
// shared across Sonarr, Lidarr, Readarr, and Prowlarr.
// App-specific adapters can embed this and add additional fields.
type BaseIndexerResource struct {
	ID                      int     `json:"id,omitempty"`
	Name                    string  `json:"name"`
	Implementation          string  `json:"implementation"`
	ConfigContract          string  `json:"configContract"`
	Protocol                string  `json:"protocol"`
	Enable                  bool    `json:"enable"`
	Priority                int     `json:"priority"`
	Tags                    []int   `json:"tags"`
	Fields                  []Field `json:"fields"`
	EnableRss               bool    `json:"enableRss"`
	EnableAutomaticSearch   bool    `json:"enableAutomaticSearch"`
	EnableInteractiveSearch bool    `json:"enableInteractiveSearch"`
}

// BaseDelayProfileResource contains common delay profile fields
// shared across Sonarr, Lidarr, and Radarr.
type BaseDelayProfileResource struct {
	ID                             int    `json:"id,omitempty"`
	Order                          int    `json:"order"`
	PreferredProtocol              string `json:"preferredProtocol"`
	UsenetDelay                    int    `json:"usenetDelay"`
	TorrentDelay                   int    `json:"torrentDelay"`
	EnableUsenet                   bool   `json:"enableUsenet"`
	EnableTorrent                  bool   `json:"enableTorrent"`
	BypassIfHighestQuality         bool   `json:"bypassIfHighestQuality"`
	BypassIfAboveCustomFormatScore bool   `json:"bypassIfAboveCustomFormatScore"`
	MinimumCustomFormatScore       int    `json:"minimumCustomFormatScore"`
	Tags                           []int  `json:"tags"`
}

// BaseCustomFormatResource contains common custom format fields
// shared across Sonarr, Lidarr, and Radarr.
type BaseCustomFormatResource struct {
	ID                              int                         `json:"id,omitempty"`
	Name                            string                      `json:"name"`
	IncludeCustomFormatWhenRenaming bool                        `json:"includeCustomFormatWhenRenaming"`
	Specifications                  []CustomFormatSpecification `json:"specifications"`
}

// CustomFormatSpecification represents a custom format specification
// used within custom format definitions.
type CustomFormatSpecification struct {
	Name           string  `json:"name"`
	Implementation string  `json:"implementation"`
	Negate         bool    `json:"negate"`
	Required       bool    `json:"required"`
	Fields         []Field `json:"fields"`
}
