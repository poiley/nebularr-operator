package v1

// QualityIR wraps video or audio quality (union type)
type QualityIR struct {
	// Video quality for Radarr/Sonarr
	Video *VideoQualityIR `json:"video,omitempty"`

	// Audio quality for Lidarr
	Audio *AudioQualityIR `json:"audio,omitempty"`
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
}

// AudioQualityTierIR represents an audio quality tier
type AudioQualityTierIR struct {
	Tier    string `json:"tier"` // lossless-hires, lossless, lossy-high, lossy-mid, lossy-low
	Allowed bool   `json:"allowed"`
}

// ReleaseProfileIR for Lidarr release filtering
type ReleaseProfileIR struct {
	Required []string `json:"required,omitempty"`
	Ignored  []string `json:"ignored,omitempty"`
}
