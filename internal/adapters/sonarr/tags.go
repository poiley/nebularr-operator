package sonarr

import (
	"context"
	"fmt"
)

// getOwnershipTagID retrieves the ID of the Nebularr ownership tag
func (a *Adapter) getOwnershipTagID(ctx context.Context, c *httpClient) (int, error) {
	var tags []TagResource
	if err := c.get(ctx, "/api/v3/tag", &tags); err != nil {
		return 0, fmt.Errorf("failed to get tags: %w", err)
	}

	for _, tag := range tags {
		if tag.Label == OwnershipTagName {
			return tag.ID, nil
		}
	}

	return 0, fmt.Errorf("ownership tag %q not found", OwnershipTagName)
}

// ensureOwnershipTag creates the ownership tag if it doesn't exist
func (a *Adapter) ensureOwnershipTag(ctx context.Context, c *httpClient) (int, error) {
	// Check if tag exists
	tagID, err := a.getOwnershipTagID(ctx, c)
	if err == nil {
		return tagID, nil
	}

	// Create the tag
	tag := TagResource{
		Label: OwnershipTagName,
	}

	var created TagResource
	if err := c.post(ctx, "/api/v3/tag", tag, &created); err != nil {
		return 0, fmt.Errorf("failed to create ownership tag: %w", err)
	}

	return created.ID, nil
}

// hasTag checks if an array of tag IDs contains the specified tag
func hasTag(tags []int, tagID int) bool {
	for _, t := range tags {
		if t == tagID {
			return true
		}
	}
	return false
}
