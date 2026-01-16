# Nebularr - Prowlarr API Mapping Reference

> **For coding agents:** Start with [README.md](./README.md) for build order. This document contains Prowlarr adapter code to copy.
>
> **Related:** [README](./README.md) | [TYPES](./TYPES.md) | [CRDS](./CRDS.md) | [OPERATIONS](./OPERATIONS.md)

This document is a reference for implementing the Prowlarr adapter. Prowlarr is an indexer manager that syncs indexer configurations to Radarr, Sonarr, and Lidarr.

---

## Key Concepts

| Concept | Description |
|---------|-------------|
| **Indexer** | A torrent/usenet source (e.g., 1337x, NZBgeek) configured in Prowlarr |
| **Application** | A connection to Radarr/Sonarr/Lidarr that receives synced indexers |
| **Indexer Proxy** | A proxy for indexer requests (FlareSolverr, HTTP, SOCKS) |
| **Sync Level** | How aggressively Prowlarr syncs indexers to applications |
| **API Version** | v1 (not v3 like Radarr/Sonarr) |

---

## 1. Indexer Management

### 1.1 Indexer Structure

Prowlarr indexers are based on "definitions" - pre-built configurations for known indexer sites:

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Display name for this indexer instance |
| `implementation` | string | Indexer definition name (e.g., "1337x", "Nyaa") |
| `configContract` | string | Settings contract (usually `{Implementation}Settings`) |
| `enable` | bool | Whether indexer is active |
| `protocol` | string | "torrent" or "usenet" |
| `priority` | int | Search priority (1-50, lower = higher priority) |
| `fields` | []Field | Configuration fields (baseUrl, apiKey, etc.) |
| `tags` | []int | Associated tag IDs |

### 1.2 Common Indexer Implementations

#### Public Torrent Indexers (No Auth)

| Implementation | Protocol | Description |
|----------------|----------|-------------|
| `1337x` | torrent | 1337x torrent site |
| `Nyaa` | torrent | Anime torrents |
| `RARBG` | torrent | General torrents |
| `ThePirateBay` | torrent | TPB torrents |
| `YTS` | torrent | Movies/YIFY |
| `EZTV` | torrent | TV shows |
| `LimeTorrents` | torrent | General torrents |
| `TorrentGalaxy` | torrent | General torrents |

#### Private Torrent Indexers (Auth Required)

| Implementation | Protocol | Auth Type | Description |
|----------------|----------|-----------|-------------|
| `IPTorrents` | torrent | Cookie | Private general tracker |
| `TorrentLeech` | torrent | API Key | Private general tracker |
| `BroadcastheNet` | torrent | API Key | Private TV tracker |
| `PassThePopcorn` | torrent | API Key | Private movie tracker |
| `Redacted` | torrent | API Key | Private music tracker |

#### Usenet Indexers

| Implementation | Protocol | Auth Type | Description |
|----------------|----------|-----------|-------------|
| `NZBgeek` | usenet | API Key | General usenet |
| `NZBFinder` | usenet | API Key | General usenet |
| `DrunkenSlug` | usenet | API Key | General usenet |
| `NZBPlanet` | usenet | API Key | General usenet |

### 1.3 Go Implementation

```go
// internal/adapters/prowlarr/indexers.go

package prowlarr

// IndexerConfig represents our abstract indexer config
type IndexerConfig struct {
    Name           string
    Implementation string            // Definition name (e.g., "1337x")
    Protocol       string            // "torrent" or "usenet"
    Enable         bool
    Priority       int               // 1-50, lower = higher priority
    BaseURL        string            // Optional URL override
    Settings       map[string]string // Implementation-specific settings
    Tags           []string          // Tag names
}

// BuildIndexerPayload creates Prowlarr indexer from our config
func BuildIndexerPayload(cfg IndexerConfig, tagIDs []int) map[string]interface{} {
    fields := []map[string]interface{}{}
    
    // Add baseUrl if specified
    if cfg.BaseURL != "" {
        fields = append(fields, map[string]interface{}{
            "name":  "baseUrl",
            "value": cfg.BaseURL,
        })
    }
    
    // Add implementation-specific settings
    for name, value := range cfg.Settings {
        fields = append(fields, map[string]interface{}{
            "name":  name,
            "value": value,
        })
    }
    
    // Determine config contract
    configContract := cfg.Implementation + "Settings"
    
    return map[string]interface{}{
        "name":           cfg.Name,
        "implementation": cfg.Implementation,
        "configContract": configContract,
        "enable":         cfg.Enable,
        "protocol":       cfg.Protocol,
        "priority":       cfg.Priority,
        "fields":         fields,
        "tags":           tagIDs,
    }
}

// CommonPublicIndexers provides pre-built configs for common public indexers
var CommonPublicIndexers = map[string]IndexerConfig{
    "1337x": {
        Implementation: "1337x",
        Protocol:       "torrent",
        Enable:         true,
        Priority:       25,
    },
    "nyaa": {
        Implementation: "Nyaa",
        Protocol:       "torrent",
        Enable:         true,
        Priority:       25,
    },
    "yts": {
        Implementation: "YTS",
        Protocol:       "torrent",
        Enable:         true,
        Priority:       25,
    },
    "eztv": {
        Implementation: "EZTV",
        Protocol:       "torrent",
        Enable:         true,
        Priority:       25,
    },
}
```

---

## 2. Indexer Proxy Management

### 2.1 Proxy Types

Prowlarr supports proxies for indexer requests:

| Type | Implementation | Config Contract | Description |
|------|----------------|-----------------|-------------|
| FlareSolverr | `FlareSolverr` | `FlareSolverrSettings` | Cloudflare bypass proxy |
| HTTP | `Http` | `HttpSettings` | Standard HTTP proxy |
| SOCKS4 | `Socks4` | `Socks4Settings` | SOCKS4 proxy |
| SOCKS5 | `Socks5` | `Socks5Settings` | SOCKS5 proxy |

### 2.2 FlareSolverr Fields

| Field | Type | Description |
|-------|------|-------------|
| `host` | string | FlareSolverr URL (e.g., `http://flaresolverr:8191`) |
| `requestTimeout` | int | Request timeout in seconds (default: 60) |

### 2.3 HTTP/SOCKS Proxy Fields

| Field | Type | Description |
|-------|------|-------------|
| `host` | string | Proxy hostname |
| `port` | int | Proxy port |
| `username` | string | Auth username (optional) |
| `password` | string | Auth password (optional) |

### 2.4 Go Implementation

```go
// internal/adapters/prowlarr/proxies.go

package prowlarr

// ProxyType represents the type of indexer proxy
type ProxyType string

const (
    ProxyTypeFlareSolverr ProxyType = "flaresolverr"
    ProxyTypeHTTP         ProxyType = "http"
    ProxyTypeSOCKS4       ProxyType = "socks4"
    ProxyTypeSOCKS5       ProxyType = "socks5"
)

// ProxyImplementationMap maps our types to Prowlarr implementations
var ProxyImplementationMap = map[ProxyType]string{
    ProxyTypeFlareSolverr: "FlareSolverr",
    ProxyTypeHTTP:         "Http",
    ProxyTypeSOCKS4:       "Socks4",
    ProxyTypeSOCKS5:       "Socks5",
}

// ProxyConfigContractMap maps our types to Prowlarr config contracts
var ProxyConfigContractMap = map[ProxyType]string{
    ProxyTypeFlareSolverr: "FlareSolverrSettings",
    ProxyTypeHTTP:         "HttpSettings",
    ProxyTypeSOCKS4:       "Socks4Settings",
    ProxyTypeSOCKS5:       "Socks5Settings",
}

// IndexerProxyConfig represents our abstract proxy config
type IndexerProxyConfig struct {
    Name           string
    Type           ProxyType
    Host           string
    Port           int      // For HTTP/SOCKS only
    Username       string   // For HTTP/SOCKS only
    Password       string   // For HTTP/SOCKS only
    RequestTimeout int      // For FlareSolverr only (seconds)
    Tags           []string // Tag names to associate
}

// BuildProxyPayload creates Prowlarr proxy from our config
func BuildProxyPayload(cfg IndexerProxyConfig, tagIDs []int) map[string]interface{} {
    implementation := ProxyImplementationMap[cfg.Type]
    configContract := ProxyConfigContractMap[cfg.Type]
    
    var fields []map[string]interface{}
    
    if cfg.Type == ProxyTypeFlareSolverr {
        fields = []map[string]interface{}{
            {"name": "host", "value": cfg.Host},
            {"name": "requestTimeout", "value": cfg.RequestTimeout},
        }
    } else {
        fields = []map[string]interface{}{
            {"name": "host", "value": cfg.Host},
            {"name": "port", "value": cfg.Port},
        }
        if cfg.Username != "" {
            fields = append(fields, map[string]interface{}{
                "name": "username", "value": cfg.Username,
            })
        }
        if cfg.Password != "" {
            fields = append(fields, map[string]interface{}{
                "name": "password", "value": cfg.Password,
            })
        }
    }
    
    return map[string]interface{}{
        "name":           cfg.Name,
        "implementation": implementation,
        "configContract": configContract,
        "fields":         fields,
        "tags":           tagIDs,
    }
}

// DefaultFlareSolverr returns a sensible FlareSolverr config
func DefaultFlareSolverr(host string) IndexerProxyConfig {
    return IndexerProxyConfig{
        Name:           "FlareSolverr",
        Type:           ProxyTypeFlareSolverr,
        Host:           host,
        RequestTimeout: 60,
    }
}
```

---

## 3. Application Management

### 3.1 Application Types

Applications are connections to downstream *arr apps:

| Type | Implementation | Config Contract | Description |
|------|----------------|-----------------|-------------|
| Radarr | `Radarr` | `RadarrSettings` | Movie management |
| Sonarr | `Sonarr` | `SonarrSettings` | TV management |
| Lidarr | `Lidarr` | `LidarrSettings` | Music management |

### 3.2 Application Fields

| Field | Type | Description |
|-------|------|-------------|
| `prowlarrUrl` | string | Prowlarr's own URL (for sync back) |
| `baseUrl` | string | Target app's URL |
| `apiKey` | string | Target app's API key |
| `syncCategories` | []int | Newznab category IDs to sync |

### 3.3 Sync Levels

| Level | API Value | Description |
|-------|-----------|-------------|
| Disabled | `disabled` | No sync |
| Add Only | `addOnly` | Add new indexers, don't remove |
| Full Sync | `fullSync` | Add and remove to match Prowlarr |

### 3.4 Sync Categories by App Type

| App Type | Typical Sync Categories |
|----------|------------------------|
| Radarr | 2000, 2010, 2020, 2030, 2040, 2045, 2050, 2060 (Movies) |
| Sonarr | 5000, 5020, 5030, 5040, 5045, 5070 (TV) |
| Lidarr | 3000, 3010, 3030, 3040 (Audio) |

### 3.5 Go Implementation

```go
// internal/adapters/prowlarr/applications.go

package prowlarr

// AppType represents the type of downstream application
type AppType string

const (
    AppTypeRadarr AppType = "radarr"
    AppTypeSonarr AppType = "sonarr"
    AppTypeLidarr AppType = "lidarr"
)

// SyncLevel represents how aggressively to sync
type SyncLevel string

const (
    SyncLevelDisabled SyncLevel = "disabled"
    SyncLevelAddOnly  SyncLevel = "addOnly"
    SyncLevelFullSync SyncLevel = "fullSync"
)

// AppImplementationMap maps our types to Prowlarr implementations
var AppImplementationMap = map[AppType]string{
    AppTypeRadarr: "Radarr",
    AppTypeSonarr: "Sonarr",
    AppTypeLidarr: "Lidarr",
}

// AppConfigContractMap maps our types to Prowlarr config contracts
var AppConfigContractMap = map[AppType]string{
    AppTypeRadarr: "RadarrSettings",
    AppTypeSonarr: "SonarrSettings",
    AppTypeLidarr: "LidarrSettings",
}

// DefaultSyncCategories returns typical categories for each app type
var DefaultSyncCategories = map[AppType][]int{
    AppTypeRadarr: {2000, 2010, 2020, 2030, 2040, 2045, 2050, 2060}, // Movies
    AppTypeSonarr: {5000, 5020, 5030, 5040, 5045, 5070},             // TV
    AppTypeLidarr: {3000, 3010, 3030, 3040},                         // Audio
}

// ApplicationConfig represents our abstract application config
type ApplicationConfig struct {
    Name           string
    Type           AppType
    URL            string    // Target app URL
    APIKey         string    // Target app API key
    ProwlarrURL    string    // Prowlarr's own URL
    SyncCategories []int     // Categories to sync
    SyncLevel      SyncLevel
    Tags           []string  // Tag names
}

// BuildApplicationPayload creates Prowlarr application from our config
func BuildApplicationPayload(cfg ApplicationConfig, tagIDs []int) map[string]interface{} {
    implementation := AppImplementationMap[cfg.Type]
    configContract := AppConfigContractMap[cfg.Type]
    
    // Use default categories if not specified
    syncCategories := cfg.SyncCategories
    if len(syncCategories) == 0 {
        syncCategories = DefaultSyncCategories[cfg.Type]
    }
    
    fields := []map[string]interface{}{
        {"name": "prowlarrUrl", "value": cfg.ProwlarrURL},
        {"name": "baseUrl", "value": cfg.URL},
        {"name": "apiKey", "value": cfg.APIKey},
        {"name": "syncCategories", "value": syncCategories},
    }
    
    return map[string]interface{}{
        "name":           cfg.Name,
        "implementation": implementation,
        "configContract": configContract,
        "syncLevel":      string(cfg.SyncLevel),
        "fields":         fields,
        "tags":           tagIDs,
    }
}
```

---

## 4. API Key Discovery

### 4.1 Overview

Prowlarr applications need API keys from target apps. In Kubernetes, these can be discovered from the target app's `config.xml` file.

### 4.2 Config.xml Location Convention

| App | Default Path |
|-----|--------------|
| Radarr | `/radarr-config/config.xml` |
| Sonarr | `/sonarr-config/config.xml` |
| Lidarr | `/lidarr-config/config.xml` |

### 4.3 Go Implementation

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

// ArrConfig represents the structure of an *arr config.xml
type ArrConfig struct {
    XMLName xml.Name `xml:"Config"`
    ApiKey  string   `xml:"ApiKey"`
}

// ParseAPIKey reads API key from config.xml
func ParseAPIKey(path string) (string, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return "", fmt.Errorf("failed to read config: %w", err)
    }
    
    var config ArrConfig
    if err := xml.Unmarshal(data, &config); err != nil {
        return "", fmt.Errorf("failed to parse config: %w", err)
    }
    
    if config.ApiKey == "" {
        return "", fmt.Errorf("API key not found in config")
    }
    
    return config.ApiKey, nil
}

// WaitForAPIKey waits for config.xml to appear and contain an API key
func WaitForAPIKey(ctx context.Context, path string, timeout time.Duration) (string, error) {
    deadline := time.Now().Add(timeout)
    ticker := time.NewTicker(5 * time.Second)
    defer ticker.Stop()
    
    for {
        select {
        case <-ctx.Done():
            return "", ctx.Err()
        case <-ticker.C:
            if time.Now().After(deadline) {
                return "", fmt.Errorf("timeout waiting for API key at %s", path)
            }
            
            apiKey, err := ParseAPIKey(path)
            if err == nil {
                return apiKey, nil
            }
            // Continue waiting if file doesn't exist or key not present
        }
    }
}

// GetConfigPath returns the conventional config.xml path for an app type
func GetConfigPath(appType string) string {
    return fmt.Sprintf("/%s-config/config.xml", appType)
}
```

---

## 5. Capability Discovery Endpoints

### 5.1 API Endpoints

**Note:** Prowlarr uses API v1.

| Capability | Endpoint | Description |
|------------|----------|-------------|
| Indexers | `GET /api/v1/indexer` | List configured indexers |
| Indexer schema | `GET /api/v1/indexer/schema` | Available indexer definitions |
| Applications | `GET /api/v1/applications` | List app connections |
| Application schema | `GET /api/v1/applications/schema` | Available app types |
| Indexer proxies | `GET /api/v1/indexerproxy` | List proxies |
| Proxy schema | `GET /api/v1/indexerproxy/schema` | Available proxy types |
| Download clients | `GET /api/v1/downloadclient` | List download clients |
| Tags | `GET /api/v1/tag` | List tags |
| System status | `GET /api/v1/system/status` | Version info |

### 5.2 Testing Endpoints

| Resource | Test Endpoint | Method |
|----------|---------------|--------|
| Indexer | `/api/v1/indexer/test` | POST |
| Application | `/api/v1/applications/test` | POST |
| Download client | `/api/v1/downloadclient/test` | POST |

### 5.3 Go Implementation

```go
// internal/adapters/prowlarr/adapter.go

package prowlarr

import (
    "context"
    "fmt"
    "log/slog"
    "time"
    
    "github.com/poiley/nebularr/internal/adapters"
    "github.com/poiley/nebularr/internal/adapters/prowlarr/client"
)

// Adapter implements the adapters.Adapter interface for Prowlarr
type Adapter struct {
    client *client.ClientWithResponses
}

// NewAdapter creates a new Prowlarr adapter
func NewAdapter() *Adapter {
    return &Adapter{}
}

// Name returns the adapter identifier
func (a *Adapter) Name() string {
    return "prowlarr"
}

// SupportedApp returns the app this adapter handles
func (a *Adapter) SupportedApp() string {
    return "prowlarr"
}

// APIVersion returns the API version used by Prowlarr
func (a *Adapter) APIVersion() string {
    return "v1"
}

// Discover queries Prowlarr for available features
func (a *Adapter) Discover(ctx context.Context, conn *adapters.Connection) (*adapters.Capabilities, error) {
    caps := &adapters.Capabilities{
        DiscoveredAt: time.Now(),
    }
    
    // Get available indexer definitions
    indexerSchemas, err := a.client.GetIndexerSchema(ctx)
    if err != nil {
        slog.Warn("failed to get indexer schema", "error", err)
    } else {
        for _, schema := range indexerSchemas {
            caps.IndexerDefinitions = append(caps.IndexerDefinitions, schema.Implementation)
        }
    }
    
    // Get available proxy types
    proxySchemas, err := a.client.GetIndexerProxySchema(ctx)
    if err != nil {
        slog.Warn("failed to get proxy schema", "error", err)
    } else {
        for _, schema := range proxySchemas {
            caps.ProxyTypes = append(caps.ProxyTypes, schema.Implementation)
        }
    }
    
    // Get available application types
    appSchemas, err := a.client.GetApplicationSchema(ctx)
    if err != nil {
        slog.Warn("failed to get application schema", "error", err)
    } else {
        for _, schema := range appSchemas {
            caps.ApplicationTypes = append(caps.ApplicationTypes, schema.Implementation)
        }
    }
    
    return caps, nil
}
```

---

## 6. Reconciliation Flow

### 6.1 Prowlarr Reconciliation Order

When reconciling Prowlarr configuration, the order matters:

1. **Tags** - Create ownership tag first
2. **Indexer Proxies** - Create proxies before indexers that use them
3. **Indexers** - Create indexers (may reference proxies via tags)
4. **Applications** - Create app connections (discovers API keys)
5. **Sync** - Prowlarr automatically syncs indexers to applications

### 6.2 Go Implementation

```go
// internal/controller/prowlarr_controller.go

func (r *ProwlarrReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    log := log.FromContext(ctx)
    
    // Fetch the ProwlarrConfig instance
    var config v1alpha1.ProwlarrConfig
    if err := r.Get(ctx, req.NamespacedName, &config); err != nil {
        return ctrl.Result{}, client.IgnoreNotFound(err)
    }
    
    // Connect to Prowlarr
    conn, err := r.resolveConnection(ctx, &config)
    if err != nil {
        return ctrl.Result{}, r.setErrorCondition(ctx, &config, "ConnectionFailed", err)
    }
    
    // Run reconciliation
    if err := r.reconcile(ctx, &config, conn); err != nil {
        return ctrl.Result{}, r.setErrorCondition(ctx, &config, "ReconcileFailed", err)
    }
    
    return ctrl.Result{RequeueAfter: config.Spec.SyncInterval.Duration}, nil
}

func (r *ProwlarrReconciler) reconcile(ctx context.Context, config *v1alpha1.ProwlarrConfig, conn *adapters.Connection) error {
    // Step 1: Ensure ownership tag exists
    tagID, err := r.adapter.EnsureOwnershipTag(ctx, conn)
    if err != nil {
        return fmt.Errorf("failed to ensure tag: %w", err)
    }
    
    // Step 2: Reconcile indexer proxies (before indexers that may use them)
    if config.Spec.Proxies != nil {
        if err := r.reconcileProxies(ctx, conn, config.Spec.Proxies, tagID); err != nil {
            return fmt.Errorf("failed to reconcile proxies: %w", err)
        }
    }
    
    // Step 3: Reconcile indexers
    if config.Spec.Indexers != nil {
        if err := r.reconcileIndexers(ctx, conn, config.Spec.Indexers, tagID); err != nil {
            return fmt.Errorf("failed to reconcile indexers: %w", err)
        }
    }
    
    // Step 4: Reconcile applications (auto-discovers API keys, syncs indexers)
    if config.Spec.Applications != nil {
        if err := r.reconcileApplications(ctx, conn, config.Spec.Applications, tagID); err != nil {
            return fmt.Errorf("failed to reconcile applications: %w", err)
        }
    }
    
    return nil
}
```

---

## 7. Granular CRD for Prowlarr

### 7.1 ProwlarrIndexerPolicy (Granular Path)

For users who want granular control over Prowlarr indexers separate from the bundled `ProwlarrConfig`, the `ProwlarrIndexerPolicy` CRD provides direct indexer management.

See [CRDS.md Section 4.4](./CRDS.md) for the authoritative CRD definition.

```go
// api/v1alpha1/prowlarrindexerpolicy_types.go

// ProwlarrIndexerPolicySpec defines indexer configuration for Prowlarr
type ProwlarrIndexerPolicySpec struct {
    // Indexers to configure in Prowlarr
    Indexers []IndexerSpec `json:"indexers,omitempty"`
    
    // Proxies for indexer requests
    Proxies []ProxySpec `json:"proxies,omitempty"`
    
    // Application connections to sync indexers to
    Applications []ApplicationSpec `json:"applications,omitempty"`
}

type IndexerSpec struct {
    // Name for this indexer instance
    Name string `json:"name"`
    
    // Indexer definition (e.g., "1337x", "Nyaa")
    Definition string `json:"definition"`
    
    // Protocol: "torrent" or "usenet"
    Protocol string `json:"protocol,omitempty"`
    
    // Enable this indexer
    Enable bool `json:"enable,omitempty"`
    
    // Search priority (1-50, lower = higher priority)
    Priority int `json:"priority,omitempty"`
    
    // Optional base URL override
    BaseURL string `json:"baseUrl,omitempty"`
    
    // Implementation-specific settings
    Settings map[string]string `json:"settings,omitempty"`
    
    // Tags to associate (e.g., for proxy assignment)
    Tags []string `json:"tags,omitempty"`
}

type ProxySpec struct {
    // Name for this proxy
    Name string `json:"name"`
    
    // Type: flaresolverr, http, socks4, socks5
    Type string `json:"type"`
    
    // Host URL or hostname
    Host string `json:"host"`
    
    // Port (for http/socks only)
    Port int `json:"port,omitempty"`
    
    // Request timeout in seconds (for flaresolverr)
    RequestTimeout int `json:"requestTimeout,omitempty"`
    
    // Auth credentials (for http/socks)
    CredentialsSecretRef *SecretRef `json:"credentialsSecretRef,omitempty"`
    
    // Tags to associate with this proxy
    Tags []string `json:"tags,omitempty"`
}

type ApplicationSpec struct {
    // Name for this application connection
    Name string `json:"name"`
    
    // Type: radarr, sonarr, lidarr
    Type string `json:"type"`
    
    // Target application URL
    URL string `json:"url"`
    
    // API key source (secret ref or config.xml path)
    APIKeySource APIKeySource `json:"apiKeySource"`
    
    // Categories to sync (uses defaults if not specified)
    SyncCategories []int `json:"syncCategories,omitempty"`
    
    // Sync level: disabled, addOnly, fullSync
    SyncLevel string `json:"syncLevel,omitempty"`
}

type APIKeySource struct {
    // Reference to a K8s secret containing the API key
    SecretRef *SecretRef `json:"secretRef,omitempty"`
    
    // Path to config.xml (for auto-discovery)
    ConfigPath string `json:"configPath,omitempty"`
}
```

---

## 8. IR Types for Prowlarr

### 8.1 Prowlarr IR Types

```go
// internal/ir/v1/prowlarr.go

package v1

// ProwlarrIR is the intermediate representation for Prowlarr configuration
type ProwlarrIR struct {
    // Indexers to configure
    Indexers []IndexerIR `json:"indexers,omitempty"`
    
    // Proxies for indexer requests
    Proxies []IndexerProxyIR `json:"proxies,omitempty"`
    
    // Application connections
    Applications []ApplicationIR `json:"applications,omitempty"`
}

// IndexerIR represents a Prowlarr indexer
type IndexerIR struct {
    Name           string            `json:"name"`
    Implementation string            `json:"implementation"`
    Protocol       string            `json:"protocol"`
    Enable         bool              `json:"enable"`
    Priority       int               `json:"priority"`
    BaseURL        string            `json:"baseUrl,omitempty"`
    Settings       map[string]string `json:"settings,omitempty"`
    Tags           []string          `json:"tags,omitempty"`
}

// IndexerProxyIR represents an indexer proxy
type IndexerProxyIR struct {
    Name           string   `json:"name"`
    Type           string   `json:"type"` // flaresolverr, http, socks4, socks5
    Host           string   `json:"host"`
    Port           int      `json:"port,omitempty"`
    Username       string   `json:"username,omitempty"`
    Password       string   `json:"password,omitempty"`
    RequestTimeout int      `json:"requestTimeout,omitempty"`
    Tags           []string `json:"tags,omitempty"`
}

// ApplicationIR represents an app connection
type ApplicationIR struct {
    Name           string `json:"name"`
    Type           string `json:"type"` // radarr, sonarr, lidarr
    URL            string `json:"url"`
    APIKey         string `json:"apiKey"`
    ProwlarrURL    string `json:"prowlarrUrl"`
    SyncCategories []int  `json:"syncCategories,omitempty"`
    SyncLevel      string `json:"syncLevel"`
}
```

---

## 9. Ownership Tagging

Same pattern as other adapters:

```go
const OwnershipTagName = "nebularr-managed"

// EnsureOwnershipTag creates the ownership tag if it doesn't exist
func (a *Adapter) EnsureOwnershipTag(ctx context.Context) (int, error) {
    tags, err := a.client.ListTags(ctx)
    if err != nil {
        return 0, err
    }
    
    for _, tag := range tags {
        if tag.Label == OwnershipTagName {
            return tag.ID, nil
        }
    }
    
    newTag, err := a.client.CreateTag(ctx, OwnershipTagName)
    if err != nil {
        return 0, err
    }
    
    return newTag.ID, nil
}
```

---

## 10. Related Documents

- [README](./README.md) - Build order, file mapping (start here)
- [RADARR](./RADARR.md) - Radarr adapter (sync target)
- [SONARR](./SONARR.md) - Sonarr adapter (sync target)
- [LIDARR](./LIDARR.md) - Lidarr adapter (sync target)
- [TYPES](./TYPES.md) - IR types and adapter interface
- [CRDS](./CRDS.md) - CRD definitions
- [OPERATIONS](./OPERATIONS.md) - Auto-discovery, Prowlarr integration patterns
