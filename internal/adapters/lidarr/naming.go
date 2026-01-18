package lidarr

import (
	"context"

	"github.com/poiley/nebularr-operator/internal/adapters"
	"github.com/poiley/nebularr-operator/internal/adapters/httpclient"
	irv1 "github.com/poiley/nebularr-operator/internal/ir/v1"
)

// namingConfigID stores the ID for the naming config (always 1 in Lidarr)
var namingConfigID = 1

// getNamingConfig retrieves the naming configuration
func (a *Adapter) getNamingConfig(ctx context.Context, c *httpclient.Client) (*irv1.LidarrNamingIR, error) {
	var naming NamingConfigResource
	if err := c.Get(ctx, "/api/v1/config/naming", &naming); err != nil {
		return nil, err
	}

	namingConfigID = naming.ID

	return &irv1.LidarrNamingIR{
		RenameTracks:             naming.RenameTracks,
		ReplaceIllegalCharacters: naming.ReplaceIllegalCharacters,
		StandardTrackFormat:      naming.StandardTrackFormat,
		MultiDiscTrackFormat:     naming.MultiDiscTrackFormat,
		ArtistFolderFormat:       naming.ArtistFolderFormat,
		AlbumFolderFormat:        naming.AlbumFolderFormat,
	}, nil
}

// diffNaming computes changes needed for naming config
func (a *Adapter) diffNaming(current, desired *irv1.IR, changes *adapters.ChangeSet) error {
	var currentNaming *irv1.LidarrNamingIR
	var desiredNaming *irv1.LidarrNamingIR

	if current.Naming != nil {
		currentNaming = current.Naming.Lidarr
	}
	if desired.Naming != nil {
		desiredNaming = desired.Naming.Lidarr
	}

	// No desired naming config - nothing to do
	if desiredNaming == nil {
		return nil
	}

	// Naming config always exists in Lidarr (ID=1), so we just update it
	if namingNeedsUpdate(currentNaming, desiredNaming) {
		changes.Updates = append(changes.Updates, adapters.Change{
			ResourceType: adapters.ResourceNamingConfig,
			Name:         "naming",
			ID:           &namingConfigID,
			Payload:      desiredNaming,
		})
	}

	return nil
}

// namingNeedsUpdate checks if naming config needs updating
func namingNeedsUpdate(current, desired *irv1.LidarrNamingIR) bool {
	if current == nil {
		return true
	}
	if current.RenameTracks != desired.RenameTracks {
		return true
	}
	if current.StandardTrackFormat != desired.StandardTrackFormat {
		return true
	}
	if current.MultiDiscTrackFormat != desired.MultiDiscTrackFormat {
		return true
	}
	if current.ArtistFolderFormat != desired.ArtistFolderFormat {
		return true
	}
	if current.AlbumFolderFormat != desired.AlbumFolderFormat {
		return true
	}
	return false
}
