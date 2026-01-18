package lidarr

import (
	"context"

	"github.com/poiley/nebularr-operator/internal/adapters"
	"github.com/poiley/nebularr-operator/internal/adapters/httpclient"
	irv1 "github.com/poiley/nebularr-operator/internal/ir/v1"
)

// managedProfileID tracks the managed quality profile ID
var managedProfileID *int

// getManagedQualityProfiles retrieves quality profiles managed by Nebularr
func (a *Adapter) getManagedQualityProfiles(ctx context.Context, c *httpclient.Client, _ int) ([]*irv1.AudioQualityIR, error) {
	var profiles []QualityProfileResource
	if err := c.Get(ctx, "/api/v1/qualityprofile", &profiles); err != nil {
		return nil, err
	}

	var managed []*irv1.AudioQualityIR
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

// profileToIR converts a Lidarr quality profile to IR
func (a *Adapter) profileToIR(p *QualityProfileResource) *irv1.AudioQualityIR {
	ir := &irv1.AudioQualityIR{
		ProfileName:    p.Name,
		UpgradeAllowed: p.UpgradeAllowed,
	}

	// Convert items to tiers
	for _, item := range p.Items {
		if item.Allowed && item.Quality != nil {
			tier := qualityToTier(item.Quality.Name)
			if tier != "" {
				ir.Tiers = append(ir.Tiers, irv1.AudioQualityTierIR{
					Tier:    tier,
					Allowed: true,
				})
			}
		}
	}

	return ir
}

// qualityToTier maps Lidarr quality names to IR tier names
func qualityToTier(qualityName string) string {
	// Map common Lidarr quality names to tiers
	switch qualityName {
	case "FLAC 24bit", "FLAC 24bit Lossless":
		return "lossless-hires"
	case "FLAC", "FLAC Lossless", "ALAC", "APE", "WAV":
		return "lossless"
	case "MP3-320", "AAC-320", "OGG Vorbis Q10":
		return "lossy-high"
	case "MP3-256", "AAC-256", "OGG Vorbis Q8":
		return "lossy-mid"
	case "MP3-128", "AAC-128", "MP3-VBR", "AAC-VBR":
		return "lossy-low"
	default:
		return ""
	}
}

// diffQualityProfiles computes changes needed for quality profiles using shared logic
func (a *Adapter) diffQualityProfiles(current, desired *irv1.IR, changes *adapters.ChangeSet) error {
	var currentProfile *irv1.AudioQualityIR
	var desiredProfile *irv1.AudioQualityIR

	if current.Quality != nil {
		currentProfile = current.Quality.Audio
	}
	if desired.Quality != nil {
		desiredProfile = desired.Quality.Audio
	}

	// Use shared diff logic for audio quality profiles
	adapters.DiffAudioQualityProfiles(currentProfile, desiredProfile, managedProfileID, changes)
	return nil
}
