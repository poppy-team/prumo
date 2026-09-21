# 88 — Arquitetura Documental v2: Canonical Graph, IR, Microcontextos e Promotion

> Authority: canonical specification.
> Logical ID: CONST-88
> Source: Notion Living Book (3e29bb7d023f816c9f3fc45bdb57db7c)
> Status: Documento especializado: 88 — Arquitetura Documental v2 Canonical Graph, IR.


<aside>
📚

**Status: arquitetura documental consolidada.** Esta página une Documentation Contracts, Knowledge Runtime, Documentation Compiler, Human Docs Planner e Notebook-to-Repository Promotion em um único fluxo. O objetivo é permitir documentação extremamente rica sem obrigar agents a ler um corpus enorme e sem permitir que prosa plausível vire verdade.

</aside>

# Duas funções, uma base

O Prumo produz duas famílias documentais:

1. **agent-facing documentation** — contracts, manifests, context packs, handoffs, implementation dossiers e machine-readable projections;
2. **human-facing documentation** — README, architecture book, manuals, tutorials, how-tos, reference, troubleshooting, changelog, release notes e development chronicle.

Ambas derivam do mesmo Knowledge IR e Document IR. Não manter duas ontologias concorrentes.

# Source layers

A autoridade deve ser explicitamente separada:

- **Notion Living Book** = design, research, planejamento e registro de decisão;
- **Repository canonical records/contracts** = autoridade operacional;
- **Code/tests/evidence** = realidade implementada e verificável;
- **Generated docs/indexes** = materialized views derivadas e reconstruíveis.

O Notion não é automaticamente runtime truth. Mudança discutida no caderno precisa passar por promotion/reconciliation para alterar o repositório canônico.

# Canonical Knowledge Graph

Entidades mínimas:

- Decision;
- Requirement;
- Constraint;
- Assumption;
- Unknown;
- Claim;
- ResearchRecord;
- Evidence;
- Gate;
- Goal;
- Plan;
- Task;
- Risk;
- Dependency;
- Surface;
- Capability;
- Skill;
- Agent;
- Recipe;
- BudgetPolicy;
- ImplementationStatus;
- DriftRecord;
- Waiver;
- ContextManifest;
- ImplementationDossier.

Relacionamentos mínimos:

- requires;
- implements;
- verifies;
- contradicts;
- supersedes;
- depends_on;
- affects;
- owned_by;
- rendered_into;
- derived_from;
- evidenced_by;
- valid_for;
- blocks;
- resolves.

# No prose-only critical invariants

Toda regra crítica precisa possuir representação estruturada suficiente para:

- resolver;
- validar;
- versionar;
- testar;
- explicar;
- detectar drift.

Prosa continua essencial para compreensão humana, mas não pode ser o único enforcement de segurança, authority, completion, budget ou protocol.

# Knowledge IR → Document IR

Pipeline:

Canonical Records → Knowledge IR → semantic projections → Document IR → deterministic renderer → artifacts.

LLM não deve ser chamada para:

- ordenar;
- contar;
- criar backlinks;
- calcular coverage;
- decidir stable IDs;
- materializar dados já estruturados;
- preencher campos determinísticos.

# Documentation ownership modes

Cada DocumentationUnit deve declarar ownership:

- GENERATED — totalmente derivável;
- ASSISTED — LLM propõe a partir de authoritative brief;
- CURATED — humano/agent mantém sob contract;
- IMPORTED_SNAPSHOT — provenance/input de migração, nunca runtime authority;
- HISTORICAL — somente histórico;
- EXTERNAL_REFERENCE — fonte externa com freshness/provenance.

# GAP, não hallucination

Se um DocumentationUnit precisa de informação ausente:

insufficient authoritative knowledge → GAP → blocking/non-blocking question → owner → impact.

Nunca gerar prosa “provável” apenas para satisfazer template ou completeness score.

# Implementation Dossier

Antes de mudança material, o Prumo compila um pacote curto e versionado:

## Identity

- dossier_id;
- project/workspace;
- Goal ID;
- Plan/Task IDs;
- source revision;
- ContextManifest ID;
- dossier schema version.

## Scope

- objective;
- in-scope;
- non-goals;
- accepted decisions;
- invariants;
- explicit defaults;
- forbidden assumptions.

## Implementation reality

- relevant modules/files;
- current symbols/interfaces;
- current tests;
- observed implementation status;
- known drift;
- dependencies;
- compatibility boundary.

## Mutation contract

- allowed mutations;
- forbidden mutations;
- ownership boundaries;
- generated/managed regions;
- data/schema migration rules;
- public API constraints;
- SCM rules.

## Acceptance

- acceptance criteria;
- negative requirements;
- state-transition requirements;
- failure/recovery expectations;
- security/privacy constraints;
- performance/resource budgets;
- compatibility requirements;
- accessibility/UX requirements when applicable.

## Execution

- required skills;
- optional skills;
- selected agents/roles;
- tools and permissions;
- environment/sandbox;
- model route;
- budget envelope;
- stop/escalation policy.

## Verification

- required tests;
- test oracles;
- evidence requirements;
- independent review needs;
- Gauntlet profile;
- documentation delta expectations.

## Uncertainty

- unknowns;
- assumptions;
- open questions;
- blocked_external;
- waivers.

## Provenance

- authoritative source refs;
- digests;
- superseded sources excluded;
- research refs;
- generation/compiler versions.

O agent recebe o Dossier, não o Livro Vivo inteiro.

# Microcontext packs

Cada role recebe somente o necessário.

## Planner

Recebe:

- product/architecture;
- scope;
- open decisions;
- constraints;
- research questions;
- budget/risk envelope.

Não recebe detalhes de implementação irrelevantes.

## Implementer

Recebe:

- task contract;
- current implementation facts;
- exact interfaces;
- allowed mutation;
- examples relevantes;
- acceptance;
- failure/recovery;
- tests/evidence.

Não recebe brainstorming histórico por default.

## Reviewer

Recebe:

- acceptance;
- diff/change manifest;
- evidence;
- negative requirements;
- risk model;
- known unknowns;
- independent source refs.

Não reutilizar automaticamente o mesmo summary do implementer quando isso reduzir independência.

## Documentation agent

Recebe:

- changed claims;
- impacted DocumentationUnits;
- authoritative refs;
- ownership mode;
- audience/intent;
- forbidden claims;
- source digests.

## UI/UX agent

Recebe:

- user goal;
- interaction contract;
- state ownership;
- backend capabilities realmente existentes;
- empty/loading/error/disabled states;
- accessibility;
- performance constraints;
- evidence/screenshot requirements.

Nunca inferir backend por affordance visual.

# Context tiers

Aplicar progressive disclosure:

- L0 — identity, safety e invariants;
- L1 — task/acceptance contract;
- L2 — immediate implementation context;
- L3 — relevant domain knowledge;
- L4 — optional research/history.

History, superseded e broad corpus não entram por default.

# Context Manifest

Toda compilação de contexto produz manifest reproduzível contendo:

- manifest ID/version;
- task/role;
- source snapshot/revision;
- selected records;
- source IDs/digests;
- excluded stale/superseded records;
- selection reason codes;
- budget requested/used;
- compaction operations;
- skills/agents;
- retrieval queries;
- unresolved unknowns;
- assumptions;
- privacy/egress classification.

# Freshness

Freshness pertence ao claim/record, não apenas ao arquivo.

Triggers possíveis:

- dependency version changed;
- schema/protocol changed;
- source decision superseded;
- implementation changed;
- test/evidence invalidated;
- provider/model changed quando o claim depende disso;
- benchmark environment changed;
- product/profile changed;
- external research age threshold exceeded.

# Documentation Delta

Diff de código/contract gera um delta estruturado:

- changed capabilities;
- changed public surfaces;
- changed internal contracts;
- changed behavior;
- changed failure modes;
- changed NFRs;
- changed compatibility;
- changed security assumptions;
- impacted DocumentationUnits;
- translation/media impact;
- required review.

# Impact-aware rebuild

Usar dependency graph e reverse closure.

Não reescrever toda documentação quando uma Decision muda.

Passos:

1. fingerprint inputs;
2. localizar reverse dependents;
3. recomputar semantic projections afetadas;
4. comparar semantic output digest;
5. parar propagação se resultado semanticamente não mudou;
6. renderizar somente artifacts afetados;
7. validar;
8. atomic write;
9. atualizar provenance por último.

# Managed regions

Quando arquivo mistura humano + generated:

- AST-aware anchors;
- stable IDs;
- ownership hashes;
- generation digest;
- conflict/review se humano alterou região;
- nunca regex cega;
- nunca sobrescrever conteúdo curated sem policy.

# Promotion: Notion → Repository

Pipeline obrigatório:

freeze snapshot → inventory → normalize knowledge → classify authority/status → drift analysis → documentation plan → candidate generation → validate → review → promote.

O snapshot do Notion permanece provenance/historical input após a promoção e sai do retrieval operacional padrão.

# Promotion: Conversation → Canonical

Conversa não vira verdade diretamente.

conversation statement → candidate decision/requirement → confirmation/authority → canonical record → documentation delta.

Para instrução explícita atual do usuário, o sistema pode criar CandidateRecord com authority adequada, mas a promoção ao repository continua registrada e auditável.

# Decision ledger

Toda decisão material deve possuir:

- stable ID;
- semantic title;
- status;
- date/revision;
- owner/authority;
- rationale;
- alternatives;
- consequences;
- supersedes;
- affected capabilities;
- implementation status;
- verification status;
- migration notes;
- source refs.

# Planned vs implemented vs verified

Sempre separar:

- decision_status;
- implementation_status;
- verification_status;
- release_status.

Exemplo válido:

decision_status = ACCEPTED;

implementation_status = PARTIAL;

verification_status = NOT_VERIFIED;

release_status = NOT_READY.

Nunca comprimir em “feito”.

# Documentation Profiles

Perfis ativam contracts, não árvores rígidas.

Composição pode incluir:

- Core Software;
- CLI;
- Agent Harness;
- TUI;
- Desktop GUI;
- Protocol/SDK;
- Compiler/Language;
- Game/DCC;
- Web;
- Service/API;
- Plugin Host;
- Documentation Product.

# UI/UX documentation contract

Além de telas, documentar:

- user goals;
- personas/roles quando úteis;
- state ownership;
- information hierarchy;
- interaction states;
- empty/loading/error/disabled;
- keyboard/focus;
- accessibility;
- hover/click-outside/collapse behavior quando aplicável;
- backend capability dependency;
- latency expectations;
- optimistic/pessimistic updates;
- offline/disconnected behavior;
- conflict behavior;
- evidence/screenshot requirements.

# Machine readability

Preferir JSON/JSON Schema para contracts e manifests.

Markdown continua authoritative prose quando o contract indicar, sempre acompanhado por stable IDs e metadata estruturada suficiente.

# Repository entrypoints

Agents não devem começar por corpus livre.

Entry flow:

[AGENTS.md](http://AGENTS.md) ou [PRUMO.md](http://PRUMO.md) → project manifest → Goal/Task → Implementation Dossier/Context Manifest → exact authoritative docs.

# Search policy

Retrieval deve ponderar nesta ordem conceitual:

1. authority;
2. applicability;
3. freshness;
4. task relevance;
5. evidence;
6. context cost.

Semantic similarity sozinha não define verdade.

# Contradiction Engine

Conflito não deve ser resumido até desaparecer.

Produzir:

- claims conflitantes;
- sources;
- authority;
- status;
- scope;
- possible resolutions;
- affected artifacts;
- migration impact;
- blocking severity.

# Human Docs Planner

Cria Page Graph antes de filesystem tree.

Evita page explosion, preserva audience e Diátaxis intent e mantém navigation como materialized view.

# Agent-facing documentation quality

Métricas mínimas:

- retrieval precision;
- retrieval recall de critical contracts;
- context token cost;
- missing critical contract rate;
- stale-claim rate;
- superseded-claim leakage;
- hallucination-caused defect rate;
- duplicated-context rate;
- context explainability;
- dossier build latency.

# Anti-context-bloat

Não resolver incerteza aumentando corpus indiscriminadamente.

Quando faltar informação:

1. identificar tipo de gap;
2. buscar fonte específica;
3. ampliar somente o tier necessário;
4. parar quando o claim material estiver resolvido.

# Definition of Done

- source layers formalizadas;
- Implementation Dossier implementável;
- microcontexts por role;
- GAP não vira prosa inventada;
- Notion→repo usa promotion;
- planned/implemented/verified separados;
- Context Manifest auditável;
- docs compiler incremental e determinístico;
- retrieval considera authority/freshness;
- agents não precisam ler o Livro Vivo inteiro;
- documentação humana e agent-facing compartilham a mesma base sem compartilhar indiscriminadamente o mesmo contexto.