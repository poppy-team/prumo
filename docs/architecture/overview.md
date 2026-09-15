# Prumo Architecture Overview

## High-Level Topology

```mermaid
flowchart LR
    H[Human / Agent Request] --> HA[Harness Adapter]
    HA --> CORE[Prumo Core - Go]
    CORE --> PP[Protocol & Policy Engine]
    CORE --> DE[Documentation Engine]
    CORE --> PL[Planning Engine]
    CORE --> AD[Adoption Engine]
    CORE --> EX[Experience Engine]
    CORE --> KN[Knowledge Engine]
    CORE --> VG[Evidence & Gate Engine]
    CORE --> CP[Control Plane]
    PP --> STORE[(Canonical Repository: Git + Markdown + JSON + Schemas)]
    DE --> STORE
    PL --> STORE
    AD --> STORE
    EX --> IDX[(Derived Index: SQLite / Search)]
    KN --> IDX
    VG --> STORE
    CP --> IDX
    CP --> EP[Execution Plane]
    EP --> IP[Integration Plane]
    IP --> H
```

## Planes & Boundaries

### Major Components

The major components are Protocol Domain, Project Service, Resolution Service, Documentation Engine, Planning Engine, Adoption Engine, Knowledge Engine, Experience Engine, Evidence/Gate Engine, Control Plane, Execution Plane, and Integration Plane.

### Control Plane (Horizontal Infrastructure)
| Service | Responsibility |
|---------|----------------|
| `RunService` | Run lifecycle, checkpoints, retry, resume, cancellation, livelock detection |
| `BudgetService` | Hierarchical budget envelopes, cost governance, rate limits |
| `ContextService` | Context Compiler pipeline, token budgets, cache awareness, compaction |
| `ModelRoutingService` | Model Registry, eval-driven router, drift detection, fallback |
| `ToolGateway` | Tool descriptors, lazy discovery, side-effect journal, MCP governance |
| `EnvironmentService` | Sandbox contract, isolation requirements, execution environments |
| `AutomationService` | Event-driven rules, DLQ, idempotency, concurrency keys |
| `ObservabilityService` | Structured telemetry, explainability, incident bundles |
| `PackageRuntimeService` | `prumo.lock`, provider isolation, supply chain verification |

### Knowledge Plane (Canonical State)
| Engine | Responsibility |
|--------|----------------|
| Goals | Lifecycle, locks, amendments, state transitions |
| Documentation | Contracts, profiles, readiness, delta, contradictions |
| Planning | Interview, decisions, open questions, confidence |
| Adoption | Scanner, semantic mapping, gap detection, migration |
| Experience | Session events, summaries, handoffs, proposals |
| Traceability | Typed edges: req↔dec↔code↔test↔doc↔evidence |

### Quality Plane (Verification)
| Component | Responsibility |
|-----------|----------------|
| Test Providers | Contract-based: Playwright, ZAP, CodeQL, fuzzers, sanitizers, UIA/AX/AT-SPI, KUnit, syzkaller |
| Quality Orchestrator | Deterministic plan from change impact + risk + capabilities |
| Evidence Normalization | Common schema for runs, findings, artifacts, environment fingerprint |
| Security Verifier | Independent verification for high/critical risk |

### Execution Plane (External Providers)
Models, tools/MCPs, sandboxes/runtimes, external providers. **Never import canonical rules.**

### Integration Plane (Adapters)
OpenCode, Codex, Claude Code, Gemini CLI, Copilot CLI, Kiro, Generic. **Thin adapters only — no canonical logic.**

## Dependency Direction

```mermaid
flowchart TD
    IP[Integration Plane] --> CP[Control Plane]
    CP --> KP[Knowledge Plane]
    CP --> QP[Quality Plane]
    CP --> EP[Execution Plane]
    QP --> EP
    KP --> CR[(Canonical Repository)]
    CP --> DS[(Derived Runtime State)]
    QP --> DS
```

**Rules:**
- Domain/Protocol never imports CLI, concrete storage, or harness
- CLI depends on Application Services
- Storage implements ports defined by consumers
- Integrations call Prumo Core via stable CLI JSON/stdio or versioned API
- SQLite never required to interpret canonical state
- External harness never controls invariants

## Interface Boundaries

Interfaces exist **only** at real boundaries with multiple implementations:
- `Repository` (Git filesystem)
- `EventSink` (local file, OTLP, in-memory)
- `DerivedIndex` (SQLite, in-memory, future)
- `HarnessTransport` (stdio, HTTP, MCP)
- `Clock` (tests)
- `EmbeddingProvider` (future)

**Avoid premature interfaces** during Go migration.

## Internal Machine API

Versioned, stable commands for plugins/adapters:
```
prumo internal project-status --json
prumo internal resolve --json
prumo internal validate-tool --json
prumo internal docs-impact --json
prumo internal context --json
prumo internal handoff --json
```

## Protocol Version Negotiation

Every adapter declares:
- connector id/version
- Prumo protocol version supported
- capabilities
- lifecycle hooks
- write-blocking support
- agent/skill primitives

Prumo refuses "strict" enforcement when harness lacks sufficient primitives.

## Design Principle

> **Hard-code invariants; configure policies; evaluate heuristics.**
>
> - Invariants: Goal lock, schema validation, evidence requirements
> - Policies: independent verifier for critical risk, budget limits
> - Heuristics: workforce size, model routing, context packing (subject to evals)