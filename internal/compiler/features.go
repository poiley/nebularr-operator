package compiler

import (
	"fmt"

	irv1 "github.com/poiley/nebularr-operator/internal/ir/v1"
)

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
