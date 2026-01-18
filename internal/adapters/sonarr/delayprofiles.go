package sonarr

import (
	"context"
	"fmt"

	"github.com/poiley/nebularr-operator/internal/adapters"
	"github.com/poiley/nebularr-operator/internal/adapters/httpclient"
	"github.com/poiley/nebularr-operator/internal/adapters/shared"
	irv1 "github.com/poiley/nebularr-operator/internal/ir/v1"
)

// getManagedDelayProfiles retrieves delay profiles
// Unlike other resources, we manage ALL delay profiles (not just tagged ones)
// because Sonarr creates a default delay profile that we may need to modify
func (a *Adapter) getManagedDelayProfiles(ctx context.Context, c *httpclient.Client) ([]irv1.DelayProfileIR, error) {
	var profiles []DelayProfileResource
	if err := c.Get(ctx, "/api/v3/delayprofile", &profiles); err != nil {
		return nil, fmt.Errorf("failed to get delay profiles: %w", err)
	}

	result := make([]irv1.DelayProfileIR, 0, len(profiles))
	for _, p := range profiles {
		ir := a.delayProfileToIR(&p)
		result = append(result, ir)
	}

	return result, nil
}

// delayProfileToIR converts a Sonarr delay profile to IR
func (a *Adapter) delayProfileToIR(p *DelayProfileResource) irv1.DelayProfileIR {
	ir := irv1.DelayProfileIR{
		ID:    p.ID,
		Order: p.Order,
	}

	switch p.PreferredProtocol {
	case "usenet":
		ir.PreferredProtocol = irv1.ProtocolUsenet
	case "torrent":
		ir.PreferredProtocol = irv1.ProtocolTorrent
	}

	ir.UsenetDelay = p.UsenetDelay
	ir.TorrentDelay = p.TorrentDelay
	ir.EnableUsenet = p.EnableUsenet
	ir.EnableTorrent = p.EnableTorrent
	ir.BypassIfHighestQuality = p.BypassIfHighestQuality
	ir.BypassIfAboveCustomFormatScore = p.BypassIfAboveCustomFormatScore
	ir.MinimumCustomFormatScore = p.MinimumCustomFormatScore

	if len(p.Tags) > 0 {
		ir.Tags = make([]int, len(p.Tags))
		copy(ir.Tags, p.Tags)
	}

	return ir
}

// irToDelayProfile converts IR to Sonarr delay profile resource
func (a *Adapter) irToDelayProfile(ir *irv1.DelayProfileIR, tagIDs []int) DelayProfileResource {
	protocol := "usenet"
	switch ir.PreferredProtocol {
	case irv1.ProtocolUsenet:
		protocol = "usenet"
	case irv1.ProtocolTorrent:
		protocol = "torrent"
	}

	p := DelayProfileResource{
		Order:                          ir.Order,
		PreferredProtocol:              protocol,
		UsenetDelay:                    ir.UsenetDelay,
		TorrentDelay:                   ir.TorrentDelay,
		EnableUsenet:                   ir.EnableUsenet,
		EnableTorrent:                  ir.EnableTorrent,
		BypassIfHighestQuality:         ir.BypassIfHighestQuality,
		BypassIfAboveCustomFormatScore: ir.BypassIfAboveCustomFormatScore,
		MinimumCustomFormatScore:       ir.MinimumCustomFormatScore,
	}

	if len(tagIDs) > 0 {
		p.Tags = tagIDs
	} else {
		// Empty tags means applies to all
		p.Tags = []int{}
	}

	return p
}

// diffDelayProfiles computes changes needed for delay profiles using shared logic
func (a *Adapter) diffDelayProfiles(current, desired *irv1.IR, changes *adapters.ChangeSet) error {
	shared.DiffDelayProfiles(current.DelayProfiles, desired.DelayProfiles, changes)
	return nil
}

// createDelayProfile creates a new delay profile
func (a *Adapter) createDelayProfile(ctx context.Context, c *httpclient.Client, ir *irv1.DelayProfileIR, tagID int) error {
	// Convert tag names to IDs if needed
	var tagIDs []int
	if tagID > 0 && len(ir.TagNames) == 0 {
		// Use the ownership tag
		tagIDs = []int{tagID}
	}
	// If ir.Tags is populated (from resolved tag names), use those
	tagIDs = append(tagIDs, ir.Tags...)

	profile := a.irToDelayProfile(ir, tagIDs)

	return c.Post(ctx, "/api/v3/delayprofile", profile, nil)
}

// updateDelayProfile updates an existing delay profile
func (a *Adapter) updateDelayProfile(ctx context.Context, c *httpclient.Client, ir *irv1.DelayProfileIR, tagID int) error {
	// Convert tag names to IDs if needed
	var tagIDs []int
	if tagID > 0 && len(ir.TagNames) == 0 {
		tagIDs = []int{tagID}
	}
	tagIDs = append(tagIDs, ir.Tags...)

	profile := a.irToDelayProfile(ir, tagIDs)
	profile.ID = ir.ID

	return c.Put(ctx, fmt.Sprintf("/api/v3/delayprofile/%d", ir.ID), profile, nil)
}

// deleteDelayProfile deletes a delay profile
func (a *Adapter) deleteDelayProfile(ctx context.Context, c *httpclient.Client, id int) error {
	return c.Delete(ctx, fmt.Sprintf("/api/v3/delayprofile/%d", id))
}
