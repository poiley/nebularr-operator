package lidarr

import (
	"context"
	"fmt"

	"github.com/poiley/nebularr-operator/internal/adapters"
	irv1 "github.com/poiley/nebularr-operator/internal/ir/v1"
)

// applyCreate creates a new resource
func (a *Adapter) applyCreate(ctx context.Context, c *httpClient, change adapters.Change, tagID int) error {
	switch change.ResourceType {
	case adapters.ResourceQualityProfile:
		return a.createQualityProfile(ctx, c, change.Payload.(*irv1.AudioQualityIR))
	case adapters.ResourceDownloadClient:
		return a.createDownloadClient(ctx, c, change.Payload.(irv1.DownloadClientIR), tagID)
	case adapters.ResourceIndexer:
		return a.createIndexer(ctx, c, change.Payload.(irv1.IndexerIR), tagID)
	case adapters.ResourceRootFolder:
		return a.createRootFolder(ctx, c, change.Payload.(irv1.RootFolderIR))
	default:
		return fmt.Errorf("unsupported resource type for create: %s", change.ResourceType)
	}
}

// applyUpdate updates an existing resource
func (a *Adapter) applyUpdate(ctx context.Context, c *httpClient, change adapters.Change, tagID int) error {
	switch change.ResourceType {
	case adapters.ResourceQualityProfile:
		return a.updateQualityProfile(ctx, c, *change.ID, change.Payload.(*irv1.AudioQualityIR))
	case adapters.ResourceDownloadClient:
		return a.updateDownloadClient(ctx, c, *change.ID, change.Payload.(irv1.DownloadClientIR), tagID)
	case adapters.ResourceIndexer:
		return a.updateIndexer(ctx, c, *change.ID, change.Payload.(irv1.IndexerIR), tagID)
	case adapters.ResourceNamingConfig:
		return a.updateNaming(ctx, c, change.Payload.(*irv1.LidarrNamingIR))
	default:
		return fmt.Errorf("unsupported resource type for update: %s", change.ResourceType)
	}
}

// applyDelete deletes a resource
func (a *Adapter) applyDelete(ctx context.Context, c *httpClient, change adapters.Change) error {
	if change.ID == nil {
		return fmt.Errorf("cannot delete resource without ID")
	}

	switch change.ResourceType {
	case adapters.ResourceQualityProfile:
		return c.delete(ctx, fmt.Sprintf("/api/v1/qualityprofile/%d", *change.ID))
	case adapters.ResourceDownloadClient:
		return c.delete(ctx, fmt.Sprintf("/api/v1/downloadclient/%d", *change.ID))
	case adapters.ResourceIndexer:
		return c.delete(ctx, fmt.Sprintf("/api/v1/indexer/%d", *change.ID))
	case adapters.ResourceRootFolder:
		return c.delete(ctx, fmt.Sprintf("/api/v1/rootfolder/%d", *change.ID))
	default:
		return fmt.Errorf("unsupported resource type for delete: %s", change.ResourceType)
	}
}

// createQualityProfile creates a quality profile using schema for complete structure
func (a *Adapter) createQualityProfile(ctx context.Context, c *httpClient, profile *irv1.AudioQualityIR) error {
	// Get schema to get full items structure
	var schema QualityProfileResource
	if err := c.get(ctx, "/api/v1/qualityprofile/schema", &schema); err != nil {
		return fmt.Errorf("failed to get quality profile schema: %w", err)
	}

	// Build profile from schema with our tier preferences
	resource := a.profileFromSchema(&schema, profile)
	return c.post(ctx, "/api/v1/qualityprofile", resource, nil)
}

// updateQualityProfile updates a quality profile using schema for complete structure
func (a *Adapter) updateQualityProfile(ctx context.Context, c *httpClient, id int, profile *irv1.AudioQualityIR) error {
	// Get schema to get full items structure
	var schema QualityProfileResource
	if err := c.get(ctx, "/api/v1/qualityprofile/schema", &schema); err != nil {
		return fmt.Errorf("failed to get quality profile schema: %w", err)
	}

	// Build profile from schema with our tier preferences
	resource := a.profileFromSchema(&schema, profile)
	resource.ID = id
	return c.put(ctx, fmt.Sprintf("/api/v1/qualityprofile/%d", id), resource, nil)
}

// profileFromSchema builds a quality profile from schema with tier preferences applied
func (a *Adapter) profileFromSchema(schema *QualityProfileResource, profile *irv1.AudioQualityIR) QualityProfileResource {
	resource := QualityProfileResource{
		Name:           profile.ProfileName,
		UpgradeAllowed: profile.UpgradeAllowed,
		Items:          make([]QualityProfileItem, len(schema.Items)),
	}

	// Build tier allowed map
	tierAllowed := make(map[string]bool)
	for _, tier := range profile.Tiers {
		tierAllowed[tier.Tier] = tier.Allowed
	}

	// Copy schema items and apply our tier preferences
	var cutoffID int
	for i, item := range schema.Items {
		resource.Items[i] = item

		// Map Lidarr group names to our tier names
		tierName := groupNameToTier(item.Name)
		if tierName == "" && item.Quality != nil {
			tierName = qualityNameToTier(item.Quality.Name)
		}

		// Apply allowed status from our tiers
		if allowed, ok := tierAllowed[tierName]; ok {
			resource.Items[i].Allowed = allowed

			// Track cutoff - use the highest allowed tier
			if allowed && profile.Cutoff == tierName {
				if item.ID != 0 {
					cutoffID = item.ID
				} else if item.Quality != nil {
					cutoffID = item.Quality.ID
				}
			}
		}
	}

	// Set cutoff
	if cutoffID == 0 {
		// Default to Lossless group if no cutoff specified
		cutoffID = 1005
	}
	resource.Cutoff = cutoffID

	return resource
}

// groupNameToTier maps Lidarr quality group names to IR tier names
func groupNameToTier(groupName string) string {
	switch groupName {
	case "Lossless":
		return "lossless"
	case "High Quality Lossy":
		return "lossy-high"
	case "Mid Quality Lossy":
		return "lossy-mid"
	case "Low Quality Lossy":
		return "lossy-low"
	case "Poor Quality Lossy", "Trash Quality Lossy":
		return "" // Not used in our tiers
	default:
		return ""
	}
}

// qualityNameToTier maps individual quality names to tier names
func qualityNameToTier(qualityName string) string {
	switch qualityName {
	case "FLAC 24bit", "ALAC 24bit":
		return "lossless-hires"
	case "FLAC", "ALAC", "APE", "WavPack", "WAV":
		return "lossless"
	case "MP3-320", "AAC-320", "OGG Vorbis Q10", "OGG Vorbis Q9", "MP3-VBR-V0", "AAC-VBR":
		return "lossy-high"
	case "MP3-256", "AAC-256", "OGG Vorbis Q8", "OGG Vorbis Q7", "MP3-VBR-V2":
		return "lossy-mid"
	case "MP3-192", "AAC-192", "OGG Vorbis Q6", "WMA", "MP3-160", "MP3-128":
		return "lossy-low"
	default:
		return ""
	}
}

// createDownloadClient creates a download client
func (a *Adapter) createDownloadClient(ctx context.Context, c *httpClient, dc irv1.DownloadClientIR, tagID int) error {
	resource := a.downloadClientFromIR(dc, tagID)
	return c.post(ctx, "/api/v1/downloadclient", resource, nil)
}

// updateDownloadClient updates a download client
func (a *Adapter) updateDownloadClient(ctx context.Context, c *httpClient, id int, dc irv1.DownloadClientIR, tagID int) error {
	resource := a.downloadClientFromIR(dc, tagID)
	resource.ID = id
	return c.put(ctx, fmt.Sprintf("/api/v1/downloadclient/%d", id), resource, nil)
}

// downloadClientFromIR converts IR to Lidarr download client
func (a *Adapter) downloadClientFromIR(dc irv1.DownloadClientIR, tagID int) DownloadClientResource {
	resource := DownloadClientResource{
		Name:           dc.Name,
		Implementation: dc.Implementation,
		ConfigContract: dc.Implementation + "Settings",
		Protocol:       dc.Protocol,
		Enable:         dc.Enable,
		Priority:       dc.Priority,
		Tags:           []int{tagID},
		Fields: []Field{
			{Name: "host", Value: dc.Host},
			{Name: "port", Value: dc.Port},
			{Name: "useSsl", Value: dc.UseTLS},
			{Name: "username", Value: dc.Username},
			{Name: "password", Value: dc.Password},
			{Name: "musicCategory", Value: dc.Category},
		},
		RemoveCompletedDownloads: dc.RemoveCompletedDownloads,
		RemoveFailedDownloads:    dc.RemoveFailedDownloads,
	}
	return resource
}

// createIndexer creates an indexer
func (a *Adapter) createIndexer(ctx context.Context, c *httpClient, idx irv1.IndexerIR, tagID int) error {
	resource := a.indexerFromIR(idx, tagID)
	return c.post(ctx, "/api/v1/indexer", resource, nil)
}

// updateIndexer updates an indexer
func (a *Adapter) updateIndexer(ctx context.Context, c *httpClient, id int, idx irv1.IndexerIR, tagID int) error {
	resource := a.indexerFromIR(idx, tagID)
	resource.ID = id
	return c.put(ctx, fmt.Sprintf("/api/v1/indexer/%d", id), resource, nil)
}

// indexerFromIR converts IR to Lidarr indexer
func (a *Adapter) indexerFromIR(idx irv1.IndexerIR, tagID int) IndexerResource {
	resource := IndexerResource{
		Name:                    idx.Name,
		Implementation:          idx.Implementation,
		ConfigContract:          idx.Implementation + "Settings",
		Protocol:                idx.Protocol,
		Enable:                  idx.Enable,
		Priority:                idx.Priority,
		Tags:                    []int{tagID},
		EnableRss:               idx.EnableRss,
		EnableAutomaticSearch:   idx.EnableAutomaticSearch,
		EnableInteractiveSearch: idx.EnableInteractiveSearch,
		Fields: []Field{
			{Name: "baseUrl", Value: idx.URL},
			{Name: "apiKey", Value: idx.APIKey},
			{Name: "categories", Value: idx.Categories},
		},
	}
	return resource
}

// createRootFolder creates a root folder
func (a *Adapter) createRootFolder(ctx context.Context, c *httpClient, rf irv1.RootFolderIR) error {
	resource := RootFolderResource{
		Path:                 rf.Path,
		Name:                 rf.Name,
		DefaultMonitorOption: rf.DefaultMonitor,
	}
	if resource.DefaultMonitorOption == "" {
		resource.DefaultMonitorOption = "all"
	}
	return c.post(ctx, "/api/v1/rootfolder", resource, nil)
}

// updateNaming updates the naming configuration
func (a *Adapter) updateNaming(ctx context.Context, c *httpClient, naming *irv1.LidarrNamingIR) error {
	resource := NamingConfigResource{
		ID:                       namingConfigID,
		RenameTracks:             naming.RenameTracks,
		ReplaceIllegalCharacters: naming.ReplaceIllegalCharacters,
		StandardTrackFormat:      naming.StandardTrackFormat,
		MultiDiscTrackFormat:     naming.MultiDiscTrackFormat,
		ArtistFolderFormat:       naming.ArtistFolderFormat,
		AlbumFolderFormat:        naming.AlbumFolderFormat,
	}
	return c.put(ctx, "/api/v1/config/naming", resource, nil)
}
