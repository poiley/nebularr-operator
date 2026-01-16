# Nebularr - Sonarr API Mapping Reference

> **For coding agents:** Start with [README.md](./README.md) for build order. This document contains Sonarr adapter code to copy.
>
> **Related:** [README](./README.md) | [TYPES](./TYPES.md) | [CRDS](./CRDS.md) | [RADARR](./RADARR.md)

This document is a reference for implementing the Sonarr adapter. It maps our abstract quality/format concepts to Sonarr's specific API fields and IDs.

---

## 1. Quality Tier Mapping

### 1.1 Resolution + Source -> Sonarr Quality ID

Our `QualityTier{Resolution, Source}` maps to Sonarr's quality IDs:

| Resolution | Source | Sonarr ID | Sonarr Name |
|------------|--------|-----------|-------------|
| 2160p | remux | 21 | Bluray-2160p Remux |
| 2160p | bluray | 19 | Bluray-2160p |
| 2160p | webdl | 18 | WEBDL-2160p |
| 2160p | webrip | 17 | WEBRip-2160p |
| 2160p | hdtv | 16 | HDTV-2160p |
| 1080p | remux | 20 | Bluray-1080p Remux |
| 1080p | bluray | 7 | Bluray-1080p |
| 1080p | webdl | 3 | WEBDL-1080p |
| 1080p | webrip | 15 | WEBRip-1080p |
| 1080p | hdtv | 9 | HDTV-1080p |
| 1080p | raw-hd | 10 | Raw-HD |
| 720p | bluray | 6 | Bluray-720p |
| 720p | webdl | 5 | WEBDL-720p |
| 720p | webrip | 14 | WEBRip-720p |
| 720p | hdtv | 4 | HDTV-720p |
| 576p | bluray | 22 | Bluray-576p |
| 480p | bluray | 13 | Bluray-480p |
| 480p | webdl | 8 | WEBDL-480p |
| 480p | webrip | 12 | WEBRip-480p |
| 480p | dvd | 2 | DVD |
| 480p | sdtv | 1 | SDTV |
| any | unknown | 0 | Unknown |

### 1.2 Key Differences from Radarr

| Feature | Sonarr | Radarr |
|---------|--------|--------|
| Remux IDs | 20 (1080p), 21 (2160p) | 30 (1080p), 31 (2160p) |
| 576p Bluray | Yes (ID 22) | No |
| Raw-HD | Yes (ID 10) | Yes (ID 10) |
| SDTV | Yes (ID 1) | No |
| CAM/TELECINE/TELESYNC | No | Yes |

### 1.3 Go Implementation

```go
// internal/adapters/sonarr/mapping.go

package sonarr

// QualityKey is a lookup key for mapping resolution+source to Sonarr quality IDs
// This is an adapter-internal type, not from IR. The IR uses VideoQualityTierIR
// with Sources as an array; the adapter iterates through sources and maps each.
type QualityKey struct {
    Resolution string
    Source     string
}

// QualityMapping maps resolution+source combinations to Sonarr quality IDs
var QualityMapping = map[QualityKey]int{
    // 2160p
    {Resolution: "2160p", Source: "remux"}:  21,
    {Resolution: "2160p", Source: "bluray"}: 19,
    {Resolution: "2160p", Source: "webdl"}:  18,
    {Resolution: "2160p", Source: "webrip"}: 17,
    {Resolution: "2160p", Source: "hdtv"}:   16,
    
    // 1080p
    {Resolution: "1080p", Source: "remux"}:  20,
    {Resolution: "1080p", Source: "bluray"}: 7,
    {Resolution: "1080p", Source: "webdl"}:  3,
    {Resolution: "1080p", Source: "webrip"}: 15,
    {Resolution: "1080p", Source: "hdtv"}:   9,
    {Resolution: "1080p", Source: "raw-hd"}: 10,
    
    // 720p
    {Resolution: "720p", Source: "bluray"}: 6,
    {Resolution: "720p", Source: "webdl"}:  5,
    {Resolution: "720p", Source: "webrip"}: 14,
    {Resolution: "720p", Source: "hdtv"}:   4,
    
    // 576p (PAL standard - Sonarr specific)
    {Resolution: "576p", Source: "bluray"}: 22,
    
    // 480p
    {Resolution: "480p", Source: "bluray"}: 13,
    {Resolution: "480p", Source: "webdl"}:  8,
    {Resolution: "480p", Source: "webrip"}: 12,
    {Resolution: "480p", Source: "dvd"}:    2,
    {Resolution: "480p", Source: "sdtv"}:   1,
}

// MapQuality converts resolution+source to Sonarr quality ID
// Returns (id, true) if found, (0, false) if not supported
func MapQuality(resolution, source string) (int, bool) {
    id, ok := QualityMapping[QualityKey{Resolution: resolution, Source: source}]
    return id, ok
}

// ReverseQualityMapping for converting Sonarr state back to resolution+source
var ReverseQualityMapping = func() map[int]QualityKey {
    m := make(map[int]QualityKey)
    for key, id := range QualityMapping {
        m[id] = key
    }
    return m
}()
```

---

## 2. Custom Format Specification Mapping

### 2.1 Format -> Sonarr Custom Format Spec

Sonarr uses the same custom format specification system as Radarr. Our abstract format names map to Sonarr custom format specifications:

| Our Format | Spec Type | Value/Pattern |
|------------|-----------|---------------|
| `hdr10` | ReleaseTitleSpecification | `\bHDR10\b(?![\+P])` |
| `hdr10plus` | ReleaseTitleSpecification | `\bHDR10(\+\|Plus)\b` |
| `dolby-vision` | ReleaseTitleSpecification | `\b(DV\|DoVi\|Dolby[\.\s]?Vision)\b` |
| `hlg` | ReleaseTitleSpecification | `\bHLG\b` |
| `atmos` | ReleaseTitleSpecification | `\bAtmos\b` |
| `truehd` | ReleaseTitleSpecification | `\bTrueHD\b` |
| `dts-x` | ReleaseTitleSpecification | `\bDTS[-:\s]?X\b` |
| `dts-hd` | ReleaseTitleSpecification | `\bDTS[-\s]?HD(\s?MA)?\b` |
| `aac` | ReleaseTitleSpecification | `\bAAC\b` |
| `hevc` | ReleaseTitleSpecification | `\b(HEVC\|x265\|H\.?265)\b` |
| `av1` | ReleaseTitleSpecification | `\bAV1\b` |
| `10bit` | ReleaseTitleSpecification | `\b10[-\s]?bit\b` |
| `remux` | QualityModifierSpecification | `5` (REMUX modifier ID) |
| `proper` | ReleaseTitleSpecification | `\bPROPER\b` |
| `repack` | ReleaseTitleSpecification | `\bREPACK\b` |
| `dual-audio` | ReleaseTitleSpecification | `\bDual[\.\s]?Audio\b` |
| `multi-audio` | ReleaseTitleSpecification | `\bMulti\b` |

### 2.2 TV-Specific Custom Formats

Sonarr has additional TV-specific formats that don't apply to movies:

| Our Format | Spec Type | Value/Pattern | Description |
|------------|-----------|---------------|-------------|
| `season-pack` | ReleaseTitleSpecification | `\b(S\d{2}(?!E)\|Season[\.\s]?\d+)\b` | Full season packs |
| `amzn` | ReleaseTitleSpecification | `\bAMZN\b` | Amazon Prime source |
| `nf` | ReleaseTitleSpecification | `\b(NF\|Netflix)\b` | Netflix source |
| `dsnp` | ReleaseTitleSpecification | `\bDSNP\b` | Disney+ source |
| `atvp` | ReleaseTitleSpecification | `\bATVP\b` | Apple TV+ source |
| `hmax` | ReleaseTitleSpecification | `\bHMAX\b` | HBO Max source |
| `pcok` | ReleaseTitleSpecification | `\bPCOK\b` | Peacock source |
| `pmtp` | ReleaseTitleSpecification | `\bPMTP\b` | Paramount+ source |
| `web-scene` | ReleaseTitleSpecification | `\b(WEBDL\|WEB-DL\|WEB[\.\s]?DL)\b` | Scene WEB-DL |
| `web-p2p` | ReleaseTitleSpecification | `\b(WEBRip\|WEB-Rip\|WEB[\.\s]?Rip)\b` | P2P WEBRip |
| `anime-dual` | ReleaseTitleSpecification | `\b(Dual[\.\s]?Audio\|JPN?\+ENG?)\b` | Anime dual audio |
| `anime-uncensored` | ReleaseTitleSpecification | `\bUncensored\b` | Uncensored anime |

### 2.3 Go Implementation

```go
// internal/adapters/sonarr/formats.go

package sonarr

// FormatSpecTemplate defines how to create a Sonarr custom format spec
type FormatSpecTemplate struct {
    Name           string
    Implementation string // e.g., "ReleaseTitleSpecification"
    Negate         bool
    Required       bool
    Fields         map[string]interface{}
}

// FormatMapping maps our abstract formats to Sonarr spec templates
// Includes both shared formats (same as Radarr) and TV-specific formats
var FormatMapping = map[string]FormatSpecTemplate{
    // === Video HDR Formats (shared with Radarr) ===
    "hdr10": {
        Name:           "HDR10",
        Implementation: "ReleaseTitleSpecification",
        Fields: map[string]interface{}{
            "value": `\bHDR10\b(?![\+P])`,
        },
    },
    "hdr10plus": {
        Name:           "HDR10+",
        Implementation: "ReleaseTitleSpecification",
        Fields: map[string]interface{}{
            "value": `\bHDR10(\+|Plus)\b`,
        },
    },
    "dolby-vision": {
        Name:           "Dolby Vision",
        Implementation: "ReleaseTitleSpecification",
        Fields: map[string]interface{}{
            "value": `\b(DV|DoVi|Dolby[\.\s]?Vision)\b`,
        },
    },
    "hlg": {
        Name:           "HLG",
        Implementation: "ReleaseTitleSpecification",
        Fields: map[string]interface{}{
            "value": `\bHLG\b`,
        },
    },
    
    // === Audio Formats (shared with Radarr) ===
    "atmos": {
        Name:           "Atmos",
        Implementation: "ReleaseTitleSpecification",
        Fields: map[string]interface{}{
            "value": `\bAtmos\b`,
        },
    },
    "truehd": {
        Name:           "TrueHD",
        Implementation: "ReleaseTitleSpecification",
        Fields: map[string]interface{}{
            "value": `\bTrueHD\b`,
        },
    },
    "dts-x": {
        Name:           "DTS:X",
        Implementation: "ReleaseTitleSpecification",
        Fields: map[string]interface{}{
            "value": `\bDTS[-:\s]?X\b`,
        },
    },
    "dts-hd": {
        Name:           "DTS-HD MA",
        Implementation: "ReleaseTitleSpecification",
        Fields: map[string]interface{}{
            "value": `\bDTS[-\s]?HD(\s?MA)?\b`,
        },
    },
    "aac": {
        Name:           "AAC",
        Implementation: "ReleaseTitleSpecification",
        Fields: map[string]interface{}{
            "value": `\bAAC\b`,
        },
    },
    
    // === Video Codecs (shared with Radarr) ===
    "hevc": {
        Name:           "HEVC/x265",
        Implementation: "ReleaseTitleSpecification",
        Fields: map[string]interface{}{
            "value": `\b(HEVC|x265|H\.?265)\b`,
        },
    },
    "av1": {
        Name:           "AV1",
        Implementation: "ReleaseTitleSpecification",
        Fields: map[string]interface{}{
            "value": `\bAV1\b`,
        },
    },
    "10bit": {
        Name:           "10-bit",
        Implementation: "ReleaseTitleSpecification",
        Fields: map[string]interface{}{
            "value": `\b10[-\s]?bit\b`,
        },
    },
    
    // === Release Types (shared with Radarr) ===
    "remux": {
        Name:           "Remux",
        Implementation: "QualityModifierSpecification",
        Fields: map[string]interface{}{
            "value": 5, // REMUX modifier ID
        },
    },
    "proper": {
        Name:           "PROPER",
        Implementation: "ReleaseTitleSpecification",
        Fields: map[string]interface{}{
            "value": `\bPROPER\b`,
        },
    },
    "repack": {
        Name:           "REPACK",
        Implementation: "ReleaseTitleSpecification",
        Fields: map[string]interface{}{
            "value": `\bREPACK\b`,
        },
    },
    
    // === Audio Language (shared with Radarr) ===
    "dual-audio": {
        Name:           "Dual Audio",
        Implementation: "ReleaseTitleSpecification",
        Fields: map[string]interface{}{
            "value": `\bDual[\.\s]?Audio\b`,
        },
    },
    "multi-audio": {
        Name:           "Multi-Audio",
        Implementation: "ReleaseTitleSpecification",
        Fields: map[string]interface{}{
            "value": `\bMulti\b`,
        },
    },
    
    // === TV-Specific: Streaming Sources ===
    "amzn": {
        Name:           "Amazon Prime",
        Implementation: "ReleaseTitleSpecification",
        Fields: map[string]interface{}{
            "value": `\bAMZN\b`,
        },
    },
    "nf": {
        Name:           "Netflix",
        Implementation: "ReleaseTitleSpecification",
        Fields: map[string]interface{}{
            "value": `\b(NF|Netflix)\b`,
        },
    },
    "dsnp": {
        Name:           "Disney+",
        Implementation: "ReleaseTitleSpecification",
        Fields: map[string]interface{}{
            "value": `\bDSNP\b`,
        },
    },
    "atvp": {
        Name:           "Apple TV+",
        Implementation: "ReleaseTitleSpecification",
        Fields: map[string]interface{}{
            "value": `\bATVP\b`,
        },
    },
    "hmax": {
        Name:           "HBO Max",
        Implementation: "ReleaseTitleSpecification",
        Fields: map[string]interface{}{
            "value": `\bHMAX\b`,
        },
    },
    "pcok": {
        Name:           "Peacock",
        Implementation: "ReleaseTitleSpecification",
        Fields: map[string]interface{}{
            "value": `\bPCOK\b`,
        },
    },
    "pmtp": {
        Name:           "Paramount+",
        Implementation: "ReleaseTitleSpecification",
        Fields: map[string]interface{}{
            "value": `\bPMTP\b`,
        },
    },
    
    // === TV-Specific: Release Types ===
    "season-pack": {
        Name:           "Season Pack",
        Implementation: "ReleaseTitleSpecification",
        Fields: map[string]interface{}{
            "value": `\b(S\d{2}(?!E)|Season[\.\s]?\d+)\b`,
        },
    },
    
    // === TV-Specific: Anime ===
    "anime-dual": {
        Name:           "Anime Dual Audio",
        Implementation: "ReleaseTitleSpecification",
        Fields: map[string]interface{}{
            "value": `\b(Dual[\.\s]?Audio|JPN?\+ENG?)\b`,
        },
    },
    "anime-uncensored": {
        Name:           "Uncensored",
        Implementation: "ReleaseTitleSpecification",
        Fields: map[string]interface{}{
            "value": `\bUncensored\b`,
        },
    },
}

// BuildCustomFormat creates a Sonarr custom format from our IR
func BuildCustomFormat(name string, specs []FormatSpecTemplate) map[string]interface{} {
    specifications := make([]map[string]interface{}, 0, len(specs))
    
    for _, spec := range specs {
        specifications = append(specifications, map[string]interface{}{
            "name":           spec.Name,
            "implementation": spec.Implementation,
            "negate":         spec.Negate,
            "required":       spec.Required,
            "fields":         buildFields(spec.Fields),
        })
    }
    
    return map[string]interface{}{
        "name":                  name,
        "includeCustomFormatWhenRenaming": false,
        "specifications":        specifications,
    }
}

// buildFields converts our field map to Sonarr's field array format
func buildFields(fields map[string]interface{}) []map[string]interface{} {
    result := make([]map[string]interface{}, 0, len(fields))
    for name, value := range fields {
        result = append(result, map[string]interface{}{
            "name":  name,
            "value": value,
        })
    }
    return result
}
```

---

## 3. Download Client Mapping

### 3.1 Implementation -> Sonarr Implementation Name

Same as Radarr - Sonarr uses identical download client implementations:

| Our Implementation | Sonarr Implementation |
|--------------------|----------------------|
| `qbittorrent` | `QBittorrent` |
| `transmission` | `Transmission` |
| `deluge` | `Deluge` |
| `rtorrent` | `RTorrent` |
| `utorrent` | `UTorrent` |
| `aria2` | `Aria2` |
| `nzbget` | `Nzbget` |
| `sabnzbd` | `Sabnzbd` |

### 3.2 Sonarr-Specific Field: Category

The key difference is the category field name:

| App | Category Field |
|-----|----------------|
| Radarr | `movieCategory` |
| Sonarr | `tvCategory` |

### 3.3 Go Implementation

```go
// internal/adapters/sonarr/clients.go

package sonarr

import irv1 "github.com/poiley/nebularr/internal/ir/v1"

// ImplementationMapping maps our abstract client names to Sonarr implementation names
var ImplementationMapping = map[string]string{
    "qbittorrent":  "QBittorrent",
    "transmission": "Transmission",
    "deluge":       "Deluge",
    "rtorrent":     "RTorrent",
    "utorrent":     "UTorrent",
    "aria2":        "Aria2",
    "nzbget":       "Nzbget",
    "sabnzbd":      "Sabnzbd",
}

// BuildDownloadClientPayload creates a Sonarr download client from IR
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
            // NOTE: Sonarr uses "tvCategory" instead of Radarr's "movieCategory"
            {"name": "tvCategory", "value": ir.Category},
        },
    }
    
    // Add tags for ownership tracking
    payload["tags"] = []int{} // Will be set to nebularr-managed tag ID
    
    return payload
}
```

---

## 4. Indexer Mapping

### 4.1 Implementation -> Sonarr Implementation Name

Same as Radarr:

| Our Implementation | Sonarr Implementation |
|--------------------|----------------------|
| `torznab` | `Torznab` |
| `newznab` | `Newznab` |
| `rss` | `TorrentRssIndexer` |

### 4.2 Category Mapping (TV Categories)

Sonarr uses TV-specific Newznab category IDs:

| Category Name | Newznab ID | Description |
|---------------|------------|-------------|
| TV | 5000 | All TV |
| TV/Foreign | 5020 | Foreign TV |
| TV/SD | 5030 | SD quality |
| TV/HD | 5040 | HD quality |
| TV/UHD | 5045 | 4K/UHD |
| TV/Anime | 5070 | Anime |
| TV/Documentary | 5080 | Documentaries |
| TV/Sport | 5060 | Sports |

### 4.3 Go Implementation (Category Mapping)

```go
// internal/adapters/sonarr/categories.go

package sonarr

import (
    "log/slog"
    "strconv"
    "strings"
)

// CategoryMapping maps user-friendly category names to Newznab IDs
var CategoryMapping = map[string]int{
    // General
    "tv":         5000,
    "tv-all":     5000,
    
    // Specific categories
    "tv-foreign":     5020,
    "tv-sd":          5030,
    "tv-hd":          5040,
    "tv-uhd":         5045,
    "tv-4k":          5045,
    "tv-anime":       5070,
    "tv-documentary": 5080,
    "tv-sport":       5060,
    
    // Common aliases
    "foreign":     5020,
    "hd":          5040,
    "uhd":         5045,
    "4k":          5045,
    "anime":       5070,
    "documentary": 5080,
    "sport":       5060,
}

// MapCategories converts user-friendly category names to Newznab IDs
func MapCategories(categories []string) []int {
    ids := make([]int, 0, len(categories))
    for _, cat := range categories {
        if id, ok := CategoryMapping[strings.ToLower(cat)]; ok {
            ids = append(ids, id)
        } else {
            // Try parsing as raw integer (user may specify raw Newznab ID)
            if rawID, err := strconv.Atoi(cat); err == nil {
                ids = append(ids, rawID)
            } else {
                slog.Warn("unknown category, skipping", "category", cat)
            }
        }
    }
    return ids
}

// ReverseCategoryMapping for display purposes
var ReverseCategoryMapping = func() map[int]string {
    m := make(map[int]string)
    for name, id := range CategoryMapping {
        // Prefer canonical names (tv-X format)
        if _, exists := m[id]; !exists || strings.HasPrefix(name, "tv-") {
            m[id] = name
        }
    }
    return m
}()
```

### 4.4 Go Implementation (Indexer Payload)

```go
// internal/adapters/sonarr/indexers.go

package sonarr

import irv1 "github.com/poiley/nebularr/internal/ir/v1"

// IndexerImplementationMapping maps our abstract indexer types to Sonarr implementation names
var IndexerImplementationMapping = map[string]string{
    "torznab": "Torznab",
    "newznab": "Newznab",
    "rss":     "TorrentRssIndexer",
}

// BuildIndexerPayload creates a Sonarr indexer from IR
func BuildIndexerPayload(ir *irv1.IndexerIR) map[string]interface{} {
    implementation := IndexerImplementationMapping[ir.Implementation]
    
    payload := map[string]interface{}{
        "name":           ir.Name,
        "implementation": implementation,
        "configContract": implementation + "Settings",
        "enable":         ir.Enable,
        "protocol":       ir.Protocol,
        "priority":       ir.Priority,
        "enableRss":               ir.EnableRss,
        "enableAutomaticSearch":   ir.EnableAutomaticSearch,
        "enableInteractiveSearch": ir.EnableInteractiveSearch,
        "fields": []map[string]interface{}{
            {"name": "baseUrl", "value": ir.URL},
            {"name": "apiKey", "value": ir.APIKey},
            {"name": "categories", "value": ir.Categories},
            {"name": "minimumSeeders", "value": ir.MinimumSeeders},
            {"name": "seedCriteria.seedRatio", "value": ir.SeedRatio},
            {"name": "seedCriteria.seedTime", "value": ir.SeedTimeMinutes},
            // Sonarr-specific: anime categories (optional)
            {"name": "animeCategories", "value": ir.AnimeCategories},
        },
    }
    
    payload["tags"] = []int{}
    
    return payload
}
```

---

## 5. Series Type Mapping

### 5.1 Series Types

Sonarr distinguishes between different types of series:

| Series Type | API Value | Description |
|-------------|-----------|-------------|
| Standard | `standard` | Regular TV shows with seasons (S01E01) |
| Daily | `daily` | Daily shows dated by air date (2024-01-15) |
| Anime | `anime` | Anime with absolute episode numbering |

### 5.2 Go Implementation

```go
// internal/adapters/sonarr/series.go

package sonarr

// SeriesType represents the type of series
type SeriesType string

const (
    SeriesTypeStandard SeriesType = "standard"
    SeriesTypeDaily    SeriesType = "daily"
    SeriesTypeAnime    SeriesType = "anime"
)

// ValidSeriesTypes for validation
var ValidSeriesTypes = map[SeriesType]bool{
    SeriesTypeStandard: true,
    SeriesTypeDaily:    true,
    SeriesTypeAnime:    true,
}

// DefaultSeriesType returns the default series type
func DefaultSeriesType() SeriesType {
    return SeriesTypeStandard
}
```

---

## 6. Monitor Type Mapping

### 6.1 Monitor Options

Sonarr has extensive monitoring options for series:

| Monitor Option | API Value | Description |
|----------------|-----------|-------------|
| All Episodes | `all` | Monitor all episodes |
| Future Episodes | `future` | Only future episodes |
| Missing Episodes | `missing` | Only missing episodes |
| Existing Episodes | `existing` | Only episodes that exist |
| Recent Episodes | `recent` | Episodes from recent seasons |
| Pilot | `pilot` | Only the pilot episode |
| First Season | `firstSeason` | First season only |
| Last Season | `lastSeason` | Most recent season |
| Monitor Specials | `monitorSpecials` | Include specials |
| Unmonitor Specials | `unmonitorSpecials` | Exclude specials |
| None | `none` | Don't monitor any |

### 6.2 Go Implementation

```go
// internal/adapters/sonarr/monitor.go

package sonarr

// MonitorType represents how to monitor a series
type MonitorType string

const (
    MonitorAll               MonitorType = "all"
    MonitorFuture            MonitorType = "future"
    MonitorMissing           MonitorType = "missing"
    MonitorExisting          MonitorType = "existing"
    MonitorRecent            MonitorType = "recent"
    MonitorPilot             MonitorType = "pilot"
    MonitorFirstSeason       MonitorType = "firstSeason"
    MonitorLastSeason        MonitorType = "lastSeason"
    MonitorMonitorSpecials   MonitorType = "monitorSpecials"
    MonitorUnmonitorSpecials MonitorType = "unmonitorSpecials"
    MonitorNone              MonitorType = "none"
)

// ValidMonitorTypes for validation
var ValidMonitorTypes = map[MonitorType]bool{
    MonitorAll:               true,
    MonitorFuture:            true,
    MonitorMissing:           true,
    MonitorExisting:          true,
    MonitorRecent:            true,
    MonitorPilot:             true,
    MonitorFirstSeason:       true,
    MonitorLastSeason:        true,
    MonitorMonitorSpecials:   true,
    MonitorUnmonitorSpecials: true,
    MonitorNone:              true,
}

// DefaultMonitorType returns the default monitor type
func DefaultMonitorType() MonitorType {
    return MonitorAll
}
```

---

## 7. Import List Mapping

### 7.1 Import List Types

Sonarr-specific import list implementations:

| Our Type | Sonarr Implementation | Description |
|----------|----------------------|-------------|
| `imdb` | `ImdbListImport` | IMDb lists (watchlist, custom lists) |
| `trakt-list` | `TraktListImport` | Trakt user lists |
| `trakt-popular` | `TraktPopularImport` | Trakt popular/trending |
| `plex` | `PlexImport` | Plex watchlist |
| `sonarr` | `SonarrImport` | Another Sonarr instance |
| `simkl` | `SimklUserImport` | Simkl watchlist |

### 7.2 Go Implementation

```go
// internal/adapters/sonarr/importlists.go

package sonarr

import irv1 "github.com/poiley/nebularr/internal/ir/v1"

// ImportListImplementationMapping maps our types to Sonarr implementations
var ImportListImplementationMapping = map[string]string{
    "imdb":          "ImdbListImport",
    "trakt-list":    "TraktListImport",
    "trakt-popular": "TraktPopularImport",
    "plex":          "PlexImport",
    "sonarr":        "SonarrImport",
    "simkl":         "SimklUserImport",
}

// BuildImportListPayload creates a Sonarr import list from IR
func BuildImportListPayload(ir *irv1.ImportListIR) map[string]interface{} {
    implementation := ImportListImplementationMapping[ir.Implementation]
    
    // Build fields from IR settings
    fields := []map[string]interface{}{}
    for key, value := range ir.Settings {
        fields = append(fields, map[string]interface{}{
            "name":  key,
            "value": value,
        })
    }
    
    payload := map[string]interface{}{
        "name":           ir.Name,
        "implementation": implementation,
        "configContract": implementation + "Settings",
        "enable":         ir.Enable,
        "enableAutomaticAdd": ir.EnableAuto,
        "searchForMissingEpisodes": ir.SearchOnAdd,
        "qualityProfileId": ir.QualityProfileID,
        "rootFolderPath":   ir.RootFolderPath,
        // Sonarr-specific fields
        "seriesType":       ir.SeriesType,
        "seasonFolder":     ir.SeasonFolder,
        "shouldMonitor":    ir.Monitor,
        "fields":           fields,
    }
    
    payload["tags"] = []int{}
    
    return payload
}
```

---

## 8. Naming Configuration

### 8.1 Sonarr Naming Fields

Sonarr has TV-specific naming configuration:

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `renameEpisodes` | bool | false | Enable episode renaming |
| `replaceIllegalCharacters` | bool | true | Replace illegal characters |
| `colonReplacementFormat` | int | 5 | How to replace colons (see below) |
| `standardEpisodeFormat` | string | varies | Format for standard episodes |
| `dailyEpisodeFormat` | string | varies | Format for daily shows |
| `animeEpisodeFormat` | string | varies | Format for anime |
| `seriesFolderFormat` | string | varies | Series folder naming |
| `seasonFolderFormat` | string | varies | Season folder naming |
| `specialsFolderFormat` | string | varies | Specials folder naming |
| `multiEpisodeStyle` | int | 5 | Multi-episode format style |

### 8.2 Colon Replacement Values

| Value | Description | Example |
|-------|-------------|---------|
| 0 | Delete | `Title Name` |
| 1 | Replace with Dash | `Title - Name` |
| 2 | Replace with Space Dash | `Title - Name` |
| 3 | Replace with Space Dash Space | `Title - Name` |
| 4 | Smart Replace | Context-dependent |
| 5 | Custom | User-defined |

### 8.3 Multi-Episode Style Values

| Value | Style | Example |
|-------|-------|---------|
| 0 | Extend | `S01E01-02-03` |
| 1 | Duplicate | `S01E01, S01E02, S01E03` |
| 2 | Repeat | `S01E01E02E03` |
| 3 | Scene | `S01E01-E02-E03` |
| 4 | Range | `S01E01-03` |
| 5 | Prefixed Range | `S01E01-E03` |

### 8.4 Go Implementation

```go
// internal/adapters/sonarr/naming.go

package sonarr

import irv1 "github.com/poiley/nebularr/internal/ir/v1"

// ColonReplacement values for Sonarr
const (
    ColonDelete         = 0
    ColonDash           = 1
    ColonSpaceDash      = 2
    ColonSpaceDashSpace = 3
    ColonSmart          = 4
    ColonCustom         = 5
)

// MultiEpisodeStyle values for Sonarr
const (
    MultiEpisodeExtend        = 0
    MultiEpisodeDuplicate     = 1
    MultiEpisodeRepeat        = 2
    MultiEpisodeScene         = 3
    MultiEpisodeRange         = 4
    MultiEpisodePrefixedRange = 5
)

// DefaultNamingConfig returns default Sonarr naming settings
func DefaultNamingConfig() map[string]interface{} {
    return map[string]interface{}{
        "renameEpisodes":           false,
        "replaceIllegalCharacters": true,
        "colonReplacementFormat":   ColonSmart,
        "standardEpisodeFormat":    "{Series Title} - S{season:00}E{episode:00} - {Episode Title} {Quality Full}",
        "dailyEpisodeFormat":       "{Series Title} - {Air-Date} - {Episode Title} {Quality Full}",
        "animeEpisodeFormat":       "{Series Title} - S{season:00}E{episode:00} - {Episode Title} {Quality Full}",
        "seriesFolderFormat":       "{Series Title}",
        "seasonFolderFormat":       "Season {season}",
        "specialsFolderFormat":     "Specials",
        "multiEpisodeStyle":        MultiEpisodePrefixedRange,
    }
}

// BuildNamingPayload creates Sonarr naming config from IR
func BuildNamingPayload(ir *irv1.NamingIR) map[string]interface{} {
    payload := DefaultNamingConfig()
    
    // Override with IR values if set
    if ir.RenameEpisodes != nil {
        payload["renameEpisodes"] = *ir.RenameEpisodes
    }
    if ir.ReplaceIllegalCharacters != nil {
        payload["replaceIllegalCharacters"] = *ir.ReplaceIllegalCharacters
    }
    if ir.ColonReplacement != nil {
        payload["colonReplacementFormat"] = *ir.ColonReplacement
    }
    if ir.StandardEpisodeFormat != "" {
        payload["standardEpisodeFormat"] = ir.StandardEpisodeFormat
    }
    if ir.DailyEpisodeFormat != "" {
        payload["dailyEpisodeFormat"] = ir.DailyEpisodeFormat
    }
    if ir.AnimeEpisodeFormat != "" {
        payload["animeEpisodeFormat"] = ir.AnimeEpisodeFormat
    }
    if ir.SeriesFolderFormat != "" {
        payload["seriesFolderFormat"] = ir.SeriesFolderFormat
    }
    if ir.SeasonFolderFormat != "" {
        payload["seasonFolderFormat"] = ir.SeasonFolderFormat
    }
    if ir.SpecialsFolderFormat != "" {
        payload["specialsFolderFormat"] = ir.SpecialsFolderFormat
    }
    if ir.MultiEpisodeStyle != nil {
        payload["multiEpisodeStyle"] = *ir.MultiEpisodeStyle
    }
    
    // Required for PUT requests
    payload["id"] = 1
    
    return payload
}
```

---

## 9. Capability Discovery Endpoints

### 9.1 API Endpoints for Discovery

| Capability | Endpoint | Response Field |
|------------|----------|----------------|
| Quality definitions | `GET /api/v3/qualitydefinition` | List of available qualities |
| Custom format schema | `GET /api/v3/customformat/schema` | Available spec types |
| Download client schema | `GET /api/v3/downloadclient/schema` | Available client types |
| Indexer schema | `GET /api/v3/indexer/schema` | Available indexer types |
| Import list schema | `GET /api/v3/importlist/schema` | Available import list types |
| System status | `GET /api/v3/system/status` | Version info |
| Quality profiles | `GET /api/v3/qualityprofile` | Existing quality profiles |
| Root folders | `GET /api/v3/rootfolder` | Configured root folders |

### 9.2 Go Implementation

```go
// internal/adapters/sonarr/adapter.go

package sonarr

import (
    "context"
    "fmt"
    "log/slog"
    "time"
    
    "github.com/poiley/nebularr/internal/adapters"
    "github.com/poiley/nebularr/internal/adapters/sonarr/client"
)

// Adapter implements the adapters.Adapter interface for Sonarr
type Adapter struct {
    client *client.ClientWithResponses // Generated by oapi-codegen
}

// NewAdapter creates a new Sonarr adapter
func NewAdapter() *Adapter {
    return &Adapter{}
}

// Name returns the adapter identifier
func (a *Adapter) Name() string {
    return "sonarr"
}

// SupportedApp returns the app this adapter handles
func (a *Adapter) SupportedApp() string {
    return "sonarr"
}

// Discover queries Sonarr for available features
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
        tier := ReverseQualityMapping[q.Quality.ID]
        if tier.Resolution != "" {
            caps.Resolutions = appendUnique(caps.Resolutions, tier.Resolution)
            caps.Sources = appendUnique(caps.Sources, tier.Source)
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
    
    // Get indexer types
    indexerSchemas, err := a.client.GetIndexerSchema(ctx)
    if err != nil {
        slog.Warn("failed to get indexer schema", "error", err)
    } else {
        for _, schema := range indexerSchemas {
            caps.IndexerTypes = append(caps.IndexerTypes, schema.Implementation)
        }
    }
    
    // Get import list types (Sonarr-specific)
    importListSchemas, err := a.client.GetImportListSchema(ctx)
    if err != nil {
        slog.Warn("failed to get import list schema", "error", err)
    } else {
        for _, schema := range importListSchemas {
            caps.ImportListTypes = append(caps.ImportListTypes, schema.Implementation)
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

## 10. Ownership Tagging

Same pattern as Radarr - Sonarr uses identical tag management:

```go
const OwnershipTagName = "nebularr-managed"

// EnsureOwnershipTag creates the ownership tag if it doesn't exist
func (a *Adapter) EnsureOwnershipTag(ctx context.Context) (int, error) {
    // GET /api/v3/tag - list existing tags
    tags, err := a.client.ListTags(ctx)
    if err != nil {
        return 0, err
    }
    
    // Check if tag exists
    for _, tag := range tags {
        if tag.Label == OwnershipTagName {
            return tag.ID, nil
        }
    }
    
    // POST /api/v3/tag - create new tag
    newTag, err := a.client.CreateTag(ctx, OwnershipTagName)
    if err != nil {
        return 0, err
    }
    
    return newTag.ID, nil
}
```

---

## 11. Extended IR Types for Sonarr

### 11.1 Sonarr-Specific IR Extensions

```go
// internal/ir/v1/sonarr_types.go

package v1

// SonarrNamingIR extends NamingIR with Sonarr-specific fields
type SonarrNamingIR struct {
    NamingIR
    
    // Episode formats
    StandardEpisodeFormat string `json:"standardEpisodeFormat,omitempty"`
    DailyEpisodeFormat    string `json:"dailyEpisodeFormat,omitempty"`
    AnimeEpisodeFormat    string `json:"animeEpisodeFormat,omitempty"`
    
    // Folder formats
    SeriesFolderFormat   string `json:"seriesFolderFormat,omitempty"`
    SeasonFolderFormat   string `json:"seasonFolderFormat,omitempty"`
    SpecialsFolderFormat string `json:"specialsFolderFormat,omitempty"`
    
    // Multi-episode handling
    MultiEpisodeStyle *int `json:"multiEpisodeStyle,omitempty"`
}

// SonarrImportListIR extends ImportListIR with Sonarr-specific fields
type SonarrImportListIR struct {
    ImportListIR
    
    // Series configuration
    SeriesType   string `json:"seriesType,omitempty"`   // standard, daily, anime
    SeasonFolder bool   `json:"seasonFolder,omitempty"`
    Monitor      string `json:"monitor,omitempty"`      // all, future, missing, etc.
}

// SonarrIndexerIR extends IndexerIR with Sonarr-specific fields
type SonarrIndexerIR struct {
    IndexerIR
    
    // Anime-specific categories
    AnimeCategories []int `json:"animeCategories,omitempty"`
}
```

---

## 12. Related Documents

- [README](./README.md) - Build order, file mapping (start here)
- [RADARR](./RADARR.md) - Radarr adapter (compare implementations)
- [TYPES](./TYPES.md) - IR types and adapter interface
- [CRDS](./CRDS.md) - CRD definitions
- [PRESETS](./PRESETS.md) - Quality and naming presets
- [OPERATIONS](./OPERATIONS.md) - Auto-discovery, secrets, merge rules
- [PROWLARR](./PROWLARR.md) - Prowlarr integration
