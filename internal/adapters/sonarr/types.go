package sonarr

import "time"

// SystemResource represents Sonarr system status
type SystemResource struct {
	Version   string     `json:"version"`
	StartTime *time.Time `json:"startTime"`
}

// TagResource represents a Sonarr tag
type TagResource struct {
	ID    int    `json:"id"`
	Label string `json:"label"`
}

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
type DownloadClientResource struct {
	ID                       int     `json:"id,omitempty"`
	Name                     string  `json:"name"`
	Implementation           string  `json:"implementation"`
	ConfigContract           string  `json:"configContract"`
	Protocol                 string  `json:"protocol"`
	Enable                   bool    `json:"enable"`
	Priority                 int     `json:"priority"`
	Tags                     []int   `json:"tags"`
	Fields                   []Field `json:"fields"`
	RemoveCompletedDownloads bool    `json:"removeCompletedDownloads"`
	RemoveFailedDownloads    bool    `json:"removeFailedDownloads"`
}

// IndexerResource represents a Sonarr indexer
type IndexerResource struct {
	ID                                  int     `json:"id,omitempty"`
	Name                                string  `json:"name"`
	Implementation                      string  `json:"implementation"`
	ConfigContract                      string  `json:"configContract"`
	Protocol                            string  `json:"protocol"`
	Enable                              bool    `json:"enable"`
	Priority                            int     `json:"priority"`
	Tags                                []int   `json:"tags"`
	Fields                              []Field `json:"fields"`
	EnableRss                           bool    `json:"enableRss"`
	EnableAutomaticSearch               bool    `json:"enableAutomaticSearch"`
	EnableInteractiveSearch             bool    `json:"enableInteractiveSearch"`
	SeasonSearchMaximumSingleEpisodeAge int     `json:"seasonSearchMaximumSingleEpisodeAge"`
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

// Field represents a dynamic field in resources
type Field struct {
	Name  string      `json:"name"`
	Value interface{} `json:"value"`
}

// CustomFormatResource represents a Sonarr custom format
type CustomFormatResource struct {
	ID                              int                         `json:"id,omitempty"`
	Name                            string                      `json:"name"`
	IncludeCustomFormatWhenRenaming bool                        `json:"includeCustomFormatWhenRenaming"`
	Specifications                  []CustomFormatSpecification `json:"specifications"`
}

// CustomFormatSpecification represents a spec within a custom format
type CustomFormatSpecification struct {
	Name           string  `json:"name"`
	Implementation string  `json:"implementation"`
	Negate         bool    `json:"negate"`
	Required       bool    `json:"required"`
	Fields         []Field `json:"fields"`
}

// RemotePathMappingResource represents a Sonarr remote path mapping
type RemotePathMappingResource struct {
	ID         int    `json:"id,omitempty"`
	Host       string `json:"host"`
	RemotePath string `json:"remotePath"`
	LocalPath  string `json:"localPath"`
}
