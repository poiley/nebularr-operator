# Nebularr — Design Philosophy & Architecture (v0.1)

> **For coding agents:** Start with [README.md](./README.md) for build order. This document is for architectural decisions only.
>
> **Related:** [README](./README.md) | [TYPES](./TYPES.md) | [CRDS](./CRDS.md)

## 1. Problem Statement

The *arr ecosystem (Radarr, Sonarr, Lidarr, Prowlarr, download clients) lacks a stable, declarative, reproducible configuration management solution that:

* Supports indexers and download clients
* Survives frequent upstream API and schema changes
* Runs continuously as a service
* Is suitable for Kubernetes environments
* Does not require CI/CD pipelines
* Provides operational observability

Previous attempts (e.g., Buildarr) failed primarily due to **tight coupling to unstable internal schemas** of managed services.

This project exists to solve that problem **correctly and durably**.

---

## 2. Core Philosophy

### 2.1 Intent over Implementation

* Users express *what they want* (policies, constraints, preferences) via CRDs, files, or APIs
* The system decides *how to realize that intent* given current capabilities
* Internal schemas of managed services are treated as volatile and untrusted

Mirroring Radarr/Sonarr configuration models directly is **prohibited**.

---

### 2.2 Controller, Not a Script

This project is a **long-running controller service**, not:

* A one-shot CLI
* A config generator
* A UI replacement

It continuously reconciles desired state against observed state.

---

### 2.3 Graceful Degradation Is Mandatory

If a managed service removes a field, changes behavior, or loses a feature, the controller must:

* Detect the change via **capability discovery**
* Degrade behavior safely
* Continue operating without failure

Hard failures due to upstream changes are unacceptable.

---

## 3. High-Level Architecture

### 3.1 Conceptual Layers

1. **Intent Layer**

   * CRDs (Kubernetes-native)
   * Stable and versioned
   * Never mirrors upstream schemas

2. **Intermediate Representation (IR)**

   * Domain-based (video, audio, indexers, clients)
   * Normalized, loss-tolerant
   * Expresses meaning, not knobs
   * Includes “cannot realize” states
   * Versioned for backward compatibility

3. **Policy Compiler**

   * Translates intent → IR
   * Applies defaults, resolves ambiguity
   * Prunes unsupported features using discovered capabilities

4. **Adapters**

   * One compiled-in adapter per service (v0.1: Radarr)
   * Thin interface for API, diff, and apply logic
   * Capability-aware and fail-soft
   * Versioned internally

5. **Reconciliation Engine**

   * Orchestrates discovery, diffing, and application
   * Enforces idempotency
   * Periodically reconciles state

---

### 3.2 Explicit Non-Architecture

This project is **not**:

* A direct API client library
* A schema synchronization tool
* A generic workflow engine
* A Kubernetes-only solution

---

## 4. Scope Definition

### 4.1 In Scope

* Managing *arr configuration through intent-based policies
* Indexer and download client configuration (via Prowlarr adapter future-ready)
* Quality, format, and release preferences
* Continuous reconciliation
* Kubernetes-native deployment
* Partial ownership and tagging of resources
* Drift detection and correction
* OTEL metrics
* K8s-native state management (CRD Status)

---

### 4.2 Out of Scope (Non-Negotiable)

* Full ownership of all *arr settings
* UI replacement for *arr applications
* Automatic media import or deletion logic
* Scraping undocumented endpoints
* Direct database access
* CI/CD dependencies

Violating these rules invalidates the design.

---

## 5. Adapter Design Rules

Adapters are **the only components allowed to interact with external services**.

### 5.1 Responsibilities

Adapters may:

* Discover service capabilities
* Read current managed state
* Compute diffs against desired state
* Apply safe, minimal changes

Adapters may **not**:

* Expose upstream schemas to the core
* Enforce policy logic
* Assume features exist
* Fail the controller due to missing features

---

### 5.2 Versioning Strategy

* Adapters handle multiple service API versions internally
* Core never branches based on service versions
* Adapters may log degraded capabilities

---

## 6. Capability Discovery

* **Mandatory and periodic**
* Covers codecs, quality sources, scoring, indexer types, and client features
* Cached and timestamped for reconciliation
* Any feature absence triggers safe degradation

---

## 7. State Ownership & Safety

* Controller owns **only what it creates**
* Owned resources are explicitly tagged
* No mutation/deletion of unowned resources
* Idempotent reconciliation cycles
* State stored in CRD Status for last-applied IR hash, capability cache, and reconciliation metadata

---

## 8. Deployment Model

* Single-binary Go controller
* Kubernetes-native deployment: Deployment, CRDs, Secrets
* Requires Kubernetes 1.35+
* State stored in CRD Status (no external state files)

---

## 9. Upgrade & Compatibility Strategy

* **Controller upgrades:** intent schema versioned; compiler handles migration
* **Managed service upgrades:** handled by adapters; no forced controller changes
* **IR evolution:** domain-based, versioned, backward-compatible with migration warnings

---

## 10. Observability

* **OTEL metrics required**

  * Reconciliation cycles, capability changes, drift detection, adapter apply status
* Logs must indicate intent → IR → adapter flow
* Metrics and logs designed to detect degradation and failures early

---

## 11. Development Guidelines

**To Do:**

* Treat intent schema as the public API
* Treat IR as the stability contract
* Isolate all service-specific logic in adapters
* Prefer omission over failure
* Log capability loss explicitly
* Design for maintainable long-term operation

**Not To Do:**

* Mirror Radarr/Sonarr schemas
* Assume API stability
* Require UI parity
* Introduce tight coupling
* Centralize logic in adapters
* Fail on unsupported features
* Silently delete user configuration

---

## 12. Research & Inspiration

* Kubernetes operators/controllers
* Terraform provider architecture
* Intent-based networking models
* Policy compilers
* Capability-based system design

---

## 13. Success Criteria

* Works across multiple *arr releases
* Configurations reproducible
* Partial failures do not cascade
* Drift is corrected
* Maintenance burden bounded
* Metrics and state provide full operational visibility

---

## 14. Final Design Principle

> **Stability is achieved by refusing to know too much.**

Less knowledge of upstream internals → longer survival.

---

## 15. Related Documents

| Document | Purpose |
|----------|---------|
| [README](./README.md) | Build order, file mapping (start here) |
| [OVERVIEW](./OVERVIEW.md) | Component details, testing strategy |
| [TYPES](./TYPES.md) | IR types, adapter interface, Kubebuilder scaffold |
| [CRDS](./CRDS.md) | CRD schemas, validation rules |
| [RADARR](./RADARR.md) | Quality/format mappings for adapter implementation |

---

## 16. Technical Decisions (v0.1)

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Project Name | **Nebularr** | User specified |
| API Group | `arr.rinzler.cloud` | Custom domain, stable, professional |
| CRD Kinds (Bundled) | RadarrConfig, SonarrConfig, LidarrConfig, ProwlarrConfig, BazarrConfig | Per-app type safety, simple path |
| CRD Kinds (Granular) | Radarr/Sonarr/LidarrMediaPolicy, *DownloadClientPolicy, *IndexerPolicy | Power user path |
| CRD Kinds (Shared) | QualityTemplate, NebularrDefaults, ClusterNebularrDefaults | Reusable configurations |
| Tooling | Kubebuilder go/v4, controller-runtime | Industry standard |
| Target K8s | 1.35+ | Latest stable |
| State Backend | K8s-native (CRD Status) | No external dependencies, survives restarts |
| Metrics | OTEL | Standard observability |

> **Note:** See [CRDS.md](./CRDS.md) for authoritative CRD definitions. The original v0.1 design used generic CRDs (ServiceBinding, MediaPolicy). The current design uses per-app CRDs for type safety.

