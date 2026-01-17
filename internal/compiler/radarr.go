package compiler

import (
	"context"
	"net/url"
	"strconv"
	"strings"

	arrv1alpha1 "github.com/poiley/nebularr-operator/api/v1alpha1"
	"github.com/poiley/nebularr-operator/internal/adapters"
	irv1 "github.com/poiley/nebularr-operator/internal/ir/v1"
	"github.com/poiley/nebularr-operator/internal/presets"
)

// CompileRadarrConfig compiles a RadarrConfig CRD to IR
func (c *Compiler) CompileRadarrConfig(ctx context.Context, config *arrv1alpha1.RadarrConfig, resolvedSecrets map[string]string, caps *adapters.Capabilities) (*irv1.IR, error) {
	input := CompileInput{
		App:             adapters.AppRadarr,
		ConfigName:      config.Name,
		Namespace:       config.Namespace,
		Capabilities:    caps,
		ResolvedSecrets: resolvedSecrets,
	}

	// Connection (required field, always present)
	input.URL = config.Spec.Connection.URL
	// API key should be resolved from secrets
	if apiKey, ok := resolvedSecrets["apiKey"]; ok {
		input.APIKey = apiKey
	}

	// Quality
	if config.Spec.Quality != nil {
		input.QualityPreset = config.Spec.Quality.Preset
		if len(config.Spec.Quality.Exclude) > 0 || len(config.Spec.Quality.PreferAdditional) > 0 || len(config.Spec.Quality.RejectAdditional) > 0 {
			input.QualityOverrides = &presets.QualityOverrides{
				Exclude:          config.Spec.Quality.Exclude,
				PreferAdditional: config.Spec.Quality.PreferAdditional,
				RejectAdditional: config.Spec.Quality.RejectAdditional,
			}
		}
	}

	// Naming
	if config.Spec.Naming != nil {
		input.NamingPreset = config.Spec.Naming.Preset
	}

	// Download clients
	input.DownloadClients = convertDownloadClients(config.Spec.DownloadClients, resolvedSecrets)

	// Remote path mappings
	input.RemotePathMappings = convertRemotePathMappings(config.Spec.RemotePathMappings)

	// Indexers
	input.Indexers = convertIndexers(config.Spec.Indexers, resolvedSecrets)

	// Root folders
	input.RootFolders = config.Spec.RootFolders

	// Import lists
	input.ImportLists = convertImportLists(config.Spec.ImportLists, resolvedSecrets)

	// Media management
	input.MediaManagement = convertMediaManagement(config.Spec.MediaManagement)

	// Authentication
	input.Authentication = convertAuthentication(config.Spec.Authentication, resolvedSecrets)

	// Notifications
	input.Notifications = convertNotifications(config.Spec.Notifications, resolvedSecrets)

	// Custom formats
	input.CustomFormats = convertCustomFormats(config.Spec.CustomFormats)

	// Delay profiles
	input.DelayProfiles = convertDelayProfiles(config.Spec.DelayProfiles)

	return c.Compile(ctx, input)
}

// CompileSonarrConfig compiles a SonarrConfig CRD to IR
func (c *Compiler) CompileSonarrConfig(ctx context.Context, config *arrv1alpha1.SonarrConfig, resolvedSecrets map[string]string, caps *adapters.Capabilities) (*irv1.IR, error) {
	input := CompileInput{
		App:             adapters.AppSonarr,
		ConfigName:      config.Name,
		Namespace:       config.Namespace,
		Capabilities:    caps,
		ResolvedSecrets: resolvedSecrets,
	}

	// Connection (required field, always present)
	input.URL = config.Spec.Connection.URL
	if apiKey, ok := resolvedSecrets["apiKey"]; ok {
		input.APIKey = apiKey
	}

	// Quality
	if config.Spec.Quality != nil {
		input.QualityPreset = config.Spec.Quality.Preset
		if len(config.Spec.Quality.Exclude) > 0 || len(config.Spec.Quality.PreferAdditional) > 0 || len(config.Spec.Quality.RejectAdditional) > 0 {
			input.QualityOverrides = &presets.QualityOverrides{
				Exclude:          config.Spec.Quality.Exclude,
				PreferAdditional: config.Spec.Quality.PreferAdditional,
				RejectAdditional: config.Spec.Quality.RejectAdditional,
			}
		}
	}

	// Naming
	if config.Spec.Naming != nil {
		input.NamingPreset = config.Spec.Naming.Preset
	}

	// Download clients
	input.DownloadClients = convertDownloadClients(config.Spec.DownloadClients, resolvedSecrets)

	// Remote path mappings
	input.RemotePathMappings = convertRemotePathMappings(config.Spec.RemotePathMappings)

	// Indexers
	input.Indexers = convertIndexers(config.Spec.Indexers, resolvedSecrets)

	// Root folders
	input.RootFolders = config.Spec.RootFolders

	// Import lists
	input.ImportLists = convertImportLists(config.Spec.ImportLists, resolvedSecrets)

	// Media management
	input.MediaManagement = convertMediaManagement(config.Spec.MediaManagement)

	// Authentication
	input.Authentication = convertAuthentication(config.Spec.Authentication, resolvedSecrets)

	// Notifications
	input.Notifications = convertNotifications(config.Spec.Notifications, resolvedSecrets)

	// Custom formats
	input.CustomFormats = convertCustomFormats(config.Spec.CustomFormats)

	// Delay profiles
	input.DelayProfiles = convertDelayProfiles(config.Spec.DelayProfiles)

	return c.Compile(ctx, input)
}

// CompileLidarrConfig compiles a LidarrConfig CRD to IR
func (c *Compiler) CompileLidarrConfig(ctx context.Context, config *arrv1alpha1.LidarrConfig, resolvedSecrets map[string]string, caps *adapters.Capabilities) (*irv1.IR, error) {
	input := CompileInput{
		App:             adapters.AppLidarr,
		ConfigName:      config.Name,
		Namespace:       config.Namespace,
		Capabilities:    caps,
		ResolvedSecrets: resolvedSecrets,
	}

	// Connection (required field, always present)
	input.URL = config.Spec.Connection.URL
	if apiKey, ok := resolvedSecrets["apiKey"]; ok {
		input.APIKey = apiKey
	}

	// Quality (audio)
	if config.Spec.Quality != nil {
		input.QualityPreset = config.Spec.Quality.Preset
		if len(config.Spec.Quality.Exclude) > 0 || len(config.Spec.Quality.PreferAdditional) > 0 {
			input.QualityOverrides = &presets.QualityOverrides{
				Exclude:          config.Spec.Quality.Exclude,
				PreferAdditional: config.Spec.Quality.PreferAdditional,
			}
		}
	}

	// Naming
	if config.Spec.Naming != nil {
		input.NamingPreset = config.Spec.Naming.Preset
	}

	// Download clients
	input.DownloadClients = convertDownloadClients(config.Spec.DownloadClients, resolvedSecrets)

	// Remote path mappings
	input.RemotePathMappings = convertRemotePathMappings(config.Spec.RemotePathMappings)

	// Indexers
	input.Indexers = convertIndexers(config.Spec.Indexers, resolvedSecrets)

	// Root folders - Lidarr has a different structure with LidarrRootFolder
	for _, rf := range config.Spec.RootFolders {
		input.RootFolders = append(input.RootFolders, rf.Path)
	}

	// Import lists
	input.ImportLists = convertImportLists(config.Spec.ImportLists, resolvedSecrets)

	// Media management
	input.MediaManagement = convertMediaManagement(config.Spec.MediaManagement)

	// Authentication
	input.Authentication = convertAuthentication(config.Spec.Authentication, resolvedSecrets)

	// Notifications
	input.Notifications = convertNotifications(config.Spec.Notifications, resolvedSecrets)

	return c.Compile(ctx, input)
}

// convertRemotePathMappings converts CRD RemotePathMappingSpec to compiler input
func convertRemotePathMappings(mappings []arrv1alpha1.RemotePathMappingSpec) []RemotePathMappingInput {
	if len(mappings) == 0 {
		return nil
	}

	result := make([]RemotePathMappingInput, 0, len(mappings))
	for _, m := range mappings {
		result = append(result, RemotePathMappingInput{
			Host:       m.Host,
			RemotePath: m.RemotePath,
			LocalPath:  m.LocalPath,
		})
	}
	return result
}

// convertDownloadClients converts CRD DownloadClientSpec to compiler input
func convertDownloadClients(clients []arrv1alpha1.DownloadClientSpec, resolvedSecrets map[string]string) []DownloadClientInput {
	result := make([]DownloadClientInput, 0, len(clients))

	for _, dc := range clients {
		// Parse URL to extract host, port, and TLS setting
		host, port, useTLS := parseClientURL(dc.URL)

		// Determine implementation from Type field (or infer from Name)
		impl := dc.Type
		if impl == "" {
			impl = inferImplementationFromName(dc.Name)
		}

		dcInput := DownloadClientInput{
			Name:           dc.Name,
			Implementation: normalizeImplementationName(impl),
			Host:           host,
			Port:           port,
			UseTLS:         useTLS,
			Category:       dc.Category,
			Priority:       dc.Priority,
		}

		// Resolve credentials from secrets
		if dc.CredentialsSecretRef != nil {
			usernameKey := dc.CredentialsSecretRef.UsernameKey
			if usernameKey == "" {
				usernameKey = "username"
			}
			passwordKey := dc.CredentialsSecretRef.PasswordKey
			if passwordKey == "" {
				passwordKey = "password"
			}

			secretPrefix := dc.CredentialsSecretRef.Name + "/"
			if username, ok := resolvedSecrets[secretPrefix+usernameKey]; ok {
				dcInput.Username = username
			}
			if password, ok := resolvedSecrets[secretPrefix+passwordKey]; ok {
				dcInput.Password = password
			}
		}

		result = append(result, dcInput)
	}

	return result
}

// convertIndexers converts CRD IndexersSpec to compiler input
func convertIndexers(spec *arrv1alpha1.IndexersSpec, resolvedSecrets map[string]string) *IndexersInput {
	if spec == nil {
		return nil
	}

	result := &IndexersInput{}

	// Handle Prowlarr reference
	if spec.ProwlarrRef != nil {
		autoRegister := true
		if spec.ProwlarrRef.AutoRegister != nil {
			autoRegister = *spec.ProwlarrRef.AutoRegister
		}
		result.ProwlarrRef = &ProwlarrRefInput{
			ConfigName:   spec.ProwlarrRef.Name, // CRD uses "Name", not "ConfigName"
			AutoRegister: autoRegister,
			Include:      spec.ProwlarrRef.Include,
			Exclude:      spec.ProwlarrRef.Exclude,
		}
		return result
	}

	// Handle direct indexers
	for _, idx := range spec.Direct {
		// Convert string categories to int (if they're numeric) or use category mapping
		categories := convertCategories(idx.Categories)

		// Determine protocol from Type field
		protocol := irv1.ProtocolTorrent
		if idx.Type == "usenet" {
			protocol = irv1.ProtocolUsenet
		}

		// Infer implementation from URL or type
		impl := inferIndexerImplementation(idx.URL, idx.Type)

		idxInput := IndexerInput{
			Name:                    idx.Name,
			Protocol:                protocol,
			Implementation:          impl,
			URL:                     idx.URL,
			Categories:              categories,
			Priority:                idx.Priority,
			EnableRss:               true, // Default to enabled
			EnableAutomaticSearch:   true,
			EnableInteractiveSearch: true,
		}

		// Resolve API key from secret
		if idx.APIKeySecretRef != nil {
			keyName := idx.APIKeySecretRef.Key
			if keyName == "" {
				keyName = "apiKey"
			}
			secretKey := idx.APIKeySecretRef.Name + "/" + keyName
			if apiKey, ok := resolvedSecrets[secretKey]; ok {
				idxInput.APIKey = apiKey
			}
		}

		result.Direct = append(result.Direct, idxInput)
	}

	return result
}

// parseClientURL parses a download client URL into host, port, and TLS setting
func parseClientURL(rawURL string) (host string, port int, useTLS bool) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL, 0, false
	}

	host = parsed.Hostname()
	useTLS = parsed.Scheme == "https"

	// Parse port
	portStr := parsed.Port()
	if portStr != "" {
		port, _ = strconv.Atoi(portStr)
	} else {
		// Default ports based on scheme
		if useTLS {
			port = 443
		} else {
			port = 80
		}
	}

	return host, port, useTLS
}

// inferImplementationFromName tries to determine the download client type from its name
func inferImplementationFromName(name string) string {
	nameLower := strings.ToLower(name)

	switch {
	case strings.Contains(nameLower, "qbit") || strings.Contains(nameLower, "qbittorrent"):
		return "qbittorrent"
	case strings.Contains(nameLower, "transmission"):
		return "transmission"
	case strings.Contains(nameLower, "deluge"):
		return "deluge"
	case strings.Contains(nameLower, "rtorrent") || strings.Contains(nameLower, "rutorrent"):
		return "rtorrent"
	case strings.Contains(nameLower, "sabnzbd") || strings.Contains(nameLower, "sab"):
		return "sabnzbd"
	case strings.Contains(nameLower, "nzbget"):
		return "nzbget"
	default:
		return "qbittorrent" // Default fallback
	}
}

// normalizeImplementationName converts user-friendly type names to API implementation names
func normalizeImplementationName(impl string) string {
	switch strings.ToLower(impl) {
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

// inferIndexerImplementation determines the indexer implementation from URL and type
func inferIndexerImplementation(_ string, indexerType string) string {
	if indexerType == "usenet" {
		return "Newznab"
	}
	return "Torznab"
}

// convertCategories converts string category names to numeric IDs
// Supports both numeric strings and human-readable names
func convertCategories(categories []string) []int {
	result := make([]int, 0, len(categories))

	for _, cat := range categories {
		// Try parsing as integer first
		if id, err := strconv.Atoi(cat); err == nil {
			result = append(result, id)
			continue
		}

		// Map human-readable names to IDs (Newznab/Torznab categories)
		id := mapCategoryName(cat)
		if id > 0 {
			result = append(result, id)
		}
	}

	return result
}

// mapCategoryName maps human-readable category names to Newznab/Torznab IDs
func mapCategoryName(name string) int {
	categories := map[string]int{
		// Movies
		"movies":        2000,
		"movies-sd":     2030,
		"movies-hd":     2040,
		"movies-uhd":    2045,
		"movies-4k":     2045,
		"movies-bluray": 2050,
		"movies-3d":     2060,

		// TV
		"tv":     5000,
		"tv-sd":  5030,
		"tv-hd":  5040,
		"tv-uhd": 5045,
		"tv-4k":  5045,

		// Audio/Music
		"audio":          3000,
		"music":          3000,
		"audio-mp3":      3010,
		"audio-flac":     3040,
		"audio-lossless": 3040,

		// Other common categories
		"xxx":   6000,
		"other": 7000,
	}

	return categories[strings.ToLower(name)]
}

// convertImportLists converts CRD ImportListSpec to compiler input
func convertImportLists(lists []arrv1alpha1.ImportListSpec, resolvedSecrets map[string]string) []ImportListInput {
	if len(lists) == 0 {
		return nil
	}

	result := make([]ImportListInput, 0, len(lists))
	for _, list := range lists {
		input := ImportListInput{
			Name:               list.Name,
			Type:               list.Type,
			Enabled:            ptrBoolOrDefault(list.Enabled, true),
			EnableAuto:         ptrBoolOrDefault(list.EnableAuto, true),
			SearchOnAdd:        ptrBoolOrDefault(list.SearchOnAdd, true),
			QualityProfileName: list.QualityProfile,
			RootFolderPath:     list.RootFolder,
			// Radarr-specific
			Monitor:             defaultString(list.Monitor, "movieOnly"),
			MinimumAvailability: defaultString(list.MinimumAvailability, "announced"),
			// Sonarr-specific
			SeriesType:    defaultString(list.SeriesType, "standard"),
			SeasonFolder:  ptrBoolOrDefault(list.SeasonFolder, true),
			ShouldMonitor: defaultString(list.ShouldMonitor, "all"),
			// Copy settings
			Settings: make(map[string]string),
		}

		// Copy settings from spec
		for k, v := range list.Settings {
			input.Settings[k] = v
		}

		// Resolve settings from secret if specified
		if list.SettingsSecretRef != nil {
			secretPrefix := list.SettingsSecretRef.Name + "/"
			// Look for all resolved secrets with this prefix and add to settings
			for key, value := range resolvedSecrets {
				if strings.HasPrefix(key, secretPrefix) {
					settingKey := strings.TrimPrefix(key, secretPrefix)
					input.Settings[settingKey] = value
				}
			}
		}

		result = append(result, input)
	}

	return result
}

// convertMediaManagement converts CRD MediaManagementSpec to compiler input
func convertMediaManagement(spec *arrv1alpha1.MediaManagementSpec) *MediaManagementInput {
	if spec == nil {
		return nil
	}

	return &MediaManagementInput{
		RecycleBin:             spec.RecycleBin,
		RecycleBinCleanupDays:  ptrIntOrDefault(spec.RecycleBinCleanupDays, 7),
		SetPermissions:         ptrBoolOrDefault(spec.SetPermissions, false),
		ChmodFolder:            defaultString(spec.ChmodFolder, "755"),
		ChownGroup:             spec.ChownGroup,
		DeleteEmptyFolders:     ptrBoolOrDefault(spec.DeleteEmptyFolders, false),
		CreateEmptyFolders:     ptrBoolOrDefault(spec.CreateEmptyFolders, false),
		UseHardlinks:           ptrBoolOrDefault(spec.UseHardlinks, true),
		WatchLibraryForChanges: spec.WatchLibraryForChanges,
		AllowFingerprinting:    spec.AllowFingerprinting,
	}
}

// convertAuthentication converts CRD AuthenticationSpec to compiler input
func convertAuthentication(spec *arrv1alpha1.AuthenticationSpec, resolvedSecrets map[string]string) *AuthenticationInput {
	if spec == nil {
		return nil
	}

	input := &AuthenticationInput{
		Method:                 defaultString(spec.Method, "none"),
		Username:               spec.Username,
		AuthenticationRequired: defaultString(spec.AuthenticationRequired, "enabled"),
	}

	// Resolve password from secret if specified
	if spec.PasswordSecretRef != nil {
		keyName := spec.PasswordSecretRef.Key
		if keyName == "" {
			keyName = "password"
		}
		secretKey := spec.PasswordSecretRef.Name + "/" + keyName
		if password, ok := resolvedSecrets[secretKey]; ok {
			input.Password = password
		}
	}

	return input
}

// Helper functions for handling pointers and defaults

func ptrBoolOrDefault(ptr *bool, def bool) bool {
	if ptr != nil {
		return *ptr
	}
	return def
}

func ptrIntOrDefault(ptr *int, def int) int {
	if ptr != nil {
		return *ptr
	}
	return def
}

func defaultString(s string, def string) string {
	if s == "" {
		return def
	}
	return s
}

// convertNotifications converts CRD NotificationSpec to compiler input
func convertNotifications(notifications []arrv1alpha1.NotificationSpec, resolvedSecrets map[string]string) []NotificationInput {
	if len(notifications) == 0 {
		return nil
	}

	result := make([]NotificationInput, 0, len(notifications))
	for _, n := range notifications {
		input := NotificationInput{
			Name:           n.Name,
			Implementation: n.Type,

			// Common event triggers
			OnGrab:                      ptrBoolOrDefault(n.OnGrab, false),
			OnDownload:                  ptrBoolOrDefault(n.OnDownload, false),
			OnUpgrade:                   ptrBoolOrDefault(n.OnUpgrade, false),
			OnRename:                    ptrBoolOrDefault(n.OnRename, false),
			OnHealthIssue:               ptrBoolOrDefault(n.OnHealthIssue, false),
			OnHealthRestored:            ptrBoolOrDefault(n.OnHealthRestored, false),
			OnApplicationUpdate:         ptrBoolOrDefault(n.OnApplicationUpdate, false),
			OnManualInteractionRequired: ptrBoolOrDefault(n.OnManualInteractionRequired, false),
			IncludeHealthWarnings:       ptrBoolOrDefault(n.IncludeHealthWarnings, false),

			// Radarr-specific events
			OnMovieAdded:                ptrBoolOrDefault(n.OnMovieAdded, false),
			OnMovieDelete:               ptrBoolOrDefault(n.OnMovieDelete, false),
			OnMovieFileDelete:           ptrBoolOrDefault(n.OnMovieFileDelete, false),
			OnMovieFileDeleteForUpgrade: ptrBoolOrDefault(n.OnMovieFileDeleteForUpgrade, false),

			// Sonarr-specific events
			OnSeriesAdd:                   ptrBoolOrDefault(n.OnSeriesAdd, false),
			OnSeriesDelete:                ptrBoolOrDefault(n.OnSeriesDelete, false),
			OnEpisodeFileDelete:           ptrBoolOrDefault(n.OnEpisodeFileDelete, false),
			OnEpisodeFileDeleteForUpgrade: ptrBoolOrDefault(n.OnEpisodeFileDeleteForUpgrade, false),

			// Lidarr-specific events
			OnReleaseImport:   ptrBoolOrDefault(n.OnReleaseImport, false),
			OnArtistAdd:       ptrBoolOrDefault(n.OnArtistAdd, false),
			OnArtistDelete:    ptrBoolOrDefault(n.OnArtistDelete, false),
			OnAlbumDelete:     ptrBoolOrDefault(n.OnAlbumDelete, false),
			OnTrackRetag:      ptrBoolOrDefault(n.OnTrackRetag, false),
			OnDownloadFailure: ptrBoolOrDefault(n.OnDownloadFailure, false),
			OnImportFailure:   ptrBoolOrDefault(n.OnImportFailure, false),

			// Tags
			Tags: n.Tags,

			// Fields from settings
			Fields: make(map[string]interface{}),
		}

		// Copy settings to fields
		for k, v := range n.Settings {
			input.Fields[k] = v
		}

		// Resolve settings from secret if specified
		if n.SettingsSecretRef != nil {
			secretPrefix := n.SettingsSecretRef.Name + "/"
			for key, value := range resolvedSecrets {
				if strings.HasPrefix(key, secretPrefix) {
					settingKey := strings.TrimPrefix(key, secretPrefix)
					input.Fields[settingKey] = value
				}
			}
		}

		result = append(result, input)
	}

	return result
}

// convertCustomFormats converts CRD CustomFormatSpec to compiler input
func convertCustomFormats(customFormats []arrv1alpha1.CustomFormatSpec) []CustomFormatInput {
	if len(customFormats) == 0 {
		return nil
	}

	result := make([]CustomFormatInput, 0, len(customFormats))
	for _, cf := range customFormats {
		input := CustomFormatInput{
			Name:                cf.Name,
			IncludeWhenRenaming: ptrBoolOrDefault(cf.IncludeWhenRenaming, false),
			Score:               cf.Score,
			Specifications:      make([]CustomFormatSpecInput, 0, len(cf.Specifications)),
		}

		for _, spec := range cf.Specifications {
			input.Specifications = append(input.Specifications, CustomFormatSpecInput{
				Name:     spec.Name,
				Type:     spec.Type,
				Negate:   ptrBoolOrDefault(spec.Negate, false),
				Required: ptrBoolOrDefault(spec.Required, false),
				Value:    spec.Value,
			})
		}

		result = append(result, input)
	}

	return result
}

// convertDelayProfiles converts CRD DelayProfileSpec to compiler input
func convertDelayProfiles(profiles []arrv1alpha1.DelayProfileSpec) []DelayProfileInput {
	if len(profiles) == 0 {
		return nil
	}

	result := make([]DelayProfileInput, 0, len(profiles))
	for i, p := range profiles {
		// Default order based on position if not specified
		order := 0
		if p.Order != nil {
			order = *p.Order
		} else {
			order = i + 1
		}

		input := DelayProfileInput{
			Name:                           p.Name,
			PreferredProtocol:              p.PreferredProtocol,
			UsenetDelay:                    p.UsenetDelay,
			TorrentDelay:                   p.TorrentDelay,
			EnableUsenet:                   ptrBoolOrDefault(p.EnableUsenet, true),
			EnableTorrent:                  ptrBoolOrDefault(p.EnableTorrent, true),
			BypassIfHighestQuality:         ptrBoolOrDefault(p.BypassIfHighestQuality, false),
			BypassIfAboveCustomFormatScore: ptrBoolOrDefault(p.BypassIfAboveCustomFormatScore, false),
			MinimumCustomFormatScore:       p.MinimumCustomFormatScore,
			Tags:                           p.Tags,
			Order:                          order,
		}
		result = append(result, input)
	}

	return result
}
