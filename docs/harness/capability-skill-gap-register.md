# 93 — Capability & Skill Gap Register: Lacunas Obrigatórias, Anti-Duplicação e Evals

> Authority: canonical specification.
> Logical ID: CONST-93
> Source: Notion Living Book (3e29bb7d023f81dd9258f046a178894b)
> Status: Constituição 93: Capability & Skill Gap Register do Harness.


<aside>
🧩

**Status: registro canônico de lacunas após a fusão.** Complementa o Programa 79 e 79.AA. O objetivo é tornar explícito o que ainda precisa existir como Core check, Skill, Agent, Recipe, Provider ou Documentation Contract, evitando lacunas e inflação artificial do workforce.

</aside>

# Regra

Classificar toda lacuna em CORE_CHECK, CORE_SERVICE, SKILL, AGENT_ROLE, RECIPE, PROVIDER_ADAPTER, DOCUMENTATION_CONTRACT, PROFILE/BUNDLE, RESEARCH_ONLY ou REJECT.

# P0 — confiança sistêmica

## canonical-authority-resolution

CORE_SERVICE + checks. Resolve authority, status, scope, supersession e freshness. Não criar skill.

## directive-compiler-conformance

CORE_CHECK + eval suite. Valida DirectiveIR, precedência e projeções.

## grounded-implementation

SKILL. Implementar antes de campanhas grandes de agents.

## implementation-reality-verification

SKILL. Distingue presence, reachability, exercise e verification.

## surface-protocol-conformance

SKILL + RECIPE. Obrigatória para CLI/TUI/GUI/ACP.

## evidence-integrity

CORE_CHECK + extensão de testing/evidence: freshness, provenance, oracle strength, digest e invalidation.

## recovery-idempotency

Extensão P0 de resilience/testing; só promover a skill própria se evals justificarem.

## permission-egress-boundary

CORE_SERVICE + security contracts.

## prompt-untrusted-content-boundary

CORE_CHECK. Repo/web/tool output é data, não control-plane instruction.

# P1 — maturidade operacional

## agent-aware-ui-ux

SKILL para Run states, streaming, approvals, evidence, budget, reconnect e human×agent conflict.

## documentation-promotion-reconciliation

SKILL + RECIPE para Notion/conversation/research → candidate → reconciliation → canonical promotion.

## protocol-evolution-compatibility

Extensão de compatibility-migrations: version negotiation, old/new clients, replay/event versioning, deprecation.

## budget-routing-operations

CORE_SERVICE + orchestration knowledge. Não criar cost-governor agent.

## context-quality-auditor

SKILL candidata: superseded leakage, missing critical contract, duplicated context, role contamination. Criar somente se checks do Context Compiler forem insuficientes.

## dependency-decision

SKILL candidata: add/keep/remove dependency com license, supply-chain, binary size, memory/perf, maintenance e replacement boundary.

## release-readiness-verification

RECIPE + reviewer specialization; evitar agent separado se composição atual cobrir.

## cost-quality-economics

REPORT/ANALYTICS: cost per accepted outcome, tokens per verified requirement, retry waste e review reserve utilization.

# P1 — documentação e anti-drift

## decision-impact-analysis

RECIPE: nova Decision → contratos afetados → implementation/docs/tests/surfaces → invalidation → migration.

## documentation-delta-verification

SKILL/CORE hybrid. Core detecta provável impacto; skill resolve conteúdo sem inventar.

## example-code-conformance

SKILL/check. Snippets compilam/executam quando aplicável e seguem current API.

## docs-link-and-symbol-integrity

CHECK determinístico para links, commands, flags, symbols e paths.

# P1 — qualidade do workforce

## agent-contract-linter

CHECK de role, scope, permissions, inputs/outputs, handoff, failures e evidence.

## skill-contract-linter

CHECK de manifest, activation positiva/negativa, permissions, output schema, freshness, examples e evals.

## recipe-dag-linter

CHECK de cycles, owner, retry ilimitado, compensation, gate e incompatibilidade de outputs.

## resolver-explainability

CORE capability. Toda seleção/rejeição de skill/agent/recipe possui reason codes.

# P2 — somente com evidência

- privacy-engineering;
- internationalization-localization;
- fuzz-property-testing;
- license-compliance;
- incident-response;
- schema-data-migration;
- performance-regression;
- reliability/incident-responder;
- recovery-verifier especializado;
- conformance-reviewer especializado;
- documentation-reconciler.

Todos passam por RFC anti-duplicação do Programa 79.

# Agents proibidos como solução arquitetural

Não criar anti-hallucination-agent, model-cost-governor, permission-agent como autoridade, completion-agent como autoridade, truth-agent por opinião ou documentation-filler que preencha gaps. Essas responsabilidades pertencem ao Core ou a roles assistivas sem autoridade final.

# Evals por classe

## Skill

activation precision/recall, false-positive/negative cost, output compliance, task success delta, stale-source behavior, ambiguous-input behavior, context overhead e abstention correctness.

## Agent

role-boundary adherence, handoff completeness, permission discipline, failure classification, independent-review quality, side-effect discipline e recovery.

## Recipe

DAG coverage, branch correctness, retry bound, compensation, gates, resumability, idempotency e failure propagation.

## Core Check

fixtures determinísticas, false-positive/negative rate, reason code e estabilidade em inputs equivalentes.

# Lifecycle

draft → experimental → recommended → verified → stable.

Gates:

- draft: owner, purpose, non-goals;
- experimental: schema + fixtures;
- recommended: positive/negative evals;
- verified: dogfood + baseline + incident handling;
- stable: freshness owner + migration/deprecation.

# Estrutura de skill estável

Manifest; Purpose/Non-goals; Activation/Negative triggers; Inputs; Procedure; Checks; Outputs; Evidence; Failure modes; Handoff; Examples good/bad; Evals; Freshness; Dependencies/conflicts; Version/deprecation.

# Context budget

Cada skill declara manifest_tokens, summary_tokens, workflow_tokens, knowledge_tokens, examples_tokens e maximum_context_tokens. O resolver carrega progressivamente e registra custo no Context Manifest.

# Anti-megaprompt

Não concatenar todas as skills. Compor invariant summary, role contract, task workflows, exact knowledge blocks e examples apenas quando discriminativos.

# Missing-skill discovery

Propor gap quando tasks recorrentes repetem failures, skills repetem boilerplate, há evidence/output sem owner, resolver exige pacote excessivo ou incident aponta capability ausente. Proposta não significa criação.

# Métricas do catálogo

orphan rate, unused rate, activation precision/recall, avg skills per Task, context cost per accepted outcome, duplicate capability rate, stale knowledge rate, unresolved ownership, failed handoff, evidence completeness, regression by skill update e capability coverage por profile.

# Definition of Done

- P0 classificado com owner;
- checks determinísticos não viraram prompts;
- skills têm negative triggers/evals;
- agents têm handoff/failure/permissions;
- recipes têm recovery;
- resolver explica decisões;
- catálogo mede uso e qualidade;
- gaps nascem de evidence, não de moda.