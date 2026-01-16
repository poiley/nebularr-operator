# Nebularr v2 - Additional Features Specification

> **DEPRECATION NOTICE:** This document uses outdated patterns (`ServiceBinding`, generic `IndexerPolicy`) from an earlier design iteration. The current design uses **per-app CRDs** (`RadarrConfig`, `SonarrConfig`, etc.).
>
> **Status:** The feature concepts (Import Lists, Notifications, Metadata, etc.) are still valid and planned. The CRD patterns need to be updated to match the per-app design in [CRDS.md](./CRDS.md).
>
> **For coding agents:** Do NOT copy code from this file. Use [README.md](./README.md) for build order and [CRDS.md](./CRDS.md) for authoritative CRD definitions.
>
> **Related:** [README](./README.md) | [CRDS](./CRDS.md) | [TYPES](./TYPES.md) | [OPERATIONS](./OPERATIONS.md)

---

## Migration Notes

The features in this document should be migrated to the per-app CRD pattern:

| Old Pattern | New Pattern |
|-------------|-------------|
| `ImportListPolicy` (generic) | `RadarrImportListPolicy`, `SonarrImportListPolicy` |
| `NotificationPolicy` (generic) | `RadarrNotificationPolicy`, `SonarrNotificationPolicy`, etc. |
| `MetadataPolicy` (generic) | `RadarrMetadataPolicy`, `SonarrMetadataPolicy` |
| `ServiceBinding` | Removed - use `RadarrConfig`, `SonarrConfig`, etc. |

---

This document extends v2 to cover features from the v1 Python implementation that weren't included in the original design.

---

## 1. Import Lists (Radarr/Sonarr)

Import lists automatically add media to Radarr/Sonarr from external sources like IMDb, Trakt, Letterboxd, or RSS feeds.

### 1.1 ImportListPolicy CRD

```go
// api/v1alpha1/importlistpolicy_types.go

package v1alpha1

import (
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Lists",type=integer,JSONPath=`.status.listCount`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// ImportListPolicy defines import list configuration for Radarr/Sonarr
type ImportListPolicy struct {
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty"`

    Spec   ImportListPolicySpec   `json:"spec,omitempty"`
    Status ImportListPolicyStatus `json:"status,omitempty"`
}

// ImportListPolicySpec defines the desired import list configuration
type ImportListPolicySpec struct {
    // Description is a human-readable description.
    // +optional
    Description string `json:"description,omitempty"`

    // Lists defines the import lists to configure.
    // +kubebuilder:validation:MinItems=1
    // +kubebuilder:validation:Required
    Lists []ImportListConfig `json:"lists"`
}

// ImportListConfig defines a single import list
type ImportListConfig struct {
    // Name is a unique identifier for this import list.
    // +kubebuilder:validation:Required
    // +kubebuilder:validation:MinLength=1
    Name string `json:"name"`

    // Type is the import list implementation.
    // +kubebuilder:validation:Enum=imdb;imdb-list;trakt-user;trakt-list;trakt-popular;plex;rss;radarr-list;sonarr-import
    // +kubebuilder:validation:Required
    Type ImportListType `json:"type"`

    // Enabled toggles this import list.
    // +optional
    // +kubebuilder:default=true
    Enabled *bool `json:"enabled,omitempty"`

    // EnableAuto automatically adds items from this list.
    // +optional
    // +kubebuilder:default=true
    EnableAuto *bool `json:"enableAuto,omitempty"`

    // SearchOnAdd triggers search when items are added.
    // +optional
    // +kubebuilder:default=true
    SearchOnAdd *bool `json:"searchOnAdd,omitempty"`

    // QualityProfile is the quality profile name (resolved to ID at sync time).
    // +kubebuilder:validation:Required
    QualityProfile string `json:"qualityProfile"`

    // RootFolder is the root folder path (validated at sync time).
    // +kubebuilder:validation:Required
    RootFolder string `json:"rootFolder"`

    // Monitor defines what to monitor for new items.
    // +kubebuilder:validation:Enum=movieOnly;movieAndCollection;none;all;future;missing;existing;firstSeason;latestSeason;pilot
    // +optional
    // +kubebuilder:default="movieOnly"
    Monitor string `json:"monitor,omitempty"`

    // MinimumAvailability for Radarr (when to consider available).
    // +kubebuilder:validation:Enum=announced;inCinemas;released;preDB
    // +optional
    // +kubebuilder:default="announced"
    MinimumAvailability string `json:"minimumAvailability,omitempty"`

    // SeriesType for Sonarr.
    // +kubebuilder:validation:Enum=standard;daily;anime
    // +optional
    // +kubebuilder:default="standard"
    SeriesType string `json:"seriesType,omitempty"`

    // SeasonFolder creates season folders for Sonarr.
    // +optional
    // +kubebuilder:default=true
    SeasonFolder *bool `json:"seasonFolder,omitempty"`

    // Settings contains type-specific configuration.
    // +optional
    Settings ImportListSettings `json:"settings,omitempty"`
}

// ImportListType represents the type of import list
// +kubebuilder:validation:Enum=imdb;imdb-list;trakt-user;trakt-list;trakt-popular;plex;rss;radarr-list;sonarr-import
type ImportListType string

const (
    ImportListTypeIMDb         ImportListType = "imdb"
    ImportListTypeIMDbList     ImportListType = "imdb-list"
    ImportListTypeTraktUser    ImportListType = "trakt-user"
    ImportListTypeTraktList    ImportListType = "trakt-list"
    ImportListTypeTraktPopular ImportListType = "trakt-popular"
    ImportListTypePlex         ImportListType = "plex"
    ImportListTypeRSS          ImportListType = "rss"
    ImportListTypeRadarrList   ImportListType = "radarr-list"
    ImportListTypeSonarrImport ImportListType = "sonarr-import"
)

// ImportListSettings contains type-specific settings
type ImportListSettings struct {
    // ListID for IMDb lists (e.g., "ls123456789", "top250", "popular").
    // +optional
    ListID string `json:"listId,omitempty"`

    // URL for RSS feeds or list proxies (e.g., Letterboxd via radarr-list-proxy).
    // +optional
    URL string `json:"url,omitempty"`

    // Username for Trakt.
    // +optional
    Username string `json:"username,omitempty"`

    // ListName for Trakt custom lists.
    // +optional
    ListName string `json:"listName,omitempty"`

    // AccessTokenSecretRef for services requiring OAuth.
    // +optional
    AccessTokenSecretRef *SecretKeySelector `json:"accessTokenSecretRef,omitempty"`

    // RefreshTokenSecretRef for Trakt OAuth refresh.
    // +optional
    RefreshTokenSecretRef *SecretKeySelector `json:"refreshTokenSecretRef,omitempty"`

    // TraktListType for Trakt lists.
    // +kubebuilder:validation:Enum=userWatchList;userWatchedList;userCollectionList;userRecommendationsList
    // +optional
    TraktListType string `json:"traktListType,omitempty"`

    // Limit for number of items to import.
    // +optional
    // +kubebuilder:validation:Minimum=1
    Limit *int `json:"limit,omitempty"`
}

// ImportListPolicyStatus defines observed state
type ImportListPolicyStatus struct {
    // Realized indicates the policy was successfully applied.
    // +optional
    Realized bool `json:"realized,omitempty"`

    // ListCount is the number of configured lists.
    // +optional
    ListCount int `json:"listCount,omitempty"`

    // UnrealizedLists lists import lists that could not be created.
    // +optional
    UnrealizedLists []string `json:"unrealizedLists,omitempty"`

    // ServiceMappings maps to service-specific resources.
    // +optional
    ServiceMappings []ImportListServiceMapping `json:"serviceMappings,omitempty"`

    // Conditions represent the latest observations.
    // +optional
    Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// ImportListServiceMapping maps to service-specific IDs
type ImportListServiceMapping struct {
    BindingName   string `json:"bindingName"`
    ImportListIDs []int  `json:"importListIds,omitempty"`
}

// +kubebuilder:object:root=true

type ImportListPolicyList struct {
    metav1.TypeMeta `json:",inline"`
    metav1.ListMeta `json:"metadata,omitempty"`
    Items           []ImportListPolicy `json:"items"`
}

func init() {
    SchemeBuilder.Register(&ImportListPolicy{}, &ImportListPolicyList{})
}
```

### 1.2 Import List IR Types

```go
// internal/ir/v1/import_lists.go

package v1

// ImportListIR represents an import list configuration
type ImportListIR struct {
    // Name is the list name (generated: "nebularr-{list-name}")
    Name string `json:"name"`

    // Type is the list implementation
    Type string `json:"type"`

    // Enable toggles the list
    Enable bool `json:"enable"`

    // EnableAuto adds items automatically
    EnableAuto bool `json:"enableAuto"`

    // SearchOnAdd searches when items are added
    SearchOnAdd bool `json:"searchOnAdd"`

    // QualityProfileID is the resolved profile ID
    QualityProfileID int `json:"qualityProfileId"`

    // RootFolderPath is the validated root folder
    RootFolderPath string `json:"rootFolderPath"`

    // Monitor setting
    Monitor string `json:"monitor"`

    // MinimumAvailability for Radarr
    MinimumAvailability string `json:"minimumAvailability,omitempty"`

    // SeriesType for Sonarr
    SeriesType string `json:"seriesType,omitempty"`

    // SeasonFolder for Sonarr
    SeasonFolder bool `json:"seasonFolder,omitempty"`

    // TypeSettings are implementation-specific settings
    TypeSettings map[string]interface{} `json:"typeSettings,omitempty"`
}
```

### 1.3 Import List API Mapping (Radarr)

```go
// internal/adapters/radarr/import_lists.go

package radarr

// ImportListImplementationMapping maps our types to Radarr implementations
var ImportListImplementationMapping = map[string]string{
    "imdb":          "IMDbListImport",       // IMDb built-in lists (Top 250, Popular)
    "imdb-list":     "IMDbListImport",       // IMDb user lists (ls123456)
    "trakt-user":    "TraktUserImport",      // Trakt user watchlist/collection
    "trakt-list":    "TraktListImport",      // Trakt custom lists
    "trakt-popular": "TraktPopularImport",   // Trakt popular/trending
    "plex":          "PlexImport",           // Plex watchlist
    "rss":           "RSSImport",            // Generic RSS
    "radarr-list":   "RadarrListImport",     // External URL (Letterboxd proxy)
}

// BuildImportListPayload creates a Radarr import list from IR
func BuildImportListPayload(ir *ImportListIR) map[string]interface{} {
    implementation := ImportListImplementationMapping[ir.Type]

    fields := buildImportListFields(ir)

    return map[string]interface{}{
        "name":                ir.Name,
        "enabled":             ir.Enable,
        "enableAuto":          ir.EnableAuto,
        "searchOnAdd":         ir.SearchOnAdd,
        "qualityProfileId":    ir.QualityProfileID,
        "rootFolderPath":      ir.RootFolderPath,
        "monitor":             ir.Monitor,
        "minimumAvailability": ir.MinimumAvailability,
        "listType":            "program",
        "listOrder":           0,
        "implementation":      implementation,
        "configContract":      implementation + "Settings",
        "fields":              fields,
        "tags":                []int{}, // Set to nebularr-managed tag ID
    }
}

func buildImportListFields(ir *ImportListIR) []map[string]interface{} {
    fields := []map[string]interface{}{}

    for name, value := range ir.TypeSettings {
        fields = append(fields, map[string]interface{}{
            "name":  name,
            "value": value,
        })
    }

    return fields
}
```

### 1.4 Example YAML

```yaml
apiVersion: arr.rinzler.cloud/v1alpha1
kind: ImportListPolicy
metadata:
  name: movie-watchlists
  namespace: media
spec:
  description: "Automated movie imports from various sources"
  lists:
    # IMDb Top 250
    - name: "IMDb Top 250"
      type: imdb
      qualityProfile: "HD-1080p"
      rootFolder: "/media/movies"
      monitor: movieOnly
      minimumAvailability: released
      settings:
        listId: "top250"

    # Letterboxd watchlist via proxy
    - name: "My Letterboxd"
      type: radarr-list
      qualityProfile: "HD-1080p"
      rootFolder: "/media/movies"
      settings:
        url: "https://letterboxd-list-radarr.onrender.com/myuser/watchlist/"

    # Trakt watchlist
    - name: "Trakt Watchlist"
      type: trakt-user
      qualityProfile: "4K-HDR"
      rootFolder: "/media/movies/4k"
      settings:
        username: "mytrakt"
        traktListType: "userWatchList"
        accessTokenSecretRef:
          name: trakt-credentials
          key: accessToken
```

---

## 2. Prowlarr Integration

Prowlarr is the indexer manager that syncs indexers to Radarr/Sonarr/Lidarr. This section covers:
- Prowlarr Applications (connections TO Radarr/Sonarr/Lidarr)
- Prowlarr Indexers (native indexer definitions)
- Indexer Proxies (FlareSolverr, etc.)

### 2.1 Extended IndexerPolicy CRD

The existing `IndexerPolicy` CRD is extended to support both:
1. **Prowlarr native indexers** (Cardigann/generic indexer definitions)
2. **Prowlarr app connections** (sync indexers to other *arr apps)

```go
// api/v1alpha1/indexerpolicy_types.go (extended)

// IndexerPolicySpec defines indexer configuration
type IndexerPolicySpec struct {
    // Description is a human-readable description.
    // +optional
    Description string `json:"description,omitempty"`

    // Indexers lists the indexers to configure (for Radarr/Sonarr/Lidarr).
    // These create Torznab/Newznab indexers pointing to Prowlarr.
    // +optional
    Indexers []IndexerConfig `json:"indexers,omitempty"`

    // ProwlarrIndexers lists native Prowlarr indexer definitions.
    // Only used when ServiceBinding targets Prowlarr.
    // +optional
    ProwlarrIndexers []ProwlarrIndexerConfig `json:"prowlarrIndexers,omitempty"`

    // IndexerProxies defines proxy servers for indexers (FlareSolverr, etc.).
    // Only used when ServiceBinding targets Prowlarr.
    // +optional
    IndexerProxies []IndexerProxyConfig `json:"indexerProxies,omitempty"`

    // Applications defines Prowlarr app connections (sync TO Radarr/Sonarr/Lidarr).
    // Only used when ServiceBinding targets Prowlarr.
    // +optional
    Applications []ProwlarrApplicationConfig `json:"applications,omitempty"`

    // ProwlarrSync configures syncing Prowlarr indexers to this app.
    // Used in Radarr/Sonarr ServiceBindings.
    // +optional
    ProwlarrSync *ProwlarrSyncConfig `json:"prowlarrSync,omitempty"`

    // GlobalSettings apply to all indexers in this policy.
    // +optional
    GlobalSettings *IndexerGlobalSettings `json:"globalSettings,omitempty"`
}

// ProwlarrIndexerConfig defines a Prowlarr native indexer
type ProwlarrIndexerConfig struct {
    // Name is a unique identifier for this indexer.
    // +kubebuilder:validation:Required
    Name string `json:"name"`

    // Type is the indexer definition name (e.g., "thepiratebay", "1337x", "nyaa").
    // Must match a Prowlarr indexer definition.
    // +kubebuilder:validation:Required
    Type string `json:"type"`

    // BaseURL overrides the default indexer URL.
    // +optional
    BaseURL string `json:"baseUrl,omitempty"`

    // Priority affects search order (higher = searched first).
    // +optional
    // +kubebuilder:validation:Minimum=1
    // +kubebuilder:validation:Maximum=50
    // +kubebuilder:default=25
    Priority *int `json:"priority,omitempty"`

    // Tags are tag names to associate with this indexer.
    // Used to link indexers to proxies.
    // +optional
    Tags []string `json:"tags,omitempty"`

    // SeedRatio target for torrents.
    // +optional
    SeedRatio *float64 `json:"seedRatio,omitempty"`

    // SeedTime minimum in minutes for torrents.
    // +optional
    SeedTime *int `json:"seedTime,omitempty"`

    // APIKeySecretRef for indexers requiring authentication.
    // +optional
    APIKeySecretRef *SecretKeySelector `json:"apiKeySecretRef,omitempty"`
}

// IndexerProxyConfig defines an indexer proxy (e.g., FlareSolverr)
type IndexerProxyConfig struct {
    // Name is a unique identifier for this proxy.
    // +kubebuilder:validation:Required
    Name string `json:"name"`

    // Type is the proxy type.
    // +kubebuilder:validation:Enum=flaresolverr;http;socks4;socks5
    // +kubebuilder:validation:Required
    Type IndexerProxyType `json:"type"`

    // Host is the proxy host/URL.
    // For FlareSolverr: full URL (e.g., http://flaresolverr:8191).
    // For HTTP/SOCKS: hostname only.
    // +kubebuilder:validation:Required
    Host string `json:"host"`

    // Port for HTTP/SOCKS proxies.
    // +optional
    // +kubebuilder:validation:Minimum=1
    // +kubebuilder:validation:Maximum=65535
    Port *int `json:"port,omitempty"`

    // CredentialsSecretRef for authenticated proxies.
    // +optional
    CredentialsSecretRef *CredentialsSecretSelector `json:"credentialsSecretRef,omitempty"`

    // RequestTimeout in seconds for FlareSolverr.
    // +optional
    // +kubebuilder:default=60
    RequestTimeout *int `json:"requestTimeout,omitempty"`

    // Tags are tag names that indexers use to reference this proxy.
    // +optional
    Tags []string `json:"tags,omitempty"`
}

// IndexerProxyType represents the proxy implementation
// +kubebuilder:validation:Enum=flaresolverr;http;socks4;socks5
type IndexerProxyType string

const (
    IndexerProxyTypeFlareSolverr IndexerProxyType = "flaresolverr"
    IndexerProxyTypeHTTP         IndexerProxyType = "http"
    IndexerProxyTypeSocks4       IndexerProxyType = "socks4"
    IndexerProxyTypeSocks5       IndexerProxyType = "socks5"
)

// ProwlarrApplicationConfig defines a Prowlarr app connection
type ProwlarrApplicationConfig struct {
    // Name is the display name for this connection.
    // +kubebuilder:validation:Required
    Name string `json:"name"`

    // Type is the target application type.
    // +kubebuilder:validation:Enum=radarr;sonarr;lidarr;readarr
    // +kubebuilder:validation:Required
    Type string `json:"type"`

    // URL is the target application URL.
    // +kubebuilder:validation:Required
    // +kubebuilder:validation:Pattern=`^https?://`
    URL string `json:"url"`

    // APIKeyDiscovery specifies how to discover the target app's API key.
    // +kubebuilder:validation:Required
    APIKeyDiscovery APIKeyDiscoveryConfig `json:"apiKeyDiscovery"`

    // SyncCategories are the newznab category IDs to sync.
    // +optional
    SyncCategories []int `json:"syncCategories,omitempty"`

    // SyncLevel controls what Prowlarr syncs.
    // +kubebuilder:validation:Enum=disabled;addOnly;fullSync
    // +optional
    // +kubebuilder:default="fullSync"
    SyncLevel string `json:"syncLevel,omitempty"`
}

// APIKeyDiscoveryConfig specifies how to discover an API key
type APIKeyDiscoveryConfig struct {
    // ConfigXMLPath is the path to the app's config.xml file.
    // API key is extracted from <ApiKey> element.
    // +optional
    ConfigXMLPath string `json:"configXmlPath,omitempty"`

    // SecretRef references a K8s Secret containing the API key.
    // +optional
    SecretRef *SecretKeySelector `json:"secretRef,omitempty"`
}

// ProwlarrSyncConfig configures syncing Prowlarr indexers to an *arr app
type ProwlarrSyncConfig struct {
    // ProwlarrURL is Prowlarr's URL.
    // +kubebuilder:validation:Required
    // +kubebuilder:validation:Pattern=`^https?://`
    ProwlarrURL string `json:"prowlarrUrl"`

    // APIKeyDiscovery specifies how to discover Prowlarr's API key.
    // +kubebuilder:validation:Required
    APIKeyDiscovery APIKeyDiscoveryConfig `json:"apiKeyDiscovery"`

    // IndexerNames lists which Prowlarr indexers to sync (by name).
    // +kubebuilder:validation:MinItems=1
    // +kubebuilder:validation:Required
    IndexerNames []string `json:"indexerNames"`

    // Categories are the newznab category IDs for this app.
    // +optional
    Categories []int `json:"categories,omitempty"`

    // EnableRSS enables RSS sync.
    // +optional
    // +kubebuilder:default=true
    EnableRSS *bool `json:"enableRss,omitempty"`

    // EnableAutomaticSearch enables automatic search.
    // +optional
    // +kubebuilder:default=true
    EnableAutomaticSearch *bool `json:"enableAutomaticSearch,omitempty"`

    // EnableInteractiveSearch enables manual/interactive search.
    // +optional
    // +kubebuilder:default=true
    EnableInteractiveSearch *bool `json:"enableInteractiveSearch,omitempty"`

    // MinimumSeeders minimum seeders required.
    // +optional
    // +kubebuilder:default=1
    MinimumSeeders *int `json:"minimumSeeders,omitempty"`
}
```

### 2.2 Prowlarr IR Types

```go
// internal/ir/v1/prowlarr.go

package v1

// ProwlarrIndexerIR represents a Prowlarr native indexer
type ProwlarrIndexerIR struct {
    // Name is the indexer name (generated: "nebularr-{name}")
    Name string `json:"name"`

    // Type is the definition name (e.g., "thepiratebay")
    Type string `json:"type"`

    // BaseURL override
    BaseURL string `json:"baseUrl,omitempty"`

    // Priority (1-50)
    Priority int `json:"priority"`

    // Tags are tag IDs (resolved from names)
    Tags []int `json:"tags,omitempty"`

    // SeedRatio target
    SeedRatio float64 `json:"seedRatio,omitempty"`

    // SeedTimeMinutes minimum seed time
    SeedTimeMinutes int `json:"seedTimeMinutes,omitempty"`

    // APIKey if required
    APIKey string `json:"apiKey,omitempty"`
}

// IndexerProxyIR represents an indexer proxy
type IndexerProxyIR struct {
    // Name is the proxy name (generated: "nebularr-{name}")
    Name string `json:"name"`

    // Type is the proxy implementation
    Type string `json:"type"`

    // Host/URL
    Host string `json:"host"`

    // Port for HTTP/SOCKS
    Port int `json:"port,omitempty"`

    // Username for authentication
    Username string `json:"username,omitempty"`

    // Password for authentication
    Password string `json:"password,omitempty"`

    // RequestTimeout for FlareSolverr
    RequestTimeout int `json:"requestTimeout,omitempty"`

    // Tags are tag IDs
    Tags []int `json:"tags,omitempty"`
}

// ProwlarrApplicationIR represents a Prowlarr app connection
type ProwlarrApplicationIR struct {
    // Name is the connection name (generated: "nebularr-{name}")
    Name string `json:"name"`

    // Type is the app type (radarr, sonarr, lidarr)
    Type string `json:"type"`

    // URL is the target app URL
    URL string `json:"url"`

    // APIKey is the target app's API key (discovered)
    APIKey string `json:"apiKey"`

    // ProwlarrURL for sync back
    ProwlarrURL string `json:"prowlarrUrl"`

    // SyncCategories are newznab category IDs
    SyncCategories []int `json:"syncCategories,omitempty"`

    // SyncLevel (disabled, addOnly, fullSync)
    SyncLevel string `json:"syncLevel"`
}

// ProwlarrSyncIndexerIR represents a Torznab indexer pointing to Prowlarr
// Used in Radarr/Sonarr to create indexers that proxy through Prowlarr
type ProwlarrSyncIndexerIR struct {
    // Name is the indexer name (generated: "{prowlarr-name} (Prowlarr)")
    Name string `json:"name"`

    // ProwlarrIndexerID is the indexer ID in Prowlarr
    ProwlarrIndexerID int `json:"prowlarrIndexerId"`

    // ProwlarrURL base URL
    ProwlarrURL string `json:"prowlarrUrl"`

    // ProwlarrAPIKey for authentication
    ProwlarrAPIKey string `json:"prowlarrApiKey"`

    // Categories are newznab category IDs
    Categories []int `json:"categories,omitempty"`

    // EnableRSS enables RSS sync
    EnableRSS bool `json:"enableRss"`

    // EnableAutomaticSearch enables auto search
    EnableAutomaticSearch bool `json:"enableAutomaticSearch"`

    // EnableInteractiveSearch enables manual search
    EnableInteractiveSearch bool `json:"enableInteractiveSearch"`

    // MinimumSeeders required
    MinimumSeeders int `json:"minimumSeeders"`
}
```

### 2.3 Prowlarr Adapter Implementation

```go
// internal/adapters/prowlarr/adapter.go

package prowlarr

import (
    "context"

    "github.com/poiley/nebularr/internal/adapters"
)

// Adapter implements the adapters.Adapter interface for Prowlarr
type Adapter struct {
    client *Client
}

// NewAdapter creates a new Prowlarr adapter
func NewAdapter() *Adapter {
    return &Adapter{}
}

// Name returns the adapter identifier
func (a *Adapter) Name() string {
    return "prowlarr"
}

// SupportedServiceType returns the service type this adapter handles
func (a *Adapter) SupportedServiceType() string {
    return "prowlarr"
}
```

```go
// internal/adapters/prowlarr/indexers.go

package prowlarr

import irv1 "github.com/poiley/nebularr/internal/ir/v1"

// BuildIndexerPayload creates a Prowlarr indexer from IR
func BuildIndexerPayload(ir *irv1.ProwlarrIndexerIR, schema map[string]interface{}) map[string]interface{} {
    fields := buildFieldsFromSchema(schema, ir)

    return map[string]interface{}{
        "name":           ir.Name,
        "enable":         true,
        "priority":       ir.Priority,
        "appProfileId":   1, // Standard profile
        "implementation": schema["implementation"],
        "configContract": schema["configContract"],
        "fields":         fields,
        "tags":           ir.Tags,
    }
}

func buildFieldsFromSchema(schema map[string]interface{}, ir *irv1.ProwlarrIndexerIR) []map[string]interface{} {
    fields := []map[string]interface{}{}

    schemaFields, _ := schema["fields"].([]interface{})
    for _, f := range schemaFields {
        field := f.(map[string]interface{})
        fieldName := field["name"].(string)

        fieldCopy := map[string]interface{}{
            "name": fieldName,
        }

        // Set value based on IR or schema default
        switch fieldName {
        case "baseUrl":
            if ir.BaseURL != "" {
                fieldCopy["value"] = ir.BaseURL
            } else if v, ok := field["value"]; ok {
                fieldCopy["value"] = v
            }
        case "torrentBaseSettings.seedRatio":
            if ir.SeedRatio > 0 {
                fieldCopy["value"] = ir.SeedRatio
            }
        case "torrentBaseSettings.seedTime":
            if ir.SeedTimeMinutes > 0 {
                fieldCopy["value"] = ir.SeedTimeMinutes
            }
        default:
            if v, ok := field["value"]; ok {
                fieldCopy["value"] = v
            }
        }

        fields = append(fields, fieldCopy)
    }

    return fields
}
```

```go
// internal/adapters/prowlarr/proxies.go

package prowlarr

import irv1 "github.com/poiley/nebularr/internal/ir/v1"

// ProxyImplementationMapping maps our types to Prowlarr implementations
var ProxyImplementationMapping = map[string]string{
    "flaresolverr": "FlareSolverr",
    "http":         "Http",
    "socks4":       "Socks4",
    "socks5":       "Socks5",
}

// ProxyConfigContractMapping maps to config contracts
var ProxyConfigContractMapping = map[string]string{
    "flaresolverr": "FlareSolverrSettings",
    "http":         "HttpSettings",
    "socks4":       "Socks4Settings",
    "socks5":       "Socks5Settings",
}

// BuildIndexerProxyPayload creates a Prowlarr indexer proxy from IR
func BuildIndexerProxyPayload(ir *irv1.IndexerProxyIR) map[string]interface{} {
    implementation := ProxyImplementationMapping[ir.Type]
    configContract := ProxyConfigContractMapping[ir.Type]

    var fields []map[string]interface{}

    if ir.Type == "flaresolverr" {
        fields = []map[string]interface{}{
            {"name": "host", "value": ir.Host},
            {"name": "requestTimeout", "value": ir.RequestTimeout},
        }
    } else {
        fields = []map[string]interface{}{
            {"name": "host", "value": ir.Host},
            {"name": "port", "value": ir.Port},
        }
        if ir.Username != "" {
            fields = append(fields, map[string]interface{}{"name": "username", "value": ir.Username})
        }
        if ir.Password != "" {
            fields = append(fields, map[string]interface{}{"name": "password", "value": ir.Password})
        }
    }

    return map[string]interface{}{
        "name":           ir.Name,
        "implementation": implementation,
        "configContract": configContract,
        "fields":         fields,
        "tags":           ir.Tags,
    }
}
```

```go
// internal/adapters/prowlarr/applications.go

package prowlarr

import irv1 "github.com/poiley/nebularr/internal/ir/v1"

// ApplicationImplementationMapping maps app types to Prowlarr implementations
var ApplicationImplementationMapping = map[string]string{
    "radarr":  "Radarr",
    "sonarr":  "Sonarr",
    "lidarr":  "Lidarr",
    "readarr": "Readarr",
}

// ApplicationConfigContractMapping maps to config contracts
var ApplicationConfigContractMapping = map[string]string{
    "radarr":  "RadarrSettings",
    "sonarr":  "SonarrSettings",
    "lidarr":  "LidarrSettings",
    "readarr": "ReadarrSettings",
}

// BuildApplicationPayload creates a Prowlarr application from IR
func BuildApplicationPayload(ir *irv1.ProwlarrApplicationIR) map[string]interface{} {
    implementation := ApplicationImplementationMapping[ir.Type]
    configContract := ApplicationConfigContractMapping[ir.Type]

    fields := []map[string]interface{}{
        {"name": "prowlarrUrl", "value": ir.ProwlarrURL},
        {"name": "baseUrl", "value": ir.URL},
        {"name": "apiKey", "value": ir.APIKey},
        {"name": "syncCategories", "value": ir.SyncCategories},
    }

    return map[string]interface{}{
        "name":           ir.Name,
        "implementation": implementation,
        "configContract": configContract,
        "syncLevel":      ir.SyncLevel,
        "fields":         fields,
        "tags":           []int{}, // Set to nebularr-managed tag ID
    }
}
```

### 2.4 Example YAMLs

#### Prowlarr ServiceBinding with Indexers and Applications

```yaml
apiVersion: arr.rinzler.cloud/v1alpha1
kind: ServiceBinding
metadata:
  name: prowlarr-main
  namespace: media
spec:
  serviceType: prowlarr
  connection:
    url: https://prowlarr.example.com
    apiKeySecretRef:
      name: prowlarr-credentials
      key: apiKey
  policyRefs:
    - kind: IndexerPolicy
      name: public-indexers
    - kind: DownloadClientPolicy
      name: transmission-config
---
apiVersion: arr.rinzler.cloud/v1alpha1
kind: IndexerPolicy
metadata:
  name: public-indexers
  namespace: media
spec:
  description: "Public torrent indexers with FlareSolverr"

  # Indexer proxies (for cloudflare-protected sites)
  indexerProxies:
    - name: FlareSolverr
      type: flaresolverr
      host: http://flaresolverr.media.svc.cluster.local:8191
      requestTimeout: 60
      tags:
        - flaresolverr

  # Native Prowlarr indexers
  prowlarrIndexers:
    - name: "1337x"
      type: "1337x"
      priority: 25
      tags:
        - flaresolverr  # Links to FlareSolverr proxy
      seedRatio: 1.0
      seedTime: 1440  # 24 hours

    - name: "The Pirate Bay"
      type: "thepiratebay"
      priority: 20

    - name: "Nyaa"
      type: "nyaa"
      priority: 30

  # Prowlarr app connections (sync TO these apps)
  applications:
    - name: Radarr
      type: radarr
      url: http://radarr.media.svc.cluster.local:7878
      apiKeyDiscovery:
        configXmlPath: /radarr-config/config.xml
      syncCategories: [2000, 2010, 2020, 2030, 2040, 2045, 2050]
      syncLevel: fullSync

    - name: Sonarr
      type: sonarr
      url: http://sonarr.media.svc.cluster.local:8989
      apiKeyDiscovery:
        configXmlPath: /sonarr-config/config.xml
      syncCategories: [5000, 5010, 5020, 5030, 5040, 5045, 5050]
      syncLevel: fullSync

    - name: Lidarr
      type: lidarr
      url: http://lidarr.media.svc.cluster.local:8686
      apiKeyDiscovery:
        configXmlPath: /lidarr-config/config.xml
      syncCategories: [3000, 3010, 3020, 3030, 3040]
      syncLevel: fullSync
```

#### Radarr ServiceBinding with Prowlarr Sync

```yaml
apiVersion: arr.rinzler.cloud/v1alpha1
kind: ServiceBinding
metadata:
  name: radarr-main
  namespace: media
spec:
  serviceType: radarr
  connection:
    url: https://radarr.example.com
    apiKeySecretRef:
      name: radarr-credentials
  policyRefs:
    - kind: MediaPolicy
      name: 4k-quality
    - kind: IndexerPolicy
      name: radarr-indexers
---
apiVersion: arr.rinzler.cloud/v1alpha1
kind: IndexerPolicy
metadata:
  name: radarr-indexers
  namespace: media
spec:
  description: "Prowlarr indexers synced to Radarr"

  # Sync indexers FROM Prowlarr
  prowlarrSync:
    prowlarrUrl: http://prowlarr.media.svc.cluster.local:9696
    apiKeyDiscovery:
      configXmlPath: /prowlarr-config/config.xml
    indexerNames:
      - "1337x"
      - "The Pirate Bay"
    categories: [2000, 2010, 2020, 2030, 2040, 2045, 2050]  # Movie categories
    enableRss: true
    enableAutomaticSearch: true
    enableInteractiveSearch: true
    minimumSeeders: 5

  globalSettings:
    rssSync: true
    automaticSearch: true
    interactiveSearch: true
```

---

## 3. Bazarr Integration

Bazarr doesn't have a REST API for configuration like other *arr apps. Instead, it reads from `config.yaml`. Nebularr handles this by generating the config file via an init container pattern.

### 3.1 BazarrBinding CRD

```go
// api/v1alpha1/bazarrbinding_types.go

package v1alpha1

import (
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Generated",type=boolean,JSONPath=`.status.configGenerated`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// BazarrBinding configures Bazarr via config.yaml generation
type BazarrBinding struct {
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty"`

    Spec   BazarrBindingSpec   `json:"spec,omitempty"`
    Status BazarrBindingStatus `json:"status,omitempty"`
}

// BazarrBindingSpec defines Bazarr configuration
type BazarrBindingSpec struct {
    // OutputPath is where to write config.yaml.
    // +optional
    // +kubebuilder:default="/config/config/config.yaml"
    OutputPath string `json:"outputPath,omitempty"`

    // Sonarr connection configuration.
    // +kubebuilder:validation:Required
    Sonarr BazarrAppConnection `json:"sonarr"`

    // Radarr connection configuration.
    // +kubebuilder:validation:Required
    Radarr BazarrAppConnection `json:"radarr"`

    // LanguageProfiles define subtitle language preferences.
    // +optional
    LanguageProfiles []BazarrLanguageProfile `json:"languageProfiles,omitempty"`

    // Providers configure subtitle providers.
    // +optional
    Providers []BazarrProviderConfig `json:"providers,omitempty"`

    // Authentication configures Bazarr login.
    // +optional
    Authentication *AuthenticationConfig `json:"authentication,omitempty"`
}

// BazarrAppConnection defines connection to Sonarr/Radarr
type BazarrAppConnection struct {
    // URL is the app URL.
    // +kubebuilder:validation:Required
    // +kubebuilder:validation:Pattern=`^https?://`
    URL string `json:"url"`

    // APIKeyDiscovery specifies how to get the API key.
    // +kubebuilder:validation:Required
    APIKeyDiscovery APIKeyDiscoveryConfig `json:"apiKeyDiscovery"`
}

// BazarrLanguageProfile defines subtitle language preferences
type BazarrLanguageProfile struct {
    // Name is the profile name.
    // +kubebuilder:validation:Required
    Name string `json:"name"`

    // Languages in priority order.
    // +kubebuilder:validation:MinItems=1
    // +kubebuilder:validation:Required
    Languages []BazarrLanguage `json:"languages"`

    // DefaultForSeries makes this the default for TV.
    // +optional
    DefaultForSeries bool `json:"defaultForSeries,omitempty"`

    // DefaultForMovies makes this the default for movies.
    // +optional
    DefaultForMovies bool `json:"defaultForMovies,omitempty"`
}

// BazarrLanguage defines a subtitle language
type BazarrLanguage struct {
    // Code is the ISO 639-1 language code (e.g., "en", "es").
    // +kubebuilder:validation:Required
    // +kubebuilder:validation:MinLength=2
    // +kubebuilder:validation:MaxLength=3
    Code string `json:"code"`

    // Forced indicates forced/foreign-only subtitles.
    // +optional
    Forced bool `json:"forced,omitempty"`

    // HI indicates hearing-impaired subtitles.
    // +optional
    HI bool `json:"hi,omitempty"`
}

// BazarrProviderConfig defines a subtitle provider
type BazarrProviderConfig struct {
    // Name is the provider name (e.g., "opensubtitlescom", "yifysubtitles").
    // +kubebuilder:validation:Required
    Name string `json:"name"`

    // CredentialsSecretRef for providers requiring authentication.
    // +optional
    CredentialsSecretRef *CredentialsSecretSelector `json:"credentialsSecretRef,omitempty"`
}

// BazarrBindingStatus defines observed state
type BazarrBindingStatus struct {
    // ConfigGenerated indicates config.yaml was successfully written.
    // +optional
    ConfigGenerated bool `json:"configGenerated,omitempty"`

    // LastGenerated is when config was last written.
    // +optional
    LastGenerated *metav1.Time `json:"lastGenerated,omitempty"`

    // SonarrConnected indicates Sonarr API key was discovered.
    // +optional
    SonarrConnected bool `json:"sonarrConnected,omitempty"`

    // RadarrConnected indicates Radarr API key was discovered.
    // +optional
    RadarrConnected bool `json:"radarrConnected,omitempty"`

    // Conditions represent the latest observations.
    // +optional
    Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true

type BazarrBindingList struct {
    metav1.TypeMeta `json:",inline"`
    metav1.ListMeta `json:"metadata,omitempty"`
    Items           []BazarrBinding `json:"items"`
}

func init() {
    SchemeBuilder.Register(&BazarrBinding{}, &BazarrBindingList{})
}
```

### 3.2 Bazarr Adapter (Config Generator)

Since Bazarr doesn't use an API, the "adapter" generates a config.yaml file instead.

```go
// internal/adapters/bazarr/generator.go

package bazarr

import (
    "crypto/md5"
    "encoding/hex"
    "fmt"
    "net/url"
    "os"
    "path/filepath"

    "gopkg.in/yaml.v3"
    
    "github.com/poiley/nebularr/api/v1alpha1"
)

// GenerateConfig creates Bazarr config.yaml content
func GenerateConfig(spec *v1alpha1.BazarrBindingSpec, sonarrAPIKey, radarrAPIKey string) ([]byte, error) {
    sonarrHost, sonarrPort, sonarrBase, sonarrSSL := parseURL(spec.Sonarr.URL)
    radarrHost, radarrPort, radarrBase, radarrSSL := parseURL(spec.Radarr.URL)

    config := map[string]interface{}{
        "sonarr": map[string]interface{}{
            "ip":       sonarrHost,
            "port":     sonarrPort,
            "base_url": sonarrBase,
            "ssl":      sonarrSSL,
            "apikey":   sonarrAPIKey,
        },
        "radarr": map[string]interface{}{
            "ip":       radarrHost,
            "port":     radarrPort,
            "base_url": radarrBase,
            "ssl":      radarrSSL,
            "apikey":   radarrAPIKey,
        },
        "general": buildGeneralSection(spec),
    }

    // Add language profiles
    if len(spec.LanguageProfiles) > 0 {
        config["languages"] = buildLanguageProfiles(spec.LanguageProfiles)
    }

    // Add authentication
    if spec.Authentication != nil {
        config["auth"] = buildAuthSection(spec.Authentication)
    }

    // Add provider configs
    for _, provider := range spec.Providers {
        if provider.CredentialsSecretRef != nil {
            // Provider credentials would be resolved from secrets
            config[provider.Name] = map[string]interface{}{
                // Credentials populated at runtime
            }
        }
    }

    return yaml.Marshal(config)
}

func parseURL(rawURL string) (host string, port int, basePath string, ssl bool) {
    u, err := url.Parse(rawURL)
    if err != nil {
        return "localhost", 80, "/", false
    }

    host = u.Hostname()
    if u.Port() != "" {
        fmt.Sscanf(u.Port(), "%d", &port)
    } else if u.Scheme == "https" {
        port = 443
    } else {
        port = 80
    }
    basePath = u.Path
    if basePath == "" {
        basePath = "/"
    }
    ssl = u.Scheme == "https"

    return
}

func buildGeneralSection(spec *v1alpha1.BazarrBindingSpec) map[string]interface{} {
    serieDefault := 1
    movieDefault := 1

    for i, profile := range spec.LanguageProfiles {
        if profile.DefaultForSeries {
            serieDefault = i + 1
        }
        if profile.DefaultForMovies {
            movieDefault = i + 1
        }
    }

    enabledProviders := []string{}
    for _, p := range spec.Providers {
        enabledProviders = append(enabledProviders, p.Name)
    }

    return map[string]interface{}{
        "use_sonarr":             true,
        "use_radarr":             true,
        "enabled_providers":      enabledProviders,
        "single_language":        false,
        "serie_default_enabled":  true,
        "serie_default_profile":  serieDefault,
        "movie_default_enabled":  true,
        "movie_default_profile":  movieDefault,
    }
}

func buildLanguageProfiles(profiles []v1alpha1.BazarrLanguageProfile) []map[string]interface{} {
    result := []map[string]interface{}{}

    for i, profile := range profiles {
        items := []map[string]interface{}{}
        for j, lang := range profile.Languages {
            items = append(items, map[string]interface{}{
                "id":            j + 1,
                "language":      lang.Code,
                "forced":        lang.Forced,
                "hi":            lang.HI,
                "audio_exclude": false,
            })
        }

        result = append(result, map[string]interface{}{
            "name":      profile.Name,
            "profileId": i + 1,
            "cutoff":    nil,
            "items":     items,
        })
    }

    return result
}

func buildAuthSection(auth *v1alpha1.AuthenticationConfig) map[string]interface{} {
    if auth.Method == "none" {
        return map[string]interface{}{"type": nil}
    }

    result := map[string]interface{}{
        "type": auth.Method,
    }

    if auth.Username != "" {
        result["username"] = auth.Username
    }
    // Password is MD5 hashed in Bazarr
    // This would be resolved from secret and hashed at runtime

    return result
}

// MD5Hash creates MD5 hash for Bazarr passwords
func MD5Hash(s string) string {
    h := md5.Sum([]byte(s))
    return hex.EncodeToString(h[:])
}

// WriteConfig writes the config to the specified path
func WriteConfig(content []byte, outputPath string) error {
    dir := filepath.Dir(outputPath)
    if err := os.MkdirAll(dir, 0755); err != nil {
        return fmt.Errorf("failed to create directory: %w", err)
    }

    return os.WriteFile(outputPath, content, 0644)
}
```

### 3.3 Example YAML

```yaml
apiVersion: arr.rinzler.cloud/v1alpha1
kind: BazarrBinding
metadata:
  name: bazarr-config
  namespace: media
spec:
  outputPath: /config/config/config.yaml

  sonarr:
    url: http://sonarr.media.svc.cluster.local:8989
    apiKeyDiscovery:
      configXmlPath: /sonarr-config/config.xml

  radarr:
    url: http://radarr.media.svc.cluster.local:7878
    apiKeyDiscovery:
      configXmlPath: /radarr-config/config.xml

  languageProfiles:
    - name: "English"
      defaultForSeries: true
      defaultForMovies: true
      languages:
        - code: "en"
          forced: false
          hi: false

    - name: "English + Spanish"
      languages:
        - code: "en"
        - code: "es"

  providers:
    - name: opensubtitlescom
      credentialsSecretRef:
        name: bazarr-providers
        usernameKey: opensubtitles_user
        passwordKey: opensubtitles_pass

    - name: yifysubtitles
      # No credentials needed

    - name: podnapisi
      # No credentials needed

  authentication:
    method: forms
    username: admin
    password: "${BAZARR_PASSWORD}"  # Resolved from secret, MD5 hashed
```

---

## 4. API Key Discovery

A key feature for cross-app integration is discovering API keys from config.xml files.

### 4.1 Discovery Utility

```go
// internal/discovery/apikey.go

package discovery

import (
    "context"
    "encoding/xml"
    "fmt"
    "os"
    "time"
)

// ConfigXML represents the structure of *arr config.xml files
type ConfigXML struct {
    XMLName xml.Name `xml:"Config"`
    APIKey  string   `xml:"ApiKey"`
    Port    int      `xml:"Port"`
}

// DiscoverAPIKey extracts the API key from a config.xml file
func DiscoverAPIKey(configPath string) (string, error) {
    data, err := os.ReadFile(configPath)
    if err != nil {
        return "", fmt.Errorf("failed to read config.xml: %w", err)
    }

    var config ConfigXML
    if err := xml.Unmarshal(data, &config); err != nil {
        return "", fmt.Errorf("failed to parse config.xml: %w", err)
    }

    if config.APIKey == "" {
        return "", fmt.Errorf("API key not found in config.xml")
    }

    return config.APIKey, nil
}

// WaitForAPIKey waits for the config.xml to exist and contain an API key
func WaitForAPIKey(ctx context.Context, configPath string, timeout time.Duration) (string, error) {
    deadline := time.Now().Add(timeout)

    for {
        if time.Now().After(deadline) {
            return "", fmt.Errorf("timeout waiting for API key at %s", configPath)
        }

        apiKey, err := DiscoverAPIKey(configPath)
        if err == nil && apiKey != "" {
            return apiKey, nil
        }

        select {
        case <-ctx.Done():
            return "", ctx.Err()
        case <-time.After(10 * time.Second):
            // Retry
        }
    }
}
```

---

## 5. Updated Phase Ordering

With these additional features, the build phases expand:

### Phase 7: Import List Support

| Step | File to Create | Reference |
|------|----------------|-----------|
| 1 | `api/v1alpha1/importlistpolicy_types.go` | Section 1.1 above |
| 2 | `internal/ir/v1/import_lists.go` | Section 1.2 above |
| 3 | `internal/adapters/radarr/import_lists.go` | Section 1.3 above |
| 4 | `internal/adapters/sonarr/import_lists.go` | Similar to Radarr |

### Phase 8: Extended Prowlarr Support

| Step | File to Create | Reference |
|------|----------------|-----------|
| 1 | Extended `api/v1alpha1/indexerpolicy_types.go` | Section 2.1 above |
| 2 | `internal/ir/v1/prowlarr.go` | Section 2.2 above |
| 3 | `internal/adapters/prowlarr/adapter.go` | Section 2.3 above |
| 4 | `internal/adapters/prowlarr/indexers.go` | Section 2.3 above |
| 5 | `internal/adapters/prowlarr/proxies.go` | Section 2.3 above |
| 6 | `internal/adapters/prowlarr/applications.go` | Section 2.3 above |

### Phase 9: Bazarr Support

| Step | File to Create | Reference |
|------|----------------|-----------|
| 1 | `api/v1alpha1/bazarrbinding_types.go` | Section 3.1 above |
| 2 | `internal/adapters/bazarr/generator.go` | Section 3.2 above |

### Phase 10: API Key Discovery

| Step | File to Create | Reference |
|------|----------------|-----------|
| 1 | `internal/discovery/apikey.go` | Section 4.1 above |

---

## 6. Summary of New CRDs

| CRD | Purpose | Target Apps |
|-----|---------|-------------|
| `ImportListPolicy` | Automated media imports from IMDb, Trakt, RSS, etc. | Radarr, Sonarr |
| `IndexerPolicy` (extended) | Now includes Prowlarr-specific indexers, proxies, and app connections | Prowlarr, Radarr, Sonarr |
| `BazarrBinding` | Config.yaml generation for Bazarr | Bazarr |

---

## 7. Related Documents

| Document | Purpose |
|----------|---------|
| [README](./README.md) | Build order, file mapping (start here) |
| [CRDS](./CRDS.md) | Core CRD schemas (ServiceBinding, MediaPolicy, etc.) |
| [TYPES](./TYPES.md) | Core IR types, adapter interface |
| [RADARR](./RADARR.md) | Radarr API mappings |
| [DESIGN](./DESIGN.md) | Architecture and philosophy |
| **This document** | Import lists, Prowlarr integration, Bazarr |
