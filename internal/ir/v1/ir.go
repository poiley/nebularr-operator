// Package v1 contains the Intermediate Representation (IR) types for Nebularr.
// The IR is the internal representation that the Policy Compiler produces and Adapters consume.
// It is domain-based, versioned, and must not import any adapter-specific types.
package v1

import "time"

// IR is the top-level intermediate representation for an *arr app
type IR struct {
	// Version of this IR schema
	Version string `json:"version"`

	// GeneratedAt is when this IR was compiled
	GeneratedAt time.Time `json:"generatedAt"`

	// SourceHash is a hash of the intent that produced this IR
	// Used for drift detection (if hash unchanged, skip reconciliation)
	SourceHash string `json:"sourceHash"`

	// App identifies which app this IR is for: radarr, sonarr, lidarr, prowlarr
	App string `json:"app"`

	// Connection details for the app
	Connection *ConnectionIR `json:"connection,omitempty"`

	// Quality configuration (Video for Radarr/Sonarr, Audio for Lidarr)
	Quality *QualityIR `json:"quality,omitempty"`

	// DownloadClients configuration
	DownloadClients []DownloadClientIR `json:"downloadClients,omitempty"`

	// RemotePathMappings configuration
	RemotePathMappings []RemotePathMappingIR `json:"remotePathMappings,omitempty"`

	// Indexers configuration (or ProwlarrRef) - for Radarr/Sonarr/Lidarr
	Indexers *IndexersIR `json:"indexers,omitempty"`

	// Naming configuration
	Naming *NamingIR `json:"naming,omitempty"`

	// RootFolders configuration
	RootFolders []RootFolderIR `json:"rootFolders,omitempty"`

	// ImportLists configuration - for Radarr/Sonarr/Lidarr
	ImportLists []ImportListIR `json:"importLists,omitempty"`

	// MediaManagement configuration
	MediaManagement *MediaManagementIR `json:"mediaManagement,omitempty"`

	// Authentication configuration
	Authentication *AuthenticationIR `json:"authentication,omitempty"`

	// Prowlarr-specific configuration (only populated when App == "prowlarr")
	Prowlarr *ProwlarrIR `json:"prowlarr,omitempty"`

	// Unrealized tracks features that could not be compiled
	// (due to missing capabilities)
	Unrealized []UnrealizedFeature `json:"unrealized,omitempty"`
}

// UnrealizedFeature represents something the user requested
// that cannot be realized given current capabilities
type UnrealizedFeature struct {
	Feature string `json:"feature"` // e.g., "format:dolby-vision"
	Reason  string `json:"reason"`  // e.g., "not supported by service version"
}

// IRVersion is the current version of the IR schema
const IRVersion = "v1"
