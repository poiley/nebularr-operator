package readarr

import (
	"context"
	"fmt"

	"github.com/poiley/nebularr-operator/internal/adapters"
	"github.com/poiley/nebularr-operator/internal/adapters/httpclient"
	irv1 "github.com/poiley/nebularr-operator/internal/ir/v1"
)

// getManagedQualityProfiles retrieves quality profiles that are managed by Nebularr.
// Quality profiles in Readarr don't have tags, so we identify managed profiles by name prefix.
func (a *Adapter) getManagedQualityProfiles(ctx context.Context, c *httpclient.Client) ([]*irv1.BookQualityIR, error) {
	var profiles []QualityProfileResource
	if err := c.Get(ctx, "/api/v1/qualityprofile", &profiles); err != nil {
		return nil, fmt.Errorf("failed to get quality profiles: %w", err)
	}

	result := make([]*irv1.BookQualityIR, 0, len(profiles))
	for i := range profiles {
		profile := profiles[i]
		// Check if this profile is managed by Nebularr (has ownership prefix)
		if len(profile.Name) < 9 || profile.Name[:9] != "nebularr-" {
			continue
		}

		ir := a.qualityProfileToIR(&profile)
		result = append(result, ir)
	}

	return result, nil
}

// qualityProfileToIR converts a Readarr quality profile to IR
func (a *Adapter) qualityProfileToIR(profile *QualityProfileResource) *irv1.BookQualityIR {
	ir := &irv1.BookQualityIR{
		ProfileName:    profile.Name,
		UpgradeAllowed: profile.UpgradeAllowed,
	}

	// Extract tiers from items
	for _, item := range profile.Items {
		tier := a.qualityItemToTier(&item)
		if tier != nil {
			ir.Tiers = append(ir.Tiers, *tier)
		}
	}

	// Extract cutoff
	if profile.Cutoff > 0 {
		cutoffTier := a.findCutoffTier(profile.Items, profile.Cutoff)
		if cutoffTier != nil {
			ir.Cutoff = *cutoffTier
		}
	}

	return ir
}

// findCutoffTier searches through quality items to find the tier matching the cutoff ID
func (a *Adapter) findCutoffTier(items []QualityProfileItemResource, cutoffID int) *irv1.BookQualityTierIR {
	for _, item := range items {
		// Check if this item's ID matches the cutoff (for groups)
		if item.ID == cutoffID {
			return a.qualityItemToTier(&item)
		}

		// Check nested items (for groups, check individual qualities)
		for _, subItem := range item.Items {
			if subItem.Quality != nil && subItem.Quality.ID == cutoffID {
				return a.qualityItemToTier(&subItem)
			}
		}

		// Check individual quality ID (for non-group items)
		if item.Quality != nil && item.Quality.ID == cutoffID {
			return a.qualityItemToTier(&item)
		}
	}
	return nil
}

// qualityItemToTier converts a quality profile item to a tier
func (a *Adapter) qualityItemToTier(item *QualityProfileItemResource) *irv1.BookQualityTierIR {
	if item == nil {
		return nil
	}

	tier := &irv1.BookQualityTierIR{
		Allowed: item.Allowed,
	}

	// If it's a group, extract the items
	if len(item.Items) > 0 {
		tier.Name = item.Name
		for _, subItem := range item.Items {
			if subItem.Quality != nil {
				format := a.qualityToFormat(subItem.Quality)
				if format != "" {
					tier.Formats = append(tier.Formats, format)
				}
			}
		}
	} else if item.Quality != nil {
		// Single quality
		tier.Name = item.Quality.Name
		format := a.qualityToFormat(item.Quality)
		if format != "" {
			tier.Formats = []string{format}
		}
	}

	return tier
}

// qualityToFormat maps a Readarr quality to a format string
func (a *Adapter) qualityToFormat(q *QualityResource) string {
	if q == nil {
		return ""
	}
	return q.Name
}

// diffQualityProfiles computes changes needed for quality profiles
func (a *Adapter) diffQualityProfiles(current, desired *irv1.IR, changes *adapters.ChangeSet) error {
	var currentProfile *irv1.BookQualityIR
	var desiredProfile *irv1.BookQualityIR

	if current.Quality != nil && current.Quality.Book != nil {
		currentProfile = current.Quality.Book
	}
	if desired.Quality != nil && desired.Quality.Book != nil {
		desiredProfile = desired.Quality.Book
	}

	// Use shared diff logic
	adapters.DiffBookQualityProfiles(currentProfile, desiredProfile, changes)
	return nil
}

// createQualityProfile creates a new quality profile in Readarr
func (a *Adapter) createQualityProfile(ctx context.Context, c *httpclient.Client, ir *irv1.BookQualityIR, _ int) error {
	// First, get the quality definitions to build proper items
	var qualities []QualityDefinitionResource
	if err := c.Get(ctx, "/api/v1/qualitydefinition", &qualities); err != nil {
		return fmt.Errorf("failed to get quality definitions: %w", err)
	}

	// Build the profile resource
	profile := QualityProfileResource{
		Name:           ir.ProfileName,
		UpgradeAllowed: ir.UpgradeAllowed,
	}

	// Build items from tiers
	for _, tier := range ir.Tiers {
		item := a.tierToQualityItem(&tier, qualities)
		if item != nil {
			profile.Items = append(profile.Items, *item)
		}
	}

	// Set cutoff based on the cutoff tier
	if ir.Cutoff.Name != "" {
		cutoffID := a.findQualityIDByName(qualities, ir.Cutoff.Name)
		if cutoffID > 0 {
			profile.Cutoff = cutoffID
		}
	}

	var result QualityProfileResource
	return c.Post(ctx, "/api/v1/qualityprofile", profile, &result)
}

// updateQualityProfile updates an existing quality profile
func (a *Adapter) updateQualityProfile(ctx context.Context, c *httpclient.Client, ir *irv1.BookQualityIR, profileID int) error {
	// Get quality definitions
	var qualities []QualityDefinitionResource
	if err := c.Get(ctx, "/api/v1/qualitydefinition", &qualities); err != nil {
		return fmt.Errorf("failed to get quality definitions: %w", err)
	}

	// Build the updated profile
	profile := QualityProfileResource{
		ID:             profileID,
		Name:           ir.ProfileName,
		UpgradeAllowed: ir.UpgradeAllowed,
	}

	// Build items from tiers
	for _, tier := range ir.Tiers {
		item := a.tierToQualityItem(&tier, qualities)
		if item != nil {
			profile.Items = append(profile.Items, *item)
		}
	}

	// Set cutoff
	if ir.Cutoff.Name != "" {
		cutoffID := a.findQualityIDByName(qualities, ir.Cutoff.Name)
		if cutoffID > 0 {
			profile.Cutoff = cutoffID
		}
	}

	var result QualityProfileResource
	return c.Put(ctx, fmt.Sprintf("/api/v1/qualityprofile/%d", profileID), profile, &result)
}

// tierToQualityItem converts an IR tier to a quality profile item
func (a *Adapter) tierToQualityItem(tier *irv1.BookQualityTierIR, qualities []QualityDefinitionResource) *QualityProfileItemResource {
	if tier == nil {
		return nil
	}

	item := &QualityProfileItemResource{
		Name:    tier.Name,
		Allowed: tier.Allowed,
	}

	if len(tier.Formats) > 0 {
		// This is a group with multiple formats
		for _, format := range tier.Formats {
			qualityDef := a.findQualityByName(qualities, format)
			if qualityDef != nil {
				subItem := QualityProfileItemResource{
					Allowed: tier.Allowed,
					Quality: &QualityResource{
						ID:   qualityDef.Quality.ID,
						Name: qualityDef.Quality.Name,
					},
				}
				item.Items = append(item.Items, subItem)
			}
		}
	} else if tier.Name != "" {
		// Single quality
		qualityDef := a.findQualityByName(qualities, tier.Name)
		if qualityDef != nil {
			item.Quality = &QualityResource{
				ID:   qualityDef.Quality.ID,
				Name: qualityDef.Quality.Name,
			}
		}
	}

	return item
}

// findQualityByName finds a quality definition by name
func (a *Adapter) findQualityByName(qualities []QualityDefinitionResource, name string) *QualityDefinitionResource {
	for i := range qualities {
		if qualities[i].Quality.Name == name {
			return &qualities[i]
		}
	}
	return nil
}

// findQualityIDByName finds a quality ID by name
func (a *Adapter) findQualityIDByName(qualities []QualityDefinitionResource, name string) int {
	def := a.findQualityByName(qualities, name)
	if def != nil {
		return def.Quality.ID
	}
	return 0
}

// deleteQualityProfileByName deletes a quality profile by name
func (a *Adapter) deleteQualityProfileByName(ctx context.Context, c *httpclient.Client, name string) error {
	var profiles []QualityProfileResource
	if err := c.Get(ctx, "/api/v1/qualityprofile", &profiles); err != nil {
		return err
	}

	for _, profile := range profiles {
		if profile.Name == name {
			return c.Delete(ctx, fmt.Sprintf("/api/v1/qualityprofile/%d", profile.ID))
		}
	}

	return nil // Not found is not an error
}
