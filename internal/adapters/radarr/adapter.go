// Package radarr provides the Radarr adapter for Nebularr.
// It implements the adapters.Adapter interface for managing Radarr configuration.
package radarr

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/poiley/nebularr-operator/internal/adapters"
	"github.com/poiley/nebularr-operator/internal/adapters/radarr/client"
	irv1 "github.com/poiley/nebularr-operator/internal/ir/v1"
)

const (
	// OwnershipTagName is the tag name used to identify Nebularr-managed resources
	OwnershipTagName = "nebularr-managed"
)

// Adapter implements the adapters.Adapter interface for Radarr
type Adapter struct{}

// Ensure Adapter implements the interface
var _ adapters.Adapter = (*Adapter)(nil)

// Name returns a unique identifier for this adapter
func (a *Adapter) Name() string {
	return "radarr"
}

// SupportedApp returns the app this adapter handles
func (a *Adapter) SupportedApp() string {
	return adapters.AppRadarr
}

// Connect tests connectivity and retrieves service info
func (a *Adapter) Connect(ctx context.Context, conn *irv1.ConnectionIR) (*adapters.ServiceInfo, error) {
	c, err := a.newClient(conn)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	resp, err := c.GetApiV3SystemStatus(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Radarr: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var status client.SystemResource
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, fmt.Errorf("failed to decode system status: %w", err)
	}

	info := &adapters.ServiceInfo{
		Version: ptrToString(status.Version),
	}

	if status.StartTime != nil {
		info.StartTime = *status.StartTime
	}

	return info, nil
}

// Discover queries Radarr for its capabilities
func (a *Adapter) Discover(ctx context.Context, conn *irv1.ConnectionIR) (*adapters.Capabilities, error) {
	c, err := a.newClient(conn)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	caps := &adapters.Capabilities{
		DiscoveredAt: time.Now(),
		// Radarr supports these resolutions
		Resolutions: []string{"2160p", "1080p", "720p", "480p"},
		// Radarr supports these sources
		Sources: []string{"bluray", "webdl", "webrip", "hdtv", "dvd", "cam", "telesync", "telecine", "workprint"},
	}

	// Discover custom format specs
	resp, err := c.GetApiV3CustomformatSchema(ctx)
	if err == nil && resp.StatusCode == http.StatusOK {
		defer resp.Body.Close()
		var schemas []client.CustomFormatSpecificationSchema
		if err := json.NewDecoder(resp.Body).Decode(&schemas); err == nil {
			for _, schema := range schemas {
				caps.CustomFormatSpecs = append(caps.CustomFormatSpecs, adapters.CustomFormatSpec{
					Name:           ptrToString(schema.Name),
					Implementation: ptrToString(schema.Implementation),
				})
			}
		}
	}

	// Discover download client types
	resp, err = c.GetApiV3DownloadclientSchema(ctx)
	if err == nil && resp.StatusCode == http.StatusOK {
		defer resp.Body.Close()
		var schemas []client.DownloadClientResource
		if err := json.NewDecoder(resp.Body).Decode(&schemas); err == nil {
			seen := make(map[string]bool)
			for _, schema := range schemas {
				impl := ptrToString(schema.Implementation)
				if impl != "" && !seen[impl] {
					caps.DownloadClientTypes = append(caps.DownloadClientTypes, impl)
					seen[impl] = true
				}
			}
		}
	}

	// Discover indexer types
	resp, err = c.GetApiV3IndexerSchema(ctx)
	if err == nil && resp.StatusCode == http.StatusOK {
		defer resp.Body.Close()
		var schemas []client.IndexerResource
		if err := json.NewDecoder(resp.Body).Decode(&schemas); err == nil {
			seen := make(map[string]bool)
			for _, schema := range schemas {
				impl := ptrToString(schema.Implementation)
				if impl != "" && !seen[impl] {
					caps.IndexerTypes = append(caps.IndexerTypes, impl)
					seen[impl] = true
				}
			}
		}
	}

	return caps, nil
}

// CurrentState retrieves the current managed state from Radarr
func (a *Adapter) CurrentState(ctx context.Context, conn *irv1.ConnectionIR) (*irv1.IR, error) {
	c, err := a.newClient(conn)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	ir := &irv1.IR{
		Version:     irv1.IRVersion,
		GeneratedAt: time.Now(),
		App:         adapters.AppRadarr,
		Connection:  conn,
	}

	// Get ownership tag ID
	tagID, err := a.getOwnershipTagID(ctx, c)
	if err != nil {
		// No ownership tag means no managed resources
		return ir, nil
	}

	// Get quality profiles tagged with ownership tag
	if profiles, err := a.getManagedQualityProfiles(ctx, c, tagID); err == nil && len(profiles) > 0 {
		ir.Quality = &irv1.QualityIR{
			Video: profiles[0], // For now, we only manage one profile per config
		}
	}

	// Get custom formats (all of them, since they don't have tags)
	// This is needed to prevent "Must be unique" errors on re-reconcile
	if customFormats, err := a.getManagedCustomFormats(ctx, c); err == nil && len(customFormats) > 0 {
		if ir.Quality == nil {
			ir.Quality = &irv1.QualityIR{
				Video: &irv1.VideoQualityIR{},
			}
		}
		if ir.Quality.Video == nil {
			ir.Quality.Video = &irv1.VideoQualityIR{}
		}
		ir.Quality.Video.CustomFormats = customFormats
	}

	// Get download clients tagged with ownership tag
	if clients, err := a.getManagedDownloadClients(ctx, c, tagID); err == nil {
		ir.DownloadClients = clients
	}

	// Get indexers tagged with ownership tag
	if indexers, err := a.getManagedIndexers(ctx, c, tagID); err == nil && len(indexers) > 0 {
		ir.Indexers = &irv1.IndexersIR{
			Direct: indexers,
		}
	}

	// Get root folders (not tagged, but we track all of them)
	if folders, err := a.getRootFolders(ctx, c); err == nil {
		ir.RootFolders = folders
	}

	// Get naming config
	if naming, err := a.getNamingConfig(ctx, c); err == nil {
		ir.Naming = &irv1.NamingIR{
			Radarr: naming,
		}
	}

	// Get import lists tagged with ownership tag
	if importLists, err := a.getManagedImportLists(ctx, c, tagID); err == nil {
		ir.ImportLists = importLists
	}

	// Get media management config
	if mediaManagement, err := a.getMediaManagementIR(ctx, c); err == nil {
		ir.MediaManagement = mediaManagement
	}

	// Get authentication config
	if auth, err := a.getAuthenticationIR(ctx, c); err == nil {
		ir.Authentication = auth
	}

	return ir, nil
}

// Diff computes the changes needed to move from current to desired state
func (a *Adapter) Diff(current, desired *irv1.IR, caps *adapters.Capabilities) (*adapters.ChangeSet, error) {
	changes := &adapters.ChangeSet{
		Creates: []adapters.Change{},
		Updates: []adapters.Change{},
		Deletes: []adapters.Change{},
	}

	// Diff quality profiles
	if err := a.diffQualityProfiles(current, desired, changes); err != nil {
		return nil, fmt.Errorf("failed to diff quality profiles: %w", err)
	}

	// Diff custom formats
	if err := a.diffCustomFormats(current, desired, changes); err != nil {
		return nil, fmt.Errorf("failed to diff custom formats: %w", err)
	}

	// Diff download clients
	if err := a.diffDownloadClients(current, desired, changes); err != nil {
		return nil, fmt.Errorf("failed to diff download clients: %w", err)
	}

	// Diff indexers
	if err := a.diffIndexers(current, desired, changes); err != nil {
		return nil, fmt.Errorf("failed to diff indexers: %w", err)
	}

	// Diff root folders
	if err := a.diffRootFolders(current, desired, changes); err != nil {
		return nil, fmt.Errorf("failed to diff root folders: %w", err)
	}

	// Diff naming config
	if err := a.diffNaming(current, desired, changes); err != nil {
		return nil, fmt.Errorf("failed to diff naming: %w", err)
	}

	return changes, nil
}

// Apply executes the changes against Radarr
func (a *Adapter) Apply(ctx context.Context, conn *irv1.ConnectionIR, changes *adapters.ChangeSet) (*adapters.ApplyResult, error) {
	c, err := a.newClient(conn)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	result := &adapters.ApplyResult{}

	// Ensure ownership tag exists
	tagID, err := a.ensureOwnershipTag(ctx, c)
	if err != nil {
		return nil, fmt.Errorf("failed to ensure ownership tag: %w", err)
	}

	// Apply creates
	for _, change := range changes.Creates {
		if err := a.applyCreate(ctx, c, change, tagID); err != nil {
			result.Failed++
			result.Errors = append(result.Errors, adapters.ApplyError{
				Change: change,
				Error:  err,
			})
		} else {
			result.Applied++
		}
	}

	// Apply updates
	for _, change := range changes.Updates {
		if err := a.applyUpdate(ctx, c, change, tagID); err != nil {
			result.Failed++
			result.Errors = append(result.Errors, adapters.ApplyError{
				Change: change,
				Error:  err,
			})
		} else {
			result.Applied++
		}
	}

	// Apply deletes
	for _, change := range changes.Deletes {
		if err := a.applyDelete(ctx, c, change); err != nil {
			result.Failed++
			result.Errors = append(result.Errors, adapters.ApplyError{
				Change: change,
				Error:  err,
			})
		} else {
			result.Applied++
		}
	}

	return result, nil
}

// ApplyDirect applies configuration directly from IR (not via ChangeSet)
// This is used for resources like import lists, media management, and authentication
// that use a different sync pattern (direct apply rather than diff-based)
func (a *Adapter) ApplyDirect(ctx context.Context, conn *irv1.ConnectionIR, ir *irv1.IR) (*adapters.ApplyResult, error) {
	c, err := a.newClient(conn)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	result := &adapters.ApplyResult{}

	// Ensure ownership tag exists
	tagID, err := a.ensureOwnershipTag(ctx, c)
	if err != nil {
		return nil, fmt.Errorf("failed to ensure ownership tag: %w", err)
	}

	// Apply import lists directly (they handle their own diff internally)
	if len(ir.ImportLists) > 0 {
		stats, err := a.applyImportLists(ctx, c, ir, tagID)
		if err != nil {
			return nil, fmt.Errorf("failed to apply import lists: %w", err)
		}

		result.Applied += stats.Created + stats.Updated + stats.Deleted
		result.Skipped += stats.Skipped
		for _, e := range stats.Errors {
			result.Failed++
			result.Errors = append(result.Errors, adapters.ApplyError{
				Change: adapters.Change{ResourceType: adapters.ResourceImportList},
				Error:  e,
			})
		}
	}

	// Apply media management configuration
	if ir.MediaManagement != nil {
		if err := a.applyMediaManagement(ctx, c, ir.MediaManagement); err != nil {
			result.Failed++
			result.Errors = append(result.Errors, adapters.ApplyError{
				Change: adapters.Change{ResourceType: adapters.ResourceMediaManagement},
				Error:  err,
			})
		} else {
			result.Applied++
		}
	}

	// Apply authentication configuration
	if ir.Authentication != nil {
		if err := a.applyAuthentication(ctx, c, ir.Authentication); err != nil {
			result.Failed++
			result.Errors = append(result.Errors, adapters.ApplyError{
				Change: adapters.Change{ResourceType: adapters.ResourceAuthentication},
				Error:  err,
			})
		} else {
			result.Applied++
		}
	}

	return result, nil
}

// newClient creates a new Radarr API client
func (a *Adapter) newClient(conn *irv1.ConnectionIR) (*client.Client, error) {
	// Create HTTP client with TLS config if needed
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	if conn.InsecureSkipVerify {
		httpClient.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec // User explicitly requested insecure
		}
	}

	// Create the oapi-codegen client
	c, err := client.NewClient(conn.URL, client.WithHTTPClient(httpClient), client.WithRequestEditorFn(func(ctx context.Context, req *http.Request) error {
		req.Header.Set("X-Api-Key", conn.APIKey)
		return nil
	}))
	if err != nil {
		return nil, err
	}

	return c, nil
}

// Helper functions

func ptrToString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func ptrToInt(i *int32) int {
	if i == nil {
		return 0
	}
	return int(*i)
}

func intPtr(i int) *int32 {
	v := int32(i)
	return &v
}

func stringPtr(s string) *string {
	return &s
}

func boolPtr(b bool) *bool {
	return &b
}
