// Package compiler transforms CRD intent into Intermediate Representation (IR).
// It handles preset expansion, defaults merging, and capability pruning.
package compiler

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"

	"github.com/poiley/nebularr-operator/internal/adapters"
	irv1 "github.com/poiley/nebularr-operator/internal/ir/v1"
	"github.com/poiley/nebularr-operator/internal/presets"
)

// Compiler transforms CRD intent into IR
type Compiler struct {
	expander *presets.Expander
}

// New creates a new Compiler
func New() *Compiler {
	return &Compiler{
		expander: presets.NewExpander(),
	}
}

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

// Compile transforms CRD intent into IR
func (c *Compiler) Compile(ctx context.Context, input CompileInput) (*irv1.IR, error) {
	ir := &irv1.IR{
		Version:     irv1.IRVersion,
		GeneratedAt: time.Now(),
		App:         input.App,
	}

	// 1. Set connection
	ir.Connection = &irv1.ConnectionIR{
		URL:    input.URL,
		APIKey: input.APIKey,
	}

	// 2. Expand quality preset based on app type
	profileName := fmt.Sprintf("nebularr-%s", input.ConfigName)
	switch input.App {
	case adapters.AppRadarr, adapters.AppSonarr:
		presetName := input.QualityPreset
		if presetName == "" {
			presetName = presets.DefaultVideoPreset
		}
		ir.Quality = &irv1.QualityIR{
			Video: c.expander.ExpandVideoPreset(presetName, input.QualityOverrides, profileName),
		}
	case adapters.AppLidarr:
		presetName := input.QualityPreset
		if presetName == "" {
			presetName = presets.DefaultAudioPreset
		}
		ir.Quality = &irv1.QualityIR{
			Audio: c.expander.ExpandAudioPreset(presetName, input.QualityOverrides, profileName),
		}
	case adapters.AppReadarr:
		// Readarr uses book quality profiles
		if input.BookQuality != nil {
			ir.Quality = &irv1.QualityIR{
				Book: c.compileBookQuality(input.BookQuality, profileName),
			}
		}
		// Handle metadata profiles
		if input.MetadataProfile != nil {
			ir.MetadataProfiles = []*irv1.MetadataProfileIR{
				c.compileMetadataProfile(input.MetadataProfile, profileName),
			}
		}
	}

	// 3. Expand naming preset based on app type
	namingPreset := input.NamingPreset
	if namingPreset == "" {
		namingPreset = presets.DefaultNamingPreset
	}
	switch input.App {
	case adapters.AppRadarr:
		ir.Naming = &irv1.NamingIR{
			Radarr: c.expander.ExpandRadarrNaming(namingPreset),
		}
	case adapters.AppSonarr:
		ir.Naming = &irv1.NamingIR{
			Sonarr: c.expander.ExpandSonarrNaming(namingPreset),
		}
	case adapters.AppLidarr:
		ir.Naming = &irv1.NamingIR{
			Lidarr: c.expander.ExpandLidarrNaming(namingPreset),
		}
	}

	// 4. Compile download clients
	ir.DownloadClients = c.compileDownloadClients(input.DownloadClients, input.ConfigName)

	// 5. Compile remote path mappings
	ir.RemotePathMappings = c.compileRemotePathMappings(input.RemotePathMappings)

	// 6. Compile indexers
	ir.Indexers = c.compileIndexers(input.Indexers, input.ConfigName)

	// 7. Compile root folders
	for _, path := range input.RootFolders {
		ir.RootFolders = append(ir.RootFolders, irv1.RootFolderIR{Path: path})
	}

	// 8. Compile import lists
	ir.ImportLists = c.compileImportListsToIR(input.ImportLists)

	// 9. Compile media management
	ir.MediaManagement = c.compileMediaManagementToIR(input.MediaManagement)

	// 10. Compile authentication
	ir.Authentication = c.compileAuthenticationToIR(input.Authentication)

	// 11. Compile notifications
	ir.Notifications = c.compileNotificationsToIR(input.Notifications, input.ConfigName)

	// 12. Compile custom formats (Radarr/Sonarr/Lidarr)
	if input.App == adapters.AppRadarr || input.App == adapters.AppSonarr || input.App == adapters.AppLidarr {
		ir.CustomFormats = c.compileCustomFormatsToIR(input.CustomFormats, input.ConfigName)

		// Populate format scores in quality profile from custom format scores
		if ir.Quality != nil && len(input.CustomFormats) > 0 {
			if ir.Quality.Video != nil {
				ir.Quality.Video.FormatScores = c.compileFormatScores(input.CustomFormats, input.ConfigName)
			}
			if ir.Quality.Audio != nil {
				ir.Quality.Audio.FormatScores = c.compileFormatScores(input.CustomFormats, input.ConfigName)
			}
		}
	}

	// 13. Compile delay profiles (Radarr/Sonarr/Lidarr)
	if input.App == adapters.AppRadarr || input.App == adapters.AppSonarr || input.App == adapters.AppLidarr {
		ir.DelayProfiles = c.compileDelayProfilesToIR(input.DelayProfiles)
	}

	// 14. Prune unsupported features based on capabilities
	if input.Capabilities != nil {
		ir.Unrealized = c.pruneUnsupported(ir, input.Capabilities)
	}

	// 15. Generate source hash for drift detection
	ir.SourceHash = c.hashInput(input)

	return ir, nil
}

// compileDownloadClients converts download client inputs to IR
func (c *Compiler) compileDownloadClients(clients []DownloadClientInput, configName string) []irv1.DownloadClientIR {
	result := make([]irv1.DownloadClientIR, 0, len(clients))

	for _, dc := range clients {
		ir := irv1.DownloadClientIR{
			Name:                     fmt.Sprintf("nebularr-%s-%s", configName, dc.Name),
			Implementation:           dc.Implementation,
			Protocol:                 inferProtocol(dc.Implementation),
			Enable:                   true,
			Priority:                 dc.Priority,
			Host:                     dc.Host,
			Port:                     dc.Port,
			UseTLS:                   dc.UseTLS,
			Username:                 dc.Username,
			Password:                 dc.Password,
			Category:                 dc.Category,
			RemoveCompletedDownloads: dc.RemoveCompletedDownloads,
			RemoveFailedDownloads:    dc.RemoveFailedDownloads,
		}
		result = append(result, ir)
	}

	return result
}

// compileRemotePathMappings converts remote path mapping inputs to IR
func (c *Compiler) compileRemotePathMappings(mappings []RemotePathMappingInput) []irv1.RemotePathMappingIR {
	if len(mappings) == 0 {
		return nil
	}

	result := make([]irv1.RemotePathMappingIR, 0, len(mappings))
	for _, m := range mappings {
		ir := irv1.RemotePathMappingIR{
			Host:       m.Host,
			RemotePath: m.RemotePath,
			LocalPath:  m.LocalPath,
		}
		result = append(result, ir)
	}
	return result
}

// compileIndexers converts indexer inputs to IR
func (c *Compiler) compileIndexers(input *IndexersInput, configName string) *irv1.IndexersIR {
	if input == nil {
		return nil
	}

	result := &irv1.IndexersIR{}

	// Handle Prowlarr reference
	if input.ProwlarrRef != nil {
		result.ProwlarrRef = &irv1.ProwlarrRefIR{
			ConfigName:   input.ProwlarrRef.ConfigName,
			AutoRegister: input.ProwlarrRef.AutoRegister,
			Include:      input.ProwlarrRef.Include,
			Exclude:      input.ProwlarrRef.Exclude,
		}
	}

	// Handle direct indexers
	for _, idx := range input.Direct {
		ir := irv1.IndexerIR{
			Name:                    fmt.Sprintf("nebularr-%s-%s", configName, idx.Name),
			Protocol:                idx.Protocol,
			Implementation:          idx.Implementation,
			Enable:                  true,
			Priority:                idx.Priority,
			URL:                     idx.URL,
			APIKey:                  idx.APIKey,
			Categories:              idx.Categories,
			MinimumSeeders:          idx.MinimumSeeders,
			SeedRatio:               idx.SeedRatio,
			SeedTimeMinutes:         idx.SeedTimeMinutes,
			EnableRss:               idx.EnableRss,
			EnableAutomaticSearch:   idx.EnableAutomaticSearch,
			EnableInteractiveSearch: idx.EnableInteractiveSearch,
		}
		result.Direct = append(result.Direct, ir)
	}

	return result
}

// compileBookQuality converts BookQualityInput to BookQualityIR
func (c *Compiler) compileBookQuality(input *BookQualityInput, profileName string) *irv1.BookQualityIR {
	if input == nil {
		return nil
	}

	ir := &irv1.BookQualityIR{
		ProfileName:    profileName,
		UpgradeAllowed: input.UpgradeAllowed,
	}

	// Build tiers from allowed formats
	for _, format := range input.AllowedFormats {
		tier := irv1.BookQualityTierIR{
			Name:    format,
			Formats: []string{format},
			Allowed: true,
		}
		ir.Tiers = append(ir.Tiers, tier)

		// Set cutoff if this format matches
		if format == input.CutoffFormat {
			ir.Cutoff = tier
		}
	}

	return ir
}

// compileMetadataProfile converts MetadataProfileInput to MetadataProfileIR
func (c *Compiler) compileMetadataProfile(input *MetadataProfileInput, profileName string) *irv1.MetadataProfileIR {
	if input == nil {
		return nil
	}

	// Use the profile name with nebularr prefix for identification
	name := profileName
	if input.Name != "" {
		name = fmt.Sprintf("nebularr-%s", input.Name)
	}

	ir := &irv1.MetadataProfileIR{
		Name:                name,
		MinPopularity:       float64(input.MinPopularity),
		SkipMissingDate:     input.SkipMissingDate,
		SkipMissingIsbn:     input.SkipMissingIsbn,
		SkipPartsAndSets:    input.SkipPartsAndSets,
		SkipSeriesSecondary: input.SkipSeriesSecondary,
	}

	// Convert allowed languages to comma-separated string
	if len(input.AllowedLanguages) > 0 {
		for i, lang := range input.AllowedLanguages {
			if i > 0 {
				ir.AllowedLanguages += ","
			}
			ir.AllowedLanguages += lang
		}
	}

	return ir
}

// pruneUnsupported removes features not supported by the service capabilities
func (c *Compiler) pruneUnsupported(ir *irv1.IR, caps *adapters.Capabilities) []irv1.UnrealizedFeature {
	var unrealized []irv1.UnrealizedFeature

	// Check video quality tiers against supported resolutions
	if ir.Quality != nil && ir.Quality.Video != nil {
		supportedRes := make(map[string]bool)
		for _, res := range caps.Resolutions {
			supportedRes[res] = true
		}

		prunedTiers := make([]irv1.VideoQualityTierIR, 0)
		for _, tier := range ir.Quality.Video.Tiers {
			if supportedRes[tier.Resolution] {
				prunedTiers = append(prunedTiers, tier)
			} else {
				unrealized = append(unrealized, irv1.UnrealizedFeature{
					Feature: fmt.Sprintf("resolution:%s", tier.Resolution),
					Reason:  "not supported by service",
				})
			}
		}
		ir.Quality.Video.Tiers = prunedTiers
	}

	// Check download client types
	if len(caps.DownloadClientTypes) > 0 {
		supportedTypes := make(map[string]bool)
		for _, t := range caps.DownloadClientTypes {
			supportedTypes[t] = true
		}

		prunedClients := make([]irv1.DownloadClientIR, 0)
		for _, dc := range ir.DownloadClients {
			// Normalize implementation name for comparison
			impl := normalizeImplementation(dc.Implementation)
			if supportedTypes[impl] {
				prunedClients = append(prunedClients, dc)
			} else {
				unrealized = append(unrealized, irv1.UnrealizedFeature{
					Feature: fmt.Sprintf("downloadclient:%s", dc.Implementation),
					Reason:  "not supported by service",
				})
			}
		}
		ir.DownloadClients = prunedClients
	}

	// Check indexer types
	if ir.Indexers != nil && len(caps.IndexerTypes) > 0 {
		supportedTypes := make(map[string]bool)
		for _, t := range caps.IndexerTypes {
			supportedTypes[t] = true
		}

		prunedIndexers := make([]irv1.IndexerIR, 0)
		for _, idx := range ir.Indexers.Direct {
			if supportedTypes[idx.Implementation] {
				prunedIndexers = append(prunedIndexers, idx)
			} else {
				unrealized = append(unrealized, irv1.UnrealizedFeature{
					Feature: fmt.Sprintf("indexer:%s", idx.Implementation),
					Reason:  "not supported by service",
				})
			}
		}
		ir.Indexers.Direct = prunedIndexers
	}

	return unrealized
}

// hashInput generates a deterministic hash of the compilation input
func (c *Compiler) hashInput(input CompileInput) string {
	// Create a simplified struct for hashing (exclude resolved secrets for security)
	hashable := struct {
		App                string
		ConfigName         string
		QualityPreset      string
		QualityOverrides   *presets.QualityOverrides
		NamingPreset       string
		DownloadClients    []DownloadClientInput
		RemotePathMappings []RemotePathMappingInput
		Indexers           *IndexersInput
		RootFolders        []string
		Notifications      []NotificationInput
		CustomFormats      []CustomFormatInput
		DelayProfiles      []DelayProfileInput
	}{
		App:                input.App,
		ConfigName:         input.ConfigName,
		QualityPreset:      input.QualityPreset,
		QualityOverrides:   input.QualityOverrides,
		NamingPreset:       input.NamingPreset,
		DownloadClients:    input.DownloadClients,
		RemotePathMappings: input.RemotePathMappings,
		Indexers:           input.Indexers,
		RootFolders:        input.RootFolders,
		Notifications:      input.Notifications,
		CustomFormats:      input.CustomFormats,
		DelayProfiles:      input.DelayProfiles,
	}

	data, err := json.Marshal(hashable)
	if err != nil {
		return ""
	}

	hash := sha256.Sum256(data)
	return fmt.Sprintf("%x", hash[:8]) // Use first 8 bytes (16 hex chars)
}

// inferProtocol determines the protocol from implementation type
func inferProtocol(impl string) string {
	switch impl {
	case "qbittorrent", "QBittorrent", "transmission", "Transmission",
		"deluge", "Deluge", "rtorrent", "RTorrent", "Torznab":
		return irv1.ProtocolTorrent
	case "sabnzbd", "Sabnzbd", "nzbget", "NzbGet", "Newznab":
		return irv1.ProtocolUsenet
	default:
		return ""
	}
}

// normalizeImplementation converts implementation names to their canonical form
func normalizeImplementation(impl string) string {
	switch impl {
	case "qbittorrent":
		return "QBittorrent"
	case "transmission":
		return "Transmission"
	case "deluge":
		return "Deluge"
	case "rtorrent":
		return "RTorrent"
	case "sabnzbd":
		return "Sabnzbd"
	case "nzbget":
		return "NzbGet"
	default:
		return impl
	}
}

// compileImportListsToIR converts import list inputs to IR
func (c *Compiler) compileImportListsToIR(lists []ImportListInput) []irv1.ImportListIR {
	if len(lists) == 0 {
		return nil
	}

	result := make([]irv1.ImportListIR, 0, len(lists))
	for _, list := range lists {
		ir := irv1.ImportListIR{
			Name:               list.Name,
			Type:               list.Type,
			Enabled:            list.Enabled,
			EnableAuto:         list.EnableAuto,
			SearchOnAdd:        list.SearchOnAdd,
			QualityProfileName: list.QualityProfileName,
			RootFolderPath:     list.RootFolderPath,
			// Radarr-specific
			Monitor:             list.Monitor,
			MinimumAvailability: list.MinimumAvailability,
			// Sonarr-specific
			SeriesType:    list.SeriesType,
			SeasonFolder:  list.SeasonFolder,
			ShouldMonitor: list.ShouldMonitor,
			// Type-specific settings
			Settings: list.Settings,
		}
		result = append(result, ir)
	}
	return result
}

// compileMediaManagementToIR converts media management input to IR
func (c *Compiler) compileMediaManagementToIR(input *MediaManagementInput) *irv1.MediaManagementIR {
	if input == nil {
		return nil
	}

	return &irv1.MediaManagementIR{
		RecycleBin:             input.RecycleBin,
		RecycleBinCleanupDays:  input.RecycleBinCleanupDays,
		SetPermissions:         input.SetPermissions,
		ChmodFolder:            input.ChmodFolder,
		ChownGroup:             input.ChownGroup,
		DeleteEmptyFolders:     input.DeleteEmptyFolders,
		CreateEmptyFolders:     input.CreateEmptyFolders,
		UseHardlinks:           input.UseHardlinks,
		WatchLibraryForChanges: input.WatchLibraryForChanges,
		AllowFingerprinting:    input.AllowFingerprinting,
	}
}

// compileAuthenticationToIR converts authentication input to IR
func (c *Compiler) compileAuthenticationToIR(input *AuthenticationInput) *irv1.AuthenticationIR {
	if input == nil {
		return nil
	}

	return &irv1.AuthenticationIR{
		Method:                 input.Method,
		Username:               input.Username,
		Password:               input.Password,
		AuthenticationRequired: input.AuthenticationRequired,
	}
}

// compileNotificationsToIR converts notification inputs to IR
func (c *Compiler) compileNotificationsToIR(notifications []NotificationInput, configName string) []irv1.NotificationIR {
	if len(notifications) == 0 {
		return nil
	}

	result := make([]irv1.NotificationIR, 0, len(notifications))
	for _, n := range notifications {
		ir := irv1.NotificationIR{
			Name:           fmt.Sprintf("nebularr-%s-%s", configName, n.Name),
			Implementation: n.Implementation,
			ConfigContract: n.Implementation + "Settings",
			Enabled:        true,

			// Common event triggers
			OnGrab:                      n.OnGrab,
			OnDownload:                  n.OnDownload,
			OnUpgrade:                   n.OnUpgrade,
			OnRename:                    n.OnRename,
			OnHealthIssue:               n.OnHealthIssue,
			OnHealthRestored:            n.OnHealthRestored,
			OnApplicationUpdate:         n.OnApplicationUpdate,
			OnManualInteractionRequired: n.OnManualInteractionRequired,
			IncludeHealthWarnings:       n.IncludeHealthWarnings,

			// Radarr-specific events
			OnMovieAdded:                n.OnMovieAdded,
			OnMovieDelete:               n.OnMovieDelete,
			OnMovieFileDelete:           n.OnMovieFileDelete,
			OnMovieFileDeleteForUpgrade: n.OnMovieFileDeleteForUpgrade,

			// Sonarr-specific events
			OnSeriesAdd:                   n.OnSeriesAdd,
			OnSeriesDelete:                n.OnSeriesDelete,
			OnEpisodeFileDelete:           n.OnEpisodeFileDelete,
			OnEpisodeFileDeleteForUpgrade: n.OnEpisodeFileDeleteForUpgrade,

			// Lidarr-specific events
			OnReleaseImport:   n.OnReleaseImport,
			OnArtistAdd:       n.OnArtistAdd,
			OnArtistDelete:    n.OnArtistDelete,
			OnAlbumDelete:     n.OnAlbumDelete,
			OnTrackRetag:      n.OnTrackRetag,
			OnDownloadFailure: n.OnDownloadFailure,
			OnImportFailure:   n.OnImportFailure,

			// Type-specific settings
			Fields: n.Fields,
		}
		result = append(result, ir)
	}
	return result
}

// compileCustomFormatsToIR converts custom format inputs to IR
func (c *Compiler) compileCustomFormatsToIR(formats []CustomFormatInput, configName string) []irv1.CustomFormatIR {
	if len(formats) == 0 {
		return nil
	}

	result := make([]irv1.CustomFormatIR, 0, len(formats))
	for _, cf := range formats {
		ir := irv1.CustomFormatIR{
			Name:                fmt.Sprintf("nebularr-%s-%s", configName, cf.Name),
			IncludeWhenRenaming: cf.IncludeWhenRenaming,
			Specifications:      make([]irv1.FormatSpecIR, 0, len(cf.Specifications)),
		}

		for _, spec := range cf.Specifications {
			ir.Specifications = append(ir.Specifications, irv1.FormatSpecIR{
				Type:     spec.Type,
				Name:     spec.Name,
				Negate:   spec.Negate,
				Required: spec.Required,
				Value:    spec.Value,
			})
		}

		result = append(result, ir)
	}
	return result
}

// compileFormatScores extracts format scores from custom format inputs
// This maps custom format names to their scores for use in quality profiles
func (c *Compiler) compileFormatScores(formats []CustomFormatInput, configName string) map[string]int {
	if len(formats) == 0 {
		return nil
	}

	scores := make(map[string]int)
	for _, cf := range formats {
		// Only include formats with non-zero scores
		if cf.Score != 0 {
			// Use the full name (with nebularr prefix) to match the custom format name
			fullName := fmt.Sprintf("nebularr-%s-%s", configName, cf.Name)
			scores[fullName] = cf.Score
		}
	}

	if len(scores) == 0 {
		return nil
	}
	return scores
}

// compileDelayProfilesToIR converts delay profile inputs to IR
func (c *Compiler) compileDelayProfilesToIR(profiles []DelayProfileInput) []irv1.DelayProfileIR {
	if len(profiles) == 0 {
		return nil
	}

	result := make([]irv1.DelayProfileIR, 0, len(profiles))
	for i, p := range profiles {
		// Default order based on position if not specified
		order := p.Order
		if order == 0 {
			order = i + 1
		}

		// Default protocol
		preferredProtocol := p.PreferredProtocol
		if preferredProtocol == "" {
			preferredProtocol = irv1.ProtocolUsenet
		}

		ir := irv1.DelayProfileIR{
			Name:                           p.Name,
			Order:                          order,
			PreferredProtocol:              preferredProtocol,
			UsenetDelay:                    p.UsenetDelay,
			TorrentDelay:                   p.TorrentDelay,
			EnableUsenet:                   p.EnableUsenet,
			EnableTorrent:                  p.EnableTorrent,
			BypassIfHighestQuality:         p.BypassIfHighestQuality,
			BypassIfAboveCustomFormatScore: p.BypassIfAboveCustomFormatScore,
			MinimumCustomFormatScore:       p.MinimumCustomFormatScore,
			TagNames:                       p.Tags,
		}
		result = append(result, ir)
	}

	return result
}
