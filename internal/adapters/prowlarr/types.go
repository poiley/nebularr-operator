package prowlarr

import "time"

// SystemResource represents Prowlarr system status
type SystemResource struct {
	Version   string     `json:"version"`
	StartTime *time.Time `json:"startTime,omitempty"`
}

// TagResource represents a Prowlarr tag
type TagResource struct {
	ID    int    `json:"id,omitempty"`
	Label string `json:"label"`
}

// IndexerResource represents a Prowlarr indexer
type IndexerResource struct {
	ID             int            `json:"id,omitempty"`
	Name           string         `json:"name"`
	DefinitionName string         `json:"definitionName,omitempty"`
	Implementation string         `json:"implementation,omitempty"`
	ConfigContract string         `json:"configContract,omitempty"`
	Enable         bool           `json:"enable"`
	Protocol       string         `json:"protocol,omitempty"`
	Priority       int            `json:"priority"`
	AppProfileID   int            `json:"appProfileId,omitempty"`
	Tags           []int          `json:"tags,omitempty"`
	Fields         []IndexerField `json:"fields,omitempty"`
	Capabilities   *IndexerCaps   `json:"capabilities,omitempty"`
}

// IndexerField represents a configuration field for an indexer
type IndexerField struct {
	Name     string      `json:"name"`
	Value    interface{} `json:"value"`
	Type     string      `json:"type,omitempty"`
	Label    string      `json:"label,omitempty"`
	Advanced bool        `json:"advanced,omitempty"`
}

// IndexerCaps represents indexer capabilities
type IndexerCaps struct {
	Categories []IndexerCategory `json:"categories,omitempty"`
}

// IndexerCategory represents an indexer category
type IndexerCategory struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// IndexerProxyResource represents a Prowlarr indexer proxy
type IndexerProxyResource struct {
	ID             int                 `json:"id,omitempty"`
	Name           string              `json:"name"`
	Implementation string              `json:"implementation"`
	ConfigContract string              `json:"configContract,omitempty"`
	Tags           []int               `json:"tags,omitempty"`
	Fields         []IndexerProxyField `json:"fields,omitempty"`
}

// IndexerProxyField represents a configuration field for a proxy
type IndexerProxyField struct {
	Name     string      `json:"name"`
	Value    interface{} `json:"value"`
	Type     string      `json:"type,omitempty"`
	Label    string      `json:"label,omitempty"`
	Advanced bool        `json:"advanced,omitempty"`
}

// ApplicationResource represents a Prowlarr application (sync target)
type ApplicationResource struct {
	ID             int                `json:"id,omitempty"`
	Name           string             `json:"name"`
	Implementation string             `json:"implementation"`
	ConfigContract string             `json:"configContract,omitempty"`
	SyncLevel      string             `json:"syncLevel"`
	Tags           []int              `json:"tags,omitempty"`
	Fields         []ApplicationField `json:"fields,omitempty"`
}

// ApplicationField represents a configuration field for an application
type ApplicationField struct {
	Name     string      `json:"name"`
	Value    interface{} `json:"value"`
	Type     string      `json:"type,omitempty"`
	Label    string      `json:"label,omitempty"`
	Advanced bool        `json:"advanced,omitempty"`
}

// DownloadClientResource represents a Prowlarr download client
type DownloadClientResource struct {
	ID             int                   `json:"id,omitempty"`
	Name           string                `json:"name"`
	Implementation string                `json:"implementation"`
	ConfigContract string                `json:"configContract,omitempty"`
	Protocol       string                `json:"protocol"`
	Enable         bool                  `json:"enable"`
	Priority       int                   `json:"priority"`
	Tags           []int                 `json:"tags,omitempty"`
	Fields         []DownloadClientField `json:"fields,omitempty"`
}

// DownloadClientField represents a configuration field for a download client
type DownloadClientField struct {
	Name     string      `json:"name"`
	Value    interface{} `json:"value"`
	Type     string      `json:"type,omitempty"`
	Label    string      `json:"label,omitempty"`
	Advanced bool        `json:"advanced,omitempty"`
}

// Proxy implementation constants
const (
	ProxyImplFlareSolverr = "FlareSolverr"
	ProxyImplHTTP         = "HttpIndexerProxy"
	ProxyImplSocks4       = "Socks4IndexerProxy"
	ProxyImplSocks5       = "Socks5IndexerProxy"
)

// Application implementation constants
const (
	AppImplRadarr = "Radarr"
	AppImplSonarr = "Sonarr"
	AppImplLidarr = "Lidarr"
)

// SyncLevel constants
const (
	SyncLevelDisabled = "disabled"
	SyncLevelAddOnly  = "addOnly"
	SyncLevelFullSync = "fullSync"
)

// Protocol constants
const (
	ProtocolTorrent = "torrent"
	ProtocolUsenet  = "usenet"
)
