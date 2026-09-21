# 89 — Programa de Consolidação e Implementação do Prumo Unificado

> Authority: canonical specification.
> Logical ID: CONST-89
> Source: Notion Living Book (3e29bb7d023f81fcba57c8b97604e8a4)
> Status: Constituição 89: Programa de Consolidação e Implementação do Prumo Unificado.


<aside>
🛠️

**Status: programa de migração e conformance.** Esta página define como transformar a fusão documental em arquitetura implementável sem big-bang rewrite. Cada fase possui gate de saída e deve preservar compatibilidade, evidence, rollback e rastreabilidade.

</aside>

# Objetivo

Consolidar Framework + Harness + Code Agent em uma plataforma coerente, removendo duplicações conceituais e decisões superseded sem apagar história.

A execução deve favorecer vertical slices completos em vez de subsistemas enormes parcialmente implementados.

# Princípios de migração

- preserve before transform;
- classify before delete;
- reconcile before promote;
- contract before implementation;
- deterministic enforcement before prompt;
- one semantic owner per invariant;
- compatibility before convenience;
- evidence before completion;
- small reversible steps;
- migration itself is testable.

# M0 — Freeze e inventário

Entregas:

- snapshot dos dois cadernos pré-fusão;
- lista de páginas/subpáginas;
- decision inventory;
- map de overlaps;
- map de contradictions;
- map planned-vs-implemented;
- alias map de nomenclatura Atlas/Prumo;
- dependency map entre páginas constitucionais e específicas.

Gate:

- nenhuma página órfã;
- todas as decisões materiais referenciáveis por ID/URL;
- histórico preservado;
- nenhum conteúdo deletado apenas por duplicidade aparente.

# M1 — Authority e supersession

Implementar schemas para:

- Claim;
- Decision;
- Assumption;
- Unknown;
- Drift;
- Supersession;
- AuthorityRef;
- FreshnessPolicy.

Migrar conflitos conhecidos:

- Floem desktop → historical/superseded pela direção Oxi;
- referências Atlas antigas → aliases/historical conforme migration;
- caderno separado Code Agent → subdomínio do Livro Vivo unificado;
- proposals antigas que parecem ACCEPTED apenas por linguagem → revisar status.

Gate:

- Context Compiler consegue excluir superseded;
- nenhum current decision depende apenas de ordem textual;
- cada conflito importante possui resolution record.

# M2 — Canonical Documentation Runtime

Entregas:

- Knowledge IR;
- Document IR;
- authority resolver;
- freshness;
- contradiction;
- Implementation Dossier;
- Context Manifest;
- promotion workflow;
- Documentation Delta;
- source map/provenance.

Gate:

- task material recebe dossier pequeno;
- missing data gera GAP/UNKNOWN;
- deterministic rebuild test passa;
- no-op rebuild produz zero writes;
- provenance permite rastrear claim até source.

# M3 — Harness/Agent execution alignment

Entregas:

- NativeAgent state machine;
- AgentProvider boundary;
- ModelProvider boundary;
- Tool/Permission integration;
- checkpoint/resume;
- Handoff;
- cancellation;
- retry taxonomy;
- side-effect journal.

Gate:

- fake provider conformance;
- crash/resume;
- no duplicated side effects;
- state survives client restart;
- cancellation deixa estado conhecido;
- pending permission é retomável.

# M4 — Budget e Model Portfolio

Entregas:

- monthly/workspace/project/Goal/run/agent/step envelopes;
- pricing revisions;
- quota state;
- route policy;
- account pool policy;
- escalation reason;
- review reserve;
- budget checkpoint;
- cost evidence.

Gate:

- profile de R$100 pode hard-stop com checkpoint;
- unknown quota nunca aparece como número fabricado;
- cost report reconcilia observed usage;
- required verification não é silenciosamente removida por economia.

# M5 — Workforce e Skills hardening

Ordem obrigatória:

contracts/schemas/lint → agents → recipes → critical skills → missing capabilities → evals → promotion.

Entregas:

- Agent Contract;
- Skill Package v3;
- Recipe DAG Contract;
- capability registry;
- permission declaration;
- evidence outputs;
- negative activation;
- failure taxonomy;
- handoff contract;
- eval corpus.

Gate:

- resolver explica seleção/rejeição;
- skill outputs críticos têm schemas;
- negative triggers existem;
- least privilege;
- eval baseline;
- nenhuma nova skill existe apenas como prompt rebatizado.

# M6 — Surface unification

## CLI

- ask;
- agent run;
- explain;
- budget/route/context inspection;
- machine-readable output.

## TUI

- Bubble Tea v2;
- consome protocol/services;
- multi-agent projections;
- no duplicated runtime;
- approvals, evidence e budget visíveis.

## GUI

- Oxi-derived Workspace Viewer;
- agent-aware explorer;
- Follow Agent;
- diff/evidence;
- pequenas edições;
- protocol-only state;
- conflitos human×agent explícitos.

Gate:

- Surface Conformance Suite passa;
- same command semantics;
- reconnect/replay;
- client restart não altera Run;
- GUI/TUI não bypassam Permission Engine.

# M7 — Gauntlet/Assurance integration

Entregas:

- surface coverage;
- negative-space tests;
- state-transition testing;
- recovery;
- performance/resource budgets;
- evidence freshness;
- independent review;
- release gates;
- proof-of-use quando profile exigir.

Gate:

- no DONE from agent text;
- release report é derived evidence;
- stale evidence não satisfaz gate;
- known blocker não desaparece em summary.

# M8 — Decision Runtime

Começar com:

1. deterministic rules;
2. heuristics;
3. structured LLM shadow;
4. embeddings/classifier;
5. calibration/abstention;
6. low-risk selective automation.

Nunca iniciar pela automação de maior risco.

Gate:

- abstention;
- risk/coverage;
- reason codes;
- no direct arbitrary side effects;
- provider-neutral;
- fail-safe/conservative fallback;
- shadow evaluation antes de autoridade.

# M9 — Global Learning

Entregas:

- candidate patterns;
- cross-project evidence;
- accept/reject workflow;
- skill/policy proposals;
- freshness/decay;
- project isolation;
- privacy controls.

Gate:

- nenhuma experiência altera policy automaticamente;
- provenance;
- rollback;
- learning proposal pode ser rejeitada sem afetar canonical state.

# M10 — Dogfood

Prumo deve ser gerenciado pelo próprio Prumo.

Cenários obrigatórios:

- architecture change;
- CLI feature;
- provider failure;
- UI change;
- migration;
- budget exhaustion;
- stale docs;
- conflicting decision;
- resume after crash;
- denied permission;
- flaky test;
- partial tool failure;
- external agent handoff;
- documentation promotion;
- protocol version mismatch.

# Required conformance suites

- canonical truth;
- documentation compiler;
- context selection;
- provider;
- native agent;
- external agent;
- tool/permission;
- budget;
- handoff;
- protocol;
- surface;
- workforce;
- skill;
- recipe;
- evidence/quality;
- migration;
- compatibility;
- recovery.

# Vertical Slice Rule

Uma feature só expande para novos adapters/clients depois de atravessar:

contract → implementation → protocol/surface exposure → evidence → documentation.

Isso evita criar cinco UIs para uma capability cuja semântica ainda muda.

# No-big-bang rule

Uma fase nova não exige concluir toda a visão.

Cada work package deve ser:

- pequeno o suficiente para review;
- reversível ou possuir migration;
- testável isoladamente;
- integrável;
- medido;
- documentado.

# Change discipline para agents

Toda implementação derivada deste Livro Vivo deve:

1. citar Goal/Task/Decision IDs;
2. listar files/modules afetados;
3. declarar non-goals;
4. registrar assumptions/unknowns;
5. executar required tests;
6. produzir evidence;
7. atualizar Documentation Delta;
8. registrar drift encontrado;
9. não ampliar scope silenciosamente;
10. não declarar complete antes dos gates.

# Implementation order inside a work package

1. read Dossier;
2. verify repository reality;
3. run baseline tests;
4. make smallest coherent change;
5. run focused tests;
6. run applicable regression;
7. collect evidence;
8. independent/blind-spot review quando requerido;
9. documentation delta;
10. final readiness derivation.

# PR/commit discipline

PR deve conter:

- intent;
- Goal/Task refs;
- Decision/contract refs;
- implementation summary;
- changed surfaces;
- evidence;
- test commands/results;
- budget/performance deltas quando aplicável;
- docs delta;
- known limitations;
- assumptions resolved/unresolved;
- security/migration impact;
- rollback notes quando necessário.

Commits devem ser semanticamente coesos; evitar misturar refactor amplo não necessário com feature.

# Migration safety

Para schema/protocol/state migration:

- compatibility range;
- preflight;
- backup/restore quando aplicável;
- forward migration;
- rollback/roll-forward policy;
- partial failure semantics;
- idempotency;
- version marker;
- old/new client behavior;
- evidence.

# Stopping rule

Uma work package fecha somente quando:

- acceptance coberta;
- hard gates passam;
- required unknowns resolvidos ou formalmente blocked;
- docs delta resolvido;
- evidence fresca;
- no unintended scope;
- budget state registrado;
- recovery/rollback conhecido quando aplicável;
- no unresolved P0 contradiction.

# Implementation status vocabulary

Usar somente estados estruturados:

- not_started;
- planned;
- in_progress;
- partial;
- implemented;
- evidenced;
- verified;
- accepted;
- blocked;
- deprecated;
- superseded.

UI pode usar labels amigáveis, mas deve mapear a esses estados.

# Definition of Done do programa

- existe um único Livro Vivo do Prumo;
- Code Agent preserva suas responsabilidades como subdomínio;
- decisões duplicadas têm precedence/supersession;
- Oxi/TUI/CLI seguem protocol comum;
- budget e workforce são first-class;
- anti-invention é enforcement, não só prompt;
- skills possuem contracts/evals;
- repo consegue operar sem enviar o notebook inteiro ao agent;
- dogfood comprova o fluxo end-to-end;
- migração pode ser reexecutada/auditada;
- nenhuma simplificação documental perdeu informação necessária.

# Amendment 2026-09-21 — Enforcement da consolidação

As fases M1–M7 devem consumir [92 — Directive Compiler: Regras Executáveis para LLMs, Agents, Harness e Surfaces](../contracts/directive-compiler.md) como contrato de compilação e enforcement e [93 — Capability & Skill Gap Register: Lacunas Obrigatórias, Anti-Duplicação e Evals](../harness/capability-skill-gap-register.md) como fonte para classificação de gaps do Workforce.

Novos work packages não podem introduzir regra normativa apenas em prompt, surface ou provider adapter. Quando uma regra afeta authority, permissions, budget, scope, evidence, completion, protocol ou supersession, seu owner deve ser o Core/contract correspondente e a camada LLM recebe somente a projeção necessária.