// Package prowlarr provides the Prowlarr adapter for Nebularr.
// It implements the adapters.Adapter interface for managing Prowlarr configuration.
// Note: Prowlarr uses API v1, not v3.
// Prowlarr is different from other *arr apps - it manages indexers natively
// and syncs them to downstream applications.
package prowlarr

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

// Adapter implements the adapters.Adapter interface for Prowlarr
type Adapter struct{}

// Ensure Adapter implements the interface
var _ adapters.Adapter = (*Adapter)(nil)

// Name returns a unique identifier for this adapter
func (a *Adapter) Name() string {
	return "prowlarr"
}

// SupportedApp returns the app this adapter handles
func (a *Adapter) SupportedApp() string {
	return adapters.AppProwlarr
}

// Connect tests connectivity and retrieves service info
func (a *Adapter) Connect(ctx context.Context, conn *irv1.ConnectionIR) (*adapters.ServiceInfo, error) {
	c := a.newClient(conn)

	var status SystemResource
	if err := c.get(ctx, "/api/v1/system/status", &status); err != nil {
		return nil, fmt.Errorf("failed to connect to Prowlarr: %w", err)
	}

	info := &adapters.ServiceInfo{
		Version: status.Version,
	}

	if status.StartTime != nil {
		info.StartTime = *status.StartTime
	}

	return info, nil
}

// Discover queries Prowlarr for its capabilities
func (a *Adapter) Discover(ctx context.Context, conn *irv1.ConnectionIR) (*adapters.Capabilities, error) {
	c := a.newClient(conn)

	caps := &adapters.Capabilities{
		DiscoveredAt: time.Now(),
	}

	// Discover download client types
	var dcSchemas []DownloadClientResource
	if err := c.get(ctx, "/api/v1/downloadclient/schema", &dcSchemas); err == nil {
		seen := make(map[string]bool)
		for _, schema := range dcSchemas {
			if schema.Implementation != "" && !seen[schema.Implementation] {
				caps.DownloadClientTypes = append(caps.DownloadClientTypes, schema.Implementation)
				seen[schema.Implementation] = true
			}
		}
	}

	// Discover indexer definitions (Prowlarr has many more indexer types)
	var idxSchemas []IndexerResource
	if err := c.get(ctx, "/api/v1/indexer/schema", &idxSchemas); err == nil {
		seen := make(map[string]bool)
		for _, schema := range idxSchemas {
			if schema.DefinitionName != "" && !seen[schema.DefinitionName] {
				caps.IndexerTypes = append(caps.IndexerTypes, schema.DefinitionName)
				seen[schema.DefinitionName] = true
			}
		}
	}

	return caps, nil
}

// CurrentState retrieves the current managed state from Prowlarr
func (a *Adapter) CurrentState(ctx context.Context, conn *irv1.ConnectionIR) (*irv1.IR, error) {
	c := a.newClient(conn)

	ir := &irv1.IR{
		Version:     irv1.IRVersion,
		GeneratedAt: time.Now(),
		App:         adapters.AppProwlarr,
		Connection:  conn,
		Prowlarr: &irv1.ProwlarrIR{
			Connection: conn,
		},
	}

	// Get ownership tag ID
	tagID, err := a.getOwnershipTagID(ctx, c)
	if err != nil {
		// No ownership tag means no managed resources
		return ir, nil
	}

	// Get managed indexers
	if indexers, err := a.getManagedIndexers(ctx, c, tagID); err == nil {
		ir.Prowlarr.Indexers = indexers
	}

	// Get managed proxies
	if proxies, err := a.getManagedProxies(ctx, c, tagID); err == nil {
		ir.Prowlarr.Proxies = proxies
	}

	// Get managed applications
	if apps, err := a.getManagedApplications(ctx, c, tagID); err == nil {
		ir.Prowlarr.Applications = apps
	}

	// Get managed download clients
	if clients, err := a.getManagedDownloadClients(ctx, c, tagID); err == nil {
		ir.Prowlarr.DownloadClients = clients
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

	// Ensure both have Prowlarr IR
	if desired.Prowlarr == nil {
		return changes, nil
	}

	currentProwlarr := current.Prowlarr
	if currentProwlarr == nil {
		currentProwlarr = &irv1.ProwlarrIR{}
	}

	// Diff indexers
	if err := a.diffIndexers(currentProwlarr, desired.Prowlarr, changes); err != nil {
		return nil, fmt.Errorf("failed to diff indexers: %w", err)
	}

	// Diff proxies
	if err := a.diffProxies(currentProwlarr, desired.Prowlarr, changes); err != nil {
		return nil, fmt.Errorf("failed to diff proxies: %w", err)
	}

	// Diff applications
	if err := a.diffApplications(currentProwlarr, desired.Prowlarr, changes); err != nil {
		return nil, fmt.Errorf("failed to diff applications: %w", err)
	}

	// Diff download clients
	if err := a.diffDownloadClients(currentProwlarr, desired.Prowlarr, changes); err != nil {
		return nil, fmt.Errorf("failed to diff download clients: %w", err)
	}

	return changes, nil
}

// Apply executes the changes against Prowlarr
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

// httpClient is a simple HTTP client for Prowlarr API
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
	defer func() { _ = resp.Body.Close() }()

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
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	if result != nil {
		return json.NewDecoder(resp.Body).Decode(result)
	}
	return nil
}

func (c *httpClient) put(ctx context.Context, path string, body, result interface{}) error { //nolint:unparam // result may be used in future
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
	defer func() { _ = resp.Body.Close() }()

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
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
