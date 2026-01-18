package lidarr

import (
	"context"
	"fmt"

	"github.com/poiley/nebularr-operator/internal/adapters"
	"github.com/poiley/nebularr-operator/internal/adapters/httpclient"
	"github.com/poiley/nebularr-operator/internal/adapters/shared"
	irv1 "github.com/poiley/nebularr-operator/internal/ir/v1"
)

// applyCreate creates a new resource
func (a *Adapter) applyCreate(ctx context.Context, c *httpclient.Client, change adapters.Change, tagID int) error {
	switch change.ResourceType {
	case adapters.ResourceQualityProfile:
		return a.createQualityProfile(ctx, c, change.Payload.(*irv1.AudioQualityIR))
	case adapters.ResourceDownloadClient:
		return a.createDownloadClient(ctx, c, change.Payload.(irv1.DownloadClientIR), tagID)
	case adapters.ResourceIndexer:
		return a.createIndexer(ctx, c, change.Payload.(irv1.IndexerIR), tagID)
	case adapters.ResourceRootFolder:
		return a.createRootFolder(ctx, c, change.Payload.(irv1.RootFolderIR))
	case adapters.ResourceRemotePathMapping:
		return a.createRemotePathMapping(ctx, c, change.Payload.(*irv1.RemotePathMappingIR))
	case adapters.ResourceNotification:
		return a.createNotification(ctx, c, change.Payload.(*irv1.NotificationIR), tagID)
	case adapters.ResourceCustomFormat:
		return a.createCustomFormat(ctx, c, change.Payload.(*irv1.CustomFormatIR))
	case adapters.ResourceDelayProfile:
		return a.createDelayProfile(ctx, c, change.Payload.(*irv1.DelayProfileIR), tagID)
	default:
		return fmt.Errorf("unsupported resource type for create: %s", change.ResourceType)
	}
}

// applyUpdate updates an existing resource
func (a *Adapter) applyUpdate(ctx context.Context, c *httpclient.Client, change adapters.Change, tagID int) error {
	switch change.ResourceType {
	case adapters.ResourceQualityProfile:
		return a.updateQualityProfile(ctx, c, *change.ID, change.Payload.(*irv1.AudioQualityIR))
	case adapters.ResourceDownloadClient:
		return a.updateDownloadClient(ctx, c, *change.ID, change.Payload.(irv1.DownloadClientIR), tagID)
	case adapters.ResourceIndexer:
		return a.updateIndexer(ctx, c, *change.ID, change.Payload.(irv1.IndexerIR), tagID)
	case adapters.ResourceNamingConfig:
		return a.updateNaming(ctx, c, change.Payload.(*irv1.LidarrNamingIR))
	case adapters.ResourceRemotePathMapping:
		return a.updateRemotePathMapping(ctx, c, change.Payload.(*irv1.RemotePathMappingIR))
	case adapters.ResourceNotification:
		return a.updateNotification(ctx, c, change.Payload.(*irv1.NotificationIR), tagID)
	case adapters.ResourceCustomFormat:
		return a.updateCustomFormat(ctx, c, change.Payload.(*irv1.CustomFormatIR))
	case adapters.ResourceDelayProfile:
		return a.updateDelayProfile(ctx, c, change.Payload.(*irv1.DelayProfileIR), tagID)
	default:
		return fmt.Errorf("unsupported resource type for update: %s", change.ResourceType)
	}
}

// applyDelete deletes a resource
func (a *Adapter) applyDelete(ctx context.Context, c *httpclient.Client, change adapters.Change) error {
	if change.ID == nil {
		return fmt.Errorf("cannot delete resource without ID")
	}

	switch change.ResourceType {
	case adapters.ResourceQualityProfile:
		return c.Delete(ctx, fmt.Sprintf("/api/v1/qualityprofile/%d", *change.ID))
	case adapters.ResourceDownloadClient:
		return c.Delete(ctx, fmt.Sprintf("/api/v1/downloadclient/%d", *change.ID))
	case adapters.ResourceIndexer:
		return c.Delete(ctx, fmt.Sprintf("/api/v1/indexer/%d", *change.ID))
	case adapters.ResourceRootFolder:
		return c.Delete(ctx, fmt.Sprintf("/api/v1/rootfolder/%d", *change.ID))
	case adapters.ResourceRemotePathMapping:
		return a.deleteRemotePathMapping(ctx, c, *change.ID)
	case adapters.ResourceNotification:
		return a.deleteNotification(ctx, c, *change.ID)
	case adapters.ResourceCustomFormat:
		return a.deleteCustomFormat(ctx, c, *change.ID)
	case adapters.ResourceDelayProfile:
		return a.deleteDelayProfile(ctx, c, *change.ID)
	default:
		return fmt.Errorf("unsupported resource type for delete: %s", change.ResourceType)
	}
}

// createQualityProfile creates a quality profile using schema for complete structure
func (a *Adapter) createQualityProfile(ctx context.Context, c *httpclient.Client, profile *irv1.AudioQualityIR) error {
	// Get schema to get full items structure
	var schema QualityProfileResource
	if err := c.Get(ctx, "/api/v1/qualityprofile/schema", &schema); err != nil {
		return fmt.Errorf("failed to get quality profile schema: %w", err)
	}

	// Build profile from schema with our tier preferences
	resource := a.profileFromSchema(&schema, profile)
	return c.Post(ctx, "/api/v1/qualityprofile", resource, nil)
}

// updateQualityProfile updates a quality profile using schema for complete structure
func (a *Adapter) updateQualityProfile(ctx context.Context, c *httpclient.Client, id int, profile *irv1.AudioQualityIR) error {
	// Get schema to get full items structure
	var schema QualityProfileResource
	if err := c.Get(ctx, "/api/v1/qualityprofile/schema", &schema); err != nil {
		return fmt.Errorf("failed to get quality profile schema: %w", err)
	}

	// Build profile from schema with our tier preferences
	resource := a.profileFromSchema(&schema, profile)
	resource.ID = id
	return c.Put(ctx, fmt.Sprintf("/api/v1/qualityprofile/%d", id), resource, nil)
}

// profileFromSchema builds a quality profile from schema with tier preferences applied
func (a *Adapter) profileFromSchema(schema *QualityProfileResource, profile *irv1.AudioQualityIR) QualityProfileResource {
	resource := QualityProfileResource{
		Name:              profile.ProfileName,
		UpgradeAllowed:    profile.UpgradeAllowed,
		Items:             make([]QualityProfileItem, len(schema.Items)),
		MinFormatScore:    0,
		CutoffFormatScore: 0,
		FormatItems:       []interface{}{}, // Empty slice, not nil
	}

	// Build tier allowed map
	tierAllowed := make(map[string]bool)
	for _, tier := range profile.Tiers {
		tierAllowed[tier.Tier] = tier.Allowed
	}

	// Copy schema items and apply our tier preferences
	var cutoffID int
	for i, item := range schema.Items {
		// Deep copy the item to avoid reference issues
		resource.Items[i] = copyQualityItem(item)

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

// copyQualityItem creates a deep copy of a QualityProfileItem
func copyQualityItem(item QualityProfileItem) QualityProfileItem {
	copied := QualityProfileItem{
		ID:      item.ID,
		Name:    item.Name,
		Allowed: item.Allowed,
		Items:   make([]QualityProfileItem, len(item.Items)), // Always create slice, never nil
	}

	// Copy quality if present
	if item.Quality != nil {
		copied.Quality = &Quality{
			ID:   item.Quality.ID,
			Name: item.Quality.Name,
		}
	}

	// Recursively copy nested items
	for i, subItem := range item.Items {
		copied.Items[i] = copyQualityItem(subItem)
	}

	return copied
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
func (a *Adapter) createDownloadClient(ctx context.Context, c *httpclient.Client, dc irv1.DownloadClientIR, tagID int) error {
	resource := a.downloadClientFromIR(dc, tagID)
	return c.Post(ctx, "/api/v1/downloadclient", resource, nil)
}

// updateDownloadClient updates a download client
func (a *Adapter) updateDownloadClient(ctx context.Context, c *httpclient.Client, id int, dc irv1.DownloadClientIR, tagID int) error {
	resource := a.downloadClientFromIR(dc, tagID)
	resource.ID = id
	return c.Put(ctx, fmt.Sprintf("/api/v1/downloadclient/%d", id), resource, nil)
}

// downloadClientFromIR converts IR to Lidarr download client
func (a *Adapter) downloadClientFromIR(dc irv1.DownloadClientIR, tagID int) DownloadClientResource {
	resource := DownloadClientResource{
		BaseDownloadClientResource: shared.BaseDownloadClientResource{
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
		},
		RemoveCompletedDownloads: dc.RemoveCompletedDownloads,
		RemoveFailedDownloads:    dc.RemoveFailedDownloads,
	}
	return resource
}

// createIndexer creates an indexer
func (a *Adapter) createIndexer(ctx context.Context, c *httpclient.Client, idx irv1.IndexerIR, tagID int) error {
	resource := a.indexerFromIR(idx, tagID)
	return c.Post(ctx, "/api/v1/indexer", resource, nil)
}

// updateIndexer updates an indexer
func (a *Adapter) updateIndexer(ctx context.Context, c *httpclient.Client, id int, idx irv1.IndexerIR, tagID int) error {
	resource := a.indexerFromIR(idx, tagID)
	resource.ID = id
	return c.Put(ctx, fmt.Sprintf("/api/v1/indexer/%d", id), resource, nil)
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
func (a *Adapter) createRootFolder(ctx context.Context, c *httpclient.Client, rf irv1.RootFolderIR) error {
	// Lidarr requires DefaultMetadataProfileId and DefaultQualityProfileId
	// Fetch them to get valid default values

	// Get first metadata profile
	var metadataProfiles []MetadataProfileResource
	if err := c.Get(ctx, "/api/v1/metadataprofile", &metadataProfiles); err != nil {
		return fmt.Errorf("failed to get metadata profiles: %w", err)
	}
	metadataProfileID := 1 // Default fallback
	if len(metadataProfiles) > 0 {
		metadataProfileID = metadataProfiles[0].ID
	}

	// Get first quality profile (prefer our managed one if it exists)
	var qualityProfiles []QualityProfileResource
	if err := c.Get(ctx, "/api/v1/qualityprofile", &qualityProfiles); err != nil {
		return fmt.Errorf("failed to get quality profiles: %w", err)
	}
	qualityProfileID := 1 // Default fallback
	for _, qp := range qualityProfiles {
		if len(qp.Name) > 9 && qp.Name[:9] == "nebularr-" {
			qualityProfileID = qp.ID
			break
		}
	}
	if qualityProfileID == 1 && len(qualityProfiles) > 0 {
		qualityProfileID = qualityProfiles[0].ID
	}

	// Generate name from path if not provided
	name := rf.Name
	if name == "" {
		// Extract last component of path as name
		parts := splitPath(rf.Path)
		if len(parts) > 0 {
			name = parts[len(parts)-1]
		} else {
			name = rf.Path
		}
	}

	resource := RootFolderResource{
		Path:                     rf.Path,
		Name:                     name,
		DefaultMonitorOption:     rf.DefaultMonitor,
		DefaultMetadataProfileId: metadataProfileID,
		DefaultQualityProfileId:  qualityProfileID,
	}
	if resource.DefaultMonitorOption == "" {
		resource.DefaultMonitorOption = "all"
	}
	return c.Post(ctx, "/api/v1/rootfolder", resource, nil)
}

// splitPath splits a path into components
func splitPath(path string) []string {
	var parts []string
	current := ""
	for _, c := range path {
		if c == '/' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(c)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}

// updateNaming updates the naming configuration
func (a *Adapter) updateNaming(ctx context.Context, c *httpclient.Client, naming *irv1.LidarrNamingIR) error {
	resource := NamingConfigResource{
		ID:                       namingConfigID,
		RenameTracks:             naming.RenameTracks,
		ReplaceIllegalCharacters: naming.ReplaceIllegalCharacters,
		StandardTrackFormat:      naming.StandardTrackFormat,
		MultiDiscTrackFormat:     naming.MultiDiscTrackFormat,
		ArtistFolderFormat:       naming.ArtistFolderFormat,
		AlbumFolderFormat:        naming.AlbumFolderFormat,
	}
	return c.Put(ctx, "/api/v1/config/naming", resource, nil)
}
