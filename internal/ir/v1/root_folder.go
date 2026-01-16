package v1

// RootFolderIR represents a root folder
type RootFolderIR struct {
	Path string `json:"path"`

	// Lidarr-specific fields
	Name           string `json:"name,omitempty"`           // Lidarr only
	DefaultMonitor string `json:"defaultMonitor,omitempty"` // Lidarr: all, future, missing, etc.
}

// Lidarr monitor constants
const (
	MonitorAll        = "all"
	MonitorFuture     = "future"
	MonitorMissing    = "missing"
	MonitorExisting   = "existing"
	MonitorFirstAlbum = "firstAlbum"
	MonitorLatest     = "latest"
	MonitorNone       = "none"
)
