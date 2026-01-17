package presets

import (
	irv1 "github.com/poiley/nebularr-operator/internal/ir/v1"
)

// Expander handles preset expansion to IR types
type Expander struct{}

// NewExpander creates a new preset expander
func NewExpander() *Expander {
	return &Expander{}
}

// ExpandVideoPreset expands a video preset (optionally with overrides) to VideoQualityIR
func (e *Expander) ExpandVideoPreset(presetName string, overrides *QualityOverrides, profileName string) *irv1.VideoQualityIR {
	preset, ok := GetVideoPreset(presetName)
	if !ok {
		// Fall back to default preset
		preset = VideoPresets[DefaultVideoPreset]
	}

	// Apply overrides if provided
	if overrides != nil {
		preset = ApplyVideoOverrides(preset, *overrides)
	}

	ir := &irv1.VideoQualityIR{
		ProfileName:    profileName,
		UpgradeAllowed: true,
		Tiers:          make([]irv1.VideoQualityTierIR, 0, len(preset.Tiers)),
		FormatScores:   make(map[string]int),
	}

	// Convert tiers
	for _, tier := range preset.Tiers {
		irTier := irv1.VideoQualityTierIR{
			Resolution: tier.Resolution,
			Sources:    tier.Sources,
			Allowed:    true,
		}
		ir.Tiers = append(ir.Tiers, irTier)
	}

	// Set cutoff from UpgradeUntil
	if preset.UpgradeUntil != nil {
		ir.Cutoff = irv1.VideoQualityTierIR{
			Resolution: preset.UpgradeUntil.Resolution,
			Sources:    preset.UpgradeUntil.Sources,
			Allowed:    true,
		}
	}

	// Convert preferred formats to custom formats with positive scores
	for _, format := range preset.PreferredFormats {
		cf := e.formatToCustomFormat(format, false)
		if cf != nil {
			ir.CustomFormats = append(ir.CustomFormats, *cf)
			ir.FormatScores[cf.Name] = 100 // Positive score for preferred
		}
	}

	// Convert rejected formats to custom formats with negative scores
	for _, format := range preset.RejectFormats {
		cf := e.formatToCustomFormat(format, true)
		if cf != nil {
			ir.CustomFormats = append(ir.CustomFormats, *cf)
			ir.FormatScores[cf.Name] = -10000 // Large negative score to reject
		}
	}

	return ir
}

// ExpandAudioPreset expands an audio preset (optionally with overrides) to AudioQualityIR
func (e *Expander) ExpandAudioPreset(presetName string, overrides *QualityOverrides, profileName string) *irv1.AudioQualityIR {
	preset, ok := GetAudioPreset(presetName)
	if !ok {
		// Fall back to default preset
		preset = AudioPresets[DefaultAudioPreset]
	}

	// Apply overrides if provided
	if overrides != nil {
		preset = ApplyAudioOverrides(preset, *overrides)
	}

	ir := &irv1.AudioQualityIR{
		ProfileName:    profileName,
		UpgradeAllowed: true,
		Cutoff:         preset.UpgradeUntil,
		Tiers:          make([]irv1.AudioQualityTierIR, 0, len(preset.Tiers)),
	}

	// Convert tiers
	rejectSet := make(map[string]bool)
	for _, tier := range preset.RejectTiers {
		rejectSet[tier] = true
	}

	for _, tier := range preset.Tiers {
		irTier := irv1.AudioQualityTierIR{
			Tier:    tier,
			Allowed: !rejectSet[tier],
		}
		ir.Tiers = append(ir.Tiers, irTier)
	}

	return ir
}

// ExpandRadarrNaming expands a naming preset to RadarrNamingIR
func (e *Expander) ExpandRadarrNaming(presetName string) *irv1.RadarrNamingIR {
	expansion := GetRadarrNaming(presetName)
	return &irv1.RadarrNamingIR{
		RenameMovies:             expansion.RenameMovies,
		ReplaceIllegalCharacters: expansion.ReplaceIllegalCharacters,
		ColonReplacementFormat:   expansion.ColonReplacement,
		StandardMovieFormat:      expansion.StandardMovieFormat,
		MovieFolderFormat:        expansion.MovieFolderFormat,
	}
}

// ExpandSonarrNaming expands a naming preset to SonarrNamingIR
func (e *Expander) ExpandSonarrNaming(presetName string) *irv1.SonarrNamingIR {
	expansion := GetSonarrNaming(presetName)
	return &irv1.SonarrNamingIR{
		RenameEpisodes:           expansion.RenameEpisodes,
		ReplaceIllegalCharacters: expansion.ReplaceIllegalCharacters,
		ColonReplacementFormat:   expansion.ColonReplacement,
		StandardEpisodeFormat:    expansion.StandardEpisodeFormat,
		DailyEpisodeFormat:       expansion.DailyEpisodeFormat,
		AnimeEpisodeFormat:       expansion.AnimeEpisodeFormat,
		SeriesFolderFormat:       expansion.SeriesFolderFormat,
		SeasonFolderFormat:       expansion.SeasonFolderFormat,
		SpecialsFolderFormat:     expansion.SpecialsFolderFormat,
		MultiEpisodeStyle:        expansion.MultiEpisodeStyle,
	}
}

// ExpandLidarrNaming expands a naming preset to LidarrNamingIR
func (e *Expander) ExpandLidarrNaming(presetName string) *irv1.LidarrNamingIR {
	expansion := GetLidarrNaming(presetName)
	return &irv1.LidarrNamingIR{
		RenameTracks:             expansion.RenameTracks,
		ReplaceIllegalCharacters: expansion.ReplaceIllegalCharacters,
		ColonReplacementFormat:   expansion.ColonReplacement,
		StandardTrackFormat:      expansion.StandardTrackFormat,
		MultiDiscTrackFormat:     expansion.MultiDiscTrackFormat,
		ArtistFolderFormat:       expansion.ArtistFolderFormat,
		AlbumFolderFormat:        expansion.AlbumFolderFormat,
	}
}

// formatToCustomFormat converts a format name to a CustomFormatIR
// This creates the custom format definitions that will be applied in Radarr/Sonarr
func (e *Expander) formatToCustomFormat(format string, isReject bool) *irv1.CustomFormatIR {
	// Map format names to custom format specifications
	specs := e.getFormatSpecs(format)
	if len(specs) == 0 {
		return nil
	}

	var name string
	if isReject {
		name = "Reject: " + format
	} else {
		name = "Prefer: " + format
	}

	return &irv1.CustomFormatIR{
		Name:                name,
		IncludeWhenRenaming: false,
		Specifications:      specs,
	}
}

// getFormatSpecs returns the specifications for a format name
func (e *Expander) getFormatSpecs(format string) []irv1.FormatSpecIR {
	// Map of format names to their regex patterns
	formatPatterns := map[string]string{
		// HDR formats
		"hdr10":        `\bHDR10\b`,
		"hdr10plus":    `\bHDR10\+|HDR10Plus\b`,
		"dolby-vision": `\b(DV|DoVi|Dolby[\.\s]?Vision)\b`,
		"hlg":          `\bHLG\b`,

		// Audio formats
		"atmos":  `\b(Atmos|ATMOS)\b`,
		"truehd": `\b(TrueHD|True[\.\s]?HD)\b`,
		"dts-x":  `\b(DTS[\.\-\s]?X)\b`,
		"dts-hd": `\b(DTS[\.\-\s]?(HD[\.\-\s]?)?(MA)?)\b`,

		// Video codecs
		"hevc": `\b(HEVC|[xh][\.\s]?265)\b`,
		"av1":  `\bAV1\b`,

		// Audio codecs
		"aac": `\bAAC\b`,

		// Unwanted
		"cam":       `\b(CAM|CAMRIP|CAM[\.\-\s]?RIP)\b`,
		"telesync":  `\b(TS|TELESYNC|TELE[\.\-\s]?SYNC)\b`,
		"telecine":  `\b(TC|TELECINE|TELE[\.\-\s]?CINE)\b`,
		"workprint": `\b(WP|WORKPRINT|WORK[\.\-\s]?PRINT)\b`,
		"3d":        `\b3D\b`,
		"dubbed":    `\b(DUBBED|DUB)\b`,

		// Extras
		"remux":    `\b(REMUX|Remux)\b`,
		"imax":     `\bIMAX\b`,
		"extended": `\b(EXTENDED|Extended)\b`,
	}

	pattern, ok := formatPatterns[format]
	if !ok {
		return nil
	}

	return []irv1.FormatSpecIR{
		{
			Type:     "ReleaseTitleSpecification",
			Name:     format,
			Negate:   false,
			Required: true,
			Value:    pattern,
		},
	}
}
