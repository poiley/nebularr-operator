package lidarr

import (
	"github.com/poiley/nebularr-operator/internal/adapters/shared"
)

// Type aliases for shared types - provides backwards compatibility
type (
	SystemResource            = shared.SystemResource
	TagResource               = shared.TagResource
	Field                     = shared.Field
	RemotePathMappingResource = shared.RemotePathMappingResource
	HealthResource            = shared.HealthResource
	CustomFormatSpecification = shared.CustomFormatSpecification
)

// QualityProfileResource represents a Lidarr quality profile
type QualityProfileResource struct {
	ID                int                  `json:"id,omitempty"`
	Name              string               `json:"name"`
	UpgradeAllowed    bool                 `json:"upgradeAllowed"`
	Cutoff            int                  `json:"cutoff"`
	Items             []QualityProfileItem `json:"items"`
	MinFormatScore    int                  `json:"minFormatScore"`
	CutoffFormatScore int                  `json:"cutoffFormatScore"`
	FormatItems       []interface{}        `json:"formatItems"` // Use interface{} since we don't manage custom formats for Lidarr yet
}

// QualityProfileItem represents a quality in a profile
type QualityProfileItem struct {
	ID      int                  `json:"id,omitempty"`
	Name    string               `json:"name,omitempty"`
	Quality *Quality             `json:"quality,omitempty"`
	Items   []QualityProfileItem `json:"items"` // Must always be present (not omitempty)
	Allowed bool                 `json:"allowed"`
}

// Quality represents a quality definition
type Quality struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// DownloadClientResource represents a Lidarr download client
// Embeds BaseDownloadClientResource and adds Lidarr-specific fields
type DownloadClientResource struct {
	shared.BaseDownloadClientResource
	RemoveCompletedDownloads bool `json:"removeCompletedDownloads"`
	RemoveFailedDownloads    bool `json:"removeFailedDownloads"`
}

// IndexerResource represents a Lidarr indexer
// Type alias to shared base type (Lidarr uses all base fields)
type IndexerResource = shared.BaseIndexerResource

// RootFolderResource represents a Lidarr root folder
type RootFolderResource struct {
	ID                       int    `json:"id,omitempty"`
	Path                     string `json:"path"`
	Name                     string `json:"name,omitempty"`
	DefaultMetadataProfileId int    `json:"defaultMetadataProfileId,omitempty"`
	DefaultQualityProfileId  int    `json:"defaultQualityProfileId,omitempty"`
	DefaultMonitorOption     string `json:"defaultMonitorOption,omitempty"`
}

// NamingConfigResource represents Lidarr naming configuration
type NamingConfigResource struct {
	ID                       int    `json:"id,omitempty"`
	RenameTracks             bool   `json:"renameTracks"`
	ReplaceIllegalCharacters bool   `json:"replaceIllegalCharacters"`
	StandardTrackFormat      string `json:"standardTrackFormat"`
	MultiDiscTrackFormat     string `json:"multiDiscTrackFormat"`
	ArtistFolderFormat       string `json:"artistFolderFormat"`
	AlbumFolderFormat        string `json:"albumFolderFormat,omitempty"`
}

// MetadataProfileResource represents a Lidarr metadata profile
// We only need the ID and Name for root folder creation
type MetadataProfileResource struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// NotificationResource represents a Lidarr notification
type NotificationResource struct {
	ID                    int     `json:"id,omitempty"`
	Name                  string  `json:"name"`
	Implementation        string  `json:"implementation"`
	ConfigContract        string  `json:"configContract"`
	Tags                  []int   `json:"tags"`
	Fields                []Field `json:"fields"`
	OnGrab                bool    `json:"onGrab"`
	OnReleaseImport       bool    `json:"onReleaseImport"`
	OnUpgrade             bool    `json:"onUpgrade"`
	OnRename              bool    `json:"onRename"`
	OnArtistAdd           bool    `json:"onArtistAdd"`
	OnArtistDelete        bool    `json:"onArtistDelete"`
	OnAlbumDelete         bool    `json:"onAlbumDelete"`
	OnTrackRetag          bool    `json:"onTrackRetag"`
	OnDownloadFailure     bool    `json:"onDownloadFailure"`
	OnImportFailure       bool    `json:"onImportFailure"`
	OnHealthIssue         bool    `json:"onHealthIssue"`
	OnHealthRestored      bool    `json:"onHealthRestored"`
	OnApplicationUpdate   bool    `json:"onApplicationUpdate"`
	IncludeHealthWarnings bool    `json:"includeHealthWarnings"`
}

// CustomFormatResource represents a Lidarr custom format (v2.0+)
// Embeds BaseCustomFormatResource for shared fields
type CustomFormatResource struct {
	shared.BaseCustomFormatResource
}

// DelayProfileResource represents a Lidarr delay profile
// Type alias to shared base type
type DelayProfileResource = shared.BaseDelayProfileResource
