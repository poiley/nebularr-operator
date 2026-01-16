# Nebularr - Coding Agent Instructions

> **This file contains ready-to-use prompts for coding agents implementing Nebularr.**

---

## Quick Start (Single Agent)

Copy this prompt to start a single agent that implements everything sequentially:

```
Implement the Nebularr Kubernetes controller project.

Follow docs/ideas/nebularr_v2/README.md exactly, executing each phase in order.

Rules:
1. Execute phases 1-14 sequentially - do not skip phases
2. Copy code directly from the referenced documentation sections
3. Run `make build` after each phase to verify compilation
4. If a phase fails to build, fix errors before proceeding
5. Do not improvise - use only what's in the documentation

Start with Phase 1 (scaffold) and continue through Phase 14 (testing).
```

---

## Phased Approach (Multiple Agents)

For faster implementation, use separate agents for each phase.

### Agent 1: Foundation (Run First)

```
Create the Nebularr Kubernetes controller project foundation.

Working directory: Create new directory `nebularr/`

Tasks:
1. Execute Phase 1 from docs/ideas/nebularr_v2/README.md:
   - Run all Kubebuilder scaffold commands exactly as shown
   - Create the internal package directories

2. Execute Phase 2 (Bundled Config CRDs) from README.md:
   - Copy RadarrConfig types from CRDS.md Section 3.1 into api/v1alpha1/radarrconfig_types.go
   - Copy SonarrConfig types from CRDS.md Section 3.2 into api/v1alpha1/sonarrconfig_types.go
   - Copy LidarrConfig types from CRDS.md Section 3.3 into api/v1alpha1/lidarrconfig_types.go
   - Copy ProwlarrConfig types from CRDS.md Section 3.4 into api/v1alpha1/prowlarrconfig_types.go
   - Copy common types (ConnectionSpec, VideoQualitySpec, AudioQualitySpec) from CRDS.md Section 2

3. Verify:
   - Run `make manifests generate`
   - Run `make build`
   - All must succeed

Output: Confirm Phase 1-2 (CRDs) complete and building.
```

### Agent 2: IR Types (Run After Agent 1)

```
Implement Nebularr Intermediate Representation (IR) types.

Working directory: nebularr/

Tasks:
1. Execute Phase 4 from docs/ideas/nebularr_v2/README.md:
   - Copy IR envelope from TYPES.md Section 8.1 into internal/ir/v1/ir.go
   - Copy VideoIR from TYPES.md Section 8.2 into internal/ir/v1/video.go
   - Copy AudioIR from TYPES.md Section 8.3 into internal/ir/v1/audio.go
   - Copy DownloadClientIR from TYPES.md Section 8.4 into internal/ir/v1/clients.go
   - Copy IndexerIR from TYPES.md Section 8.5 into internal/ir/v1/indexers.go

2. Execute Phase 5 from README.md:
   - Copy Adapter interface from TYPES.md Section 6.1 into internal/adapters/adapter.go
   - Include all types: Connection, Capabilities, ChangeSet, Change, ApplyResult, ApplyError

3. Execute Phase 6 from README.md:
   - Copy Compiler interface from TYPES.md Section 2.2 into internal/compiler/compiler.go

4. Verify:
   - Run `make build`
   - Must succeed

Output: Confirm IR types and interfaces complete and building.
```

### Agent 3: Radarr Adapter (Run After Agent 2)

```
Implement the Nebularr Radarr adapter.

Working directory: nebularr/

Tasks:
1. Execute Phase 8 from docs/ideas/nebularr_v2/README.md:
   - Install oapi-codegen
   - Download Radarr OpenAPI spec
   - Generate client into internal/adapters/radarr/client/client.go

2. Execute Phase 9 from README.md:
   - Copy quality mapping from RADARR.md Section 1.2 into internal/adapters/radarr/mapping.go
   - Copy format mapping from RADARR.md Section 2.2 into internal/adapters/radarr/formats.go
   - Copy client mapping from RADARR.md Section 3.2 into internal/adapters/radarr/clients.go
   - Copy indexer mapping from RADARR.md Section 4.4 into internal/adapters/radarr/indexers.go
   - Copy category mapping from RADARR.md Section 4.3 into internal/adapters/radarr/categories.go
   - Copy adapter struct from RADARR.md Section 6.2 into internal/adapters/radarr/adapter.go
   - Copy tag functions from RADARR.md Section 5.1 into internal/adapters/radarr/tags.go

3. Verify:
   - Run `make build`
   - Must succeed

Output: Confirm Radarr adapter complete and building.
```

### Agent 4: Controllers (Run After Agent 3)

```
Wire Nebularr controllers to complete the implementation.

Working directory: nebularr/

Tasks:
1. Implement RadarrConfig controller reconciliation in internal/controller/radarrconfig_controller.go:
   
   The Reconcile function must:
   a. Get the RadarrConfig from the API
   b. Resolve connection (URL + API key from secret or config.xml auto-discovery)
   c. Get or create Radarr adapter
   d. Call adapter.Discover() to get capabilities (cache in Status)
   e. Compile intent to IR using compiler (prune based on capabilities)
   f. Call adapter.CurrentState() to get current managed state
   g. Call adapter.Diff() to compute changes
   h. Call adapter.Apply() to apply changes
   i. Update RadarrConfig.Status with results
   j. Requeue after Spec.SyncInterval
   
   Reference: TYPES.md Section 4.1 for reconciliation flow

2. Register the Radarr adapter in internal/adapters/registry.go

3. Add OTEL metrics scaffolding in internal/telemetry/metrics.go
   Reference: TYPES.md Section 5.1 for metric definitions

4. Verify:
   - Run `make build`
   - Run `make test`
   - Both must succeed

5. Create sample CRs in config/samples/ for testing:
   - config/samples/arr_v1alpha1_radarrconfig.yaml

Output: Confirm controllers wired and tests passing.
```

---

## Verification Commands

After each agent completes, verify:

```bash
cd nebularr

# Must all succeed:
make manifests    # Generate CRD manifests
make generate     # Generate deepcopy functions
make build        # Compile
make test         # Run tests
```

---

## Key Constraints (Agents Must Follow)

1. **Copy code verbatim** - Do not improvise or "improve" the documented code
2. **Follow phase order** - Dependencies exist between phases
3. **Build after each phase** - Catch errors immediately
4. **Use documented file paths** - Don't reorganize the structure
5. **Reference README.md** - It has the authoritative file mapping and phase order

---

## CRD Hierarchy (Per-App Pattern)

Nebularr uses per-app CRDs for type safety:

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

## Document Reference

| Document | Contains | Used In |
|----------|----------|---------|
| [README.md](./README.md) | Build phases, file mapping | All phases |
| [TYPES.md](./TYPES.md) | IR types, adapter interface, compiler | Phases 4-6, 13 |
| [CRDS.md](./CRDS.md) | CRD type definitions | Phases 2-3 |
| [RADARR.md](./RADARR.md) | Radarr adapter code | Phases 8-9 |
| [SONARR.md](./SONARR.md) | Sonarr adapter code | Phases 10 |
| [LIDARR.md](./LIDARR.md) | Lidarr adapter code | Phases 10 |
| [PROWLARR.md](./PROWLARR.md) | Prowlarr adapter code | Phases 11 |
| [PRESETS.md](./PRESETS.md) | Quality and naming presets | Phase 7 |
| [OPERATIONS.md](./OPERATIONS.md) | Auto-discovery, merge rules | Reference |
| [DESIGN.md](./DESIGN.md) | Architecture philosophy | Reference only |

---

## Troubleshooting

### "make build" fails after Phase 1
- Ensure Go 1.22+ is installed
- Ensure Kubebuilder 4.x is installed
- Run `go mod tidy`

### "make manifests" fails after Phase 2
- Check kubebuilder markers are correct in CRD types
- Ensure all types have `+kubebuilder:object:root=true` markers

### Generated client fails to compile (Phase 8)
- The OpenAPI spec may have issues - wrap problematic types
- Check oapi-codegen version is v2

### Import errors
- Run `go mod tidy` after adding new files
- Ensure package names match directory names
