package readarr

import (
	"context"
	"fmt"

	"github.com/poiley/nebularr-operator/internal/adapters"
	"github.com/poiley/nebularr-operator/internal/adapters/httpclient"
	irv1 "github.com/poiley/nebularr-operator/internal/ir/v1"
)

// getManagedMetadataProfiles retrieves metadata profiles that are managed by Nebularr.
// Metadata profiles in Readarr don't have tags, so we identify managed profiles by name prefix.
func (a *Adapter) getManagedMetadataProfiles(ctx context.Context, c *httpclient.Client) ([]*irv1.MetadataProfileIR, error) {
	var profiles []MetadataProfileResource
	if err := c.Get(ctx, "/api/v1/metadataprofile", &profiles); err != nil {
		return nil, fmt.Errorf("failed to get metadata profiles: %w", err)
	}

	result := make([]*irv1.MetadataProfileIR, 0, len(profiles))
	for i := range profiles {
		profile := profiles[i]
		// Check if this profile is managed by Nebularr (has ownership prefix)
		if len(profile.Name) < 9 || profile.Name[:9] != "nebularr-" {
			continue
		}

		ir := a.metadataProfileToIR(&profile)
		result = append(result, ir)
	}

	return result, nil
}

// metadataProfileToIR converts a Readarr metadata profile to IR
func (a *Adapter) metadataProfileToIR(profile *MetadataProfileResource) *irv1.MetadataProfileIR {
	id := profile.ID
	return &irv1.MetadataProfileIR{
		ID:                  &id,
		Name:                profile.Name,
		MinPopularity:       profile.MinPopularity,
		SkipMissingDate:     profile.SkipMissingDate,
		SkipMissingIsbn:     profile.SkipMissingIsbn,
		SkipPartsAndSets:    profile.SkipPartsAndSets,
		SkipSeriesSecondary: profile.SkipSeriesSecondary,
		AllowedLanguages:    profile.AllowedLanguages,
	}
}

// diffMetadataProfiles computes changes needed for metadata profiles
func (a *Adapter) diffMetadataProfiles(current, desired []*irv1.MetadataProfileIR, changes *adapters.ChangeSet) error {
	currentMap := make(map[string]*irv1.MetadataProfileIR)
	for _, profile := range current {
		currentMap[profile.Name] = profile
	}

	desiredMap := make(map[string]*irv1.MetadataProfileIR)
	for _, profile := range desired {
		desiredMap[profile.Name] = profile
	}

	// Find creates and updates
	for name, desiredProfile := range desiredMap {
		currentProfile, exists := currentMap[name]
		if !exists {
			changes.Creates = append(changes.Creates, adapters.Change{
				ResourceType: adapters.ResourceMetadataProfile,
				Name:         name,
				Payload:      desiredProfile,
			})
		} else if !metadataProfilesEqual(currentProfile, desiredProfile) {
			changes.Updates = append(changes.Updates, adapters.Change{
				ResourceType: adapters.ResourceMetadataProfile,
				Name:         name,
				ID:           currentProfile.ID,
				Payload:      desiredProfile,
			})
		}
	}

	// Find deletes
	for name, currentProfile := range currentMap {
		if _, exists := desiredMap[name]; !exists {
			changes.Deletes = append(changes.Deletes, adapters.Change{
				ResourceType: adapters.ResourceMetadataProfile,
				Name:         name,
				ID:           currentProfile.ID,
			})
		}
	}

	return nil
}

// metadataProfilesEqual compares two metadata profiles
func metadataProfilesEqual(current, desired *irv1.MetadataProfileIR) bool {
	if current == nil || desired == nil {
		return current == desired
	}

	return current.Name == desired.Name &&
		current.MinPopularity == desired.MinPopularity &&
		current.SkipMissingDate == desired.SkipMissingDate &&
		current.SkipMissingIsbn == desired.SkipMissingIsbn &&
		current.SkipPartsAndSets == desired.SkipPartsAndSets &&
		current.SkipSeriesSecondary == desired.SkipSeriesSecondary &&
		current.AllowedLanguages == desired.AllowedLanguages
}

// createMetadataProfile creates a new metadata profile in Readarr
func (a *Adapter) createMetadataProfile(ctx context.Context, c *httpclient.Client, ir *irv1.MetadataProfileIR) error {
	profile := MetadataProfileResource{
		Name:                ir.Name,
		MinPopularity:       ir.MinPopularity,
		SkipMissingDate:     ir.SkipMissingDate,
		SkipMissingIsbn:     ir.SkipMissingIsbn,
		SkipPartsAndSets:    ir.SkipPartsAndSets,
		SkipSeriesSecondary: ir.SkipSeriesSecondary,
		AllowedLanguages:    ir.AllowedLanguages,
	}

	var result MetadataProfileResource
	return c.Post(ctx, "/api/v1/metadataprofile", profile, &result)
}

// updateMetadataProfile updates an existing metadata profile
func (a *Adapter) updateMetadataProfile(ctx context.Context, c *httpclient.Client, ir *irv1.MetadataProfileIR, profileID int) error {
	profile := MetadataProfileResource{
		ID:                  profileID,
		Name:                ir.Name,
		MinPopularity:       ir.MinPopularity,
		SkipMissingDate:     ir.SkipMissingDate,
		SkipMissingIsbn:     ir.SkipMissingIsbn,
		SkipPartsAndSets:    ir.SkipPartsAndSets,
		SkipSeriesSecondary: ir.SkipSeriesSecondary,
		AllowedLanguages:    ir.AllowedLanguages,
	}

	var result MetadataProfileResource
	return c.Put(ctx, fmt.Sprintf("/api/v1/metadataprofile/%d", profileID), profile, &result)
}

// deleteMetadataProfileByName deletes a metadata profile by name
func (a *Adapter) deleteMetadataProfileByName(ctx context.Context, c *httpclient.Client, name string) error {
	var profiles []MetadataProfileResource
	if err := c.Get(ctx, "/api/v1/metadataprofile", &profiles); err != nil {
		return err
	}

	for _, profile := range profiles {
		if profile.Name == name {
			return c.Delete(ctx, fmt.Sprintf("/api/v1/metadataprofile/%d", profile.ID))
		}
	}

	return nil // Not found is not an error
}
