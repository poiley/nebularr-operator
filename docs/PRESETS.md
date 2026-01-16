# Nebularr - Presets Reference

> **For coding agents:** This document defines built-in presets and how they expand.
>
> **Related:** [CRDS](./CRDS.md) | [TYPES](./TYPES.md) | [RADARR](./RADARR.md) | [SONARR](./SONARR.md) | [LIDARR](./LIDARR.md)

Presets provide sensible defaults for common configurations. Users can use presets as-is, customize with overrides/excludes, or ignore presets entirely for full manual control.

---

## 1. Video Quality Presets

For Radarr and Sonarr.

### 1.1 Preset Definitions

| Preset | Description | Target Users |
|--------|-------------|--------------|
| `4k-hdr` | 4K with HDR, falls back to 1080p | Home theater, HDR display |
| `4k-sdr` | 4K without HDR requirement | 4K display, no HDR |
| `1080p-quality` | 1080p bluray/remux preferred | Quality-focused |
| `1080p-streaming` | 1080p web sources | Smaller files |
| `720p` | 720p any source | Limited storage/bandwidth |
| `balanced` | 1080p preferred, accepts 720p-4K | **Default preset** |
| `any` | Accept anything, upgrade when better | Availability-focused |
| `storage-optimized` | Balance quality vs file size | Limited storage |

### 1.2 Preset Expansions

#### `4k-hdr`

```yaml
# Expands to:
tiers:
  - resolution: "2160p"
    sources: ["remux", "bluray", "webdl", "webrip"]
  - resolution: "1080p"
    sources: ["remux", "bluray", "webdl", "webrip"]
upgradeUntil:
  resolution: "2160p"
  source: "remux"
preferredFormats:
  - "hdr10"
  - "hdr10plus"
  - "dolby-vision"
  - "atmos"
  - "truehd"
  - "dts-x"
rejectFormats:
  - "cam"
  - "telesync"
  - "telecine"
  - "workprint"
  - "3d"
```

#### `4k-sdr`

```yaml
tiers:
  - resolution: "2160p"
    sources: ["remux", "bluray", "webdl", "webrip"]
  - resolution: "1080p"
    sources: ["remux", "bluray", "webdl", "webrip"]
upgradeUntil:
  resolution: "2160p"
  source: "remux"
preferredFormats:
  - "atmos"
  - "truehd"
  - "dts-x"
  - "dts-hd"
# No HDR formats preferred
rejectFormats:
  - "cam"
  - "telesync"
  - "telecine"
  - "workprint"
  - "3d"
```

#### `1080p-quality`

```yaml
tiers:
  - resolution: "1080p"
    sources: ["remux", "bluray"]
  - resolution: "1080p"
    sources: ["webdl", "webrip"]
  - resolution: "720p"
    sources: ["bluray", "webdl"]
upgradeUntil:
  resolution: "1080p"
  source: "remux"
preferredFormats:
  - "truehd"
  - "dts-hd"
  - "atmos"
rejectFormats:
  - "cam"
  - "telesync"
  - "telecine"
  - "workprint"
```

#### `1080p-streaming`

```yaml
tiers:
  - resolution: "1080p"
    sources: ["webdl", "webrip"]
  - resolution: "1080p"
    sources: ["hdtv"]
  - resolution: "720p"
    sources: ["webdl", "webrip"]
upgradeUntil:
  resolution: "1080p"
  source: "webdl"
preferredFormats:
  - "hevc"  # Smaller files
  - "aac"
rejectFormats:
  - "cam"
  - "telesync"
  - "telecine"
  - "workprint"
```

#### `720p`

```yaml
tiers:
  - resolution: "720p"
    sources: ["bluray", "webdl", "webrip", "hdtv"]
  - resolution: "480p"
    sources: ["webdl", "dvd"]
upgradeUntil:
  resolution: "720p"
  source: "bluray"
rejectFormats:
  - "cam"
  - "telesync"
  - "telecine"
  - "workprint"
```

#### `balanced` (Default)

```yaml
tiers:
  - resolution: "2160p"
    sources: ["bluray", "webdl"]
  - resolution: "1080p"
    sources: ["remux", "bluray", "webdl", "webrip"]
  - resolution: "720p"
    sources: ["bluray", "webdl"]
upgradeUntil:
  resolution: "1080p"
  source: "bluray"
preferredFormats:
  - "hdr10"
  - "dolby-vision"
rejectFormats:
  - "cam"
  - "telesync"
  - "telecine"
  - "workprint"
```

#### `any`

```yaml
tiers:
  - resolution: "2160p"
    sources: ["remux", "bluray", "webdl", "webrip", "hdtv"]
  - resolution: "1080p"
    sources: ["remux", "bluray", "webdl", "webrip", "hdtv"]
  - resolution: "720p"
    sources: ["bluray", "webdl", "webrip", "hdtv"]
  - resolution: "480p"
    sources: ["webdl", "webrip", "dvd", "sdtv"]
upgradeUntil:
  resolution: "2160p"
  source: "remux"
# No preferred formats - accept anything
# No rejected formats except truly unwatchable
rejectFormats:
  - "cam"
  - "workprint"
```

#### `storage-optimized`

```yaml
tiers:
  - resolution: "1080p"
    sources: ["webdl", "webrip"]
  - resolution: "720p"
    sources: ["webdl", "webrip"]
upgradeUntil:
  resolution: "1080p"
  source: "webdl"
preferredFormats:
  - "hevc"   # ~40% smaller than h264
  - "av1"    # Even smaller
  - "aac"    # Smaller than lossless
rejectFormats:
  - "remux"  # Too large
  - "truehd" # Large audio
  - "dts-hd"
  - "cam"
  - "telesync"
```

### 1.3 Go Implementation

```go
// internal/presets/video.go

package presets

// VideoQualityPreset defines a video quality preset
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
        UpgradeUntil:     &QualityTier{Resolution: "2160p", Source: "remux"},
        PreferredFormats: []string{"hdr10", "hdr10plus", "dolby-vision", "atmos", "truehd", "dts-x"},
        RejectFormats:    []string{"cam", "telesync", "telecine", "workprint", "3d"},
    },
    "balanced": {
        Name:        "balanced",
        Description: "1080p preferred, accepts 720p-4K (default)",
        Tiers: []QualityTier{
            {Resolution: "2160p", Sources: []string{"bluray", "webdl"}},
            {Resolution: "1080p", Sources: []string{"remux", "bluray", "webdl", "webrip"}},
            {Resolution: "720p", Sources: []string{"bluray", "webdl"}},
        },
        UpgradeUntil:     &QualityTier{Resolution: "1080p", Source: "bluray"},
        PreferredFormats: []string{"hdr10", "dolby-vision"},
        RejectFormats:    []string{"cam", "telesync", "telecine", "workprint"},
    },
    // ... other presets
}

// DefaultVideoPreset is used when no preset is specified
const DefaultVideoPreset = "balanced"

// GetVideoPreset returns a preset by name
func GetVideoPreset(name string) (VideoQualityPreset, bool) {
    preset, ok := VideoPresets[name]
    return preset, ok
}
```

---

## 2. Audio Quality Presets

For Lidarr.

### 2.1 Preset Definitions

| Preset | Description | Target Users |
|--------|-------------|--------------|
| `lossless-hires` | 24-bit lossless preferred | Audiophiles, hi-fi |
| `lossless` | 16-bit lossless (FLAC, ALAC) | Quality-focused |
| `high-quality` | 320kbps lossy or lossless | Good quality, reasonable size |
| `balanced` | 256kbps+ lossy or lossless | **Default preset** |
| `portable` | 192-256kbps for mobile | Phone/portable |
| `any` | Accept anything | Availability-focused |

### 2.2 Preset Expansions

#### `lossless-hires`

```yaml
tiers:
  - "lossless-hires"  # FLAC 24bit, ALAC 24bit
  - "lossless"        # FLAC, ALAC, APE, WavPack
upgradeUntil: "lossless-hires"
preferredFormats:
  - "flac"
  - "alac"
rejectTiers:
  - "lossy-poor"
  - "lossy-trash"
```

#### `lossless`

```yaml
tiers:
  - "lossless"        # FLAC, ALAC, APE, WavPack
  - "lossless-hires"  # Also accept 24-bit
  - "lossy-high"      # Fallback to 320kbps
upgradeUntil: "lossless"
preferredFormats:
  - "flac"
rejectTiers:
  - "lossy-poor"
  - "lossy-trash"
```

#### `high-quality`

```yaml
tiers:
  - "lossless"
  - "lossy-high"      # MP3-320, AAC-320, Vorbis Q9-10
upgradeUntil: "lossless"
preferredFormats:
  - "flac"
  - "mp3-320"
rejectTiers:
  - "lossy-low"
  - "lossy-poor"
  - "lossy-trash"
```

#### `balanced` (Default)

```yaml
tiers:
  - "lossless"
  - "lossy-high"
  - "lossy-mid"       # MP3-256, AAC-256
upgradeUntil: "lossy-high"
rejectTiers:
  - "lossy-poor"
  - "lossy-trash"
```

#### `portable`

```yaml
tiers:
  - "lossy-high"
  - "lossy-mid"
  - "lossy-low"       # MP3-192
upgradeUntil: "lossy-high"
preferredFormats:
  - "aac"             # Better quality at same bitrate
  - "mp3-320"
rejectTiers:
  - "lossy-trash"
  - "lossless-raw"    # WAV too large for mobile
```

#### `any`

```yaml
tiers:
  - "lossless-hires"
  - "lossless"
  - "lossy-high"
  - "lossy-mid"
  - "lossy-low"
  - "lossy-poor"
upgradeUntil: "lossless"
# No rejections
```

### 2.3 Go Implementation

```go
// internal/presets/audio.go

package presets

// AudioQualityPreset defines an audio quality preset
type AudioQualityPreset struct {
    Name             string
    Description      string
    Tiers            []string  // Tier names: lossless-hires, lossless, lossy-high, etc.
    UpgradeUntil     string
    PreferredFormats []string
    RejectTiers      []string
}

// AudioPresets contains all built-in audio presets
var AudioPresets = map[string]AudioQualityPreset{
    "lossless-hires": {
        Name:             "lossless-hires",
        Description:      "24-bit lossless preferred",
        Tiers:            []string{"lossless-hires", "lossless"},
        UpgradeUntil:     "lossless-hires",
        PreferredFormats: []string{"flac", "alac"},
        RejectTiers:      []string{"lossy-poor", "lossy-trash"},
    },
    "balanced": {
        Name:         "balanced",
        Description:  "256kbps+ lossy or lossless (default)",
        Tiers:        []string{"lossless", "lossy-high", "lossy-mid"},
        UpgradeUntil: "lossy-high",
        RejectTiers:  []string{"lossy-poor", "lossy-trash"},
    },
    // ... other presets
}

// DefaultAudioPreset is used when no preset is specified
const DefaultAudioPreset = "balanced"
```

---

## 3. Naming Presets

For all apps (with app-specific expansions).

### 3.1 Preset Definitions

| Preset | Description | Example Output |
|--------|-------------|----------------|
| `plex-friendly` | Optimized for Plex metadata matching | `Movie Name (2024)/Movie Name (2024).mkv` |
| `jellyfin-friendly` | Optimized for Jellyfin | Same as plex-friendly |
| `kodi-friendly` | Optimized for Kodi | Similar, with NFO support naming |
| `detailed` | Maximum info in filename | `Movie (2024) [Bluray-1080p][DTS-HD MA 5.1][HDR10].mkv` |
| `minimal` | Clean, simple names | `Movie (2024).mkv` |
| `scene` | Scene-style naming | `Movie.2024.1080p.BluRay.x264-GROUP.mkv` |

### 3.2 Radarr Naming Expansion

#### `plex-friendly` / `jellyfin-friendly`

```yaml
renameMovies: true
replaceIllegalCharacters: true
colonReplacement: "smart"
standardMovieFormat: "{Movie CleanTitle} ({Release Year}) - {Quality Full}"
movieFolderFormat: "{Movie CleanTitle} ({Release Year})"
```

#### `detailed`

```yaml
renameMovies: true
replaceIllegalCharacters: true
colonReplacement: "smart"
standardMovieFormat: "{Movie CleanTitle} ({Release Year}) [{Quality Full}]{[MediaInfo AudioCodec]}{[MediaInfo AudioChannels]}{[MediaInfo VideoCodec]}{[MediaInfo VideoDynamicRange]}{-Release Group}"
movieFolderFormat: "{Movie CleanTitle} ({Release Year})"
```

#### `minimal`

```yaml
renameMovies: true
replaceIllegalCharacters: true
colonReplacement: "delete"
standardMovieFormat: "{Movie CleanTitle} ({Release Year})"
movieFolderFormat: "{Movie CleanTitle} ({Release Year})"
```

### 3.3 Sonarr Naming Expansion

#### `plex-friendly`

```yaml
renameEpisodes: true
replaceIllegalCharacters: true
colonReplacement: 4  # Smart
standardEpisodeFormat: "{Series Title} - S{season:00}E{episode:00} - {Episode Title} {Quality Full}"
dailyEpisodeFormat: "{Series Title} - {Air-Date} - {Episode Title} {Quality Full}"
animeEpisodeFormat: "{Series Title} - S{season:00}E{episode:00} - {Episode Title} {Quality Full}"
seriesFolderFormat: "{Series Title}"
seasonFolderFormat: "Season {season:00}"
specialsFolderFormat: "Specials"
multiEpisodeStyle: 5  # Prefixed Range
```

### 3.4 Lidarr Naming Expansion

#### `plex-friendly`

```yaml
renameTracks: true
replaceIllegalCharacters: true
colonReplacement: 4  # Smart
standardTrackFormat: "{Album Artist} - {Album Title} - {track:00} - {Track Title}"
multiDiscTrackFormat: "{Album Artist} - {Album Title} - {medium:0}{track:00} - {Track Title}"
artistFolderFormat: "{Artist Name}"
albumFolderFormat: "{Album Title} ({Release Year})"
```

### 3.5 Go Implementation

```go
// internal/presets/naming.go

package presets

// NamingPreset defines a naming preset
type NamingPreset struct {
    Name        string
    Description string
    // App-specific expansions stored separately
}

// RadarrNamingExpansion expands a preset for Radarr
type RadarrNamingExpansion struct {
    RenameMovies             bool
    ReplaceIllegalCharacters bool
    ColonReplacement         string
    StandardMovieFormat      string
    MovieFolderFormat        string
}

// NamingPresets contains all built-in naming presets
var NamingPresets = map[string]NamingPreset{
    "plex-friendly":     {Name: "plex-friendly", Description: "Optimized for Plex"},
    "jellyfin-friendly": {Name: "jellyfin-friendly", Description: "Optimized for Jellyfin"},
    "detailed":          {Name: "detailed", Description: "Maximum info in filename"},
    "minimal":           {Name: "minimal", Description: "Clean, simple names"},
}

// DefaultNamingPreset is used when no preset is specified
const DefaultNamingPreset = "plex-friendly"

// GetRadarrNaming returns the Radarr expansion for a preset
func GetRadarrNaming(presetName string) RadarrNamingExpansion {
    switch presetName {
    case "plex-friendly", "jellyfin-friendly":
        return RadarrNamingExpansion{
            RenameMovies:             true,
            ReplaceIllegalCharacters: true,
            ColonReplacement:         "smart",
            StandardMovieFormat:      "{Movie CleanTitle} ({Release Year}) - {Quality Full}",
            MovieFolderFormat:        "{Movie CleanTitle} ({Release Year})",
        }
    case "detailed":
        return RadarrNamingExpansion{
            RenameMovies:             true,
            ReplaceIllegalCharacters: true,
            ColonReplacement:         "smart",
            StandardMovieFormat:      "{Movie CleanTitle} ({Release Year}) [{Quality Full}]{[MediaInfo AudioCodec]}{[MediaInfo AudioChannels]}{[MediaInfo VideoCodec]}{[MediaInfo VideoDynamicRange]}{-Release Group}",
            MovieFolderFormat:        "{Movie CleanTitle} ({Release Year})",
        }
    case "minimal":
        return RadarrNamingExpansion{
            RenameMovies:             true,
            ReplaceIllegalCharacters: true,
            ColonReplacement:         "delete",
            StandardMovieFormat:      "{Movie CleanTitle} ({Release Year})",
            MovieFolderFormat:        "{Movie CleanTitle} ({Release Year})",
        }
    default:
        return GetRadarrNaming("plex-friendly")
    }
}
```

---

## 4. Custom Presets via QualityTemplate

Users can define reusable quality configurations:

```yaml
apiVersion: arr.rinzler.cloud/v1alpha1
kind: QualityTemplate
metadata:
  name: household-4k
  namespace: media
spec:
  # For video apps (Radarr/Sonarr)
  video:
    tiers:
      - resolution: "2160p"
        sources: ["bluray", "webdl"]
      - resolution: "1080p"
        sources: ["bluray", "webdl"]
    upgradeUntil:
      resolution: "2160p"
      source: "bluray"
    preferredFormats: ["hdr10", "atmos"]
    rejectFormats: ["3d", "cam"]
  
  # For audio apps (Lidarr)
  audio:
    tiers: ["lossless", "lossy-high"]
    upgradeUntil: "lossless"
    preferredFormats: ["flac"]
```

Reference in configs:
```yaml
kind: RadarrConfig
spec:
  quality:
    templateRef:
      name: household-4k
    # Overrides still apply
    exclude: ["dolby-vision"]
```

---

## 5. Override Syntax

### 5.1 Exclude

Remove items from preset:

```yaml
quality:
  preset: "4k-hdr"
  exclude:
    - "dolby-vision"  # I don't have DV display
    - "3d"            # Already in preset, but for clarity
```

### 5.2 Prefer Additional

Add to preferred formats:

```yaml
quality:
  preset: "4k-hdr"
  preferAdditional:
    - "imax"
    - "extended"
```

### 5.3 Reject Additional

Add to rejected formats:

```yaml
quality:
  preset: "balanced"
  rejectAdditional:
    - "dubbed"
    - "hdtv"  # I only want disc/web sources
```

### 5.4 Combined Example

```yaml
quality:
  preset: "4k-hdr"
  exclude:
    - "dolby-vision"
    - "dts-x"
  preferAdditional:
    - "imax"
  rejectAdditional:
    - "dubbed"
```

### 5.5 Go Implementation

```go
// internal/presets/override.go

package presets

// QualityOverrides defines modifications to a preset
type QualityOverrides struct {
    Exclude          []string `json:"exclude,omitempty"`
    PreferAdditional []string `json:"preferAdditional,omitempty"`
    RejectAdditional []string `json:"rejectAdditional,omitempty"`
}

// ApplyVideoOverrides applies overrides to a video preset
func ApplyVideoOverrides(preset VideoQualityPreset, overrides QualityOverrides) VideoQualityPreset {
    result := preset
    
    // Remove excluded formats from preferred
    result.PreferredFormats = removeItems(result.PreferredFormats, overrides.Exclude)
    
    // Remove excluded formats from reject (in case user wants to un-reject)
    result.RejectFormats = removeItems(result.RejectFormats, overrides.Exclude)
    
    // Add additional preferred formats
    result.PreferredFormats = append(result.PreferredFormats, overrides.PreferAdditional...)
    
    // Add additional rejected formats
    result.RejectFormats = append(result.RejectFormats, overrides.RejectAdditional...)
    
    return result
}

func removeItems(slice []string, toRemove []string) []string {
    removeSet := make(map[string]bool)
    for _, item := range toRemove {
        removeSet[item] = true
    }
    
    result := make([]string, 0, len(slice))
    for _, item := range slice {
        if !removeSet[item] {
            result = append(result, item)
        }
    }
    return result
}
```

---

## 6. Preset Versioning & Stability

### 6.1 Expansion Behavior

Presets are **expanded at reconciliation time**, not at CRD creation. This means:

1. If you create a `RadarrConfig` with `preset: "4k-hdr"` today
2. And nebularr is upgraded to a version with a modified `4k-hdr` preset
3. Your config will use the **new preset definition** on next reconciliation

### 6.2 Implications

| Scenario | Behavior |
|----------|----------|
| Nebularr upgrade changes preset | Config updated on next reconcile |
| User wants pinned quality settings | Use manual `tiers:` instead of `preset:` |
| User wants preset + minor tweaks | Use `preset:` with `exclude:`/`preferAdditional:` |

### 6.3 Stability Guarantees

| Guarantee | Description |
|-----------|-------------|
| Preset names stable | Preset names (e.g., `4k-hdr`) won't be removed or renamed |
| Semantic intent preserved | `4k-hdr` will always target 4K HDR content |
| Details may change | Specific formats, regex patterns may be updated |

### 6.4 Pinning Quality Settings

If you need deterministic, version-independent quality settings, use manual configuration:

```yaml
# Instead of:
quality:
  preset: "4k-hdr"

# Use explicit tiers:
quality:
  tiers:
    - resolution: "2160p"
      sources: ["remux", "bluray", "webdl"]
    - resolution: "1080p"
      sources: ["remux", "bluray", "webdl"]
  preferredFormats: ["hdr10", "dolby-vision", "atmos"]
  rejectFormats: ["cam", "telesync"]
```

This ensures your quality settings remain unchanged across nebularr upgrades.

---

## 7. Related Documents

- [CRDS](./CRDS.md) - CRD definitions using presets
- [TYPES](./TYPES.md) - IR types including preset expansion
- [RADARR](./RADARR.md) - Radarr format/quality mappings
- [SONARR](./SONARR.md) - Sonarr format/quality mappings
- [LIDARR](./LIDARR.md) - Lidarr quality tier mappings
