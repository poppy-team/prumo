---
title: Granular Patch Implementation Plan (v0.6.0 → v1.0.0)
description: Detailed intermediate patch versions roadmap and competitive benchmark of leading CLI coding agents.
---

# Granular Patch Implementation Plan (v0.6.0 → v1.0.0)

> **Canonical Document**: See also [`ROADMAP.md`](https://github.com/poppy-team/prumo/blob/main/ROADMAP.md) and [`docs/development/granular-implementation-plan.md`](https://github.com/poppy-team/prumo/blob/main/docs/development/granular-implementation-plan.md).

---

## 1. Competitive Benchmark & Research on Leading CLI Code Agents

To ensure that **Prumo TUI (`prumo-tui`)** and the **Harness Core (`prumo`)** deliver a world-class developer experience, we conducted an in-depth benchmark across prominent terminal coding agents:

1. **Claude Code (Anthropic)**: Gold standard in terminal REPL ergonomics, progressive micro-compaction at 70–75% of context window, real-time USD/token spend telemetry ($/req, cache tokens), collapsible reasoning accordions, and safe turn yield.
2. **OpenCode & Crush (Anomaly / Charm Go lineage)**: Reference in decoupled client-server architecture (IPC via socket/HTTP), deterministic tool guards, streaming via Charm v2 (Bubble Tea v2 / Lip Gloss v2), and clean session management.
3. **Aider (`aider-chat`)**: Pioneer in Git-native developer pairing: repository mapping via Tree-sitter AST + Personalized PageRank, atomic auto-commits with conventional messages, the `/undo` safety net, dual-agent split (Architect vs. Editor), and automated lint/test loops.
4. **Cline & Roo Code (with Freebuff)**: Leader in specialized persona modes (`.roomodes`: Architect, Code, Ask, Debug) with strict per-mode tool whitelists, extensive Model Context Protocol (MCP) tool support, and pre-execution guards.
5. **OpenAI Codex / Operator CLI & Command-Code**: Terminal patterns with background execution, native diff reviewers (`/review`), sandbox isolation, and message history buffers.
6. **Antigravity CLI / Gemini CLI**: Unified agentic terminal harness with modular Agent Skills, specialized subagents, and enterprise MCP tool gating.

---

### Comparative Feature Matrix

| Dimension | Claude Code | OpenCode / Crush | Aider | Cline / Roo Code | **Prumo TUI (`prumo-tui`)** |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **Language / Runtime** | Node.js (Ink TUI) | TS / Go (Charm v2) | Python (`prompt_toolkit`) | TypeScript (VS Code / CLI) | **Native Go (Charm v2, static zero-dep binary)** |
| **Process Architecture** | Monolithic interactive REPL | Client / Server IPC | Monolithic interactive CLI | Node extension / CLI | **Decoupled TUI over Daemon `agentd` (ADR 013/018)** |
| **Provider Independence** | Proprietary (Anthropic) | Provider-agnostic | 100% Agnostic (LiteLLM) | Provider-agnostic | **100% Agnostic (Dynamic Roster, zero hardcoded lists)** |
| **Repository Mapping** | `CLAUDE.md` + file search | Context hooks + search | **Tree-sitter AST + PageRank** | Embeddings / Semantic search | **LPC + Tree-sitter Repo Map + PageRank (v0.7.3)** |
| **Git Safety** | Terminal prompts | Internal transcript | **Auto-commit + `/undo` + dirty check** | Local file history | **Conventional auto-commit + atomic `/undo` (v0.7.2)** |
| **Agent Roles** | Single agent with sub-threads | Distinct subagents | **Dual: Architect + Editor** | **Multi-mode (`.roomodes`)** | **1-level Subagents in Git Worktrees (ADR 017, v0.9.0)** |
| **Verification Loop** | Manual bash execution | Session hooks | **Auto-lint + Auto-test repair loop** | Terminal commands | **Verification-Guided Repair Loop with visual cards (v0.9.2)** |
| **Reasoning (`<think>`)** | Collapsible accordion (`Ctrl+O`)| Formatted duration card | Transcribed in chat | Collapsible card | **Collapsible accordion with duration/tokens (v0.7.1)** |
| **Cost Telemetry** | Tokens in/out/cache + USD | Total tokens | Total tokens + cost | USD cost and token counts | **Real-time in/out/cache tokens, USD, ~$USD/req & Hit Ratio %** |
| **Diff Inspection** | Inline unified diff (`/diff`) | Files dialog | Search/replace diffs | Interactive inline diffs | **Unified & Side-by-Side Chroma highlight on demand (GAP-084)**|

---

## 2. Granular Patch Release Roadmap

```
v0.6.x (Hardening)      v0.7.x (Production TUI)       v0.8.x (Gateway & Resilience)      v0.9.x (Multi-Agent)           v1.0.x (Official Release & IDE)
├── v0.6.0 (Completed)  ├── v0.7.0 (Oct 2026)          ├── v0.8.0 (Nov 2026)              ├── v0.9.0 (Dec 2026)          ├── v1.0.0-alpha (Jan 2027)
├── v0.6.1 (Completed)  ├── v0.7.1 (TUI Ergonomics)    ├── v0.8.1 (Provider Failover)     ├── v0.9.1 (Dual Architect)    ├── v1.0.0-beta (Feb 2027)
├── v0.6.2 (Onda 0 Fix) ├── v0.7.2 (Git & Web in TUI)  └── v0.8.2 (Local Models)          └── v0.9.2 (Verification Loop)├── v1.0.0-rc1/rc2 (Mar 2027)
└── v0.6.3 (Onda 1 Fix) └── v0.7.3 (Tree-sitter Repo)                                                                    └── v1.0.0 GA (March 2027)
```

---

### v0.6.x Line — Core Hardening & Integrity

- **v0.6.0 (Completed ✅)**: Unified CLI `prumo` (ADR 016), Brownfield Adoption v3 (`prumo adopt`), Connector lifecycle, dynamic model resolution without hardcoded vendor rosters.
- **v0.6.1 (Completed ✅)**: CLI ergonomics overhaul, automated tech stack detection, zero-arg `prumo init` with interactive cognitive dashboard, and official documentation portal.
- **v0.6.2 (Target: Early Oct 2026) — Onda 0: Execution Fidelity**:
  - Propagate actual subprocess exit codes (GAP-097; broken tests fail closed).
  - Honest session resume preventing premature task completion (GAP-123).
- **v0.6.3 (Target: Mid Oct 2026) — Onda 1: Security Boundaries & Sandboxing**:
  - Strict canonical symlink resolution before mutations (GAP-110).
  - Provider endpoint verification against allowlists protecting credentials (GAP-111).
  - Path traversal protection on execution IDs via `safepath` (GAP-112).

---

### v0.7.x Line — Production Terminal Client (`prumo-tui`)

- **v0.7.0 (Target: 15 Oct 2026) — Canonical TUI Release**:
  - Dedicated self-contained `prumo-tui` binary launched transparently via `prumo agent` / `prumo tui`.
  - **Leader Key System (`Ctrl+X`)**: OpenCode-inspired mnemonic sub-chords (`Ctrl+X N` new session, `Ctrl+X L` list, `Ctrl+X U` undo, `Ctrl+X M` model) avoiding GNU Readline terminal collisions.
  - **Composer Input Prefixes**:
    - `@`: Fuzzy file and folder attachment autocomplete.
    - `!`: Immediate shell pass-through command execution without calling LLM (e.g., `!git status`).
    - `/`: Slash commands (`/help`, `/model`, `/provider`, `/sidebar`, `/init`).
  - **Unified Trinity of Operating Modes**: Full interactive TUI (`prumo tui`), quiet scriptable CLI one-shot (`prumo agent "<prompt>" -q` / `prumo run`), and headless daemon (`prumo daemon`).
  - Collision-free socket supervision (GAP-086).
  - Persistent multi-session switching with constant-memory timeline folding (GAP-052, GAP-080).
  - Touched files inspector (`Ctrl+G`) with syntax diffs (GAP-084).
  - User Markdown command palette (`commands/*.md`) with dynamic placeholders `{{arg}}` (GAP-082).
  - Model capabilities badges (`[text reasoning vision tools audio]`, GAP-089).
  - Full WCAG accessibility: 9 themes with 4.5:1 contrast floor (GAP-070), reduced motion mode (GAP-065), glyph focus indicators (GAP-062), and linear screen-reader mode (`--prompt`, `--plain`, `--json`, GAP-074).
  - Plaintext timeline export with separated token, cache, and cost metrics (GAP-067, GAP-087).
- **v0.7.1 (Target: 25 Oct 2026) — Advanced TUI Ergonomics & LSP**:
  - **Semantic LSP Tooling**: Native integration of `internal/harness/lsp` (`goToDefinition`, `findReferences`, and `getDiagnostics`), catching compiler diagnostics before running slow test suites.
  - Collapsible `<think>` reasoning accordions with duration timer and `Ctrl+O` toggle.
  - Paginated tool execution cards (collapse outputs > 10 lines with modal viewer).
  - Real-time Cache Hit Ratio % and Context Window gauge on statusline.
  - Optional Vim keybindings (`j`/`k` scroll in transcript).
  - External prompt drafting in `$EDITOR` (`Ctrl+E`).
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

### v0.8.x Line — Resilient Gateway & Quota Router

- **v0.8.0 (Target: 25 Nov 2026) — Gateway in Production**:
  - Connect `internal/harness/gateway` to production execution loop (ADR 020, GAP-102).
  - Circuit breaker cooldown auto-recovery (GAP-103).
  - Exponential backoff with jitter and `Retry-After` header respect on 429 errors (GAP-108).
  - Hard daemon budget ceilings and preflight validation (GAP-098, GAP-100).
  - Atomic daemon PID lock with `O_EXCL`/`flock` and graceful drain shutdown (GAP-104, GAP-105).
  - Persistent approval decision reloading by fingerprint across restarts (GAP-106, GAP-107).
- **v0.8.1 (Target: 05 Dec 2026) — Multi-Provider Failover**:
  - Priority routing (Local/Free → Cloud) with seamless failover upon quota exhaustion.
- **v0.8.2 (Target: 15 Dec 2026) — Local Intel & Offline Support**:
  - Connectors for `llama-server`, Ollama, and vLLM (GAP-025, GAP-040) with context calibration.

---

### v0.9.x Line — Multi-Agent DAGs & Delegation

- **v0.9.0 (Target: 22 Dec 2026) — Production Agent Teams & Git Worktrees**:
  - Activate `team.Runner` / `RunWork` in production CLI and daemon (GAP-003, GAP-101).
  - 1-level subagent delegation (`agent.delegate`, `agent.ask`) with fixed depth cap (ADR 017, GAP-022, GAP-109).
  - Ephemeral Git worktrees with 3-way merge verification (GAP-008, GAP-168).
  - Cross-provider handoff protocol with typed session summaries (Phase M9, GAP-135).
  - Task-aware workforce resolver based on Goal semantic impact (GAP-146, DOC-GAP-024).
- **v0.9.1 (Target: 10 Jan 2027) — Dual Architect / Editor Pattern**:
  - Dedicated `/architect` and `/code` modes toggleable via `Ctrl+M`.
- **v0.9.2 (Target: 20 Jan 2027) — Verification-Guided Repair Loop**:
  - Automated test-and-lint feedback injection with bounded $K$-retry auto-repair.

---

### v1.0.x Line — Official Flagship Release & Rust Native IDE

- **v1.0.0-alpha (Target: 30 Jan 2027) — Prumo Native Workspace Viewer / IDE**:
  - Desktop GUI built from scratch in Rust on **Freya 0.4+**, Skia, Torin, and Tree-sitter.
  - Embedded PTY terminal (`portable-pty` + `vt100`) and diff viewer (`similar`).
- **v1.0.0-beta (Target: 15 Feb 2027) — Extension SDK & Living Plan**:
  - Out-of-process `prumo-extension-sdk` with LSP and DAP conformance.
  - Conversational Living Plan interview engine (Phase M6).
- **v1.0.0-rc1 / rc2 (Target: 01 Mar 2027) — System Daemon & Traceability**:
  - System user daemon (ADR 018), credential vault, and end-to-end Traceability Graph (Phase M8).
  - Zero-warning Docs Gauntlet verification (`prumo docs verify --strict`).
- **v1.0.0 (Target: 15 Mar 2027) — General Availability (GA) 🚀**:
  - Official enterprise distribution via Homebrew, WinGet, Scoop, and AUR.
