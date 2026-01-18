# Nebularr - Bazarr Configuration Reference

> **For coding agents:** Start with [README.md](./README.md) for build order. This document contains Bazarr adapter code to copy.
>
> **Related:** [README](./README.md) | [TYPES](./TYPES.md) | [CRDS](./CRDS.md)

This document is a reference for implementing the Bazarr adapter. Bazarr manages subtitle downloads and integrates with Sonarr and Radarr for automated subtitle matching.

---

## 1. Overview

Bazarr is a companion application to Sonarr and Radarr that:
- **Automatically downloads subtitles** for movies and TV shows
- **Integrates with Sonarr/Radarr** for media library awareness
- **Supports multiple providers** (OpenSubtitles, Subscene, etc.)
- **Manages language profiles** for multi-language libraries

---

## 2. Architecture

### 2.1 Bazarr Configuration Approach

Unlike other *arr apps which use a REST API for all configuration, Bazarr uses a hybrid approach:

1. **File-based config**: Core settings via `config.yaml`
2. **API-based config**: Runtime settings via `/api/` endpoints

The BazarrConfig controller supports both approaches:
- **ConfigMap generation**: Generates `config.yaml` for init-container mounting
- **API client**: Configures Bazarr at runtime after startup

### 2.2 Integration Flow

```
                    +-----------+
                    |  Sonarr   |
                    +-----+-----+
                          |
                          v
+-------------+     +-----------+     +------------+
| BazarrConfig|---->|  Bazarr   |---->| Providers  |
+-------------+     +-----------+     +------------+
                          |
                          v
                    +-----------+
                    |  Radarr   |
                    +-----+-----+
```

---

## 3. Connection Configuration

### 3.1 Sonarr/Radarr Connections

BazarrConfig requires connections to both Sonarr and Radarr:

```yaml
spec:
  sonarr:
    url: http://sonarr:8989
    apiKeySecretRef:
      name: sonarr-credentials
      key: apiKey
    # Or auto-discover from config.xml:
    # configPath: /sonarr-config/config.xml
  
  radarr:
    url: http://radarr:7878
    apiKeySecretRef:
      name: radarr-credentials
      key: apiKey
```

### 3.2 API Key Auto-Discovery

Bazarr can auto-discover API keys from Sonarr/Radarr config files:

```go
// internal/adapters/bazarr/discovery.go

package bazarr

import (
    "encoding/xml"
    "os"
)

// ArrConfig represents the structure of config.xml
type ArrConfig struct {
    XMLName xml.Name `xml:"Config"`
    ApiKey  string   `xml:"ApiKey"`
}

// DiscoverAPIKey reads the API key from a config.xml file
func DiscoverAPIKey(configPath string) (string, error) {
    data, err := os.ReadFile(configPath)
    if err != nil {
        return "", err
    }
    
    var config ArrConfig
    if err := xml.Unmarshal(data, &config); err != nil {
        return "", err
    }
    
    return config.ApiKey, nil
}
```

---

## 4. Language Profiles

### 4.1 Language Profile Structure

```yaml
languageProfiles:
  - name: english-primary
    languages:
      - code: en
        forced: false
        hearingImpaired: false
    defaultForSeries: true
    defaultForMovies: true

  - name: multilingual
    languages:
      - code: en
        forced: false
        hearingImpaired: false
      - code: es
        forced: false
        hearingImpaired: false
      - code: fr
        forced: false
        hearingImpaired: false
```

### 4.2 Language Codes

ISO 639-1 language codes supported:

| Code | Language |
|------|----------|
| `en` | English |
| `es` | Spanish |
| `fr` | French |
| `de` | German |
| `it` | Italian |
| `pt` | Portuguese |
| `ru` | Russian |
| `ja` | Japanese |
| `ko` | Korean |
| `zh` | Chinese |

### 4.3 API Endpoints

```
GET /api/languages
GET /api/languages/profiles
POST /api/languages/profiles
PUT /api/languages/profiles/{id}
DELETE /api/languages/profiles/{id}
```

---

## 5. Subtitle Providers

### 5.1 Supported Providers

| Provider | Auth Type | API Key | Username/Password |
|----------|-----------|---------|-------------------|
| `opensubtitles` | Password | No | Yes |
| `opensubtitlescom` | Password | No | Yes |
| `subscene` | None | No | No |
| `podnapisi` | None | No | No |
| `addic7ed` | Password | No | Yes |
| `legendasdivx` | Password | No | Yes |
| `napisy24` | Password | No | Yes |
| `titlovi` | Password | No | Yes |
| `subz` | None | No | No |
| `supersubtitles` | None | No | No |

### 5.2 Provider Configuration

```yaml
providers:
  - name: opensubtitles
    username: your-username
    passwordSecretRef:
      name: bazarr-providers
      key: opensubtitles-password

  - name: opensubtitlescom
    username: your-username
    passwordSecretRef:
      name: bazarr-providers
      key: opensubtitlescom-password

  - name: subscene
    # No authentication required

  - name: addic7ed
    username: your-username
    passwordSecretRef:
      name: bazarr-providers
      key: addic7ed-password
```

### 5.3 Provider API Endpoints

```
GET /api/providers
GET /api/providers/movies
GET /api/providers/episodes
POST /api/providers
PUT /api/providers
```

---

## 6. Bazarr API Reference

### 6.1 System Endpoints

```
GET /api/system/status          # System status
GET /api/system/health          # Health check
GET /api/system/settings        # Get settings
POST /api/system/settings       # Update settings
```

### 6.2 Series/Movies Endpoints

```
GET /api/series                 # List series (from Sonarr)
GET /api/movies                 # List movies (from Radarr)
GET /api/episodes               # List episodes
```

### 6.3 Subtitles Endpoints

```
GET /api/subtitles              # List subtitles
POST /api/subtitles             # Search subtitles
DELETE /api/subtitles/{id}      # Delete subtitle
POST /api/subtitles/download    # Download subtitle
```

### 6.4 Settings Structure

```go
// internal/adapters/bazarr/types.go

package bazarr

// SystemSettings represents Bazarr system settings
type SystemSettings struct {
    General     GeneralSettings     `json:"general"`
    Sonarr      SonarrSettings      `json:"sonarr"`
    Radarr      RadarrSettings      `json:"radarr"`
    Providers   ProviderSettings    `json:"providers"`
    Subtitles   SubtitleSettings    `json:"subtitles"`
    Languages   LanguageSettings    `json:"languages"`
    Auth        AuthSettings        `json:"auth"`
}

type GeneralSettings struct {
    IP          string `json:"ip"`
    Port        int    `json:"port"`
    BaseURL     string `json:"base_url"`
    Debug       bool   `json:"debug"`
    BranchName  string `json:"branch"`
}

type SonarrSettings struct {
    IP          string `json:"ip"`
    Port        int    `json:"port"`
    BaseURL     string `json:"base_url"`
    APIKey      string `json:"apikey"`
    SSL         bool   `json:"ssl"`
}

type RadarrSettings struct {
    IP          string `json:"ip"`
    Port        int    `json:"port"`
    BaseURL     string `json:"base_url"`
    APIKey      string `json:"apikey"`
    SSL         bool   `json:"ssl"`
}
```

---

## 7. Config.yaml Generation

### 7.1 Config Structure

Bazarr's `config.yaml` is generated from the CRD spec:

```yaml
# Generated config.yaml
general:
  ip: 0.0.0.0
  port: 6767
  base_url: /
  debug: false

sonarr:
  ip: sonarr.media.svc.cluster.local
  port: 8989
  base_url: /
  apikey: <from-secret>
  ssl: false

radarr:
  ip: radarr.media.svc.cluster.local
  port: 7878
  base_url: /
  apikey: <from-secret>
  ssl: false

auth:
  type: basic
  username: admin
  password: <from-secret>
```

### 7.2 Generator Implementation

```go
// internal/adapters/bazarr/generator.go

package bazarr

import (
    "fmt"
    "net/url"
    
    arrv1alpha1 "github.com/your-org/nebularr/api/v1alpha1"
    "gopkg.in/yaml.v3"
)

// GenerateConfig creates Bazarr config.yaml content
func GenerateConfig(spec *arrv1alpha1.BazarrConfigSpec, secrets map[string]string) (string, error) {
    // Parse Sonarr URL
    sonarrURL, err := url.Parse(spec.Sonarr.URL)
    if err != nil {
        return "", fmt.Errorf("invalid sonarr URL: %w", err)
    }
    
    // Parse Radarr URL
    radarrURL, err := url.Parse(spec.Radarr.URL)
    if err != nil {
        return "", fmt.Errorf("invalid radarr URL: %w", err)
    }
    
    config := map[string]interface{}{
        "general": map[string]interface{}{
            "ip":       "0.0.0.0",
            "port":     6767,
            "base_url": "/",
            "debug":    false,
        },
        "sonarr": map[string]interface{}{
            "ip":       sonarrURL.Hostname(),
            "port":     sonarrURL.Port(),
            "base_url": sonarrURL.Path,
            "apikey":   secrets["sonarr-apikey"],
            "ssl":      sonarrURL.Scheme == "https",
        },
        "radarr": map[string]interface{}{
            "ip":       radarrURL.Hostname(),
            "port":     radarrURL.Port(),
            "base_url": radarrURL.Path,
            "apikey":   secrets["radarr-apikey"],
            "ssl":      radarrURL.Scheme == "https",
        },
    }
    
    // Add authentication if configured
    if spec.Authentication != nil {
        config["auth"] = map[string]interface{}{
            "type":     spec.Authentication.Method,
            "username": spec.Authentication.Username,
            "password": secrets["bazarr-password"],
        }
    }
    
    data, err := yaml.Marshal(config)
    if err != nil {
        return "", fmt.Errorf("failed to marshal config: %w", err)
    }
    
    return string(data), nil
}
```

---

## 8. CRD Example

```yaml
apiVersion: arr.rinzler.cloud/v1alpha1
kind: BazarrConfig
metadata:
  name: bazarr-main
spec:
  # Sonarr connection
  sonarr:
    url: http://sonarr.media.svc.cluster.local:8989
    apiKeySecretRef:
      name: sonarr-credentials
      key: apiKey

  # Radarr connection
  radarr:
    url: http://radarr.media.svc.cluster.local:7878
    apiKeySecretRef:
      name: radarr-credentials
      key: apiKey

  # Language profiles
  languageProfiles:
    - name: english-default
      languages:
        - code: en
          forced: false
          hearingImpaired: false
      defaultForSeries: true
      defaultForMovies: true

    - name: hearing-impaired
      languages:
        - code: en
          forced: false
          hearingImpaired: true

  # Subtitle providers
  providers:
    - name: opensubtitles
      username: myuser
      passwordSecretRef:
        name: bazarr-providers
        key: opensubtitles-password

    - name: subscene

    - name: podnapisi

  # Authentication
  authentication:
    method: basic
    username: admin
    passwordSecretRef:
      name: bazarr-credentials
      key: password

  # Output to ConfigMap
  configMapRef:
    name: bazarr-config

  reconciliation:
    interval: 5m
    suspend: false
```

---

## 9. Deployment Example

### 9.1 Using ConfigMap for Config

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: bazarr
spec:
  template:
    spec:
      initContainers:
        - name: config-init
          image: busybox
          command: ['sh', '-c', 'cp /config-source/config.yaml /config/config/config.yaml']
          volumeMounts:
            - name: config-source
              mountPath: /config-source
            - name: config
              mountPath: /config/config
      containers:
        - name: bazarr
          image: lscr.io/linuxserver/bazarr:latest
          volumeMounts:
            - name: config
              mountPath: /config
      volumes:
        - name: config-source
          configMap:
            name: bazarr-config
        - name: config
          emptyDir: {}
```

---

## 10. Troubleshooting

### 10.1 Connection Issues

If Bazarr can't connect to Sonarr/Radarr:
1. Check URL format (must include `http://` or `https://`)
2. Verify API key is correct
3. Ensure network connectivity between pods

### 10.2 Provider Issues

Common provider problems:
- **OpenSubtitles**: Rate limited, use credentials
- **Subscene**: May require CAPTCHA solving
- **Addic7ed**: Strict rate limits, consider alternatives

### 10.3 Status Fields

The BazarrConfigStatus provides connection info:

| Field | Description |
|-------|-------------|
| `sonarrConnected` | Sonarr reachable |
| `radarrConnected` | Radarr reachable |
| `configGenerated` | Config.yaml created |
| `lastReconcile` | Last sync timestamp |
