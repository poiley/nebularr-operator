package lidarr

import (
	"context"

	"github.com/poiley/nebularr-operator/internal/adapters/httpclient"
	"github.com/poiley/nebularr-operator/internal/adapters/shared"
)

// getOwnershipTagID retrieves the ID of the Nebularr ownership tag
func (a *Adapter) getOwnershipTagID(ctx context.Context, c *httpclient.Client) (int, error) {
	return shared.GetOwnershipTagID(ctx, c, "v1")
}

// ensureOwnershipTag creates the ownership tag if it doesn't exist
func (a *Adapter) ensureOwnershipTag(ctx context.Context, c *httpclient.Client) (int, error) {
	return shared.EnsureOwnershipTag(ctx, c, "v1")
}

// hasTag checks if an array of tag IDs contains the specified tag
func hasTag(tags []int, tagID int) bool {
	return shared.HasTag(tags, tagID)
}
