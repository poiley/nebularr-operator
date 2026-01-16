# Nebularr - Implementation Plan (v0.2)

> **For coding agents:** Start with [README.md](./README.md) for build order. This document contains type definitions to copy.
>
> **Related:** [README](./README.md) | [CRDS](./CRDS.md) | [PRESETS](./PRESETS.md) | [DESIGN](./DESIGN.md)

---

## 1. Project Overview

| Property | Value |
|----------|-------|
| **Project Name** | Nebularr |
| **Go Module** | `github.com/poiley/nebularr` |
| **API Group** | `arr.rinzler.cloud` |
| **Initial API Version** | `v1alpha1` |
| **Target Kubernetes** | 1.35+ |
| **Tooling** | Kubebuilder go/v4, controller-runtime, kustomize/v2 |
| **Language** | Go 1.22+ |

---

## 2. Kubebuilder Project Scaffold

### 2.1 Prerequisites

Before scaffolding, ensure the following are installed:

```bash
# Required tools
go version          # Go 1.22+
kubebuilder version # Kubebuilder 4.x
kubectl version     # For testing
kind version        # For local K8s cluster (optional)

# Install oapi-codegen for API client generation
go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest
```

### 2.2 Scaffold Commands

Execute these commands in order to create the project structure:

```bash
# Step 1: Create project directory
mkdir -p nebularr && cd nebularr

# Step 2: Initialize Kubebuilder project
kubebuilder init \
  --domain rinzler.cloud \
  --repo github.com/poiley/nebularr \
  --plugins=go/v4

# Step 3: Create Bundled Config CRDs (Simple Path)
kubebuilder create api --group arr --version v1alpha1 --kind RadarrConfig --resource --controller
kubebuilder create api --group arr --version v1alpha1 --kind SonarrConfig --resource --controller
kubebuilder create api --group arr --version v1alpha1 --kind LidarrConfig --resource --controller
kubebuilder create api --group arr --version v1alpha1 --kind ProwlarrConfig --resource --controller

# Step 4: Create Shared Resources
kubebuilder create api --group arr --version v1alpha1 --kind QualityTemplate --resource
kubebuilder create api --group arr --version v1alpha1 --kind NebularrDefaults --resource
kubebuilder create api --group arr --version v1alpha1 --kind ClusterNebularrDefaults --resource --namespaced=false

# Step 5: Create Granular Policies (Power User Path)
kubebuilder create api --group arr --version v1alpha1 --kind RadarrMediaPolicy --resource --controller
kubebuilder create api --group arr --version v1alpha1 --kind SonarrMediaPolicy --resource --controller
kubebuilder create api --group arr --version v1alpha1 --kind LidarrMusicPolicy --resource --controller
kubebuilder create api --group arr --version v1alpha1 --kind RadarrDownloadClientPolicy --resource --controller
kubebuilder create api --group arr --version v1alpha1 --kind RadarrIndexerPolicy --resource --controller
# Similar for Sonarr/Lidarr download client and indexer policies

# Step 6: Create Special CRDs
kubebuilder create api --group arr --version v1alpha1 --kind BazarrConfig --resource --controller

# Step 7: Generate manifests
make manifests generate
```

### 2.3 Post-Scaffold Directory Structure

After scaffolding, the project should have this structure:

```
nebularr/
├── api/
│   └── v1alpha1/
│       ├── radarrconfig_types.go         # Bundled config
│       ├── sonarrconfig_types.go
│       ├── lidarrconfig_types.go
│       ├── prowlarrconfig_types.go
│       ├── qualitytemplate_types.go      # Shared resources
│       ├── nebularrdefaults_types.go
│       ├── clusternebularrdefaults_types.go
│       ├── radarrmediapolicy_types.go    # Granular policies
│       ├── sonarrmediapolicy_types.go
│       ├── lidarrmusicpolicy_types.go
│       ├── bazarrconfig_types.go         # Special
│       ├── common_types.go               # Shared type definitions
│       ├── groupversion_info.go          # Auto-generated
│       └── zz_generated.deepcopy.go      # Auto-generated
├── cmd/
│   └── main.go
├── config/
│   ├── crd/
│   ├── rbac/
│   ├── manager/
│   └── samples/
├── internal/
│   ├── controller/
│   │   ├── radarrconfig_controller.go
│   │   ├── sonarrconfig_controller.go
│   │   └── ...
│   ├── compiler/
│   │   ├── compiler.go
│   │   ├── defaults.go
│   │   ├── merge.go
│   │   └── pruner.go
│   ├── ir/v1/
│   │   ├── ir.go
│   │   ├── connection.go
│   │   ├── quality.go
│   │   ├── download_client.go
│   │   ├── indexer.go
│   │   ├── naming.go
│   │   └── presets.go
│   ├── presets/
│   │   ├── video.go
│   │   ├── audio.go
│   │   ├── naming.go
│   │   └── override.go
│   ├── adapters/
│   │   ├── adapter.go
│   │   ├── registry.go
│   │   ├── radarr/
│   │   ├── sonarr/
│   │   ├── lidarr/
│   │   └── prowlarr/
│   ├── secrets/
│   │   └── resolver.go
│   ├── discovery/
│   │   └── apikey.go
│   └── telemetry/
│       └── metrics.go
├── Dockerfile
├── Makefile
├── PROJECT
├── go.mod
└── go.sum
```

---

## 3. Package Structure

### 3.1 Package Responsibilities

| Package | Purpose | Key Files |
|---------|---------|-----------|
| `internal/compiler` | Transforms CRDs into IR with preset expansion | `compiler.go`, `defaults.go`, `merge.go`, `pruner.go` |
| `internal/ir/v1` | Intermediate Representation types | `ir.go`, `quality.go`, `download_client.go`, `indexer.go`, `naming.go` |
| `internal/presets` | Built-in preset definitions and expansion | `video.go`, `audio.go`, `naming.go`, `override.go` |
| `internal/adapters` | Adapter interface and implementations | `adapter.go`, `registry.go` |
| `internal/adapters/radarr` | Radarr-specific adapter | `adapter.go`, `mapping.go`, `formats.go` |
| `internal/adapters/sonarr` | Sonarr-specific adapter | `adapter.go`, `mapping.go`, `formats.go` |
| `internal/adapters/lidarr` | Lidarr-specific adapter (API v1) | `adapter.go`, `mapping.go`, `releases.go` |
| `internal/adapters/prowlarr` | Prowlarr-specific adapter (API v1) | `adapter.go`, `indexers.go`, `applications.go` |
| `internal/secrets` | Secret resolution from K8s/config.xml | `resolver.go` |
| `internal/discovery` | API key auto-discovery | `apikey.go` |
| `internal/telemetry` | OTEL metrics and logging | `metrics.go` |

---

## 4. Implementation Phases

### Phase 1: Foundation (Weeks 1-2)

#### 1.1 Kubebuilder Scaffold
- [ ] Run scaffold commands from Section 2.2
- [ ] Verify project compiles: `make build`
- [ ] Verify tests pass: `make test`

#### 1.2 Common Types
- [ ] Create `api/v1alpha1/common_types.go` with shared types (see [CRDS.md Section 2](./CRDS.md#2-common-types))
- [ ] Implement `ConnectionSpec`, `SecretKeySelector`, `ReconciliationSpec`
- [ ] Implement `VideoQualitySpec`, `AudioQualitySpec`
- [ ] Implement `DownloadClientSpec`, `IndexersSpec`
- [ ] Implement `NamingSpec`

#### 1.3 Bundled Config CRDs
- [ ] Implement `RadarrConfigSpec` and `RadarrConfigStatus` (see [CRDS.md Section 3.1](./CRDS.md#31-radarrconfig))
- [ ] Implement `SonarrConfigSpec` and `SonarrConfigStatus`
- [ ] Implement `LidarrConfigSpec` and `LidarrConfigStatus`
- [ ] Implement `ProwlarrConfigSpec` and `ProwlarrConfigStatus`
- [ ] Add kubebuilder validation markers
- [ ] Regenerate: `make manifests generate`

#### 1.4 Shared Resources
- [ ] Implement `QualityTemplateSpec`
- [ ] Implement `NebularrDefaultsSpec`
- [ ] Implement `ClusterNebularrDefaultsSpec` (cluster-scoped)

**Deliverable:** CRDs install cleanly, validation works, `kubectl apply` succeeds.

---

### Phase 2: Presets & IR (Weeks 3-4)

#### 2.1 Presets Package
- [ ] Define `internal/presets/video.go` - Video quality presets (see [PRESETS.md Section 1](./PRESETS.md#1-video-quality-presets))
- [ ] Define `internal/presets/audio.go` - Audio quality presets
- [ ] Define `internal/presets/naming.go` - Naming presets
- [ ] Define `internal/presets/override.go` - Override application logic

#### 2.2 Intermediate Representation (IR)
- [ ] Define `internal/ir/v1/ir.go` - IR envelope type
- [ ] Define `internal/ir/v1/quality.go` - VideoQualityIR, AudioQualityIR
- [ ] Define `internal/ir/v1/download_client.go` - DownloadClientIR
- [ ] Define `internal/ir/v1/indexer.go` - IndexerIR, ProwlarrRefIR
- [ ] Define `internal/ir/v1/naming.go` - NamingIR (app-specific variants)
- [ ] Define `internal/ir/v1/connection.go` - ConnectionIR
- [ ] IR must support "unrealized" states for capability pruning
- [ ] IR must be serializable (for state hashing)

#### 2.3 Policy Compiler
- [ ] Create `internal/compiler/compiler.go` - Main compiler interface
- [ ] Create `internal/compiler/defaults.go` - Default merging logic
- [ ] Create `internal/compiler/merge.go` - Cluster/Namespace/Config merge rules
- [ ] Create `internal/compiler/pruner.go` - Prune unsupported features based on capabilities
- [ ] Compiler must be deterministic (same input = same output)

**Deliverable:** Presets expand correctly, IR generation works.

---

### Phase 3: Radarr Adapter (Weeks 5-6)

#### 3.1 Adapter Interface
- [ ] Define `internal/adapters/adapter.go` - See Section 6.1 for interface definition
- [ ] Define `internal/adapters/registry.go` - Adapter registration

#### 3.2 Radarr API Client
- [ ] Create `internal/adapters/radarr/client/` package
- [ ] Generate client from Radarr OpenAPI spec (API v3)
- [ ] OpenAPI spec URL: `https://raw.githubusercontent.com/Radarr/Radarr/develop/src/Radarr.Api.V3/openapi.json`

#### 3.3 Radarr Adapter Implementation
- [ ] Create `internal/adapters/radarr/adapter.go` - Main adapter
- [ ] Create `internal/adapters/radarr/mapping.go` - Quality tier mappings (see [RADARR.md](./RADARR.md))
- [ ] Create `internal/adapters/radarr/formats.go` - Custom format mappings
- [ ] Create `internal/adapters/radarr/clients.go` - Download client mappings
- [ ] Create `internal/adapters/radarr/indexers.go` - Indexer mappings
- [ ] Create `internal/adapters/radarr/naming.go` - Naming config
- [ ] Create `internal/adapters/radarr/tags.go` - Ownership tagging
- [ ] Create `internal/adapters/radarr/diff.go` - Diff logic
- [ ] Create `internal/adapters/radarr/apply.go` - Apply logic

**Deliverable:** Radarr adapter can discover, diff, and apply changes.

---

### Phase 4: Reconciliation (Week 7)

#### 4.1 Core Reconciliation Loop
- [ ] Modify `internal/controller/radarrconfig_controller.go`
- [ ] Reconciliation flow:
  1. Load ClusterNebularrDefaults (if exists)
  2. Load NebularrDefaults (if exists)
  3. Merge defaults with RadarrConfig
  4. Check for granular policies (RadarrMediaPolicy, etc.) - overlay
  5. Resolve secrets (auto-discovery if not specified)
  6. Discover capabilities (periodic, cached)
  7. Compile intent to IR (expand presets, prune unsupported)
  8. Diff current state vs desired IR
  9. Apply changes
  10. Update RadarrConfig status
  11. Requeue after interval

#### 4.2 Auto-Discovery
- [ ] Create `internal/discovery/apikey.go` - Parse API key from config.xml
- [ ] Create `internal/secrets/resolver.go` - Secret resolution with fallback
- [ ] Implement download client type inference from name

#### 4.3 State Management
- [ ] State stored in CRD Status fields (no external state file)
- [ ] RadarrConfigStatus includes:
  - Last reconciliation timestamp
  - Managed resource IDs
  - Connection status
  - Capability cache (with TTL)

**Deliverable:** Full reconciliation loop works via controller-runtime.

---

### Phase 5: Additional Adapters (Weeks 8-10)

#### 5.1 Sonarr Adapter
- [ ] Generate Sonarr API client (API v3)
- [ ] Implement adapter following Radarr pattern
- [ ] See [SONARR.md](./SONARR.md) for mappings

#### 5.2 Lidarr Adapter
- [ ] Generate Lidarr API client (API v1)
- [ ] Implement adapter with audio quality IR
- [ ] See [LIDARR.md](./LIDARR.md) for mappings

#### 5.3 Prowlarr Adapter
- [ ] Generate Prowlarr API client (API v1)
- [ ] Implement indexer management
- [ ] Implement application auto-registration
- [ ] See [PROWLARR.md](./PROWLARR.md) for mappings

#### 5.4 Bazarr Generator
- [ ] Create `internal/adapters/bazarr/generator.go`
- [ ] Generate config.yaml for ConfigMap
- [ ] No API reconciliation (config pre-seeding only)

**Deliverable:** All adapters operational.

---

### Phase 6: Granular Policies (Week 11)

#### 6.1 Policy CRDs
- [ ] Implement `RadarrMediaPolicySpec` (see [CRDS.md Section 4](./CRDS.md#4-granular-policies))
- [ ] Implement `SonarrMediaPolicySpec`
- [ ] Implement `LidarrMusicPolicySpec`
- [ ] Implement download client policies
- [ ] Implement indexer policies

#### 6.2 Policy Controllers
- [ ] Policy controllers watch for changes
- [ ] Trigger reconciliation on parent config
- [ ] Policies override config sections

**Deliverable:** Granular policies work as overlays.

---

### Phase 7: Observability & Polish (Week 12)

#### 7.1 OTEL Metrics
- [ ] Create `internal/telemetry/metrics.go`
- [ ] Implement metrics (see OPERATIONS.md Section 6.1)

#### 7.2 Documentation & Deployment
- [ ] Update `config/manager/` with resource limits, health probes
- [ ] Create Helm chart in `charts/nebularr/`
- [ ] Create sample CRs in `config/samples/`

**Deliverable:** Production-ready observability, deployment manifests.

---

## 5. Intermediate Representation (IR) Types

The IR is the internal representation that the Policy Compiler produces and Adapters consume. It is domain-based, versioned, and must not import any adapter-specific types.

### 5.1 IR Envelope

```go
// internal/ir/v1/ir.go

package v1

import "time"

// IR is the top-level intermediate representation for an *arr app
type IR struct {
    // Version of this IR schema
    Version string `json:"version"`

    // GeneratedAt is when this IR was compiled
    GeneratedAt time.Time `json:"generatedAt"`

    // SourceHash is a hash of the intent that produced this IR
    // Used for drift detection (if hash unchanged, skip reconciliation)
    SourceHash string `json:"sourceHash"`

    // App identifies which app this IR is for: radarr, sonarr, lidarr, prowlarr
    App string `json:"app"`

    // Connection details for the app
    Connection *ConnectionIR `json:"connection,omitempty"`

    // Quality configuration (Video for Radarr/Sonarr, Audio for Lidarr)
    Quality *QualityIR `json:"quality,omitempty"`

    // DownloadClients configuration
    DownloadClients []DownloadClientIR `json:"downloadClients,omitempty"`

    // Indexers configuration (or ProwlarrRef)
    Indexers *IndexersIR `json:"indexers,omitempty"`

    // Naming configuration
    Naming *NamingIR `json:"naming,omitempty"`

    // RootFolders configuration
    RootFolders []RootFolderIR `json:"rootFolders,omitempty"`

    // Unrealized tracks features that could not be compiled
    // (due to missing capabilities)
    Unrealized []UnrealizedFeature `json:"unrealized,omitempty"`
}

// UnrealizedFeature represents something the user requested
// that cannot be realized given current capabilities
type UnrealizedFeature struct {
    Feature string `json:"feature"` // e.g., "format:dolby-vision"
    Reason  string `json:"reason"`  // e.g., "not supported by service version"
}
```

### 5.2 Connection IR

```go
// internal/ir/v1/connection.go

package v1

// ConnectionIR holds resolved connection details
type ConnectionIR struct {
    URL                string `json:"url"`
    APIKey             string `json:"apiKey"` // Resolved from secret or auto-discovery
    InsecureSkipVerify bool   `json:"insecureSkipVerify,omitempty"`
}
```

### 5.3 Quality IR

```go
// internal/ir/v1/quality.go

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
    Name                string         `json:"name"`
    IncludeWhenRenaming bool           `json:"includeWhenRenaming,omitempty"`
    Specifications      []FormatSpecIR `json:"specifications"`
}

// FormatSpecIR represents a single format matching rule
type FormatSpecIR struct {
    Type     string `json:"type"`     // ReleaseTitleSpecification, SourceSpecification, etc.
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
```

### 5.4 Download Client IR

```go
// internal/ir/v1/download_client.go

package v1

// DownloadClientIR represents a download client configuration
type DownloadClientIR struct {
    // Name is the client name (generated: "nebularr-{name}")
    Name string `json:"name"`

    // Protocol is "torrent" or "usenet"
    Protocol string `json:"protocol"`

    // Implementation is the client type (qbittorrent, transmission, etc.)
    Implementation string `json:"implementation"`

    // Enable toggles the client
    Enable bool `json:"enable"`

    // Priority affects selection order (higher = preferred)
    Priority int `json:"priority"`

    // RemoveCompletedDownloads after import
    RemoveCompletedDownloads bool `json:"removeCompletedDownloads,omitempty"`

    // RemoveFailedDownloads on failure
    RemoveFailedDownloads bool `json:"removeFailedDownloads,omitempty"`

    // Connection details
    Host     string `json:"host"`
    Port     int    `json:"port"`
    UseTLS   bool   `json:"useTls,omitempty"`
    Username string `json:"username,omitempty"`
    Password string `json:"password,omitempty"` // Resolved from K8s Secret

    // Category for downloads (app-specific field names at adapter level)
    // For Radarr: movieCategory, for Sonarr: tvCategory, for Lidarr: musicCategory
    Category string `json:"category,omitempty"`

    // Directory override
    Directory string `json:"directory,omitempty"`
}
```

### 5.5 Indexer IR

```go
// internal/ir/v1/indexer.go

package v1

// IndexersIR represents indexer configuration
type IndexersIR struct {
    // ProwlarrRef delegates indexer management to Prowlarr
    // Mutually exclusive with Direct
    ProwlarrRef *ProwlarrRefIR `json:"prowlarrRef,omitempty"`

    // Direct configures indexers directly
    Direct []IndexerIR `json:"direct,omitempty"`
}

// ProwlarrRefIR references a Prowlarr instance
type ProwlarrRefIR struct {
    // ConfigName is the ProwlarrConfig name
    ConfigName string `json:"configName"`

    // AutoRegister this app with Prowlarr
    AutoRegister bool `json:"autoRegister"`

    // Include filters which indexers to sync (empty = all)
    Include []string `json:"include,omitempty"`

    // Exclude filters out specific indexers
    Exclude []string `json:"exclude,omitempty"`
}

// IndexerIR represents a direct indexer configuration
type IndexerIR struct {
    // Name is the indexer name (generated: "nebularr-{name}")
    Name string `json:"name"`

    // Protocol is "torrent" or "usenet"
    Protocol string `json:"protocol"`

    // Implementation is the indexer protocol (Torznab, Newznab)
    Implementation string `json:"implementation"`

    // Enable toggles the indexer
    Enable bool `json:"enable"`

    // Priority affects search order (lower = searched first in most apps)
    Priority int `json:"priority"`

    // URL is the indexer base URL
    URL string `json:"url"`

    // APIKey for authentication (resolved from K8s Secret)
    APIKey string `json:"apiKey,omitempty"`

    // Categories to search (numeric IDs)
    Categories []int `json:"categories,omitempty"`

    // MinimumSeeders for torrents
    MinimumSeeders int `json:"minimumSeeders,omitempty"`

    // SeedRatio target
    SeedRatio float64 `json:"seedRatio,omitempty"`

    // SeedTimeMinutes minimum seed time
    SeedTimeMinutes int `json:"seedTimeMinutes,omitempty"`

    // RSS/Search toggles
    EnableRss               bool `json:"enableRss"`
    EnableAutomaticSearch   bool `json:"enableAutomaticSearch"`
    EnableInteractiveSearch bool `json:"enableInteractiveSearch"`
}
```

### 5.6 Naming IR

```go
// internal/ir/v1/naming.go

package v1

// NamingIR represents naming configuration (union type for app-specific)
type NamingIR struct {
    // Radarr naming
    Radarr *RadarrNamingIR `json:"radarr,omitempty"`

    // Sonarr naming
    Sonarr *SonarrNamingIR `json:"sonarr,omitempty"`

    // Lidarr naming
    Lidarr *LidarrNamingIR `json:"lidarr,omitempty"`
}

// RadarrNamingIR for Radarr naming config
type RadarrNamingIR struct {
    RenameMovies             bool   `json:"renameMovies"`
    ReplaceIllegalCharacters bool   `json:"replaceIllegalCharacters"`
    ColonReplacementFormat   int    `json:"colonReplacementFormat"` // 0=delete, 1=dash, 2=space, 4=smart
    StandardMovieFormat      string `json:"standardMovieFormat"`
    MovieFolderFormat        string `json:"movieFolderFormat"`
}

// SonarrNamingIR for Sonarr naming config
type SonarrNamingIR struct {
    RenameEpisodes           bool   `json:"renameEpisodes"`
    ReplaceIllegalCharacters bool   `json:"replaceIllegalCharacters"`
    ColonReplacementFormat   int    `json:"colonReplacementFormat"`
    StandardEpisodeFormat    string `json:"standardEpisodeFormat"`
    DailyEpisodeFormat       string `json:"dailyEpisodeFormat"`
    AnimeEpisodeFormat       string `json:"animeEpisodeFormat"`
    SeriesFolderFormat       string `json:"seriesFolderFormat"`
    SeasonFolderFormat       string `json:"seasonFolderFormat"`
    SpecialsFolderFormat     string `json:"specialsFolderFormat"`
    MultiEpisodeStyle        int    `json:"multiEpisodeStyle"`
}

// LidarrNamingIR for Lidarr naming config
type LidarrNamingIR struct {
    RenameTracks             bool   `json:"renameTracks"`
    ReplaceIllegalCharacters bool   `json:"replaceIllegalCharacters"`
    ColonReplacementFormat   int    `json:"colonReplacementFormat"`
    StandardTrackFormat      string `json:"standardTrackFormat"`
    MultiDiscTrackFormat     string `json:"multiDiscTrackFormat"`
    ArtistFolderFormat       string `json:"artistFolderFormat"`
    AlbumFolderFormat        string `json:"albumFolderFormat"`
}
```

### 5.7 Root Folder IR

```go
// internal/ir/v1/root_folder.go

package v1

// RootFolderIR represents a root folder
type RootFolderIR struct {
    Path string `json:"path"`

    // Lidarr-specific fields
    Name           string `json:"name,omitempty"`           // Lidarr only
    DefaultMonitor string `json:"defaultMonitor,omitempty"` // Lidarr: all, future, missing, etc.
}
```

---

## 6. Adapter Interface Contract

### 6.1 Interface Definition

```go
// internal/adapters/adapter.go

package adapters

import (
    "context"
    "time"

    irv1 "github.com/poiley/nebularr/internal/ir/v1"
)

// Adapter defines the contract for service adapters
type Adapter interface {
    // Name returns a unique identifier for this adapter
    Name() string

    // SupportedApp returns the app this adapter handles: radarr, sonarr, lidarr, prowlarr
    SupportedApp() string

    // Connect tests connectivity and retrieves service info
    Connect(ctx context.Context, conn *irv1.ConnectionIR) (*ServiceInfo, error)

    // Discover queries the service for its capabilities
    // MUST NOT return error for missing features (degrade gracefully)
    // MUST return error only for connection failures
    Discover(ctx context.Context, conn *irv1.ConnectionIR) (*Capabilities, error)

    // CurrentState retrieves the current managed state from the service
    // Only returns resources tagged as owned by Nebularr
    CurrentState(ctx context.Context, conn *irv1.ConnectionIR) (*irv1.IR, error)

    // Diff computes the changes needed to move from current to desired state
    // MUST be deterministic (same inputs = same outputs)
    Diff(current, desired *irv1.IR, caps *Capabilities) (*ChangeSet, error)

    // Apply executes the changes against the service
    // MUST be idempotent (safe to retry)
    // MUST be fail-soft (continue on partial failures)
    // MUST tag created resources with ownership marker
    Apply(ctx context.Context, conn *irv1.ConnectionIR, changes *ChangeSet) (*ApplyResult, error)
}

// ServiceInfo describes the connected service
type ServiceInfo struct {
    Version   string
    StartTime time.Time
}

// Capabilities describes what features a service supports
type Capabilities struct {
    DiscoveredAt time.Time

    // Video/Quality capabilities (Radarr/Sonarr)
    Resolutions       []string           // e.g., ["2160p", "1080p", "720p"]
    Sources           []string           // e.g., ["bluray", "webdl", "hdtv"]
    CustomFormatSpecs []CustomFormatSpec // Available custom format specification types

    // Audio capabilities (Lidarr)
    AudioTiers []string // e.g., ["lossless-hires", "lossless", "lossy-high"]

    // Download client capabilities
    DownloadClientTypes []string

    // Indexer capabilities
    IndexerTypes []string
}

// CustomFormatSpec describes an available custom format specification type
type CustomFormatSpec struct {
    Name           string // e.g., "ReleaseTitleSpecification"
    Implementation string
}

// ChangeSet describes changes to apply
type ChangeSet struct {
    Creates []Change
    Updates []Change
    Deletes []Change
}

type Change struct {
    ResourceType string      // e.g., "QualityProfile", "CustomFormat"
    Name         string      // Human-readable name
    ID           *int        // Service-specific ID (nil for creates)
    Payload      interface{} // Service-specific payload
}

// ApplyResult describes the outcome of applying changes
type ApplyResult struct {
    Applied int
    Failed  int
    Skipped int
    Errors  []ApplyError
}

type ApplyError struct {
    Change Change
    Error  error
}
```

### 6.2 Adapter Registration

```go
// internal/adapters/registry.go

package adapters

var registry = map[string]Adapter{}

func Register(a Adapter) {
    registry[a.SupportedApp()] = a
}

func Get(app string) (Adapter, bool) {
    a, ok := registry[app]
    return a, ok
}

func init() {
    Register(&radarr.Adapter{})
    Register(&sonarr.Adapter{})
    Register(&lidarr.Adapter{})
    Register(&prowlarr.Adapter{})
}
```

---

## 7. Compiler Interface

### 7.1 Compiler Definition

```go
// internal/compiler/compiler.go

package compiler

import (
    "context"

    arrv1alpha1 "github.com/poiley/nebularr/api/v1alpha1"
    "github.com/poiley/nebularr/internal/adapters"
    irv1 "github.com/poiley/nebularr/internal/ir/v1"
)

// Compiler transforms CRD intent into IR
type Compiler struct {
    presets *PresetExpander
    merger  *DefaultsMerger
}

// CompileInput holds all inputs for compilation
type CompileInput struct {
    // ClusterDefaults (optional)
    ClusterDefaults *arrv1alpha1.ClusterNebularrDefaults

    // NamespaceDefaults (optional)
    NamespaceDefaults *arrv1alpha1.NebularrDefaults

    // Config is the bundled config (RadarrConfig, SonarrConfig, etc.)
    Config interface{}

    // Policies are granular policy overlays (optional)
    Policies []interface{}

    // Capabilities for pruning
    Capabilities *adapters.Capabilities

    // ResolvedSecrets maps secret references to resolved values
    ResolvedSecrets map[string]string
}

// Compile transforms CRD intent into IR
func (c *Compiler) Compile(ctx context.Context, input CompileInput) (*irv1.IR, error) {
    // 1. Merge defaults: Cluster < Namespace < Config
    merged := c.merger.Merge(input.ClusterDefaults, input.NamespaceDefaults, input.Config)

    // 2. Apply policy overlays
    for _, policy := range input.Policies {
        merged = c.merger.ApplyPolicy(merged, policy)
    }

    // 3. Expand presets (e.g., "4k-hdr" -> full VideoQualityIR with tiers, formats, etc.)
    //    IMPORTANT: This happens AFTER merge so user overrides (exclude/preferAdditional)
    //    apply to the expanded preset, not the preset name itself.
    expanded := c.presets.Expand(merged)

    // 4. Compile to IR
    ir := c.toIR(expanded, input.ResolvedSecrets)

    // 5. Prune features not supported by capabilities
    ir, unrealized := c.pruneUnsupported(ir, input.Capabilities)
    ir.Unrealized = unrealized

    // 6. Generate source hash
    ir.SourceHash = c.hashInput(input)

    return ir, nil
}
```

### 7.2 Defaults Merger

```go
// internal/compiler/merge.go

package compiler

import (
    arrv1alpha1 "github.com/poiley/nebularr/api/v1alpha1"
)

// DefaultsMerger handles the merge hierarchy
type DefaultsMerger struct{}

// Merge applies the merge hierarchy:
// ClusterDefaults < NamespaceDefaults < BundledConfig
func (m *DefaultsMerger) Merge(
    cluster *arrv1alpha1.ClusterNebularrDefaults,
    namespace *arrv1alpha1.NebularrDefaults,
    config interface{},
) interface{} {
    // Start with cluster defaults
    result := m.cloneConfig(config)

    // Apply cluster defaults (lowest priority)
    if cluster != nil {
        result = m.applyDefaults(result, cluster.Spec)
    }

    // Apply namespace defaults (overrides cluster)
    if namespace != nil {
        result = m.applyDefaults(result, namespace.Spec)
    }

    // Config values override defaults (already in result)
    return result
}

// ApplyPolicy overlays a granular policy on top of config
// Policy fields override config fields where specified
func (m *DefaultsMerger) ApplyPolicy(config interface{}, policy interface{}) interface{} {
    // Implementation: deep merge policy onto config
    return config
}
```

---

## 8. State Management

State is stored in CRD Status fields - no external state files.

### 8.1 RadarrConfig Status

```go
// RadarrConfigStatus stores reconciliation state
type RadarrConfigStatus struct {
    // Connection state
    Connected        bool         `json:"connected,omitempty"`
    LastConnected    *metav1.Time `json:"lastConnected,omitempty"`
    ServiceVersion   string       `json:"serviceVersion,omitempty"`

    // Reconciliation state
    LastReconcile       *metav1.Time `json:"lastReconcile,omitempty"`
    LastReconcileStatus string       `json:"lastReconcileStatus,omitempty"` // Success, PartialSuccess, Failed
    LastAppliedIRHash   string       `json:"lastAppliedIRHash,omitempty"`

    // Managed resources (for safe deletion tracking)
    ManagedResources ManagedResources `json:"managedResources,omitempty"`

    // Capability cache (stored in status to survive restarts)
    Capabilities *CachedCapabilities `json:"capabilities,omitempty"`

    // Unrealized features (from capability pruning)
    UnrealizedFeatures []string `json:"unrealizedFeatures,omitempty"`

    // Standard conditions
    Conditions []metav1.Condition `json:"conditions,omitempty"`
}

type ManagedResources struct {
    QualityProfileID   *int  `json:"qualityProfileId,omitempty"`
    CustomFormatIDs    []int `json:"customFormatIds,omitempty"`
    DownloadClientIDs  []int `json:"downloadClientIds,omitempty"`
    IndexerIDs         []int `json:"indexerIds,omitempty"`
    TagID              *int  `json:"tagId,omitempty"` // Ownership tag
}

type CachedCapabilities struct {
    DiscoveredAt        metav1.Time `json:"discoveredAt"`
    Resolutions         []string    `json:"resolutions,omitempty"`
    Sources             []string    `json:"sources,omitempty"`
    DownloadClientTypes []string    `json:"downloadClientTypes,omitempty"`
    IndexerTypes        []string    `json:"indexerTypes,omitempty"`
}
```

---

## 9. Testing Strategy

### 9.1 Unit Tests

| Package | Test Focus | Mocking |
|---------|------------|---------|
| `internal/presets` | Preset expansion, overrides | None |
| `internal/compiler` | Merge logic, IR generation | Mock capabilities |
| `internal/ir/v1` | IR serialization, hashing | None |
| `internal/adapters/radarr` | Diff logic, mappings | Mock API responses |

### 9.2 Integration Tests

Use envtest for controller testing, httptest for API mocking.

### 9.3 E2E Tests

Use testcontainers for real *arr instances (optional, slow).

---

## 10. Success Criteria

- [ ] CRDs install and validate correctly
- [ ] Presets expand correctly with overrides
- [ ] Defaults merge correctly (Cluster < Namespace < Config < Policy)
- [ ] Auto-discovery works for API keys and download client types
- [ ] Radarr adapter discovers capabilities
- [ ] Diff produces minimal, correct change sets
- [ ] Apply is idempotent (running twice = no changes second time)
- [ ] Owned resources are tagged and trackable
- [ ] Unowned resources are never modified
- [ ] Granular policies override bundled config sections
- [ ] Metrics are exposed and meaningful
- [ ] Logs trace CRD -> IR -> adapter flow

---

## 11. Related Documents

- [README](./README.md) - Build order, file mapping (start here)
- [CRDS](./CRDS.md) - CRD schemas and validation
- [PRESETS](./PRESETS.md) - Quality and naming preset definitions
- [DESIGN](./DESIGN.md) - Core philosophy and constraints
- [OPERATIONS](./OPERATIONS.md) - Secrets, multi-instance, errors, testing
- [RADARR](./RADARR.md) - Quality/format mappings for Radarr adapter
- [SONARR](./SONARR.md) - Quality/format mappings for Sonarr adapter
- [LIDARR](./LIDARR.md) - Audio quality mappings for Lidarr adapter
- [PROWLARR](./PROWLARR.md) - Indexer management for Prowlarr adapter
