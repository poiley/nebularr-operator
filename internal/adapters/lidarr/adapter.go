// Package lidarr provides the Lidarr adapter for Nebularr.
// It implements the adapters.Adapter interface for managing Lidarr configuration.
// Note: Lidarr uses API v1, not v3 like Radarr/Sonarr.
package lidarr

import (
	"context"
	"fmt"
	"time"

	"github.com/poiley/nebularr-operator/internal/adapters"
	"github.com/poiley/nebularr-operator/internal/adapters/httpclient"
	irv1 "github.com/poiley/nebularr-operator/internal/ir/v1"
)

// Adapter implements the adapters.Adapter interface for Lidarr
type Adapter struct{}

// Ensure Adapter implements the interface
var _ adapters.Adapter = (*Adapter)(nil)

// Name returns a unique identifier for this adapter
func (a *Adapter) Name() string {
	return "lidarr"
}

// SupportedApp returns the app this adapter handles
func (a *Adapter) SupportedApp() string {
	return adapters.AppLidarr
}

// Connect tests connectivity and retrieves service info
func (a *Adapter) Connect(ctx context.Context, conn *irv1.ConnectionIR) (*adapters.ServiceInfo, error) {
	c := a.newClient(conn)

	var status SystemResource
	if err := c.Get(ctx, "/api/v1/system/status", &status); err != nil {
		return nil, fmt.Errorf("failed to connect to Lidarr: %w", err)
	}

	info := &adapters.ServiceInfo{
		Version: status.Version,
	}

	if status.StartTime != nil {
		info.StartTime = *status.StartTime
	}

	return info, nil
}

// Discover queries Lidarr for its capabilities
func (a *Adapter) Discover(ctx context.Context, conn *irv1.ConnectionIR) (*adapters.Capabilities, error) {
	c := a.newClient(conn)

	caps := &adapters.Capabilities{
		DiscoveredAt: time.Now(),
		// Lidarr audio tiers
		AudioTiers: []string{"lossless-hires", "lossless", "lossy-high", "lossy-mid", "lossy-low"},
	}

	// Discover download client types
	var dcSchemas []DownloadClientResource
	if err := c.Get(ctx, "/api/v1/downloadclient/schema", &dcSchemas); err == nil {
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
	if err := c.Get(ctx, "/api/v1/indexer/schema", &idxSchemas); err == nil {
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

// CurrentState retrieves the current managed state from Lidarr
func (a *Adapter) CurrentState(ctx context.Context, conn *irv1.ConnectionIR) (*irv1.IR, error) {
	c := a.newClient(conn)

	ir := &irv1.IR{
		Version:     irv1.IRVersion,
		GeneratedAt: time.Now(),
		App:         adapters.AppLidarr,
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
			Audio: profiles[0],
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

	// Get remote path mappings (not tag-based, get all)
	if mappings, err := a.getRemotePathMappings(ctx, c); err == nil {
		ir.RemotePathMappings = mappings
	}

	// Get notifications tagged with ownership tag
	if notifications, err := a.getManagedNotifications(ctx, c, tagID); err == nil {
		ir.Notifications = notifications
	}

	// Get naming config
	if naming, err := a.getNamingConfig(ctx, c); err == nil {
		ir.Naming = &irv1.NamingIR{
			Lidarr: naming,
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

	// Get custom formats (all of them, since they don't have tags)
	// Note: Requires Lidarr v2.0+
	if customFormats, err := a.getAllCustomFormats(ctx, c); err == nil {
		ir.CustomFormats = customFormats
	}

	// Get delay profiles (all of them, not just tagged)
	if delayProfiles, err := a.getManagedDelayProfiles(ctx, c); err == nil {
		ir.DelayProfiles = delayProfiles
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

	// Diff remote path mappings
	if err := a.diffRemotePathMappings(current, desired, changes); err != nil {
		return nil, fmt.Errorf("failed to diff remote path mappings: %w", err)
	}

	// Diff notifications
	if err := a.diffNotifications(current, desired, changes); err != nil {
		return nil, fmt.Errorf("failed to diff notifications: %w", err)
	}

	// Diff naming config
	if err := a.diffNaming(current, desired, changes); err != nil {
		return nil, fmt.Errorf("failed to diff naming: %w", err)
	}

	// Diff custom formats
	if err := a.diffCustomFormats(current, desired, changes); err != nil {
		return nil, fmt.Errorf("failed to diff custom formats: %w", err)
	}

	// Diff delay profiles
	if err := a.diffDelayProfiles(current, desired, changes); err != nil {
		return nil, fmt.Errorf("failed to diff delay profiles: %w", err)
	}

	return changes, nil
}

// Apply executes the changes against Lidarr
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

// newClient creates a new HTTP client for Lidarr API communication
func (a *Adapter) newClient(conn *irv1.ConnectionIR) *httpclient.Client {
	return httpclient.New(httpclient.Config{
		BaseURL:            conn.URL,
		APIKey:             conn.APIKey,
		InsecureSkipVerify: conn.InsecureSkipVerify,
	})
}

// Ensure Adapter implements HealthChecker
// Note: HealthResource is now defined as a type alias in types.go
var _ adapters.HealthChecker = (*Adapter)(nil)

// GetHealth fetches the current health status from Lidarr
func (a *Adapter) GetHealth(ctx context.Context, conn *irv1.ConnectionIR) (*irv1.HealthStatus, error) {
	c := a.newClient(conn)

	var healthChecks []HealthResource
	if err := c.Get(ctx, "/api/v1/health", &healthChecks); err != nil {
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
