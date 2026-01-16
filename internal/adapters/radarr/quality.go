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

// getManagedQualityProfiles retrieves quality profiles tagged with the ownership tag
func (a *Adapter) getManagedQualityProfiles(ctx context.Context, c *client.Client, tagID int) ([]*irv1.VideoQualityIR, error) {
	resp, err := c.GetApiV3Qualityprofile(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get quality profiles: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var profiles []client.QualityProfileResource
	if err := json.NewDecoder(resp.Body).Decode(&profiles); err != nil {
		return nil, fmt.Errorf("failed to decode quality profiles: %w", err)
	}

	var result []*irv1.VideoQualityIR
	for _, profile := range profiles {
		// Check if this profile is managed by Nebularr (has ownership tag)
		// Quality profiles don't have tags in Radarr, so we check by name prefix
		name := ptrToString(profile.Name)
		if len(name) < 9 || name[:9] != "nebularr-" {
			continue
		}

		ir := a.qualityProfileToIR(&profile)
		result = append(result, ir)
	}

	return result, nil
}

// qualityProfileToIR converts a Radarr quality profile to IR
func (a *Adapter) qualityProfileToIR(profile *client.QualityProfileResource) *irv1.VideoQualityIR {
	ir := &irv1.VideoQualityIR{
		ProfileName:    ptrToString(profile.Name),
		UpgradeAllowed: ptrToBool(profile.UpgradeAllowed),
		FormatScores:   make(map[string]int),
	}

	// Extract cutoff
	if profile.Cutoff != nil {
		// We'd need to map the cutoff ID back to a tier
		// For now, we'll leave this as a TODO
	}

	// Extract tiers from items
	if profile.Items != nil {
		for _, item := range *profile.Items {
			tier := a.qualityItemToTier(&item)
			if tier != nil {
				ir.Tiers = append(ir.Tiers, *tier)
			}
		}
	}

	// Extract format scores
	if profile.FormatItems != nil {
		for _, fi := range *profile.FormatItems {
			if fi.Name != nil && fi.Score != nil {
				name := ptrToString(fi.Name)
				if name != "" {
					ir.FormatScores[name] = int(*fi.Score)
				}
			}
		}
	}

	if profile.MinFormatScore != nil {
		ir.MinimumCustomFormatScore = int(*profile.MinFormatScore)
	}

	if profile.CutoffFormatScore != nil {
		ir.UpgradeUntilCustomFormatScore = int(*profile.CutoffFormatScore)
	}

	return ir
}

// qualityItemToTier converts a quality profile item to a tier
func (a *Adapter) qualityItemToTier(item *client.QualityProfileQualityItemResource) *irv1.VideoQualityTierIR {
	if item == nil {
		return nil
	}

	tier := &irv1.VideoQualityTierIR{
		Allowed: ptrToBool(item.Allowed),
	}

	// If it's a group, extract the items
	if item.Items != nil && len(*item.Items) > 0 {
		tier.Resolution = ptrToString(item.Name)
		for _, subItem := range *item.Items {
			if subItem.Quality != nil {
				source := a.qualityToSource(subItem.Quality)
				if source != "" {
					tier.Sources = append(tier.Sources, source)
				}
			}
		}
	} else if item.Quality != nil {
		// Single quality
		tier.Resolution = a.qualityToResolution(item.Quality)
		source := a.qualityToSource(item.Quality)
		if source != "" {
			tier.Sources = []string{source}
		}
	}

	return tier
}

// qualityToResolution maps a Radarr quality to a resolution string
func (a *Adapter) qualityToResolution(q *client.Quality) string {
	if q == nil || q.Resolution == nil {
		return ""
	}
	return fmt.Sprintf("%dp", *q.Resolution)
}

// qualityToSource maps a Radarr quality to a source string
func (a *Adapter) qualityToSource(q *client.Quality) string {
	if q == nil || q.Source == nil {
		return ""
	}
	// Map Radarr source enum to our source strings
	switch *q.Source {
	case client.Bluray:
		return "bluray"
	case client.Webdl:
		return "webdl"
	case client.Webrip:
		return "webrip"
	case client.Tv:
		return "hdtv"
	case client.Dvd:
		return "dvd"
	case client.Cam:
		return "cam"
	case client.Telesync:
		return "telesync"
	case client.Telecine:
		return "telecine"
	case client.Workprint:
		return "workprint"
	default:
		return ""
	}
}

// diffQualityProfiles computes changes needed for quality profiles
func (a *Adapter) diffQualityProfiles(current, desired *irv1.IR, changes *adapters.ChangeSet) error {
	// Get current and desired quality profiles
	var currentProfile *irv1.VideoQualityIR
	var desiredProfile *irv1.VideoQualityIR

	if current.Quality != nil {
		currentProfile = current.Quality.Video
	}
	if desired.Quality != nil {
		desiredProfile = desired.Quality.Video
	}

	// No desired profile - delete current if exists
	if desiredProfile == nil {
		if currentProfile != nil {
			changes.Deletes = append(changes.Deletes, adapters.Change{
				ResourceType: adapters.ResourceQualityProfile,
				Name:         currentProfile.ProfileName,
			})
		}
		return nil
	}

	// No current profile - create
	if currentProfile == nil {
		changes.Creates = append(changes.Creates, adapters.Change{
			ResourceType: adapters.ResourceQualityProfile,
			Name:         desiredProfile.ProfileName,
			Payload:      desiredProfile,
		})
		return nil
	}

	// Both exist - check if update needed
	// For now, we always update (proper diff would compare fields)
	changes.Updates = append(changes.Updates, adapters.Change{
		ResourceType: adapters.ResourceQualityProfile,
		Name:         desiredProfile.ProfileName,
		Payload:      desiredProfile,
	})

	return nil
}

// diffCustomFormats computes changes needed for custom formats
func (a *Adapter) diffCustomFormats(current, desired *irv1.IR, changes *adapters.ChangeSet) error {
	// Build maps for comparison
	currentFormats := make(map[string]*irv1.CustomFormatIR)
	desiredFormats := make(map[string]*irv1.CustomFormatIR)

	if current.Quality != nil && current.Quality.Video != nil {
		for i := range current.Quality.Video.CustomFormats {
			cf := &current.Quality.Video.CustomFormats[i]
			currentFormats[cf.Name] = cf
		}
	}

	if desired.Quality != nil && desired.Quality.Video != nil {
		for i := range desired.Quality.Video.CustomFormats {
			cf := &desired.Quality.Video.CustomFormats[i]
			desiredFormats[cf.Name] = cf
		}
	}

	// Find creates and updates
	for name, desiredCF := range desiredFormats {
		if _, exists := currentFormats[name]; !exists {
			changes.Creates = append(changes.Creates, adapters.Change{
				ResourceType: adapters.ResourceCustomFormat,
				Name:         name,
				Payload:      desiredCF,
			})
		} else {
			// Update (proper diff would compare specs)
			changes.Updates = append(changes.Updates, adapters.Change{
				ResourceType: adapters.ResourceCustomFormat,
				Name:         name,
				Payload:      desiredCF,
			})
		}
	}

	// Find deletes
	for name := range currentFormats {
		if _, exists := desiredFormats[name]; !exists {
			changes.Deletes = append(changes.Deletes, adapters.Change{
				ResourceType: adapters.ResourceCustomFormat,
				Name:         name,
			})
		}
	}

	return nil
}

func ptrToBool(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}
