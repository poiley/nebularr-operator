package v1

// RemotePathMappingIR represents a remote path mapping in the IR.
// Remote path mappings translate paths between download clients and *arr apps
// when they see the same files at different paths.
type RemotePathMappingIR struct {
	// ID is the mapping ID in the *arr app (0 for new mappings)
	ID int `json:"id,omitempty"`

	// Host is the download client hostname
	// Must match the host configured in the download client
	Host string `json:"host"`

	// RemotePath is the path as reported by the download client
	RemotePath string `json:"remotePath"`

	// LocalPath is the path as seen by the *arr app
	LocalPath string `json:"localPath"`
}
