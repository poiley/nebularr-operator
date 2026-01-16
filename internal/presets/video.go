// Package presets provides built-in preset definitions and expansion logic.
package presets

// VideoQualityPreset defines a video quality preset for Radarr/Sonarr
type VideoQualityPreset struct {
	Name             string
	Description      string
	Tiers            []QualityTier
	UpgradeUntil     *QualityTier
	PreferredFormats []string
	RejectFormats    []string
}

// QualityTier represents a resolution + sources combination
type QualityTier struct {
	Resolution string
	Sources    []string
}

// VideoPresets contains all built-in video presets
var VideoPresets = map[string]VideoQualityPreset{
	"4k-hdr": {
		Name:        "4k-hdr",
		Description: "4K with HDR, falls back to 1080p",
		Tiers: []QualityTier{
			{Resolution: "2160p", Sources: []string{"remux", "bluray", "webdl", "webrip"}},
			{Resolution: "1080p", Sources: []string{"remux", "bluray", "webdl", "webrip"}},
		},
		UpgradeUntil:     &QualityTier{Resolution: "2160p", Sources: []string{"remux"}},
		PreferredFormats: []string{"hdr10", "hdr10plus", "dolby-vision", "atmos", "truehd", "dts-x"},
		RejectFormats:    []string{"cam", "telesync", "telecine", "workprint", "3d"},
	},
	"4k-sdr": {
		Name:        "4k-sdr",
		Description: "4K without HDR requirement",
		Tiers: []QualityTier{
			{Resolution: "2160p", Sources: []string{"remux", "bluray", "webdl", "webrip"}},
			{Resolution: "1080p", Sources: []string{"remux", "bluray", "webdl", "webrip"}},
		},
		UpgradeUntil:     &QualityTier{Resolution: "2160p", Sources: []string{"remux"}},
		PreferredFormats: []string{"atmos", "truehd", "dts-x", "dts-hd"},
		RejectFormats:    []string{"cam", "telesync", "telecine", "workprint", "3d"},
	},
	"1080p-quality": {
		Name:        "1080p-quality",
		Description: "1080p bluray/remux preferred",
		Tiers: []QualityTier{
			{Resolution: "1080p", Sources: []string{"remux", "bluray"}},
			{Resolution: "1080p", Sources: []string{"webdl", "webrip"}},
			{Resolution: "720p", Sources: []string{"bluray", "webdl"}},
		},
		UpgradeUntil:     &QualityTier{Resolution: "1080p", Sources: []string{"remux"}},
		PreferredFormats: []string{"truehd", "dts-hd", "atmos"},
		RejectFormats:    []string{"cam", "telesync", "telecine", "workprint"},
	},
	"1080p-streaming": {
		Name:        "1080p-streaming",
		Description: "1080p web sources for smaller files",
		Tiers: []QualityTier{
			{Resolution: "1080p", Sources: []string{"webdl", "webrip"}},
			{Resolution: "1080p", Sources: []string{"hdtv"}},
			{Resolution: "720p", Sources: []string{"webdl", "webrip"}},
		},
		UpgradeUntil:     &QualityTier{Resolution: "1080p", Sources: []string{"webdl"}},
		PreferredFormats: []string{"hevc", "aac"},
		RejectFormats:    []string{"cam", "telesync", "telecine", "workprint"},
	},
	"720p": {
		Name:        "720p",
		Description: "720p any source for limited storage/bandwidth",
		Tiers: []QualityTier{
			{Resolution: "720p", Sources: []string{"bluray", "webdl", "webrip", "hdtv"}},
			{Resolution: "480p", Sources: []string{"webdl", "dvd"}},
		},
		UpgradeUntil:  &QualityTier{Resolution: "720p", Sources: []string{"bluray"}},
		RejectFormats: []string{"cam", "telesync", "telecine", "workprint"},
	},
	"balanced": {
		Name:        "balanced",
		Description: "1080p preferred, accepts 720p-4K (default)",
		Tiers: []QualityTier{
			{Resolution: "2160p", Sources: []string{"bluray", "webdl"}},
			{Resolution: "1080p", Sources: []string{"remux", "bluray", "webdl", "webrip"}},
			{Resolution: "720p", Sources: []string{"bluray", "webdl"}},
		},
		UpgradeUntil:     &QualityTier{Resolution: "1080p", Sources: []string{"bluray"}},
		PreferredFormats: []string{"hdr10", "dolby-vision"},
		RejectFormats:    []string{"cam", "telesync", "telecine", "workprint"},
	},
	"any": {
		Name:        "any",
		Description: "Accept anything, upgrade when better",
		Tiers: []QualityTier{
			{Resolution: "2160p", Sources: []string{"remux", "bluray", "webdl", "webrip", "hdtv"}},
			{Resolution: "1080p", Sources: []string{"remux", "bluray", "webdl", "webrip", "hdtv"}},
			{Resolution: "720p", Sources: []string{"bluray", "webdl", "webrip", "hdtv"}},
			{Resolution: "480p", Sources: []string{"webdl", "webrip", "dvd", "sdtv"}},
		},
		UpgradeUntil:  &QualityTier{Resolution: "2160p", Sources: []string{"remux"}},
		RejectFormats: []string{"cam", "workprint"},
	},
	"storage-optimized": {
		Name:        "storage-optimized",
		Description: "Balance quality vs file size",
		Tiers: []QualityTier{
			{Resolution: "1080p", Sources: []string{"webdl", "webrip"}},
			{Resolution: "720p", Sources: []string{"webdl", "webrip"}},
		},
		UpgradeUntil:     &QualityTier{Resolution: "1080p", Sources: []string{"webdl"}},
		PreferredFormats: []string{"hevc", "av1", "aac"},
		RejectFormats:    []string{"remux", "truehd", "dts-hd", "cam", "telesync"},
	},
}

// DefaultVideoPreset is used when no preset is specified
const DefaultVideoPreset = "balanced"

// GetVideoPreset returns a preset by name
func GetVideoPreset(name string) (VideoQualityPreset, bool) {
	preset, ok := VideoPresets[name]
	return preset, ok
}

// ListVideoPresets returns all available video preset names
func ListVideoPresets() []string {
	names := make([]string, 0, len(VideoPresets))
	for name := range VideoPresets {
		names = append(names, name)
	}
	return names
}
