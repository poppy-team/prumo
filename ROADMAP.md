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
 Complete TUI              Resilient Gateway         Multi-Agent DAGs          Production Release
  (prumo-tui)              & Quota Router            & Delegation             & Freya Native IDE
```

---

### v0.7.0 — Production Terminal User Interface (`prumo-tui`)
*Target: October 2026*  
*Objective: Deliver a complete, beautiful, keyboard-first, self-contained terminal experience for interactive agent pairing.*

- **Surface**: `prumo-tui` (built from `prumo-tui/cmd/prumo-tui`, launched transparently via `prumo agent` or `prumo tui`).
- **Core Features**:
  1. **Unified TUI Supervision**: `prumo agent` / `prumo tui` automatically checks local socket availability, spawning or connecting to the headless daemon transparently without double-daemon collisions (GAP-086).
  2. **Multi-Session Management & Conversation Folding**: Full persistent session browsing, resumption, and timeline compaction with constant-memory rendering of message tails (GAP-052, GAP-080).
  3. **Interactive Touched Files & Diff Viewer**: `ctrl+g` / files dialog with syntax highlighting and side-by-side or unified diff viewing powered by the `diff` protocol operation (GAP-084).
  4. **Dynamic User Command Palette**: User-defined prompt library (`$XDG_CONFIG_HOME/prumo-tui/commands/*.md`) with frontmatter metadata and interactive placeholder prompts (`{{arg}}`) (GAP-082).
  5. **Model Capabilities & Badges**: Dynamic inspection badges (`[text reasoning vision tools audio]`) displaying capabilities declared in `.prumo/models.json` (GAP-089).
  6. **A11y & Cognitive Clarity Compliance**:
     - 9 verified built-in themes compliant with WCAG 4.5:1 text-to-background contrast ratio (GAP-070).
     - Reduced motion mode (`--reduced-motion`) disabling caret blinking and spinners without losing status context (GAP-065).
     - Glyph-based focus indicators (`━` vs `─`) ensuring clear focus visibility even in non-color terminals (GAP-062).
     - Headless screen-reader linear output mode (`prumo tui --prompt "..." --plain` / `--json`) (GAP-074).
  7. **Timeline Plaintext Export**: Export complete session logs to `.prumo/runtime/exports/<run>.txt` with deterministic key ordering and separated token/cache cost metrics (GAP-067, GAP-087).

---

### v0.8.0 — Resilient Gateway & Quota-Aware Multi-Provider Router
*Target: November 2026*  
*Objective: Solidify process safety, honest execution reporting, and intelligent multi-model routing in production.*

- **Truthful Execution & Exit Codes (Onda 0)**:
  - **GAP-097**: Fix runner tool execution to propagate real process exit codes; ensure broken tests fail closed instead of reporting exit code 0.
  - **GAP-123**: Ensure session `resume` operations resume work honestly rather than prematurely marking runs as completed.
- **Security & Sandboxing Boundaries (Onda 1)**:
  - **GAP-110**: Symlink sandbox containment — enforce canonical path resolution before file operations to prevent symlink traversal outside project boundaries.
  - **GAP-111**: Secure provider endpoint routing — validate custom provider URLs against allowlists to prevent credential and API key exfiltration to untrusted endpoints.
  - **GAP-112**: Path traversal defense in runtime state — sanitize and validate `run_id` parameters using `safepath` to prevent directory traversal.
- **Intelligent Gateway in Production (Onda 4)**:
  - **GAP-102**: Connect `internal/harness/gateway` directly into `r.Svc.Models.Stream` in production runtime (ADR 020).
  - **GAP-103**: Circuit breaker cooldown fix — ensure open circuits recover automatically once `CooldownUntil` timestamps expire.
  - **GAP-108**: Exponential backoff with jitter and standard HTTP `Retry-After` header adherence for rate-limited providers (429 / `RESOURCE_EXHAUSTED`).
- **Budgeting & Daemon Lifecycle**:
  - **GAP-098 & GAP-100**: Enforce daemon-level hard budget caps and preflight token/cost validation before initiating LLM requests.
  - **GAP-104 & GAP-105**: Atomic daemon PID file locking with `O_EXCL`/`flock` and graceful drain shutdown on SIGTERM.
  - **GAP-106 & GAP-107**: Persistent human approval decisions reloaded by action/resource fingerprint across daemon restarts.

---

### v0.9.0 — Multi-Agent DAG Orchestration & Subagent Delegation
*Target: December 2026*  
*Objective: Enable scalable, autonomous multi-agent task execution with bounded hierarchy and worktree isolation.*

- **Production Team Execution**:
  - **GAP-003 & GAP-101**: Activate `team.Runner` / `RunWork` in the production CLI and daemon, allowing roles to execute nested runs with distinct budgets and checkpoints.
- **Bounded 1-Level Subagent Delegation**:
  - **ADR 017, GAP-022, GAP-109**: Expose `agent.delegate` and `agent.ask` tools to agents with strict depth enforcement (`max_delegation_depth = 1`). Prevent runaway recursive delegation while enabling specialist task handoff.
- **Worktree Isolation & Safe Merge**:
  - **GAP-008 & GAP-168**: Orchestrate isolated Git worktrees per subagent; provide automated three-way merge verification with explicit conflict surfacing before touching the main worktree.
- **Dynamic Cross-Provider Handoff Protocol (M9 / GAP-135)**:
  - Structured state transfer between agents running on different provider backends (e.g. Claude Code ↔ Codex ↔ Local Models) using typed session summaries.
- **Task-Aware Workforce Resolution**:
  - **GAP-146 & DOC-GAP-024**: Dynamic resolution of workforce skills and recipes based on the semantic impact analysis of the target Goal.

---

### v1.0.0 — Production Release: Prumo Native Workspace Viewer (Rust IDE GUI) & Enterprise Ecosystem
*Target: Q1 2027*  
*Objective: Full production readiness across all surfaces: Go Core Harness, Bubble Tea TUI, and Freya Native Desktop IDE.*

- **Prumo IDE / Native Workspace Viewer (`prumo-native` / `prumo-viewer`)**:
  - **Native GUI Stack**: Built from scratch in Rust using **Freya 0.4+** (declarative Rust GUI, Skia GPU/CPU rasterization, Torin layout engine).
  - **Agent-Aware Architecture (CONST-82)**: Designed as an ultra-fast, lightweight workspace viewer and editor to observe agent execution in real-time, inspect file trees, review diffs, and perform targeted human edits without opening external tools.
  - **Code Editor & Tree-sitter**: Integrated syntax highlighting and navigation powered by `freya-code-editor` and Tree-sitter parsers (Rust, Go, JSON, YAML, Markdown, JS, TS, TOML).
  - **Integrated Diff Viewer**: Inline and side-by-side diff visualization powered by the `similar` engine.
  - **Embedded PTY Terminal**: Full terminal emulation powered by `portable-pty` and `vt100`.
  - **File System Watching**: High-efficiency, debounced file system monitoring via `notify`.
  - **Native Extension SDK (`prumo-extension-sdk`)**: Out-of-process extension host supporting Language Server Protocol (LSP) and Debug Adapter Protocol (DAP) with cryptographic manifest verification and permission grants.
- **Living Plan System (Phase M6)**:
  - Interactive conversational interview engine (`prumo plan interview`) guiding users from initial intent to implementation-ready specs without manual megaprompts (`docs/runtime/living-plan.md`).
  - Incremental Documentation Delta engine automatically updating project specifications as architectural decisions are made.
- **Global Daemon & Multi-Project Ecosystem (Phase M10, M11, ADR 018)**:
  - System user daemon (`systemd` / macOS `launchd` / Windows Service) managing shared model connections, credential vault, and per-project isolated executors.
  - Connector SDK kit for building third-party IDE and editor integrations.
- **Typed End-to-End Traceability Graph (Phase M8)**:
  - Complete queryable traceability connecting Requirements ↔ Goals ↔ Decisions ↔ Code ↔ Tests ↔ Evidence (`prumo trace <ref>`).
- **Enterprise Distribution & Packaging**:
  - Signed universal binaries for Linux (amd64, arm64), macOS (Apple Silicon, Intel), and Windows (x86_64).
  - Official distribution channels: Homebrew tap, WinGet, Scoop, and Arch AUR.

---

## 4. Comprehensive Implementation Roadmap Table

| Version | Target Date | Primary Focus | Technology Stack | Key Deliverables & Milestones | DoD & Exit Gate |
|---|---|---|---|---|---|
| **v0.1.0** | 2026-08-10 | Foundation Prototype | Python 3.11+ | POP, LPC concept, Workforce Catalog (189 skills, 39 agents, 20 recipes), Goal v1 | Prototype goals and workforce resolution working in Python |
| **v0.2.0** | 2026-08-15 | Conformance Baseline | Python 3.11+ | Machine JSON envelopes, schema validation (Draft 2020-12), approval protocol baseline | Conformance test suite with golden fixtures |
| **v0.3.0** | 2026-08-20 | Migration Freeze (M0) | Python 3.11+ | Protocol inventory, contract freeze, model discovery (`models` op), ADR 001/002 | 127/127 Python tests green; Go migration blueprint approved |
| **v0.4.0** | 2026-09-09 | Go Core Migration | Go 1.22+ | M1–M4 complete, single binary CLI, 33 language profiles, push `subscribe`, on-demand `diff`, H10 TUI spike | 100% contract parity Go vs Python fixtures |
| **v0.5.0** | 2026-09-10 | Doc Control Plane | Go 1.22+ | Rebrand to Prumo, Waves W0–W21, Semantic Doc Plane (W15), Docs Gauntlet, ACP/MCP bridges | `prumo docs readiness` reports semantic verified |
| **v0.6.0** | 2026-09-27 | CLI Unification (Current) | Go 1.26 | ADR 016 (`prumo`), Adoption v3 (`prumo adopt`), Connectors (`prumo connector`), Dynamic Model Roster, Images (GAP-090), Opencode (GAP-093) | Single binary harness; 0 hardcoded model lists; CI 100% green |
| **v0.7.0** | 2026-10-15 | Complete TUI Client | Go, Bubble Tea v2, Lipgloss v2 | Self-contained `prumo-tui`, multi-session folding, command palette (`commands/*.md`), files diff dialog (`ctrl+g`), full A11y (WCAG 4.5:1 contrast, reduced motion, screen reader) | End-to-end TUI pairing verified across all 22 UI states; manual screen reader attestation |
| **v0.8.0** | 2026-11-15 | Resilient Gateway & Quota Router | Go 1.26 | Honest exit codes (GAP-097), sandboxing containment (GAP-110–112), production Gateway (GAP-102/103/108), daemon budget caps (GAP-098/100), atomic PID lock (GAP-104) | Failed tests exit != 0; circuits recover; zero path escapes in sandbox audit |
| **v0.9.0** | 2026-12-15 | Multi-Agent DAGs & Delegation | Go 1.26, Git Worktrees | Multi-agent teams in production (GAP-003/101), 1-level delegation (ADR 017, GAP-022/109), cross-provider handoff (M9/GAP-135), isolated worktree merges (GAP-008/168) | Multi-agent DAG execution passes with verified worktree isolation and bounded depth |
| **v1.0.0** | 2027-Q1 | Flagship Production Release | Rust 2024 (Freya 0.4+, Skia, Torin), Go Core | **Prumo Native Workspace Viewer / IDE** (`prumo-viewer`), Tree-sitter, Extension SDK (LSP/DAP), Living Plan interview engine (M6), Global Daemon (ADR 018), Traceability Graph (M8) | Native IDE compiles and runs on Linux/macOS/Windows; Extension SDK conformance suite passes; Enterprise packages live |

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
