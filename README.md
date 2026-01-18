# Nebularr Operator

A Kubernetes operator for declarative configuration management of the *arr media stack (Radarr, Sonarr, Lidarr, Prowlarr, Bazarr) and download clients (Transmission, qBittorrent, SABnzbd).

## Overview

Nebularr enables GitOps-style management of your media automation stack. Instead of manually configuring each application through their web UIs, you define your desired state as Kubernetes Custom Resources, and Nebularr ensures your applications match that configuration.

### Key Features

- **Declarative Configuration**: Define quality profiles, download clients, indexers, and more as YAML
- **GitOps Ready**: Store your media stack configuration in Git and let ArgoCD/Flux sync it
- **Secret Management**: Integrates with Kubernetes Secrets for API keys and credentials
- **Prowlarr Integration**: Automatic indexer synchronization across all *arr applications
- **Download Client Management**: Configure Transmission, qBittorrent, and SABnzbd
- **Quality Presets**: Built-in presets for common quality configurations (balanced, 4k-optimized, etc.)

### Supported Applications

| Application | CRD | Description |
|-------------|-----|-------------|
| Radarr | `RadarrConfig` | Movie collection management |
| Sonarr | `SonarrConfig` | TV series management |
| Lidarr | `LidarrConfig` | Music collection management |
| Prowlarr | `ProwlarrConfig` | Indexer management and sync |
| Bazarr | `BazarrConfig` | Subtitle management |
| Transmission | `DownloadStackConfig` | Torrent download client |

## Architecture

```
                    +------------------+
                    |  Kubernetes API  |
                    +--------+---------+
                             |
              +----CRDs------+------Secrets-----+
              |              |                  |
    +---------v----+ +-------v------+ +--------v-------+
    | RadarrConfig | | SonarrConfig | | ProwlarrConfig |
    +---------+----+ +-------+------+ +--------+-------+
              |              |                  |
              +------+-------+--------+---------+
                     |                |
            +--------v--------+  +----v----+
            | Nebularr        |  |  *arr   |
            | Controller      |--| APIs    |
            +-----------------+  +---------+
```

The operator uses an **Intermediate Representation (IR)** layer to abstract *arr API specifics:
1. Controllers compile CRD specs to IR
2. Adapters translate IR to application-specific API calls
3. This allows for consistent configuration across different *arr applications

## Getting Started

### Prerequisites

- Kubernetes v1.26+
- kubectl v1.26+
- Helm v3 (optional, for Helm-based installation)
- Docker (for building from source)
- *arr applications deployed and accessible within the cluster

### Installation

#### Using kubectl

```bash
# Install CRDs
kubectl apply -f https://raw.githubusercontent.com/poiley/nebularr/main/dist/install.yaml
```

#### From Source

```bash
# Clone the repository
git clone https://github.com/poiley/nebularr.git
cd nebularr

# Install CRDs
make install

# Deploy the operator
make deploy IMG=ghcr.io/poiley/nebularr:latest
```

### Quick Start

1. **Create a Secret with your Radarr API key:**

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: radarr-credentials
  namespace: media
type: Opaque
stringData:
  apiKey: "your-radarr-api-key-here"
```

2. **Create a RadarrConfig resource:**

```yaml
apiVersion: arr.rinzler.cloud/v1alpha1
kind: RadarrConfig
metadata:
  name: radarr-production
  namespace: media
spec:
  connection:
    url: http://radarr.media.svc.cluster.local:7878
    apiKeySecretRef:
      name: radarr-credentials
      key: apiKey

  quality:
    preset: balanced  # Use built-in preset

  rootFolders:
    - /movies

  downloadClients:
    - name: transmission
      type: transmission
      url: http://transmission.media.svc.cluster.local:9091
      credentialsSecretRef:
        name: transmission-credentials
        usernameKey: username
        passwordKey: password
      category: movies

  naming:
    preset: plex-friendly
```

3. **Apply the configuration:**

```bash
kubectl apply -f radarr-config.yaml
```

4. **Check the status:**

```bash
kubectl get radarrconfig radarr-production -n media -o yaml
```

## Configuration Reference

### RadarrConfig / SonarrConfig

```yaml
spec:
  connection:
    url: string                    # Required: URL of the *arr instance
    apiKeySecretRef:               # Required: Reference to API key secret
      name: string
      key: string

  quality:
    preset: string                 # One of: balanced, 4k-optimized, storage-optimized
    # OR custom tiers:
    tiers:
      - resolution: string         # 2160p, 1080p, 720p, etc.
        sources: [string]          # bluray, webdl, webrip, hdtv, etc.
        allowed: bool

  rootFolders: [string]            # List of root folder paths

  downloadClients:
    - name: string
      type: string                 # transmission, qbittorrent, sabnzbd
      url: string
      credentialsSecretRef:        # For clients requiring user/pass
        name: string
        usernameKey: string
        passwordKey: string
      apiKeySecretRef:             # For clients requiring API key
        name: string
        key: string
      category: string
      priority: int

  indexers:
    prowlarrRef:                   # Sync indexers from Prowlarr
      name: string
      autoRegister: bool

  naming:
    preset: string                 # plex-friendly, jellyfin-friendly, custom
    # OR custom formats (see samples for details)

  customFormats: [...]             # Custom format definitions
  delayProfiles: [...]             # Delay profile configurations
  importLists: [...]               # Import list configurations
  notifications: [...]             # Notification configurations

  reconciliation:
    interval: duration             # How often to reconcile (default: 5m)
    suspend: bool                  # Pause reconciliation
```

### ProwlarrConfig

```yaml
spec:
  connection:
    url: string
    apiKeySecretRef:
      name: string
      key: string

  indexers:
    - name: string
      type: string                 # Indexer type (see Prowlarr docs)
      enabled: bool
      settings:                    # Indexer-specific settings
        key: value
      secretRef:                   # For sensitive settings
        name: string

  syncTargets:                     # Automatically sync to *arr apps
    - type: radarr
      configRef:
        name: string
    - type: sonarr
      configRef:
        name: string
```

### DownloadStackConfig

```yaml
spec:
  transmission:
    enabled: bool
    url: string
    credentialsSecretRef:
      name: string
      usernameKey: string
      passwordKey: string
    settings:
      downloadDir: string
      incompleteDir: string
      peerPort: int

  gluetun:                         # VPN configuration
    enabled: bool
    provider: string               # mullvad, nordvpn, etc.
    secretRef:
      name: string
```

## Examples

See the [config/samples/](config/samples/) directory for complete examples:

- [RadarrConfig](config/samples/arr_v1alpha1_radarrconfig.yaml) - Movie management with custom formats
- [SonarrConfig](config/samples/arr_v1alpha1_sonarrconfig.yaml) - TV show management with anime support
- [LidarrConfig](config/samples/arr_v1alpha1_lidarrconfig.yaml) - Music management with metadata profiles
- [ProwlarrConfig](config/samples/arr_v1alpha1_prowlarrconfig.yaml) - Indexer management

## Development

### Prerequisites

- Go 1.24+
- Docker
- Kind (for local testing)
- kubebuilder

### Building

```bash
# Build the operator binary
make build

# Build the Docker image
make docker-build IMG=nebularr:dev

# Run tests
make test

# Run integration tests (requires Docker)
make test-integration

# Run full E2E tests (requires Kind)
make test-e2e
```

### Project Structure

```
.
├── api/v1alpha1/          # CRD type definitions
├── cmd/                   # Main entry point
├── config/
│   ├── crd/              # Generated CRD manifests
│   ├── manager/          # Operator deployment manifests
│   └── samples/          # Example CRs
├── internal/
│   ├── adapters/         # *arr API adapters
│   │   ├── radarr/
│   │   ├── sonarr/
│   │   ├── lidarr/
│   │   ├── prowlarr/
│   │   └── downloadstack/
│   ├── controller/       # Reconciliation controllers
│   ├── discovery/        # API key discovery utilities
│   └── ir/v1/           # Intermediate Representation types
└── test/
    ├── e2e/             # End-to-end tests
    └── utils/           # Test utilities
```

### Adding a New *arr Application

1. Define types in `api/v1alpha1/<app>config_types.go`
2. Create IR mappings in `internal/ir/v1/`
3. Implement adapter in `internal/adapters/<app>/`
4. Create controller in `internal/controller/<app>config_controller.go`
5. Register with the adapter registry
6. Add sample configuration

## Troubleshooting

### Common Issues

**CRD shows "NotReady" status:**
```bash
# Check the operator logs
kubectl logs -n nebularr-system deployment/nebularr-controller-manager

# Check events on the CR
kubectl describe radarrconfig <name> -n <namespace>
```

**Cannot connect to *arr instance:**
- Verify the URL is accessible from within the cluster
- Check that the API key secret exists and contains the correct key
- Ensure network policies allow traffic from the operator

**Changes not being applied:**
- Check if reconciliation is suspended (`spec.reconciliation.suspend: true`)
- Verify the operator has RBAC permissions to read secrets
- Check for validation errors in the CR status

## Contributing

Contributions are welcome! Please read our contributing guidelines and submit pull requests to the main repository.

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run tests: `make test`
5. Submit a pull request

## License

Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
