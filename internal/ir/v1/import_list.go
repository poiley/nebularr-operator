package v1

// ImportListIR represents an import list configuration
type ImportListIR struct {
	// Name is the display name for this import list
	Name string `json:"name"`

	// Type is the implementation type (e.g., "IMDbListImport", "TraktListImport")
	Type string `json:"type"`

	// Enabled controls whether this list is active
	Enabled bool `json:"enabled"`

	// EnableAuto automatically adds items from this list
	EnableAuto bool `json:"enableAuto"`

	// SearchOnAdd searches for items when added from this list
	SearchOnAdd bool `json:"searchOnAdd"`

	// QualityProfileID is the resolved quality profile ID
	QualityProfileID int `json:"qualityProfileId"`

	// QualityProfileName is for reference/logging
	QualityProfileName string `json:"qualityProfileName,omitempty"`

	// RootFolderPath is the validated root folder path
	RootFolderPath string `json:"rootFolderPath"`

	// --- Radarr-specific ---

	// Monitor: movieOnly, movieAndCollection, none
	Monitor string `json:"monitor,omitempty"`

	// MinimumAvailability: tba, announced, inCinemas, released
	MinimumAvailability string `json:"minimumAvailability,omitempty"`

	// --- Sonarr-specific ---

	// SeriesType: standard, daily, anime
	SeriesType string `json:"seriesType,omitempty"`

	// SeasonFolder enables season folders
	SeasonFolder bool `json:"seasonFolder,omitempty"`

	// ShouldMonitor: all, future, missing, existing, firstSeason, latestSeason, pilot, none
	ShouldMonitor string `json:"shouldMonitor,omitempty"`

	// --- Type-specific settings ---

	// Settings contains type-specific field values
	// Keys are the API field names (camelCase)
	Settings map[string]string `json:"settings,omitempty"`
}
