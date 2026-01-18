// Package readarr provides the Readarr adapter for Nebularr.
// It implements the adapters.Adapter interface for managing Readarr configuration.
// Readarr is for ebooks and audiobooks, using API v1.
package readarr

import (
	"bytes"
	"context"
	"crypto/tls"
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

// Adapter implements the adapters.Adapter interface for Readarr
type Adapter struct{}

// Ensure Adapter implements the interface
var _ adapters.Adapter = (*Adapter)(nil)

// Name returns a unique identifier for this adapter
func (a *Adapter) Name() string {
	return "readarr"
}

// SupportedApp returns the app this adapter handles
func (a *Adapter) SupportedApp() string {
	return adapters.AppReadarr
}

// Connect tests connectivity and retrieves service info
func (a *Adapter) Connect(ctx context.Context, conn *irv1.ConnectionIR) (*adapters.ServiceInfo, error) {
	c := a.newClient(conn)

	var status SystemResource
	if err := c.get(ctx, "/api/v1/system/status", &status); err != nil {
		return nil, fmt.Errorf("failed to connect to Readarr: %w", err)
	}

	info := &adapters.ServiceInfo{
		Version: status.Version,
	}

	if status.StartTime != nil {
		info.StartTime = *status.StartTime
	}

	return info, nil
}

// Discover queries Readarr for its capabilities
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

	// Discover indexer types
	var idxSchemas []IndexerResource
	if err := c.get(ctx, "/api/v1/indexer/schema", &idxSchemas); err == nil {
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

// CurrentState retrieves the current managed state from Readarr
func (a *Adapter) CurrentState(ctx context.Context, conn *irv1.ConnectionIR) (*irv1.IR, error) {
	c := a.newClient(conn)

	ir := &irv1.IR{
		Version:     irv1.IRVersion,
		GeneratedAt: time.Now(),
		App:         adapters.AppReadarr,
		Connection:  conn,
	}

	// Get ownership tag ID
	tagID, err := a.getOwnershipTagID(ctx, c)
	if err != nil {
		// No ownership tag means no managed resources
		return ir, nil
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

	return ir, nil
}

// Diff computes the changes needed to move from current to desired state
func (a *Adapter) Diff(current, desired *irv1.IR, caps *adapters.Capabilities) (*adapters.ChangeSet, error) {
	changes := &adapters.ChangeSet{
		Creates: []adapters.Change{},
		Updates: []adapters.Change{},
		Deletes: []adapters.Change{},
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

	return changes, nil
}

// Apply executes the changes against Readarr
func (a *Adapter) Apply(ctx context.Context, conn *irv1.ConnectionIR, changes *adapters.ChangeSet) (*adapters.ApplyResult, error) {
	c := a.newClient(conn)
	result := &adapters.ApplyResult{}

	// Ensure ownership tag exists
	tagID, err := a.ensureOwnershipTag(ctx, c)
	if err != nil {
		return result, fmt.Errorf("failed to ensure ownership tag: %w", err)
	}

	// Process creates
	for _, change := range changes.Creates {
		if err := a.applyCreate(ctx, c, change, tagID); err != nil {
			result.Failed++
			result.Errors = append(result.Errors, adapters.ApplyError{Change: change, Error: err})
		} else {
			result.Applied++
		}
	}

	// Process updates
	for _, change := range changes.Updates {
		if err := a.applyUpdate(ctx, c, change, tagID); err != nil {
			result.Failed++
			result.Errors = append(result.Errors, adapters.ApplyError{Change: change, Error: err})
		} else {
			result.Applied++
		}
	}

	// Process deletes
	for _, change := range changes.Deletes {
		if err := a.applyDelete(ctx, c, change); err != nil {
			result.Failed++
			result.Errors = append(result.Errors, adapters.ApplyError{Change: change, Error: err})
		} else {
			result.Applied++
		}
	}

	return result, nil
}

// diffDownloadClients computes changes for download clients
func (a *Adapter) diffDownloadClients(current, desired *irv1.IR, changes *adapters.ChangeSet) error {
	currentMap := make(map[string]irv1.DownloadClientIR)
	for _, dc := range current.DownloadClients {
		currentMap[dc.Name] = dc
	}

	desiredMap := make(map[string]irv1.DownloadClientIR)
	for _, dc := range desired.DownloadClients {
		desiredMap[dc.Name] = dc
	}

	// Creates and updates
	for name, desiredDC := range desiredMap {
		if _, exists := currentMap[name]; !exists {
			changes.Creates = append(changes.Creates, adapters.Change{
				ResourceType: adapters.ResourceDownloadClient,
				Name:         name,
				Payload:      desiredDC,
			})
		}
		// Note: For simplicity, we're not doing deep updates here
		// A production implementation would compare fields
	}

	// Deletes (only managed resources)
	for name := range currentMap {
		if _, exists := desiredMap[name]; !exists {
			changes.Deletes = append(changes.Deletes, adapters.Change{
				ResourceType: adapters.ResourceDownloadClient,
				Name:         name,
			})
		}
	}

	return nil
}

// diffIndexers computes changes for indexers
func (a *Adapter) diffIndexers(current, desired *irv1.IR, changes *adapters.ChangeSet) error {
	currentIndexers := make(map[string]irv1.IndexerIR)
	if current.Indexers != nil {
		for _, idx := range current.Indexers.Direct {
			currentIndexers[idx.Name] = idx
		}
	}

	desiredIndexers := make(map[string]irv1.IndexerIR)
	if desired.Indexers != nil {
		for _, idx := range desired.Indexers.Direct {
			desiredIndexers[idx.Name] = idx
		}
	}

	// Creates
	for name, desiredIdx := range desiredIndexers {
		if _, exists := currentIndexers[name]; !exists {
			changes.Creates = append(changes.Creates, adapters.Change{
				ResourceType: adapters.ResourceIndexer,
				Name:         name,
				Payload:      desiredIdx,
			})
		}
	}

	// Deletes
	for name := range currentIndexers {
		if _, exists := desiredIndexers[name]; !exists {
			changes.Deletes = append(changes.Deletes, adapters.Change{
				ResourceType: adapters.ResourceIndexer,
				Name:         name,
			})
		}
	}

	return nil
}

// diffRootFolders computes changes for root folders
func (a *Adapter) diffRootFolders(current, desired *irv1.IR, changes *adapters.ChangeSet) error {
	currentPaths := make(map[string]bool)
	for _, rf := range current.RootFolders {
		currentPaths[rf.Path] = true
	}

	// Creates only - we don't delete root folders automatically
	for _, rf := range desired.RootFolders {
		if !currentPaths[rf.Path] {
			changes.Creates = append(changes.Creates, adapters.Change{
				ResourceType: adapters.ResourceRootFolder,
				Name:         rf.Path,
				Payload:      rf,
			})
		}
	}

	return nil
}

// applyCreate handles creation of a resource
func (a *Adapter) applyCreate(ctx context.Context, c *httpClient, change adapters.Change, tagID int) error {
	switch change.ResourceType {
	case adapters.ResourceDownloadClient:
		return a.createDownloadClient(ctx, c, change.Payload.(irv1.DownloadClientIR), tagID)
	case adapters.ResourceIndexer:
		return a.createIndexer(ctx, c, change.Payload.(irv1.IndexerIR), tagID)
	case adapters.ResourceRootFolder:
		return a.createRootFolder(ctx, c, change.Payload.(irv1.RootFolderIR))
	default:
		return fmt.Errorf("unknown resource type: %s", change.ResourceType)
	}
}

// applyUpdate handles updating a resource
func (a *Adapter) applyUpdate(ctx context.Context, c *httpClient, change adapters.Change, tagID int) error {
	// For now, we just log that updates are not fully implemented
	return nil
}

// applyDelete handles deletion of a resource
func (a *Adapter) applyDelete(ctx context.Context, c *httpClient, change adapters.Change) error {
	switch change.ResourceType {
	case adapters.ResourceDownloadClient:
		return a.deleteDownloadClientByName(ctx, c, change.Name)
	case adapters.ResourceIndexer:
		return a.deleteIndexerByName(ctx, c, change.Name)
	default:
		return fmt.Errorf("unknown resource type: %s", change.ResourceType)
	}
}

// createDownloadClient creates a new download client
func (a *Adapter) createDownloadClient(ctx context.Context, c *httpClient, dc irv1.DownloadClientIR, tagID int) error {
	// Build the resource from IR
	resource := DownloadClientResource{
		Name:           dc.Name,
		Implementation: dc.Implementation,
		Protocol:       dc.Protocol,
		Enable:         dc.Enable,
		Priority:       dc.Priority,
		Tags:           []int{tagID},
		Fields: []FieldResource{
			{Name: "host", Value: dc.Host},
			{Name: "port", Value: dc.Port},
			{Name: "useSsl", Value: dc.UseTLS},
			{Name: "username", Value: dc.Username},
			{Name: "password", Value: dc.Password},
		},
	}

	if dc.Category != "" {
		resource.Fields = append(resource.Fields, FieldResource{Name: "bookCategory", Value: dc.Category})
	}

	var result DownloadClientResource
	return c.post(ctx, "/api/v1/downloadclient", resource, &result)
}

// createIndexer creates a new indexer
func (a *Adapter) createIndexer(ctx context.Context, c *httpClient, idx irv1.IndexerIR, tagID int) error {
	resource := IndexerResource{
		Name:           idx.Name,
		Implementation: idx.Implementation,
		Protocol:       idx.Protocol,
		Enable:         idx.Enable,
		Priority:       idx.Priority,
		Tags:           []int{tagID},
		Fields: []FieldResource{
			{Name: "baseUrl", Value: idx.URL},
			{Name: "apiKey", Value: idx.APIKey},
			{Name: "minimumSeeders", Value: idx.MinimumSeeders},
			{Name: "enableRss", Value: idx.EnableRss},
			{Name: "enableAutomaticSearch", Value: idx.EnableAutomaticSearch},
			{Name: "enableInteractiveSearch", Value: idx.EnableInteractiveSearch},
		},
	}

	var result IndexerResource
	return c.post(ctx, "/api/v1/indexer", resource, &result)
}

// createRootFolder creates a new root folder
func (a *Adapter) createRootFolder(ctx context.Context, c *httpClient, rf irv1.RootFolderIR) error {
	resource := RootFolderResource{
		Path: rf.Path,
		Name: rf.Name,
	}

	var result RootFolderResource
	return c.post(ctx, "/api/v1/rootfolder", resource, &result)
}

// deleteDownloadClientByName finds and deletes a download client by name
func (a *Adapter) deleteDownloadClientByName(ctx context.Context, c *httpClient, name string) error {
	var clients []DownloadClientResource
	if err := c.get(ctx, "/api/v1/downloadclient", &clients); err != nil {
		return err
	}

	for _, client := range clients {
		if client.Name == name {
			return c.delete(ctx, fmt.Sprintf("/api/v1/downloadclient/%d", client.ID))
		}
	}

	return nil // Not found is not an error
}

// deleteIndexerByName finds and deletes an indexer by name
func (a *Adapter) deleteIndexerByName(ctx context.Context, c *httpClient, name string) error {
	var indexers []IndexerResource
	if err := c.get(ctx, "/api/v1/indexer", &indexers); err != nil {
		return err
	}

	for _, indexer := range indexers {
		if indexer.Name == name {
			return c.delete(ctx, fmt.Sprintf("/api/v1/indexer/%d", indexer.ID))
		}
	}

	return nil // Not found is not an error
}

// httpClient is a simple HTTP client for Readarr API
type httpClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

func (a *Adapter) newClient(conn *irv1.ConnectionIR) *httpClient {
	hc := &http.Client{
		Timeout: 30 * time.Second,
	}

	if conn.InsecureSkipVerify {
		hc.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec // User explicitly requested insecure
		}
	}

	return &httpClient{
		baseURL:    conn.URL,
		apiKey:     conn.APIKey,
		httpClient: hc,
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

// HealthResource represents a health check from Readarr API
type HealthResource struct {
	Source  string `json:"source"`
	Type    string `json:"type"` // error, warning, notice
	Message string `json:"message"`
	WikiURL string `json:"wikiUrl"`
}

// getOwnershipTagID finds or returns -1 if the ownership tag doesn't exist
func (a *Adapter) getOwnershipTagID(ctx context.Context, c *httpClient) (int, error) {
	var tags []TagResource
	if err := c.get(ctx, "/api/v1/tag", &tags); err != nil {
		return -1, err
	}

	for _, tag := range tags {
		if tag.Label == OwnershipTagName {
			return tag.ID, nil
		}
	}

	return -1, fmt.Errorf("ownership tag not found")
}

// ensureOwnershipTag creates the ownership tag if it doesn't exist and returns its ID
func (a *Adapter) ensureOwnershipTag(ctx context.Context, c *httpClient) (int, error) {
	tagID, err := a.getOwnershipTagID(ctx, c)
	if err == nil {
		return tagID, nil
	}

	// Create the tag
	newTag := TagResource{Label: OwnershipTagName}
	var result TagResource
	if err := c.post(ctx, "/api/v1/tag", newTag, &result); err != nil {
		return -1, fmt.Errorf("failed to create ownership tag: %w", err)
	}

	return result.ID, nil
}

// getManagedDownloadClients retrieves download clients tagged with the ownership tag
func (a *Adapter) getManagedDownloadClients(ctx context.Context, c *httpClient, tagID int) ([]irv1.DownloadClientIR, error) {
	var clients []DownloadClientResource
	if err := c.get(ctx, "/api/v1/downloadclient", &clients); err != nil {
		return nil, err
	}

	var managed []irv1.DownloadClientIR
	for _, client := range clients {
		if containsTag(client.Tags, tagID) {
			managed = append(managed, a.downloadClientToIR(&client))
		}
	}

	return managed, nil
}

// getManagedIndexers retrieves indexers tagged with the ownership tag
func (a *Adapter) getManagedIndexers(ctx context.Context, c *httpClient, tagID int) ([]irv1.IndexerIR, error) {
	var indexers []IndexerResource
	if err := c.get(ctx, "/api/v1/indexer", &indexers); err != nil {
		return nil, err
	}

	var managed []irv1.IndexerIR
	for _, indexer := range indexers {
		if containsTag(indexer.Tags, tagID) {
			managed = append(managed, a.indexerToIR(&indexer))
		}
	}

	return managed, nil
}

// getRootFolders retrieves all root folders
func (a *Adapter) getRootFolders(ctx context.Context, c *httpClient) ([]irv1.RootFolderIR, error) {
	var folders []RootFolderResource
	if err := c.get(ctx, "/api/v1/rootfolder", &folders); err != nil {
		return nil, err
	}

	var result []irv1.RootFolderIR
	for _, folder := range folders {
		result = append(result, irv1.RootFolderIR{Path: folder.Path})
	}

	return result, nil
}

// downloadClientToIR converts a download client resource to IR
func (a *Adapter) downloadClientToIR(dc *DownloadClientResource) irv1.DownloadClientIR {
	ir := irv1.DownloadClientIR{
		Name:           dc.Name,
		Implementation: dc.Implementation,
		Protocol:       dc.Protocol,
		Enable:         dc.Enable,
		Priority:       dc.Priority,
	}

	// Extract settings from fields
	for _, field := range dc.Fields {
		if field.Value == nil {
			continue
		}
		switch field.Name {
		case "host":
			if v, ok := field.Value.(string); ok {
				ir.Host = v
			}
		case "port":
			if v, ok := field.Value.(float64); ok {
				ir.Port = int(v)
			}
		case "useSsl":
			if v, ok := field.Value.(bool); ok {
				ir.UseTLS = v
			}
		case "username":
			if v, ok := field.Value.(string); ok {
				ir.Username = v
			}
		case "category", "bookCategory":
			if v, ok := field.Value.(string); ok {
				ir.Category = v
			}
		}
	}

	return ir
}

// indexerToIR converts an indexer resource to IR
func (a *Adapter) indexerToIR(idx *IndexerResource) irv1.IndexerIR {
	ir := irv1.IndexerIR{
		Name:           idx.Name,
		Implementation: idx.Implementation,
		Protocol:       idx.Protocol,
		Enable:         idx.Enable,
		Priority:       idx.Priority,
	}

	// Extract settings from fields
	for _, field := range idx.Fields {
		if field.Value == nil {
			continue
		}
		switch field.Name {
		case "baseUrl":
			if v, ok := field.Value.(string); ok {
				ir.URL = v
			}
		case "minimumSeeders":
			if v, ok := field.Value.(float64); ok {
				ir.MinimumSeeders = int(v)
			}
		case "enableRss":
			if v, ok := field.Value.(bool); ok {
				ir.EnableRss = v
			}
		case "enableAutomaticSearch":
			if v, ok := field.Value.(bool); ok {
				ir.EnableAutomaticSearch = v
			}
		case "enableInteractiveSearch":
			if v, ok := field.Value.(bool); ok {
				ir.EnableInteractiveSearch = v
			}
		}
	}

	return ir
}

// containsTag checks if a slice of tag IDs contains a specific tag
func containsTag(tags []int, tagID int) bool {
	for _, t := range tags {
		if t == tagID {
			return true
		}
	}
	return false
}

// Ensure Adapter implements HealthChecker
var _ adapters.HealthChecker = (*Adapter)(nil)

// GetHealth fetches the current health status from Readarr
func (a *Adapter) GetHealth(ctx context.Context, conn *irv1.ConnectionIR) (*irv1.HealthStatus, error) {
	c := a.newClient(conn)

	var healthChecks []HealthResource
	if err := c.get(ctx, "/api/v1/health", &healthChecks); err != nil {
		return nil, fmt.Errorf("failed to get health: %w", err)
	}

	status := &irv1.HealthStatus{
		Healthy: true,
		Issues:  make([]irv1.HealthIssue, 0, len(healthChecks)),
	}

	for _, check := range healthChecks {
		issueType := irv1.HealthIssueTypeNotice
		switch check.Type {
		case "error":
			issueType = irv1.HealthIssueTypeError
			status.Healthy = false
		case "warning":
			issueType = irv1.HealthIssueTypeWarning
		}

		status.Issues = append(status.Issues, irv1.HealthIssue{
			Source:  check.Source,
			Type:    issueType,
			Message: check.Message,
			WikiURL: check.WikiURL,
		})
	}

	return status, nil
}
