# Nebularr - Implementation Guide for Coding Agents

> **What is this?** A Kubernetes controller that manages *arr ecosystem (Radarr, Sonarr, Lidarr, Prowlarr) configuration through declarative CRDs.

---

## Quick Reference

| Property | Value |
|----------|-------|
| Go Module | `github.com/poiley/nebularr` |
| API Group | `arr.rinzler.cloud` |
| API Version | `v1alpha1` |
| Language | Go 1.22+ |
| Framework | Kubebuilder go/v4, controller-runtime |
| Target K8s | 1.35+ |

---

## CRD Hierarchy

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

---

## Build Order (Execute Sequentially)

### Phase 1: Scaffold (Do First)

```bash
# 1. Create project
mkdir -p nebularr && cd nebularr

# 2. Initialize Kubebuilder
kubebuilder init --domain rinzler.cloud --repo github.com/poiley/nebularr --plugins=go/v4

# 3. Create Bundled Config CRDs (Simple Path)
kubebuilder create api --group arr --version v1alpha1 --kind RadarrConfig --resource --controller
kubebuilder create api --group arr --version v1alpha1 --kind SonarrConfig --resource --controller
kubebuilder create api --group arr --version v1alpha1 --kind LidarrConfig --resource --controller
kubebuilder create api --group arr --version v1alpha1 --kind ProwlarrConfig --resource --controller

# 4. Create Shared Resources
kubebuilder create api --group arr --version v1alpha1 --kind QualityTemplate --resource
kubebuilder create api --group arr --version v1alpha1 --kind NebularrDefaults --resource
kubebuilder create api --group arr --version v1alpha1 --kind ClusterNebularrDefaults --resource --namespaced=false

# 5. Create Special CRDs
kubebuilder create api --group arr --version v1alpha1 --kind BazarrConfig --resource --controller

# 6. Create internal packages
mkdir -p internal/compiler internal/ir/v1 internal/presets
mkdir -p internal/adapters/radarr internal/adapters/sonarr internal/adapters/lidarr internal/adapters/prowlarr
mkdir -p internal/secrets internal/discovery internal/telemetry

# 7. Generate manifests
make manifests generate
```

### Phase 2: Implement Common Types

| Step | File to Create | Copy Types From |
|------|----------------|-----------------|
| 1 | `api/v1alpha1/common_types.go` | [CRDS.md Section 2](./CRDS.md#2-common-types) |
| 2 | `api/v1alpha1/radarrconfig_types.go` | [CRDS.md Section 3.1](./CRDS.md#31-radarrconfig) |
| 3 | `api/v1alpha1/sonarrconfig_types.go` | [CRDS.md Section 3.2](./CRDS.md#32-sonarrconfig) |
| 4 | `api/v1alpha1/lidarrconfig_types.go` | [CRDS.md Section 3.3](./CRDS.md#33-lidarrconfig) |
| 5 | `api/v1alpha1/prowlarrconfig_types.go` | [CRDS.md Section 3.4](./CRDS.md#34-prowlarrconfig) |
| 6 | `api/v1alpha1/qualitytemplate_types.go` | [CRDS.md Section 5.1](./CRDS.md#51-qualitytemplate) |
| 7 | `api/v1alpha1/nebularrdefaults_types.go` | [CRDS.md Section 5.2](./CRDS.md#52-nebularrdefaults) |

### Phase 3: Implement Presets

| Step | File to Create | Reference |
|------|----------------|-----------|
| 1 | `internal/presets/video.go` | [PRESETS.md Section 1](./PRESETS.md#1-video-quality-presets) |
| 2 | `internal/presets/audio.go` | [PRESETS.md Section 2](./PRESETS.md#2-audio-quality-presets) |
| 3 | `internal/presets/naming.go` | [PRESETS.md Section 3](./PRESETS.md#3-naming-presets) |
| 4 | `internal/presets/override.go` | [PRESETS.md Section 5](./PRESETS.md#5-override-syntax) |

### Phase 4: Implement IR Types

| Step | File to Create | Reference |
|------|----------------|-----------|
| 1 | `internal/ir/v1/ir.go` | [TYPES.md Section 5.1](./TYPES.md#51-ir-envelope) |
| 2 | `internal/ir/v1/connection.go` | [TYPES.md Section 5.2](./TYPES.md#52-connection-ir) |
| 3 | `internal/ir/v1/quality.go` | [TYPES.md Section 5.3](./TYPES.md#53-quality-ir) |
| 4 | `internal/ir/v1/download_client.go` | [TYPES.md Section 5.4](./TYPES.md#54-download-client-ir) |
| 5 | `internal/ir/v1/indexer.go` | [TYPES.md Section 5.5](./TYPES.md#55-indexer-ir) |
| 6 | `internal/ir/v1/naming.go` | [TYPES.md Section 5.6](./TYPES.md#56-naming-ir) |

### Phase 5: Implement Core Logic

| Step | File to Create | Reference |
|------|----------------|-----------|
| 1 | `internal/adapters/adapter.go` | [TYPES.md Section 6.1](./TYPES.md#61-interface-definition) |
| 2 | `internal/adapters/registry.go` | [TYPES.md Section 6.2](./TYPES.md#62-adapter-registration) |
| 3 | `internal/compiler/compiler.go` | [TYPES.md Section 7.1](./TYPES.md#71-compiler-definition) |
| 4 | `internal/compiler/merge.go` | [OPERATIONS.md Section 3](./OPERATIONS.md#3-defaults-merge-rules) |
| 5 | `internal/secrets/resolver.go` | [OPERATIONS.md Section 1.2](./OPERATIONS.md#12-auto-discovery-flow) |
| 6 | `internal/discovery/apikey.go` | [OPERATIONS.md Section 1.3](./OPERATIONS.md#13-configxml-parser) |
| 7 | `internal/discovery/download_client.go` | [OPERATIONS.md Section 2](./OPERATIONS.md#2-download-client-type-inference) |
| 8 | `internal/discovery/categories.go` | [OPERATIONS.md Section 8](./OPERATIONS.md#8-category-mapping) |

### Phase 6: Implement Radarr Adapter

| Step | File to Create | Reference |
|------|----------------|-----------|
| 1 | `internal/adapters/radarr/mapping.go` | [RADARR.md Section 1.2](./RADARR.md#12-go-implementation) |
| 2 | `internal/adapters/radarr/formats.go` | [RADARR.md Section 2.2](./RADARR.md#22-go-implementation) |
| 3 | `internal/adapters/radarr/clients.go` | [RADARR.md Section 3.2](./RADARR.md#32-go-implementation) |
| 4 | `internal/adapters/radarr/indexers.go` | [RADARR.md Section 4.4](./RADARR.md#44-go-implementation-indexer-mapping--payload) |
| 5 | `internal/adapters/radarr/naming.go` | [RADARR.md Section 5](./RADARR.md#5-naming-configuration) |
| 6 | `internal/adapters/radarr/tags.go` | [RADARR.md Section 5.1](./RADARR.md#51-tag-creation) |
| 7 | `internal/adapters/radarr/adapter.go` | [RADARR.md Section 6.2](./RADARR.md#62-go-implementation) |

### Phase 7: Generate Radarr API Client

```bash
# Install oapi-codegen
go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest

# Download Radarr OpenAPI spec
curl -o radarr-openapi.json https://raw.githubusercontent.com/Radarr/Radarr/develop/src/Radarr.Api.V3/openapi.json

# Generate client
mkdir -p internal/adapters/radarr/client
oapi-codegen -package client -generate types,client radarr-openapi.json > internal/adapters/radarr/client/client.go
```

### Phase 8: Wire RadarrConfig Controller

Modify `internal/controller/radarrconfig_controller.go` to implement reconciliation loop.
See [TYPES.md Section 4](./TYPES.md#phase-4-reconciliation-week-7) for the reconciliation flow.

### Phase 9: Implement Sonarr Adapter

| Step | File to Create | Reference |
|------|----------------|-----------|
| 1 | `internal/adapters/sonarr/mapping.go` | [SONARR.md Section 1.3](./SONARR.md#13-go-implementation) |
| 2 | `internal/adapters/sonarr/formats.go` | [SONARR.md Section 2.3](./SONARR.md#23-go-implementation) |
| 3 | `internal/adapters/sonarr/clients.go` | [SONARR.md Section 3.3](./SONARR.md#33-go-implementation) |
| 4 | `internal/adapters/sonarr/indexers.go` | [SONARR.md Section 4.4](./SONARR.md#44-go-implementation-indexer-payload) |
| 5 | `internal/adapters/sonarr/naming.go` | [SONARR.md Section 8.4](./SONARR.md#84-go-implementation) |
| 6 | `internal/adapters/sonarr/adapter.go` | [SONARR.md Section 9.2](./SONARR.md#92-go-implementation) |

### Phase 10: Generate Sonarr API Client

```bash
curl -o sonarr-openapi.json https://raw.githubusercontent.com/Sonarr/Sonarr/develop/src/Sonarr.Api.V3/openapi.json
mkdir -p internal/adapters/sonarr/client
oapi-codegen -package client -generate types,client sonarr-openapi.json > internal/adapters/sonarr/client/client.go
```

### Phase 11: Implement Lidarr Adapter (API v1)

| Step | File to Create | Reference |
|------|----------------|-----------|
| 1 | `internal/adapters/lidarr/mapping.go` | [LIDARR.md Section 1.3](./LIDARR.md#13-go-implementation) |
| 2 | `internal/adapters/lidarr/metadata.go` | [LIDARR.md Section 2.4](./LIDARR.md#24-go-implementation) |
| 3 | `internal/adapters/lidarr/releases.go` | [LIDARR.md Section 3.3](./LIDARR.md#33-go-implementation) |
| 4 | `internal/adapters/lidarr/clients.go` | [LIDARR.md Section 4.3](./LIDARR.md#43-go-implementation) |
| 5 | `internal/adapters/lidarr/naming.go` | [LIDARR.md Section 8.2](./LIDARR.md#82-go-implementation) |
| 6 | `internal/adapters/lidarr/adapter.go` | [LIDARR.md Section 9.2](./LIDARR.md#92-go-implementation) |

### Phase 12: Implement Prowlarr Adapter (API v1)

| Step | File to Create | Reference |
|------|----------------|-----------|
| 1 | `internal/adapters/prowlarr/indexers.go` | [PROWLARR.md Section 1.3](./PROWLARR.md#13-go-implementation) |
| 2 | `internal/adapters/prowlarr/proxies.go` | [PROWLARR.md Section 2.4](./PROWLARR.md#24-go-implementation) |
| 3 | `internal/adapters/prowlarr/applications.go` | [PROWLARR.md Section 3.5](./PROWLARR.md#35-go-implementation) |
| 4 | `internal/adapters/prowlarr/adapter.go` | [PROWLARR.md Section 5.3](./PROWLARR.md#53-go-implementation) |

### Phase 13: Implement Granular Policies (Optional - Power Users)

```bash
# Create granular policy CRDs
kubebuilder create api --group arr --version v1alpha1 --kind RadarrMediaPolicy --resource --controller
kubebuilder create api --group arr --version v1alpha1 --kind SonarrMediaPolicy --resource --controller
kubebuilder create api --group arr --version v1alpha1 --kind LidarrMusicPolicy --resource --controller
kubebuilder create api --group arr --version v1alpha1 --kind RadarrDownloadClientPolicy --resource --controller
kubebuilder create api --group arr --version v1alpha1 --kind RadarrIndexerPolicy --resource --controller
# ... similar for Sonarr/Lidarr
```

> **Controller Behavior:** Granular policy controllers watch their respective CRDs and trigger reconciliation on the parent bundled config. When a `RadarrMediaPolicy` changes, the controller finds the referenced `RadarrConfig` and enqueues it for reconciliation. The bundled config controller then merges the policy overlay during its reconcile loop.

| Step | File to Create | Reference |
|------|----------------|-----------|
| 1 | `api/v1alpha1/radarrmediapolicy_types.go` | [CRDS.md Section 4.1](./CRDS.md#41-radarrmediapolicy) |
| 2 | `api/v1alpha1/sonarrmediapolicy_types.go` | [CRDS.md Section 4.2](./CRDS.md#42-sonarrmediapolicy) |
| 3 | `api/v1alpha1/lidarrmusicpolicy_types.go` | [CRDS.md Section 4.3](./CRDS.md#43-lidarrmusicpolicy) |

### Phase 14: Implement BazarrConfig Generator

| Step | File to Create | Reference |
|------|----------------|-----------|
| 1 | `api/v1alpha1/bazarrconfig_types.go` | [CRDS.md Section 6](./CRDS.md#6-bazarrconfig) |
| 2 | `internal/adapters/bazarr/generator.go` | Generate config.yaml as ConfigMap |

### Phase 15: Operational Infrastructure

| Step | File to Create | Reference |
|------|----------------|-----------|
| 1 | `internal/telemetry/metrics.go` | [OPERATIONS.md Section 10](./OPERATIONS.md#10-observability) |
| 2 | `internal/webhook/radarrconfig_webhook.go` | [OPERATIONS.md Section 5.1](./OPERATIONS.md#51-url-uniqueness) |
| 3 | `internal/adapters/retry.go` | [OPERATIONS.md Section 6.2](./OPERATIONS.md#62-retry-configuration) |

---

## Document Reference

| Document | Purpose | When to Use |
|----------|---------|-------------|
| **README.md** | Build order, file mapping | Start here, follow sequentially |
| [CRDS.md](./CRDS.md) | CRD type definitions, validation | Copy CRD types |
| [PRESETS.md](./PRESETS.md) | Quality and naming presets | Implement preset expansion |
| [TYPES.md](./TYPES.md) | IR types, adapter interface, compiler | Copy IR definitions |
| [OPERATIONS.md](./OPERATIONS.md) | Secrets, auto-discovery, merge rules, testing | Operational concerns |
| [DESIGN.md](./DESIGN.md) | Design philosophy, constraints | Architectural decisions |
| [RADARR.md](./RADARR.md) | Radarr API mappings (v3), adapter code | Implement Radarr adapter |
| [SONARR.md](./SONARR.md) | Sonarr API mappings (v3), adapter code | Implement Sonarr adapter |
| [LIDARR.md](./LIDARR.md) | Lidarr API mappings (v1), music quality | Implement Lidarr adapter |
| [PROWLARR.md](./PROWLARR.md) | Prowlarr API mappings (v1), indexer management | Implement Prowlarr adapter |

---

## Key Design Decisions

### Per-App CRDs (No ServiceBinding)

- **Old design:** `ServiceBinding` + `MediaPolicy` (generic)
- **New design:** `RadarrConfig`, `SonarrConfig`, etc. (app-specific)
- **Rationale:** Type safety, better validation, simpler mental model

### Bundled Configs + Granular Policies

- **Bundled configs:** All-in-one for quick setup (`RadarrConfig`)
- **Granular policies:** Override specific sections (`RadarrMediaPolicy`)
- **Rationale:** Simple path for 90% of users, power user path for customization

### Presets with Overrides

- **Presets:** `preset: "4k-hdr"` expands to full quality config
- **Overrides:** `exclude: ["dolby-vision"]` removes unwanted items
- **Rationale:** Sensible defaults, easy customization

### Auto-Discovery

| Feature | Discovery Method |
|---------|------------------|
| API keys | Secret ref > config.xml > convention-based secret |
| Client types | Inferred from name (`qbittorrent` -> QBittorrent) |
| Prowlarr apps | Auto-registration when `autoRegister: true` |
| Categories | Human-readable (`movies-hd`) -> numeric (2040) |

### Merge Hierarchy

```
ClusterNebularrDefaults  (lowest priority)
        ↓
   NebularrDefaults      (namespace-scoped)
        ↓
   BundledConfig         (RadarrConfig)
        ↓
   GranularPolicies      (highest priority)
```

---

## Key Constraints (Do Not Violate)

1. **Never mirror Radarr schemas** - Use abstract IR types
2. **Fail soft** - Missing features degrade, don't crash
3. **Tag ownership** - Only modify resources tagged `nebularr-managed`
4. **Idempotent** - Running twice = same result
5. **CRD-only state** - No external state files, use Status fields
6. **Per-app type safety** - RadarrConfig for Radarr, not generic MediaPolicy

---

## Expected Final Structure

```
nebularr/
├── api/v1alpha1/
│   ├── common_types.go            # Shared type definitions
│   ├── radarrconfig_types.go      # Bundled config
│   ├── sonarrconfig_types.go
│   ├── lidarrconfig_types.go
│   ├── prowlarrconfig_types.go
│   ├── qualitytemplate_types.go   # Shared resources
│   ├── nebularrdefaults_types.go
│   ├── clusternebularrdefaults_types.go
│   ├── radarrmediapolicy_types.go # Granular policies
│   ├── sonarrmediapolicy_types.go
│   ├── lidarrmusicpolicy_types.go
│   └── bazarrconfig_types.go      # Special
├── internal/
│   ├── compiler/
│   │   ├── compiler.go
│   │   └── merge.go
│   ├── ir/v1/
│   │   ├── ir.go
│   │   ├── connection.go
│   │   ├── quality.go
│   │   ├── download_client.go
│   │   ├── indexer.go
│   │   └── naming.go
│   ├── presets/
│   │   ├── video.go
│   │   ├── audio.go
│   │   ├── naming.go
│   │   └── override.go
│   ├── adapters/
│   │   ├── adapter.go
│   │   ├── registry.go
│   │   ├── retry.go
│   │   ├── radarr/
│   │   │   ├── adapter.go
│   │   │   ├── mapping.go
│   │   │   ├── formats.go
│   │   │   ├── clients.go
│   │   │   ├── indexers.go
│   │   │   ├── naming.go
│   │   │   ├── tags.go
│   │   │   └── client/client.go (generated)
│   │   ├── sonarr/
│   │   │   └── ... (similar structure)
│   │   ├── lidarr/
│   │   │   └── ... (similar structure)
│   │   ├── prowlarr/
│   │   │   └── ... (similar structure)
│   │   └── bazarr/
│   │       └── generator.go
│   ├── secrets/
│   │   └── resolver.go
│   ├── discovery/
│   │   ├── apikey.go
│   │   ├── download_client.go
│   │   └── categories.go
│   ├── webhook/
│   │   └── radarrconfig_webhook.go
│   ├── controller/
│   │   ├── radarrconfig_controller.go
│   │   ├── sonarrconfig_controller.go
│   │   └── ...
│   └── telemetry/
│       └── metrics.go
├── config/
│   ├── crd/
│   ├── rbac/
│   └── samples/
├── Dockerfile
├── Makefile
└── go.mod
```

---

## Verification Commands

```bash
# After each phase, verify:
make build      # Compiles
make test       # Tests pass
make manifests  # CRDs generate

# Test CRD installation:
kubectl apply -f config/crd/bases/
kubectl get crds | grep arr.rinzler.cloud

# Test sample CRs:
kubectl apply -f config/samples/
```
