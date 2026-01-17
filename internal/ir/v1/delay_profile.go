package v1

// DelayProfileIR represents a delay profile configuration.
// Delay profiles control download timing to wait for better releases.
type DelayProfileIR struct {
	// ID is the profile ID (set by the service, used for updates/deletes)
	ID int `json:"id,omitempty"`

	// Name is a display name for identification (not sent to API)
	Name string `json:"name,omitempty"`

	// Order determines priority (lower = higher priority)
	Order int `json:"order"`

	// PreferredProtocol: "usenet" or "torrent"
	PreferredProtocol string `json:"preferredProtocol"`

	// UsenetDelay in minutes
	UsenetDelay int `json:"usenetDelay"`

	// TorrentDelay in minutes
	TorrentDelay int `json:"torrentDelay"`

	// EnableUsenet allows downloading from Usenet
	EnableUsenet bool `json:"enableUsenet"`

	// EnableTorrent allows downloading from torrents
	EnableTorrent bool `json:"enableTorrent"`

	// BypassIfHighestQuality bypasses delay if release is at cutoff quality
	BypassIfHighestQuality bool `json:"bypassIfHighestQuality"`

	// BypassIfAboveCustomFormatScore bypasses delay based on CF score
	BypassIfAboveCustomFormatScore bool `json:"bypassIfAboveCustomFormatScore"`

	// MinimumCustomFormatScore is the CF score threshold for bypass
	MinimumCustomFormatScore int `json:"minimumCustomFormatScore"`

	// Tags restricts this profile to items with these tags (tag IDs)
	Tags []int `json:"tags,omitempty"`

	// TagNames are the tag names (used by compiler, resolved to IDs by adapter)
	TagNames []string `json:"tagNames,omitempty"`
}
