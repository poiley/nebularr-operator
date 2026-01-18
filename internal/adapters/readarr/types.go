package readarr

import "time"

// SystemResource represents the system status response from Readarr
type SystemResource struct {
	Version   string     `json:"version"`
	StartTime *time.Time `json:"startTime"`
}

// DownloadClientResource represents a download client in Readarr
type DownloadClientResource struct {
	ID             int             `json:"id"`
	Name           string          `json:"name"`
	Implementation string          `json:"implementation"`
	Protocol       string          `json:"protocol"`
	Enable         bool            `json:"enable"`
	Priority       int             `json:"priority"`
	Tags           []int           `json:"tags,omitempty"`
	Fields         []FieldResource `json:"fields,omitempty"`
}

// IndexerResource represents an indexer in Readarr
type IndexerResource struct {
	ID             int             `json:"id"`
	Name           string          `json:"name"`
	Implementation string          `json:"implementation"`
	Protocol       string          `json:"protocol"`
	Enable         bool            `json:"enable"`
	Priority       int             `json:"priority"`
	Tags           []int           `json:"tags,omitempty"`
	Fields         []FieldResource `json:"fields,omitempty"`
}

// FieldResource represents a field in a schema-based resource
type FieldResource struct {
	Name  string      `json:"name"`
	Value interface{} `json:"value"`
}

// TagResource represents a tag in Readarr
type TagResource struct {
	ID    int    `json:"id"`
	Label string `json:"label"`
}

// RootFolderResource represents a root folder in Readarr
type RootFolderResource struct {
	ID                          int    `json:"id"`
	Path                        string `json:"path"`
	Name                        string `json:"name,omitempty"`
	DefaultMetadataProfileId    int    `json:"defaultMetadataProfileId,omitempty"`
	DefaultQualityProfileId     int    `json:"defaultQualityProfileId,omitempty"`
	DefaultMonitorOption        string `json:"defaultMonitorOption,omitempty"`
	DefaultNewItemMonitorOption string `json:"defaultNewItemMonitorOption,omitempty"`
	DefaultTags                 []int  `json:"defaultTags,omitempty"`
	IsCalibreLibrary            bool   `json:"isCalibreLibrary,omitempty"`
	Host                        string `json:"host,omitempty"`
	Port                        int    `json:"port,omitempty"`
	UrlBase                     string `json:"urlBase,omitempty"`
	Username                    string `json:"username,omitempty"`
	Password                    string `json:"password,omitempty"`
	Library                     string `json:"library,omitempty"`
	OutputFormat                string `json:"outputFormat,omitempty"`
	OutputProfile               int    `json:"outputProfile,omitempty"`
	UseSsl                      bool   `json:"useSsl,omitempty"`
}

// QualityProfileResource represents a quality profile in Readarr
type QualityProfileResource struct {
	ID             int                          `json:"id"`
	Name           string                       `json:"name"`
	UpgradeAllowed bool                         `json:"upgradeAllowed"`
	Cutoff         int                          `json:"cutoff"`
	Items          []QualityProfileItemResource `json:"items"`
}

// QualityProfileItemResource represents an item in a quality profile
type QualityProfileItemResource struct {
	ID      int                          `json:"id,omitempty"`
	Name    string                       `json:"name,omitempty"`
	Quality *QualityResource             `json:"quality,omitempty"`
	Items   []QualityProfileItemResource `json:"items,omitempty"`
	Allowed bool                         `json:"allowed"`
}

// QualityResource represents a quality definition
type QualityResource struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// MetadataProfileResource represents a metadata profile in Readarr
type MetadataProfileResource struct {
	ID                  int     `json:"id"`
	Name                string  `json:"name"`
	MinPopularity       float64 `json:"minPopularity"`
	SkipMissingDate     bool    `json:"skipMissingDate"`
	SkipMissingIsbn     bool    `json:"skipMissingIsbn"`
	SkipPartsAndSets    bool    `json:"skipPartsAndSets"`
	SkipSeriesSecondary bool    `json:"skipSeriesSecondary"`
	AllowedLanguages    string  `json:"allowedLanguages,omitempty"`
}

// NamingConfigResource represents naming configuration in Readarr
type NamingConfigResource struct {
	ID                       int    `json:"id"`
	RenameBooks              bool   `json:"renameBooks"`
	ReplaceIllegalCharacters bool   `json:"replaceIllegalCharacters"`
	ColonReplacementFormat   int    `json:"colonReplacementFormat"`
	StandardBookFormat       string `json:"standardBookFormat"`
	AuthorFolderFormat       string `json:"authorFolderFormat"`
}

// ImportListResource represents an import list in Readarr
type ImportListResource struct {
	ID                 int             `json:"id"`
	Name               string          `json:"name"`
	Implementation     string          `json:"implementation"`
	Enable             bool            `json:"enable"`
	EnableAutomaticAdd bool            `json:"enableAutomaticAdd"`
	ShouldMonitor      string          `json:"shouldMonitor"`
	RootFolderPath     string          `json:"rootFolderPath"`
	QualityProfileId   int             `json:"qualityProfileId"`
	MetadataProfileId  int             `json:"metadataProfileId"`
	Tags               []int           `json:"tags,omitempty"`
	Fields             []FieldResource `json:"fields,omitempty"`
}

// NotificationResource represents a notification in Readarr
type NotificationResource struct {
	ID                         int             `json:"id"`
	Name                       string          `json:"name"`
	Implementation             string          `json:"implementation"`
	OnGrab                     bool            `json:"onGrab"`
	OnReleaseImport            bool            `json:"onReleaseImport"`
	OnUpgrade                  bool            `json:"onUpgrade"`
	OnRename                   bool            `json:"onRename"`
	OnAuthorDelete             bool            `json:"onAuthorDelete"`
	OnBookDelete               bool            `json:"onBookDelete"`
	OnBookFileDelete           bool            `json:"onBookFileDelete"`
	OnBookFileDeleteForUpgrade bool            `json:"onBookFileDeleteForUpgrade"`
	OnHealthIssue              bool            `json:"onHealthIssue"`
	OnDownloadFailure          bool            `json:"onDownloadFailure"`
	OnImportFailure            bool            `json:"onImportFailure"`
	OnBookRetag                bool            `json:"onBookRetag"`
	IncludeHealthWarnings      bool            `json:"includeHealthWarnings"`
	SupportsOnGrab             bool            `json:"supportsOnGrab"`
	SupportsOnReleaseImport    bool            `json:"supportsOnReleaseImport"`
	SupportsOnUpgrade          bool            `json:"supportsOnUpgrade"`
	SupportsOnRename           bool            `json:"supportsOnRename"`
	Tags                       []int           `json:"tags,omitempty"`
	Fields                     []FieldResource `json:"fields,omitempty"`
}
