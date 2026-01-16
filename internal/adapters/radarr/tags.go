package radarr

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/poiley/nebularr-operator/internal/adapters/radarr/client"
)

// getOwnershipTagID retrieves the ID of the Nebularr ownership tag
func (a *Adapter) getOwnershipTagID(ctx context.Context, c *client.Client) (int, error) {
	resp, err := c.GetApiV3Tag(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get tags: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var tags []client.TagResource
	if err := json.NewDecoder(resp.Body).Decode(&tags); err != nil {
		return 0, fmt.Errorf("failed to decode tags: %w", err)
	}

	for _, tag := range tags {
		if ptrToString(tag.Label) == OwnershipTagName {
			return ptrToInt(tag.Id), nil
		}
	}

	return 0, fmt.Errorf("ownership tag not found")
}

// ensureOwnershipTag ensures the Nebularr ownership tag exists and returns its ID
func (a *Adapter) ensureOwnershipTag(ctx context.Context, c *client.Client) (int, error) {
	// First try to get existing tag
	tagID, err := a.getOwnershipTagID(ctx, c)
	if err == nil {
		return tagID, nil
	}

	// Create the tag
	resp, err := c.PostApiV3Tag(ctx, client.PostApiV3TagJSONRequestBody{
		Label: stringPtr(OwnershipTagName),
	})
	if err != nil {
		return 0, fmt.Errorf("failed to create ownership tag: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("unexpected status code creating tag: %d", resp.StatusCode)
	}

	var tag client.TagResource
	if err := json.NewDecoder(resp.Body).Decode(&tag); err != nil {
		return 0, fmt.Errorf("failed to decode created tag: %w", err)
	}

	return ptrToInt(tag.Id), nil
}

// hasTag checks if a resource has the given tag ID
func hasTag(tags *[]int32, tagID int) bool {
	if tags == nil {
		return false
	}
	for _, t := range *tags {
		if int(t) == tagID {
			return true
		}
	}
	return false
}

// addTag adds a tag ID to a slice if not already present
func addTag(tags *[]int32, tagID int) *[]int32 {
	if tags == nil {
		return &[]int32{int32(tagID)}
	}
	for _, t := range *tags {
		if int(t) == tagID {
			return tags
		}
	}
	newTags := append(*tags, int32(tagID))
	return &newTags
}
