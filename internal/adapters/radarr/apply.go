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

// applyCreate creates a new resource in Radarr
func (a *Adapter) applyCreate(ctx context.Context, c *client.Client, change adapters.Change, tagID int) error {
	switch change.ResourceType {
	case adapters.ResourceQualityProfile:
		return a.createQualityProfile(ctx, c, change.Payload.(*irv1.VideoQualityIR))
	case adapters.ResourceCustomFormat:
		return a.createCustomFormat(ctx, c, change.Payload.(*irv1.CustomFormatIR))
	case adapters.ResourceDownloadClient:
		return a.createDownloadClient(ctx, c, change.Payload.(*irv1.DownloadClientIR), tagID)
	case adapters.ResourceIndexer:
		return a.createIndexer(ctx, c, change.Payload.(*irv1.IndexerIR), tagID)
	case adapters.ResourceRootFolder:
		return a.createRootFolder(ctx, c, change.Payload.(*irv1.RootFolderIR))
	default:
		return fmt.Errorf("unknown resource type for create: %s", change.ResourceType)
	}
}

// applyUpdate updates an existing resource in Radarr
func (a *Adapter) applyUpdate(ctx context.Context, c *client.Client, change adapters.Change, tagID int) error {
	switch change.ResourceType {
	case adapters.ResourceQualityProfile:
		return a.updateQualityProfile(ctx, c, change.Payload.(*irv1.VideoQualityIR))
	case adapters.ResourceCustomFormat:
		return a.updateCustomFormat(ctx, c, change.Payload.(*irv1.CustomFormatIR))
	case adapters.ResourceDownloadClient:
		return a.updateDownloadClient(ctx, c, change.Payload.(*irv1.DownloadClientIR), tagID)
	case adapters.ResourceIndexer:
		return a.updateIndexer(ctx, c, change.Payload.(*irv1.IndexerIR), tagID)
	case adapters.ResourceNamingConfig:
		return a.updateNamingConfig(ctx, c, change.Payload.(*irv1.RadarrNamingIR))
	default:
		return fmt.Errorf("unknown resource type for update: %s", change.ResourceType)
	}
}

// applyDelete deletes a resource from Radarr
func (a *Adapter) applyDelete(ctx context.Context, c *client.Client, change adapters.Change) error {
	switch change.ResourceType {
	case adapters.ResourceQualityProfile:
		return a.deleteQualityProfile(ctx, c, change.Name)
	case adapters.ResourceCustomFormat:
		return a.deleteCustomFormat(ctx, c, change.Name)
	case adapters.ResourceDownloadClient:
		return a.deleteDownloadClient(ctx, c, change.Name)
	case adapters.ResourceIndexer:
		return a.deleteIndexer(ctx, c, change.Name)
	default:
		return fmt.Errorf("unknown resource type for delete: %s", change.ResourceType)
	}
}

// Quality Profile operations

func (a *Adapter) createQualityProfile(ctx context.Context, c *client.Client, ir *irv1.VideoQualityIR) error {
	profile := a.irToQualityProfile(ir)

	resp, err := c.PostApiV3Qualityprofile(ctx, profile)
	if err != nil {
		return fmt.Errorf("failed to create quality profile: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

func (a *Adapter) updateQualityProfile(ctx context.Context, c *client.Client, ir *irv1.VideoQualityIR) error {
	// First, find the existing profile by name
	profileID, err := a.findQualityProfileIDByName(ctx, c, ir.ProfileName)
	if err != nil {
		return err
	}

	profile := a.irToQualityProfile(ir)
	profile.Id = intPtr(profileID)

	resp, err := c.PutApiV3QualityprofileId(ctx, fmt.Sprintf("%d", profileID), profile)
	if err != nil {
		return fmt.Errorf("failed to update quality profile: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

func (a *Adapter) deleteQualityProfile(ctx context.Context, c *client.Client, name string) error {
	profileID, err := a.findQualityProfileIDByName(ctx, c, name)
	if err != nil {
		return err
	}

	resp, err := c.DeleteApiV3QualityprofileId(ctx, int32(profileID))
	if err != nil {
		return fmt.Errorf("failed to delete quality profile: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

func (a *Adapter) findQualityProfileIDByName(ctx context.Context, c *client.Client, name string) (int, error) {
	resp, err := c.GetApiV3Qualityprofile(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get quality profiles: %w", err)
	}
	defer resp.Body.Close()

	var profiles []client.QualityProfileResource
	if err := json.NewDecoder(resp.Body).Decode(&profiles); err != nil {
		return 0, fmt.Errorf("failed to decode quality profiles: %w", err)
	}

	for _, p := range profiles {
		if ptrToString(p.Name) == name {
			return ptrToInt(p.Id), nil
		}
	}

	return 0, fmt.Errorf("quality profile not found: %s", name)
}

func (a *Adapter) irToQualityProfile(ir *irv1.VideoQualityIR) client.PostApiV3QualityprofileJSONRequestBody {
	profile := client.QualityProfileResource{
		Name:           stringPtr(ir.ProfileName),
		UpgradeAllowed: boolPtr(ir.UpgradeAllowed),
		MinFormatScore: intPtr(ir.MinimumCustomFormatScore),
	}

	if ir.UpgradeUntilCustomFormatScore > 0 {
		profile.CutoffFormatScore = intPtr(ir.UpgradeUntilCustomFormatScore)
	}

	// TODO: Convert tiers to Items and set Cutoff
	// This requires mapping our abstract tiers to Radarr's quality definitions

	return profile
}

// Custom Format operations

func (a *Adapter) createCustomFormat(ctx context.Context, c *client.Client, ir *irv1.CustomFormatIR) error {
	cf := a.irToCustomFormat(ir)

	resp, err := c.PostApiV3Customformat(ctx, cf)
	if err != nil {
		return fmt.Errorf("failed to create custom format: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

func (a *Adapter) updateCustomFormat(ctx context.Context, c *client.Client, ir *irv1.CustomFormatIR) error {
	cfID, err := a.findCustomFormatIDByName(ctx, c, ir.Name)
	if err != nil {
		return err
	}

	cf := a.irToCustomFormat(ir)
	cf.Id = intPtr(cfID)

	resp, err := c.PutApiV3CustomformatId(ctx, fmt.Sprintf("%d", cfID), cf)
	if err != nil {
		return fmt.Errorf("failed to update custom format: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

func (a *Adapter) deleteCustomFormat(ctx context.Context, c *client.Client, name string) error {
	cfID, err := a.findCustomFormatIDByName(ctx, c, name)
	if err != nil {
		return err
	}

	resp, err := c.DeleteApiV3CustomformatId(ctx, int32(cfID))
	if err != nil {
		return fmt.Errorf("failed to delete custom format: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

func (a *Adapter) findCustomFormatIDByName(ctx context.Context, c *client.Client, name string) (int, error) {
	resp, err := c.GetApiV3Customformat(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get custom formats: %w", err)
	}
	defer resp.Body.Close()

	var formats []client.CustomFormatResource
	if err := json.NewDecoder(resp.Body).Decode(&formats); err != nil {
		return 0, fmt.Errorf("failed to decode custom formats: %w", err)
	}

	for _, f := range formats {
		if ptrToString(f.Name) == name {
			return ptrToInt(f.Id), nil
		}
	}

	return 0, fmt.Errorf("custom format not found: %s", name)
}

func (a *Adapter) irToCustomFormat(ir *irv1.CustomFormatIR) client.CustomFormatResource {
	cf := client.CustomFormatResource{
		Name:                            stringPtr(ir.Name),
		IncludeCustomFormatWhenRenaming: boolPtr(ir.IncludeWhenRenaming),
	}

	// Convert specifications
	specs := make([]client.CustomFormatSpecificationSchema, 0, len(ir.Specifications))
	for _, spec := range ir.Specifications {
		s := client.CustomFormatSpecificationSchema{
			Name:     stringPtr(spec.Name),
			Negate:   boolPtr(spec.Negate),
			Required: boolPtr(spec.Required),
		}
		// TODO: Set Implementation and Fields based on spec.Type
		specs = append(specs, s)
	}
	cf.Specifications = &specs

	return cf
}

// Download Client operations

func (a *Adapter) createDownloadClient(ctx context.Context, c *client.Client, ir *irv1.DownloadClientIR, tagID int) error {
	dc := a.irToDownloadClient(ir, tagID)

	resp, err := c.PostApiV3Downloadclient(ctx, nil, dc)
	if err != nil {
		return fmt.Errorf("failed to create download client: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

func (a *Adapter) updateDownloadClient(ctx context.Context, c *client.Client, ir *irv1.DownloadClientIR, tagID int) error {
	dcID, err := a.findDownloadClientIDByName(ctx, c, ir.Name)
	if err != nil {
		return err
	}

	dc := a.irToDownloadClient(ir, tagID)
	dc.Id = intPtr(dcID)

	resp, err := c.PutApiV3DownloadclientId(ctx, int32(dcID), nil, dc)
	if err != nil {
		return fmt.Errorf("failed to update download client: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

func (a *Adapter) deleteDownloadClient(ctx context.Context, c *client.Client, name string) error {
	dcID, err := a.findDownloadClientIDByName(ctx, c, name)
	if err != nil {
		return err
	}

	resp, err := c.DeleteApiV3DownloadclientId(ctx, int32(dcID))
	if err != nil {
		return fmt.Errorf("failed to delete download client: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

func (a *Adapter) findDownloadClientIDByName(ctx context.Context, c *client.Client, name string) (int, error) {
	resp, err := c.GetApiV3Downloadclient(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get download clients: %w", err)
	}
	defer resp.Body.Close()

	var clients []client.DownloadClientResource
	if err := json.NewDecoder(resp.Body).Decode(&clients); err != nil {
		return 0, fmt.Errorf("failed to decode download clients: %w", err)
	}

	for _, dc := range clients {
		if ptrToString(dc.Name) == name {
			return ptrToInt(dc.Id), nil
		}
	}

	return 0, fmt.Errorf("download client not found: %s", name)
}

func (a *Adapter) irToDownloadClient(ir *irv1.DownloadClientIR, tagID int) client.DownloadClientResource {
	dc := client.DownloadClientResource{
		Name:                     stringPtr(ir.Name),
		Enable:                   boolPtr(ir.Enable),
		Priority:                 intPtr(ir.Priority),
		Implementation:           stringPtr(a.normalizeImplementation(ir.Implementation)),
		RemoveCompletedDownloads: boolPtr(ir.RemoveCompletedDownloads),
		RemoveFailedDownloads:    boolPtr(ir.RemoveFailedDownloads),
		Tags:                     &[]int32{int32(tagID)},
	}

	// Set protocol
	if ir.Protocol == irv1.ProtocolTorrent {
		protocol := client.DownloadProtocolTorrent
		dc.Protocol = &protocol
	} else if ir.Protocol == irv1.ProtocolUsenet {
		protocol := client.DownloadProtocolUsenet
		dc.Protocol = &protocol
	}

	// TODO: Set Fields based on implementation type

	return dc
}

func (a *Adapter) normalizeImplementation(impl string) string {
	// Map our lowercase implementation names to Radarr's pascal case
	switch impl {
	case "qbittorrent":
		return "QBittorrent"
	case "transmission":
		return "Transmission"
	case "deluge":
		return "Deluge"
	case "rtorrent":
		return "RTorrent"
	case "sabnzbd":
		return "Sabnzbd"
	case "nzbget":
		return "NzbGet"
	default:
		return impl
	}
}

// Indexer operations

func (a *Adapter) createIndexer(ctx context.Context, c *client.Client, ir *irv1.IndexerIR, tagID int) error {
	idx := a.irToIndexer(ir, tagID)

	resp, err := c.PostApiV3Indexer(ctx, nil, idx)
	if err != nil {
		return fmt.Errorf("failed to create indexer: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

func (a *Adapter) updateIndexer(ctx context.Context, c *client.Client, ir *irv1.IndexerIR, tagID int) error {
	idxID, err := a.findIndexerIDByName(ctx, c, ir.Name)
	if err != nil {
		return err
	}

	idx := a.irToIndexer(ir, tagID)
	idx.Id = intPtr(idxID)

	resp, err := c.PutApiV3IndexerId(ctx, int32(idxID), nil, idx)
	if err != nil {
		return fmt.Errorf("failed to update indexer: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

func (a *Adapter) deleteIndexer(ctx context.Context, c *client.Client, name string) error {
	idxID, err := a.findIndexerIDByName(ctx, c, name)
	if err != nil {
		return err
	}

	resp, err := c.DeleteApiV3IndexerId(ctx, int32(idxID))
	if err != nil {
		return fmt.Errorf("failed to delete indexer: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

func (a *Adapter) findIndexerIDByName(ctx context.Context, c *client.Client, name string) (int, error) {
	resp, err := c.GetApiV3Indexer(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get indexers: %w", err)
	}
	defer resp.Body.Close()

	var indexers []client.IndexerResource
	if err := json.NewDecoder(resp.Body).Decode(&indexers); err != nil {
		return 0, fmt.Errorf("failed to decode indexers: %w", err)
	}

	for _, idx := range indexers {
		if ptrToString(idx.Name) == name {
			return ptrToInt(idx.Id), nil
		}
	}

	return 0, fmt.Errorf("indexer not found: %s", name)
}

func (a *Adapter) irToIndexer(ir *irv1.IndexerIR, tagID int) client.IndexerResource {
	idx := client.IndexerResource{
		Name:                    stringPtr(ir.Name),
		Priority:                intPtr(ir.Priority),
		Implementation:          stringPtr(ir.Implementation),
		EnableRss:               boolPtr(ir.Enable && ir.EnableRss),
		EnableAutomaticSearch:   boolPtr(ir.Enable && ir.EnableAutomaticSearch),
		EnableInteractiveSearch: boolPtr(ir.Enable && ir.EnableInteractiveSearch),
		Tags:                    &[]int32{int32(tagID)},
	}

	// Set protocol
	if ir.Protocol == irv1.ProtocolTorrent {
		protocol := client.DownloadProtocolTorrent
		idx.Protocol = &protocol
	} else if ir.Protocol == irv1.ProtocolUsenet {
		protocol := client.DownloadProtocolUsenet
		idx.Protocol = &protocol
	}

	// TODO: Set Fields based on implementation type

	return idx
}

// Root Folder operations

func (a *Adapter) createRootFolder(ctx context.Context, c *client.Client, ir *irv1.RootFolderIR) error {
	rf := client.RootFolderResource{
		Path: stringPtr(ir.Path),
	}

	resp, err := c.PostApiV3Rootfolder(ctx, rf)
	if err != nil {
		return fmt.Errorf("failed to create root folder: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

// Naming Config operations

func (a *Adapter) updateNamingConfig(ctx context.Context, c *client.Client, ir *irv1.RadarrNamingIR) error {
	// Get the current naming config to get its ID
	resp, err := c.GetApiV3ConfigNaming(ctx)
	if err != nil {
		return fmt.Errorf("failed to get naming config: %w", err)
	}
	defer resp.Body.Close()

	var current client.NamingConfigResource
	if err := json.NewDecoder(resp.Body).Decode(&current); err != nil {
		return fmt.Errorf("failed to decode naming config: %w", err)
	}

	// Update the config
	colonFormat := intToColonReplacement(ir.ColonReplacementFormat)
	updated := client.NamingConfigResource{
		Id:                       current.Id,
		RenameMovies:             boolPtr(ir.RenameMovies),
		ReplaceIllegalCharacters: boolPtr(ir.ReplaceIllegalCharacters),
		ColonReplacementFormat:   &colonFormat,
		StandardMovieFormat:      stringPtr(ir.StandardMovieFormat),
		MovieFolderFormat:        stringPtr(ir.MovieFolderFormat),
	}

	idStr := fmt.Sprintf("%d", ptrToInt(current.Id))
	resp, err = c.PutApiV3ConfigNamingId(ctx, idStr, updated)
	if err != nil {
		return fmt.Errorf("failed to update naming config: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}
