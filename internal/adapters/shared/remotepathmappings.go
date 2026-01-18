// Package shared provides common functionality used across multiple *arr adapters.
package shared

import (
	"fmt"

	"github.com/poiley/nebularr-operator/internal/adapters"
	irv1 "github.com/poiley/nebularr-operator/internal/ir/v1"
)

// DiffRemotePathMappings computes changes for remote path mappings.
// Remote path mappings are keyed by host+remotePath combination.
func DiffRemotePathMappings(
	current []irv1.RemotePathMappingIR,
	desired []irv1.RemotePathMappingIR,
	changes *adapters.ChangeSet,
) {
	// Build lookup maps by host+remotePath (unique key)
	currentByKey := make(map[string]irv1.RemotePathMappingIR)
	for _, m := range current {
		key := m.Host + "|" + m.RemotePath
		currentByKey[key] = m
	}

	desiredByKey := make(map[string]irv1.RemotePathMappingIR)
	for _, m := range desired {
		key := m.Host + "|" + m.RemotePath
		desiredByKey[key] = m
	}

	// Find creates and updates
	for key, d := range desiredByKey {
		if c, exists := currentByKey[key]; exists {
			// Check if update needed (local path changed)
			if c.LocalPath != d.LocalPath {
				updated := d
				updated.ID = c.ID // Preserve the ID for update
				changes.Updates = append(changes.Updates, adapters.Change{
					ResourceType: adapters.ResourceRemotePathMapping,
					Name:         fmt.Sprintf("%s -> %s", d.RemotePath, d.LocalPath),
					ID:           IntPtr(c.ID),
					Payload:      &updated,
				})
			}
		} else {
			// Create
			mapping := d // Copy to avoid pointer issues
			changes.Creates = append(changes.Creates, adapters.Change{
				ResourceType: adapters.ResourceRemotePathMapping,
				Name:         fmt.Sprintf("%s -> %s", d.RemotePath, d.LocalPath),
				Payload:      &mapping,
			})
		}
	}

	// Find deletes
	for key, c := range currentByKey {
		if _, exists := desiredByKey[key]; !exists {
			mapping := c // Copy to avoid pointer issues
			changes.Deletes = append(changes.Deletes, adapters.Change{
				ResourceType: adapters.ResourceRemotePathMapping,
				Name:         fmt.Sprintf("%s -> %s", c.RemotePath, c.LocalPath),
				ID:           IntPtr(c.ID),
				Payload:      &mapping,
			})
		}
	}
}
