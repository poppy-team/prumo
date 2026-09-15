# Prumo — Product Vision v0.4

## Purpose

Prumo evolves from a Git-native execution-ready protocol into a **complete project engineering layer** assisted by humans and agents. It integrates planning, deep living documentation, execution, governance, episodic and experiential memory, traceability, validation, handoff, and adapter portability across harnesses without ceding project authority to any single harness.

### Problem and Users

Software engineering teams, open-source maintainers, and autonomous coding agents suffer from fragmented documentation, context rot, lost architectural decisions between sessions, and ungrounded execution. Prumo serves engineers and agents who require deterministic project governance, verifiable implementation readiness, and durable memory that outlives transient chat sessions.

### Product Outcome

The primary product outcome is a self-governing, knowledge-aware engineering harness: an executable control plane where intent compiles deterministically into verified code, evidence, living documentation, and clean handoffs with zero vendor lock-in.

## Core Capabilities (v0.4)

| Capability | Description |
|------------|-------------|
| **Protocol Domain** | Goals, Plans, Tasks, Events, Evidence, Gates with deterministic lifecycle, schema validation, locks, amendments, and invariants. |
| **Project Service** | Root discovery, `prumo.json`, capabilities, profiles, repository metadata, state snapshots. |
| **Resolution Service** | Risk assessment, recipe selection, workforce resolution, model policy, context selection, documentation applicability. |
| **Documentation Engine** | Contracts, profiles, gap analysis, ownership, delta tracking, contradiction checking, coverage/readiness. |
| **Planning Engine** | Conversational interview, decision extraction, open questions, confidence scoring, authority model, readiness gates. |
| **Adoption Engine** | Repository scanner, semantic documentation mapping, capability detection, confidence ledger, migration proposals. |
| **Knowledge Engine** | Typed traceability graph (requirements ↔ decisions ↔ code ↔ tests ↔ docs ↔ evidence), retrieval, contradiction detection, history. |
| **Experience Engine** | Session events, summaries, handoffs, cross-session patterns, experience proposals with validation. |
| **Evidence/Gate Engine** | Deterministic validation, risk-based evidence requirements, release readiness, completion gates. |
| **Control Plane** | Run Engine, Budget/Cost Governance, Context Compiler, Model Router, Tool Gateway, Automation, Execution Environments, Observability. |
| **Integration Layer** | OpenCode (first native), Codex, Claude Code, Gemini CLI, Copilot CLI, Kiro, Generic fallback — all as thin adapters. |

## Non-Goals (v0.4)

- Distributed scheduler / central team server (deferred to v1.0+)
- Mandatory cloud dependencies or SaaS
- External graph database
- Public Go SDK (stabilize CLI/machine protocol first)
- Embeddings as hard requirement
- Full web dashboard
- Forking any harness

## Authority Hierarchy

1. **Canonical Repository** — schemas, ADRs, code, tests
2. **Approved Notion Design** — Living Book v0.4 pages
3. **Agent Inference** — bounded, evidence-backed

## Success Boundaries

Prumo succeeds when canonical project state is explicit, implementation readiness is explainable, changes are evidenced, and a fresh executor can continue work without private conversation history. Prumo does not own product business decisions, replace the project's source code, or grant harnesses authority over protocol invariants.

## Operational Target

A human opens any supported harness and asks *"implement custom shader support"*. Prumo resolves intent, context, Goal/Plan/Task, risk, recipe, workforce, applicable docs, implementation, tests, review, documentation delta, journal, evidence, and final state — without megaprompts or conversation memory.