---
title: Official Roadmap (v0.1.0 → v1.0.0)
description: Historical milestones and evolutionary projection of the Prumo Framework towards version 1.0.
---

# Official Prumo Roadmap (v0.1.0 → v1.0.0)

> **Canonical Document**: See also [`ROADMAP.md`](https://github.com/poppy-team/prumo/blob/main/ROADMAP.md) at the repository root for the full governance log and technical gap traceability.

---

## Executive Vision & Core Invariants

**Prumo** is an open-source engineering protocol and autonomous harness designed for paired human-agent and multi-agent software engineering. The project is governed by four core architectural invariants:

1. **Protocol-Oriented Programming (POP)**: System interactions occur via versioned, typed contracts with machine-verifiable JSON envelopes (`protocol_version`, `ok`, `data`, `diagnostics`, `warnings`).
2. **Lean Progressive Context (LPC)**: Token budgets and context windows are strictly managed via hierarchical disclosure tiers (L0 metadata → L1 summaries → L2 full texts → L3 AST symbols → L4 deep context).
3. **Provider Neutrality**: No specific LLM vendor or orchestrator is hardcoded or mandatory. Model selection and quotas are dynamically resolved from the workspace or active runtime environment.
4. **Clear Surface Boundaries**:
   - **Core Harness & CLI (`prumo`)**: Pure Go single binary providing protocol authority, execution runtime, sandboxed tool gateway (ACI), and headless automation.
   - **Terminal User Interface (`prumo-tui`)**: Dedicated, keyboard-first terminal client built with Go and Bubble Tea v2, scheduled for full production delivery in **v0.7.0**.
   - **Desktop Workspace Viewer & IDE (`prumo-viewer` / `prumo-native`)**: Native, agent-aware desktop GUI built from scratch in Rust with **Freya 0.4+** (Skia rasterization, Torin layout engine, Tree-sitter syntax highlighting, embedded PTY, and out-of-process Extension SDK), scheduled for production release in **v1.0.0**.

---

## Historical Milestones (v0.1.0 — v0.6.0)

```
2026-08-10      2026-08-15      2026-08-20      2026-09-09      2026-09-10      2026-09-27
   v0.1.0          v0.2.0          v0.3.0          v0.4.0          v0.5.0          v0.6.0 (Current)
     │               │               │               │               │               │
     ▼               ▼               ▼               ▼               ▼               ▼
  Python          Protocol        Python          Go Core         Brand &         Unified CLI,
  Genesis        Conformance      Freeze         Migration       Doc Control     Adoption v3 &
 Prototype       & Approvals     (ADR 001)       (M1 - M4)       Plane (W0-W21)  Dynamic Roster
```

- **v0.1.0 (August 2026)**: Initial formalization of POP and LPC; first AI Workforce Catalog (`catalog.json`) with 39 agents, 189 skills, and 20 recipes; initial Goal lifecycle with SHA-256 state hashing.
- **v0.2.0 (August 2026)**: Standardized machine envelope contracts; JSON Schema validation (Draft 2020-12); human-in-the-loop approval protocol baseline; golden regression fixtures.
- **v0.3.0 (August 2026)**: Comprehensive protocol inventory (`docs/migration/protocol-inventory.md`); freeze of Python codebase with 127 tests; model discovery operation (`models` op); ADR 001/002 accepted for Go Core migration.
- **v0.4.0 (September 2026)**: Execution of migration phases M1 to M4 with 100% parity; 33 language engineering profiles; real-time push streaming (`subscribe` op, ADR 014); on-demand syntax diffs (`diff` op); H10 TUI spike.
- **v0.5.0 (September 2026)**: Rebranded to Prumo (ADR 004); Pre-TUI Waves W0 to W21; Semantic Documentation Control Plane (W15); Docs Gauntlet runtime; ACP v1 and MCP bridges.
- **v0.6.0 / v0.6.1 (Current)**: CLI & Harness unification in single `prumo` binary (ADR 016); Brownfield Adoption Engine v3 (`prumo adopt`); connector management (`prumo connector status/install`); dynamic model resolution; multimodal image resolution (GAP-090); delegated opencode execution (GAP-093).

---

## Projection Roadmap (v0.7.0 — v1.0.0)

```
2026-10 (Target)          2026-11 (Target)          2026-12 (Target)          2027-Q1 (Target)
     v0.7.0                    v0.8.0                    v0.9.0                    v1.0.0
       │                         │                         │                         │
       ▼                         ▼                         ▼                         ▼
 Complete TUI              Resilient Gateway         Multi-Agent DAGs          Production Release
  (prumo-tui)              & Quota Router            & Delegation             & Freya Native IDE
```

### v0.7.0 — Production Terminal User Interface (`prumo-tui`)
*Target: October 2026*

- **Transparent TUI Supervision**: Official `prumo agent` / `prumo tui` launcher automatically connects to existing daemons or spawns managed instances without port/socket collisions (GAP-086).
- **Leader Key System (`Ctrl+X`)**: OpenCode/tmux mnemonic sub-chords (`Ctrl+X N` new session, `Ctrl+X L` list, `Ctrl+X U` undo, `Ctrl+X M` model) preventing collisions with standard GNU Readline shortcuts.
- **Fast Input Prefixes**: Fuzzy file completion (`@`), immediate shell pass-through without invoking LLM (`!command`), and slash commands (`/command`).
- **Operating Trinity**: Unified behavior across full interactive TUI (`prumo tui`), quiet scriptable CLI (`prumo agent "<prompt>" -q` / `prumo run`), and headless daemon (`prumo daemon`).
- **Multi-Session Management & Conversation Folding**: Persistent session switching and timeline fold resume with constant-memory tail rendering (GAP-052, GAP-080).
- **Touched Files & Diff Viewer**: Interactive files dialog (`ctrl+g`) with colorized syntax diffs via the `diff` protocol op (GAP-084).
- **Dynamic User Command Palette**: User-defined Markdown commands (`commands/*.md`) with argument placeholders `{{arg}}` (GAP-082).
- **Model Capabilities & Badges**: Dynamic badges (`[text reasoning vision tools audio]`) reflecting declared model capabilities (GAP-089).
- **Comprehensive Accessibility (WCAG)**: 9 verified themes with 4.5:1 contrast floor (GAP-070), reduced motion mode (GAP-065), glyph-based focus indicators (GAP-062), and headless linear screen-reader outputs (`--prompt`, `--plain`, `--json`) (GAP-074).
- **Timeline Plaintext Export**: Export complete session logs to `.prumo/runtime/exports/<run>.txt` with separated token, cache, and cost metrics (GAP-067, GAP-087).

---

### v0.8.0 — Resilient Gateway & Quota-Aware Multi-Provider Router
*Target: November 2026*

- **Truthful Execution & Honest Exit Codes (Onda 0)**:
  - **GAP-097**: Propagate actual subprocess exit codes; ensure broken tests fail closed instead of reporting code 0.
  - **GAP-123**: Session `resume` honestly resumes work instead of prematurely declaring tasks complete.
- **Security & Sandboxing Boundaries (Onda 1)**:
  - **GAP-110**: Symlink sandbox containment preventing escape from workspace root.
  - **GAP-111**: Provider endpoint verification against allowlists protecting API keys from external exfiltration.
  - **GAP-112**: Path traversal protection for execution IDs via `safepath`.
- **Production Intelligent Gateway (Onda 4)**:
  - **GAP-102**: Connect `internal/harness/gateway` directly into the agent runtime execution loop (ADR 020).
  - **GAP-103**: Automatic circuit breaker cooldown recovery upon expiry of `CooldownUntil`.
  - **GAP-108**: Exponential backoff with jitter and standard HTTP `Retry-After` adherence for rate-limited requests (429).
- **Budgeting & Daemon Lifecycle**:
  - **GAP-098 & GAP-100**: Hard daemon budget ceilings and preflight cost validation before initiating LLM calls.
  - **GAP-104 & GAP-105**: Atomic PID locking with `O_EXCL`/`flock` and graceful drain shutdown on SIGTERM.
  - **GAP-106 & GAP-107**: Reload persistent approval decisions by action/resource fingerprint across restarts.

---

### v0.9.0 — Multi-Agent DAG Orchestration & Subagent Delegation
*Target: December 2026*

- **Production Team Execution**:
  - **GAP-003 & GAP-101**: Activate `team.Runner` / `RunWork` in production CLI and daemon, isolating budgets and checkpoints per role.
- **Bounded 1-Level Subagent Delegation**:
  - **ADR 017, GAP-022, GAP-109**: Expose `agent.delegate` and `agent.ask` tools with a strict 1-level depth cap to prevent runaway recursive cost loops.
- **Isolated Git Worktrees**:
  - **GAP-008 & GAP-168**: Ephemeral isolated worktrees per subagent with automated three-way merge verification before merging to main.
- **Dynamic Cross-Provider Handoff Protocol**:
  - **Phase M9 / GAP-135**: Structured state transfer and session summaries across provider backends (Claude Code ↔ Codex ↔ Local Models).
- **Task-Aware Workforce Resolution**:
  - **GAP-146 & DOC-GAP-024**: Dynamic resolution of skills and recipes based on Goal semantic impact analysis.

---

### v1.0.0 — Production Release: Prumo Native Workspace Viewer (Rust IDE GUI) & Enterprise Ecosystem
*Target: Q1 2027*

- **Prumo IDE / Native Workspace Viewer (`prumo-native` / `prumo-viewer`)**:
  - **Native Rust GUI Stack**: Built from scratch using **Freya 0.4+** (declarative Rust GUI, Skia GPU/CPU rasterization, Torin layout engine).
  - **Agent-Aware Architecture (CONST-82)**: Ultra-fast, lightweight workspace viewer and editor to observe agent execution in real-time, inspect file trees, review diffs, and perform targeted human edits without heavy Electron overhead.
  - **Tree-sitter Code Editor**: Syntax highlighting and AST inspection for Rust, Go, JSON, YAML, Markdown, JS, TS, and TOML.
  - **Diff Viewer & Embedded PTY**: Word-level diffing via `similar`; embedded terminal via `portable-pty` and `vt100`.
  - **Native Extension SDK (`prumo-extension-sdk`)**: Out-of-process extension bus supporting Language Server Protocol (LSP) and Debug Adapter Protocol (DAP).
- **Living Plan System (Phase M6)**:
  - Interactive conversational interview engine (`prumo plan interview`) taking projects from intent to implementation-ready specs.
  - Incremental Documentation Delta engine automatically updating specifications as architectural decisions are recorded.
- **Global Daemon & Connector Ecosystem (Phase M10/M11, ADR 018)**:
  - System user daemon managing model routing, credential vault, and per-project isolated executors.
- **Typed End-to-End Traceability Graph (Phase M8)**:
  - Full bidirectional graph connecting Requirements ↔ Goals ↔ Decisions ↔ Code ↔ Tests ↔ Evidence (`prumo trace <ref>`).
- **Enterprise Distribution**:
  - Signed static universal binaries for Linux, macOS, and Windows.
  - Official packages for Homebrew, WinGet, Scoop, and Arch AUR.

---

> **Detailed Granular Plan**: For the complete patch-by-patch technical specification, file-level deliverables, and competitive analysis against Claude Code, OpenCode, Aider, and Cline, see the [**Granular Patch Implementation Plan**](/en/granular-plan).

## Implementation Comparison Matrix

| Version | Target Date | Primary Focus | Technology Stack | Key Deliverables | Exit Gate / DoD |
|---|---|---|---|---|---|
| **v0.1.0** | 2026-08-10 | Conceptual Prototype | Python 3.11+ | POP, LPC, Workforce Catalog (189 skills, 39 agents, 20 recipes), Goal v1 | Prototype goals and workforce working in Python |
| **v0.2.0** | 2026-08-15 | Conformance Baseline | Python 3.11+ | Standard machine envelopes, JSON Schema Draft 2020-12, approval baseline | Golden test fixtures passing |
| **v0.3.0** | 2026-08-20 | Migration Freeze (M0) | Python 3.11+ | Protocol inventory, model discovery (`models`), Go migration decision (ADR 001) | 127 Python tests green; Go migration blueprint approved |
| **v0.4.0** | 2026-09-09 | Go Core Migration | Go 1.22+ | M1–M4 complete, single binary CLI, push `subscribe`, on-demand `diff` | 100% contract parity Go vs Python fixtures |
| **v0.5.0** | 2026-09-10 | Doc Control Plane | Go 1.22+ | Rebrand to Prumo, Waves W0–W21, Semantic Doc Plane (W15), Docs Gauntlet | Semantic validator passes with verified report |
| **v0.6.0** | 2026-09-27 | CLI Unification (Current) | Go 1.26 | Unified binary `prumo` (ADR 016), Adoption v3 (`prumo adopt`), Connectors, Dynamic Models | Single binary harness; 0 hardcoded models; CI green |
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
