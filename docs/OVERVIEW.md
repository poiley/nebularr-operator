# Nebularr — Implementation Overview (v0.1)

> **For coding agents:** Start with [README.md](./README.md) for build order. This document provides component overview.
>
> **Related:** [README](./README.md) | [DESIGN](./DESIGN.md) | [TYPES](./TYPES.md)

## 1. Implementation Strategy

**v0.1 build order:**

1. CRD definitions (Kubebuilder scaffold)
2. Intent validation (kubebuilder markers + semantic)
3. Intermediate Representation (IR) types
4. Policy compiler (intent → IR)
5. Radarr adapter (capability discovery, diff, apply)
6. Reconciliation via controller-runtime
7. OTEL metrics

No steps may be skipped.

---

## 2. Intent Layer

* CRD-only ingestion (no file fallback)
* **Per-app CRDs** for type safety (RadarrConfig, SonarrConfig, LidarrConfig, ProwlarrConfig)
* Two paths: Bundled configs (simple) and Granular policies (power users)
* Kubebuilder markers for schema validation
* Semantic validation in validating webhook or controller
* Must feed compiler consistently
* See [CRDS](./CRDS.md) for full details

---

## 3. Intermediate Representation (IR)

* Domain-based: video, audio, indexers, clients
* Must handle “cannot realize” states
* Versioned (e.g., v0.1)
* Modular and extensible
* Must not import adapter/service schemas

---

## 4. Policy Compiler

* Transforms intent → IR
* Applies defaults
* Resolves ambiguities
* Prunes unsupported features using capability discovery
* Deterministic and testable

---

## 5. Capability Discovery

* Periodic
* Caches capabilities
* Detects feature removal/additions
* Triggers safe degradation
* Used by compiler to prune intent

---

## 6. Adapter Implementation

* Compiled-in per service (v0.1: Radarr)
* Thin interface:

  * Discover - Query service capabilities
  * CurrentState - Fetch current managed state from service
  * Diff - Compare current state with IR (desired state)
  * Apply - Execute changes against service
* IR from compiler IS the desired state (no separate DesiredState method)
* Versioning internal to adapter
* Must be safe, fail-soft, and isolated
* Replaceable without changing core

---

## 7. Reconciliation Engine

* Core loop:

  1. Load intent
  2. Run compiler
  3. Iterate adapters
  4. Apply changes safely
  5. Sleep → repeat
* Idempotent
* Scoped failure handling
* Logs and metrics integrated

---

## 8. State Management

* K8s-native (CRD Status fields)
* Stores last-applied IR hash, capability cache, owned resources
* Survives controller restart (etcd persistence)
* No external state files needed

---

## 9. Metrics & Observability

* OTEL metrics required
* Log flow from intent → IR → adapter
* Metrics cover:

  * Reconciliation cycles
  * Capability changes
  * Drift detection
  * Adapter apply success/failure
* Must support alerting/monitoring in K8s

---

## 10. Deployment Model

* Single-binary Go controller
* Kubernetes-native: Deployment + CRDs + Secrets
* State stored in CRD Status fields
* Requires Kubernetes 1.35+

---

## 11. Testing Strategy

* Test order:

  1. Intent validation
  2. Compiler
  3. IR stability
  4. Adapter diff logic
  5. Reconciliation idempotency
* Do not test upstream schemas directly
* Do not rely on UI behavior

---

## 12. Open Implementation Questions (Resolved for v0.1)

1. Language: Go
2. Intent ingestion: CRD-only (K8s native)
3. Adapter: Compiled-in per service (v0.1 Radarr)
4. IR evolution: Versioned, domain-based, backward-compatible
5. State backend: K8s-native (CRD Status fields)
6. Metrics: OTEL
7. API client: Generated from OpenAPI spec (oapi-codegen)
8. v0.1 scope: Full implementation with all above features

---

## 13. Guiding Principle

> When in doubt, make the system less clever, not more powerful.

Complexity may be added later; coupling cannot.

---

## 14. Related Documents

| Document | Purpose |
|----------|---------|
| [README](./README.md) | Build phases, file mapping (start here) |
| [DESIGN](./DESIGN.md) | Core philosophy, architecture, constraints |
| [CRDS](./CRDS.md) | Per-app CRD definitions (RadarrConfig, etc.) |
| [TYPES](./TYPES.md) | IR types, adapter interface |
| [PRESETS](./PRESETS.md) | Quality and naming presets |
| [OPERATIONS](./OPERATIONS.md) | Auto-discovery, merge rules, Prowlarr integration |
| [RADARR](./RADARR.md) | Radarr adapter implementation |
| [SONARR](./SONARR.md) | Sonarr adapter implementation |
| [LIDARR](./LIDARR.md) | Lidarr adapter implementation |
| [PROWLARR](./PROWLARR.md) | Prowlarr adapter implementation |


