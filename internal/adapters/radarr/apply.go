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
	defer func() { _ = resp.Body.Close() }()

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
	defer func() { _ = resp.Body.Close() }()

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
	defer func() { _ = resp.Body.Close() }()

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
	defer func() { _ = resp.Body.Close() }()

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
	// Fetch the quality profile schema - this gives us all items with proper structure
	resp, err := c.GetApiV3QualityprofileSchema(ctx)
	if err != nil {
		return client.QualityProfileResource{}, fmt.Errorf("failed to get quality profile schema: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return client.QualityProfileResource{}, fmt.Errorf("unexpected status code getting schema: %d", resp.StatusCode)
	}

	var profile client.QualityProfileResource
	if err := json.NewDecoder(resp.Body).Decode(&profile); err != nil {
		return client.QualityProfileResource{}, fmt.Errorf("failed to decode quality profile schema: %w", err)
	}

	// Set profile metadata
	profile.Name = stringPtr(ir.ProfileName)
	profile.UpgradeAllowed = boolPtr(ir.UpgradeAllowed)
	profile.MinFormatScore = intPtr(ir.MinimumCustomFormatScore)

	// Set default language to "Original" (id: -2)
	langID := int32(-2)
	langName := "Original"
	profile.Language = &client.Language{
		Id:   &langID,
		Name: &langName,
	}

	if ir.UpgradeUntilCustomFormatScore > 0 {
		profile.CutoffFormatScore = intPtr(ir.UpgradeUntilCustomFormatScore)
	}

	// Build a set of allowed quality IDs from our tiers
	allowedQualities := a.buildAllowedQualitySet(ir.Tiers)

	// Find cutoff quality ID
	cutoffID := a.findCutoffQualityID(ir.Cutoff, profile.Items)

	// Update items to mark allowed qualities
	if profile.Items != nil {
		a.markAllowedQualities(*profile.Items, allowedQualities)
	}

	// Set cutoff
	if cutoffID > 0 {
		profile.Cutoff = intPtr(cutoffID)
	}

	return profile, nil
}

// buildAllowedQualitySet builds a set of quality IDs that should be allowed based on tiers
func (a *Adapter) buildAllowedQualitySet(tiers []irv1.VideoQualityTierIR) map[int]bool {
	allowed := make(map[int]bool)

	// Map our tier definitions to Radarr quality names
	for _, tier := range tiers {
		if !tier.Allowed {
			continue
		}
		res := parseResolution(tier.Resolution)
		for _, source := range tier.Sources {
			// Add all matching quality IDs
			qualityName := a.buildQualityName(res, source)
			if id := a.qualityNameToID(qualityName); id > 0 {
				allowed[id] = true
			}
		}
	}

	return allowed
}

// buildQualityName builds the Radarr quality name from resolution and source
func (a *Adapter) buildQualityName(resolution int, source string) string {
	switch source {
	case "bluray":
		return fmt.Sprintf("Bluray-%dp", resolution)
	case "remux":
		return fmt.Sprintf("Remux-%dp", resolution)
	case "webdl":
		return fmt.Sprintf("WEBDL-%dp", resolution)
	case "webrip":
		return fmt.Sprintf("WEBRip-%dp", resolution)
	case "hdtv":
		return fmt.Sprintf("HDTV-%dp", resolution)
	case "dvd":
		return "DVD"
	default:
		return ""
	}
}

// qualityNameToID returns the Radarr quality ID for a quality name
func (a *Adapter) qualityNameToID(name string) int {
	// Standard Radarr quality IDs
	ids := map[string]int{
		"Unknown":      0,
		"SDTV":         1,
		"DVD":          2,
		"WEBDL-1080p":  3,
		"HDTV-720p":    4,
		"WEBDL-720p":   5,
		"Bluray-720p":  6,
		"Bluray-1080p": 7,
		"WEBDL-480p":   8,
		"HDTV-1080p":   9,
		"Raw-HD":       10,
		"WEBRip-480p":  12,
		"WEBRip-720p":  14,
		"WEBRip-1080p": 15,
		"HDTV-2160p":   16,
		"WEBRip-2160p": 17,
		"WEBDL-2160p":  18,
		"Bluray-2160p": 19,
		"Bluray-480p":  20,
		"Bluray-576p":  21,
		"BR-DISK":      22,
		"DVD-R":        23,
		"WORKPRINT":    24,
		"CAM":          25,
		"TELESYNC":     26,
		"TELECINE":     27,
		"DVDSCR":       28,
		"REGIONAL":     29,
		"Remux-1080p":  30,
		"Remux-2160p":  31,
	}
	return ids[name]
}

// findCutoffQualityID finds the quality ID for the cutoff tier
func (a *Adapter) findCutoffQualityID(cutoff irv1.VideoQualityTierIR, _ *[]client.QualityProfileQualityItemResource) int {
	if len(cutoff.Sources) == 0 {
		return 0
	}
	res := parseResolution(cutoff.Resolution)
	// Use first source for cutoff
	qualityName := a.buildQualityName(res, cutoff.Sources[0])
	return a.qualityNameToID(qualityName)
}

// markAllowedQualities updates the items to mark allowed qualities
func (a *Adapter) markAllowedQualities(items []client.QualityProfileQualityItemResource, allowed map[int]bool) {
	for i := range items {
		item := &items[i]
		if item.Quality != nil && item.Quality.Id != nil {
			qualityID := int(*item.Quality.Id)
			item.Allowed = boolPtr(allowed[qualityID])
		}
		// Also check nested items (for groups)
		if item.Items != nil && len(*item.Items) > 0 {
			a.markAllowedQualities(*item.Items, allowed)
			// If any nested item is allowed, the group should be allowed
			for _, nested := range *item.Items {
				if nested.Allowed != nil && *nested.Allowed {
					item.Allowed = boolPtr(true)
					break
				}
			}
		}
	}
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
	defer func() { _ = resp.Body.Close() }()

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
	defer func() { _ = resp.Body.Close() }()

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
	defer func() { _ = resp.Body.Close() }()

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
	defer func() { _ = resp.Body.Close() }()

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
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
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
	defer func() { _ = resp.Body.Close() }()

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
	defer func() { _ = resp.Body.Close() }()

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
	defer func() { _ = resp.Body.Close() }()

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
		ConfigContract:           stringPtr(a.normalizeImplementation(ir.Implementation) + "Settings"),
		RemoveCompletedDownloads: boolPtr(ir.RemoveCompletedDownloads),
		RemoveFailedDownloads:    boolPtr(ir.RemoveFailedDownloads),
		Tags:                     &[]int32{int32(tagID)},
	}

	// Set protocol
	switch ir.Protocol {
	case irv1.ProtocolTorrent:
		protocol := client.DownloadProtocolTorrent
		dc.Protocol = &protocol
	case irv1.ProtocolUsenet:
		protocol := client.DownloadProtocolUsenet
		dc.Protocol = &protocol
	}

	// Build fields based on implementation type
	fields := a.buildDownloadClientFields(ir)
	dc.Fields = &fields

	return dc
}

// buildDownloadClientFields creates the Fields array for a download client
func (a *Adapter) buildDownloadClientFields(ir *irv1.DownloadClientIR) []client.Field {
	fields := []client.Field{
		{Name: stringPtr("host"), Value: ir.Host},
		{Name: stringPtr("port"), Value: ir.Port},
		{Name: stringPtr("useSsl"), Value: ir.UseTLS},
	}

	// Add username/password if provided
	if ir.Username != "" {
		fields = append(fields, client.Field{Name: stringPtr("username"), Value: ir.Username})
	}
	if ir.Password != "" {
		fields = append(fields, client.Field{Name: stringPtr("password"), Value: ir.Password})
	}

	// Add implementation-specific fields
	switch ir.Implementation {
	case irv1.ImplementationTransmission:
		fields = append(fields,
			client.Field{Name: stringPtr("urlBase"), Value: "/transmission/"},
			client.Field{Name: stringPtr("movieCategory"), Value: ir.Category},
			client.Field{Name: stringPtr("addPaused"), Value: false},
		)
		if ir.Directory != "" {
			fields = append(fields, client.Field{Name: stringPtr("movieDirectory"), Value: ir.Directory})
		}
	case irv1.ImplementationQBittorrent:
		fields = append(fields,
			client.Field{Name: stringPtr("movieCategory"), Value: ir.Category},
			client.Field{Name: stringPtr("initialState"), Value: 0}, // Start downloading
		)
		if ir.Directory != "" {
			fields = append(fields, client.Field{Name: stringPtr("movieDirectory"), Value: ir.Directory})
		}
	case irv1.ImplementationDeluge:
		fields = append(fields,
			client.Field{Name: stringPtr("movieCategory"), Value: ir.Category},
			client.Field{Name: stringPtr("addPaused"), Value: false},
		)
		if ir.Directory != "" {
			fields = append(fields, client.Field{Name: stringPtr("movieDirectory"), Value: ir.Directory})
		}
	case irv1.ImplementationSABnzbd, irv1.ImplementationNZBGet:
		// Usenet clients use tvCategory for category
		fields = append(fields, client.Field{Name: stringPtr("movieCategory"), Value: ir.Category})
	}

	return fields
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
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
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
	defer func() { _ = resp.Body.Close() }()

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
	defer func() { _ = resp.Body.Close() }()

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
	defer func() { _ = resp.Body.Close() }()

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
	// Determine the implementation and config contract
	impl := ir.Implementation
	if impl == "" {
		if ir.Protocol == irv1.ProtocolTorrent {
			impl = "Torznab"
		} else {
			impl = "Newznab"
		}
	}
	configContract := impl + "Settings"

	idx := client.IndexerResource{
		Name:                    stringPtr(ir.Name),
		Priority:                intPtr(ir.Priority),
		Implementation:          stringPtr(impl),
		ConfigContract:          stringPtr(configContract),
		EnableRss:               boolPtr(ir.Enable && ir.EnableRss),
		EnableAutomaticSearch:   boolPtr(ir.Enable && ir.EnableAutomaticSearch),
		EnableInteractiveSearch: boolPtr(ir.Enable && ir.EnableInteractiveSearch),
		Tags:                    &[]int32{int32(tagID)},
	}

	// Set protocol
	switch ir.Protocol {
	case irv1.ProtocolTorrent:
		protocol := client.DownloadProtocolTorrent
		idx.Protocol = &protocol
	case irv1.ProtocolUsenet:
		protocol := client.DownloadProtocolUsenet
		idx.Protocol = &protocol
	}

	// Build fields
	fields := a.buildIndexerFields(ir)
	idx.Fields = &fields

	return idx
}

// buildIndexerFields creates the Fields array for an indexer
func (a *Adapter) buildIndexerFields(ir *irv1.IndexerIR) []client.Field {
	fields := []client.Field{
		{Name: stringPtr("baseUrl"), Value: ir.URL},
		{Name: stringPtr("apiPath"), Value: "/api"},
	}

	if ir.APIKey != "" {
		fields = append(fields, client.Field{Name: stringPtr("apiKey"), Value: ir.APIKey})
	}

	// Add categories if specified
	if len(ir.Categories) > 0 {
		fields = append(fields, client.Field{Name: stringPtr("categories"), Value: ir.Categories})
	}

	// Add torrent-specific fields
	if ir.Protocol == irv1.ProtocolTorrent {
		if ir.MinimumSeeders > 0 {
			fields = append(fields, client.Field{Name: stringPtr("minimumSeeders"), Value: ir.MinimumSeeders})
		}
		if ir.SeedRatio > 0 {
			fields = append(fields, client.Field{Name: stringPtr("seedCriteria.seedRatio"), Value: ir.SeedRatio})
		}
		if ir.SeedTimeMinutes > 0 {
			fields = append(fields, client.Field{Name: stringPtr("seedCriteria.seedTime"), Value: ir.SeedTimeMinutes})
		}
	}

	return fields
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
	defer func() { _ = resp.Body.Close() }()

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
	defer func() { _ = resp.Body.Close() }()

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
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}
