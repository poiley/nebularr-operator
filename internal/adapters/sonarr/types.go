package sonarr

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

// QualityProfileResource represents a Sonarr quality profile
type QualityProfileResource struct {
	ID                    int                  `json:"id,omitempty"`
	Name                  string               `json:"name"`
	UpgradeAllowed        bool                 `json:"upgradeAllowed"`
	Cutoff                int                  `json:"cutoff"`
	Items                 []QualityProfileItem `json:"items"`
	FormatItems           []ProfileFormatItem  `json:"formatItems"`
	MinFormatScore        int                  `json:"minFormatScore"`
	MinUpgradeFormatScore int                  `json:"minUpgradeFormatScore"`
	CutoffFormatScore     int                  `json:"cutoffFormatScore"`
}

// QualityProfileItem represents a quality in a profile
type QualityProfileItem struct {
	ID      int                  `json:"id,omitempty"`
	Name    string               `json:"name,omitempty"`
	Quality *Quality             `json:"quality,omitempty"`
	Items   []QualityProfileItem `json:"items,omitempty"`
	Allowed bool                 `json:"allowed"`
}

// Quality represents a quality definition
type Quality struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	Source     string `json:"source"`
	Resolution int    `json:"resolution"`
}

// ProfileFormatItem represents a custom format in a profile
type ProfileFormatItem struct {
	Format int `json:"format"`
	Score  int `json:"score"`
}

// DownloadClientResource represents a Sonarr download client
// Embeds BaseDownloadClientResource and adds Sonarr-specific fields
type DownloadClientResource struct {
	shared.BaseDownloadClientResource
	RemoveCompletedDownloads bool `json:"removeCompletedDownloads"`
	RemoveFailedDownloads    bool `json:"removeFailedDownloads"`
}

// IndexerResource represents a Sonarr indexer
// Embeds BaseIndexerResource and adds Sonarr-specific fields
type IndexerResource struct {
	shared.BaseIndexerResource
	SeasonSearchMaximumSingleEpisodeAge int `json:"seasonSearchMaximumSingleEpisodeAge"`
}

// RootFolderResource represents a Sonarr root folder
type RootFolderResource struct {
	ID   int    `json:"id,omitempty"`
	Path string `json:"path"`
}

// NamingConfigResource represents Sonarr naming configuration
type NamingConfigResource struct {
	ID                       int    `json:"id,omitempty"`
	RenameEpisodes           bool   `json:"renameEpisodes"`
	ReplaceIllegalCharacters bool   `json:"replaceIllegalCharacters"`
	StandardEpisodeFormat    string `json:"standardEpisodeFormat"`
	DailyEpisodeFormat       string `json:"dailyEpisodeFormat"`
	AnimeEpisodeFormat       string `json:"animeEpisodeFormat"`
	SeriesFolderFormat       string `json:"seriesFolderFormat"`
	SeasonFolderFormat       string `json:"seasonFolderFormat"`
	SpecialsFolderFormat     string `json:"specialsFolderFormat"`
	MultiEpisodeStyle        int    `json:"multiEpisodeStyle"`
}

// CustomFormatResource represents a Sonarr custom format
// Embeds BaseCustomFormatResource for shared fields
type CustomFormatResource struct {
	shared.BaseCustomFormatResource
}

// NotificationResource represents a Sonarr notification
type NotificationResource struct {
	ID                            int     `json:"id,omitempty"`
	Name                          string  `json:"name"`
	Implementation                string  `json:"implementation"`
	ConfigContract                string  `json:"configContract"`
	Tags                          []int   `json:"tags"`
	Fields                        []Field `json:"fields"`
	OnGrab                        bool    `json:"onGrab"`
	OnDownload                    bool    `json:"onDownload"`
	OnUpgrade                     bool    `json:"onUpgrade"`
	OnRename                      bool    `json:"onRename"`
	OnSeriesAdd                   bool    `json:"onSeriesAdd"`
	OnSeriesDelete                bool    `json:"onSeriesDelete"`
	OnEpisodeFileDelete           bool    `json:"onEpisodeFileDelete"`
	OnEpisodeFileDeleteForUpgrade bool    `json:"onEpisodeFileDeleteForUpgrade"`
	OnHealthIssue                 bool    `json:"onHealthIssue"`
	OnHealthRestored              bool    `json:"onHealthRestored"`
	OnApplicationUpdate           bool    `json:"onApplicationUpdate"`
	OnManualInteractionRequired   bool    `json:"onManualInteractionRequired"`
	IncludeHealthWarnings         bool    `json:"includeHealthWarnings"`
}

// DelayProfileResource represents a Sonarr delay profile
// Type alias to shared base type
type DelayProfileResource = shared.BaseDelayProfileResource
