package sonarr

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
		return a.createQualityProfile(ctx, c, change.Payload.(*irv1.VideoQualityIR))
	case adapters.ResourceCustomFormat:
		return a.createCustomFormat(ctx, c, change.Payload.(*irv1.CustomFormatIR))
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
	default:
		return fmt.Errorf("unsupported resource type for create: %s", change.ResourceType)
	}
}

// applyUpdate updates an existing resource
func (a *Adapter) applyUpdate(ctx context.Context, c *httpClient, change adapters.Change, tagID int) error {
	switch change.ResourceType {
	case adapters.ResourceQualityProfile:
		return a.updateQualityProfile(ctx, c, *change.ID, change.Payload.(*irv1.VideoQualityIR))
	case adapters.ResourceCustomFormat:
		return a.updateCustomFormat(ctx, c, change.Payload.(*irv1.CustomFormatIR))
	case adapters.ResourceDownloadClient:
		return a.updateDownloadClient(ctx, c, *change.ID, change.Payload.(irv1.DownloadClientIR), tagID)
	case adapters.ResourceIndexer:
		return a.updateIndexer(ctx, c, *change.ID, change.Payload.(irv1.IndexerIR), tagID)
	case adapters.ResourceNamingConfig:
		return a.updateNaming(ctx, c, change.Payload.(*irv1.SonarrNamingIR))
	case adapters.ResourceRemotePathMapping:
		return a.updateRemotePathMapping(ctx, c, change.Payload.(*irv1.RemotePathMappingIR))
	case adapters.ResourceNotification:
		return a.updateNotification(ctx, c, change.Payload.(*irv1.NotificationIR), tagID)
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
		return c.delete(ctx, fmt.Sprintf("/api/v3/qualityprofile/%d", *change.ID))
	case adapters.ResourceCustomFormat:
		return a.deleteCustomFormat(ctx, c, *change.ID)
	case adapters.ResourceDownloadClient:
		return c.delete(ctx, fmt.Sprintf("/api/v3/downloadclient/%d", *change.ID))
	case adapters.ResourceIndexer:
		return c.delete(ctx, fmt.Sprintf("/api/v3/indexer/%d", *change.ID))
	case adapters.ResourceRootFolder:
		return c.delete(ctx, fmt.Sprintf("/api/v3/rootfolder/%d", *change.ID))
	case adapters.ResourceRemotePathMapping:
		return a.deleteRemotePathMapping(ctx, c, *change.ID)
	case adapters.ResourceNotification:
		return a.deleteNotification(ctx, c, *change.ID)
	default:
		return fmt.Errorf("unsupported resource type for delete: %s", change.ResourceType)
	}
}

// createQualityProfile creates a quality profile using the schema
func (a *Adapter) createQualityProfile(ctx context.Context, c *httpClient, profile *irv1.VideoQualityIR) error {
	// Fetch schema to get all quality items with proper structure
	var schema QualityProfileResource
	if err := c.get(ctx, "/api/v3/qualityprofile/schema", &schema); err != nil {
		return fmt.Errorf("failed to get quality profile schema: %w", err)
	}

	// Build allowed qualities map from tiers
	allowedQualities := a.buildAllowedQualitiesMap(profile.Tiers)

	// Use schema items and mark appropriate ones as allowed
	items, cutoffID := a.processSchemaItems(schema.Items, allowedQualities, profile.Cutoff)

	resource := QualityProfileResource{
		Name:                  profile.ProfileName,
		UpgradeAllowed:        profile.UpgradeAllowed,
		Cutoff:                cutoffID,
		Items:                 items,
		FormatItems:           schema.FormatItems,
		MinFormatScore:        schema.MinFormatScore,
		MinUpgradeFormatScore: schema.MinUpgradeFormatScore,
		CutoffFormatScore:     schema.CutoffFormatScore,
	}

	return c.post(ctx, "/api/v3/qualityprofile", resource, nil)
}

// updateQualityProfile updates a quality profile using the schema
func (a *Adapter) updateQualityProfile(ctx context.Context, c *httpClient, id int, profile *irv1.VideoQualityIR) error {
	// Fetch schema to get all quality items with proper structure
	var schema QualityProfileResource
	if err := c.get(ctx, "/api/v3/qualityprofile/schema", &schema); err != nil {
		return fmt.Errorf("failed to get quality profile schema: %w", err)
	}

	// Build allowed qualities map from tiers
	allowedQualities := a.buildAllowedQualitiesMap(profile.Tiers)

	// Use schema items and mark appropriate ones as allowed
	items, cutoffID := a.processSchemaItems(schema.Items, allowedQualities, profile.Cutoff)

	resource := QualityProfileResource{
		ID:                    id,
		Name:                  profile.ProfileName,
		UpgradeAllowed:        profile.UpgradeAllowed,
		Cutoff:                cutoffID,
		Items:                 items,
		FormatItems:           schema.FormatItems,
		MinFormatScore:        schema.MinFormatScore,
		MinUpgradeFormatScore: schema.MinUpgradeFormatScore,
		CutoffFormatScore:     schema.CutoffFormatScore,
	}

	return c.put(ctx, fmt.Sprintf("/api/v3/qualityprofile/%d", id), resource, nil)
}

// buildAllowedQualitiesMap creates a map of resolution -> sources that are allowed
func (a *Adapter) buildAllowedQualitiesMap(tiers []irv1.VideoQualityTierIR) map[string]map[string]bool {
	allowed := make(map[string]map[string]bool)
	for _, tier := range tiers {
		if !tier.Allowed {
			continue
		}
		res := tier.Resolution
		if allowed[res] == nil {
			allowed[res] = make(map[string]bool)
		}
		for _, source := range tier.Sources {
			allowed[res][source] = true
		}
	}
	return allowed
}

// processSchemaItems processes schema items and marks allowed ones
func (a *Adapter) processSchemaItems(schemaItems []QualityProfileItem, allowedQualities map[string]map[string]bool, cutoffTier irv1.VideoQualityTierIR) ([]QualityProfileItem, int) {
	items := make([]QualityProfileItem, len(schemaItems))
	cutoffID := 1 // Default cutoff

	for i, schemaItem := range schemaItems {
		item := schemaItem

		// Check if this is a group (has nested items)
		if len(item.Items) > 0 {
			// Process group items
			groupItems := make([]QualityProfileItem, len(item.Items))
			groupAllowed := false
			for j, subItem := range item.Items {
				groupItems[j] = subItem
				if subItem.Quality != nil {
					isAllowed := a.isQualityAllowed(subItem.Quality, allowedQualities)
					groupItems[j].Allowed = isAllowed
					if isAllowed {
						groupAllowed = true
					}
					// Check for cutoff
					if a.isQualityCutoff(subItem.Quality, cutoffTier) && item.ID > 0 {
						cutoffID = item.ID
					}
				}
			}
			item.Items = groupItems
			item.Allowed = groupAllowed
		} else if item.Quality != nil {
			// Single quality item
			item.Allowed = a.isQualityAllowed(item.Quality, allowedQualities)
			// Check for cutoff
			if a.isQualityCutoff(item.Quality, cutoffTier) && item.Quality.ID > 0 {
				cutoffID = item.Quality.ID
			}
		}

		items[i] = item
	}

	return items, cutoffID
}

// isQualityAllowed checks if a quality matches any allowed tier
func (a *Adapter) isQualityAllowed(quality *Quality, allowedQualities map[string]map[string]bool) bool {
	if quality == nil {
		return false
	}
	res := resolutionToString(quality.Resolution)
	sources := allowedQualities[res]
	if sources == nil {
		return false
	}
	// Map Sonarr source names to our source names
	source := mapSonarrSource(quality.Source)
	return sources[source]
}

// isQualityCutoff checks if this quality should be the cutoff
func (a *Adapter) isQualityCutoff(quality *Quality, cutoffTier irv1.VideoQualityTierIR) bool {
	if quality == nil {
		return false
	}
	res := resolutionToString(quality.Resolution)
	if res != cutoffTier.Resolution {
		return false
	}
	source := mapSonarrSource(quality.Source)
	for _, s := range cutoffTier.Sources {
		if s == source {
			return true
		}
	}
	return false
}

// resolutionToString converts resolution int to string
func resolutionToString(res int) string {
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
		return ""
	}
}

// mapSonarrSource maps Sonarr source names to our normalized names
func mapSonarrSource(source string) string {
	switch source {
	case "television", "televisionRaw":
		return "hdtv"
	case "web", "webdl":
		return "webdl"
	case "webRip":
		return "webrip"
	case "bluray", "blurayRaw":
		return "bluray"
	case "dvd":
		return "dvd"
	default:
		return source
	}
}

// createDownloadClient creates a download client
func (a *Adapter) createDownloadClient(ctx context.Context, c *httpClient, dc irv1.DownloadClientIR, tagID int) error {
	resource := a.downloadClientFromIR(dc, tagID)
	return c.post(ctx, "/api/v3/downloadclient", resource, nil)
}

// updateDownloadClient updates a download client
func (a *Adapter) updateDownloadClient(ctx context.Context, c *httpClient, id int, dc irv1.DownloadClientIR, tagID int) error {
	resource := a.downloadClientFromIR(dc, tagID)
	resource.ID = id
	return c.put(ctx, fmt.Sprintf("/api/v3/downloadclient/%d", id), resource, nil)
}

// downloadClientFromIR converts IR to Sonarr download client
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
			{Name: "tvCategory", Value: dc.Category},
		},
		RemoveCompletedDownloads: dc.RemoveCompletedDownloads,
		RemoveFailedDownloads:    dc.RemoveFailedDownloads,
	}
	return resource
}

// createIndexer creates an indexer
func (a *Adapter) createIndexer(ctx context.Context, c *httpClient, idx irv1.IndexerIR, tagID int) error {
	resource := a.indexerFromIR(idx, tagID)
	return c.post(ctx, "/api/v3/indexer", resource, nil)
}

// updateIndexer updates an indexer
func (a *Adapter) updateIndexer(ctx context.Context, c *httpClient, id int, idx irv1.IndexerIR, tagID int) error {
	resource := a.indexerFromIR(idx, tagID)
	resource.ID = id
	return c.put(ctx, fmt.Sprintf("/api/v3/indexer/%d", id), resource, nil)
}

// indexerFromIR converts IR to Sonarr indexer
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
			{Name: "minimumSeeders", Value: idx.MinimumSeeders},
		},
	}
	return resource
}

// createRootFolder creates a root folder
func (a *Adapter) createRootFolder(ctx context.Context, c *httpClient, rf irv1.RootFolderIR) error {
	resource := RootFolderResource{
		Path: rf.Path,
	}
	return c.post(ctx, "/api/v3/rootfolder", resource, nil)
}

// updateNaming updates the naming configuration
func (a *Adapter) updateNaming(ctx context.Context, c *httpClient, naming *irv1.SonarrNamingIR) error {
	resource := NamingConfigResource{
		ID:                       namingConfigID,
		RenameEpisodes:           naming.RenameEpisodes,
		ReplaceIllegalCharacters: naming.ReplaceIllegalCharacters,
		StandardEpisodeFormat:    naming.StandardEpisodeFormat,
		DailyEpisodeFormat:       naming.DailyEpisodeFormat,
		AnimeEpisodeFormat:       naming.AnimeEpisodeFormat,
		SeriesFolderFormat:       naming.SeriesFolderFormat,
		SeasonFolderFormat:       naming.SeasonFolderFormat,
		SpecialsFolderFormat:     naming.SpecialsFolderFormat,
		MultiEpisodeStyle:        naming.MultiEpisodeStyle,
	}
	return c.put(ctx, "/api/v3/config/naming", resource, nil)
}
