# 72 — Fase G: Skill System v3 + Quality Platform

> Authority: canonical specification.
> Logical ID: PHASE-72
> Source: Notion Living Book (3d69bb7d023f81a3a3f2c4f75b9efbbf)
> Status: Fase/Gate de implementação (72 — Fase G Skill System v3 + Quality Platform).


## Papel no programa

Esta fase transforma “skills” de prompts em packages versionados/evaluated e transforma testes externos em Providers normalizados. Skills e Quality passam a depender do Runtime Control Plane em vez de duplicar permissions, context, instalação ou evidence.

## Fontes

- [21 — Skill Package v3, Resolver, Configuração e Evals](../../framework/specs/ch21.md)
- [22 — Cognitive Accessibility, Clean Code e Ambient Skills](../../framework/specs/ch22.md)
- [32 — Test Provider Contract, Quality Orchestrator e Evidence Normalization](../../framework/specs/ch32.md)
- [57 — Atlas Package & Runtime Manager, atlas.lock e Provider Isolation](../../framework/specs/ch57.md)
- [61 — Atlas Harness Eval Suite, Canary e Runtime Quality Metrics](../../framework/specs/ch61.md)

## Objetivo

Entregar:

1. Skill Package v3 + deterministic resolver/config layering;
2. lifecycle/maturity/provenance/permissions/context budgets;
3. Skill Eval Harness + Curator baseline;
4. reference ambient skills;
5. Test Provider Contract;
6. Quality Orchestrator + Evidence normalization;
7. pelo menos um reference provider externo suficientemente útil para validar o contract, sem explodir catálogo.

## Dependencies

- D2 Tool Gateway/permissions/context/model/env;
- D3 Package/Runtime Manager + Harness Evals;
- Documentation System;
- Evidence/Gates;
- Repository Governance.

## Non-goals

- migrar mecanicamente ~98 skills antigas;
- dezenas de Test Providers;
- Game Suite completa;
- full security suite;
- marketplace pública;
- skill auto-improvement sem review/evidence.

# 1. Definição de Skill

> Skill = package versionado de knowledge + procedure + tools + policies + checkers + tests que implementa capability reutilizável para agents.
> 

Skill não é sinônimo de prompt, system message ou agent.

## Package structure conceitual

```
skills/<skill-id>/
├── manifest.json
├── SKILL.md
├── CHANGELOG.md
├── knowledge/
├── workflows/
├── templates/
├── checks/
├── scripts/
├── schemas/
├── tests/
│   ├── fixtures/
│   ├── positive/
│   ├── negative/
│   └── evals/
└── examples/
    ├── good/
    └── bad/
```

Nem todo diretório é obrigatório. Não criar vazio.

# 2. Skill manifest

Campos aceitos/necessários:

- id;
- version/schema_version;
- description;
- maturity;
- activation;
- capabilities;
- requires;
- conflicts;
- permissions;
- knowledge/workflows/checks/scripts/tests refs;
- provenance;
- compatibility;
- context_budget;
- outputs.

Persisted manifest validado por schema.

# 3. Activation

Estados/modes:

- `auto`;
- `contextual`;
- `manual`;
- `required`;
- `disabled`.

Semântica:

- auto/contextual podem ser candidates pelo resolver;
- manual requer explícito selection/request;
- required vem de policy/profile/Goal e não pode ser descartado por heuristic;
- disabled não ativa salvo override autorizado de layer superior quando policy permite.

# 4. Permissions

Skill **declara** permissions; não controla permissions. Tool Gateway/Core decide effective permission.

Uma skill não pode elevar access escrevendo instruction em `SKILL.md`.

# 5. Progressive disclosure

Resolver trabalha metadata-first:

1. manifest/index;
2. candidate selection;
3. carregar [SKILL.md/workflow](http://SKILL.md/workflow) necessário;
4. carregar knowledge/checks on-demand.

Evitar preload de catálogo inteiro no context.

# 6. Lifecycle/maturity

- draft;
- experimental;
- recommended;
- verified;
- deprecated;
- retired.

Promotion exige eval/evidence compatível. Deprecated/retired gera migration guidance.

# 7. Config layering

Baseline aceito:

```
built-in default
→ user
→ workspace
→ project
→ recipe
→ session
```

Effective precedence normalmente:

`session > recipe > project > workspace > user > built-in`, exceto `required`/non-disableable policies que seguem rules explícitas.

Resolver deve conseguir `atlas skill explain <id>` mostrando layers que produziram estado.

# 8. Requires/conflicts

Dependency DAG deve detectar cycle, missing version/capability e conflict. Resolution determinística; conflito não resolvível não escolhe winner arbitrário.

# 9. Bundles

Bundle é convenience/candidate set, não novo authority layer. Ativações individuais ainda passam resolver/permissions/context budget.

# 10. Skill evals

Comparar **baseline sem skill** vs **com skill** quando meaningful.

Metrics:

- task correctness/success;
- contract coverage;
- policy violations;
- activation precision/recall;
- token/context overhead;
- latency/cost;
- false positive/negative checks;
- required output adherence.

Skill não é “verified” só porque texto parece bom.

# 11. Skill Curator

Inicialmente report-only/proposal-only. Detecta:

- overlap/duplicates;
- conflicts;
- stale knowledge;
- context bloat;
- bad activation triggers;
- missing deps;
- eval regression;
- candidates for consolidation/deprecation.

Não auto-delete/consolidate semantic content.

# 12. Ambient Skills

Default-on quando aplicáveis, sem flood de contexto. Reference skills iniciais:

- `clean-code`;
- `cognitive-accessibility`;
- `test-discipline`;
- `go-engineering`.

Possíveis futuras: documentation-engineering, compatibility-awareness, dependency-hygiene, error-design, security-baseline, performance-awareness, observability, ui-accessibility, localization.

Core non-disableable policies como secrets/Goal integrity/canonical state/destructive ops/evidence **não são simplesmente skills**.

# 13. Reference skill acceptance

Cada reference skill precisa:

- manifest;
- focused instructions/knowledge;
- positive/negative fixtures;
- eval baseline;
- activation tests;
- context-budget measurement;
- compatibility metadata.

Não migrar catálogo antigo até essas quatro provarem o Package v3.

# 14. Skill CLI

Target:

- `atlas skill list`;
- `atlas skill inspect/show <id>`;
- `atlas skill enable|disable <id>`;
- `atlas skill status`;
- `atlas skill explain <id>`;
- `atlas skill eval <id>`;
- `atlas skill verify <id>`;
- `atlas skills curate`.

Config mutation segue repository/project/global ownership e Governance.

# 15. Test Provider Contract

Provider de teste declara como uma capability de Quality é instalada/executada/normalizada.

Campos mínimos:

- id/version;
- host/platform requirements;
- install/runtime strategy;
- capabilities;
- input/config schema;
- invocation;
- output parser/normalizer;
- evidence artifacts;
- isolation/environment requirements;
- permissions/egress/secrets;
- timeout/resource model;
- risk/side effects;
- cleanup;
- compatibility.

Package Manager distribui; Test Provider manifest descreve semantics de quality.

# 16. Quality capability taxonomy

Exemplos:

- unit/integration/system;
- web e2e;
- accessibility;
- visual;
- security SAST/DAST/SCA;
- fuzz;
- sanitizer;
- differential;
- GUI automation;
- performance;
- compiler conformance.

Não implementar todas agora.

# 17. Quality Orchestrator

Dado Goal/change/profile/risk:

1. consulta evidence requirements;
2. resolve **minimum sufficient test plan**;
3. seleciona providers disponíveis;
4. verifica environment/budget/permissions;
5. executa Runs/provider calls;
6. normaliza Evidence;
7. aplica failure/flakiness policy;
8. produz Gate result.

Evitar “run every test always”.

# 18. Evidence normalization

Provider-specific outputs convergem em estrutura Atlas contendo:

- provider/version;
- capability/test type;
- subject/revision/environment;
- status;
- assertions/findings summary;
- artifacts/pointers/hashes;
- duration/resources;
- flakiness/retry history;
- raw output reference quando safe;
- trust/provenance.

False pass é critical metric.

# 19. Failure/flakiness

Distinguir:

- product failure;
- test assertion failure;
- provider/tool failure;
- environment failure;
- timeout/resource;
- flaky/unstable;
- inconclusive.

Retry nunca transforma flaky em pass silencioso. Histórico fica em Evidence.

# 20. Independent verifier

High-risk work pode requerer provider/test/model verifier independente. Verifier avalia canonical spec + evidence, não reasoning privado do implementador.

# 21. Reference provider

Implementar primeiro provider externo que prove install/runtime/isolation/parser/evidence. Playwright é forte candidato para web quando isso não alongar indevidamente a fase, pois valida Node runtime isolado; alternativamente um provider simples já disponível no repo pode provar o contract antes.

Regra: **Contract antes de catálogo**.

# 22. CLI Quality

- `atlas test plan`;
- `atlas test run [--profile]`;
- `atlas test providers`;
- `atlas test explain` quando útil;
- machine output/evidence refs.

# 23. Goal decomposition

- G-G01: Skill Package v3 schema/model.
- G-G02: resolver/activation/config/deps/conflicts.
- G-G03: lazy content/context budget integration.
- G-G04: skill eval harness/lifecycle.
- G-G05: four reference skills.
- G-G06: Curator report/proposals.
- G-G07: Test Provider Contract + package integration.
- G-G08: Evidence normalization + failure taxonomy.
- G-G09: Quality Orchestrator minimum test plan.
- G-G10: reference provider + provider conformance kit.
- G-G11: integrated dogfood/evals.

# 24. Tests/Evals

Skills: dependency cycles, conflicts, precedence, required vs disabled, activation precision, lazy load, permission denial, context budget, version compatibility, eval regression.

Providers: install/cleanup, parser malformed output, timeout, environment unavailable, permission denial, flaky retries, artifacts/hashes, evidence stability.

Quality: risk-profile plan, minimum sufficient selection, independent verifier, provider unavailable fallback/no false pass.

# 25. Security

Scripts/checks in third-party skills/providers são untrusted packages. Installation trust + Tool Gateway/Environment/Egress enforce execution. Package signature não concede permissions.

# 26. Dogfood

- Atlas development Runs resolve reference skills;
- clean-code/go-engineering activate somente quando aplicável;
- skill context usage aparece no Context Manifest;
- Quality Orchestrator cria test plan de um PR real;
- reference provider gera normalized Evidence/Gate;
- Curator detecta ao menos synthetic duplicate/conflict fixture.

# Exit Gate — SKILLS & QUALITY FOUNDATION READY

- Package v3 resolve/activation/permissions/context/evals funciona;
- quatro reference skills possuem evidence comparativa;
- catálogo antigo não foi migrado cegamente;
- Test Provider Contract está estável e package-managed;
- Evidence normalization/failure taxonomy impedem false-pass silencioso;
- Quality Orchestrator produz minimum sufficient plans;
- ao menos um external/reference provider passa conformance;
- Harness Evals incluem skill/provider failure scenarios.