# Core Concepts and Mental Model

To use Prumo as effectively as possible, humans and code agents must understand its **two-plane mental model**: the **Host Plane (Global Environment)** and the **Repository Plane (Local Workspace)**.

---

## The Two-Plane Model

```text
┌─────────────────────────────────────────────────────────────┐
│                 HOST PLANE (Global Environment)             │
│  Directory: ~/.prumo                                        │
│                                                             │
│  • prumo executable (/usr/local/bin/prumo)                  │
│  • Execution daemon and background services (prumo serve)   │
│  • Installed global connectors (~/.prumo/connectors/)       │
│  • Auto-update subsystem (prumo upgrade)                    │
│  • Global system audit (prumo doctor)                       │
└──────────────────────────────┬──────────────────────────────┘
                               │
            operates on and syncs rules with
                               │
                               ▼
┌─────────────────────────────────────────────────────────────┐
│            REPOSITORY PLANE (Local Workspace)               │
│  Project directory (./)                                     │
│                                                             │
│  • Canonical Configuration: prumo.json (Protocol v3)        │
│  • Canonical Documentation: docs/PRUMO.md, PROJECT_STATE.md │
│  • Active Workforce: .ai/ (agents, skills, recipes)         │
│  • Compiled Adapters: AGENTS.md, CLAUDE.md, .opencode       │
│  • Goals and Plans with SHA-256 Cryptographic Lock          │
└─────────────────────────────────────────────────────────────┘
```

---

## The 10 Canonical Principles of Prumo

1. **Repository over Conversation Memory**: The project's canonical truth lives exclusively in files in Git, not in an LLM's ephemeral chat window.
2. **Protocol over Harness**: Claude Code, OpenCode, Codex, Gemini, and Cursor are interchangeable clients; Prumo provides the common, neutral protocol.
3. **Canonical before Generated**: Markdown, JSON, and JSON Schema are sources of truth maintained by humans and authorized agents; adapter files (`AGENTS.md`, `.cursorrules`), caches, and indexes are derived and compiled.
4. **Goals as the Unit of Delivery**: Goals represent measurable engineering deliverables. Once they enter the `LOCKED` state, their acceptance criteria become immutable through a cryptographic digest.
5. **Lean Progressive Context (LPC)**: AI agents receive only the smallest context sufficient for the task at hand. Context expansion happens progressively and only on proven evidence.
6. **Evidence over Assertion**: A task is declared complete only if it presents verifiable evidence (tests executed, green linters, passing builds, completed reviews).
7. **Deterministic before Probabilistic**: Architectural invariants, syntax checks, and quality gates are evaluated by deterministic rules in Go code, never by the LLM's probabilistic intuition.
8. **Strict Incompatibility against Drift (Anti-Drift)**: Changes that break architectural contracts or JSON schemas fail immediately in `prumo validate`.
9. **Decoupling and Provider Independence**: The framework core does not mandate any LLM model or proprietary provider.
10. **Non-Destructive Adoption**: Existing projects keep their directories and source files intact throughout the entire lifecycle.
