package prowlarr

import (
	"context"

	"github.com/poiley/nebularr-operator/internal/adapters/httpclient"
	"github.com/poiley/nebularr-operator/internal/adapters/shared"
)

// getOwnershipTagID retrieves the ownership tag ID, returning error if not found
func (a *Adapter) getOwnershipTagID(ctx context.Context, c *httpclient.Client) (int, error) {
	return shared.GetOwnershipTagID(ctx, c, "v1")
}

// ensureOwnershipTag creates the ownership tag if it doesn't exist and returns its ID
func (a *Adapter) ensureOwnershipTag(ctx context.Context, c *httpclient.Client) (int, error) {
	return shared.EnsureOwnershipTag(ctx, c, "v1")
}

// hasTag checks if a resource has the specified tag
func hasTag(tags []int, tagID int) bool {
	return shared.HasTag(tags, tagID)
}
