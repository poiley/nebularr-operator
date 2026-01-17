# Nebularr — Radarr API Mapping Reference

> **For coding agents:** Start with [README.md](./README.md) for build order. This document contains Radarr adapter code to copy.
>
> **Related:** [README](./README.md) | [TYPES](./TYPES.md) | [CRDS](./CRDS.md)

This document is a reference for implementing the Radarr adapter. It maps our abstract quality/format concepts to Radarr's specific API fields and IDs.

---

## 1. Quality Tier Mapping

### 1.1 Resolution + Source → Radarr Quality ID

Our `QualityTier{Resolution, Source}` maps to Radarr's quality IDs:

| Resolution | Source | Radarr ID | Radarr Name |
|------------|--------|-----------|-------------|
| 2160p | remux | 31 | Remux-2160p |
| 2160p | bluray | 19 | Bluray-2160p |
| 2160p | webdl | 18 | WEBDL-2160p |
| 2160p | webrip | 17 | WEBRip-2160p |
| 2160p | hdtv | 16 | HDTV-2160p |
| 1080p | remux | 30 | Remux-1080p |
| 1080p | bluray | 7 | Bluray-1080p |
| 1080p | webdl | 3 | WEBDL-1080p |
| 1080p | webrip | 15 | WEBRip-1080p |
| 1080p | hdtv | 9 | HDTV-1080p |
| 720p | bluray | 6 | Bluray-720p |
| 720p | webdl | 5 | WEBDL-720p |
| 720p | webrip | 14 | WEBRip-720p |
| 720p | hdtv | 4 | HDTV-720p |
| 480p | bluray | 23 | Bluray-480p |
| 480p | webdl | 12 | WEBDL-480p |
| 480p | webrip | 22 | WEBRip-480p |
| 480p | dvd | 2 | DVD |
| any | cam | 25 | CAM |
| any | telecine | 26 | TELECINE |
| any | telesync | 24 | TELESYNC |
| any | workprint | 29 | WORKPRINT |
| any | raw-hd | 10 | Raw-HD |

### 1.2 Go Implementation

```go
// internal/adapters/radarr/mapping.go

package radarr

// QualityKey is a lookup key for mapping resolution+source to Radarr quality IDs
// This is an adapter-internal type, not from IR. The IR uses VideoQualityTierIR
// with Sources as an array; the adapter iterates through sources and maps each.
type QualityKey struct {
    Resolution string
    Source     string
}

// QualityMapping maps resolution+source combinations to Radarr quality IDs
var QualityMapping = map[QualityKey]int{
    // 2160p
    {Resolution: "2160p", Source: "remux"}:  31,
    {Resolution: "2160p", Source: "bluray"}: 19,
    {Resolution: "2160p", Source: "webdl"}:  18,
    {Resolution: "2160p", Source: "webrip"}: 17,
    {Resolution: "2160p", Source: "hdtv"}:   16,
    
    // 1080p
    {Resolution: "1080p", Source: "remux"}:  30,
    {Resolution: "1080p", Source: "bluray"}: 7,
    {Resolution: "1080p", Source: "webdl"}:  3,
    {Resolution: "1080p", Source: "webrip"}: 15,
    {Resolution: "1080p", Source: "hdtv"}:   9,
    
    // 720p
    {Resolution: "720p", Source: "bluray"}: 6,
    {Resolution: "720p", Source: "webdl"}:  5,
    {Resolution: "720p", Source: "webrip"}: 14,
    {Resolution: "720p", Source: "hdtv"}:   4,
    
    // 480p
    {Resolution: "480p", Source: "bluray"}: 23,
    {Resolution: "480p", Source: "webdl"}:  12,
    {Resolution: "480p", Source: "webrip"}: 22,
    {Resolution: "480p", Source: "dvd"}:    2,
    
    // Low quality (resolution-agnostic)
    {Resolution: "any", Source: "cam"}:      25,
    {Resolution: "any", Source: "telecine"}: 26,
    {Resolution: "any", Source: "telesync"}: 24,
    {Resolution: "any", Source: "workprint"}: 29,
}

// MapQuality converts resolution+source to Radarr quality ID
// Returns (id, true) if found, (0, false) if not supported
func MapQuality(resolution, source string) (int, bool) {
    id, ok := QualityMapping[QualityKey{Resolution: resolution, Source: source}]
    return id, ok
}

// ReverseQualityMapping for converting Radarr state back to resolution+source
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

### 2.1 Format → Radarr Custom Format Spec

Our abstract format names map to Radarr custom format specifications:

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
| `vp9` | ReleaseTitleSpecification | `\bVP9\b` |
| `10bit` | ReleaseTitleSpecification | `\b10[-\s]?bit\b` |
| `3d` | ReleaseTitleSpecification | `\b3D\b` |
| `remux` | QualityModifierSpecification | `5` (REMUX modifier ID) |
| `proper` | ReleaseTitleSpecification | `\bPROPER\b` |
| `repack` | ReleaseTitleSpecification | `\bREPACK\b` |
| `extended` | ReleaseTitleSpecification | `\b(Extended\|Uncut\|Unrated)\b` |
| `directors-cut` | ReleaseTitleSpecification | `\bDirector'?s?\s?Cut\b` |
| `unrated` | ReleaseTitleSpecification | `\bUnrated\b` |
| `imax` | ReleaseTitleSpecification | `\bIMAX\b` |
| `open-matte` | ReleaseTitleSpecification | `\bOpen[\.\s]?Matte\b` |
| `multi-audio` | ReleaseTitleSpecification | `\bMulti\b` |
| `dual-audio` | ReleaseTitleSpecification | `\bDual[\.\s]?Audio\b` |
| `commentary` | ReleaseTitleSpecification | `\bCommentary\b` |

### 2.2 Go Implementation

```go
// internal/adapters/radarr/formats.go

package radarr

// FormatSpecTemplate defines how to create a Radarr custom format spec
type FormatSpecTemplate struct {
    Name           string
    Implementation string // e.g., "ReleaseTitleSpecification"
    Negate         bool
    Required       bool
    Fields         map[string]interface{}
}

// FormatMapping maps our abstract formats to Radarr spec templates
var FormatMapping = map[string]FormatSpecTemplate{
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
    "hevc": {
        Name:           "HEVC/x265",
        Implementation: "ReleaseTitleSpecification",
        Fields: map[string]interface{}{
            "value": `\b(HEVC|x265|H\.?265)\b`,
        },
    },
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
    "hlg": {
        Name:           "HLG",
        Implementation: "ReleaseTitleSpecification",
        Fields: map[string]interface{}{
            "value": `\bHLG\b`,
        },
    },
    "av1": {
        Name:           "AV1",
        Implementation: "ReleaseTitleSpecification",
        Fields: map[string]interface{}{
            "value": `\bAV1\b`,
        },
    },
    "vp9": {
        Name:           "VP9",
        Implementation: "ReleaseTitleSpecification",
        Fields: map[string]interface{}{
            "value": `\bVP9\b`,
        },
    },
    "10bit": {
        Name:           "10-bit",
        Implementation: "ReleaseTitleSpecification",
        Fields: map[string]interface{}{
            "value": `\b10[-\s]?bit\b`,
        },
    },
    "3d": {
        Name:           "3D",
        Implementation: "ReleaseTitleSpecification",
        Fields: map[string]interface{}{
            "value": `\b3D\b`,
        },
    },
    "extended": {
        Name:           "Extended Cut",
        Implementation: "ReleaseTitleSpecification",
        Fields: map[string]interface{}{
            "value": `\b(Extended|Uncut|Unrated)\b`,
        },
    },
    "directors-cut": {
        Name:           "Director's Cut",
        Implementation: "ReleaseTitleSpecification",
        Fields: map[string]interface{}{
            "value": `\bDirector'?s?\s?Cut\b`,
        },
    },
    "unrated": {
        Name:           "Unrated",
        Implementation: "ReleaseTitleSpecification",
        Fields: map[string]interface{}{
            "value": `\bUnrated\b`,
        },
    },
    "imax": {
        Name:           "IMAX",
        Implementation: "ReleaseTitleSpecification",
        Fields: map[string]interface{}{
            "value": `\bIMAX\b`,
        },
    },
    "open-matte": {
        Name:           "Open Matte",
        Implementation: "ReleaseTitleSpecification",
        Fields: map[string]interface{}{
            "value": `\bOpen[\.\s]?Matte\b`,
        },
    },
    "multi-audio": {
        Name:           "Multi-Audio",
        Implementation: "ReleaseTitleSpecification",
        Fields: map[string]interface{}{
            "value": `\bMulti\b`,
        },
    },
    "dual-audio": {
        Name:           "Dual Audio",
        Implementation: "ReleaseTitleSpecification",
        Fields: map[string]interface{}{
            "value": `\bDual[\.\s]?Audio\b`,
        },
    },
    "aac": {
        Name:           "AAC",
        Implementation: "ReleaseTitleSpecification",
        Fields: map[string]interface{}{
            "value": `\bAAC\b`,
        },
    },
    "commentary": {
        Name:           "Commentary",
        Implementation: "ReleaseTitleSpecification",
        Fields: map[string]interface{}{
            "value": `\bCommentary\b`,
        },
    },
}

// BuildCustomFormat creates a Radarr custom format from our IR
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

// buildFields converts our field map to Radarr's field array format
// Radarr expects: [{"name": "fieldName", "value": fieldValue}, ...]
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

### 3.1 Implementation → Radarr Implementation Name

| Our Implementation | Radarr Implementation |
|--------------------|----------------------|
| `qbittorrent` | `QBittorrent` |
| `transmission` | `Transmission` |
| `deluge` | `Deluge` |
| `rtorrent` | `RTorrent` |
| `utorrent` | `UTorrent` |
| `aria2` | `Aria2` |
| `nzbget` | `Nzbget` |
| `sabnzbd` | `Sabnzbd` |

### 3.2 Go Implementation

```go
// internal/adapters/radarr/clients.go

package radarr

import irv1 "github.com/poiley/nebularr/internal/ir/v1"

// ImplementationMapping maps our abstract client names to Radarr implementation names
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

// BuildDownloadClientPayload creates a Radarr download client from IR
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
            {"name": "movieCategory", "value": ir.Category},
        },
    }
    
    // Add tags for ownership tracking
    payload["tags"] = []int{} // Will be set to nebularr-managed tag ID
    
    return payload
}
```

---

## 4. Indexer Mapping

### 4.1 Implementation → Radarr Implementation Name

| Our Implementation | Radarr Implementation |
|--------------------|----------------------|
| `torznab` | `Torznab` |
| `newznab` | `Newznab` |
| `rss` | `TorrentRssIndexer` |

### 4.2 Category Mapping

Radarr uses numeric category IDs (Newznab standard). Common movie categories:

| Category Name | Newznab ID | Description |
|---------------|------------|-------------|
| Movies | 2000 | All movies |
| Movies/Foreign | 2010 | Foreign movies |
| Movies/Other | 2020 | Other movies |
| Movies/SD | 2030 | SD quality |
| Movies/HD | 2040 | HD quality |
| Movies/UHD | 2045 | 4K/UHD |
| Movies/BluRay | 2050 | Blu-ray |
| Movies/3D | 2060 | 3D movies |

### 4.3 Go Implementation (Category Mapping)

```go
// internal/adapters/radarr/categories.go

package radarr

import (
    "log/slog"
    "strconv"
    "strings"
)

// CategoryMapping maps user-friendly category names to Newznab IDs
var CategoryMapping = map[string]int{
    // General
    "movies":         2000,
    "movies-all":     2000,
    
    // Specific categories
    "movies-foreign": 2010,
    "movies-other":   2020,
    "movies-sd":      2030,
    "movies-hd":      2040,
    "movies-uhd":     2045,
    "movies-4k":      2045,
    "movies-bluray":  2050,
    "movies-3d":      2060,
    
    // Common aliases
    "foreign":        2010,
    "hd":             2040,
    "uhd":            2045,
    "4k":             2045,
    "bluray":         2050,
    "3d":             2060,
}

// MapCategories converts user-friendly category names to Newznab IDs
// Unknown categories are logged as warnings and skipped
func MapCategories(categories []string) []int {
    ids := make([]int, 0, len(categories))
    for _, cat := range categories {
        if id, ok := CategoryMapping[cat]; ok {
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
        // Prefer canonical names (movies-X format)
        if _, exists := m[id]; !exists || strings.HasPrefix(name, "movies-") {
            m[id] = name
        }
    }
    return m
}()
```

### 4.4 Go Implementation (Indexer Mapping + Payload)

```go
// internal/adapters/radarr/indexers.go

package radarr

import irv1 "github.com/poiley/nebularr/internal/ir/v1"

// IndexerImplementationMapping maps our abstract indexer types to Radarr implementation names
var IndexerImplementationMapping = map[string]string{
    "torznab": "Torznab",
    "newznab": "Newznab",
    "rss":     "TorrentRssIndexer",
}

// BuildIndexerPayload creates a Radarr indexer from IR
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
        },
    }
    
    // Add tags for ownership tracking
    payload["tags"] = []int{} // Will be set to nebularr-managed tag ID
    
    return payload
}
```

---

## 5. Ownership Tagging

Nebularr tracks owned resources using a dedicated tag.

### 5.1 Tag Creation

```go
const OwnershipTagName = "nebularr-managed"

// EnsureOwnershipTag creates the ownership tag if it doesn't exist
func (a *Adapter) EnsureOwnershipTag(ctx context.Context, conn *Connection) (int, error) {
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

### 5.2 Resource Naming Convention

All resources created by Nebularr follow this naming pattern:

| Resource Type | Naming Pattern | Example |
|---------------|----------------|---------|
| Quality Profile | `nebularr-{policy-name}` | `nebularr-4k-quality` |
| Custom Format | `nebularr-{format-name}` | `nebularr-hdr10` |
| Download Client | `nebularr-{policy-name}` | `nebularr-qbittorrent-config` |
| Indexer | `nebularr-{indexer-name}` | `nebularr-1337x` |
| Tag | `nebularr-managed` | `nebularr-managed` |

---

## 6. Capability Discovery Endpoints

### 6.1 API Endpoints for Discovery

| Capability | Endpoint | Response Field |
|------------|----------|----------------|
| Quality definitions | `GET /api/v3/qualitydefinition` | List of available qualities |
| Custom format schema | `GET /api/v3/customformat/schema` | Available spec types |
| Download client schema | `GET /api/v3/downloadclient/schema` | Available client types |
| Indexer schema | `GET /api/v3/indexer/schema` | Available indexer types |
| System status | `GET /api/v3/system/status` | Version info |

### 6.2 Go Implementation

```go
// internal/adapters/radarr/adapter.go

package radarr

import (
    "context"
    "fmt"
    "log/slog"
    "time"
    
    "github.com/poiley/nebularr/internal/adapters"
    "github.com/poiley/nebularr/internal/adapters/radarr/client"
)

// Adapter implements the adapters.Adapter interface for Radarr
type Adapter struct {
    client *client.ClientWithResponses // Generated by oapi-codegen
}

// NewAdapter creates a new Radarr adapter
func NewAdapter() *Adapter {
    return &Adapter{}
}

// Name returns the adapter identifier
func (a *Adapter) Name() string {
    return "radarr"
}

// SupportedApp returns the app this adapter handles
func (a *Adapter) SupportedApp() string {
    return "radarr"
}

// Discover queries Radarr for available features
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
        // Log warning but don't fail - degrade gracefully
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

## 7. Delay Profile Mapping

### 7.1 Delay Profile Structure

Delay profiles control when downloads should start based on protocol preferences and timing:

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `enableUsenet` | bool | true | Enable Usenet downloads |
| `enableTorrent` | bool | true | Enable torrent downloads |
| `preferredProtocol` | string | usenet | Preferred protocol: `usenet` or `torrent` |
| `usenetDelay` | int | 0 | Minutes to wait for Usenet releases |
| `torrentDelay` | int | 0 | Minutes to wait for torrent releases |
| `bypassIfHighestQuality` | bool | false | Skip delay if release meets quality cutoff |
| `bypassIfAboveCustomFormatScore` | bool | false | Skip delay if CF score threshold met |
| `minimumCustomFormatScore` | int | 0 | Minimum CF score to bypass delay |
| `order` | int | varies | Priority (lower = higher priority) |
| `tags` | []int | [] | Restrict to items with these tags |

### 7.2 Go Implementation

```go
// internal/adapters/radarr/delayprofiles.go

package radarr

import irv1 "github.com/poiley/nebularr/internal/ir/v1"

// BuildDelayProfilePayload creates a Radarr delay profile from IR
func BuildDelayProfilePayload(ir *irv1.DelayProfileIR, tagIDs []int) map[string]interface{} {
    enableUsenet := true
    if ir.EnableUsenet != nil {
        enableUsenet = *ir.EnableUsenet
    }
    
    enableTorrent := true
    if ir.EnableTorrent != nil {
        enableTorrent = *ir.EnableTorrent
    }
    
    bypassIfHighestQuality := false
    if ir.BypassIfHighestQuality != nil {
        bypassIfHighestQuality = *ir.BypassIfHighestQuality
    }
    
    bypassIfAboveCustomFormatScore := false
    if ir.BypassIfAboveCustomFormatScore != nil {
        bypassIfAboveCustomFormatScore = *ir.BypassIfAboveCustomFormatScore
    }
    
    preferredProtocol := "usenet"
    if ir.PreferredProtocol != "" {
        preferredProtocol = ir.PreferredProtocol
    }
    
    return map[string]interface{}{
        "enableUsenet":                   enableUsenet,
        "enableTorrent":                  enableTorrent,
        "preferredProtocol":              preferredProtocol,
        "usenetDelay":                    ir.UsenetDelay,
        "torrentDelay":                   ir.TorrentDelay,
        "bypassIfHighestQuality":         bypassIfHighestQuality,
        "bypassIfAboveCustomFormatScore": bypassIfAboveCustomFormatScore,
        "minimumCustomFormatScore":       ir.MinimumCustomFormatScore,
        "order":                          ir.Order,
        "tags":                           tagIDs,
    }
}
```

### 7.3 Example CRD Usage

```yaml
apiVersion: nebularr.io/v1alpha1
kind: RadarrConfig
metadata:
  name: radarr
spec:
  connection:
    url: http://radarr:7878
    apiKeySecretRef:
      name: radarr-secret
      key: api-key
  
  delayProfiles:
    # Default: prefer Usenet, wait 2 hours for torrents
    - name: Default
      preferredProtocol: usenet
      usenetDelay: 0
      torrentDelay: 120
      bypassIfHighestQuality: true
    
    # 4K content: longer delays for better releases
    - name: 4K Releases
      preferredProtocol: usenet
      usenetDelay: 60
      torrentDelay: 240
      bypassIfAboveCustomFormatScore: true
      minimumCustomFormatScore: 1000
      tags:
        - 4k
```

---

## 8. Related Documents

- [README](./README.md) - Build order, file mapping (start here)
- [TYPES](./TYPES.md) - IR types and adapter interface
- [CRDS](./CRDS.md) - CRD definitions
- [PRESETS](./PRESETS.md) - Quality and naming presets
- [OPERATIONS](./OPERATIONS.md) - Auto-discovery, secrets, merge rules
