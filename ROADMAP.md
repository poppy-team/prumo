# Prumo Evolution Roadmap: v0.1.0 → v1.0.0

> **Authority**: Canonical specification.  
> **Classification**: `docs/AUTHORITY_MAP.json` (`role: canonical`, `owner: prumo-maintained`).  
> **Status**: Authoritative project roadmap, historical record, and release projection.

---

## 1. Executive Vision & Architectural Invariants

**Prumo** is the open-source engineering protocol and autonomous harness designed for paired human-agent and multi-agent software engineering. Rather than treating AI code assistants as opaque chat widgets or unconstrained script runners, Prumo provides a deterministic control plane based on strict architectural invariants:

1. **Protocol-Oriented Programming (POP)**: System interactions occur via versioned, typed contracts with machine-verifiable JSON envelopes (`protocol_version`, `ok`, `data`, `diagnostics`, `warnings`).
2. **Lean Progressive Context (LPC)**: Token budgets and context windows are strictly managed via hierarchical disclosure tiers (L0 metadata → L1 summaries → L2 full texts → L3 AST symbols → L4 deep context), preventing context rot and model hallucination.
3. **Provider Neutrality**: No specific LLM vendor or orchestrator is hardcoded or mandatory. Model selection, quotas, and capabilities are dynamically configured by the workspace or resolved from active runtime environment variables (`PRUMO_MODEL`, `PRUMO_PROVIDER`).
4. **Clear Surface Boundaries**:
   - **Core Harness & CLI (`prumo`)**: Pure Go single binary providing protocol authority, execution runtime, sandboxed tool gateway (ACI), and headless automation.
   - **Terminal User Interface (`prumo-tui`)**: Dedicated, keyboard-driven terminal client built with Go and Bubble Tea v2, scheduled for full production delivery in **v0.7.0**.
   - **Desktop Workspace Viewer & IDE (`prumo-viewer` / `prumo-native`)**: Native, agent-aware desktop GUI built from scratch in Rust with **Freya 0.4+** (Skia rasterization, Torin layout engine, Tree-sitter syntax highlighting, embedded PTY, and out-of-process Extension SDK), scheduled for production release in **v1.0.0**.

---

## 2. Historical Milestones (v0.1.0 — v0.6.0)

```
2026-08-10      2026-08-15      2026-08-20      2026-09-09      2026-09-10      2026-09-27
   v0.1.0          v0.2.0          v0.3.0          v0.4.0          v0.5.0          v0.6.0 (Current)
     │               │               │               │               │               │
     ▼               ▼               ▼               ▼               ▼               ▼
  Python          Protocol        Python          Go Core         Brand &         Unified CLI,
  Genesis        Conformance      Freeze         Migration       Doc Control     Adoption v3 &
 Prototype       & Approvals     (ADR 001)       (M1 - M4)       Plane (W0-W21)  Dynamic Roster
```

### v0.1.0 — Genesis & Conceptual Prototype (August 2026)
- **Core Technology**: Python 3.11+.
- **Key Deliverables**:
  - Initial formalization of Protocol-Oriented Programming (POP) and Lean Progressive Context (LPC).
  - Definition of the canonical AI Workforce Catalog (`catalog.json`): 39 specialized agent archetypes, 189 engineering skills, and 20 deterministic execution recipes.
  - Basic Goal lifecycle prototype (`new`, `lock`, `amend`) with SHA-256 state hashing.
  - Prototype rule compilers targeting early markdown rulesets.

### v0.2.0 — Protocol Conformance & Envelope Standardization (August 2026)
- **Key Deliverables**:
  - Introduction of standard machine envelope contracts across CLI operations.
  - JSON Schema validation baseline (Draft 2020-12) for project manifests (`prumo.json`), goals, and workforce packages.
  - Interactive human-in-the-loop approval protocol baseline for risky tool executions.
  - Generation of deterministic golden fixtures for regression test suites.

### v0.3.0 — Specification Freeze & Protocol Inventory (August 2026)
- **Key Deliverables**:
  - Exhaustive protocol inventory cataloging all CLI commands, exit codes, and filesystem mutations (`docs/migration/protocol-inventory.md`).
  - Establishment of the immutable migration baseline (`conformance/V03_BASELINE.json`) backed by 127 passing Python test suites.
  - Model discovery protocol operation (`models` op) exposing provider model capabilities.
  - Architectural decision freeze: **ADR 001** and **ADR 002** accepted to retire Python and rewrite the Core into a high-performance, single-binary Go application.

### v0.4.0 — The Go Core Migration & Hardening (September 2026)
- **Core Technology**: Go 1.22+ (Single static binary `prumo-agent`).
- **Key Deliverables**:
  - Execution of Migration Phases M1 through M4: 100% protocol domain parity with zero regressions against Python fixtures.
  - 33 language engineering and safety skill profiles embedded into the Go binary.
  - High-performance Agent Runtime and Headless Daemon with TCP+TLS transport and bearer token authentication.
  - Real-time push streaming protocol (`subscribe` op, ADR 014) replacing polling loops.
  - On-demand syntax-aware file diff inspection (`diff` op, ADR 014).
  - H10 TUI experimental vertical slice (Bubble Tea v2 proving terminal client viability).

### v0.5.0 — Identity Rebranding & Documentation Control Plane (September 2026)
- **Key Deliverables**:
  - Official framework rebrand from Project Atlas to **Prumo** (ADR 004).
  - Execution of Pre-TUI Waves W0 through W21.
  - Implementation of the **Semantic Documentation Control Plane (W15)**: moving from naive lexical matching to verifiable requirement → claim → evidence validation (`prumo docs readiness`).
  - Continuous Documentation Gauntlet runtime (W19) enforcing strict zero-drift gates (`prumo docs verify --strict`).
  - Standardized ACP (Agent Communication Protocol) v1 server and MCP (Model Context Protocol) stdio/HTTP transports.

### v0.6.0 / v0.6.1 — Unified CLI, Modern Harness & Adoption Engine v3 (Current)
- **Core Technology**: Pure Go 1.26 toolchain; single unified binary `prumo` (ADR 016).
- **Key Deliverables**:
  - **CLI & Harness Consolidation (ADR 016)**: Merged separate tools into a single coherent binary `prumo`, providing zero-ceremony `prumo init` with interactive cognitive dashboard.
  - **Brownfield Adoption Engine v3 (`prumo adopt`)**: Automated detection of 15+ tech stacks and package managers, reversible non-destructive adoption, and intelligent configuration generation.
  - **Connector Lifecycle Subsystem (`prumo connector status/install`)**: Native detection and setup for Windsurf, Cursor, Claude Code, Cline, OpenCode, Codex, Antigravity, and Gemini CLI.
  - **Dynamic & Neutral Model Resolution**: Eradication of hardcoded vendor model lists; dynamic fallback to runtime active models or `PRUMO_MODEL`/`PRUMO_PROVIDER`.
  - **Multimodal Image Resolution (GAP-090)**: Text `@path` image references resolved under declared vision capability gates with strict 10 MB bounds.
  - **Delegated Provider Execution (GAP-093)**: Full delegated execution via `--provider opencode` utilizing external provider auth, tooling, and cost accounting.

---

## 3. Projection Roadmap (v0.7.0 — v1.0.0)

```
2026-10 (Target)          2026-11 (Target)          2026-12 (Target)          2027-Q1 (Target)
     v0.7.0                    v0.8.0                    v0.9.0                    v1.0.0
       │                         │                         │                         │
       ▼                         ▼                         ▼                         ▼
> **Detailed Granular Plan**: For the complete patch-by-patch technical specification, file-level deliverables, and competitive analysis against Claude Code, OpenCode, Aider, and Cline, see [`docs/development/granular-implementation-plan.md`](file:///home/raillen/Documentos/Projetos/prumo/docs/development/granular-implementation-plan.md) and [`docs/granular-plan.md`](file:///home/raillen/Documentos/Projetos/prumo/docs/granular-plan.md).

```
v0.6.x (Hardening)      v0.7.x (Production TUI)       v0.8.x (Gateway & Resilience)      v0.9.x (Multi-Agent)           v1.0.x (Official Release & IDE)
├── v0.6.0 (Completed)  ├── v0.7.0 (Oct 2026)          ├── v0.8.0 (Nov 2026)              ├── v0.9.0 (Dec 2026)          ├── v1.0.0-alpha (Jan 2027)
├── v0.6.1 (Completed)  ├── v0.7.1 (TUI Ergonomics)    ├── v0.8.1 (Provider Failover)     ├── v0.9.1 (Dual Architect)    ├── v1.0.0-beta (Feb 2027)
├── v0.6.2 (Onda 0 Fix) ├── v0.7.2 (Git & Web in TUI)  └── v0.8.2 (Local Models)          └── v0.9.2 (Verification Loop)├── v1.0.0-rc1/rc2 (Mar 2027)
└── v0.6.3 (Onda 1 Fix) └── v0.7.3 (Tree-sitter Repo)                                                                    └── v1.0.0 GA (March 2027)
```

---

### v0.6.x Line — Core Hardening & Integrity

- **v0.6.0 (Completed ✅)**: Unified CLI `prumo` (ADR 016), Brownfield Adoption v3 (`prumo adopt`), Connector lifecycle management, dynamic model resolution.
- **v0.6.1 (Completed ✅)**: CLI ergonomics overhaul, automated stack detection, zero-arg `prumo init` with interactive cognitive dashboard, and official documentation portal.
- **v0.6.2 (Target: Early Oct 2026) — Onda 0: Execution Fidelity**:
  - Propagate actual subprocess exit codes (GAP-097; broken tests fail closed).
  - Honest session resume preventing premature task completion (GAP-123).
- **v0.6.3 (Target: Mid Oct 2026) — Onda 1: Security Boundaries & Sandboxing**:
  - Strict canonical symlink resolution before mutations (GAP-110).
  - Provider endpoint verification against allowlists protecting credentials (GAP-111).
  - Path traversal protection on execution IDs via `safepath` (GAP-112).

---

### v0.7.0 — Production Terminal User Interface (`prumo-tui`)
*Target: 15 October 2026*  
*Objective: Deliver a complete, beautiful, keyboard-first, self-contained terminal experience for interactive agent pairing.*

- **Surface**: `prumo-tui` (built from `prumo-tui/cmd/prumo-tui`, launched transparently via `prumo agent` or `prumo tui`).
- **Key Features (v0.7.0)**:
  1. **Leader Key System (`Ctrl+X`)**: OpenCode-inspired mnemonic sub-chords (`Ctrl+X N` new session, `Ctrl+X L` list, `Ctrl+X U` undo, `Ctrl+X M` model) avoiding GNU Readline terminal collisions.
  2. **Composer Input Prefixes**:
     - `@`: Fuzzy file and folder attachment autocomplete.
     - `!`: Immediate shell pass-through command execution without calling LLM (e.g., `!git status`).
     - `/`: Slash commands (`/help`, `/model`, `/provider`, `/sidebar`, `/init`).
  3. **Unified Trinity of Operating Modes**: Full interactive TUI (`prumo tui`), quiet scriptable CLI one-shot (`prumo agent "<prompt>" -q` / `prumo run`), and headless daemon (`prumo daemon`).
  4. **Unified TUI Supervision**: `prumo agent` / `prumo tui` automatically checks local socket availability, spawning or connecting to the headless daemon transparently without double-daemon collisions (GAP-086).
  5. **Multi-Session Management & Conversation Folding**: Full persistent session browsing, resumption, and timeline compaction with constant-memory rendering of message tails (GAP-052, GAP-080).
  6. **Interactive Touched Files & Diff Viewer**: `ctrl+g` / files dialog with syntax highlighting and side-by-side or unified diff viewing powered by the `diff` protocol operation (GAP-084).
  7. **Dynamic User Command Palette**: User-defined prompt library (`$XDG_CONFIG_HOME/prumo-tui/commands/*.md`) with frontmatter metadata and interactive placeholder prompts (`{{arg}}`) (GAP-082).
  8. **Model Capabilities & Badges**: Dynamic inspection badges (`[text reasoning vision tools audio]`) displaying capabilities declared in `.prumo/models.json` (GAP-089).
  9. **A11y & Cognitive Clarity Compliance**: 9 verified built-in themes with WCAG 4.5:1 contrast (GAP-070), reduced motion mode (GAP-065), glyph-based focus indicators (GAP-062), and linear screen-reader outputs (`--prompt`, `--plain`, `--json`, GAP-074).
  10. **Timeline Plaintext Export**: Export complete session logs to `.prumo/runtime/exports/<run>.txt` with deterministic key ordering and separated token/cache cost metrics (GAP-067, GAP-087).

- **v0.7.1 (Target: 25 Oct 2026) — TUI Ergonomics, LSP & Real-Time Telemetry**:
  - **Semantic LSP Tooling**: Native integration of `internal/harness/lsp` (`goToDefinition`, `findReferences`, and `getDiagnostics`), catching compiler diagnostics before running slow test suites.
  - Interactive collapsible `<think>` reasoning accordions with duration timer and `Ctrl+O` toggle.
  - Paginated tool execution cards (auto-fold outputs > 10 lines with modal viewer).
  - Real-time Cache Hit Ratio % and Context Window gauge on statusline (`ctx: 35%`).
  - Optional Vim keybindings (`j`/`k` scroll in transcript) and `$EDITOR` prompt drafting (`Ctrl+E`).

- **v0.7.2 (Target: 05 Nov 2026) — Git & Web Tooling in TUI**:
  - Atomic `/undo` command (`git reset --hard HEAD~1` + checkpoint rollback).
  - Pre-flight dirty working tree protection dialog (`[s] Stash | [c] Commit | [a] Abort`).
  - Automated Conventional Commit generation upon task completion.
  - Web content distillation (`/web <url>`) converting pages to clean Markdown capsules.
  - Direct clipboard image pasting (`Ctrl+V`) with 24-bit ANSI half-block preview.

- **v0.7.3 (Target: 15 Nov 2026) — Tree-Sitter Repo Map & PageRank**:
  - Language-agnostic Tree-sitter AST queries replacing regex scanner.
  - Personalized PageRank over repository symbol reference graph.
  - Binary search fitting into strict token budget (`--map-tokens`).

---

### v0.8.x Line — Resilient Gateway & Quota-Aware Multi-Provider Router
*Target: November 2026*  
*Objective: Solidify process safety, honest execution reporting, and intelligent multi-model routing in production.*

- **v0.8.0 (Target: 25 Nov 2026) — Gateway in Production & Resilient Runtime**:
  - **GAP-102**: Connect `internal/harness/gateway` directly into `r.Svc.Models.Stream` in production runtime (ADR 020).
  - **GAP-103**: Circuit breaker cooldown auto-recovery upon expiry of `CooldownUntil`.
  - **GAP-108**: Exponential backoff with jitter and standard HTTP `Retry-After` adherence for rate-limited providers (429 / `RESOURCE_EXHAUSTED`).
  - **GAP-098 & GAP-100**: Hard daemon budget ceilings and preflight token/cost validation before initiating LLM requests.
  - **GAP-104 & GAP-105**: Atomic daemon PID file locking with `O_EXCL`/`flock` and graceful drain shutdown on SIGTERM.
  - **GAP-106 & GAP-107**: Persistent human approval decisions reloaded by action/resource fingerprint across restarts.

- **v0.8.1 (Target: 05 Dec 2026) — Multi-Provider Failover & Quota Optimization**:
  - Priority routing (Local/Free → Cloud) with seamless failover upon quota exhaustion.
  - Real-time token quota tracking and dynamic tier reallocation without session drop.

- **v0.8.2 (Target: 15 Dec 2026) — Local Intel & Offline Support**:
  - Native inference connectors for `llama-server`, Ollama, and vLLM (GAP-025, GAP-040).
  - Hardware-aware context sizing and search/replace diff fallback parsing for smaller open-weight models (7B–14B).

---

### v0.9.x Line — Multi-Agent DAG Orchestration & Subagent Delegation
*Target: December 2026*  
*Objective: Enable scalable, autonomous multi-agent task execution with bounded hierarchy and worktree isolation.*

- **v0.9.0 (Target: 22 Dec 2026) — Production Agent Teams & Git Worktrees**:
  - **GAP-003 & GAP-101**: Activate `team.Runner` / `RunWork` in the production CLI and daemon, allowing roles to execute nested runs with distinct budgets and checkpoints.
  - **ADR 017, GAP-022, GAP-109**: Expose `agent.delegate` and `agent.ask` tools to agents with strict depth enforcement (`max_delegation_depth = 1`). Prevent runaway recursive delegation.
  - **GAP-008 & GAP-168**: Orchestrate isolated Git worktrees per subagent; automated three-way merge verification with explicit conflict surfacing.
  - **Dynamic Cross-Provider Handoff Protocol (M9 / GAP-135)**: Structured state transfer between agents running on different provider backends (Claude Code ↔ Codex ↔ Local Models).
  - **GAP-146 & DOC-GAP-024**: Dynamic resolution of workforce skills and recipes based on Goal semantic impact analysis.

- **v0.9.1 (Target: 10 Jan 2027) — Dual Architect / Editor Pattern**:
  - Dedicated `/architect` and `/code` modes toggleable via `Ctrl+M`.
  - Architect generates task blueprints with read-only tools; Editor executes surgical search/replace edits.

- **v0.9.2 (Target: 20 Jan 2027) — Verification-Guided Repair Loop**:
  - Automated test-and-lint feedback injection in runtime `EvaluateStop` phase.
  - Bounded $K$-retry repair loop with visual TUI execution cards.

---

### v1.0.x Line — Flagship Production Release: Prumo Native Workspace Viewer (Rust IDE GUI) & Enterprise Ecosystem
*Target: Q1 2027*  
*Objective: Full production readiness across all surfaces: Go Core Harness, Bubble Tea TUI, and Freya Native Desktop IDE.*

- **v1.0.0-alpha (Target: 30 Jan 2027) — Prumo Native Workspace Viewer / IDE**:
  - Desktop GUI built from scratch in Rust on **Freya 0.4+** (declarative Rust GUI, Skia GPU/CPU rasterization, Torin layout engine).
  - Agent-Aware Architecture (CONST-82): ultra-fast workspace viewer and editor to observe agent execution in real-time, inspect file trees, review diffs, and perform targeted human edits without heavy Electron overhead.
  - Tree-sitter code editor, embedded PTY terminal (`portable-pty` + `vt100`), and diff viewer (`similar`).

- **v1.0.0-beta (Target: 15 Feb 2027) — Extension SDK & Living Plan**:
  - Out-of-process `prumo-extension-sdk` with Language Server Protocol (LSP) and Debug Adapter Protocol (DAP) conformance.
  - Conversational Living Plan interview engine (`prumo plan interview`, Phase M6).

- **v1.0.0-rc1 / rc2 (Target: 01 Mar 2027) — System Daemon & Traceability**:
  - System user daemon (ADR 018), credential vault, and end-to-end Traceability Graph (Phase M8).
  - Zero-warning Docs Gauntlet verification (`prumo docs verify --strict`).

- **v1.0.0 (Target: 15 Mar 2027) — General Availability (GA) 🚀**:
  - Signed universal binaries for Linux (amd64, arm64), macOS (Apple Silicon, Intel), and Windows (x86_64).
  - Official distribution channels: Homebrew tap, WinGet, Scoop, and Arch AUR.

---

## 4. Comprehensive Implementation Roadmap Table

| Version | Target Date | Primary Focus | Technology Stack | Key Deliverables & Milestones | DoD & Exit Gate |
|---|---|---|---|---|---|
| **v0.1.0** | 2026-08-10 | Foundation Prototype | Python 3.11+ | POP, LPC concept, Workforce Catalog (189 skills, 39 agents, 20 recipes), Goal v1 | Prototype goals and workforce resolution working in Python |
| **v0.2.0** | 2026-08-15 | Conformance Baseline | Python 3.11+ | Machine JSON envelopes, schema validation (Draft 2020-12), approval protocol baseline | Conformance test suite with golden fixtures |
| **v0.3.0** | 2026-08-20 | Migration Freeze (M0) | Python 3.11+ | Protocol inventory, contract freeze, model discovery (`models` op), ADR 001/002 | 127/127 Python tests green; Go migration blueprint approved |
| **v0.4.0** | 2026-09-09 | Go Core Migration | Go 1.22+ | M1–M4 complete, single binary CLI, 33 language profiles, push `subscribe`, on-demand `diff` | 100% contract parity Go vs Python fixtures |
| **v0.5.0** | 2026-09-10 | Doc Control Plane | Go 1.22+ | Rebrand to Prumo, Waves W0–W21, Semantic Doc Plane (W15), Docs Gauntlet | `prumo docs readiness` reports semantic verified |
| **v0.6.0** | 2026-09-27 | CLI Unification (Current) | Go 1.26 | ADR 016 (`prumo`), Adoption v3 (`prumo adopt`), Connectors, Dynamic Models | Single binary harness; 0 hardcoded models; CI green |
| **v0.6.1** | 2026-09-28 | CLI Ergonomics | Go 1.26 | Stack autodetect, zero-arg `prumo init` cognitive dashboard, autoupdate fix | Zero-ceremony init and dashboard verified |
| **v0.6.2** | 2026-10-05 | Onda 0: Real Exit Codes | Go 1.26 | Honest process exit codes (GAP-097), honest session resume (GAP-123) | Broken tests exit != 0; resume does not mark complete |
| **v0.6.3** | 2026-10-10 | Onda 1: Security Bounds | Go 1.26 | Symlink resolution (GAP-110), provider allowlist (GAP-111), safepath (GAP-112) | Sandbox escape tests fail closed |
| **v0.7.0** | 2026-10-15 | Complete TUI Client | Go, Bubble Tea v2 | `prumo-tui` standalone, multi-session, files diff dialog (`ctrl+g`), full A11y (WCAG 4.5:1) | 22 UI states verified; screen reader attestation |
| **v0.7.1** | 2026-10-25 | TUI Ergonomics | Go, Bubble Tea v2 | Collapsible `<think>` accordions (`Ctrl+O`), paginated tools, cache hit % gauge | Frame snapshot tests green |
| **v0.7.2** | 2026-11-05 | Git & Web in TUI | Go, Git CLI | Atomic `/undo`, dirty tree protection, Conventional auto-commits, `/web <url>` | `/undo` restores previous commit cleanly |
| **v0.7.3** | 2026-11-15 | Tree-sitter Repo Map | Go, Tree-sitter | Tree-sitter symbol graph + Personalized PageRank within LPC budget | 40% navigation precision boost under 2048 tokens |
| **v0.8.0** | 2026-11-25 | Resilient Gateway | Go 1.26 | Gateway active in production (GAP-102/103/108), budget caps (GAP-098/100), atomic lock (GAP-104) | Circuits auto-recover; budget limits enforced |
| **v0.8.1** | 2026-12-05 | Provider Failover | Go 1.26 | Dynamic priority routing (Local/Free → Cloud) with seamless failover | Provider failover with zero session drop |
| **v0.8.2** | 2026-12-15 | Local Intel & Offline | Go, llama/vLLM | Connectors for Ollama, vLLM, llama-server with search/replace diff fallback | End-to-end execution offline |
| **v0.9.0** | 2026-12-22 | Multi-Agent Teams | Go 1.26, Worktrees | Team execution in prod (GAP-003/101), 1-level subagents (ADR 017), worktrees (GAP-008/168) | Distributed DAG execution with 3-way merge |
| **v0.9.1** | 2027-01-10 | Dual Architect/Editor | Go 1.26 | Architect mode (`/architect`, read-only) + Editor mode (`/code`, write tools) | >50% token cost reduction on refactoring |
| **v0.9.2** | 2027-01-20 | Verification Loop | Go 1.26 | Automated test-and-lint error feedback with bounded $K$-retry auto-repair | Autonomous repair of induced syntax bugs |
| **v1.0.0-alpha**| 2027-01-30 | Freya Desktop IDE | Rust 2024 (Freya 0.4+) | Prumo Native Workspace Viewer (`prumo-viewer`), Skia, Torin, Tree-sitter, PTY | Sub-100ms startup; real-time agent observability |
| **v1.0.0-beta** | 2027-02-15 | Extension SDK & Plan | Rust, Go | `prumo-extension-sdk` (LSP/DAP), Living Plan interview engine (Phase M6) | Extension conformance suite 100% green |
| **v1.0.0-rc1/2**| 2027-03-01 | Daemon & Traceability | Go, Rust | Global user daemon (ADR 018), Traceability Graph (Phase M8), Docs Gauntlet strict | Strict Gauntlet pass with zero warnings |
| **v1.0.0** | 2027-03-15 | General Availability | Multi-platform | Signed universal binaries, Homebrew, WinGet, Scoop, Arch AUR packages | Official enterprise distribution live |

---

## 5. Technical Gap & Open Work Traceability

Every upcoming milestone directly resolves open items from the canonical Harness Gap Register (`docs/harness/gap-register.md`):

```
v0.7.0 (TUI)             v0.8.0 (Gateway & Resilience)      v0.9.0 (Multi-Agent)           v1.0.0 (Native IDE & Living Plan)
├── GAP-052 (Fold cursor)  ├── GAP-097 (Real exit codes)      ├── GAP-003 (Team binding)     ├── CONST-82 (Freya Desktop IDE)
├── GAP-062 (Focus glyph)  ├── GAP-098 (Daemon budget cap)    ├── GAP-022 (Subagents)        ├── Phase M6 (Living Plan interview)
├── GAP-065 (Motion mode)  ├── GAP-100 (Preflight budget)     ├── GAP-101 (Team in prod)     ├── Phase M8 (Traceability graph)
├── GAP-067 (Timeline exp) ├── GAP-102 (Gateway wire-up)      ├── GAP-109 (Depth bounds)     ├── Phase M10/M11 (Connector SDK)
├── GAP-070 (Contrast 4.5) ├── GAP-103 (Circuit recovery)     ├── GAP-135 (Handoff proto)    ├── ADR 018 (Global system daemon)
├── GAP-074 (Screen reader)├── GAP-104 (Atomic PID lock)      ├── GAP-146 (Workforce route)  └── ADR 021 (Viewer extension SDK)
└── GAP-082 (User commands)├── GAP-105 (Drain shutdown)      └── GAP-168 (Worktree merge)
                           ├── GAP-106 (Approval replay)
                           ├── GAP-108 (Backoff/retry-after)
                           ├── GAP-110 (Symlink sandbox)
                           ├── GAP-111 (Provider security)
                           ├── GAP-112 (Safepath traversal)
                           └── GAP-123 (Resume honesty)
```

---

## 6. Governance & Review Cadence

- **Source of Truth**: This document represents canonical authority on framework release planning alongside `docs/governance/authority.md`.
- **Review Cycle**: Updated upon each minor version tag or when architecture decisions (ADRs) introduce or alter roadmap invariants.
- **Drift Control**: Governed by the continuous documentation validation gate (`prumo docs authority` and `prumo docs verify --strict`).
