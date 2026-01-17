package radarr

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
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
	profile, err := a.irToQualityProfile(ctx, c, ir)
	if err != nil {
		return fmt.Errorf("failed to build quality profile: %w", err)
	}

	resp, err := c.PostApiV3Qualityprofile(ctx, profile)
	if err != nil {
		return fmt.Errorf("failed to create quality profile: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}

func (a *Adapter) updateQualityProfile(ctx context.Context, c *client.Client, ir *irv1.VideoQualityIR) error {
	// First, find the existing profile by name
	profileID, err := a.findQualityProfileIDByName(ctx, c, ir.ProfileName)
	if err != nil {
		return err
	}

	profile, err := a.irToQualityProfile(ctx, c, ir)
	if err != nil {
		return fmt.Errorf("failed to build quality profile: %w", err)
	}
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

func (a *Adapter) irToQualityProfile(ctx context.Context, c *client.Client, ir *irv1.VideoQualityIR) (client.PostApiV3QualityprofileJSONRequestBody, error) {
	// Set default language to "Original" (id: -2)
	langID := int32(-2)
	langName := "Original"
	profile := client.QualityProfileResource{
		Name:           stringPtr(ir.ProfileName),
		UpgradeAllowed: boolPtr(ir.UpgradeAllowed),
		MinFormatScore: intPtr(ir.MinimumCustomFormatScore),
		Language: &client.Language{
			Id:   &langID,
			Name: &langName,
		},
	}

	if ir.UpgradeUntilCustomFormatScore > 0 {
		profile.CutoffFormatScore = intPtr(ir.UpgradeUntilCustomFormatScore)
	}

	// Fetch quality definitions to map our tiers to Radarr's quality IDs
	qualityDefs, err := a.getQualityDefinitions(ctx, c)
	if err != nil {
		return profile, fmt.Errorf("failed to get quality definitions: %w", err)
	}

	// Build Items array from tiers
	items, cutoffID := a.buildQualityItems(ir.Tiers, ir.Cutoff, qualityDefs)
	profile.Items = &items

	// Set cutoff if we found a matching quality
	if cutoffID > 0 {
		profile.Cutoff = intPtr(cutoffID)
	}

	return profile, nil
}

// qualityDef holds parsed quality definition info
type qualityDef struct {
	ID         int
	Name       string
	Source     string
	Resolution int
	Modifier   string
}

// getQualityDefinitions fetches and parses quality definitions from Radarr
func (a *Adapter) getQualityDefinitions(ctx context.Context, c *client.Client) ([]qualityDef, error) {
	resp, err := c.GetApiV3Qualitydefinition(ctx)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var defs []client.QualityDefinitionResource
	if err := json.NewDecoder(resp.Body).Decode(&defs); err != nil {
		return nil, fmt.Errorf("failed to decode quality definitions: %w", err)
	}

	result := make([]qualityDef, 0, len(defs))
	for _, def := range defs {
		if def.Quality == nil {
			continue
		}
		qd := qualityDef{
			ID:   ptrToInt(def.Quality.Id),
			Name: ptrToString(def.Quality.Name),
		}
		if def.Quality.Source != nil {
			qd.Source = string(*def.Quality.Source)
		}
		if def.Quality.Resolution != nil {
			qd.Resolution = int(*def.Quality.Resolution)
		}
		if def.Quality.Modifier != nil {
			qd.Modifier = string(*def.Quality.Modifier)
		}
		result = append(result, qd)
	}
	return result, nil
}

// buildQualityItems builds the Items array for a quality profile from IR tiers
func (a *Adapter) buildQualityItems(tiers []irv1.VideoQualityTierIR, cutoff irv1.VideoQualityTierIR, defs []qualityDef) ([]client.QualityProfileQualityItemResource, int) {
	var items []client.QualityProfileQualityItemResource
	cutoffID := 0

	// Create a lookup map for quality definitions by (resolution, source)
	qualityLookup := make(map[string]qualityDef)
	for _, def := range defs {
		key := fmt.Sprintf("%d-%s", def.Resolution, def.Source)
		qualityLookup[key] = def
	}

	// Process each tier as a group
	groupID := 1000 // Start with a high number for group IDs
	for _, tier := range tiers {
		res := parseResolution(tier.Resolution)
		if res == 0 {
			continue
		}

		// Build group items for this tier
		var groupItems []client.QualityProfileQualityItemResource
		for _, source := range tier.Sources {
			radarrSource := mapSourceToRadarr(source)
			key := fmt.Sprintf("%d-%s", res, radarrSource)

			if def, ok := qualityLookup[key]; ok {
				// Each item needs an empty Items array
				emptyItems := []client.QualityProfileQualityItemResource{}
				// Build full Quality object with all required fields
				qualitySource := client.QualitySource(def.Source)
				qualityModifier := client.Modifier(def.Modifier)
				qualityRes := int32(def.Resolution)
				item := client.QualityProfileQualityItemResource{
					Quality: &client.Quality{
						Id:         intPtr(def.ID),
						Name:       stringPtr(def.Name),
						Source:     &qualitySource,
						Resolution: &qualityRes,
						Modifier:   &qualityModifier,
					},
					Items:   &emptyItems,
					Allowed: boolPtr(tier.Allowed),
				}
				groupItems = append(groupItems, item)

				// Check if this is the cutoff tier
				if tier.Resolution == cutoff.Resolution && containsSource(cutoff.Sources, source) {
					cutoffID = def.ID
				}
			}
		}

		// Add as a group if multiple items, or as single item
		if len(groupItems) > 1 {
			groupName := fmt.Sprintf("%s", tier.Resolution)
			group := client.QualityProfileQualityItemResource{
				Id:      intPtr(groupID),
				Name:    stringPtr(groupName),
				Items:   &groupItems,
				Allowed: boolPtr(tier.Allowed),
			}
			items = append(items, group)
			groupID++

			// For groups, cutoff should be the group ID
			if cutoffID > 0 && tier.Resolution == cutoff.Resolution {
				cutoffID = groupID - 1 // Use the group ID we just assigned
			}
		} else if len(groupItems) == 1 {
			items = append(items, groupItems[0])
		}
	}

	return items, cutoffID
}

// parseResolution converts "2160p", "1080p", etc. to integer
func parseResolution(res string) int {
	switch res {
	case "2160p":
		return 2160
	case "1080p":
		return 1080
	case "720p":
		return 720
	case "480p":
		return 480
	case "360p":
		return 360
	default:
		return 0
	}
}

// mapSourceToRadarr converts our source names to Radarr's source names
func mapSourceToRadarr(source string) string {
	switch source {
	case "bluray":
		return "bluray"
	case "webdl":
		return "webdl"
	case "webrip":
		return "webrip"
	case "hdtv":
		return "tv"
	case "dvd":
		return "dvd"
	case "cam":
		return "cam"
	case "telesync":
		return "telesync"
	case "telecine":
		return "telecine"
	case "workprint":
		return "workprint"
	case "remux":
		return "bluray" // Remux is bluray source with remux modifier
	default:
		return source
	}
}

// containsSource checks if a source is in the list
func containsSource(sources []string, source string) bool {
	for _, s := range sources {
		if s == source {
			return true
		}
	}
	return false
}

// Custom Format operations

func (a *Adapter) createCustomFormat(ctx context.Context, c *client.Client, ir *irv1.CustomFormatIR) error {
	// Use minimal struct to avoid null serialization issues with generated client
	cf := a.buildMinimalCustomFormat(ir)

	// Serialize to JSON
	jsonBody, err := json.Marshal(cf)
	if err != nil {
		return fmt.Errorf("failed to marshal custom format: %w", err)
	}

	// Use WithBody to send our custom JSON
	resp, err := c.PostApiV3CustomformatWithBody(ctx, "application/json", bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create custom format: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		// Read response body for error details
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}

func (a *Adapter) updateCustomFormat(ctx context.Context, c *client.Client, ir *irv1.CustomFormatIR) error {
	cfID, err := a.findCustomFormatIDByName(ctx, c, ir.Name)
	if err != nil {
		return err
	}

	// Use minimal struct to avoid null serialization issues
	cf := a.buildMinimalCustomFormat(ir)
	cf.ID = &cfID

	// Serialize to JSON
	jsonBody, err := json.Marshal(cf)
	if err != nil {
		return fmt.Errorf("failed to marshal custom format: %w", err)
	}

	resp, err := c.PutApiV3CustomformatIdWithBody(ctx, fmt.Sprintf("%d", cfID), "application/json", bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to update custom format: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
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

// minimalField is a simplified field struct that only includes required fields
// This avoids the generated client.Field which serializes nulls for optional fields
type minimalField struct {
	Name  string      `json:"name"`
	Value interface{} `json:"value"`
}

// minimalSpec is a simplified specification struct
type minimalSpec struct {
	Name           string         `json:"name"`
	Implementation string         `json:"implementation"`
	Negate         bool           `json:"negate"`
	Required       bool           `json:"required"`
	Fields         []minimalField `json:"fields"`
}

// minimalCustomFormat is a simplified custom format struct for creation/update
type minimalCustomFormat struct {
	ID                              *int          `json:"id,omitempty"`
	Name                            string        `json:"name"`
	IncludeCustomFormatWhenRenaming bool          `json:"includeCustomFormatWhenRenaming"`
	Specifications                  []minimalSpec `json:"specifications"`
}

// buildMinimalCustomFormat creates a minimal custom format payload for creation
func (a *Adapter) buildMinimalCustomFormat(ir *irv1.CustomFormatIR) minimalCustomFormat {
	cf := minimalCustomFormat{
		Name:                            ir.Name,
		IncludeCustomFormatWhenRenaming: ir.IncludeWhenRenaming,
		Specifications:                  make([]minimalSpec, 0, len(ir.Specifications)),
	}

	for _, spec := range ir.Specifications {
		s := minimalSpec{
			Name:           spec.Name,
			Implementation: spec.Type,
			Negate:         spec.Negate,
			Required:       spec.Required,
			Fields:         a.buildMinimalFields(spec),
		}
		cf.Specifications = append(cf.Specifications, s)
	}

	return cf
}

// buildMinimalFields creates the Fields array for a custom format specification
func (a *Adapter) buildMinimalFields(spec irv1.FormatSpecIR) []minimalField {
	switch spec.Type {
	case "ReleaseTitleSpecification":
		return []minimalField{
			{
				Name:  "value",
				Value: spec.Value,
			},
		}
	case "SourceSpecification":
		return []minimalField{
			{
				Name:  "value",
				Value: a.sourceToInt(spec.Value),
			},
		}
	case "ResolutionSpecification":
		return []minimalField{
			{
				Name:  "value",
				Value: a.resolutionToInt(spec.Value),
			},
		}
	default:
		return []minimalField{
			{
				Name:  "value",
				Value: spec.Value,
			},
		}
	}
}

// sourceToInt converts source string to Radarr source enum value
func (a *Adapter) sourceToInt(source string) int {
	sources := map[string]int{
		"cam":       1,
		"telesync":  2,
		"telecine":  3,
		"workprint": 4,
		"dvd":       5,
		"tv":        6,
		"webdl":     7,
		"webrip":    8,
		"bluray":    9,
	}
	if v, ok := sources[source]; ok {
		return v
	}
	return 0
}

// resolutionToInt converts resolution string to Radarr resolution enum value
func (a *Adapter) resolutionToInt(res string) int {
	resolutions := map[string]int{
		"r360p":  360,
		"r480p":  480,
		"r576p":  576,
		"r720p":  720,
		"r1080p": 1080,
		"r2160p": 2160,
	}
	if v, ok := resolutions[res]; ok {
		return v
	}
	return 0
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
