# 92 — Directive Compiler: Regras Executáveis para LLMs, Agents, Harness e Surfaces

> Authority: canonical specification.
> Logical ID: CONST-92
> Source: Notion Living Book (3e29bb7d023f816ba3acc5247e301af5)
> Status: Constituição 92: Directive Compiler e Regras Executáveis.


<aside>
🧬

**Status: constituição operacional executável.** Esta página traduz as constituições 84–91 em um contrato compilável. Nenhuma surface, agent, skill, recipe ou provider pode reinterpretar essas regras livremente. O objetivo é tornar a filosofia anti-invenção, anti-drift e evidence-first simultaneamente explícita em instruções e implícita na arquitetura.

</aside>

# Objetivo

O Prumo não deve confiar que uma LLM se comporte bem apenas porque recebeu um bom prompt. O sistema deve compilar um **Execution Contract** a partir de estado canônico, policies, Dossier, Context Manifest, Workforce, Budget e permissions, e aplicar enforcement determinístico antes, durante e depois da execução.

# Regra de precedência

1. hard safety/security/permission policies;
2. canonical authority/supersession;
3. protocol/schema invariants;
4. current accepted Decision/Requirement/Constraint;
5. Task/Dossier scope e acceptance;
6. selected Skill/Recipe procedures;
7. provider-specific adaptation;
8. current conversational convenience.

Nenhuma camada inferior pode relaxar uma superior.

# Directive IR

Toda execução material deve ser compilada para uma estrutura intermediária versionada, **DirectiveIR**, com:

- directive_ir_version;
- project/workspace/repository revision;
- Goal/Plan/Task IDs;
- authority snapshot;
- active decisions/requirements/constraints;
- explicit non-goals;
- unknowns e assumptions;
- mutation boundaries;
- permission envelope;
- tool capability set;
- model/agent route;
- workforce bindings;
- budget envelope + review reserve;
- required skills/recipes;
- required evidence;
- quality gates;
- stop/escalation conditions;
- documentation delta contract;
- output schema;
- source digests.

LLMs recebem uma projeção dessa estrutura; o Core mantém a estrutura original.

# Compilation pipeline

TaskIntent → Authority Resolver → Implementation Dossier → Context Compiler → Workforce Resolver → Budget/Route → Permission/Tool Policy → DirectiveIR → Provider/Agent Adapter → Execution → Evidence/Quality → Derived Outcome.

# Invariantes de compilação

- UNKNOWN nunca é convertido em fato para completar template.
- SUPERSEDED e STALE ficam fora do contexto padrão.
- PLANNED_TARGET nunca é apresentado como VERIFIED_IMPLEMENTATION.
- paths/symbols de realidade atual precisam de repository grounding.
- nenhum provider recebe credencial/tool além do envelope necessário.
- nenhuma skill pode elevar sua própria prioridade acima de hard policy.
- budget não pode remover gate obrigatório.
- output done do agent não altera readiness.

# Missing authority

- read-only: continuar com UNKNOWN, limites explícitos e alternativas rotuladas;
- reversível/baixo risco: somente default versionado ou assumption permitida;
- mutation material: bloquear parte dependente;
- destructive/security/migration/public-contract: fail closed.

# Scope firewall

Materializar IN, OUT e INCIDENTAL. Tentativa de tocar OUT deve ser rejeitada ou virar scope-change proposal. Refactor oportunista permanece finding.

# Tool firewall

Cada ação de tool inclui capability, target, side-effect class, expected output, idempotency/retry, permission decision, budget charge e evidence hook. Shell não é exceção.

# Claim firewall

Claims críticos exigem source/evidence, revision, authority, freshness e applicability. Categorias: path/symbol, test result, readiness, quota, protocol support, permission e migration state.

# Completion firewall

Estados derivados: implemented → reachable → exercised → evidenced → verified → accepted → released.

Run final pode ser completed_verified, completed_unverified, partial, blocked_unknown, blocked_budget, blocked_permission, failed ou canceled. Clients não podem mascarar isso.

# Prompt assembly

1. hard policies;
2. identity/role;
3. canonical invariants;
4. scope/acceptance/non-goals;
5. implementation facts;
6. relevant skill procedures;
7. tools/permissions;
8. evidence/gates;
9. budget/stop rules;
10. output contract.

Histórico e brainstorming só entram por progressive disclosure.

# Provider adaptation

Adapters podem mudar formato, nunca semântica. Devem mapear tools, instructions, context limits, structured output, cancellation, streaming e errors sem redefinir Decision, Permission, Budget, Evidence ou Run.

# External AgentProvider

O Prumo normaliza events, observa side effects, revalida outputs, ignora auto-declaração de completion sem evidence e produz handoff em falha/quota/limite.

# Workforce directives

Cada AgentBinding declara role, purpose, allowed tasks, forbidden responsibilities, route, permissions, budget reservation, context tier, output schema, evidence responsibility e handoff target.

# Skill directives

Skills fornecem procedimento e conhecimento. Não podem criar requirement, alterar budget, elevar permission, declarar completion nem sobrescrever canonical decision.

# Recipe directives

Recipes são DAGs versionados. Cada step declara preconditions, capability, actor, inputs, outputs, side effects, retries, compensation/recovery, gate e next-step conditions.

# CLI

Compartilha services, oferece schemas versionados, stdout limpo em machine mode, exit codes coerentes e explainability por reason codes/refs.

# TUI

É client do Protocol; não possui rule engine paralela; mostra stale/reconnect, blockers, approvals, budget, evidence e ownership; multi-agent é projeção do Workforce Runtime.

# GUI

Não é fonte de verdade; não persiste Run por bypass; mostra unknown/stale/conflict; não cria controle sem capability real; Follow Agent é opt-in; conflito human×agent é first-class.

# Mensagens de erro

Toda falha possui machine code, human message, retryability, owner/boundary, refs e safe next action quando determinístico.

# Conformance suites

- DirectiveIR schema;
- precedence;
- prompt projection snapshots;
- provider semantic equivalence;
- scope firewall;
- claim grounding;
- permission/tool firewall;
- budget enforcement;
- completion derivation;
- surface state equivalence.

# Evals adversariais

Cobrir prompt injection em repo, instrução conflitante em issue/log, path inexistente, decisão antiga similar, teste não executado, botão sem backend, budget esgotado antes do review, provider done prematuro, tool não autorizada, fallback pós-side-effect e compaction que remove constraint.

# Observabilidade

Toda Run deve explicar, sem raciocínio privado: fontes, decisões ativas, skills/agents selecionados, reason codes, tools autorizadas/negadas, budget, evidence e unknowns/assumptions.

# Definition of Done

- DirectiveIR versionado;
- compilação determinística onde aplicável;
- provider adapters não mudam semântica;
- hard rules não dependem de prompt;
- scope/permissions/budget/completion têm enforcement;
- CLI/TUI/GUI exibem o mesmo estado;
- external agents passam pelos mesmos gates;
- evals preferem UNKNOWN/BLOCKED a fabricação plausível.