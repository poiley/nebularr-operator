package prowlarr

import (
	"context"
	"fmt"
)

// Package-level cache for tag IDs
var (
	ownershipTagIDCache = make(map[string]int) // baseURL -> tagID
)

// getOwnershipTagID retrieves the ownership tag ID, returning error if not found
func (a *Adapter) getOwnershipTagID(ctx context.Context, c *httpClient) (int, error) {
	// Check cache first
	if id, ok := ownershipTagIDCache[c.baseURL]; ok {
		return id, nil
	}

	var tags []TagResource
	if err := c.get(ctx, "/api/v1/tag", &tags); err != nil {
		return 0, fmt.Errorf("failed to get tags: %w", err)
	}

	for _, tag := range tags {
		if tag.Label == OwnershipTagName {
			ownershipTagIDCache[c.baseURL] = tag.ID
			return tag.ID, nil
		}
	}

	return 0, fmt.Errorf("ownership tag %q not found", OwnershipTagName)
}

// ensureOwnershipTag creates the ownership tag if it doesn't exist and returns its ID
func (a *Adapter) ensureOwnershipTag(ctx context.Context, c *httpClient) (int, error) {
	// Try to get existing tag
	id, err := a.getOwnershipTagID(ctx, c)
	if err == nil {
		return id, nil
	}

	// Create the tag
	tag := TagResource{
		Label: OwnershipTagName,
	}

	var created TagResource
	if err := c.post(ctx, "/api/v1/tag", tag, &created); err != nil {
		return 0, fmt.Errorf("failed to create ownership tag: %w", err)
	}

	// Cache and return
	ownershipTagIDCache[c.baseURL] = created.ID
	return created.ID, nil
}

// hasTag checks if a resource has the specified tag
func hasTag(tags []int, tagID int) bool {
	for _, t := range tags {
		if t == tagID {
			return true
		}
	}
	return false
}

// ensureTag ensures a tag is in the list
func ensureTag(tags []int, tagID int) []int {
	if hasTag(tags, tagID) {
		return tags
	}
	return append(tags, tagID)
}
