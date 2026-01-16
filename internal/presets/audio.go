package presets

// AudioQualityPreset defines an audio quality preset for Lidarr
type AudioQualityPreset struct {
	Name             string
	Description      string
	Tiers            []string // Tier names: lossless-hires, lossless, lossy-high, etc.
	UpgradeUntil     string
	PreferredFormats []string
	RejectTiers      []string
}

// AudioPresets contains all built-in audio presets
var AudioPresets = map[string]AudioQualityPreset{
	"lossless-hires": {
		Name:             "lossless-hires",
		Description:      "24-bit lossless preferred for audiophiles",
		Tiers:            []string{"lossless-hires", "lossless"},
		UpgradeUntil:     "lossless-hires",
		PreferredFormats: []string{"flac", "alac"},
		RejectTiers:      []string{"lossy-poor", "lossy-trash"},
	},
	"lossless": {
		Name:             "lossless",
		Description:      "16-bit lossless (FLAC, ALAC)",
		Tiers:            []string{"lossless", "lossless-hires", "lossy-high"},
		UpgradeUntil:     "lossless",
		PreferredFormats: []string{"flac"},
		RejectTiers:      []string{"lossy-poor", "lossy-trash"},
	},
	"high-quality": {
		Name:             "high-quality",
		Description:      "320kbps lossy or lossless",
		Tiers:            []string{"lossless", "lossy-high"},
		UpgradeUntil:     "lossless",
		PreferredFormats: []string{"flac", "mp3-320"},
		RejectTiers:      []string{"lossy-low", "lossy-poor", "lossy-trash"},
	},
	"balanced": {
		Name:         "balanced",
		Description:  "256kbps+ lossy or lossless (default)",
		Tiers:        []string{"lossless", "lossy-high", "lossy-mid"},
		UpgradeUntil: "lossy-high",
		RejectTiers:  []string{"lossy-poor", "lossy-trash"},
	},
	"portable": {
		Name:             "portable",
		Description:      "192-256kbps for mobile devices",
		Tiers:            []string{"lossy-high", "lossy-mid", "lossy-low"},
		UpgradeUntil:     "lossy-high",
		PreferredFormats: []string{"aac", "mp3-320"},
		RejectTiers:      []string{"lossy-trash", "lossless-raw"},
	},
	"any": {
		Name:         "any",
		Description:  "Accept anything, upgrade when better",
		Tiers:        []string{"lossless-hires", "lossless", "lossy-high", "lossy-mid", "lossy-low", "lossy-poor"},
		UpgradeUntil: "lossless",
		// No rejections
	},
}

// DefaultAudioPreset is used when no preset is specified
const DefaultAudioPreset = "balanced"

// GetAudioPreset returns a preset by name
func GetAudioPreset(name string) (AudioQualityPreset, bool) {
	preset, ok := AudioPresets[name]
	return preset, ok
}

// ListAudioPresets returns all available audio preset names
func ListAudioPresets() []string {
	names := make([]string, 0, len(AudioPresets))
	for name := range AudioPresets {
		names = append(names, name)
	}
	return names
}

// AudioTierDefinitions maps tier names to their Lidarr quality IDs
// These are the quality definitions that Lidarr uses internally
var AudioTierDefinitions = map[string][]int{
	"lossless-hires": {1001, 1002},             // FLAC 24bit, ALAC 24bit
	"lossless":       {1003, 1004, 1005, 1006}, // FLAC, ALAC, APE, WavPack
	"lossy-high":     {2001, 2002, 2003},       // MP3-320, AAC-320, Vorbis Q9-10
	"lossy-mid":      {2004, 2005, 2006},       // MP3-256, AAC-256, Vorbis Q7-8
	"lossy-low":      {2007, 2008},             // MP3-192, AAC-192
	"lossy-poor":     {2009, 2010},             // MP3-128, AAC-128
	"lossy-trash":    {2011, 2012},             // MP3 VBR, low quality
	"lossless-raw":   {1007},                   // WAV (uncompressed)
}
