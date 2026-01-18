// Package shared provides common functionality used across multiple *arr adapters.
package shared

import (
	"github.com/poiley/nebularr-operator/internal/adapters"
	irv1 "github.com/poiley/nebularr-operator/internal/ir/v1"
)

// DiffDelayProfiles computes changes needed for delay profiles.
// Delay profiles are keyed by Order (priority), not name.
// Order 1 is the default profile that always exists and should not be deleted.
func DiffDelayProfiles(
	current []irv1.DelayProfileIR,
	desired []irv1.DelayProfileIR,
	changes *adapters.ChangeSet,
) {
	// Build maps for comparison
	currentByOrder := make(map[int]irv1.DelayProfileIR)
	for _, p := range current {
		currentByOrder[p.Order] = p
	}

	desiredByOrder := make(map[int]irv1.DelayProfileIR)
	for _, p := range desired {
		desiredByOrder[p.Order] = p
	}

	// Find creates and updates
	for _, dp := range desired {
		currentDP, exists := currentByOrder[dp.Order]
		if !exists {
			// Create new delay profile
			payload := dp // Copy to avoid pointer issues
			changes.Creates = append(changes.Creates, adapters.Change{
				ResourceType: adapters.ResourceDelayProfile,
				Name:         dp.Name,
				Payload:      &payload,
			})
		} else if !DelayProfilesEqual(currentDP, dp) {
			// Update existing delay profile
			updated := dp
			updated.ID = currentDP.ID
			changes.Updates = append(changes.Updates, adapters.Change{
				ResourceType: adapters.ResourceDelayProfile,
				Name:         dp.Name,
				ID:           IntPtr(currentDP.ID),
				Payload:      &updated,
			})
		}
	}

	// Find deletes - only delete profiles with order > 1
	// (Order 1 is the default profile that always exists)
	for _, cp := range current {
		if cp.Order == 1 {
			// Don't delete the default profile
			continue
		}
		if _, exists := desiredByOrder[cp.Order]; !exists {
			changes.Deletes = append(changes.Deletes, adapters.Change{
				ResourceType: adapters.ResourceDelayProfile,
				Name:         cp.Name,
				ID:           IntPtr(cp.ID),
			})
		}
	}
}

// DelayProfilesEqual compares two delay profiles for equality.
func DelayProfilesEqual(a, b irv1.DelayProfileIR) bool {
	if a.Order != b.Order {
		return false
	}
	if a.PreferredProtocol != b.PreferredProtocol {
		return false
	}
	if a.UsenetDelay != b.UsenetDelay {
		return false
	}
	if a.TorrentDelay != b.TorrentDelay {
		return false
	}
	if a.EnableUsenet != b.EnableUsenet {
		return false
	}
	if a.EnableTorrent != b.EnableTorrent {
		return false
	}
	if a.BypassIfHighestQuality != b.BypassIfHighestQuality {
		return false
	}
	if a.BypassIfAboveCustomFormatScore != b.BypassIfAboveCustomFormatScore {
		return false
	}
	if a.MinimumCustomFormatScore != b.MinimumCustomFormatScore {
		return false
	}
	// Compare tags
	if len(a.Tags) != len(b.Tags) {
		return false
	}
	tagSet := make(map[int]bool)
	for _, t := range a.Tags {
		tagSet[t] = true
	}
	for _, t := range b.Tags {
		if !tagSet[t] {
			return false
		}
	}
	return true
}

// IntPtr returns a pointer to the given int value.
// This is a helper function used across adapters.
func IntPtr(i int) *int {
	return &i
}
