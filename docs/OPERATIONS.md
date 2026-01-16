# Nebularr - Operations & Design Decisions

> **For coding agents:** This document covers operational concerns for implementation.
>
> **Related:** [README](./README.md) | [TYPES](./TYPES.md) | [CRDS](./CRDS.md) | [PRESETS](./PRESETS.md)

This document covers secret management, auto-discovery, defaults merging, multi-instance support, conflict resolution, error handling, and testing strategies.

---

## 1. Secret Management

### 1.1 Secret Sources

Nebularr needs to access API keys for *arr applications and credentials for download clients/indexers.

| Source | Description | Priority |
|--------|-------------|----------|
| **Explicit Secret** | API key stored in Kubernetes Secret | Highest |
| **Config.xml Auto-Discovery** | Read API key from mounted config.xml | Fallback |
| **Convention-Based** | Look for `{app-name}-api-key` secret | Last resort |

### 1.2 Auto-Discovery Flow

```go
// internal/secrets/resolver.go

package secrets

import (
    "context"
    "fmt"
    "os"
    "path/filepath"

    corev1 "k8s.io/api/core/v1"
    "sigs.k8s.io/controller-runtime/pkg/client"

    "github.com/poiley/nebularr/internal/discovery"
)

// Resolver resolves API keys and credentials from various sources
type Resolver struct {
    client client.Client
}

// NewResolver creates a new secret resolver
func NewResolver(c client.Client) *Resolver {
    return &Resolver{client: c}
}

// ResolveAPIKey resolves an API key using the priority chain
func (r *Resolver) ResolveAPIKey(ctx context.Context, namespace string, spec ConnectionSpec) (string, string, error) {
    // Priority 1: Explicit secret reference
    if spec.APIKeySecretRef != nil {
        apiKey, err := r.resolveFromSecret(ctx, namespace, spec.APIKeySecretRef)
        if err == nil {
            return apiKey, "secret", nil
        }
        // If explicit secret is specified but not found, fail fast
        return "", "", fmt.Errorf("explicit secret not found: %w", err)
    }

    // Priority 2: Config.xml auto-discovery
    if spec.ConfigPath != "" {
        apiKey, err := r.resolveFromConfig(spec.ConfigPath)
        if err == nil {
            return apiKey, "config.xml", nil
        }
        // Log warning, continue to convention-based
    }

    // Priority 3: Default config.xml path (/config/config.xml)
    defaultPath := "/config/config.xml"
    if apiKey, err := r.resolveFromConfig(defaultPath); err == nil {
        return apiKey, "config.xml (default path)", nil
    }

    // Priority 4: Convention-based secret name
    appName := extractAppName(spec.URL) // radarr, sonarr, etc.
    conventionName := fmt.Sprintf("%s-api-key", appName)
    apiKey, err := r.resolveFromSecret(ctx, namespace, &SecretKeySelector{
        Name: conventionName,
        Key:  "apiKey",
    })
    if err == nil {
        return apiKey, "convention", nil
    }

    return "", "", fmt.Errorf("could not resolve API key: no source available")
}

func (r *Resolver) resolveFromSecret(ctx context.Context, namespace string, ref *SecretKeySelector) (string, error) {
    key := ref.Key
    if key == "" {
        key = "apiKey"
    }

    secret := &corev1.Secret{}
    if err := r.client.Get(ctx, client.ObjectKey{
        Namespace: namespace,
        Name:      ref.Name,
    }, secret); err != nil {
        return "", fmt.Errorf("failed to get secret %s/%s: %w", namespace, ref.Name, err)
    }

    apiKey, ok := secret.Data[key]
    if !ok {
        return "", fmt.Errorf("key %q not found in secret %s/%s", key, namespace, ref.Name)
    }

    return string(apiKey), nil
}

func (r *Resolver) resolveFromConfig(path string) (string, error) {
    return discovery.ParseAPIKey(path)
}

// Error Conditions - surfaced clearly in CRD Status.Conditions:
//
// | Condition Type | Reason | Message Example |
// |----------------|--------|-----------------|
// | SecretResolved | SecretNotFound | "Secret 'radarr-api-key' not found in namespace 'media'" |
// | SecretResolved | KeyNotFound | "Key 'apiKey' not found in secret 'radarr-api-key'" |
// | SecretResolved | ConfigXmlNotReadable | "Failed to read /config/config.xml: permission denied" |
// | SecretResolved | ConfigXmlInvalid | "Failed to parse /config/config.xml: invalid XML" |
// | SecretResolved | NoSourceAvailable | "No API key source available (tried: secretRef, config.xml, convention)" |
// | SecretResolved | True | "API key resolved from secret 'radarr-api-key'" |
```

### 1.3 Config.xml Parser

```go
// internal/discovery/apikey.go

package discovery

import (
    "encoding/xml"
    "fmt"
    "os"
)

// Config represents the relevant parts of *arr config.xml
type Config struct {
    XMLName xml.Name `xml:"Config"`
    ApiKey  string   `xml:"ApiKey"`
}

// ParseAPIKey extracts the API key from a config.xml file
func ParseAPIKey(path string) (string, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return "", fmt.Errorf("failed to read config.xml: %w", err)
    }

    var config Config
    if err := xml.Unmarshal(data, &config); err != nil {
        return "", fmt.Errorf("failed to parse config.xml: %w", err)
    }

    if config.ApiKey == "" {
        return "", fmt.Errorf("ApiKey not found in config.xml")
    }

    return config.ApiKey, nil
}
```

### 1.4 RBAC Requirements

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: nebularr-manager-role
rules:
  # CRD permissions
  - apiGroups: ["arr.rinzler.cloud"]
    resources: ["*"]
    verbs: ["*"]
  # Secret read permission
  - apiGroups: [""]
    resources: ["secrets"]
    verbs: ["get", "list", "watch"]
  # ConfigMap read/write (for Bazarr)
  - apiGroups: [""]
    resources: ["configmaps"]
    verbs: ["get", "list", "watch", "create", "update", "patch"]
```

---

## 2. Download Client Type Inference

### 2.1 Inference Rules

When `type` is not specified in DownloadClientSpec, infer from name:

```go
// internal/discovery/download_client.go

package discovery

import (
    "strings"
)

// InferDownloadClientType infers client type from name
func InferDownloadClientType(name string) string {
    lower := strings.ToLower(name)

    // Torrent clients
    switch {
    case strings.Contains(lower, "qbittorrent"), strings.Contains(lower, "qbit"):
        return "qbittorrent"
    case strings.Contains(lower, "transmission"):
        return "transmission"
    case strings.Contains(lower, "deluge"):
        return "deluge"
    case strings.Contains(lower, "rtorrent"), strings.Contains(lower, "rutorrent"):
        return "rtorrent"
    case strings.Contains(lower, "flood"):
        return "flood"
    case strings.Contains(lower, "aria"):
        return "aria2"

    // Usenet clients
    case strings.Contains(lower, "nzbget"):
        return "nzbget"
    case strings.Contains(lower, "sabnzbd"), strings.Contains(lower, "sab"):
        return "sabnzbd"

    default:
        return "" // Unknown, user must specify
    }
}

// InferProtocol infers protocol from client type
func InferProtocol(clientType string) string {
    switch clientType {
    case "qbittorrent", "transmission", "deluge", "rtorrent", "flood", "aria2":
        return "torrent"
    case "nzbget", "sabnzbd":
        return "usenet"
    default:
        return "torrent" // Default
    }
}
```

### 2.2 URL-Based Port Inference

```go
// InferPort returns default port for a client type
func InferPort(clientType string, urlHasPort bool) int {
    if urlHasPort {
        return 0 // Use URL port
    }

    switch clientType {
    case "qbittorrent":
        return 8080
    case "transmission":
        return 9091
    case "deluge":
        return 8112
    case "rtorrent":
        return 8080
    case "nzbget":
        return 6789
    case "sabnzbd":
        return 8080
    default:
        return 0
    }
}
```

---

## 3. Defaults Merge Rules

### 3.1 Merge Hierarchy

Configuration merges follow this priority (later overrides earlier):

```
ClusterNebularrDefaults  (lowest priority)
        ↓
   NebularrDefaults      (namespace-scoped)
        ↓
   BundledConfig         (RadarrConfig, SonarrConfig, etc.)
        ↓
   GranularPolicies      (highest priority)
```

### 3.2 Merge Behavior by Field Type

| Field Type | Merge Behavior |
|------------|----------------|
| **Scalar** (string, int, bool) | Later value wins |
| **Struct** | Deep merge (field by field) |
| **Array** | Replace entire array (no merge) |
| **Map** | Merge keys (later wins per key) |
| **Nil/Omitted** | Skip (don't override with nil) |

### 3.2.1 Policy Overlay Merge Semantics

When a granular policy (e.g., `RadarrMediaPolicy`) coexists with a bundled config (`RadarrConfig`), the policy **replaces entire sections**, not individual fields:

| Config Section | Merge Behavior |
|----------------|----------------|
| **Quality** | Policy replaces entirely if present; config used if policy omits |
| **DownloadClients** | Policy replaces entire list if present |
| **Indexers** | Policy replaces entirely if present |
| **Naming** | Policy replaces entirely if present |
| **RootFolders** | Policy replaces entire list if present |
| **Connection** | Never from policy (always from bundled config) |

**Example:**
```yaml
# RadarrConfig has:
quality:
  preset: "4k-hdr"
downloadClients:
  - name: qbit1
  - name: qbit2

# RadarrMediaPolicy has:
quality:
  preset: "1080p-quality"
# (no downloadClients section)

# Result:
quality: preset: "1080p-quality"  # From policy
downloadClients: [qbit1, qbit2]   # From config (policy didn't specify)
```

### 3.3 Implementation

```go
// internal/compiler/merge.go

package compiler

import (
    "reflect"

    arrv1alpha1 "github.com/poiley/nebularr/api/v1alpha1"
)

// DefaultsMerger handles the merge hierarchy
type DefaultsMerger struct{}

// MergeRadarrConfig applies defaults to a RadarrConfig
func (m *DefaultsMerger) MergeRadarrConfig(
    cluster *arrv1alpha1.ClusterNebularrDefaults,
    namespace *arrv1alpha1.NebularrDefaults,
    config *arrv1alpha1.RadarrConfig,
) *arrv1alpha1.RadarrConfig {
    result := config.DeepCopy()

    // Apply cluster defaults
    if cluster != nil {
        m.applyVideoQualityDefaults(&result.Spec.Quality, cluster.Spec.VideoQuality)
        m.applyNamingDefaults(&result.Spec.Naming, cluster.Spec.Naming)
        m.applyDownloadClientDefaults(&result.Spec.DownloadClients, cluster.Spec.DownloadClients)
        m.applyReconciliationDefaults(&result.Spec.Reconciliation, cluster.Spec.Reconciliation)
    }

    // Apply namespace defaults (overrides cluster)
    if namespace != nil {
        m.applyVideoQualityDefaults(&result.Spec.Quality, namespace.Spec.VideoQuality)
        m.applyNamingDefaults(&result.Spec.Naming, namespace.Spec.Naming)
        m.applyDownloadClientDefaults(&result.Spec.DownloadClients, namespace.Spec.DownloadClients)
        m.applyReconciliationDefaults(&result.Spec.Reconciliation, namespace.Spec.Reconciliation)
    }

    return result
}

// applyVideoQualityDefaults applies defaults only if target field is nil/empty
func (m *DefaultsMerger) applyVideoQualityDefaults(target **arrv1alpha1.VideoQualitySpec, defaults *arrv1alpha1.VideoQualitySpec) {
    if defaults == nil {
        return
    }

    if *target == nil {
        // No config-level quality, use entire defaults
        *target = defaults.DeepCopy()
        return
    }

    // Config has quality, only fill in missing fields
    t := *target
    if t.Preset == "" && len(t.Tiers) == 0 && defaults.Preset != "" {
        t.Preset = defaults.Preset
    }
    // Don't merge excludes/preferAdditional - those are user intent
}

func (m *DefaultsMerger) applyNamingDefaults(target **arrv1alpha1.NamingSpec, defaults *arrv1alpha1.NamingSpec) {
    if defaults == nil {
        return
    }

    if *target == nil {
        *target = defaults.DeepCopy()
        return
    }

    t := *target
    if t.Preset == "" && defaults.Preset != "" {
        t.Preset = defaults.Preset
    }
}

// applyDownloadClientDefaults merges download clients by name
func (m *DefaultsMerger) applyDownloadClientDefaults(target *[]arrv1alpha1.DownloadClientSpec, defaults []arrv1alpha1.DownloadClientSpec) {
    if len(defaults) == 0 {
        return
    }

    if len(*target) == 0 {
        // No config-level clients, use defaults
        *target = make([]arrv1alpha1.DownloadClientSpec, len(defaults))
        copy(*target, defaults)
        return
    }

    // Config has clients - don't merge, config wins entirely
    // (Merging arrays by name is complex and error-prone)
}

func (m *DefaultsMerger) applyReconciliationDefaults(target **arrv1alpha1.ReconciliationSpec, defaults *arrv1alpha1.ReconciliationSpec) {
    if defaults == nil {
        return
    }

    if *target == nil {
        *target = defaults.DeepCopy()
        return
    }

    t := *target
    if t.Interval == nil && defaults.Interval != nil {
        t.Interval = defaults.Interval
    }
}
```

### 3.4 Policy Overlay

Granular policies override specific sections of the bundled config:

```go
// ApplyRadarrMediaPolicy overlays a media policy on a config
func (m *DefaultsMerger) ApplyRadarrMediaPolicy(
    config *arrv1alpha1.RadarrConfig,
    policy *arrv1alpha1.RadarrMediaPolicy,
) *arrv1alpha1.RadarrConfig {
    result := config.DeepCopy()

    // Policy quality completely replaces config quality
    result.Spec.Quality = &policy.Spec.Quality

    // Policy formats override if specified
    if policy.Spec.Formats != nil {
        // Apply format overrides
    }

    return result
}
```

---

## 4. Multi-Instance Support

### 4.1 Design

Each *arrConfig CRD represents one app instance. Multiple instances = multiple CRDs.

```yaml
# Radarr instance 1 (4K movies)
apiVersion: arr.rinzler.cloud/v1alpha1
kind: RadarrConfig
metadata:
  name: radarr-4k
  namespace: media
spec:
  connection:
    url: http://radarr-4k:7878
  quality:
    preset: "4k-hdr"
  rootFolders:
    - /movies/4k
---
# Radarr instance 2 (1080p movies)
apiVersion: arr.rinzler.cloud/v1alpha1
kind: RadarrConfig
metadata:
  name: radarr-1080p
  namespace: media
spec:
  connection:
    url: http://radarr-1080p:7878
  quality:
    preset: "1080p-quality"
  rootFolders:
    - /movies/hd
```

### 4.2 Namespace Scoping

- All CRDs are namespace-scoped (except ClusterNebularrDefaults)
- CRDs in different namespaces are completely independent
- Policies reference configs by name within the same namespace

---

## 5. Conflict Resolution

### 5.1 URL Uniqueness

**Rule:** Each URL can only be managed by one CRD.

Validation webhook rejects a second CRD targeting the same URL:

```go
// internal/webhook/radarrconfig_webhook.go

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

### 5.2 Manual Modification Handling

Resources created by Nebularr are tagged. On reconciliation:

1. **Detect changes:** Compare current state with desired IR
2. **Re-apply:** Overwrite manual changes with desired state
3. **Log warning:** Alert user that changes were reverted

```go
func (r *Reconciler) reconcileQualityProfile(ctx context.Context, desired *QualityProfile, current *QualityProfile) error {
    if !reflect.DeepEqual(desired, current) {
        if isOwnedByNebularr(current) {
            log.Warn("overwriting manual modification",
                "resource", current.Name,
                "type", "QualityProfile")
        }
        return r.adapter.UpdateQualityProfile(ctx, desired)
    }
    return nil
}
```

### 5.3 Orphan Cleanup

When a config or policy is deleted:

1. Controller receives delete event
2. Finalizer triggers cleanup
3. All tagged resources are deleted

```go
func (r *Reconciler) handleDeletion(ctx context.Context, config *RadarrConfig) error {
    // Get ownership tag
    tagID := config.Status.ManagedResources.TagID
    if tagID == nil {
        return nil // Nothing to clean up
    }

    // Delete all resources with our tag
    return r.adapter.DeleteTaggedResources(ctx, *tagID)
}
```

---

## 6. Error Handling & Retry

### 6.1 Error Categories

| Category | Examples | Retry Strategy |
|----------|----------|----------------|
| **Transient** | Network timeout, 503 | Exponential backoff |
| **Rate limit** | 429 Too Many Requests | Respect Retry-After header |
| **Client error** | 400, 404 | No retry, log and update status |
| **Server error** | 500, 502 | Exponential backoff |
| **Configuration** | Invalid CRD values | No retry, update status |

### 6.2 Retry Configuration

```go
// internal/adapters/retry.go

package adapters

import (
    "net/http"
    "time"

    "github.com/cenkalti/backoff/v4"
)

// RetryConfig defines retry behavior
type RetryConfig struct {
    MaxRetries     int
    InitialBackoff time.Duration
    MaxBackoff     time.Duration
    Multiplier     float64
}

// DefaultRetryConfig returns sensible defaults
func DefaultRetryConfig() RetryConfig {
    return RetryConfig{
        MaxRetries:     5,
        InitialBackoff: 1 * time.Second,
        MaxBackoff:     30 * time.Second,
        Multiplier:     2.0,
    }
}

func shouldRetry(statusCode int) bool {
    switch statusCode {
    case http.StatusTooManyRequests,
         http.StatusServiceUnavailable,
         http.StatusBadGateway,
         http.StatusGatewayTimeout:
        return true
    default:
        return statusCode >= 500
    }
}
```

### 6.3 Controller-Runtime Requeue

```go
func (r *RadarrConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    config := &RadarrConfig{}
    if err := r.Get(ctx, req.NamespacedName, config); err != nil {
        return ctrl.Result{}, client.IgnoreNotFound(err)
    }

    // Check if suspended
    if config.Spec.Reconciliation != nil && config.Spec.Reconciliation.Suspend {
        return ctrl.Result{}, nil
    }

    // Attempt reconciliation
    if err := r.reconcile(ctx, config); err != nil {
        if isTransient(err) {
            // Requeue with backoff
            return ctrl.Result{RequeueAfter: calculateBackoff(config)}, nil
        }

        if isConfigurationError(err) {
            // Don't requeue, update status
            r.updateStatus(ctx, config, ConditionFailed, err.Error())
            return ctrl.Result{}, nil
        }

        // Unknown error, requeue with default backoff
        return ctrl.Result{RequeueAfter: 30 * time.Second}, err
    }

    // Success - requeue for periodic sync
    interval := 5 * time.Minute
    if config.Spec.Reconciliation != nil && config.Spec.Reconciliation.Interval != nil {
        interval = config.Spec.Reconciliation.Interval.Duration
    }
    return ctrl.Result{RequeueAfter: interval}, nil
}
```

### 6.4 Graceful Degradation

Continue reconciling what works when partial failures occur:

```go
func (r *Reconciler) reconcile(ctx context.Context, config *RadarrConfig) error {
    var errs []error

    // Try each resource type independently
    if err := r.reconcileQualityProfiles(ctx, config); err != nil {
        errs = append(errs, fmt.Errorf("quality profiles: %w", err))
    }

    if err := r.reconcileCustomFormats(ctx, config); err != nil {
        errs = append(errs, fmt.Errorf("custom formats: %w", err))
    }

    if err := r.reconcileDownloadClients(ctx, config); err != nil {
        errs = append(errs, fmt.Errorf("download clients: %w", err))
    }

    if err := r.reconcileIndexers(ctx, config); err != nil {
        errs = append(errs, fmt.Errorf("indexers: %w", err))
    }

    if err := r.reconcileNaming(ctx, config); err != nil {
        errs = append(errs, fmt.Errorf("naming: %w", err))
    }

    if len(errs) > 0 {
        r.updateStatus(ctx, config, ConditionDegraded, fmt.Sprintf("%d errors", len(errs)))
        return errors.Join(errs...)
    }

    r.updateStatus(ctx, config, ConditionReady, "")
    return nil
}
```

---

## 7. Prowlarr Integration

### 7.0 Dependency Handling

When a bundled config (e.g., `RadarrConfig`) references a `ProwlarrConfig` that hasn't been reconciled yet:

**Scenario:**
1. `RadarrConfig` references `ProwlarrConfig` for indexers
2. `RadarrConfig` has `autoRegister: true`
3. `ProwlarrConfig` hasn't been reconciled yet (not ready)

**Controller Behavior:**
1. Set condition `ProwlarrNotReady` with message "Waiting for ProwlarrConfig 'my-prowlarr' to become ready"
2. Requeue with exponential backoff (starting at 30s)
3. Continue reconciling other sections (quality, download clients, naming)
4. Mark overall status as `PartialSuccess` until Prowlarr is ready

```go
// In reconcileIndexers:
if config.Spec.Indexers.ProwlarrRef != nil {
    prowlarr := &ProwlarrConfig{}
    if err := r.Get(ctx, prowlarrKey, prowlarr); err != nil {
        if apierrors.IsNotFound(err) {
            meta.SetStatusCondition(&config.Status.Conditions, metav1.Condition{
                Type:    "ProwlarrReady",
                Status:  metav1.ConditionFalse,
                Reason:  "ProwlarrNotFound",
                Message: fmt.Sprintf("ProwlarrConfig '%s' not found", ref.Name),
            })
            return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
        }
        return ctrl.Result{}, err
    }
    
    if !isProwlarrReady(prowlarr) {
        meta.SetStatusCondition(&config.Status.Conditions, metav1.Condition{
            Type:    "ProwlarrReady",
            Status:  metav1.ConditionFalse,
            Reason:  "ProwlarrNotReady",
            Message: fmt.Sprintf("ProwlarrConfig '%s' is not ready", ref.Name),
        })
        return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
    }
    // Continue with registration...
}
```

### 7.1 Auto-Registration Flow

When a RadarrConfig has `prowlarrRef` with `autoRegister: true`:

```
1. RadarrConfig created/updated
2. Controller resolves ProwlarrConfig reference
3. Controller gets Radarr connection info (URL, API key)
4. Controller calls Prowlarr adapter to register app
5. Prowlarr creates Application entry
6. Indexers sync automatically to Radarr
```

### 7.2 Implementation

```go
// internal/controller/radarrconfig_controller.go

func (r *RadarrConfigReconciler) reconcileIndexers(ctx context.Context, config *RadarrConfig) error {
    if config.Spec.Indexers == nil {
        return nil
    }

    if config.Spec.Indexers.ProwlarrRef != nil {
        return r.reconcileProwlarrRef(ctx, config)
    }

    if len(config.Spec.Indexers.Direct) > 0 {
        return r.reconcileDirectIndexers(ctx, config)
    }

    return nil
}

func (r *RadarrConfigReconciler) reconcileProwlarrRef(ctx context.Context, config *RadarrConfig) error {
    ref := config.Spec.Indexers.ProwlarrRef

    // Get ProwlarrConfig
    prowlarr := &ProwlarrConfig{}
    if err := r.Get(ctx, client.ObjectKey{
        Namespace: config.Namespace,
        Name:      ref.Name,
    }, prowlarr); err != nil {
        return fmt.Errorf("failed to get ProwlarrConfig %s: %w", ref.Name, err)
    }

    // Auto-register if enabled
    if ref.AutoRegister == nil || *ref.AutoRegister {
        // Get Radarr connection info
        radarrURL := config.Spec.Connection.URL
        radarrAPIKey, _, err := r.secretResolver.ResolveAPIKey(ctx, config.Namespace, config.Spec.Connection)
        if err != nil {
            return fmt.Errorf("failed to resolve Radarr API key: %w", err)
        }

        // Get Prowlarr connection
        prowlarrConn, err := r.resolveProwlarrConnection(ctx, prowlarr)
        if err != nil {
            return err
        }

        // Register Radarr with Prowlarr
        prowlarrAdapter := r.adapters.Get("prowlarr")
        return prowlarrAdapter.RegisterApplication(ctx, prowlarrConn, &ApplicationRegistration{
            Name:     config.Name,
            Type:     "radarr",
            URL:      radarrURL,
            APIKey:   radarrAPIKey,
            Tags:     getIncludeTags(ref),
        })
    }

    return nil
}
```

---

## 8. Category Mapping

### 8.1 Human-Readable Categories

Users can specify categories as human-readable strings or numeric IDs:

```yaml
indexers:
  direct:
    - name: my-indexer
      categories:
        - "movies-hd"      # Human-readable
        - "movies-uhd"
        - "2040"           # Numeric ID (passes through)
```

### 8.2 Mapping Implementation

```go
// internal/discovery/categories.go

package discovery

// CategoryMapping maps human-readable names to Newznab category IDs
var CategoryMapping = map[string][]int{
    // Movies
    "movies":     {2000},
    "movies-sd":  {2030},
    "movies-hd":  {2040},
    "movies-uhd": {2045},
    "movies-4k":  {2045},
    "movies-3d":  {2060},

    // TV
    "tv":     {5000},
    "tv-sd":  {5030},
    "tv-hd":  {5040},
    "tv-uhd": {5045},
    "tv-4k":  {5045},

    // Audio
    "audio":         {3000},
    "audio-mp3":     {3010},
    "audio-lossless": {3040},
    "audio-flac":    {3040},

    // Anime (for Sonarr with anime support)
    "anime":    {5070},
    "anime-tv": {5070},
}

// ResolveCategories converts human-readable or numeric strings to IDs
func ResolveCategories(input []string) []int {
    var result []int
    seen := make(map[int]bool)

    for _, cat := range input {
        // Try numeric first
        if id, err := strconv.Atoi(cat); err == nil {
            if !seen[id] {
                result = append(result, id)
                seen[id] = true
            }
            continue
        }

        // Try human-readable mapping
        if ids, ok := CategoryMapping[strings.ToLower(cat)]; ok {
            for _, id := range ids {
                if !seen[id] {
                    result = append(result, id)
                    seen[id] = true
                }
            }
        }
    }

    return result
}
```

---

## 9. Testing Strategy

### 9.1 Test Pyramid

| Level | Scope | Tools | Coverage Target |
|-------|-------|-------|-----------------|
| **Unit** | Individual functions | Go testing | 80%+ |
| **Integration** | Adapter + mock API | httptest, envtest | Key paths |
| **E2E** | Full controller + real apps | Kind, testcontainers | Happy path |

### 9.2 Unit Tests

```go
// internal/compiler/merge_test.go

func TestDefaultsMerger_MergeRadarrConfig(t *testing.T) {
    tests := []struct {
        name      string
        cluster   *ClusterNebularrDefaults
        namespace *NebularrDefaults
        config    *RadarrConfig
        want      *RadarrConfig
    }{
        {
            name: "config overrides namespace overrides cluster",
            cluster: &ClusterNebularrDefaults{
                Spec: NebularrDefaultsSpec{
                    VideoQuality: &VideoQualitySpec{Preset: "any"},
                },
            },
            namespace: &NebularrDefaults{
                Spec: NebularrDefaultsSpec{
                    VideoQuality: &VideoQualitySpec{Preset: "balanced"},
                },
            },
            config: &RadarrConfig{
                Spec: RadarrConfigSpec{
                    Quality: &VideoQualitySpec{Preset: "4k-hdr"},
                },
            },
            want: &RadarrConfig{
                Spec: RadarrConfigSpec{
                    Quality: &VideoQualitySpec{Preset: "4k-hdr"}, // Config wins
                },
            },
        },
        {
            name: "namespace fills in when config omitted",
            cluster: nil,
            namespace: &NebularrDefaults{
                Spec: NebularrDefaultsSpec{
                    VideoQuality: &VideoQualitySpec{Preset: "balanced"},
                },
            },
            config: &RadarrConfig{
                Spec: RadarrConfigSpec{
                    Quality: nil, // Not specified
                },
            },
            want: &RadarrConfig{
                Spec: RadarrConfigSpec{
                    Quality: &VideoQualitySpec{Preset: "balanced"}, // Namespace fills in
                },
            },
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            m := &DefaultsMerger{}
            got := m.MergeRadarrConfig(tt.cluster, tt.namespace, tt.config)
            assert.Equal(t, tt.want.Spec.Quality, got.Spec.Quality)
        })
    }
}
```

### 9.3 Integration Tests

```go
// internal/controller/radarrconfig_controller_test.go

var _ = Describe("RadarrConfig Controller", func() {
    Context("With auto-discovery", func() {
        It("Should resolve API key from convention-based secret", func() {
            // Create secret with convention name
            secret := &corev1.Secret{
                ObjectMeta: metav1.ObjectMeta{
                    Name:      "radarr-api-key",
                    Namespace: "default",
                },
                Data: map[string][]byte{
                    "apiKey": []byte("test-api-key"),
                },
            }
            Expect(k8sClient.Create(ctx, secret)).Should(Succeed())

            // Create RadarrConfig without explicit secret ref
            config := &RadarrConfig{
                ObjectMeta: metav1.ObjectMeta{
                    Name:      "radarr",
                    Namespace: "default",
                },
                Spec: RadarrConfigSpec{
                    Connection: ConnectionSpec{
                        URL: "http://radarr:7878",
                        // No APIKeySecretRef - should auto-discover
                    },
                },
            }
            Expect(k8sClient.Create(ctx, config)).Should(Succeed())

            // Verify reconciliation succeeds
            Eventually(func() bool {
                err := k8sClient.Get(ctx, client.ObjectKeyFromObject(config), config)
                if err != nil {
                    return false
                }
                return config.Status.Connected
            }, timeout, interval).Should(BeTrue())
        })
    })
})
```

---

## 10. Observability

### 10.1 Metrics

```go
// internal/telemetry/metrics.go

package telemetry

import (
    "github.com/prometheus/client_golang/prometheus"
    "sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
    ReconcileTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "nebularr_reconcile_total",
            Help: "Total number of reconciliations",
        },
        []string{"app", "config", "result"},
    )

    ReconcileDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "nebularr_reconcile_duration_seconds",
            Help:    "Duration of reconciliation",
            Buckets: prometheus.DefBuckets,
        },
        []string{"app", "config"},
    )

    APICallTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "nebularr_api_call_total",
            Help: "Total API calls to *arr applications",
        },
        []string{"app", "operation", "status"},
    )

    ManagedResources = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "nebularr_managed_resources",
            Help: "Number of managed resources",
        },
        []string{"app", "config", "resource_type"},
    )

    SecretResolution = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "nebularr_secret_resolution_total",
            Help: "Secret resolution attempts",
        },
        []string{"app", "source", "result"},
    )
)

func init() {
    metrics.Registry.MustRegister(
        ReconcileTotal,
        ReconcileDuration,
        APICallTotal,
        ManagedResources,
        SecretResolution,
    )
}
```

---

## 11. Related Documents

- [README](./README.md) - Build order, file mapping
- [TYPES](./TYPES.md) - IR types and interfaces
- [CRDS](./CRDS.md) - CRD schemas and validation
- [PRESETS](./PRESETS.md) - Quality and naming presets
- [RADARR](./RADARR.md) - Radarr adapter mappings
- [SONARR](./SONARR.md) - Sonarr adapter mappings
- [LIDARR](./LIDARR.md) - Lidarr adapter mappings
- [PROWLARR](./PROWLARR.md) - Prowlarr adapter mappings
