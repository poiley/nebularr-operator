package lidarr

import (
	"context"
	"fmt"

	"github.com/poiley/nebularr-operator/internal/adapters"
	"github.com/poiley/nebularr-operator/internal/adapters/httpclient"
	"github.com/poiley/nebularr-operator/internal/adapters/shared"
	irv1 "github.com/poiley/nebularr-operator/internal/ir/v1"
)

// getRemotePathMappings retrieves all remote path mappings from Lidarr
func (a *Adapter) getRemotePathMappings(ctx context.Context, c *httpclient.Client) ([]irv1.RemotePathMappingIR, error) {
	var mappings []RemotePathMappingResource
	if err := c.Get(ctx, "/api/v1/remotepathmapping", &mappings); err != nil {
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

// diffRemotePathMappings computes changes for remote path mappings using shared logic
func (a *Adapter) diffRemotePathMappings(current, desired *irv1.IR, changes *adapters.ChangeSet) error {
	shared.DiffRemotePathMappings(current.RemotePathMappings, desired.RemotePathMappings, changes)
	return nil
}

// createRemotePathMapping creates a remote path mapping in Lidarr
func (a *Adapter) createRemotePathMapping(ctx context.Context, c *httpclient.Client, ir *irv1.RemotePathMappingIR) error {
	mapping := RemotePathMappingResource{
		Host:       ir.Host,
		RemotePath: ir.RemotePath,
		LocalPath:  ir.LocalPath,
	}

	var created RemotePathMappingResource
	if err := c.Post(ctx, "/api/v1/remotepathmapping", mapping, &created); err != nil {
		return fmt.Errorf("failed to create remote path mapping: %w", err)
	}

	return nil
}

// updateRemotePathMapping updates a remote path mapping in Lidarr
func (a *Adapter) updateRemotePathMapping(ctx context.Context, c *httpclient.Client, ir *irv1.RemotePathMappingIR) error {
	mapping := RemotePathMappingResource{
		ID:         ir.ID,
		Host:       ir.Host,
		RemotePath: ir.RemotePath,
		LocalPath:  ir.LocalPath,
	}

	endpoint := fmt.Sprintf("/api/v1/remotepathmapping/%d", ir.ID)
	var updated RemotePathMappingResource
	if err := c.Put(ctx, endpoint, mapping, &updated); err != nil {
		return fmt.Errorf("failed to update remote path mapping: %w", err)
	}

	return nil
}

// deleteRemotePathMapping deletes a remote path mapping from Lidarr
func (a *Adapter) deleteRemotePathMapping(ctx context.Context, c *httpclient.Client, id int) error {
	endpoint := fmt.Sprintf("/api/v1/remotepathmapping/%d", id)
	if err := c.Delete(ctx, endpoint); err != nil {
		return fmt.Errorf("failed to delete remote path mapping: %w", err)
	}

	return nil
}
