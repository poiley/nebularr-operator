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

// getManagedDelayProfiles retrieves delay profiles
// Unlike other resources, we manage ALL delay profiles (not just tagged ones)
// because Radarr creates a default delay profile that we may need to modify
func (a *Adapter) getManagedDelayProfiles(ctx context.Context, c *client.Client) ([]irv1.DelayProfileIR, error) {
	resp, err := c.GetApiV3Delayprofile(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get delay profiles: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var profiles []client.DelayProfileResource
	if err := json.NewDecoder(resp.Body).Decode(&profiles); err != nil {
		return nil, fmt.Errorf("failed to decode delay profiles: %w", err)
	}

	result := make([]irv1.DelayProfileIR, 0, len(profiles))
	for _, p := range profiles {
		ir := a.delayProfileToIR(&p)
		result = append(result, ir)
	}

	return result, nil
}

// delayProfileToIR converts a Radarr delay profile to IR
func (a *Adapter) delayProfileToIR(p *client.DelayProfileResource) irv1.DelayProfileIR {
	ir := irv1.DelayProfileIR{
		ID:    ptrToInt(p.Id),
		Order: ptrToInt(p.Order),
	}

	if p.PreferredProtocol != nil {
		switch *p.PreferredProtocol {
		case client.DownloadProtocolUsenet:
			ir.PreferredProtocol = irv1.ProtocolUsenet
		case client.DownloadProtocolTorrent:
			ir.PreferredProtocol = irv1.ProtocolTorrent
		}
	}

	ir.UsenetDelay = ptrToInt(p.UsenetDelay)
	ir.TorrentDelay = ptrToInt(p.TorrentDelay)
	ir.EnableUsenet = ptrToBool(p.EnableUsenet)
	ir.EnableTorrent = ptrToBool(p.EnableTorrent)
	ir.BypassIfHighestQuality = ptrToBool(p.BypassIfHighestQuality)
	ir.BypassIfAboveCustomFormatScore = ptrToBool(p.BypassIfAboveCustomFormatScore)
	ir.MinimumCustomFormatScore = ptrToInt(p.MinimumCustomFormatScore)

	if p.Tags != nil {
		ir.Tags = make([]int, len(*p.Tags))
		for i, t := range *p.Tags {
			ir.Tags[i] = int(t)
		}
	}

	return ir
}

// irToDelayProfile converts IR to Radarr delay profile resource
func (a *Adapter) irToDelayProfile(ir *irv1.DelayProfileIR, tagIDs []int32) client.DelayProfileResource {
	var protocol client.DownloadProtocol
	switch ir.PreferredProtocol {
	case irv1.ProtocolUsenet:
		protocol = client.DownloadProtocolUsenet
	case irv1.ProtocolTorrent:
		protocol = client.DownloadProtocolTorrent
	default:
		protocol = client.DownloadProtocolUsenet
	}

	p := client.DelayProfileResource{
		Order:                          intPtr(ir.Order),
		PreferredProtocol:              &protocol,
		UsenetDelay:                    intPtr(ir.UsenetDelay),
		TorrentDelay:                   intPtr(ir.TorrentDelay),
		EnableUsenet:                   boolPtr(ir.EnableUsenet),
		EnableTorrent:                  boolPtr(ir.EnableTorrent),
		BypassIfHighestQuality:         boolPtr(ir.BypassIfHighestQuality),
		BypassIfAboveCustomFormatScore: boolPtr(ir.BypassIfAboveCustomFormatScore),
		MinimumCustomFormatScore:       intPtr(ir.MinimumCustomFormatScore),
	}

	if len(tagIDs) > 0 {
		p.Tags = &tagIDs
	} else {
		// Empty tags means applies to all
		empty := []int32{}
		p.Tags = &empty
	}

	return p
}

// diffDelayProfiles computes changes needed for delay profiles
func (a *Adapter) diffDelayProfiles(current, desired *irv1.IR, changes *adapters.ChangeSet) error {
	currentProfiles := current.DelayProfiles
	desiredProfiles := desired.DelayProfiles

	// Build maps for comparison
	currentByOrder := make(map[int]irv1.DelayProfileIR)
	for _, p := range currentProfiles {
		currentByOrder[p.Order] = p
	}

	desiredByOrder := make(map[int]irv1.DelayProfileIR)
	for _, p := range desiredProfiles {
		desiredByOrder[p.Order] = p
	}

	// Find creates and updates
	for _, dp := range desiredProfiles {
		current, exists := currentByOrder[dp.Order]
		if !exists {
			// Create new delay profile
			changes.Creates = append(changes.Creates, adapters.Change{
				ResourceType: adapters.ResourceDelayProfile,
				Name:         dp.Name,
				Payload:      &dp,
			})
		} else if !delayProfilesEqual(current, dp) {
			// Update existing delay profile
			updated := dp
			updated.ID = current.ID
			changes.Updates = append(changes.Updates, adapters.Change{
				ResourceType: adapters.ResourceDelayProfile,
				Name:         dp.Name,
				ID:           intPtrFromInt(current.ID),
				Payload:      &updated,
			})
		}
	}

	// Find deletes - only delete profiles with order > 1
	// (Order 1 is the default profile that always exists)
	for _, cp := range currentProfiles {
		if cp.Order == 1 {
			// Don't delete the default profile
			continue
		}
		if _, exists := desiredByOrder[cp.Order]; !exists {
			changes.Deletes = append(changes.Deletes, adapters.Change{
				ResourceType: adapters.ResourceDelayProfile,
				Name:         cp.Name,
				ID:           intPtrFromInt(cp.ID),
			})
		}
	}

	return nil
}

// delayProfilesEqual compares two delay profiles for equality
func delayProfilesEqual(a, b irv1.DelayProfileIR) bool {
	if a.Order != b.Order {
		return false
	}
	if a.PreferredProtocol != b.PreferredProtocol {
		return false
	}
	if a.UsenetDelay != b.UsenetDelay {
		return false
	}
	if a.TorrentDelay != b.TorrentDelay {
		return false
	}
	if a.EnableUsenet != b.EnableUsenet {
		return false
	}
	if a.EnableTorrent != b.EnableTorrent {
		return false
	}
	if a.BypassIfHighestQuality != b.BypassIfHighestQuality {
		return false
	}
	if a.BypassIfAboveCustomFormatScore != b.BypassIfAboveCustomFormatScore {
		return false
	}
	if a.MinimumCustomFormatScore != b.MinimumCustomFormatScore {
		return false
	}
	// Compare tags
	if len(a.Tags) != len(b.Tags) {
		return false
	}
	tagSet := make(map[int]bool)
	for _, t := range a.Tags {
		tagSet[t] = true
	}
	for _, t := range b.Tags {
		if !tagSet[t] {
			return false
		}
	}
	return true
}

// createDelayProfile creates a new delay profile
func (a *Adapter) createDelayProfile(ctx context.Context, c *client.Client, ir *irv1.DelayProfileIR, tagID int) error {
	// Convert tag names to IDs if needed
	// Pre-allocate with capacity for ownership tag + ir.Tags
	capacity := len(ir.Tags)
	if tagID > 0 && len(ir.TagNames) == 0 {
		capacity++
	}
	tagIDs := make([]int32, 0, capacity)
	if tagID > 0 && len(ir.TagNames) == 0 {
		// Use the ownership tag
		tagIDs = append(tagIDs, int32(tagID))
	}
	// If ir.Tags is populated (from resolved tag names), use those
	for _, t := range ir.Tags {
		tagIDs = append(tagIDs, int32(t))
	}

	profile := a.irToDelayProfile(ir, tagIDs)

	resp, err := c.PostApiV3Delayprofile(ctx, profile)
	if err != nil {
		return fmt.Errorf("failed to create delay profile: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

// updateDelayProfile updates an existing delay profile
func (a *Adapter) updateDelayProfile(ctx context.Context, c *client.Client, ir *irv1.DelayProfileIR, tagID int) error {
	// Convert tag names to IDs if needed
	// Pre-allocate with capacity for ownership tag + ir.Tags
	capacity := len(ir.Tags)
	if tagID > 0 && len(ir.TagNames) == 0 {
		capacity++
	}
	tagIDs := make([]int32, 0, capacity)
	if tagID > 0 && len(ir.TagNames) == 0 {
		tagIDs = append(tagIDs, int32(tagID))
	}
	for _, t := range ir.Tags {
		tagIDs = append(tagIDs, int32(t))
	}

	profile := a.irToDelayProfile(ir, tagIDs)
	profile.Id = intPtr(ir.ID)

	resp, err := c.PutApiV3DelayprofileId(ctx, fmt.Sprintf("%d", ir.ID), profile)
	if err != nil {
		return fmt.Errorf("failed to update delay profile: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

// deleteDelayProfile deletes a delay profile
func (a *Adapter) deleteDelayProfile(ctx context.Context, c *client.Client, id int) error {
	resp, err := c.DeleteApiV3DelayprofileId(ctx, int32(id))
	if err != nil {
		return fmt.Errorf("failed to delete delay profile: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

// intPtrFromInt returns a pointer to an int
func intPtrFromInt(i int) *int {
	return &i
}
