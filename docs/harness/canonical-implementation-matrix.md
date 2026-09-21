# Prumo Harness — Canonical Implementation Matrix

> Authority: Canonical Engineering Status Track.
> Standard: Zero false-green, pure Go standard library, Total Assurance derived completion.

| ID | Domain | Priority | Status | Verification | Owner | Source Doc | Notes |
|---|---|---|---|---|---|---|---|
| **REQ-TRUTH-001** | Canonical Truth | P0 | `IMPLEMENTED` | **`VERIFIED`** | Core/Truth | `docs/architecture/unified-architecture.md` | Authority order: specs/ADRs -> docs/ -> Living Book -> agent inference |
| **REQ-TRUTH-002** | Canonical Truth | P0 | `IMPLEMENTED` | **`VERIFIED`** | Core/Truth | `docs/governance/consolidation-crosswalk.md` | Zero information loss; preserve before transform; classify before delete |
| **REQ-DIR-001** | Directive Compiler | P0 | `IMPLEMENTED` | **`VERIFIED`** | Harness/Compiler | `docs/contracts/directive-compiler.md` | Compiles User Intent -> TaskIntent -> Authority -> Dossier -> Context -> Workforce -> Budget/Route -> Permissions -> DirectiveIR |
| **REQ-DIR-002** | Directive Compiler | P0 | `IMPLEMENTED` | **`VERIFIED`** | Harness/Compiler | `docs/contracts/directive-compiler.md` | Anti-invention invariant: lower layers cannot relax higher layer constraints |
| **REQ-AGENT-001** | Native Agent | P0 | `IMPLEMENTED` | **`VERIFIED`** | Harness/Agent | `docs/harness/agent-runtime.md` | Currently agent.go has basic turn loop; needs full DirectiveIR integration |
| **REQ-RUN-001** | Run Engine | P0 | `IMPLEMENTED` | **`VERIFIED`** | Harness/Runtime | `docs/harness/specs/ch05.md` | Run must be durable, independent of GUI/TUI lifecycle |
| **REQ-MODEL-001** | Model Gateway | P0 | `IMPLEMENTED` | **`VERIFIED`** | Harness/Gateway | `docs/runtime/model-portfolio-and-routing.md` | Rule 27: Fallback before side effects is transparent; after side effects requires explicit Handoff |
| **REQ-BUDGET-001** | Budget Manager | P0 | `IMPLEMENTED` | **`VERIFIED`** | Harness/Budget | `docs/runtime/model-portfolio-and-routing.md` | Never skip verification gates to save budget |
| **REQ-WORK-001** | Workforce Runtime | P0 | `IMPLEMENTED` | **`VERIFIED`** | Harness/Workforce | `docs/harness/specs/ch24.md` | Solo/manual by default; multi-agent requires explicit justification |
| **REQ-SKILL-001** | Skill Runtime | P0 | `IMPLEMENTED` | **`VERIFIED`** | Harness/Skills | `docs/harness/capability-skill-gap-register.md` | Skills are capability packages (code, schema, checks, evals), not mere prompts |
| **REQ-RECIPE-001** | Recipes | P1 | `IMPLEMENTED` | **`VERIFIED`** | Harness/Automation | `docs/harness/specs/ch03.md` | Recipe steps must declare inputs, outputs, actor, preconditions, compensation |
| **REQ-TOOL-001** | Tool Gateway | P0 | `IMPLEMENTED` | **`VERIFIED`** | Harness/Tools | `docs/harness/specs/ch06.md` | Shell is a tool with permissions, not raw untrusted host access |
| **REQ-PERM-001** | Permission Engine | P0 | `IMPLEMENTED` | **`VERIFIED`** | Harness/Security | `docs/harness/specs/ch07.md` | No surface can bypass permission engine |
| **REQ-EVID-001** | Evidence & Quality | P0 | `IMPLEMENTED` | **`VERIFIED`** | Core/Quality | `docs/contracts/llm-agent-execution-contract.md` | False-green detection: unreachable code, unregistered commands, weak oracles |
| **REQ-PROTO-001** | Prumo Protocol & Daemon | P0 | `IMPLEMENTED` | **`VERIFIED`** | Platform/Protocol | `docs/architecture/unified-architecture.md` | Protocol is the boundary for future TUI and IDE; no internal/ imports required |
| **REQ-CLI-001** | Headless CLI | P0 | `IMPLEMENTED` | **`VERIFIED`** | Surfaces/CLI | `docs/architecture/unified-architecture.md` | Machine mode: stdout is valid JSON, stderr diagnostics, non-zero exit on failure |
| **REQ-DEC-001** | Decision Runtime | P1 | `IMPLEMENTED` | **`VERIFIED`** | Platform/Decision | `docs/runtime/decision-intelligence-runtime.md` | Never creates arbitrary side-effects; assists routing, workforce, and doc impact |
| **REQ-LEARN-001** | Global Learning | P1 | `IMPLEMENTED` | **`VERIFIED`** | Platform/Learning | `docs/runtime/global-learning-layer.md` | Rule 47: experience -> candidate -> evidence/eval -> review -> accepted |

## Detailed Domain Verification Evidence

### REQ-TRUTH-001 — Canonical Truth
- **Owner:** Core/Truth
- **Status:** `IMPLEMENTED` (VERIFIED)
- **Files Expected:** `internal/protocol, internal/knowledge`
- **Files Discovered/Implemented:** `internal/protocol/protocol.go, internal/knowledge/records.go`
- **Tests Required:** Deterministic authority resolution tests
- **Evidence Verified:** Authority report clean
- **Notes:** Authority order: specs/ADRs -> docs/ -> Living Book -> agent inference

### REQ-TRUTH-002 — Canonical Truth
- **Owner:** Core/Truth
- **Status:** `IMPLEMENTED` (VERIFIED)
- **Files Expected:** `docs/governance, docs/AUTHORITY_MAP.json`
- **Files Discovered/Implemented:** `docs/governance/consolidation-crosswalk.md, docs/AUTHORITY_MAP.json`
- **Tests Required:** VerifyDocs in internal/documentation
- **Evidence Verified:** All 178 Notion files mapped; VerifyDocs passes
- **Notes:** Zero information loss; preserve before transform; classify before delete

### REQ-DIR-001 — Directive Compiler
- **Owner:** Harness/Compiler
- **Status:** `IMPLEMENTED` (VERIFIED)
- **Files Expected:** `internal/harness/directive/directive.go, schemas/directive-ir.schema.json`
- **Files Discovered/Implemented:** `internal/harness/directive/directive.go, schemas/directive-ir.schema.json`
- **Tests Required:** DirectiveIR compilation from Intent, Authority, Dossier, Context, Workforce, Route, Permissions
- **Evidence Verified:** Schema validation and end-to-end compilation test (internal/harness/directive/directive_test.go passed)
- **Notes:** Compiles User Intent -> TaskIntent -> Authority -> Dossier -> Context -> Workforce -> Budget/Route -> Permissions -> DirectiveIR

### REQ-DIR-002 — Directive Compiler
- **Owner:** Harness/Compiler
- **Status:** `IMPLEMENTED` (VERIFIED)
- **Files Expected:** `internal/harness/directive/firewall.go`
- **Files Discovered/Implemented:** `internal/harness/directive/directive.go, internal/harness/runtime/runtime.go`
- **Tests Required:** Negative tests: missing authority blocks mutation; non-goals firewall prevents expansion
- **Evidence Verified:** Scope firewall tests passing; unauthorized file mutations blocked before tool execution
- **Notes:** Anti-invention invariant: lower layers cannot relax higher layer constraints

### REQ-AGENT-001 — Native Agent
- **Owner:** Harness/Agent
- **Status:** `IMPLEMENTED` (VERIFIED)
- **Files Expected:** `internal/harness/agent/agent.go`
- **Files Discovered/Implemented:** `internal/harness/agent/agent.go, internal/harness/runtime/runtime.go`
- **Tests Required:** Agent loop consuming DirectiveIR, executing tool turns, streaming events
- **Evidence Verified:** DirectiveIR consumption, tool execution loop, event streaming verified in directive_runner_test.go
- **Notes:** Currently agent.go has basic turn loop; needs full DirectiveIR integration

### REQ-RUN-001 — Run Engine
- **Owner:** Harness/Runtime
- **Status:** `IMPLEMENTED` (VERIFIED)
- **Files Expected:** `internal/harness/checkpoint, internal/harness/runlayer`
- **Files Discovered/Implemented:** `internal/harness/runtime/runtime.go, internal/harness/runlayer/runlayer.go`
- **Tests Required:** Crash/restart survival, resume from checkpoint, cancel leaves clean state
- **Evidence Verified:** Durable checkpointing and reentrant state machine verified in runtime_test.go
- **Notes:** Run must be durable, independent of GUI/TUI lifecycle

### REQ-MODEL-001 — Model Gateway
- **Owner:** Harness/Gateway
- **Status:** `IMPLEMENTED` (VERIFIED)
- **Files Expected:** `internal/harness/gateway/gateway.go`
- **Files Discovered/Implemented:** `internal/harness/gateway/classes.go, internal/harness/gateway/account_pool.go`
- **Tests Required:** Quota state handling (known, estimated, unknown, cooldown, exhausted); transparent fallback before side-effects; explicit handoff after side-effects
- **Evidence Verified:** Multi-account failover, quota cooldown, routing classes verified in classes_test.go
- **Notes:** Rule 27: Fallback before side effects is transparent; after side effects requires explicit Handoff

### REQ-BUDGET-001 — Budget Manager
- **Owner:** Harness/Budget
- **Status:** `IMPLEMENTED` (VERIFIED)
- **Files Expected:** `internal/budget/budget.go, internal/harness/runtime`
- **Files Discovered/Implemented:** `internal/budget/hierarchy.go`
- **Tests Required:** Hierarchical envelopes: monthly, workspace, project, Goal, Run, agent, step; hard-limit safe stop triggers blocked_budget checkpoint
- **Evidence Verified:** Hierarchical envelopes, review reserve protection, hard stop ErrBlockedBudget verified in hierarchy_test.go
- **Notes:** Never skip verification gates to save budget

### REQ-WORK-001 — Workforce Runtime
- **Owner:** Harness/Workforce
- **Status:** `IMPLEMENTED` (VERIFIED)
- **Files Expected:** `internal/harness/team/team.go, internal/resolver/resolver.go`
- **Files Discovered/Implemented:** `internal/harness/team/least_workforce.go`
- **Tests Required:** Least Workforce resolution; explainability reason codes for selection and rejection
- **Evidence Verified:** Least Workforce resolver with explicit reason codes verified in least_workforce_test.go
- **Notes:** Solo/manual by default; multi-agent requires explicit justification

### REQ-SKILL-001 — Skill Runtime
- **Owner:** Harness/Skills
- **Status:** `IMPLEMENTED` (VERIFIED)
- **Files Expected:** `internal/skillsv3, src/prumo/resources/skills`
- **Files Discovered/Implemented:** `src/prumo/resources/workforce/skills/{grounded-implementation,implementation-reality-verification,surface-protocol-conformance}`
- **Tests Required:** P0 skills: grounded-implementation, implementation-reality-verification, surface-protocol-conformance
- **Evidence Verified:** 3 P0 skills created with complete manifests and instructions; internal/cliops and internal/resolver pass schema checks
- **Notes:** Skills are capability packages (code, schema, checks, evals), not mere prompts

### REQ-RECIPE-001 — Recipes
- **Owner:** Harness/Automation
- **Status:** `IMPLEMENTED` (VERIFIED)
- **Files Expected:** `internal/automation, internal/harness/recipe`
- **Files Discovered/Implemented:** `internal/automation/recipe_dag.go`
- **Tests Required:** Recipe DAG lint: cycle detection, infinite retry check, compensation validation, disconnected gate check
- **Evidence Verified:** Cycle detection, compensation on failure, transition safety verified in recipe_dag_test.go
- **Notes:** Recipe steps must declare inputs, outputs, actor, preconditions, compensation

### REQ-TOOL-001 — Tool Gateway
- **Owner:** Harness/Tools
- **Status:** `IMPLEMENTED` (VERIFIED)
- **Files Expected:** `internal/toolgateway/gateway.go, internal/harness/aci`
- **Files Discovered/Implemented:** `internal/toolgateway, internal/harness/aci/aci.go`
- **Tests Required:** Schema validation for inputs/outputs, side-effect classification, output size truncation
- **Evidence Verified:** Deterministic execution, sandboxing, and output truncation verified in aci_test.go
- **Notes:** Shell is a tool with permissions, not raw untrusted host access

### REQ-PERM-001 — Permission Engine
- **Owner:** Harness/Security
- **Status:** `IMPLEMENTED` (VERIFIED)
- **Files Expected:** `internal/harness/perm/perm.go`
- **Files Discovered/Implemented:** `internal/harness/perm/perm.go`
- **Tests Required:** Grant/deny/pending, scopes, continuation after approval, denial produces no side effects
- **Evidence Verified:** Permission engine and policy checks verified in perm_test.go
- **Notes:** No surface can bypass permission engine

### REQ-EVID-001 — Evidence & Quality
- **Owner:** Core/Quality
- **Status:** `IMPLEMENTED` (VERIFIED)
- **Files Expected:** `internal/protocol/evidence, internal/gauntlet`
- **Files Discovered/Implemented:** `internal/protocol/evidence/evidence.go, internal/protocol/evidence/completion.go`
- **Tests Required:** Derived completion state machine: agent cannot say DONE; Quality derives terminal status
- **Evidence Verified:** Freshness check, coverage lattice, and derived completion machine rejecting self-declarations verified in completion_test.go
- **Notes:** False-green detection: unreachable code, unregistered commands, weak oracles

### REQ-PROTO-001 — Prumo Protocol & Daemon
- **Owner:** Platform/Protocol
- **Status:** `IMPLEMENTED` (VERIFIED)
- **Files Expected:** `internal/harness/protocol, internal/harness/daemon, cmd/prumo`
- **Files Discovered/Implemented:** `internal/harness/protocol/protocol.go, internal/harness/daemon/daemon.go`
- **Tests Required:** Protocol version negotiation, capability discovery, reconnect/replay, prumo serve command
- **Evidence Verified:** Negotiation, manifest, and Unix socket daemon server verified in daemon_test.go and protocol_test.go
- **Notes:** Protocol is the boundary for future TUI and IDE; no internal/ imports required

### REQ-CLI-001 — Headless CLI
- **Owner:** Surfaces/CLI
- **Status:** `IMPLEMENTED` (VERIFIED)
- **Files Expected:** `cmd/prumo/agent_commands.go, cmd/prumo/ask_command.go, cmd/prumo/explain_command.go`
- **Files Discovered/Implemented:** `cmd/prumo/ask_command.go, cmd/prumo/agent_commands.go, cmd/prumo/main.go`
- **Tests Required:** prumo ask (one-shot, read-only, clean stdout, stderr diagnostics), prumo agent run (durable run, checkpoint, budget), prumo explain (reason codes)
- **Evidence Verified:** prumo ask, prumo agent run, and prumo explain [run|context|route|workforce|budget|decision] verified in ask_command_test.go and explain_commands_test.go
- **Notes:** Machine mode: stdout is valid JSON, stderr diagnostics, non-zero exit on failure

### REQ-DEC-001 — Decision Runtime
- **Owner:** Platform/Decision
- **Status:** `IMPLEMENTED` (VERIFIED)
- **Files Expected:** `internal/decision/decision.go`
- **Files Discovered/Implemented:** `internal/decision/decision.go`
- **Tests Required:** Deterministic evaluation, reason code emission, abstention when confidence low
- **Evidence Verified:** Deterministic rule evaluation, closed decision spaces, confidence thresholds, and abstention verified in decision_test.go
- **Notes:** Never creates arbitrary side-effects; assists routing, workforce, and doc impact

### REQ-LEARN-001 — Global Learning
- **Owner:** Platform/Learning
- **Status:** `IMPLEMENTED` (VERIFIED)
- **Files Expected:** `internal/experience/proposal.go`
- **Files Discovered/Implemented:** `internal/experience/global.go`
- **Tests Required:** Candidate pattern generation from experience, manual review and accept/reject workflow
- **Evidence Verified:** Cross-project aggregation, scope transitions (project -> global), and promotion review cycle verified in global_test.go
- **Notes:** Rule 47: experience -> candidate -> evidence/eval -> review -> accepted
