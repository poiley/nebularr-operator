// Package compiler transforms CRD intent into Intermediate Representation (IR).
// It handles preset expansion, defaults merging, and capability pruning.
package compiler

import (
	"github.com/poiley/nebularr-operator/internal/adapters"
	"github.com/poiley/nebularr-operator/internal/presets"
)

// CompileInput holds all inputs for compilation
type CompileInput struct {
	// App identifies which app this is for: radarr, sonarr, lidarr, prowlarr
	App string

	// ConfigName is the name of the config resource (used for profile naming)
	ConfigName string

	// Namespace is the namespace of the config resource
	Namespace string

	// Connection details
	URL    string
	APIKey string

	// Quality configuration
	QualityPreset    string
	QualityOverrides *presets.QualityOverrides

	// Naming configuration
	NamingPreset string

	// Download clients
	DownloadClients []DownloadClientInput

	// Remote path mappings
	RemotePathMappings []RemotePathMappingInput

	// Indexers
	Indexers *IndexersInput

	// Root folders
	RootFolders []string

	// Import lists
	ImportLists []ImportListInput

	// Media management
	MediaManagement *MediaManagementInput

	// Authentication
	Authentication *AuthenticationInput

	// Notifications
	Notifications []NotificationInput

	// CustomFormats (Radarr/Sonarr only)
	CustomFormats []CustomFormatInput

	// DelayProfiles (Radarr/Sonarr only)
	DelayProfiles []DelayProfileInput

	// MetadataProfile (Readarr only)
	MetadataProfile *MetadataProfileInput

	// BookQuality (Readarr only)
	BookQuality *BookQualityInput

	// Capabilities for pruning unsupported features
	Capabilities *adapters.Capabilities

	// ResolvedSecrets maps secret references to resolved values
	ResolvedSecrets map[string]string
}

// DownloadClientInput holds download client configuration
type DownloadClientInput struct {
	Name                     string
	Implementation           string
	Host                     string
	Port                     int
	UseTLS                   bool
	Username                 string
	Password                 string
	Category                 string
	Priority                 int
	RemoveCompletedDownloads bool
	RemoveFailedDownloads    bool
}

// RemotePathMappingInput holds remote path mapping configuration
type RemotePathMappingInput struct {
	Host       string
	RemotePath string
	LocalPath  string
}

// IndexersInput holds indexer configuration
type IndexersInput struct {
	// ProwlarrRef for delegating to Prowlarr
	ProwlarrRef *ProwlarrRefInput

	// Direct indexers
	Direct []IndexerInput
}

// ProwlarrRefInput references a Prowlarr instance
type ProwlarrRefInput struct {
	ConfigName   string
	AutoRegister bool
	Include      []string
	Exclude      []string
}

// IndexerInput holds direct indexer configuration
type IndexerInput struct {
	Name                    string
	Protocol                string
	Implementation          string
	URL                     string
	APIKey                  string
	Categories              []int
	Priority                int
	MinimumSeeders          int
	SeedRatio               float64
	SeedTimeMinutes         int
	EnableRss               bool
	EnableAutomaticSearch   bool
	EnableInteractiveSearch bool
}

// ImportListInput holds import list configuration
type ImportListInput struct {
	Name                string
	Type                string
	Enabled             bool
	EnableAuto          bool
	SearchOnAdd         bool
	QualityProfileName  string
	RootFolderPath      string
	Monitor             string // Radarr: movieOnly, movieAndCollection, none
	MinimumAvailability string // Radarr: tba, announced, inCinemas, released
	SeriesType          string // Sonarr: standard, daily, anime
	SeasonFolder        bool   // Sonarr
	ShouldMonitor       string // Sonarr: all, future, missing, existing, firstSeason, latestSeason, pilot, none
	Settings            map[string]string
}

// MediaManagementInput holds media management configuration
type MediaManagementInput struct {
	RecycleBin             string
	RecycleBinCleanupDays  int
	SetPermissions         bool
	ChmodFolder            string
	ChownGroup             string
	DeleteEmptyFolders     bool
	CreateEmptyFolders     bool
	UseHardlinks           bool
	WatchLibraryForChanges *bool  // Lidarr
	AllowFingerprinting    string // Lidarr: never, newFiles, always
}

// AuthenticationInput holds authentication configuration
type AuthenticationInput struct {
	Method                 string // none, forms, external
	Username               string
	Password               string
	AuthenticationRequired string // enabled, disabledForLocalAddresses
}

// NotificationInput holds notification configuration
type NotificationInput struct {
	Name           string
	Implementation string

	// Event triggers
	OnGrab                      bool
	OnDownload                  bool
	OnUpgrade                   bool
	OnRename                    bool
	OnHealthIssue               bool
	OnHealthRestored            bool
	OnApplicationUpdate         bool
	OnManualInteractionRequired bool
	IncludeHealthWarnings       bool

	// Radarr-specific events
	OnMovieAdded                bool
	OnMovieDelete               bool
	OnMovieFileDelete           bool
	OnMovieFileDeleteForUpgrade bool

	// Sonarr-specific events
	OnSeriesAdd                   bool
	OnSeriesDelete                bool
	OnEpisodeFileDelete           bool
	OnEpisodeFileDeleteForUpgrade bool

	// Lidarr-specific events
	OnReleaseImport   bool
	OnArtistAdd       bool
	OnArtistDelete    bool
	OnAlbumDelete     bool
	OnTrackRetag      bool
	OnDownloadFailure bool
	OnImportFailure   bool

	// Type-specific settings (resolved from Settings and SettingsSecretRef)
	Fields map[string]interface{}

	// Tags are tag names (will be resolved to IDs by adapter)
	Tags []string
}

// CustomFormatInput holds custom format configuration
type CustomFormatInput struct {
	// Name is the display name for this custom format
	Name string

	// IncludeWhenRenaming includes this format in renamed file names
	IncludeWhenRenaming bool

	// Score is the score to assign in quality profiles
	Score int

	// Specifications define the matching rules
	Specifications []CustomFormatSpecInput
}

// CustomFormatSpecInput holds a single custom format specification
type CustomFormatSpecInput struct {
	// Name is the display name
	Name string

	// Type is the specification implementation type
	Type string

	// Negate inverts the match logic
	Negate bool

	// Required makes this specification mandatory
	Required bool

	// Value is the specification value (interpretation depends on Type)
	Value string
}

// DelayProfileInput holds delay profile configuration
type DelayProfileInput struct {
	// Name is a display name for identification
	Name string

	// PreferredProtocol: "usenet" or "torrent"
	PreferredProtocol string

	// UsenetDelay in minutes
	UsenetDelay int

	// TorrentDelay in minutes
	TorrentDelay int

	// EnableUsenet allows downloading from Usenet
	EnableUsenet bool

	// EnableTorrent allows downloading from torrents
	EnableTorrent bool

	// BypassIfHighestQuality bypasses delay if at cutoff quality
	BypassIfHighestQuality bool

	// BypassIfAboveCustomFormatScore bypasses delay based on CF score
	BypassIfAboveCustomFormatScore bool

	// MinimumCustomFormatScore is the threshold for bypass
	MinimumCustomFormatScore int

	// Tags restricts this profile to items with these tags
	Tags []string

	// Order determines priority (lower = higher priority)
	Order int
}

// MetadataProfileInput holds metadata profile configuration (Readarr only)
type MetadataProfileInput struct {
	// Name is the profile name
	Name string

	// MinPopularity is the minimum GoodReads popularity score
	MinPopularity int

	// SkipMissingDate skips books without release date
	SkipMissingDate bool

	// SkipMissingIsbn skips books without ISBN
	SkipMissingIsbn bool

	// SkipPartsAndSets skips parts and sets
	SkipPartsAndSets bool

	// SkipSeriesSecondary skips non-primary series entries
	SkipSeriesSecondary bool

	// AllowedLanguages are the languages to allow
	AllowedLanguages []string
}

// BookQualityInput holds book quality configuration (Readarr only)
type BookQualityInput struct {
	// ProfileName is the quality profile name
	ProfileName string

	// UpgradeAllowed enables quality upgrades
	UpgradeAllowed bool

	// CutoffFormat is the format where upgrades stop
	CutoffFormat string

	// AllowedFormats are the allowed book formats
	AllowedFormats []string
}
