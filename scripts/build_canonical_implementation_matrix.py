import json
import os

matrix = [
    # 1. FOUNDATION & CANONICAL TRUTH
    {
        "id": "REQ-TRUTH-001",
        "domain": "Canonical Truth",
        "owner": "Core/Truth",
        "priority": "P0",
        "source_doc": "docs/architecture/unified-architecture.md",
        "section": "Regra de ownership / Canonical Project Model",
        "status": "IMPLEMENTED",
        "verification_status": "VERIFIED",
        "files_expected": "internal/protocol, internal/knowledge",
        "files_discovered": "internal/protocol/protocol.go, internal/knowledge/records.go",
        "tests_required": "Deterministic authority resolution tests",
        "evidence_required": "Authority report clean",
        "dependencies": [],
        "blockers": [],
        "notes": "Authority order: specs/ADRs -> docs/ -> Living Book -> agent inference"
    },
    {
        "id": "REQ-TRUTH-002",
        "domain": "Canonical Truth",
        "owner": "Core/Truth",
        "priority": "P0",
        "source_doc": "docs/governance/consolidation-crosswalk.md",
        "section": "Regra de consolidação & Supersession",
        "status": "IMPLEMENTED",
        "verification_status": "VERIFIED",
        "files_expected": "docs/governance, docs/AUTHORITY_MAP.json",
        "files_discovered": "docs/governance/consolidation-crosswalk.md, docs/AUTHORITY_MAP.json",
        "tests_required": "VerifyDocs in internal/documentation",
        "evidence_required": "All 178 Notion files mapped; VerifyDocs passes",
        "dependencies": ["REQ-TRUTH-001"],
        "blockers": [],
        "notes": "Zero information loss; preserve before transform; classify before delete"
    },

    # 2. DIRECTIVE COMPILER
    {
        "id": "REQ-DIR-001",
        "domain": "Directive Compiler",
        "owner": "Harness/Compiler",
        "priority": "P0",
        "source_doc": "docs/contracts/directive-compiler.md",
        "section": "Directive IR & Compilation Pipeline",
        "status": "NOT_STARTED",
        "verification_status": "UNVERIFIED",
        "files_expected": "internal/harness/directive/directive.go, schemas/directive-ir.schema.json",
        "files_discovered": "None",
        "tests_required": "DirectiveIR compilation from Intent, Authority, Dossier, Context, Workforce, Route, Permissions",
        "evidence_required": "Schema validation and end-to-end compilation test",
        "dependencies": ["REQ-TRUTH-001"],
        "blockers": [],
        "notes": "Compiles User Intent -> TaskIntent -> Authority -> Dossier -> Context -> Workforce -> Budget/Route -> Permissions -> DirectiveIR"
    },
    {
        "id": "REQ-DIR-002",
        "domain": "Directive Compiler",
        "owner": "Harness/Compiler",
        "priority": "P0",
        "source_doc": "docs/contracts/directive-compiler.md",
        "section": "Missing Authority & Scope Firewall",
        "status": "NOT_STARTED",
        "verification_status": "UNVERIFIED",
        "files_expected": "internal/harness/directive/firewall.go",
        "files_discovered": "None",
        "tests_required": "Negative tests: missing authority blocks mutation; non-goals firewall prevents expansion",
        "evidence_required": "Unit tests verifying refusal/blocking when authority missing",
        "dependencies": ["REQ-DIR-001"],
        "blockers": [],
        "notes": "Anti-invention invariant: lower layers cannot relax higher layer constraints"
    },

    # 3. NATIVE AGENT & RUN ENGINE
    {
        "id": "REQ-AGENT-001",
        "domain": "Native Agent",
        "owner": "Harness/Agent",
        "priority": "P0",
        "source_doc": "docs/harness/agent-runtime.md",
        "section": "Reentrant Loop & Directive Consumption",
        "status": "PARTIAL",
        "verification_status": "EXERCISED",
        "files_expected": "internal/harness/agent/agent.go",
        "files_discovered": "internal/harness/agent/agent.go",
        "tests_required": "Agent loop consuming DirectiveIR, executing tool turns, streaming events",
        "evidence_required": "Agent execution test with step checkpoints and event emission",
        "dependencies": ["REQ-DIR-001"],
        "blockers": [],
        "notes": "Currently agent.go has basic turn loop; needs full DirectiveIR integration"
    },
    {
        "id": "REQ-RUN-001",
        "domain": "Run Engine",
        "owner": "Harness/Runtime",
        "priority": "P0",
        "source_doc": "docs/harness/specs/ch05.md",
        "section": "Durable State Machine, Checkpoint, Resume, Cancel",
        "status": "PARTIAL",
        "verification_status": "EXERCISED",
        "files_expected": "internal/harness/checkpoint, internal/harness/runlayer",
        "files_discovered": "internal/harness/checkpoint/checkpoint.go, internal/harness/runlayer/runlayer.go",
        "tests_required": "Crash/restart survival, resume from checkpoint, cancel leaves clean state",
        "evidence_required": "Crash recovery tests in checkpoint_test.go",
        "dependencies": [],
        "blockers": [],
        "notes": "Run must be durable, independent of GUI/TUI lifecycle"
    },

    # 4. MODEL GATEWAY & ROUTING
    {
        "id": "REQ-MODEL-001",
        "domain": "Model Gateway",
        "owner": "Harness/Gateway",
        "priority": "P0",
        "source_doc": "docs/runtime/model-portfolio-and-routing.md",
        "section": "ROUTE-CHEAP, ROUTE-PRIMARY, Quota & Fallback vs Handoff",
        "status": "PARTIAL",
        "verification_status": "EXERCISED",
        "files_expected": "internal/harness/gateway/gateway.go",
        "files_discovered": "internal/harness/gateway/gateway.go",
        "tests_required": "Quota state handling (known, estimated, unknown, cooldown, exhausted); transparent fallback before side-effects; explicit handoff after side-effects",
        "evidence_required": "Gateway test suite verifying pre-side-effect fallback and post-side-effect refusal",
        "dependencies": [],
        "blockers": [],
        "notes": "Rule 27: Fallback before side effects is transparent; after side effects requires explicit Handoff"
    },

    # 5. BUDGET MANAGER
    {
        "id": "REQ-BUDGET-001",
        "domain": "Budget Manager",
        "owner": "Harness/Budget",
        "priority": "P0",
        "source_doc": "docs/runtime/model-portfolio-and-routing.md",
        "section": "Hierarchical Envelopes, Hard Stop & Blocked Budget",
        "status": "PARTIAL",
        "verification_status": "EXERCISED",
        "files_expected": "internal/budget/budget.go, internal/harness/runtime",
        "files_discovered": "internal/budget/budget.go",
        "tests_required": "Hierarchical envelopes: monthly, workspace, project, Goal, Run, agent, step; hard-limit safe stop triggers blocked_budget checkpoint",
        "evidence_required": "Budget exhaustion test verifying checkpoint creation and safe exit",
        "dependencies": ["REQ-RUN-001"],
        "blockers": [],
        "notes": "Never skip verification gates to save budget"
    },

    # 6. WORKFORCE RUNTIME
    {
        "id": "REQ-WORK-001",
        "domain": "Workforce Runtime",
        "owner": "Harness/Workforce",
        "priority": "P0",
        "source_doc": "docs/harness/specs/ch24.md",
        "section": "Minimum Sufficient Workforce & Selection Explainability",
        "status": "PARTIAL",
        "verification_status": "EXERCISED",
        "files_expected": "internal/harness/team/team.go, internal/resolver/resolver.go",
        "files_discovered": "internal/harness/team/team.go, internal/resolver/resolver.go",
        "tests_required": "Least Workforce resolution; explainability reason codes for selection and rejection",
        "evidence_required": "Workforce resolution test asserting reason codes and minimal agent count",
        "dependencies": [],
        "blockers": [],
        "notes": "Solo/manual by default; multi-agent requires explicit justification"
    },

    # 7. SKILL RUNTIME & P0 GAP REGISTER
    {
        "id": "REQ-SKILL-001",
        "domain": "Skill Runtime",
        "owner": "Harness/Skills",
        "priority": "P0",
        "source_doc": "docs/harness/capability-skill-gap-register.md",
        "section": "P0 Confiança Sistêmica",
        "status": "PARTIAL",
        "verification_status": "UNVERIFIED",
        "files_expected": "internal/skillsv3, src/prumo/resources/skills",
        "files_discovered": "internal/skillsv3/skills.go",
        "tests_required": "P0 skills: grounded-implementation, implementation-reality-verification, surface-protocol-conformance",
        "evidence_required": "Skill manifest schemas, negative triggers, and evaluation fixtures",
        "dependencies": [],
        "blockers": [],
        "notes": "Skills are capability packages (code, schema, checks, evals), not mere prompts"
    },

    # 8. RECIPES RUNTIME
    {
        "id": "REQ-RECIPE-001",
        "domain": "Recipes",
        "owner": "Harness/Automation",
        "priority": "P1",
        "source_doc": "docs/harness/specs/ch03.md",
        "section": "Recipe DAG, Step Contracts, Compensation & Lint",
        "status": "PARTIAL",
        "verification_status": "UNVERIFIED",
        "files_expected": "internal/automation, internal/harness/recipe",
        "files_discovered": "internal/automation/runner.go",
        "tests_required": "Recipe DAG lint: cycle detection, infinite retry check, compensation validation, disconnected gate check",
        "evidence_required": "Linter tests catching invalid recipe DAGs",
        "dependencies": [],
        "blockers": [],
        "notes": "Recipe steps must declare inputs, outputs, actor, preconditions, compensation"
    },

    # 9. TOOL GATEWAY & ACI
    {
        "id": "REQ-TOOL-001",
        "domain": "Tool Gateway",
        "owner": "Harness/Tools",
        "priority": "P0",
        "source_doc": "docs/harness/specs/ch06.md",
        "section": "Tool Descriptors, Permissions & Result Budget",
        "status": "PARTIAL",
        "verification_status": "EXERCISED",
        "files_expected": "internal/toolgateway/gateway.go, internal/harness/aci",
        "files_discovered": "internal/toolgateway/gateway.go, internal/harness/aci/aci.go",
        "tests_required": "Schema validation for inputs/outputs, side-effect classification, output size truncation",
        "evidence_required": "ACI execution test with permission check and result budget enforcement",
        "dependencies": [],
        "blockers": [],
        "notes": "Shell is a tool with permissions, not raw untrusted host access"
    },

    # 10. PERMISSION ENGINE
    {
        "id": "REQ-PERM-001",
        "domain": "Permission Engine",
        "owner": "Harness/Security",
        "priority": "P0",
        "source_doc": "docs/harness/specs/ch07.md",
        "section": "Permission Lifecycle & Non-Bypassable Engine",
        "status": "IMPLEMENTED",
        "verification_status": "EXERCISED",
        "files_expected": "internal/harness/perm/perm.go",
        "files_discovered": "internal/harness/perm/perm.go",
        "tests_required": "Grant/deny/pending, scopes, continuation after approval, denial produces no side effects",
        "evidence_required": "perm_test.go passed",
        "dependencies": [],
        "blockers": [],
        "notes": "No surface can bypass permission engine"
    },

    # 11. EVIDENCE SYSTEM & QUALITY GAUNTLET
    {
        "id": "REQ-EVID-001",
        "domain": "Evidence & Quality",
        "owner": "Core/Quality",
        "priority": "P0",
        "source_doc": "docs/contracts/llm-agent-execution-contract.md",
        "section": "Structured Evidence, Freshness & Derived Completion",
        "status": "PARTIAL",
        "verification_status": "EXERCISED",
        "files_expected": "internal/protocol/evidence, internal/gauntlet",
        "files_discovered": "internal/protocol/evidence/evidence.go, internal/gauntlet/run.go",
        "tests_required": "Derived completion state machine: agent cannot say DONE; Quality derives terminal status",
        "evidence_required": "Gauntlet run producing structured evidence artifact",
        "dependencies": [],
        "blockers": [],
        "notes": "False-green detection: unreachable code, unregistered commands, weak oracles"
    },

    # 12. PRUMO PROTOCOL & DAEMON
    {
        "id": "REQ-PROTO-001",
        "domain": "Prumo Protocol & Daemon",
        "owner": "Platform/Protocol",
        "priority": "P0",
        "source_doc": "docs/architecture/unified-architecture.md",
        "section": "Prumo Protocol & Daemon (prumo serve)",
        "status": "PARTIAL",
        "verification_status": "EXERCISED",
        "files_expected": "internal/harness/protocol, internal/harness/daemon, cmd/prumo",
        "files_discovered": "internal/harness/protocol/protocol.go, internal/harness/daemon/daemon.go",
        "tests_required": "Protocol version negotiation, capability discovery, reconnect/replay, prumo serve command",
        "evidence_required": "Daemon socket connection and capability negotiation test",
        "dependencies": [],
        "blockers": [],
        "notes": "Protocol is the boundary for future TUI and IDE; no internal/ imports required"
    },

    # 13. HEADLESS CLI
    {
        "id": "REQ-CLI-001",
        "domain": "Headless CLI",
        "owner": "Surfaces/CLI",
        "priority": "P0",
        "source_doc": "docs/architecture/unified-architecture.md",
        "section": "prumo ask, prumo agent run, prumo explain",
        "status": "PARTIAL",
        "verification_status": "PARTIAL",
        "files_expected": "cmd/prumo/agent_commands.go, cmd/prumo/ask_command.go, cmd/prumo/explain_command.go",
        "files_discovered": "cmd/prumo/agent_commands.go",
        "tests_required": "prumo ask (one-shot, read-only, clean stdout, stderr diagnostics), prumo agent run (durable run, checkpoint, budget), prumo explain (reason codes)",
        "evidence_required": "CLI execution tests with --json machine-mode output",
        "dependencies": ["REQ-AGENT-001", "REQ-DIR-001", "REQ-MODEL-001"],
        "blockers": [],
        "notes": "Machine mode: stdout is valid JSON, stderr diagnostics, non-zero exit on failure"
    },

    # 14. DECISION RUNTIME
    {
        "id": "REQ-DEC-001",
        "domain": "Decision Runtime",
        "owner": "Platform/Decision",
        "priority": "P1",
        "source_doc": "docs/runtime/decision-intelligence-runtime.md",
        "section": "Small Closed Decisions, Reason Codes & Abstention",
        "status": "NOT_STARTED",
        "verification_status": "UNVERIFIED",
        "files_expected": "internal/decision/decision.go",
        "files_discovered": "None",
        "tests_required": "Deterministic evaluation, reason code emission, abstention when confidence low",
        "evidence_required": "Decision runtime test suite",
        "dependencies": [],
        "blockers": [],
        "notes": "Never creates arbitrary side-effects; assists routing, workforce, and doc impact"
    },

    # 15. GLOBAL LEARNING & EXPERIENCE
    {
        "id": "REQ-LEARN-001",
        "domain": "Global Learning",
        "owner": "Platform/Learning",
        "priority": "P1",
        "source_doc": "docs/runtime/global-learning-layer.md",
        "section": "Candidate Patterns & Review Cycle",
        "status": "PARTIAL",
        "verification_status": "EXERCISED",
        "files_expected": "internal/experience/proposal.go",
        "files_discovered": "internal/experience/proposal.go, internal/experience/summary.go",
        "tests_required": "Candidate pattern generation from experience, manual review and accept/reject workflow",
        "evidence_required": "Experience tests verifying candidates do not automatically mutate canonical policies",
        "dependencies": [],
        "blockers": [],
        "notes": "Rule 47: experience -> candidate -> evidence/eval -> review -> accepted"
    }
]

os.makedirs('docs/harness', exist_ok=True)
with open('docs/harness/canonical-implementation-matrix.json', 'w', encoding='utf-8') as f:
    json.dump(matrix, f, indent=2, ensure_ascii=False)

md_lines = [
    '# Canonical Implementation Matrix — Prumo Harness',
    '',
    'Matriz normativa de rastreabilidade entre requisitos da documentação consolidada e a realidade do código.',
    'Status possíveis: `NOT_STARTED`, `PLANNED`, `PARTIAL`, `IMPLEMENTED`, `REACHABLE`, `EXERCISED`, `EVIDENCED`, `VERIFIED`, `BLOCKED`, `SUPERSEDED`, `NOT_APPLICABLE`.',
    '',
    '| ID | Domínio | Prioridade | Status | Verificação | Arquivos Esperados | Realidade Atual | Testes / Evidência Requerida |',
    '|---|---|---|---|---|---|---|---|'
]

for item in matrix:
    md_lines.append(f"| `{item['id']}` | {item['domain']} | {item['priority']} | `{item['status']}` | `{item['verification_status']}` | `{item['files_expected']}` | `{item['files_discovered']}` | {item['tests_required']} |")

with open('docs/harness/canonical-implementation-matrix.md', 'w', encoding='utf-8') as f:
    f.write('\n'.join(md_lines) + '\n')

print(f'Canonical Implementation Matrix created with {len(matrix)} items.')
