package v1

// NotificationIR represents the internal representation of a notification connection.
// This is a schema-based resource - the available fields depend on the implementation type.
type NotificationIR struct {
	// ID is the notification ID in the *arr app (0 if new)
	ID int `json:"id,omitempty"`

	// Name is the display name for this notification
	Name string `json:"name"`

	// Implementation is the notification type (e.g., Discord, Slack, Email, Webhook, Telegram)
	Implementation string `json:"implementation"`

	// ConfigContract is the settings contract name (usually Implementation + "Settings")
	ConfigContract string `json:"configContract,omitempty"`

	// Enabled indicates if this notification is active
	Enabled bool `json:"enabled"`

	// --- Event Triggers (common) ---

	OnGrab                      bool `json:"onGrab,omitempty"`
	OnDownload                  bool `json:"onDownload,omitempty"`
	OnUpgrade                   bool `json:"onUpgrade,omitempty"`
	OnRename                    bool `json:"onRename,omitempty"`
	OnHealthIssue               bool `json:"onHealthIssue,omitempty"`
	OnHealthRestored            bool `json:"onHealthRestored,omitempty"`
	OnApplicationUpdate         bool `json:"onApplicationUpdate,omitempty"`
	OnManualInteractionRequired bool `json:"onManualInteractionRequired,omitempty"`
	IncludeHealthWarnings       bool `json:"includeHealthWarnings,omitempty"`

	// --- Radarr-specific Events ---

	OnMovieAdded                bool `json:"onMovieAdded,omitempty"`
	OnMovieDelete               bool `json:"onMovieDelete,omitempty"`
	OnMovieFileDelete           bool `json:"onMovieFileDelete,omitempty"`
	OnMovieFileDeleteForUpgrade bool `json:"onMovieFileDeleteForUpgrade,omitempty"`

	// --- Sonarr-specific Events ---

	OnSeriesAdd                   bool `json:"onSeriesAdd,omitempty"`
	OnSeriesDelete                bool `json:"onSeriesDelete,omitempty"`
	OnEpisodeFileDelete           bool `json:"onEpisodeFileDelete,omitempty"`
	OnEpisodeFileDeleteForUpgrade bool `json:"onEpisodeFileDeleteForUpgrade,omitempty"`

	// --- Lidarr-specific Events ---

	OnReleaseImport   bool `json:"onReleaseImport,omitempty"`
	OnArtistAdd       bool `json:"onArtistAdd,omitempty"`
	OnArtistDelete    bool `json:"onArtistDelete,omitempty"`
	OnAlbumDelete     bool `json:"onAlbumDelete,omitempty"`
	OnTrackRetag      bool `json:"onTrackRetag,omitempty"`
	OnDownloadFailure bool `json:"onDownloadFailure,omitempty"`
	OnImportFailure   bool `json:"onImportFailure,omitempty"`

	// --- Type-specific Settings ---

	// Fields contains the resolved type-specific configuration as key-value pairs.
	// These are passed to the API as fields array.
	Fields map[string]interface{} `json:"fields,omitempty"`

	// Tags are tag IDs to apply to this notification
	Tags []int `json:"tags,omitempty"`
}
