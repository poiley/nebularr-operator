package sonarr

import (
	"context"

	"github.com/poiley/nebularr-operator/internal/adapters"
	"github.com/poiley/nebularr-operator/internal/adapters/httpclient"
	irv1 "github.com/poiley/nebularr-operator/internal/ir/v1"
)

// profileID is used to track the managed profile ID
var managedProfileID *int

// getManagedQualityProfiles retrieves quality profiles managed by Nebularr
func (a *Adapter) getManagedQualityProfiles(ctx context.Context, c *httpclient.Client, _ int) ([]*irv1.VideoQualityIR, error) {
	var profiles []QualityProfileResource
	if err := c.Get(ctx, "/api/v3/qualityprofile", &profiles); err != nil {
		return nil, err
	}

	var managed []*irv1.VideoQualityIR
	for _, p := range profiles {
		// Check if profile name starts with "nebularr-" (our naming convention)
		if len(p.Name) > 9 && p.Name[:9] == "nebularr-" {
			ir := a.profileToIR(&p)
			managedProfileID = &p.ID
			managed = append(managed, ir)
		}
	}

	return managed, nil
}

// profileToIR converts a Sonarr quality profile to IR
func (a *Adapter) profileToIR(p *QualityProfileResource) *irv1.VideoQualityIR {
	ir := &irv1.VideoQualityIR{
		ProfileName:    p.Name,
		UpgradeAllowed: p.UpgradeAllowed,
	}

	// Convert tiers
	for _, item := range p.Items {
		if item.Allowed && item.Quality != nil {
			ir.Tiers = append(ir.Tiers, irv1.VideoQualityTierIR{
				Resolution: resolveResolution(item.Quality.Resolution),
				Sources:    []string{item.Quality.Source},
				Allowed:    true,
			})
		}
	}

	return ir
}

// diffQualityProfiles computes changes needed for quality profiles
func (a *Adapter) diffQualityProfiles(current, desired *irv1.IR, changes *adapters.ChangeSet) error {
	var currentProfile *irv1.VideoQualityIR
	var desiredProfile *irv1.VideoQualityIR

	if current.Quality != nil {
		currentProfile = current.Quality.Video
	}
	if desired.Quality != nil {
		desiredProfile = desired.Quality.Video
	}

	// Use shared diff logic
	adapters.DiffQualityProfiles(currentProfile, desiredProfile, managedProfileID, changes)
	return nil
}

// resolveResolution converts resolution int to string
func resolveResolution(res int) string {
	switch res {
	case 2160:
		return "2160p"
	case 1080:
		return "1080p"
	case 720:
		return "720p"
	case 480:
		return "480p"
	default:
		return "unknown"
	}
}
