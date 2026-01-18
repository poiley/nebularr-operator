package v1

// QualityIR wraps video, audio, or book quality (union type)
type QualityIR struct {
	// Video quality for Radarr/Sonarr
	Video *VideoQualityIR `json:"video,omitempty"`

	// Audio quality for Lidarr
	Audio *AudioQualityIR `json:"audio,omitempty"`

	// Book quality for Readarr
	Book *BookQualityIR `json:"book,omitempty"`
}

// VideoQualityIR represents video quality configuration (from preset or manual)
type VideoQualityIR struct {
	// ProfileName is the quality profile name (generated: "nebularr-{config-name}")
	ProfileName string `json:"profileName"`

	// UpgradeAllowed enables quality upgrades
	UpgradeAllowed bool `json:"upgradeAllowed"`

	// Cutoff is the quality tier where upgrades stop
	Cutoff VideoQualityTierIR `json:"cutoff"`

	// Tiers defines the quality ranking (ordered, first = lowest priority)
	Tiers []VideoQualityTierIR `json:"tiers"`

	// CustomFormats to create
	CustomFormats []CustomFormatIR `json:"customFormats,omitempty"`

	// FormatScores maps format names to scores
	FormatScores map[string]int `json:"formatScores,omitempty"`

	// MinimumCustomFormatScore for acceptance
	MinimumCustomFormatScore int `json:"minimumCustomFormatScore,omitempty"`

	// UpgradeUntilCustomFormatScore stops upgrades at this score
	UpgradeUntilCustomFormatScore int `json:"upgradeUntilCustomFormatScore,omitempty"`
}

// VideoQualityTierIR represents an abstract quality level
type VideoQualityTierIR struct {
	Resolution string   `json:"resolution"` // 2160p, 1080p, 720p, 480p
	Sources    []string `json:"sources"`    // bluray, webdl, webrip, hdtv, etc.
	Allowed    bool     `json:"allowed"`
}

// CustomFormatIR represents a custom format definition
type CustomFormatIR struct {
	ID                  int            `json:"id,omitempty"`
	Name                string         `json:"name"`
	IncludeWhenRenaming bool           `json:"includeWhenRenaming,omitempty"`
	Specifications      []FormatSpecIR `json:"specifications"`
}

// FormatSpecIR represents a single format matching rule
type FormatSpecIR struct {
	Type     string `json:"type"` // ReleaseTitleSpecification, SourceSpecification, etc.
	Name     string `json:"name"`
	Negate   bool   `json:"negate,omitempty"`
	Required bool   `json:"required,omitempty"`
	Value    string `json:"value"`
}

// AudioQualityIR represents audio quality configuration for Lidarr
type AudioQualityIR struct {
	// ProfileName is the quality profile name
	ProfileName string `json:"profileName"`

	// UpgradeAllowed enables quality upgrades
	UpgradeAllowed bool `json:"upgradeAllowed"`

	// Cutoff is the tier where upgrades stop
	Cutoff string `json:"cutoff"` // lossless-hires, lossless, lossy-high, etc.

	// Tiers defines the quality ranking
	Tiers []AudioQualityTierIR `json:"tiers"`

	// ReleaseProfile for Lidarr release filtering
	ReleaseProfile *ReleaseProfileIR `json:"releaseProfile,omitempty"`

	// FormatScores maps format names to scores (Lidarr v2+)
	FormatScores map[string]int `json:"formatScores,omitempty"`

	// MinimumCustomFormatScore for acceptance
	MinimumCustomFormatScore int `json:"minimumCustomFormatScore,omitempty"`

	// UpgradeUntilCustomFormatScore stops upgrades at this score
	UpgradeUntilCustomFormatScore int `json:"upgradeUntilCustomFormatScore,omitempty"`
}

// AudioQualityTierIR represents an audio quality tier
type AudioQualityTierIR struct {
	Tier    string `json:"tier"` // lossless-hires, lossless, lossy-high, lossy-mid, lossy-low
	Allowed bool   `json:"allowed"`
}

// ReleaseProfileIR for release filtering (Sonarr/Lidarr)
type ReleaseProfileIR struct {
	// ID is the server-side ID (nil for creates)
	ID *int `json:"id,omitempty"`
	// Name is the display name
	Name string `json:"name"`
	// Enabled indicates if the profile is active
	Enabled bool `json:"enabled"`
	// Required terms that must be present
	Required []string `json:"required,omitempty"`
	// Ignored terms that must not be present
	Ignored []string `json:"ignored,omitempty"`
	// Preferred terms with scores
	Preferred []PreferredTermIR `json:"preferred,omitempty"`
	// IncludePreferredWhenRenaming includes preferred terms in naming
	IncludePreferredWhenRenaming bool `json:"includePreferredWhenRenaming,omitempty"`
	// IndexerID restricts to specific indexer (0 = all)
	IndexerID int `json:"indexerId,omitempty"`
	// Tags restricts to specific tags
	Tags []int `json:"tags,omitempty"`
}

// PreferredTermIR represents a preferred term with score
type PreferredTermIR struct {
	Term  string `json:"term"`
	Score int    `json:"score"`
}

// BookQualityIR represents book quality configuration for Readarr
type BookQualityIR struct {
	// ProfileName is the quality profile name
	ProfileName string `json:"profileName"`

	// UpgradeAllowed enables quality upgrades
	UpgradeAllowed bool `json:"upgradeAllowed"`

	// Cutoff is the quality tier where upgrades stop
	Cutoff BookQualityTierIR `json:"cutoff"`

	// Tiers defines the quality ranking (ordered, first = lowest priority)
	Tiers []BookQualityTierIR `json:"tiers"`
}

// BookQualityTierIR represents a book quality tier
type BookQualityTierIR struct {
	// Name is the tier/group name (e.g., "EPUB", "MOBI", "AZW3", "PDF")
	Name string `json:"name"`

	// Formats are the specific formats in this tier
	Formats []string `json:"formats,omitempty"`

	// Allowed indicates if this tier is allowed
	Allowed bool `json:"allowed"`
}

// MetadataProfileIR represents a metadata profile for Readarr
type MetadataProfileIR struct {
	// ID is the server-side ID (nil for creates)
	ID *int `json:"id,omitempty"`

	// Name is the profile name
	Name string `json:"name"`

	// MinPopularity is the minimum GoodReads popularity score
	MinPopularity float64 `json:"minPopularity,omitempty"`

	// SkipMissingDate skips books without release date
	SkipMissingDate bool `json:"skipMissingDate,omitempty"`

	// SkipMissingIsbn skips books without ISBN
	SkipMissingIsbn bool `json:"skipMissingIsbn,omitempty"`

	// SkipPartsAndSets skips parts and sets
	SkipPartsAndSets bool `json:"skipPartsAndSets,omitempty"`

	// SkipSeriesSecondary skips non-primary series entries
	SkipSeriesSecondary bool `json:"skipSeriesSecondary,omitempty"`

	// AllowedLanguages is a comma-separated list of allowed language codes
	AllowedLanguages string `json:"allowedLanguages,omitempty"`
}
