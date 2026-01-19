package compiler

import (
	"fmt"

	irv1 "github.com/poiley/nebularr-operator/internal/ir/v1"
)

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
