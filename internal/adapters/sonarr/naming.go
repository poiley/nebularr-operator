package sonarr

import (
	"context"

	"github.com/poiley/nebularr-operator/internal/adapters"
	irv1 "github.com/poiley/nebularr-operator/internal/ir/v1"
)

// namingConfigID stores the ID for the naming config (always 1 in Sonarr)
var namingConfigID = 1

// getNamingConfig retrieves the naming configuration
func (a *Adapter) getNamingConfig(ctx context.Context, c *httpClient) (*irv1.SonarrNamingIR, error) {
	var naming NamingConfigResource
	if err := c.get(ctx, "/api/v3/config/naming", &naming); err != nil {
		return nil, err
	}

	namingConfigID = naming.ID

	return &irv1.SonarrNamingIR{
		RenameEpisodes:           naming.RenameEpisodes,
		ReplaceIllegalCharacters: naming.ReplaceIllegalCharacters,
		StandardEpisodeFormat:    naming.StandardEpisodeFormat,
		DailyEpisodeFormat:       naming.DailyEpisodeFormat,
		AnimeEpisodeFormat:       naming.AnimeEpisodeFormat,
		SeriesFolderFormat:       naming.SeriesFolderFormat,
		SeasonFolderFormat:       naming.SeasonFolderFormat,
		SpecialsFolderFormat:     naming.SpecialsFolderFormat,
		MultiEpisodeStyle:        naming.MultiEpisodeStyle,
	}, nil
}

// diffNaming computes changes needed for naming config
func (a *Adapter) diffNaming(current, desired *irv1.IR, changes *adapters.ChangeSet) error {
	var currentNaming *irv1.SonarrNamingIR
	var desiredNaming *irv1.SonarrNamingIR

	if current.Naming != nil {
		currentNaming = current.Naming.Sonarr
	}
	if desired.Naming != nil {
		desiredNaming = desired.Naming.Sonarr
	}

	// No desired naming config - nothing to do
	if desiredNaming == nil {
		return nil
	}

	// Naming config always exists in Sonarr (ID=1), so we just update it
	if namingNeedsUpdate(currentNaming, desiredNaming) {
		changes.Updates = append(changes.Updates, adapters.Change{
			ResourceType: adapters.ResourceNamingConfig,
			Name:         "naming",
			ID:           &namingConfigID,
			Payload:      desiredNaming,
		})
	}

	return nil
}

// namingNeedsUpdate checks if naming config needs updating
func namingNeedsUpdate(current, desired *irv1.SonarrNamingIR) bool {
	if current == nil {
		return true
	}
	if current.RenameEpisodes != desired.RenameEpisodes {
		return true
	}
	if current.StandardEpisodeFormat != desired.StandardEpisodeFormat {
		return true
	}
	if current.DailyEpisodeFormat != desired.DailyEpisodeFormat {
		return true
	}
	if current.AnimeEpisodeFormat != desired.AnimeEpisodeFormat {
		return true
	}
	if current.SeriesFolderFormat != desired.SeriesFolderFormat {
		return true
	}
	if current.SeasonFolderFormat != desired.SeasonFolderFormat {
		return true
	}
	return false
}
