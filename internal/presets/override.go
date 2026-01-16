package presets

// QualityOverrides defines modifications to a preset
type QualityOverrides struct {
	Exclude          []string `json:"exclude,omitempty"`
	PreferAdditional []string `json:"preferAdditional,omitempty"`
	RejectAdditional []string `json:"rejectAdditional,omitempty"`
}

// ApplyVideoOverrides applies overrides to a video preset
func ApplyVideoOverrides(preset VideoQualityPreset, overrides QualityOverrides) VideoQualityPreset {
	result := VideoQualityPreset{
		Name:             preset.Name,
		Description:      preset.Description,
		UpgradeUntil:     preset.UpgradeUntil,
		Tiers:            make([]QualityTier, len(preset.Tiers)),
		PreferredFormats: make([]string, 0, len(preset.PreferredFormats)+len(overrides.PreferAdditional)),
		RejectFormats:    make([]string, 0, len(preset.RejectFormats)+len(overrides.RejectAdditional)),
	}

	// Copy tiers
	copy(result.Tiers, preset.Tiers)

	// Remove excluded formats from preferred
	result.PreferredFormats = removeItems(preset.PreferredFormats, overrides.Exclude)

	// Remove excluded formats from reject (in case user wants to un-reject)
	result.RejectFormats = removeItems(preset.RejectFormats, overrides.Exclude)

	// Add additional preferred formats
	result.PreferredFormats = append(result.PreferredFormats, overrides.PreferAdditional...)

	// Add additional rejected formats
	result.RejectFormats = append(result.RejectFormats, overrides.RejectAdditional...)

	return result
}

// ApplyAudioOverrides applies overrides to an audio preset
func ApplyAudioOverrides(preset AudioQualityPreset, overrides QualityOverrides) AudioQualityPreset {
	result := AudioQualityPreset{
		Name:             preset.Name,
		Description:      preset.Description,
		UpgradeUntil:     preset.UpgradeUntil,
		Tiers:            make([]string, len(preset.Tiers)),
		PreferredFormats: make([]string, 0, len(preset.PreferredFormats)+len(overrides.PreferAdditional)),
		RejectTiers:      make([]string, 0, len(preset.RejectTiers)+len(overrides.RejectAdditional)),
	}

	// Copy tiers
	copy(result.Tiers, preset.Tiers)

	// Remove excluded formats from preferred
	result.PreferredFormats = removeItems(preset.PreferredFormats, overrides.Exclude)

	// Remove excluded tiers from reject (in case user wants to un-reject)
	result.RejectTiers = removeItems(preset.RejectTiers, overrides.Exclude)

	// Add additional preferred formats
	result.PreferredFormats = append(result.PreferredFormats, overrides.PreferAdditional...)

	// Add additional rejected tiers
	result.RejectTiers = append(result.RejectTiers, overrides.RejectAdditional...)

	return result
}

// removeItems removes items from a slice
func removeItems(slice []string, toRemove []string) []string {
	if len(toRemove) == 0 {
		// Return a copy to avoid modifying the original
		result := make([]string, len(slice))
		copy(result, slice)
		return result
	}

	removeSet := make(map[string]bool)
	for _, item := range toRemove {
		removeSet[item] = true
	}

	result := make([]string, 0, len(slice))
	for _, item := range slice {
		if !removeSet[item] {
			result = append(result, item)
		}
	}
	return result
}

// containsString checks if a slice contains a string
func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}
