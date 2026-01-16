# Nebularr - Lidarr API Mapping Reference

> **For coding agents:** Start with [README.md](./README.md) for build order. This document contains Lidarr adapter code to copy.
>
> **Related:** [README](./README.md) | [TYPES](./TYPES.md) | [CRDS](./CRDS.md) | [RADARR](./RADARR.md) | [SONARR](./SONARR.md)

This document is a reference for implementing the Lidarr adapter. Lidarr manages music libraries and has a fundamentally different quality model than video-based apps.

---

## Key Differences from Radarr/Sonarr

| Feature | Lidarr | Radarr/Sonarr |
|---------|--------|---------------|
| **API Version** | v1 | v3 |
| **Quality Model** | Audio format + bitrate | Resolution + source |
| **Profiles** | Quality + Metadata profiles | Quality profiles only |
| **Release Filtering** | Release Profiles (required/ignored terms) | Custom Formats (regex specs) |
| **Root Folders** | Require default profile IDs | Simple path only |
| **Root Folder Updates** | Supported | Not supported |

---

## 1. Quality Tier Mapping

### 1.1 Audio Format + Bitrate -> Lidarr Quality ID

Lidarr uses audio codec and bitrate instead of video resolution:

| Quality Tier | Lidarr ID | Lidarr Name | Group |
|--------------|-----------|-------------|-------|
| `lossless-hires` | 21 | FLAC 24bit | Lossless |
| `lossless-hires` | 37 | ALAC 24bit | Lossless |
| `lossless` | 6 | FLAC | Lossless |
| `lossless` | 7 | ALAC | Lossless |
| `lossless` | 35 | APE | Lossless |
| `lossless` | 36 | WavPack | Lossless |
| `lossless-raw` | 13 | WAV | Uncompressed |
| `lossy-high` | 4 | MP3-320 | High Quality Lossy |
| `lossy-high` | 2 | MP3-VBR-V0 | High Quality Lossy |
| `lossy-high` | 11 | AAC-320 | High Quality Lossy |
| `lossy-high` | 12 | AAC-VBR | High Quality Lossy |
| `lossy-high` | 14 | OGG Vorbis Q10 | High Quality Lossy |
| `lossy-high` | 15 | OGG Vorbis Q9 | High Quality Lossy |
| `lossy-mid` | 3 | MP3-256 | Mid Quality Lossy |
| `lossy-mid` | 8 | MP3-VBR-V2 | Mid Quality Lossy |
| `lossy-mid` | 10 | AAC-256 | Mid Quality Lossy |
| `lossy-mid` | 16 | OGG Vorbis Q8 | Mid Quality Lossy |
| `lossy-mid` | 17 | OGG Vorbis Q7 | Mid Quality Lossy |
| `lossy-low` | 1 | MP3-192 | Low Quality Lossy |
| `lossy-low` | 9 | AAC-192 | Low Quality Lossy |
| `lossy-low` | 18 | OGG Vorbis Q6 | Low Quality Lossy |
| `lossy-low` | 20 | WMA | Low Quality Lossy |
| `lossy-poor` | 5 | MP3-160 | Poor Quality Lossy |
| `lossy-poor` | 22 | MP3-128 | Poor Quality Lossy |
| `lossy-poor` | 19 | OGG Vorbis Q5 | Poor Quality Lossy |
| `lossy-trash` | 23-32 | MP3-96 to MP3-8 | Trash Quality Lossy |
| `unknown` | 0 | Unknown | Unknown |

### 1.2 Our Abstract Quality Tiers

For Lidarr, we use a different abstraction than video:

| Our Tier | Description | Typical Formats |
|----------|-------------|-----------------|
| `lossless-hires` | Hi-res lossless (24-bit) | FLAC 24bit, ALAC 24bit |
| `lossless` | Standard lossless (16-bit) | FLAC, ALAC, APE, WavPack |
| `lossless-raw` | Uncompressed | WAV |
| `lossy-high` | High quality lossy (320kbps+) | MP3-320, AAC-320, Vorbis Q9-10 |
| `lossy-mid` | Mid quality lossy (256kbps) | MP3-256, AAC-256, Vorbis Q7-8 |
| `lossy-low` | Low quality lossy (192kbps) | MP3-192, AAC-192, Vorbis Q6 |
| `lossy-poor` | Poor quality lossy (128-160kbps) | MP3-128, MP3-160, Vorbis Q5 |

### 1.3 Go Implementation

```go
// internal/adapters/lidarr/mapping.go

package lidarr

// AudioQualityTier represents our abstract audio quality tier
type AudioQualityTier string

const (
    TierLosslessHires AudioQualityTier = "lossless-hires"
    TierLossless      AudioQualityTier = "lossless"
    TierLosslessRaw   AudioQualityTier = "lossless-raw"
    TierLossyHigh     AudioQualityTier = "lossy-high"
    TierLossyMid      AudioQualityTier = "lossy-mid"
    TierLossyLow      AudioQualityTier = "lossy-low"
    TierLossyPoor     AudioQualityTier = "lossy-poor"
    TierUnknown       AudioQualityTier = "unknown"
)

// QualityMapping maps our abstract tiers to Lidarr quality IDs
// Multiple IDs can map to the same tier (e.g., FLAC and ALAC are both "lossless")
var QualityMapping = map[AudioQualityTier][]int{
    TierLosslessHires: {21, 37},           // FLAC 24bit, ALAC 24bit
    TierLossless:      {6, 7, 35, 36},     // FLAC, ALAC, APE, WavPack
    TierLosslessRaw:   {13},               // WAV
    TierLossyHigh:     {4, 2, 11, 12, 14, 15}, // MP3-320, MP3-VBR-V0, AAC-320, AAC-VBR, Vorbis Q10, Q9
    TierLossyMid:      {3, 8, 10, 16, 17}, // MP3-256, MP3-VBR-V2, AAC-256, Vorbis Q8, Q7
    TierLossyLow:      {1, 9, 18, 20},     // MP3-192, AAC-192, Vorbis Q6, WMA
    TierLossyPoor:     {5, 22, 19},        // MP3-160, MP3-128, Vorbis Q5
    TierUnknown:       {0},                // Unknown
}

// ReverseQualityMapping maps Lidarr quality ID to our tier
var ReverseQualityMapping = func() map[int]AudioQualityTier {
    m := make(map[int]AudioQualityTier)
    for tier, ids := range QualityMapping {
        for _, id := range ids {
            m[id] = tier
        }
    }
    return m
}()

// GetQualityIDs returns all Lidarr quality IDs for a tier
func GetQualityIDs(tier AudioQualityTier) []int {
    ids, ok := QualityMapping[tier]
    if !ok {
        return nil
    }
    return ids
}

// GetQualityTier returns our tier for a Lidarr quality ID
func GetQualityTier(id int) AudioQualityTier {
    tier, ok := ReverseQualityMapping[id]
    if !ok {
        return TierUnknown
    }
    return tier
}
```

---

## 2. Metadata Profile Mapping

### 2.1 Primary Album Types

Metadata profiles control which album types are allowed:

| Type | ID | Description |
|------|-----|-------------|
| Album | 0 | Standard studio album |
| EP | 1 | Extended Play |
| Single | 2 | Single release |
| Broadcast | 3 | Radio broadcast |
| Other | 4 | Other release types |

### 2.2 Secondary Album Types

| Type | ID | Description |
|------|-----|-------------|
| Studio | 0 | Studio recording |
| Compilation | 1 | Compilation album |
| Soundtrack | 2 | Film/TV soundtrack |
| Spokenword | 3 | Spoken word/audiobook |
| Interview | 4 | Interview recording |
| Live | 5 | Live performance |
| Remix | 6 | Remix album |
| DJ-mix | 7 | DJ mix |
| Mixtape/Street | 8 | Mixtape |
| Demo | 9 | Demo recording |

### 2.3 Release Statuses

| Status | ID | Description |
|--------|-----|-------------|
| Official | 0 | Official release |
| Promotional | 1 | Promotional release |
| Bootleg | 2 | Unofficial bootleg |
| Pseudo-Release | 3 | Pseudo-release |

### 2.4 Go Implementation

```go
// internal/adapters/lidarr/metadata.go

package lidarr

// PrimaryAlbumType represents album type
type PrimaryAlbumType int

const (
    AlbumTypeAlbum     PrimaryAlbumType = 0
    AlbumTypeEP        PrimaryAlbumType = 1
    AlbumTypeSingle    PrimaryAlbumType = 2
    AlbumTypeBroadcast PrimaryAlbumType = 3
    AlbumTypeOther     PrimaryAlbumType = 4
)

// SecondaryAlbumType represents secondary album classification
type SecondaryAlbumType int

const (
    SecondaryStudio       SecondaryAlbumType = 0
    SecondaryCompilation  SecondaryAlbumType = 1
    SecondarySoundtrack   SecondaryAlbumType = 2
    SecondarySpokenword   SecondaryAlbumType = 3
    SecondaryInterview    SecondaryAlbumType = 4
    SecondaryLive         SecondaryAlbumType = 5
    SecondaryRemix        SecondaryAlbumType = 6
    SecondaryDJMix        SecondaryAlbumType = 7
    SecondaryMixtape      SecondaryAlbumType = 8
    SecondaryDemo         SecondaryAlbumType = 9
)

// ReleaseStatus represents release status
type ReleaseStatus int

const (
    ReleaseOfficial      ReleaseStatus = 0
    ReleasePromotional   ReleaseStatus = 1
    ReleaseBootleg       ReleaseStatus = 2
    ReleasePseudoRelease ReleaseStatus = 3
)

// MetadataProfileConfig represents our abstract metadata profile
type MetadataProfileConfig struct {
    Name                  string
    PrimaryAlbumTypes     []PrimaryAlbumType
    SecondaryAlbumTypes   []SecondaryAlbumType
    ReleaseStatuses       []ReleaseStatus
}

// DefaultMetadataProfile returns a sensible default
func DefaultMetadataProfile() MetadataProfileConfig {
    return MetadataProfileConfig{
        Name: "Standard",
        PrimaryAlbumTypes: []PrimaryAlbumType{
            AlbumTypeAlbum,
            AlbumTypeEP,
        },
        SecondaryAlbumTypes: []SecondaryAlbumType{
            SecondaryStudio,
        },
        ReleaseStatuses: []ReleaseStatus{
            ReleaseOfficial,
        },
    }
}

// BuildMetadataProfilePayload creates Lidarr metadata profile from our config
func BuildMetadataProfilePayload(cfg MetadataProfileConfig) map[string]interface{} {
    primaryTypes := make([]map[string]interface{}, len(cfg.PrimaryAlbumTypes))
    for i, t := range cfg.PrimaryAlbumTypes {
        primaryTypes[i] = map[string]interface{}{
            "primaryAlbumType": map[string]interface{}{"id": int(t)},
            "allowed":          true,
        }
    }
    
    secondaryTypes := make([]map[string]interface{}, len(cfg.SecondaryAlbumTypes))
    for i, t := range cfg.SecondaryAlbumTypes {
        secondaryTypes[i] = map[string]interface{}{
            "secondaryAlbumType": map[string]interface{}{"id": int(t)},
            "allowed":            true,
        }
    }
    
    releaseStatuses := make([]map[string]interface{}, len(cfg.ReleaseStatuses))
    for i, s := range cfg.ReleaseStatuses {
        releaseStatuses[i] = map[string]interface{}{
            "releaseStatus": map[string]interface{}{"id": int(s)},
            "allowed":       true,
        }
    }
    
    return map[string]interface{}{
        "name":                cfg.Name,
        "primaryAlbumTypes":   primaryTypes,
        "secondaryAlbumTypes": secondaryTypes,
        "releaseStatuses":     releaseStatuses,
    }
}
```

---

## 3. Release Profile Mapping

### 3.1 Release Profile Structure

Lidarr uses Release Profiles instead of Custom Formats. They're simpler:

| Field | Type | Description |
|-------|------|-------------|
| `enabled` | bool | Profile is active |
| `required` | []string | Terms that MUST appear in release |
| `ignored` | []string | Terms that MUST NOT appear in release |
| `indexerId` | int | Limit to specific indexer (0 = all) |
| `tags` | []int | Associated tags |

### 3.2 Common Release Profile Patterns

| Profile Purpose | Required | Ignored |
|-----------------|----------|---------|
| Prefer FLAC | `["FLAC"]` | `[]` |
| Avoid low quality | `[]` | `["128", "96", "64"]` |
| Prefer scene | `["SCENE"]` | `[]` |
| Avoid vinyl rips | `[]` | `["vinyl", "Vinyl"]` |
| 24-bit only | `["24bit", "24-bit"]` | `[]` |

### 3.3 Go Implementation

```go
// internal/adapters/lidarr/releases.go

package lidarr

// ReleaseProfileConfig represents our abstract release profile
type ReleaseProfileConfig struct {
    Name      string   // For our reference (not sent to API)
    Enabled   bool
    Required  []string // Terms that must appear
    Ignored   []string // Terms that must not appear
    IndexerID int      // 0 = all indexers
    Tags      []int
}

// BuildReleaseProfilePayload creates Lidarr release profile from our config
func BuildReleaseProfilePayload(cfg ReleaseProfileConfig) map[string]interface{} {
    return map[string]interface{}{
        "enabled":   cfg.Enabled,
        "required":  cfg.Required,
        "ignored":   cfg.Ignored,
        "indexerId": cfg.IndexerID,
        "tags":      cfg.Tags,
    }
}

// CommonReleaseProfiles provides pre-built profiles
var CommonReleaseProfiles = map[string]ReleaseProfileConfig{
    "prefer-flac": {
        Name:     "Prefer FLAC",
        Enabled:  true,
        Required: []string{"FLAC"},
        Ignored:  []string{},
    },
    "prefer-lossless": {
        Name:     "Prefer Lossless",
        Enabled:  true,
        Required: []string{},
        Ignored:  []string{"MP3", "AAC", "OGG", "WMA", "128", "192", "256", "320"},
    },
    "avoid-low-quality": {
        Name:     "Avoid Low Quality",
        Enabled:  true,
        Required: []string{},
        Ignored:  []string{"128", "96", "64", "48", "32"},
    },
    "prefer-24bit": {
        Name:     "Prefer 24-bit",
        Enabled:  true,
        Required: []string{"24bit", "24-bit", "24 bit"},
        Ignored:  []string{},
    },
}
```

---

## 4. Download Client Mapping

### 4.1 Implementation -> Lidarr Implementation Name

Same implementations as Radarr/Sonarr:

| Our Implementation | Lidarr Implementation |
|--------------------|----------------------|
| `qbittorrent` | `QBittorrent` |
| `transmission` | `Transmission` |
| `deluge` | `Deluge` |
| `rtorrent` | `RTorrent` |
| `nzbget` | `Nzbget` |
| `sabnzbd` | `Sabnzbd` |

### 4.2 Lidarr-Specific Field: Category

| App | Category Field |
|-----|----------------|
| Radarr | `movieCategory` |
| Sonarr | `tvCategory` |
| Lidarr | `musicCategory` |

### 4.3 Go Implementation

```go
// internal/adapters/lidarr/clients.go

package lidarr

import irv1 "github.com/poiley/nebularr/internal/ir/v1"

// ImplementationMapping maps our abstract client names to Lidarr implementation names
var ImplementationMapping = map[string]string{
    "qbittorrent":  "QBittorrent",
    "transmission": "Transmission",
    "deluge":       "Deluge",
    "rtorrent":     "RTorrent",
    "nzbget":       "Nzbget",
    "sabnzbd":      "Sabnzbd",
}

// BuildDownloadClientPayload creates a Lidarr download client from IR
func BuildDownloadClientPayload(ir *irv1.DownloadClientIR) map[string]interface{} {
    implementation := ImplementationMapping[ir.Implementation]
    
    payload := map[string]interface{}{
        "name":           ir.Name,
        "implementation": implementation,
        "configContract": implementation + "Settings",
        "enable":         ir.Enable,
        "protocol":       ir.Protocol,
        "priority":       ir.Priority,
        "removeCompletedDownloads": ir.RemoveCompletedDownloads,
        "removeFailedDownloads":    ir.RemoveFailedDownloads,
        "fields": []map[string]interface{}{
            {"name": "host", "value": ir.Host},
            {"name": "port", "value": ir.Port},
            {"name": "useSsl", "value": ir.UseTLS},
            {"name": "username", "value": ir.Username},
            {"name": "password", "value": ir.Password},
            // NOTE: Lidarr uses "musicCategory"
            {"name": "musicCategory", "value": ir.Category},
        },
    }
    
    payload["tags"] = []int{}
    
    return payload
}
```

---

## 5. Indexer Mapping

### 5.1 Category Mapping (Music Categories)

Lidarr uses music-specific Newznab category IDs:

| Category Name | Newznab ID | Description |
|---------------|------------|-------------|
| Audio | 3000 | All audio |
| Audio/MP3 | 3010 | MP3 format |
| Audio/Video | 3020 | Music videos |
| Audio/Audiobook | 3030 | Audiobooks |
| Audio/Lossless | 3040 | Lossless audio |
| Audio/Other | 3050 | Other audio |
| Audio/Foreign | 3060 | Foreign audio |

### 5.2 Go Implementation

```go
// internal/adapters/lidarr/categories.go

package lidarr

import (
    "log/slog"
    "strconv"
    "strings"
)

// CategoryMapping maps user-friendly category names to Newznab IDs
var CategoryMapping = map[string]int{
    // General
    "audio":     3000,
    "audio-all": 3000,
    "music":     3000,
    
    // Specific categories
    "audio-mp3":       3010,
    "audio-video":     3020,
    "audio-audiobook": 3030,
    "audio-lossless":  3040,
    "audio-flac":      3040,
    "audio-other":     3050,
    "audio-foreign":   3060,
    
    // Common aliases
    "mp3":       3010,
    "lossless":  3040,
    "flac":      3040,
    "audiobook": 3030,
}

// MapCategories converts user-friendly category names to Newznab IDs
func MapCategories(categories []string) []int {
    ids := make([]int, 0, len(categories))
    for _, cat := range categories {
        if id, ok := CategoryMapping[strings.ToLower(cat)]; ok {
            ids = append(ids, id)
        } else {
            if rawID, err := strconv.Atoi(cat); err == nil {
                ids = append(ids, rawID)
            } else {
                slog.Warn("unknown category, skipping", "category", cat)
            }
        }
    }
    return ids
}
```

---

## 6. Monitor Type Mapping

### 6.1 Monitor Options

Lidarr has album-specific monitoring options:

| Monitor Option | API Value | Description |
|----------------|-----------|-------------|
| All Albums | `all` | Monitor all albums |
| Future Albums | `future` | Only future releases |
| Missing Albums | `missing` | Only missing albums |
| Existing Albums | `existing` | Only albums that exist |
| Latest Album | `latest` | Most recent album only |
| First Album | `first` | First album only |
| None | `none` | Don't monitor any |

### 6.2 Go Implementation

```go
// internal/adapters/lidarr/monitor.go

package lidarr

// MonitorType represents how to monitor an artist
type MonitorType string

const (
    MonitorAll      MonitorType = "all"
    MonitorFuture   MonitorType = "future"
    MonitorMissing  MonitorType = "missing"
    MonitorExisting MonitorType = "existing"
    MonitorLatest   MonitorType = "latest"
    MonitorFirst    MonitorType = "first"
    MonitorNone     MonitorType = "none"
)

// ValidMonitorTypes for validation
var ValidMonitorTypes = map[MonitorType]bool{
    MonitorAll:      true,
    MonitorFuture:   true,
    MonitorMissing:  true,
    MonitorExisting: true,
    MonitorLatest:   true,
    MonitorFirst:    true,
    MonitorNone:     true,
}

// DefaultMonitorType returns the default monitor type
func DefaultMonitorType() MonitorType {
    return MonitorAll
}
```

---

## 7. Root Folder Mapping

### 7.1 Lidarr Root Folder Requirements

Unlike Radarr/Sonarr, Lidarr root folders require default profile IDs:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `path` | string | Yes | Folder path |
| `name` | string | Yes | Display name |
| `defaultMetadataProfileId` | int | Yes | Default metadata profile |
| `defaultQualityProfileId` | int | Yes | Default quality profile |
| `defaultMonitorOption` | string | No | Default monitor option |

### 7.2 Go Implementation

```go
// internal/adapters/lidarr/rootfolders.go

package lidarr

// RootFolderConfig represents our abstract root folder config
type RootFolderConfig struct {
    Path                     string
    Name                     string
    DefaultMetadataProfileID int
    DefaultQualityProfileID  int
    DefaultMonitorOption     MonitorType
}

// BuildRootFolderPayload creates Lidarr root folder from our config
func BuildRootFolderPayload(cfg RootFolderConfig) map[string]interface{} {
    payload := map[string]interface{}{
        "path":                     cfg.Path,
        "name":                     cfg.Name,
        "defaultMetadataProfileId": cfg.DefaultMetadataProfileID,
        "defaultQualityProfileId":  cfg.DefaultQualityProfileID,
    }
    
    if cfg.DefaultMonitorOption != "" {
        payload["defaultMonitorOption"] = string(cfg.DefaultMonitorOption)
    }
    
    return payload
}
```

---

## 8. Naming Configuration

### 8.1 Lidarr Naming Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `renameTracks` | bool | false | Enable track renaming |
| `replaceIllegalCharacters` | bool | true | Replace illegal characters |
| `colonReplacementFormat` | int | 5 | How to replace colons |
| `standardTrackFormat` | string | varies | Format for standard tracks |
| `multiDiscTrackFormat` | string | varies | Format for multi-disc |
| `artistFolderFormat` | string | varies | Artist folder naming |

### 8.2 Go Implementation

```go
// internal/adapters/lidarr/naming.go

package lidarr

// ColonReplacement values for Lidarr
const (
    ColonDelete         = 0
    ColonDash           = 1
    ColonSpaceDash      = 2
    ColonSpaceDashSpace = 3
    ColonSmart          = 4
    ColonCustom         = 5
)

// DefaultNamingConfig returns default Lidarr naming settings
func DefaultNamingConfig() map[string]interface{} {
    return map[string]interface{}{
        "renameTracks":              false,
        "replaceIllegalCharacters":  true,
        "colonReplacementFormat":    ColonSmart,
        "standardTrackFormat":       "{Artist Name} - {Album Title} - {track:00} - {Track Title}",
        "multiDiscTrackFormat":      "{Artist Name} - {Album Title} - {medium:0}-{track:00} - {Track Title}",
        "artistFolderFormat":        "{Artist Name}",
    }
}

// NamingConfig represents our abstract naming config
type NamingConfig struct {
    RenameTracks              *bool
    ReplaceIllegalCharacters  *bool
    ColonReplacement          *int
    StandardTrackFormat       string
    MultiDiscTrackFormat      string
    ArtistFolderFormat        string
}

// BuildNamingPayload creates Lidarr naming config from our config
func BuildNamingPayload(cfg NamingConfig) map[string]interface{} {
    payload := DefaultNamingConfig()
    
    if cfg.RenameTracks != nil {
        payload["renameTracks"] = *cfg.RenameTracks
    }
    if cfg.ReplaceIllegalCharacters != nil {
        payload["replaceIllegalCharacters"] = *cfg.ReplaceIllegalCharacters
    }
    if cfg.ColonReplacement != nil {
        payload["colonReplacementFormat"] = *cfg.ColonReplacement
    }
    if cfg.StandardTrackFormat != "" {
        payload["standardTrackFormat"] = cfg.StandardTrackFormat
    }
    if cfg.MultiDiscTrackFormat != "" {
        payload["multiDiscTrackFormat"] = cfg.MultiDiscTrackFormat
    }
    if cfg.ArtistFolderFormat != "" {
        payload["artistFolderFormat"] = cfg.ArtistFolderFormat
    }
    
    payload["id"] = 1
    
    return payload
}
```

---

## 9. Capability Discovery Endpoints

### 9.1 API Endpoints for Discovery

**Note:** Lidarr uses API v1, not v3.

| Capability | Endpoint | Response Field |
|------------|----------|----------------|
| Quality definitions | `GET /api/v1/qualitydefinition` | List of available qualities |
| Quality profiles | `GET /api/v1/qualityprofile` | Existing quality profiles |
| Metadata profiles | `GET /api/v1/metadataprofile` | Existing metadata profiles |
| Release profiles | `GET /api/v1/releaseprofile` | Existing release profiles |
| Download client schema | `GET /api/v1/downloadclient/schema` | Available client types |
| Indexer schema | `GET /api/v1/indexer/schema` | Available indexer types |
| Root folders | `GET /api/v1/rootfolder` | Configured root folders |
| System status | `GET /api/v1/system/status` | Version info |

### 9.2 Go Implementation

```go
// internal/adapters/lidarr/adapter.go

package lidarr

import (
    "context"
    "fmt"
    "log/slog"
    "time"
    
    "github.com/poiley/nebularr/internal/adapters"
    "github.com/poiley/nebularr/internal/adapters/lidarr/client"
)

// Adapter implements the adapters.Adapter interface for Lidarr
type Adapter struct {
    client *client.ClientWithResponses
}

// NewAdapter creates a new Lidarr adapter
func NewAdapter() *Adapter {
    return &Adapter{}
}

// Name returns the adapter identifier
func (a *Adapter) Name() string {
    return "lidarr"
}

// SupportedApp returns the app this adapter handles
func (a *Adapter) SupportedApp() string {
    return "lidarr"
}

// APIVersion returns the API version used by Lidarr
func (a *Adapter) APIVersion() string {
    return "v1" // Lidarr uses v1, not v3
}

// Discover queries Lidarr for available features
func (a *Adapter) Discover(ctx context.Context, conn *adapters.Connection) (*adapters.Capabilities, error) {
    caps := &adapters.Capabilities{
        DiscoveredAt: time.Now(),
    }
    
    // Get quality definitions
    qualities, err := a.client.GetQualityDefinitions(ctx)
    if err != nil {
        return nil, fmt.Errorf("failed to get quality definitions: %w", err)
    }
    
    for _, q := range qualities {
        tier := GetQualityTier(q.Quality.ID)
        if tier != TierUnknown {
            caps.AudioQualities = appendUnique(caps.AudioQualities, string(tier))
        }
    }
    
    // Get metadata profiles
    metadataProfiles, err := a.client.GetMetadataProfiles(ctx)
    if err != nil {
        slog.Warn("failed to get metadata profiles", "error", err)
    } else {
        for _, p := range metadataProfiles {
            caps.MetadataProfiles = append(caps.MetadataProfiles, p.Name)
        }
    }
    
    // Get download client types
    clientSchemas, err := a.client.GetDownloadClientSchema(ctx)
    if err != nil {
        slog.Warn("failed to get download client schema", "error", err)
    } else {
        for _, schema := range clientSchemas {
            caps.DownloadClientTypes = append(caps.DownloadClientTypes, schema.Implementation)
        }
    }
    
    return caps, nil
}

// appendUnique appends item to slice only if not already present
func appendUnique(slice []string, item string) []string {
    for _, s := range slice {
        if s == item {
            return slice
        }
    }
    return append(slice, item)
}
```

---

## 10. Extended IR Types for Lidarr

### 10.1 Lidarr-Specific IR Types

```go
// internal/ir/v1/lidarr_types.go

package v1

// LidarrQualityIR represents audio quality configuration
type LidarrQualityIR struct {
    // Tiers to include (lossless, lossy-high, etc.)
    Tiers []string `json:"tiers"`
    
    // Specific formats to prefer (FLAC, MP3-320, etc.)
    PreferredFormats []string `json:"preferredFormats,omitempty"`
    
    // Upgrade until reaching this tier
    UpgradeUntil string `json:"upgradeUntil,omitempty"`
}

// LidarrMetadataProfileIR represents metadata profile configuration
type LidarrMetadataProfileIR struct {
    Name                string   `json:"name"`
    PrimaryAlbumTypes   []string `json:"primaryAlbumTypes"`   // album, ep, single, broadcast, other
    SecondaryAlbumTypes []string `json:"secondaryAlbumTypes"` // studio, compilation, soundtrack, live, etc.
    ReleaseStatuses     []string `json:"releaseStatuses"`     // official, promotional, bootleg
}

// LidarrReleaseProfileIR represents release profile configuration
type LidarrReleaseProfileIR struct {
    Required []string `json:"required,omitempty"`
    Ignored  []string `json:"ignored,omitempty"`
}

// LidarrRootFolderIR represents root folder with required Lidarr fields
type LidarrRootFolderIR struct {
    Path                     string `json:"path"`
    Name                     string `json:"name"`
    DefaultMetadataProfileID int    `json:"defaultMetadataProfileId,omitempty"`
    DefaultQualityProfileID  int    `json:"defaultQualityProfileId,omitempty"`
    DefaultMonitorOption     string `json:"defaultMonitorOption,omitempty"`
}

// LidarrNamingIR represents naming configuration
type LidarrNamingIR struct {
    RenameTracks              *bool  `json:"renameTracks,omitempty"`
    ReplaceIllegalCharacters  *bool  `json:"replaceIllegalCharacters,omitempty"`
    ColonReplacement          *int   `json:"colonReplacement,omitempty"`
    StandardTrackFormat       string `json:"standardTrackFormat,omitempty"`
    MultiDiscTrackFormat      string `json:"multiDiscTrackFormat,omitempty"`
    ArtistFolderFormat        string `json:"artistFolderFormat,omitempty"`
}
```

---

## 11. CRD Extensions for Lidarr

### 11.1 LidarrMusicPolicy CRD

Since Lidarr's quality model is fundamentally different from video (audio format + bitrate instead of resolution + source), it uses a dedicated `LidarrMusicPolicy` CRD instead of the `MediaPolicy` used by Radarr/Sonarr.

See [CRDS.md Section 4.3](./CRDS.md) for the authoritative CRD definition.

```go
// api/v1alpha1/lidarrmusicpolicy_types.go

// LidarrMusicPolicySpec defines audio quality preferences for Lidarr
type LidarrMusicPolicySpec struct {
    // Quality tier configuration
    Quality LidarrMusicQualitySpec `json:"quality"`
    
    // Metadata profile configuration
    Metadata *MetadataProfileSpec `json:"metadata,omitempty"`
    
    // Release profile configuration
    ReleaseProfiles []ReleaseProfileSpec `json:"releaseProfiles,omitempty"`
}

type LidarrMusicQualitySpec struct {
    // Tiers to allow: lossless-hires, lossless, lossy-high, lossy-mid, lossy-low
    Tiers []string `json:"tiers"`
    
    // Upgrade until reaching this tier
    UpgradeUntil string `json:"upgradeUntil,omitempty"`
    
    // Preferred formats within tiers (e.g., prefer FLAC over ALAC)
    PreferredFormats []string `json:"preferredFormats,omitempty"`
}

type MetadataProfileSpec struct {
    // Primary album types: album, ep, single
    PrimaryTypes []string `json:"primaryTypes"`
    
    // Secondary album types: studio, compilation, soundtrack, live
    SecondaryTypes []string `json:"secondaryTypes,omitempty"`
    
    // Release statuses: official, promotional, bootleg
    ReleaseStatuses []string `json:"releaseStatuses,omitempty"`
}

type ReleaseProfileSpec struct {
    // Terms that must appear in release name
    Required []string `json:"required,omitempty"`
    
    // Terms that must not appear in release name
    Ignored []string `json:"ignored,omitempty"`
}
```

---

## 12. Related Documents

- [README](./README.md) - Build order, file mapping (start here)
- [RADARR](./RADARR.md) - Radarr adapter (video quality model)
- [SONARR](./SONARR.md) - Sonarr adapter (TV quality model)
- [TYPES](./TYPES.md) - IR types and adapter interface
- [CRDS](./CRDS.md) - CRD definitions
- [PRESETS](./PRESETS.md) - Quality and naming presets (audio presets for Lidarr)
- [OPERATIONS](./OPERATIONS.md) - Auto-discovery, secrets, merge rules
