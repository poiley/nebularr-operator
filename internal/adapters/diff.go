// Package adapters provides shared diff logic for *arr service adapters.
package adapters

import (
	irv1 "github.com/poiley/nebularr-operator/internal/ir/v1"
)

// DiffVideoQualityProfiles computes changes needed for video quality profiles.
// This is shared logic used by Radarr and Sonarr adapters.
func DiffVideoQualityProfiles(
	currentProfile *irv1.VideoQualityIR,
	desiredProfile *irv1.VideoQualityIR,
	profileID *int,
	changes *ChangeSet,
) {
	// No desired profile - delete current if exists
	if desiredProfile == nil {
		if currentProfile != nil && profileID != nil {
			changes.Deletes = append(changes.Deletes, Change{
				ResourceType: ResourceQualityProfile,
				Name:         currentProfile.ProfileName,
				ID:           profileID,
			})
		}
		return
	}

	// No current profile - create new
	if currentProfile == nil {
		changes.Creates = append(changes.Creates, Change{
			ResourceType: ResourceQualityProfile,
			Name:         desiredProfile.ProfileName,
			Payload:      desiredProfile,
		})
		return
	}

	// Both exist - skip update since quality profiles are complex to compare
	// and re-applying the same profile is idempotent but noisy.
	// The profile structure (with nested groups) makes comparison difficult,
	// and Sonarr/Radarr will accept the same profile without issues.
}

// DiffQualityProfiles is an alias for DiffVideoQualityProfiles for backward compatibility.
func DiffQualityProfiles(
	currentProfile *irv1.VideoQualityIR,
	desiredProfile *irv1.VideoQualityIR,
	profileID *int,
	changes *ChangeSet,
) {
	DiffVideoQualityProfiles(currentProfile, desiredProfile, profileID, changes)
}

// DiffAudioQualityProfiles computes changes needed for audio quality profiles.
// This is shared logic used by the Lidarr adapter.
func DiffAudioQualityProfiles(
	currentProfile *irv1.AudioQualityIR,
	desiredProfile *irv1.AudioQualityIR,
	profileID *int,
	changes *ChangeSet,
) {
	// No desired profile - delete current if exists
	if desiredProfile == nil {
		if currentProfile != nil && profileID != nil {
			changes.Deletes = append(changes.Deletes, Change{
				ResourceType: ResourceQualityProfile,
				Name:         currentProfile.ProfileName,
				ID:           profileID,
			})
		}
		return
	}

	// No current profile - create new
	if currentProfile == nil {
		changes.Creates = append(changes.Creates, Change{
			ResourceType: ResourceQualityProfile,
			Name:         desiredProfile.ProfileName,
			Payload:      desiredProfile,
		})
		return
	}

	// Both exist - skip update since quality profiles are complex to compare
	// and re-applying the same profile is idempotent but noisy.
	// The profile structure (with nested groups) makes comparison difficult.
}

// DiffDownloadClients computes changes needed for download clients.
// Returns a map of client names to IDs for use in updates/deletes.
func DiffDownloadClients(
	current []irv1.DownloadClientIR,
	desired []irv1.DownloadClientIR,
	clientIDs map[string]int,
	changes *ChangeSet,
) {
	currentMap := make(map[string]irv1.DownloadClientIR)
	for _, dc := range current {
		currentMap[dc.Name] = dc
	}

	desiredMap := make(map[string]irv1.DownloadClientIR)
	for _, dc := range desired {
		desiredMap[dc.Name] = dc
	}

	// Find creates and updates
	for name, desiredDC := range desiredMap {
		currentDC, exists := currentMap[name]
		if !exists {
			changes.Creates = append(changes.Creates, Change{
				ResourceType: ResourceDownloadClient,
				Name:         name,
				Payload:      desiredDC,
			})
		} else if !DownloadClientsEqual(currentDC, desiredDC) {
			id := clientIDs[name]
			changes.Updates = append(changes.Updates, Change{
				ResourceType: ResourceDownloadClient,
				Name:         name,
				ID:           &id,
				Payload:      desiredDC,
			})
		}
	}

	// Find deletes
	for name := range currentMap {
		if _, exists := desiredMap[name]; !exists {
			id := clientIDs[name]
			changes.Deletes = append(changes.Deletes, Change{
				ResourceType: ResourceDownloadClient,
				Name:         name,
				ID:           &id,
			})
		}
	}
}

// DownloadClientsEqual compares two download clients to determine if they're equivalent.
func DownloadClientsEqual(current, desired irv1.DownloadClientIR) bool {
	return current.Name == desired.Name &&
		current.Implementation == desired.Implementation &&
		current.Host == desired.Host &&
		current.Port == desired.Port &&
		current.UseTLS == desired.UseTLS &&
		current.Category == desired.Category &&
		current.Enable == desired.Enable &&
		current.Priority == desired.Priority
}

// DiffCustomFormats computes changes needed for custom formats.
func DiffCustomFormats(
	current []irv1.CustomFormatIR,
	desired []irv1.CustomFormatIR,
	changes *ChangeSet,
) {
	currentMap := make(map[string]*irv1.CustomFormatIR)
	for i := range current {
		currentMap[current[i].Name] = &current[i]
	}

	desiredMap := make(map[string]*irv1.CustomFormatIR)
	for i := range desired {
		desiredMap[desired[i].Name] = &desired[i]
	}

	// Find creates and updates
	for name, desiredCF := range desiredMap {
		currentCF, exists := currentMap[name]
		if !exists {
			changes.Creates = append(changes.Creates, Change{
				ResourceType: ResourceCustomFormat,
				Name:         name,
				Payload:      desiredCF,
			})
		} else if !CustomFormatsEqual(currentCF, desiredCF) {
			changes.Updates = append(changes.Updates, Change{
				ResourceType: ResourceCustomFormat,
				Name:         name,
				Payload:      desiredCF,
			})
		}
	}

	// Find deletes
	for name := range currentMap {
		if _, exists := desiredMap[name]; !exists {
			changes.Deletes = append(changes.Deletes, Change{
				ResourceType: ResourceCustomFormat,
				Name:         name,
			})
		}
	}
}

// CustomFormatsEqual compares two custom formats to determine if they're equivalent.
func CustomFormatsEqual(current, desired *irv1.CustomFormatIR) bool {
	if current == nil || desired == nil {
		return current == desired
	}

	if current.Name != desired.Name {
		return false
	}
	if current.IncludeWhenRenaming != desired.IncludeWhenRenaming {
		return false
	}
	if len(current.Specifications) != len(desired.Specifications) {
		return false
	}

	// Build a map of current specs by name for comparison
	currentSpecs := make(map[string]irv1.FormatSpecIR)
	for _, spec := range current.Specifications {
		currentSpecs[spec.Name] = spec
	}

	// Check if all desired specs exist and match
	for _, desiredSpec := range desired.Specifications {
		currentSpec, exists := currentSpecs[desiredSpec.Name]
		if !exists {
			return false
		}
		if !FormatSpecsEqual(currentSpec, desiredSpec) {
			return false
		}
	}

	return true
}

// FormatSpecsEqual compares two format specifications.
func FormatSpecsEqual(current, desired irv1.FormatSpecIR) bool {
	return current.Type == desired.Type &&
		current.Name == desired.Name &&
		current.Negate == desired.Negate &&
		current.Required == desired.Required &&
		current.Value == desired.Value
}

// DiffIndexers computes changes needed for indexers.
func DiffIndexers(
	current []irv1.IndexerIR,
	desired []irv1.IndexerIR,
	indexerIDs map[string]int,
	changes *ChangeSet,
) {
	currentMap := make(map[string]irv1.IndexerIR)
	for _, idx := range current {
		currentMap[idx.Name] = idx
	}

	desiredMap := make(map[string]irv1.IndexerIR)
	for _, idx := range desired {
		desiredMap[idx.Name] = idx
	}

	// Find creates and updates
	for name, desiredIdx := range desiredMap {
		currentIdx, exists := currentMap[name]
		if !exists {
			changes.Creates = append(changes.Creates, Change{
				ResourceType: ResourceIndexer,
				Name:         name,
				Payload:      desiredIdx,
			})
		} else if !IndexersEqual(currentIdx, desiredIdx) {
			id := indexerIDs[name]
			changes.Updates = append(changes.Updates, Change{
				ResourceType: ResourceIndexer,
				Name:         name,
				ID:           &id,
				Payload:      desiredIdx,
			})
		}
	}

	// Find deletes
	for name := range currentMap {
		if _, exists := desiredMap[name]; !exists {
			id := indexerIDs[name]
			changes.Deletes = append(changes.Deletes, Change{
				ResourceType: ResourceIndexer,
				Name:         name,
				ID:           &id,
			})
		}
	}
}

// IndexersEqual compares two indexers to determine if they're equivalent.
func IndexersEqual(current, desired irv1.IndexerIR) bool {
	return current.Name == desired.Name &&
		current.Implementation == desired.Implementation &&
		current.URL == desired.URL &&
		current.Enable == desired.Enable &&
		current.Priority == desired.Priority &&
		current.EnableRss == desired.EnableRss &&
		current.EnableAutomaticSearch == desired.EnableAutomaticSearch &&
		current.EnableInteractiveSearch == desired.EnableInteractiveSearch
}

// DiffRootFolders computes changes needed for root folders.
// Note: We only create missing folders, never delete (folders may have content).
func DiffRootFolders(
	current []irv1.RootFolderIR,
	desired []irv1.RootFolderIR,
	changes *ChangeSet,
) {
	currentPaths := make(map[string]bool)
	for _, folder := range current {
		currentPaths[folder.Path] = true
	}

	// Find creates only (no updates or deletes for root folders)
	for _, folder := range desired {
		if !currentPaths[folder.Path] {
			changes.Creates = append(changes.Creates, Change{
				ResourceType: ResourceRootFolder,
				Name:         folder.Path,
				Payload:      irv1.RootFolderIR{Path: folder.Path},
			})
		}
	}
}
