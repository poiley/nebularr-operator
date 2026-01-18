// Package shared provides common functionality used across multiple *arr adapters.
package shared

import (
	"context"
	"fmt"

	"github.com/poiley/nebularr-operator/internal/adapters/httpclient"
)

// OwnershipTagName is the standard tag name used by Nebularr to track managed resources.
const OwnershipTagName = "nebularr-managed"

// GetOwnershipTagID retrieves the ID of the Nebularr ownership tag.
// apiVersion should be "v1" or "v3" depending on the service.
func GetOwnershipTagID(ctx context.Context, c *httpclient.Client, apiVersion string) (int, error) {
	var tags []TagResource
	endpoint := fmt.Sprintf("/api/%s/tag", apiVersion)
	if err := c.Get(ctx, endpoint, &tags); err != nil {
		return 0, fmt.Errorf("failed to get tags: %w", err)
	}

	for _, tag := range tags {
		if tag.Label == OwnershipTagName {
			return tag.ID, nil
		}
	}

	return 0, fmt.Errorf("ownership tag %q not found", OwnershipTagName)
}

// EnsureOwnershipTag creates the ownership tag if it doesn't exist and returns its ID.
// apiVersion should be "v1" or "v3" depending on the service.
func EnsureOwnershipTag(ctx context.Context, c *httpclient.Client, apiVersion string) (int, error) {
	// Check if tag exists
	tagID, err := GetOwnershipTagID(ctx, c, apiVersion)
	if err == nil {
		return tagID, nil
	}

	// Create the tag
	tag := TagResource{
		Label: OwnershipTagName,
	}

	endpoint := fmt.Sprintf("/api/%s/tag", apiVersion)
	var created TagResource
	if err := c.Post(ctx, endpoint, tag, &created); err != nil {
		return 0, fmt.Errorf("failed to create ownership tag: %w", err)
	}

	return created.ID, nil
}

// HasTag checks if an array of tag IDs contains the specified tag ID.
func HasTag(tags []int, tagID int) bool {
	for _, t := range tags {
		if t == tagID {
			return true
		}
	}
	return false
}
