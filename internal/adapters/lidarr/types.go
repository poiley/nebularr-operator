package lidarr

import "time"

// SystemResource represents Lidarr system status
type SystemResource struct {
	Version   string     `json:"version"`
	StartTime *time.Time `json:"startTime"`
}

// TagResource represents a Lidarr tag
type TagResource struct {
	ID    int    `json:"id"`
	Label string `json:"label"`
}

// QualityProfileResource represents a Lidarr quality profile
type QualityProfileResource struct {
	ID             int                  `json:"id,omitempty"`
	Name           string               `json:"name"`
	UpgradeAllowed bool                 `json:"upgradeAllowed"`
	Cutoff         int                  `json:"cutoff"`
	Items          []QualityProfileItem `json:"items"`
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

// IndexerResource represents a Lidarr indexer
type IndexerResource struct {
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
type MetadataProfileResource struct {
	ID              int      `json:"id,omitempty"`
	Name            string   `json:"name"`
	PrimaryTypes    []string `json:"primaryAlbumTypes"`
	SecondaryTypes  []string `json:"secondaryAlbumTypes"`
	ReleaseStatuses []string `json:"releaseStatuses"`
}

// Field represents a dynamic field in resources
type Field struct {
	Name  string      `json:"name"`
	Value interface{} `json:"value"`
}
