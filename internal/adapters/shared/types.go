// Package shared provides common types used across multiple *arr adapters.
// This consolidates duplicated type definitions to reduce code duplication
// and ensure consistency across Sonarr, Lidarr, Readarr, and Prowlarr adapters.
package shared

import "time"

// SystemResource represents system status response from *arr APIs.
// Used to get version information and system health.
type SystemResource struct {
	Version   string     `json:"version"`
	StartTime *time.Time `json:"startTime"`
}

// TagResource represents a tag used for managing and grouping resources.
// Tags can be applied to download clients, indexers, notifications, etc.
type TagResource struct {
	ID    int    `json:"id,omitempty"`
	Label string `json:"label"`
}

// HealthResource represents a health check response from *arr APIs.
// Contains information about system health issues, warnings, and notices.
type HealthResource struct {
	Source  string `json:"source"`
	Type    string `json:"type"` // error, warning, notice
	Message string `json:"message"`
	WikiURL string `json:"wikiUrl"`
}

// Field represents a dynamic configuration field used in download clients,
// indexers, notifications, and other configurable resources.
type Field struct {
	Name  string      `json:"name"`
	Value interface{} `json:"value"`
}

// RemotePathMappingResource represents a remote path mapping configuration.
// Used to translate paths between the *arr application and download clients.
type RemotePathMappingResource struct {
	ID         int    `json:"id,omitempty"`
	Host       string `json:"host"`
	RemotePath string `json:"remotePath"`
	LocalPath  string `json:"localPath"`
}
