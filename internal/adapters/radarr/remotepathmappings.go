package radarr

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/poiley/nebularr-operator/internal/adapters"
	"github.com/poiley/nebularr-operator/internal/adapters/radarr/client"
	irv1 "github.com/poiley/nebularr-operator/internal/ir/v1"
)

// getRemotePathMappings retrieves all remote path mappings from Radarr
func (a *Adapter) getRemotePathMappings(ctx context.Context, c *client.Client) ([]irv1.RemotePathMappingIR, error) {
	resp, err := c.GetApiV3Remotepathmapping(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get remote path mappings: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var mappings []client.RemotePathMappingResource
	if err := json.NewDecoder(resp.Body).Decode(&mappings); err != nil {
		return nil, fmt.Errorf("failed to decode remote path mappings: %w", err)
	}

	result := make([]irv1.RemotePathMappingIR, 0, len(mappings))
	for _, m := range mappings {
		result = append(result, a.remotePathMappingToIR(&m))
	}

	return result, nil
}

// remotePathMappingToIR converts a client RemotePathMappingResource to IR
func (a *Adapter) remotePathMappingToIR(m *client.RemotePathMappingResource) irv1.RemotePathMappingIR {
	ir := irv1.RemotePathMappingIR{}

	if m.Id != nil {
		ir.ID = int(*m.Id)
	}
	if m.Host != nil {
		ir.Host = *m.Host
	}
	if m.RemotePath != nil {
		ir.RemotePath = *m.RemotePath
	}
	if m.LocalPath != nil {
		ir.LocalPath = *m.LocalPath
	}

	return ir
}

// irToRemotePathMapping converts IR to a client RemotePathMappingResource
func (a *Adapter) irToRemotePathMapping(ir *irv1.RemotePathMappingIR) client.RemotePathMappingResource {
	m := client.RemotePathMappingResource{
		Host:       &ir.Host,
		RemotePath: &ir.RemotePath,
		LocalPath:  &ir.LocalPath,
	}

	if ir.ID > 0 {
		id := int32(ir.ID)
		m.Id = &id
	}

	return m
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
				id := c.ID
				changes.Updates = append(changes.Updates, adapters.Change{
					ResourceType: adapters.ResourceRemotePathMapping,
					Name:         fmt.Sprintf("%s -> %s", d.RemotePath, d.LocalPath),
					ID:           &id,
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
			id := c.ID
			changes.Deletes = append(changes.Deletes, adapters.Change{
				ResourceType: adapters.ResourceRemotePathMapping,
				Name:         fmt.Sprintf("%s -> %s", c.RemotePath, c.LocalPath),
				ID:           &id,
				Payload:      &c,
			})
		}
	}

	return nil
}

// createRemotePathMapping creates a remote path mapping in Radarr
func (a *Adapter) createRemotePathMapping(ctx context.Context, c *client.Client, ir *irv1.RemotePathMappingIR) error {
	mapping := a.irToRemotePathMapping(ir)

	resp, err := c.PostApiV3Remotepathmapping(ctx, mapping)
	if err != nil {
		return fmt.Errorf("failed to create remote path mapping: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

// updateRemotePathMapping updates a remote path mapping in Radarr
func (a *Adapter) updateRemotePathMapping(ctx context.Context, c *client.Client, ir *irv1.RemotePathMappingIR) error {
	mapping := a.irToRemotePathMapping(ir)
	id := fmt.Sprintf("%d", ir.ID)

	resp, err := c.PutApiV3RemotepathmappingId(ctx, id, mapping)
	if err != nil {
		return fmt.Errorf("failed to update remote path mapping: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

// deleteRemotePathMapping deletes a remote path mapping from Radarr
func (a *Adapter) deleteRemotePathMapping(ctx context.Context, c *client.Client, id int) error {
	resp, err := c.DeleteApiV3RemotepathmappingId(ctx, int32(id))
	if err != nil {
		return fmt.Errorf("failed to delete remote path mapping: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}
