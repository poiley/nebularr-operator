// Package adapters provides the interface and implementations for *arr service adapters.
// Adapters translate between the Intermediate Representation (IR) and service-specific APIs.
package adapters

import (
	"context"
	"time"

	irv1 "github.com/poiley/nebularr-operator/internal/ir/v1"
)

// Adapter defines the contract for service adapters
type Adapter interface {
	// Name returns a unique identifier for this adapter
	Name() string

	// SupportedApp returns the app this adapter handles: radarr, sonarr, lidarr, prowlarr
	SupportedApp() string

	// Connect tests connectivity and retrieves service info
	Connect(ctx context.Context, conn *irv1.ConnectionIR) (*ServiceInfo, error)

	// Discover queries the service for its capabilities
	// MUST NOT return error for missing features (degrade gracefully)
	// MUST return error only for connection failures
	Discover(ctx context.Context, conn *irv1.ConnectionIR) (*Capabilities, error)

	// CurrentState retrieves the current managed state from the service
	// Only returns resources tagged as owned by Nebularr
	CurrentState(ctx context.Context, conn *irv1.ConnectionIR) (*irv1.IR, error)

	// Diff computes the changes needed to move from current to desired state
	// MUST be deterministic (same inputs = same outputs)
	Diff(current, desired *irv1.IR, caps *Capabilities) (*ChangeSet, error)

	// Apply executes the changes against the service
	// MUST be idempotent (safe to retry)
	// MUST be fail-soft (continue on partial failures)
	// MUST tag created resources with ownership marker
	Apply(ctx context.Context, conn *irv1.ConnectionIR, changes *ChangeSet) (*ApplyResult, error)
}

// DirectApplier is an optional interface for adapters that support direct apply
// without going through the ChangeSet pattern. This is useful for resources like
// import lists, media management, and authentication that handle their own diff logic.
type DirectApplier interface {
	// ApplyDirect applies configuration directly from IR (not via ChangeSet)
	// This is used for resources that use a different sync pattern
	ApplyDirect(ctx context.Context, conn *irv1.ConnectionIR, ir *irv1.IR) (*ApplyResult, error)
}

// HealthChecker is an optional interface for adapters that support health checking.
// When implemented, the controller will fetch health status and emit K8s events.
type HealthChecker interface {
	// GetHealth fetches the current health status from the service
	GetHealth(ctx context.Context, conn *irv1.ConnectionIR) (*irv1.HealthStatus, error)
}

// ServiceInfo describes the connected service
type ServiceInfo struct {
	Version   string
	StartTime time.Time
}

// Capabilities describes what features a service supports
type Capabilities struct {
	DiscoveredAt time.Time

	// Video/Quality capabilities (Radarr/Sonarr)
	Resolutions       []string           // e.g., ["2160p", "1080p", "720p"]
	Sources           []string           // e.g., ["bluray", "webdl", "hdtv"]
	CustomFormatSpecs []CustomFormatSpec // Available custom format specification types

	// Audio capabilities (Lidarr)
	AudioTiers []string // e.g., ["lossless-hires", "lossless", "lossy-high"]

	// Download client capabilities
	DownloadClientTypes []string

	// Indexer capabilities
	IndexerTypes []string
}

// CustomFormatSpec describes an available custom format specification type
type CustomFormatSpec struct {
	Name           string // e.g., "ReleaseTitleSpecification"
	Implementation string
}

// ChangeSet describes changes to apply
type ChangeSet struct {
	Creates []Change
	Updates []Change
	Deletes []Change
}

// IsEmpty returns true if there are no changes to apply
func (cs *ChangeSet) IsEmpty() bool {
	return len(cs.Creates) == 0 && len(cs.Updates) == 0 && len(cs.Deletes) == 0
}

// TotalChanges returns the total number of changes
func (cs *ChangeSet) TotalChanges() int {
	return len(cs.Creates) + len(cs.Updates) + len(cs.Deletes)
}

// Change represents a single change to apply
type Change struct {
	ResourceType string      // e.g., "QualityProfile", "CustomFormat"
	Name         string      // Human-readable name
	ID           *int        // Service-specific ID (nil for creates)
	Payload      interface{} // Service-specific payload
}

// ApplyResult describes the outcome of applying changes
type ApplyResult struct {
	Applied int
	Failed  int
	Skipped int
	Errors  []ApplyError
}

// Success returns true if all changes were applied successfully
func (ar *ApplyResult) Success() bool {
	return ar.Failed == 0 && len(ar.Errors) == 0
}

// ApplyError represents a failure to apply a single change
type ApplyError struct {
	Change Change
	Error  error
}

// App constants
const (
	AppRadarr   = "radarr"
	AppSonarr   = "sonarr"
	AppLidarr   = "lidarr"
	AppProwlarr = "prowlarr"
)

// Resource type constants
const (
	ResourceQualityProfile  = "QualityProfile"
	ResourceCustomFormat    = "CustomFormat"
	ResourceDownloadClient  = "DownloadClient"
	ResourceIndexer         = "Indexer"
	ResourceRootFolder      = "RootFolder"
	ResourceTag             = "Tag"
	ResourceNamingConfig    = "NamingConfig"
	ResourceMetadataProfile = "MetadataProfile" // Lidarr
	ResourceApplication     = "Application"     // Prowlarr
	ResourceImportList      = "ImportList"      // Radarr/Sonarr/Lidarr
	ResourceMediaManagement = "MediaManagement" // All apps
	ResourceAuthentication  = "Authentication"  // All apps
)
