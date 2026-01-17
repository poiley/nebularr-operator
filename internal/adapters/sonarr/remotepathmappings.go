package sonarr

import (
	"context"
	"fmt"

	"github.com/poiley/nebularr-operator/internal/adapters"
	irv1 "github.com/poiley/nebularr-operator/internal/ir/v1"
)

// getRemotePathMappings retrieves all remote path mappings from Sonarr
func (a *Adapter) getRemotePathMappings(ctx context.Context, c *httpClient) ([]irv1.RemotePathMappingIR, error) {
	var mappings []RemotePathMappingResource
	if err := c.get(ctx, "/api/v3/remotepathmapping", &mappings); err != nil {
		return nil, fmt.Errorf("failed to get remote path mappings: %w", err)
	}

	result := make([]irv1.RemotePathMappingIR, 0, len(mappings))
	for _, m := range mappings {
		result = append(result, irv1.RemotePathMappingIR{
			ID:         m.ID,
			Host:       m.Host,
			RemotePath: m.RemotePath,
			LocalPath:  m.LocalPath,
		})
	}

	return result, nil
}

// diffRemotePathMappings computes changes for remote path mappings
func (a *Adapter) diffRemotePathMappings(current, desired *irv1.IR, changes *adapters.ChangeSet) error {
	currentMappings := current.RemotePathMappings
	desiredMappings := desired.RemotePathMappings

	// Build lookup maps by host+remotePath (unique key)
	currentByKey := make(map[string]irv1.RemotePathMappingIR)
	for _, m := range currentMappings {
		key := m.Host + "|" + m.RemotePath
		currentByKey[key] = m
	}

	desiredByKey := make(map[string]irv1.RemotePathMappingIR)
	for _, m := range desiredMappings {
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
					ID:           intPtr(c.ID),
					Payload:      &updated,
				})
			}
		} else {
			// Create
			changes.Creates = append(changes.Creates, adapters.Change{
				ResourceType: adapters.ResourceRemotePathMapping,
				Name:         fmt.Sprintf("%s -> %s", d.RemotePath, d.LocalPath),
				Payload:      &d,
			})
		}
	}

	// Find deletes
	for key, c := range currentByKey {
		if _, exists := desiredByKey[key]; !exists {
			changes.Deletes = append(changes.Deletes, adapters.Change{
				ResourceType: adapters.ResourceRemotePathMapping,
				Name:         fmt.Sprintf("%s -> %s", c.RemotePath, c.LocalPath),
				ID:           intPtr(c.ID),
				Payload:      &c,
			})
		}
	}

	return nil
}

// createRemotePathMapping creates a remote path mapping in Sonarr
func (a *Adapter) createRemotePathMapping(ctx context.Context, c *httpClient, ir *irv1.RemotePathMappingIR) error {
	mapping := RemotePathMappingResource{
		Host:       ir.Host,
		RemotePath: ir.RemotePath,
		LocalPath:  ir.LocalPath,
	}

	var created RemotePathMappingResource
	if err := c.post(ctx, "/api/v3/remotepathmapping", mapping, &created); err != nil {
		return fmt.Errorf("failed to create remote path mapping: %w", err)
	}

	return nil
}

// updateRemotePathMapping updates a remote path mapping in Sonarr
func (a *Adapter) updateRemotePathMapping(ctx context.Context, c *httpClient, ir *irv1.RemotePathMappingIR) error {
	mapping := RemotePathMappingResource{
		ID:         ir.ID,
		Host:       ir.Host,
		RemotePath: ir.RemotePath,
		LocalPath:  ir.LocalPath,
	}

	endpoint := fmt.Sprintf("/api/v3/remotepathmapping/%d", ir.ID)
	var updated RemotePathMappingResource
	if err := c.put(ctx, endpoint, mapping, &updated); err != nil {
		return fmt.Errorf("failed to update remote path mapping: %w", err)
	}

	return nil
}

// deleteRemotePathMapping deletes a remote path mapping from Sonarr
func (a *Adapter) deleteRemotePathMapping(ctx context.Context, c *httpClient, id int) error {
	endpoint := fmt.Sprintf("/api/v3/remotepathmapping/%d", id)
	if err := c.delete(ctx, endpoint); err != nil {
		return fmt.Errorf("failed to delete remote path mapping: %w", err)
	}

	return nil
}

// intPtr returns a pointer to an int
func intPtr(i int) *int {
	return &i
}
