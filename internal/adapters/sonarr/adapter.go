// Package sonarr provides the Sonarr adapter for Nebularr.
// It implements the adapters.Adapter interface for managing Sonarr configuration.
package sonarr

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/poiley/nebularr-operator/internal/adapters"
	irv1 "github.com/poiley/nebularr-operator/internal/ir/v1"
)

const (
	// OwnershipTagName is the tag name used to identify Nebularr-managed resources
	OwnershipTagName = "nebularr-managed"
)

// Adapter implements the adapters.Adapter interface for Sonarr
type Adapter struct{}

// Ensure Adapter implements the interface
var _ adapters.Adapter = (*Adapter)(nil)

// Name returns a unique identifier for this adapter
func (a *Adapter) Name() string {
	return "sonarr"
}

// SupportedApp returns the app this adapter handles
func (a *Adapter) SupportedApp() string {
	return adapters.AppSonarr
}

// Connect tests connectivity and retrieves service info
func (a *Adapter) Connect(ctx context.Context, conn *irv1.ConnectionIR) (*adapters.ServiceInfo, error) {
	c := a.newClient(conn)

	var status SystemResource
	if err := c.get(ctx, "/api/v3/system/status", &status); err != nil {
		return nil, fmt.Errorf("failed to connect to Sonarr: %w", err)
	}

	info := &adapters.ServiceInfo{
		Version: status.Version,
	}

	if status.StartTime != nil {
		info.StartTime = *status.StartTime
	}

	return info, nil
}

// Discover queries Sonarr for its capabilities
func (a *Adapter) Discover(ctx context.Context, conn *irv1.ConnectionIR) (*adapters.Capabilities, error) {
	c := a.newClient(conn)

	caps := &adapters.Capabilities{
		DiscoveredAt: time.Now(),
		// Sonarr supports these resolutions
		Resolutions: []string{"2160p", "1080p", "720p", "480p"},
		// Sonarr supports these sources
		Sources: []string{"bluray", "webdl", "webrip", "hdtv", "dvd"},
	}

	// Discover download client types
	var dcSchemas []DownloadClientResource
	if err := c.get(ctx, "/api/v3/downloadclient/schema", &dcSchemas); err == nil {
		seen := make(map[string]bool)
		for _, schema := range dcSchemas {
			if schema.Implementation != "" && !seen[schema.Implementation] {
				caps.DownloadClientTypes = append(caps.DownloadClientTypes, schema.Implementation)
				seen[schema.Implementation] = true
			}
		}
	}

	// Discover indexer types
	var idxSchemas []IndexerResource
	if err := c.get(ctx, "/api/v3/indexer/schema", &idxSchemas); err == nil {
		seen := make(map[string]bool)
		for _, schema := range idxSchemas {
			if schema.Implementation != "" && !seen[schema.Implementation] {
				caps.IndexerTypes = append(caps.IndexerTypes, schema.Implementation)
				seen[schema.Implementation] = true
			}
		}
	}

	return caps, nil
}

// CurrentState retrieves the current managed state from Sonarr
func (a *Adapter) CurrentState(ctx context.Context, conn *irv1.ConnectionIR) (*irv1.IR, error) {
	c := a.newClient(conn)

	ir := &irv1.IR{
		Version:     irv1.IRVersion,
		GeneratedAt: time.Now(),
		App:         adapters.AppSonarr,
		Connection:  conn,
	}

	// Get ownership tag ID
	tagID, err := a.getOwnershipTagID(ctx, c)
	if err != nil {
		// No ownership tag means no managed resources
		return ir, nil
	}

	// Get quality profiles
	if profiles, err := a.getManagedQualityProfiles(ctx, c, tagID); err == nil && len(profiles) > 0 {
		ir.Quality = &irv1.QualityIR{
			Video: profiles[0],
		}
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

	// Get root folders
	if folders, err := a.getRootFolders(ctx, c); err == nil {
		ir.RootFolders = folders
	}

	// Get naming config
	if naming, err := a.getNamingConfig(ctx, c); err == nil {
		ir.Naming = &irv1.NamingIR{
			Sonarr: naming,
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

// Apply executes the changes against Sonarr
func (a *Adapter) Apply(ctx context.Context, conn *irv1.ConnectionIR, changes *adapters.ChangeSet) (*adapters.ApplyResult, error) {
	c := a.newClient(conn)

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
	c := a.newClient(conn)

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

// httpClient is a simple HTTP client for Sonarr API
type httpClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

func (a *Adapter) newClient(conn *irv1.ConnectionIR) *httpClient {
	return &httpClient{
		baseURL: conn.URL,
		apiKey:  conn.APIKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *httpClient) get(ctx context.Context, path string, result interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-Api-Key", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	return json.NewDecoder(resp.Body).Decode(result)
}

func (c *httpClient) post(ctx context.Context, path string, body, result interface{}) error {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return err
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bodyReader)
	if err != nil {
		return err
	}
	req.Header.Set("X-Api-Key", c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	if result != nil {
		return json.NewDecoder(resp.Body).Decode(result)
	}
	return nil
}

func (c *httpClient) put(ctx context.Context, path string, body, result interface{}) error {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return err
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, c.baseURL+path, bodyReader)
	if err != nil {
		return err
	}
	req.Header.Set("X-Api-Key", c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	if result != nil {
		return json.NewDecoder(resp.Body).Decode(result)
	}
	return nil
}

func (c *httpClient) delete(ctx context.Context, path string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.baseURL+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-Api-Key", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
