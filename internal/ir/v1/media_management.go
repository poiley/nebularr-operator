package v1

// MediaManagementIR represents media management configuration
type MediaManagementIR struct {
	// RecycleBin is the path to the recycle bin folder
	// Empty string means disabled
	RecycleBin string `json:"recycleBin,omitempty"`

	// RecycleBinCleanupDays is the number of days before items are removed
	RecycleBinCleanupDays int `json:"recycleBinCleanupDays"`

	// SetPermissions enables setting file permissions on Linux
	SetPermissions bool `json:"setPermissions"`

	// ChmodFolder is the folder permission mode (e.g., "755")
	ChmodFolder string `json:"chmodFolder"`

	// ChownGroup is the group to set for files
	ChownGroup string `json:"chownGroup,omitempty"`

	// DeleteEmptyFolders removes empty folders after moving/deleting files
	DeleteEmptyFolders bool `json:"deleteEmptyFolders"`

	// CreateEmptyFolders creates folders for artists/movies/series even when empty
	CreateEmptyFolders bool `json:"createEmptyFolders"`

	// UseHardlinks uses hardlinks instead of copy when possible
	UseHardlinks bool `json:"useHardlinks"`

	// --- Lidarr-specific ---

	// WatchLibraryForChanges monitors the library folder for changes
	WatchLibraryForChanges *bool `json:"watchLibraryForChanges,omitempty"`

	// AllowFingerprinting: never, newFiles, always
	AllowFingerprinting string `json:"allowFingerprinting,omitempty"`
}
