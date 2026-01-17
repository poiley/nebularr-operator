# Nebularr - CRD Specifications

> **For coding agents:** Start with [README.md](./README.md) for build order. This document contains CRD type definitions to copy.
>
> **Related:** [README](./README.md) | [PRESETS](./PRESETS.md) | [TYPES](./TYPES.md) | [DESIGN](./DESIGN.md)

---

## 1. Overview

### 1.1 CRD Hierarchy

Nebularr provides two paths for configuration:

```
CRDs
├── Bundled Configs (Simple Path - Start Here)
│   ├── RadarrConfig      # All-in-one for Radarr
│   ├── SonarrConfig      # All-in-one for Sonarr
│   ├── LidarrConfig      # All-in-one for Lidarr (music)
│   └── ProwlarrConfig    # All-in-one for Prowlarr
│
├── Granular Policies (Power User Path)
│   ├── RadarrMediaPolicy / SonarrMediaPolicy / LidarrMusicPolicy
│   ├── RadarrDownloadClientPolicy / SonarrDownloadClientPolicy / ...
│   └── RadarrIndexerPolicy / SonarrIndexerPolicy / ...
│
├── Shared Resources
│   ├── QualityTemplate        # Reusable quality configurations
│   ├── NebularrDefaults       # Namespace-level defaults
│   └── ClusterNebularrDefaults # Cluster-level defaults
│
└── Special
    └── BazarrConfig           # ConfigMap generator for Bazarr
```

### 1.2 Design Principles

| Principle | Implementation |
|-----------|----------------|
| Simple by Default | Bundled configs with presets for quick setup |
| Full Control Available | Granular policies for advanced users |
| Full Parity | Bundled configs support ALL features available in granular policies (not a subset) |
| Per-App Type Safety | Separate CRDs per app (RadarrConfig vs SonarrConfig) |
| Intent over Implementation | Users specify "4K HDR", not quality profile IDs |
| Presets with Overrides | Built-in presets, customizable via exclude/prefer |
| Auto-Discovery | Secrets, types, Prowlarr registration |

### 1.3 Merge Rules

When multiple resources configure the same app:

1. **ClusterNebularrDefaults** - Lowest priority (cluster-wide base)
2. **NebularrDefaults** - Namespace-level defaults override cluster
3. **Bundled Config** - App config overrides defaults
4. **Granular Policies** - Highest priority, override bundled config sections

Example: If `RadarrConfig` specifies quality and `RadarrMediaPolicy` also exists, the policy's quality settings win.

---

## 2. Common Types

### 2.1 ConnectionSpec

Used by all bundled configs for app connection:

```go
// ConnectionSpec defines how to connect to an *arr service
type ConnectionSpec struct {
    // URL is the base URL of the service (e.g., http://radarr:7878)
    // +kubebuilder:validation:Required
    // +kubebuilder:validation:Pattern=`^https?://`
    URL string `json:"url"`

    // APIKeySecretRef references a Secret containing the API key.
    // If not specified, auto-discovery is attempted.
    // +optional
    APIKeySecretRef *SecretKeySelector `json:"apiKeySecretRef,omitempty"`

    // ConfigPath is the path to config.xml for API key auto-discovery.
    // Only used if APIKeySecretRef is not specified.
    // Defaults to /{app}-config/config.xml
    // +optional
    ConfigPath string `json:"configPath,omitempty"`

    // InsecureSkipVerify disables TLS certificate verification.
    // +optional
    InsecureSkipVerify bool `json:"insecureSkipVerify,omitempty"`

    // Timeout specifies the connection timeout.
    // +optional
    // +kubebuilder:default="30s"
    Timeout *metav1.Duration `json:"timeout,omitempty"`
}

// SecretKeySelector selects a key from a Kubernetes Secret
type SecretKeySelector struct {
    // Name is the name of the Secret in the same namespace.
    // +kubebuilder:validation:Required
    Name string `json:"name"`

    // Key is the key within the Secret.
    // +optional
    // +kubebuilder:default="apiKey"
    Key string `json:"key,omitempty"`
}
```

### 2.2 VideoQualitySpec

Used by Radarr and Sonarr:

```go
// VideoQualitySpec defines video quality preferences
type VideoQualitySpec struct {
    // Preset is a built-in quality configuration.
    // See PRESETS.md for available presets.
    // If not specified, defaults to "balanced".
    // +optional
    // +kubebuilder:validation:Enum=4k-hdr;4k-sdr;1080p-quality;1080p-streaming;720p;balanced;any;storage-optimized
    Preset string `json:"preset,omitempty"`

    // TemplateRef references a QualityTemplate for custom presets.
    // Mutually exclusive with Preset.
    // +optional
    TemplateRef *LocalObjectReference `json:"templateRef,omitempty"`

    // Exclude removes formats/features from the preset.
    // +optional
    Exclude []string `json:"exclude,omitempty"`

    // PreferAdditional adds formats to the preferred list.
    // +optional
    PreferAdditional []string `json:"preferAdditional,omitempty"`

    // RejectAdditional adds formats to the rejected list.
    // +optional
    RejectAdditional []string `json:"rejectAdditional,omitempty"`

    // --- Full manual control (overrides preset entirely if specified) ---

    // Tiers defines quality tiers in order of preference.
    // If specified, preset is ignored.
    // +optional
    Tiers []VideoQualityTier `json:"tiers,omitempty"`

    // UpgradeUntil defines the quality to upgrade until.
    // +optional
    UpgradeUntil *VideoQualityTier `json:"upgradeUntil,omitempty"`

    // PreferredFormats lists formats with positive scoring.
    // +optional
    PreferredFormats []string `json:"preferredFormats,omitempty"`

    // RejectedFormats lists formats to reject.
    // +optional
    RejectedFormats []string `json:"rejectedFormats,omitempty"`
}

// VideoQualityTier represents a resolution + source combination
type VideoQualityTier struct {
    // Resolution: 2160p, 1080p, 720p, 480p
    // +kubebuilder:validation:Enum=2160p;1080p;720p;480p
    Resolution string `json:"resolution"`

    // Sources: bluray, remux, webdl, webrip, hdtv, dvd
    // +optional
    Sources []string `json:"sources,omitempty"`
}
```

### 2.3 AudioQualitySpec

Used by Lidarr:

```go
// AudioQualitySpec defines audio quality preferences
type AudioQualitySpec struct {
    // Preset is a built-in quality configuration.
    // See PRESETS.md for available presets.
    // +optional
    // +kubebuilder:validation:Enum=lossless-hires;lossless;high-quality;balanced;portable;any
    Preset string `json:"preset,omitempty"`

    // TemplateRef references a QualityTemplate.
    // +optional
    TemplateRef *LocalObjectReference `json:"templateRef,omitempty"`

    // Exclude removes tiers/formats from the preset.
    // +optional
    Exclude []string `json:"exclude,omitempty"`

    // PreferAdditional adds formats to preferred list.
    // +optional
    PreferAdditional []string `json:"preferAdditional,omitempty"`

    // --- Full manual control ---

    // Tiers defines quality tiers: lossless-hires, lossless, lossy-high, lossy-mid, lossy-low
    // +optional
    Tiers []string `json:"tiers,omitempty"`

    // UpgradeUntil defines the tier to upgrade until.
    // +optional
    UpgradeUntil string `json:"upgradeUntil,omitempty"`

    // PreferredFormats: flac, alac, mp3-320, aac-320, etc.
    // +optional
    PreferredFormats []string `json:"preferredFormats,omitempty"`
}
```

### 2.4 DownloadClientSpec

```go
// DownloadClientSpec defines a download client
type DownloadClientSpec struct {
    // Name is the display name for this client.
    // Also used for type inference if Type is not specified.
    // +kubebuilder:validation:Required
    Name string `json:"name"`

    // URL is the client URL (e.g., http://qbittorrent:8080)
    // +kubebuilder:validation:Required
    URL string `json:"url"`

    // Type is the client type. If not specified, inferred from Name.
    // +optional
    // +kubebuilder:validation:Enum=qbittorrent;transmission;deluge;rtorrent;nzbget;sabnzbd
    Type string `json:"type,omitempty"`

    // CredentialsSecretRef references username/password.
    // +optional
    CredentialsSecretRef *CredentialsSecretRef `json:"credentialsSecretRef,omitempty"`

    // Category for downloads. Defaults to app name (e.g., "radarr").
    // +optional
    Category string `json:"category,omitempty"`

    // Priority affects client selection (higher = preferred).
    // +optional
    // +kubebuilder:validation:Minimum=1
    // +kubebuilder:validation:Maximum=100
    // +kubebuilder:default=50
    Priority int `json:"priority,omitempty"`

    // Enabled enables/disables this client.
    // +optional
    // +kubebuilder:default=true
    Enabled *bool `json:"enabled,omitempty"`
}

// CredentialsSecretRef references username/password from a Secret
type CredentialsSecretRef struct {
    Name        string `json:"name"`
    UsernameKey string `json:"usernameKey,omitempty"` // default: "username"
    PasswordKey string `json:"passwordKey,omitempty"` // default: "password"
}
```

### 2.5 IndexersSpec

```go
// IndexersSpec defines indexer configuration
type IndexersSpec struct {
    // ProwlarrRef delegates indexer management to Prowlarr.
    // Mutually exclusive with Direct.
    // +optional
    ProwlarrRef *ProwlarrRef `json:"prowlarrRef,omitempty"`

    // Direct configures indexers directly (no Prowlarr).
    // Mutually exclusive with ProwlarrRef.
    // +optional
    Direct []DirectIndexer `json:"direct,omitempty"`
}

// ProwlarrRef references a Prowlarr instance for indexer management
type ProwlarrRef struct {
    // Name is the name of a ProwlarrConfig in the same namespace.
    // +kubebuilder:validation:Required
    Name string `json:"name"`

    // AutoRegister automatically registers this app with Prowlarr.
    // +optional
    // +kubebuilder:default=true
    AutoRegister *bool `json:"autoRegister,omitempty"`

    // Include filters which Prowlarr indexers to sync.
    // If empty, all indexers are synced.
    // +optional
    Include []string `json:"include,omitempty"`

    // Exclude filters out specific Prowlarr indexers.
    // +optional
    Exclude []string `json:"exclude,omitempty"`
}

// DirectIndexer defines an indexer configured directly
type DirectIndexer struct {
    // Name is the display name.
    // +kubebuilder:validation:Required
    Name string `json:"name"`

    // URL is the indexer URL.
    // +kubebuilder:validation:Required
    URL string `json:"url"`

    // Type: torrent or usenet
    // +kubebuilder:validation:Enum=torrent;usenet
    // +kubebuilder:default=torrent
    Type string `json:"type,omitempty"`

    // APIKeySecretRef for indexer authentication.
    // +optional
    APIKeySecretRef *SecretKeySelector `json:"apiKeySecretRef,omitempty"`

    // Categories to search. Human-readable (e.g., "movies-hd") or numeric IDs.
    // +optional
    Categories []string `json:"categories,omitempty"`

    // Priority (1-50, lower = higher priority).
    // +optional
    // +kubebuilder:default=25
    Priority int `json:"priority,omitempty"`

    // Enabled enables/disables this indexer.
    // +optional
    // +kubebuilder:default=true
    Enabled *bool `json:"enabled,omitempty"`
}
```

### 2.6 NamingSpec

```go
// NamingSpec defines file/folder naming configuration
type NamingSpec struct {
    // Preset is a built-in naming configuration.
    // +optional
    // +kubebuilder:validation:Enum=plex-friendly;jellyfin-friendly;kodi-friendly;detailed;minimal;scene
    // +kubebuilder:default=plex-friendly
    Preset string `json:"preset,omitempty"`

    // --- Full manual control (overrides preset) ---

    // RenameMedia enables renaming (movies/episodes/tracks).
    // +optional
    RenameMedia *bool `json:"renameMedia,omitempty"`

    // StandardFormat is the format string for standard files.
    // +optional
    StandardFormat string `json:"standardFormat,omitempty"`

    // FolderFormat is the format string for folders.
    // +optional
    FolderFormat string `json:"folderFormat,omitempty"`
}
```

### 2.7 CustomFormatSpec

Custom formats allow fine-grained control over release quality preferences through pattern matching.
Used by Radarr, Sonarr, and Lidarr (v2.0+).

```go
// CustomFormatSpec defines a custom format for release matching
type CustomFormatSpec struct {
    // Name is the display name for this custom format.
    // +kubebuilder:validation:Required
    Name string `json:"name"`

    // IncludeWhenRenaming includes this format in renamed file names.
    // +optional
    // +kubebuilder:default=false
    IncludeWhenRenaming *bool `json:"includeWhenRenaming,omitempty"`

    // Score is the score to assign in quality profiles.
    // Positive scores prefer releases matching this format.
    // Negative scores reject releases matching this format.
    // +optional
    // +kubebuilder:default=0
    Score int `json:"score,omitempty"`

    // Specifications define the matching rules.
    // All specifications must match for the format to apply (AND logic).
    // +kubebuilder:validation:Required
    // +kubebuilder:validation:MinItems=1
    Specifications []CustomFormatSpecificationSpec `json:"specifications"`
}

// CustomFormatSpecificationSpec defines a single matching rule
type CustomFormatSpecificationSpec struct {
    // Name is the display name for this specification.
    // +kubebuilder:validation:Required
    Name string `json:"name"`

    // Type is the specification implementation type.
    // Common types:
    //   - ReleaseTitleSpecification: Match release title with regex
    //   - SourceSpecification: Match source type (bluray, webdl, etc.)
    //   - ResolutionSpecification: Match resolution (2160p, 1080p, etc.)
    //   - ReleaseGroupSpecification: Match release group with regex
    //   - QualityModifierSpecification: Match quality modifier (remux, etc.)
    // +kubebuilder:validation:Required
    Type string `json:"type"`

    // Negate inverts the match logic.
    // +optional
    // +kubebuilder:default=false
    Negate *bool `json:"negate,omitempty"`

    // Required makes this specification mandatory.
    // +optional
    // +kubebuilder:default=false
    Required *bool `json:"required,omitempty"`

    // Value is the specification value. Interpretation depends on Type:
    //   - ReleaseTitleSpecification: Regular expression pattern
    //   - SourceSpecification: Source name (bluray, webdl, webrip, hdtv, dvd)
    //   - ResolutionSpecification: Resolution (r2160p, r1080p, r720p, r480p)
    // +kubebuilder:validation:Required
    Value string `json:"value"`
}
```

#### Example: Custom Formats for 4K HDR

```yaml
apiVersion: arr.rinzler.cloud/v1alpha1
kind: RadarrConfig
metadata:
  name: radarr-4k
  namespace: media
spec:
  connection:
    url: http://radarr:7878
  
  quality:
    preset: "uhd-hdr"
  
  customFormats:
    # Prefer Dolby Vision releases
    - name: DV
      score: 1500
      specifications:
        - name: Dolby Vision
          type: ReleaseTitleSpecification
          value: "\\b(dv|dovi|dolby[ .]?vision)\\b"
          required: true
    
    # Prefer HDR10+ releases
    - name: HDR10Plus
      score: 800
      specifications:
        - name: HDR10+
          type: ReleaseTitleSpecification
          value: "\\bHDR10(\\+|Plus)\\b"
          required: true
    
    # Prefer HEVC/x265 (smaller files)
    - name: x265
      score: 100
      specifications:
        - name: x265/HEVC
          type: ReleaseTitleSpecification
          value: "[xh]\\.?265|hevc"
          required: true
        - name: Not 2160p
          type: ResolutionSpecification
          value: "r2160p"
          negate: true
    
    # Reject 3D releases
    - name: 3D
      score: -10000
      specifications:
        - name: 3D
          type: ReleaseTitleSpecification
          value: "\\b3D\\b"
          required: true
    
    # Reject CAM/TS releases
    - name: LowQuality
      score: -10000
      specifications:
        - name: CAM/TS
          type: ReleaseTitleSpecification
          value: "\\b(CAM|HDCAM|TS|TELESYNC|TELECINE)\\b"
          required: true
```

#### Example: Custom Formats for TV Shows

```yaml
apiVersion: arr.rinzler.cloud/v1alpha1
kind: SonarrConfig
metadata:
  name: sonarr
  namespace: media
spec:
  connection:
    url: http://sonarr:8989
  
  customFormats:
    # Prefer WEB-DL over WEBRip
    - name: WEB-DL
      score: 100
      specifications:
        - name: WEB-DL
          type: SourceSpecification
          value: "webdl"
          required: true
    
    # Prefer scene releases
    - name: Scene
      score: 50
      includeWhenRenaming: true
      specifications:
        - name: Scene Groups
          type: ReleaseGroupSpecification
          value: "\\b(NTb|NTG|KiNGS|FLUX|EDITH)\\b"
    
    # Prefer AMZN releases
    - name: AMZN
      score: 75
      specifications:
        - name: Amazon
          type: ReleaseTitleSpecification
          value: "\\bAMZN\\b"
          required: true
```

### 2.8 DelayProfileSpec

Delay profiles control when downloads should start based on protocol preferences and timing delays. They're useful for preferring one protocol over another or waiting for better quality releases.

**Supported by**: RadarrConfig, SonarrConfig, LidarrConfig

```go
// api/v1alpha1/common_types.go

type DelayProfileSpec struct {
    // Name is a display name for this delay profile (used for identification only).
    // +kubebuilder:validation:Required
    Name string `json:"name"`

    // PreferredProtocol specifies which protocol to prefer when both are available.
    // +optional
    // +kubebuilder:validation:Enum=usenet;torrent
    // +kubebuilder:default=usenet
    PreferredProtocol string `json:"preferredProtocol,omitempty"`

    // UsenetDelay is the delay in minutes before downloading from Usenet.
    // Set to 0 for no delay.
    // +optional
    // +kubebuilder:default=0
    // +kubebuilder:validation:Minimum=0
    UsenetDelay int `json:"usenetDelay,omitempty"`

    // TorrentDelay is the delay in minutes before downloading from torrents.
    // Set to 0 for no delay.
    // +optional
    // +kubebuilder:default=0
    // +kubebuilder:validation:Minimum=0
    TorrentDelay int `json:"torrentDelay,omitempty"`

    // EnableUsenet enables/disables Usenet for this profile.
    // +optional
    // +kubebuilder:default=true
    EnableUsenet *bool `json:"enableUsenet,omitempty"`

    // EnableTorrent enables/disables torrents for this profile.
    // +optional
    // +kubebuilder:default=true
    EnableTorrent *bool `json:"enableTorrent,omitempty"`

    // BypassIfHighestQuality bypasses the delay if the release is at or above
    // the cutoff quality defined in the quality profile.
    // +optional
    // +kubebuilder:default=false
    BypassIfHighestQuality *bool `json:"bypassIfHighestQuality,omitempty"`

    // BypassIfAboveCustomFormatScore bypasses the delay if the release's
    // custom format score is at or above MinimumCustomFormatScore.
    // +optional
    // +kubebuilder:default=false
    BypassIfAboveCustomFormatScore *bool `json:"bypassIfAboveCustomFormatScore,omitempty"`

    // MinimumCustomFormatScore is the minimum custom format score required
    // to bypass the delay when BypassIfAboveCustomFormatScore is enabled.
    // +optional
    // +kubebuilder:default=0
    MinimumCustomFormatScore int `json:"minimumCustomFormatScore,omitempty"`

    // Tags restricts this delay profile to items with matching tags.
    // If empty, the profile applies to all items.
    // +optional
    Tags []string `json:"tags,omitempty"`

    // Order determines the priority of this profile (lower = higher priority).
    // If not specified, profiles are ordered by their position in the array.
    // +optional
    Order *int `json:"order,omitempty"`
}
```

#### Field Descriptions

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `name` | string | (required) | Display name for identification |
| `preferredProtocol` | string | `usenet` | Which protocol to prefer: `usenet` or `torrent` |
| `usenetDelay` | int | `0` | Minutes to wait before downloading from Usenet |
| `torrentDelay` | int | `0` | Minutes to wait before downloading from torrents |
| `enableUsenet` | bool | `true` | Whether Usenet downloads are enabled |
| `enableTorrent` | bool | `true` | Whether torrent downloads are enabled |
| `bypassIfHighestQuality` | bool | `false` | Skip delay if release meets quality cutoff |
| `bypassIfAboveCustomFormatScore` | bool | `false` | Skip delay if custom format score threshold met |
| `minimumCustomFormatScore` | int | `0` | Score threshold for bypass (when enabled) |
| `tags` | []string | `[]` | Restrict profile to items with these tags |
| `order` | int | (position) | Priority order (lower = higher priority) |

#### Example: Prefer Usenet with Torrent Fallback

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
    # Default profile: prefer Usenet, wait 120 min for torrents
    - name: Default
      preferredProtocol: usenet
      usenetDelay: 0
      torrentDelay: 120
      bypassIfHighestQuality: true
    
    # For 4K content: longer delays for better releases
    - name: 4K Releases
      preferredProtocol: usenet
      usenetDelay: 60
      torrentDelay: 240
      bypassIfAboveCustomFormatScore: true
      minimumCustomFormatScore: 1000
      tags:
        - 4k
```

#### Example: Torrent-Only Setup

```yaml
delayProfiles:
  - name: Torrents Only
    preferredProtocol: torrent
    enableUsenet: false
    torrentDelay: 0
```

---

## 3. Bundled Configs

### 3.1 RadarrConfig

All-in-one configuration for Radarr:

```go
// api/v1alpha1/radarrconfig_types.go

package v1alpha1

import (
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="URL",type=string,JSONPath=`.spec.connection.url`
// +kubebuilder:printcolumn:name="Quality",type=string,JSONPath=`.spec.quality.preset`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// RadarrConfig is the all-in-one configuration for a Radarr instance
type RadarrConfig struct {
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty"`

    Spec   RadarrConfigSpec   `json:"spec,omitempty"`
    Status RadarrConfigStatus `json:"status,omitempty"`
}

// RadarrConfigSpec defines the desired configuration for Radarr
type RadarrConfigSpec struct {
    // Connection specifies how to connect to Radarr.
    // +kubebuilder:validation:Required
    Connection ConnectionSpec `json:"connection"`

    // Quality defines movie quality preferences.
    // Defaults to "balanced" preset if not specified.
    // +optional
    Quality *VideoQualitySpec `json:"quality,omitempty"`

    // DownloadClients configures download clients.
    // +optional
    DownloadClients []DownloadClientSpec `json:"downloadClients,omitempty"`

    // Indexers configures indexer sources.
    // +optional
    Indexers *IndexersSpec `json:"indexers,omitempty"`

    // Naming configures file/folder naming.
    // Defaults to "plex-friendly" preset if not specified.
    // +optional
    Naming *NamingSpec `json:"naming,omitempty"`

    // RootFolders configures root folder paths.
    // +optional
    RootFolders []string `json:"rootFolders,omitempty"`

    // Reconciliation configures sync behavior.
    // +optional
    Reconciliation *ReconciliationSpec `json:"reconciliation,omitempty"`
}

// RadarrConfigStatus defines the observed state
type RadarrConfigStatus struct {
    // Conditions represent the latest observations.
    Conditions []metav1.Condition `json:"conditions,omitempty"`

    // Connected indicates Radarr is reachable.
    Connected bool `json:"connected,omitempty"`

    // ServiceVersion is the Radarr version.
    ServiceVersion string `json:"serviceVersion,omitempty"`

    // LastReconcile is the last reconciliation time.
    LastReconcile *metav1.Time `json:"lastReconcile,omitempty"`

    // ManagedResources lists resources created by this config.
    ManagedResources ManagedResources `json:"managedResources,omitempty"`
}

// ManagedResources tracks created resources
type ManagedResources struct {
    QualityProfileID   *int  `json:"qualityProfileId,omitempty"`
    CustomFormatIDs    []int `json:"customFormatIds,omitempty"`
    DownloadClientIDs  []int `json:"downloadClientIds,omitempty"`
    IndexerIDs         []int `json:"indexerIds,omitempty"`
}

// ReconciliationSpec configures reconciliation behavior
type ReconciliationSpec struct {
    // Interval between reconciliations.
    // +optional
    // +kubebuilder:default="5m"
    Interval *metav1.Duration `json:"interval,omitempty"`

    // Suspend pauses reconciliation.
    // +optional
    Suspend bool `json:"suspend,omitempty"`
}
```

#### Example: Minimal RadarrConfig

```yaml
apiVersion: arr.rinzler.cloud/v1alpha1
kind: RadarrConfig
metadata:
  name: radarr
  namespace: media
spec:
  connection:
    url: http://radarr:7878
  # Everything else uses sensible defaults:
  # - quality: balanced preset
  # - naming: plex-friendly preset
  # - API key: auto-discovered from /radarr-config/config.xml
```

#### Example: Customized RadarrConfig

```yaml
apiVersion: arr.rinzler.cloud/v1alpha1
kind: RadarrConfig
metadata:
  name: radarr-4k
  namespace: media
spec:
  connection:
    url: http://radarr-4k:7878
    apiKeySecretRef:
      name: radarr-4k-api-key
  
  quality:
    preset: "4k-hdr"
    exclude:
      - "3d"
      - "dolby-vision"  # No DV display
    preferAdditional:
      - "imax"
  
  downloadClients:
    - name: qbittorrent
      url: http://qbittorrent:8080
      credentialsSecretRef:
        name: qbit-credentials
  
  indexers:
    prowlarrRef:
      name: prowlarr
      autoRegister: true
  
  naming:
    preset: "plex-friendly"
  
  rootFolders:
    - /movies/4k
```

### 3.2 SonarrConfig

```go
// SonarrConfig is the all-in-one configuration for a Sonarr instance
type SonarrConfig struct {
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty"`

    Spec   SonarrConfigSpec   `json:"spec,omitempty"`
    Status SonarrConfigStatus `json:"status,omitempty"`
}

// SonarrConfigSpec defines the desired configuration for Sonarr
type SonarrConfigSpec struct {
    // Connection specifies how to connect to Sonarr.
    // +kubebuilder:validation:Required
    Connection ConnectionSpec `json:"connection"`

    // Quality defines TV quality preferences.
    // +optional
    Quality *VideoQualitySpec `json:"quality,omitempty"`

    // DownloadClients configures download clients.
    // +optional
    DownloadClients []DownloadClientSpec `json:"downloadClients,omitempty"`

    // Indexers configures indexer sources.
    // +optional
    Indexers *IndexersSpec `json:"indexers,omitempty"`

    // Naming configures file/folder naming.
    // +optional
    Naming *SonarrNamingSpec `json:"naming,omitempty"`

    // RootFolders configures root folder paths.
    // +optional
    RootFolders []string `json:"rootFolders,omitempty"`

    // Reconciliation configures sync behavior.
    // +optional
    Reconciliation *ReconciliationSpec `json:"reconciliation,omitempty"`
}

// SonarrNamingSpec extends NamingSpec with Sonarr-specific fields
type SonarrNamingSpec struct {
    NamingSpec `json:",inline"`

    // SeasonFolderFormat for season folders.
    // +optional
    SeasonFolderFormat string `json:"seasonFolderFormat,omitempty"`

    // DailyEpisodeFormat for daily shows.
    // +optional
    DailyEpisodeFormat string `json:"dailyEpisodeFormat,omitempty"`

    // AnimeEpisodeFormat for anime.
    // +optional
    AnimeEpisodeFormat string `json:"animeEpisodeFormat,omitempty"`
}
```

### 3.3 LidarrConfig

```go
// LidarrConfig is the all-in-one configuration for a Lidarr instance
type LidarrConfig struct {
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty"`

    Spec   LidarrConfigSpec   `json:"spec,omitempty"`
    Status LidarrConfigStatus `json:"status,omitempty"`
}

// LidarrConfigSpec defines the desired configuration for Lidarr
type LidarrConfigSpec struct {
    // Connection specifies how to connect to Lidarr.
    // Note: Lidarr uses API v1, not v3.
    // +kubebuilder:validation:Required
    Connection ConnectionSpec `json:"connection"`

    // Quality defines audio quality preferences.
    // +optional
    Quality *AudioQualitySpec `json:"quality,omitempty"`

    // Metadata configures which album types to include.
    // +optional
    Metadata *MetadataProfileSpec `json:"metadata,omitempty"`

    // DownloadClients configures download clients.
    // +optional
    DownloadClients []DownloadClientSpec `json:"downloadClients,omitempty"`

    // Indexers configures indexer sources.
    // +optional
    Indexers *IndexersSpec `json:"indexers,omitempty"`

    // Naming configures file/folder naming.
    // +optional
    Naming *LidarrNamingSpec `json:"naming,omitempty"`

    // RootFolders configures root folder paths.
    // Lidarr root folders require additional metadata.
    // +optional
    RootFolders []LidarrRootFolder `json:"rootFolders,omitempty"`

    // Reconciliation configures sync behavior.
    // +optional
    Reconciliation *ReconciliationSpec `json:"reconciliation,omitempty"`
}

// MetadataProfileSpec defines which album types to allow
type MetadataProfileSpec struct {
    // PrimaryTypes: album, ep, single, broadcast, other
    // +optional
    PrimaryTypes []string `json:"primaryTypes,omitempty"`

    // SecondaryTypes: studio, compilation, soundtrack, live, remix, etc.
    // +optional
    SecondaryTypes []string `json:"secondaryTypes,omitempty"`

    // ReleaseStatuses: official, promotional, bootleg
    // +optional
    ReleaseStatuses []string `json:"releaseStatuses,omitempty"`
}

// LidarrRootFolder extends root folder with Lidarr requirements
type LidarrRootFolder struct {
    Path string `json:"path"`
    Name string `json:"name,omitempty"`
    // DefaultMonitor: all, future, missing, existing, latest, first, none
    DefaultMonitor string `json:"defaultMonitor,omitempty"`
}

// LidarrNamingSpec for Lidarr naming
type LidarrNamingSpec struct {
    NamingSpec `json:",inline"`
    
    // ArtistFolderFormat for artist folders.
    // +optional
    ArtistFolderFormat string `json:"artistFolderFormat,omitempty"`
    
    // AlbumFolderFormat for album folders.
    // +optional
    AlbumFolderFormat string `json:"albumFolderFormat,omitempty"`
}
```

### 3.4 ProwlarrConfig

```go
// ProwlarrConfig is the all-in-one configuration for a Prowlarr instance
type ProwlarrConfig struct {
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty"`

    Spec   ProwlarrConfigSpec   `json:"spec,omitempty"`
    Status ProwlarrConfigStatus `json:"status,omitempty"`
}

// ProwlarrConfigSpec defines the desired configuration for Prowlarr
type ProwlarrConfigSpec struct {
    // Connection specifies how to connect to Prowlarr.
    // Note: Prowlarr uses API v1, not v3.
    // +kubebuilder:validation:Required
    Connection ConnectionSpec `json:"connection"`

    // Indexers configures native indexers in Prowlarr.
    // +optional
    Indexers []ProwlarrIndexer `json:"indexers,omitempty"`

    // Proxies configures indexer proxies (e.g., FlareSolverr).
    // +optional
    Proxies []IndexerProxy `json:"proxies,omitempty"`

    // Applications configures sync to Radarr/Sonarr/Lidarr.
    // Usually not needed if those apps use prowlarrRef with autoRegister.
    // +optional
    Applications []ProwlarrApplication `json:"applications,omitempty"`

    // DownloadClients configures download clients in Prowlarr.
    // +optional
    DownloadClients []DownloadClientSpec `json:"downloadClients,omitempty"`

    // Reconciliation configures sync behavior.
    // +optional
    Reconciliation *ReconciliationSpec `json:"reconciliation,omitempty"`
}

// ProwlarrIndexer defines a native indexer in Prowlarr
type ProwlarrIndexer struct {
    // Name is the display name.
    // +kubebuilder:validation:Required
    Name string `json:"name"`

    // Definition is the indexer definition (e.g., "1337x", "Nyaa", "IPTorrents").
    // +kubebuilder:validation:Required
    Definition string `json:"definition"`

    // BaseURL overrides the default URL for the indexer.
    // +optional
    BaseURL string `json:"baseUrl,omitempty"`

    // Settings are definition-specific settings.
    // +optional
    Settings map[string]string `json:"settings,omitempty"`

    // APIKeySecretRef for private indexers.
    // +optional
    APIKeySecretRef *SecretKeySelector `json:"apiKeySecretRef,omitempty"`

    // Tags associate this indexer with proxies.
    // +optional
    Tags []string `json:"tags,omitempty"`

    // Priority (1-50).
    // +optional
    // +kubebuilder:default=25
    Priority int `json:"priority,omitempty"`

    // Enabled enables/disables this indexer.
    // +optional
    // +kubebuilder:default=true
    Enabled *bool `json:"enabled,omitempty"`
}

// IndexerProxy defines a proxy for indexer requests
type IndexerProxy struct {
    // Name is the display name.
    // +kubebuilder:validation:Required
    Name string `json:"name"`

    // Type: flaresolverr, http, socks4, socks5
    // +kubebuilder:validation:Enum=flaresolverr;http;socks4;socks5
    // +kubebuilder:validation:Required
    Type string `json:"type"`

    // Host is the proxy URL or hostname.
    // For FlareSolverr: full URL (http://flaresolverr:8191)
    // For HTTP/SOCKS: hostname only
    // +kubebuilder:validation:Required
    Host string `json:"host"`

    // Port for HTTP/SOCKS proxies.
    // +optional
    Port int `json:"port,omitempty"`

    // CredentialsSecretRef for authenticated proxies.
    // +optional
    CredentialsSecretRef *CredentialsSecretRef `json:"credentialsSecretRef,omitempty"`

    // RequestTimeout for FlareSolverr (seconds).
    // +optional
    // +kubebuilder:default=60
    RequestTimeout int `json:"requestTimeout,omitempty"`

    // Tags to associate with indexers that should use this proxy.
    // +optional
    Tags []string `json:"tags,omitempty"`
}

// ProwlarrApplication defines sync to a downstream app
type ProwlarrApplication struct {
    // Name is the display name.
    // +kubebuilder:validation:Required
    Name string `json:"name"`

    // Type: radarr, sonarr, lidarr
    // +kubebuilder:validation:Enum=radarr;sonarr;lidarr
    // +kubebuilder:validation:Required
    Type string `json:"type"`

    // URL is the application URL.
    // +kubebuilder:validation:Required
    URL string `json:"url"`

    // APIKeySecretRef for the application.
    // If not specified, auto-discovery is attempted.
    // +optional
    APIKeySecretRef *SecretKeySelector `json:"apiKeySecretRef,omitempty"`

    // ConfigPath for API key auto-discovery.
    // +optional
    ConfigPath string `json:"configPath,omitempty"`

    // SyncCategories to sync (human-readable or numeric).
    // Defaults based on app type if not specified.
    // +optional
    SyncCategories []string `json:"syncCategories,omitempty"`

    // SyncLevel: disabled, addOnly, fullSync
    // +optional
    // +kubebuilder:default=fullSync
    SyncLevel string `json:"syncLevel,omitempty"`
}
```

---

## 4. Granular Policies

For power users who need fine-grained control or want to update only part of a config.

### 4.1 RadarrMediaPolicy

```go
// RadarrMediaPolicy defines quality/format settings for Radarr
// Overrides the quality section of RadarrConfig if both exist.
type RadarrMediaPolicy struct {
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty"`

    Spec   RadarrMediaPolicySpec   `json:"spec,omitempty"`
    Status PolicyStatus            `json:"status,omitempty"`
}

// RadarrMediaPolicySpec defines quality configuration for Radarr
type RadarrMediaPolicySpec struct {
    // ConfigRef references the RadarrConfig this policy applies to.
    // +kubebuilder:validation:Required
    ConfigRef LocalObjectReference `json:"configRef"`

    // Quality defines movie quality preferences.
    // +kubebuilder:validation:Required
    Quality VideoQualitySpec `json:"quality"`

    // Formats defines custom format scoring (advanced).
    // +optional
    Formats *FormatSpec `json:"formats,omitempty"`

    // Release defines release filtering rules (advanced).
    // +optional
    Release *ReleaseSpec `json:"release,omitempty"`
}

// LocalObjectReference references an object in the same namespace
type LocalObjectReference struct {
    Name string `json:"name"`
}

// PolicyStatus is common status for all policies
type PolicyStatus struct {
    Conditions []metav1.Condition `json:"conditions,omitempty"`
    Realized   bool               `json:"realized,omitempty"`
    Message    string             `json:"message,omitempty"`
}
```

### 4.2 SonarrMediaPolicy

```go
// SonarrMediaPolicy defines quality/format settings for Sonarr
type SonarrMediaPolicy struct {
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty"`

    Spec   SonarrMediaPolicySpec `json:"spec,omitempty"`
    Status PolicyStatus          `json:"status,omitempty"`
}

type SonarrMediaPolicySpec struct {
    ConfigRef LocalObjectReference `json:"configRef"`
    Quality   VideoQualitySpec     `json:"quality"`
    Formats   *FormatSpec          `json:"formats,omitempty"`
    Release   *ReleaseSpec         `json:"release,omitempty"`
}
```

### 4.3 LidarrMusicPolicy

```go
// LidarrMusicPolicy defines quality settings for Lidarr (music)
type LidarrMusicPolicy struct {
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty"`

    Spec   LidarrMusicPolicySpec `json:"spec,omitempty"`
    Status PolicyStatus          `json:"status,omitempty"`
}

type LidarrMusicPolicySpec struct {
    ConfigRef LocalObjectReference `json:"configRef"`
    Quality   AudioQualitySpec     `json:"quality"`
    Metadata  *MetadataProfileSpec `json:"metadata,omitempty"`
    // ReleaseProfile for Lidarr (simpler than custom formats)
    ReleaseProfile *ReleaseProfileSpec `json:"releaseProfile,omitempty"`
}

// ReleaseProfileSpec defines Lidarr release filtering
type ReleaseProfileSpec struct {
    // Required terms that must appear in release name.
    Required []string `json:"required,omitempty"`
    // Ignored terms that must not appear.
    Ignored []string `json:"ignored,omitempty"`
}
```

### 4.4 Download Client Policies

```go
// RadarrDownloadClientPolicy configures download clients for Radarr
type RadarrDownloadClientPolicy struct {
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty"`

    Spec   DownloadClientPolicySpec `json:"spec,omitempty"`
    Status PolicyStatus             `json:"status,omitempty"`
}

type DownloadClientPolicySpec struct {
    ConfigRef       LocalObjectReference `json:"configRef"`
    DownloadClients []DownloadClientSpec `json:"downloadClients"`
}

// Similar for SonarrDownloadClientPolicy, LidarrDownloadClientPolicy
```

### 4.5 Indexer Policies

```go
// RadarrIndexerPolicy configures indexers for Radarr
type RadarrIndexerPolicy struct {
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty"`

    Spec   IndexerPolicySpec `json:"spec,omitempty"`
    Status PolicyStatus      `json:"status,omitempty"`
}

type IndexerPolicySpec struct {
    ConfigRef LocalObjectReference `json:"configRef"`
    Indexers  IndexersSpec         `json:"indexers"`
}

// Similar for SonarrIndexerPolicy, LidarrIndexerPolicy
```

---

## 5. Shared Resources

### 5.1 QualityTemplate

> **Scope:** QualityTemplate is namespace-scoped. Reference it from configs in the same namespace only.

```go
// QualityTemplate defines a reusable quality configuration
type QualityTemplate struct {
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty"`

    Spec QualityTemplateSpec `json:"spec,omitempty"`
}

type QualityTemplateSpec struct {
    // Video quality configuration (for Radarr/Sonarr)
    // +optional
    Video *VideoQualitySpec `json:"video,omitempty"`

    // Audio quality configuration (for Lidarr)
    // +optional
    Audio *AudioQualitySpec `json:"audio,omitempty"`
}
```

### 5.2 NebularrDefaults

```go
// NebularrDefaults provides namespace-level default configuration
type NebularrDefaults struct {
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty"`

    Spec NebularrDefaultsSpec `json:"spec,omitempty"`
}

type NebularrDefaultsSpec struct {
    // VideoQuality defaults for Radarr/Sonarr in this namespace.
    // +optional
    VideoQuality *VideoQualitySpec `json:"videoQuality,omitempty"`

    // AudioQuality defaults for Lidarr in this namespace.
    // +optional
    AudioQuality *AudioQualitySpec `json:"audioQuality,omitempty"`

    // Naming defaults for all apps.
    // +optional
    Naming *NamingSpec `json:"naming,omitempty"`

    // DownloadClients shared across apps.
    // +optional
    DownloadClients []DownloadClientSpec `json:"downloadClients,omitempty"`

    // Reconciliation defaults.
    // +optional
    Reconciliation *ReconciliationSpec `json:"reconciliation,omitempty"`
}
```

### 5.3 ClusterNebularrDefaults

```go
// ClusterNebularrDefaults provides cluster-level default configuration
// +kubebuilder:resource:scope=Cluster
type ClusterNebularrDefaults struct {
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty"`

    Spec NebularrDefaultsSpec `json:"spec,omitempty"`
}
```

---

## 6. BazarrConfig

Bazarr uses config.yaml pre-seeding, not API:

```go
// BazarrConfig generates a ConfigMap for Bazarr configuration
type BazarrConfig struct {
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty"`

    Spec   BazarrConfigSpec   `json:"spec,omitempty"`
    Status BazarrConfigStatus `json:"status,omitempty"`
}

type BazarrConfigSpec struct {
    // SonarrRef references a SonarrConfig for connection.
    // +optional
    SonarrRef *LocalObjectReference `json:"sonarrRef,omitempty"`

    // RadarrRef references a RadarrConfig for connection.
    // +optional
    RadarrRef *LocalObjectReference `json:"radarrRef,omitempty"`

    // Languages configures subtitle languages.
    // +optional
    Languages []BazarrLanguage `json:"languages,omitempty"`

    // Providers configures subtitle providers.
    // +optional
    Providers []BazarrProvider `json:"providers,omitempty"`

    // OutputConfigMapName is the name of the generated ConfigMap.
    // Defaults to "{name}-config".
    // +optional
    OutputConfigMapName string `json:"outputConfigMapName,omitempty"`
}

type BazarrLanguage struct {
    Code   string `json:"code"`   // e.g., "en", "es"
    Forced bool   `json:"forced,omitempty"`
    HI     bool   `json:"hi,omitempty"` // Hearing impaired
}

type BazarrProvider struct {
    Name     string `json:"name"` // e.g., "opensubtitles", "subscene"
    Username string `json:"username,omitempty"`
    // PasswordSecretRef for provider password.
    PasswordSecretRef *SecretKeySelector `json:"passwordSecretRef,omitempty"`
}

type BazarrConfigStatus struct {
    Conditions       []metav1.Condition `json:"conditions,omitempty"`
    ConfigMapRef     string             `json:"configMapRef,omitempty"`
    LastGenerated    *metav1.Time       `json:"lastGenerated,omitempty"`
}
```

---

## 7. Validation

### 7.1 Schema Validation

Handled by kubebuilder markers (Enum, Required, Pattern, etc.)

### 7.2 Semantic Validation

```go
// internal/validation/radarr.go

func ValidateRadarrConfig(config *RadarrConfig) error {
    var errs []error

    // Quality: preset or manual, not both
    if config.Spec.Quality != nil {
        q := config.Spec.Quality
        if q.Preset != "" && len(q.Tiers) > 0 {
            errs = append(errs, errors.New("cannot specify both preset and tiers"))
        }
        if q.TemplateRef != nil && q.Preset != "" {
            errs = append(errs, errors.New("cannot specify both preset and templateRef"))
        }
    }

    // Indexers: prowlarrRef or direct, not both
    if config.Spec.Indexers != nil {
        i := config.Spec.Indexers
        if i.ProwlarrRef != nil && len(i.Direct) > 0 {
            errs = append(errs, errors.New("cannot specify both prowlarrRef and direct indexers"))
        }
    }

    return errors.Join(errs...)
}
```

### 7.3 Cross-Resource Validation

```go
// Webhook validation

func (v *RadarrConfigValidator) ValidateCreate(ctx context.Context, obj runtime.Object) error {
    config := obj.(*RadarrConfig)

    // Check if another RadarrConfig targets the same URL
    var existing RadarrConfigList
    if err := v.Client.List(ctx, &existing, client.InNamespace(config.Namespace)); err != nil {
        return err
    }

    for _, other := range existing.Items {
        if other.Name != config.Name && other.Spec.Connection.URL == config.Spec.Connection.URL {
            return fmt.Errorf("RadarrConfig %q already manages %s", other.Name, config.Spec.Connection.URL)
        }
    }

    return nil
}
```

---

## 8. Related Documents

- [README](./README.md) - Build order, file mapping
- [PRESETS](./PRESETS.md) - Quality and naming presets
- [TYPES](./TYPES.md) - IR types and adapter interfaces
- [OPERATIONS](./OPERATIONS.md) - Secret management, error handling
- [RADARR](./RADARR.md) - Radarr API mappings
- [SONARR](./SONARR.md) - Sonarr API mappings
- [LIDARR](./LIDARR.md) - Lidarr API mappings
- [PROWLARR](./PROWLARR.md) - Prowlarr API mappings
