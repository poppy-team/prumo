# 69 — Fase D3: Runtime Infrastructure — Packages, Migration, Indexing, Automation e Harness Evals

> Authority: canonical specification.
> Logical ID: PHASE-69
> Source: Notion Living Book (3d69bb7d023f811cad83ea0cb6658856)
> Status: Fase/Gate de implementação (69 — Fase D3 Runtime Infrastructure — Packages, Mi).


## Papel no programa

D3 transforma o Control Plane em plataforma extensível/reprodutível. Packages/providers ganham lifecycle e lock; schemas evoluem por migration; repos grandes são indexados incrementalmente; automações reutilizam Run safety; e heurísticas do harness passam a ser julgadas por evals/canaries.

## Fontes

- [56 — Automation Engine, Event Rules, DLQ e Safety](../../framework/specs/ch56.md)
- [57 — Atlas Package & Runtime Manager, atlas.lock e Provider Isolation](../../framework/specs/ch57.md)
- [58 — Migration Engine, Unified Review Queue e State Evolution](../../framework/specs/ch58.md)
- [59 — Repository Scale, Incremental Indexing, Branches, Monorepos e Concurrency](../../framework/specs/ch59.md)
- [61 — Atlas Harness Eval Suite, Canary e Runtime Quality Metrics](../../framework/specs/ch61.md)

## Objetivo

Entregar seis fundações horizontais:

1. Package/Runtime Manager + `atlas.lock`;
2. Migration Engine + Unified Review Queue;
3. incremental Repository Index + branch/monorepo/concurrency model;
4. safe event-driven Automation foundation;
5. **Continuous Execution Program Runner** — autonomia opt-in `goal|phase|program`, worker dispatch e bounded test-fix loop;
6. Harness Eval Suite + canary/baseline promotion rules.

A especificação detalhada de Continuous Execution está em [78 — Continuous Execution Mode: Program Runner, Worker Loop e Autonomia Opt-in](phase-78.md).

## Dependencies

D2 completo; Run Engine, Budget, Tool Gateway, Environment, Observability e Egress disponíveis.

## Non-goals

- package marketplace pública;
- distributed scheduler/server;
- cloud artifact registry obrigatório;
- autonomous cron farm;
- every provider/runtime;
- vector DB como default.

# 1. Atlas Package Contract

Package representa distribuição instalável de extension capability. Tipos iniciais:

- skill;
- connector;
- test-provider;
- experience-provider;
- publication-adapter.

Package manifest mínimo:

- id/version/type;
- protocol compatibility;
- platform/architecture requirements;
- dependencies;
- permissions;
- runtime requirements;
- entrypoints/resources;
- checksums/signature/provenance;
- install strategy;
- cleanup ownership;
- capabilities;
- trust/maturity where applicable.

Manifest de package não substitui manifest interno do Skill/Test Provider; ele descreve distribution/lifecycle.

# 2. atlas.lock

Objetivo: reproducibility do conjunto externo ativo.

Lock registra resolution exata de packages/runtimes/providers sem guardar secrets. Deve ser deterministicamente regenerável a partir de config + registry/package metadata.

Características:

- versions/hashes;
- transitive deps se existirem;
- platform-specific resolution explicitada;
- protocol compatibility;
- source/provenance;
- no timestamps voláteis que causem diff inútil.

`atlas.lock` deve ser canonical quando representa o ambiente de projeto escolhido; caches/downloads não são.

# 3. Runtime isolation

Runtimes auxiliares ficam, quando possível, em:

`$ATLAS_HOME/runtimes/<package-or-provider>/<version>/...`

Ex.: Playwright pode usar Node/browser downloads isolados sem tornar Node dependency do Atlas Core.

Manager deve saber diferença entre:

- Atlas binary instalado por Homebrew/WinGet/script;
- provider runtime gerenciado pelo Atlas;
- project-local package state.

Não remover arquivo que não pertence ao Atlas.

# 4. Package CLI

Target:

- `atlas package list`;
- `atlas package inspect <id>`;
- `atlas package install <id>`;
- `atlas package update <id>`;
- `atlas package remove <id>`;
- `atlas provider install <id>`;
- `atlas runtime list`;
- `atlas runtime gc`;
- `atlas repair`;
- lock check/update flows coerentes.

Read-only plan antes de destructive mutation quando possível.

# 5. Supply chain

Antes de install:

- verify checksum;
- signature/provenance quando policy exige;
- declared permissions;
- runtime/platform compatibility;
- source trust;
- protocol compatibility.

Package não ganha permissions implícitas por estar assinado.

Offline bundle future/baseline: permitir export/import de artifacts verificados para ambientes sem internet, sem exigir implementation completa de registry.

# 6. Migration Engine

Todo schema/state evolution persistido deve usar Migration Contract:

- id/version;
- from/to versions;
- affected artifacts;
- preconditions;
- backup/snapshot strategy;
- transform;
- post-validation;
- rollback;
- reversibility flag;
- evidence;
- canonical paths affected.

## Migration workflow

```
inspect
→ plan/dry-run
→ backup/snapshot
→ migrate
→ validate
→ record migration journal
→ commit/propose canonical delta
```

Falha pós-transform deve executar rollback quando contract oferece; senão bloquear e preservar recovery evidence.

Fixtures de versões antigas são obrigatórias.

# 7. Unified Review Queue

Unificar proposals que precisam decisão humana/policy:

- Experience Proposal;
- Skill Proposal;
- Documentation contradiction/delta;
- security waiver;
- migration proposal;
- publication proposal;
- outros structured proposals.

Queue não torna todos equivalentes semanticamente; oferece lifecycle/CLI/review primitives comuns.

Target CLI:

- `atlas review list`;
- `atlas review show <id>`;
- `atlas review accept|reject <id>` quando policy permite.

Approval é event/evidence, não flag invisível.

# 8. Incremental Repository Index

Objetivo: repos grandes não podem ser rescaneados integralmente a cada command.

Index derivado usa:

- content hashes;
- Git diff/tree/revision;
- path classification;
- generated/vendor/binary exclusions;
- sharding/batching quando necessário;
- parser/index version.

SQLite stale nunca supera canonical source. Se index revision/hash diverge, recompute/mark partial.

# 9. Scan budgets

Index/scanner respeitam Budget Envelope:

- max files/bytes/time;
- partial result explicit;
- confidence/completeness metadata;
- continuation/resume.

Nunca retornar scan parcial como “complete” silenciosamente.

# 10. Branch awareness

Distinguir:

- canonical mainline state;
- current branch/worktree state;
- proposal/diff state.

Feature branch docs/Experience não viram verdade de main antes do merge.

Indexes podem ser revision-scoped.

# 11. Monorepo model

Conceito:

```
Repository
└── Workspace
    ├── Project A
    ├── Project B
    └── Project C
```

Não presumir 1 repo = 1 Atlas project. Discovery/config define boundaries explícitos.

# 12. Concurrency baseline

Antes de team server, suportar local coordination suficiente:

- Run/Task lease;
- file/workspace claim metadata;
- worktree isolation;
- patch proposal/merge;
- conflict detection;
- expiry/recovery de lease.

Não construir distributed consensus.

# 13. Automation Engine

Modelo:

```
event → rule → condition → actions → evidence
```

Automation Contract:

- id/version;
- trigger;
- conditions;
- actions;
- approval policy;
- timeout;
- retry;
- concurrency key/limit;
- budget;
- execution environment;
- failure/DLQ behavior;
- ownership.

Automation reutiliza Run Engine/failure taxonomy/Tool Gateway; não implementa segundo executor.

## Automation ≠ Program Runner

Automation responde **quando** disparar uma execução a partir de evento/regra. Program Runner responde **como** executar continuamente um conjunto de Goals até um Exit Gate. O primeiro pode iniciar o segundo, mas não devem duplicar state machine, retry, budgets ou worker semantics.

## Safety

- idempotency key;
- cancellation propagation;
- cleanup;
- no retry cego de side effects;
- DLQ para failure persistente;
- max recurrence/loop protection;
- budget por invocation/window.

## CLI

- `atlas automation list`;
- `atlas automation explain <id>`;
- `atlas automation failures`;
- `atlas automation retry <id>`.

Scheduler/time-based avançado só depois do event-driven estável; automação não precisa de daemon/cloud obrigatório.

# 14. Harness Eval Suite

O Atlas precisa testar o **harness**, não só unit code.

Scenario corpus mínimo:

- small task overhead;
- large 100K LOC repository;
- long-run context pressure/compaction;
- crash + resume;
- schema migration resume/rollback;
- provider 429/timeout/5xx;
- ambiguous side effect;
- malicious README/tool output;
- conflicting skills;
- stale docs;
- flaky tests;
- budget exhaustion;
- connector crash;
- model alias drift;
- concurrent agents/worktrees.

## Metrics

- task success;
- policy violations;
- context precision/recall;
- tokens/cost per task;
- latency;
- unnecessary tool calls;
- wrong-tool rate;
- skill activation precision/recall;
- resume/recovery success;
- evidence completeness;
- false-pass rate;
- diagnosis quality;
- blast-radius containment.

Heurística grande (Context Compiler, router, workforce, compaction) só é promovida contra baseline.

# 15. Canary

Nova model alias/provider/heuristic/connector version pode rodar em shadow/canary scenarios antes de default promotion. Failure reverte routing/config, não canonical project data.

# 16. Goal decomposition

- D3-G01: PackageContract + install ownership.
- D3-G02: dependency resolution + atlas.lock.
- D3-G03: runtime isolation + repair/gc.
- D3-G04: MigrationContract + dry-run/rollback/journal.
- D3-G05: Unified Review Queue.
- D3-G06: incremental index + scan budgets.
- D3-G07: branch/monorepo/lease baseline.
- D3-G08: AutomationContract + event execution + DLQ.
- **D3-CE01:** ExecutionPolicy + scopes `manual|goal|phase|program`.
- **D3-CE02:** ProgramRun + next-ready Goal resolver.
- **D3-CE03:** WorkUnit/WorkerResult + worker dispatch boundary.
- **D3-CE04:** bounded validation/test-fix repair loop + livelock/budget enforcement.
- **D3-CE05:** OpenCode/reference programmatic driver + generic headless process driver.
- **D3-CE06:** crash/resume + Portable Continuation + multi-Goal dogfood.
- D3-G09: Harness Eval schema/corpus/runner, incluindo continuous-execution scenarios.
- D3-G10: canary/promotion policy + dogfood.

# 17. Tests

Package checksum/provenance/permissions, lock determinism, incompatible protocol, interrupted install cleanup, migration from every fixture version, rollback, stale index, partial scan, branch divergence, monorepo boundaries, lease conflict/expiry, automation idempotency/retry/DLQ, eval reproducibility.

# 18. Dogfood

- Install ao menos um reference package via Package Manager;
- lock environment;
- migrate uma fixture Atlas antiga;
- indexar o próprio repo incrementalmente após commit change;
- executar uma automation event-driven segura;
- rodar harness eval baseline e armazenar result/evidence.

# Exit Gate — RUNTIME INFRASTRUCTURE READY

- package lifecycle e lock são reprodutíveis;
- runtimes auxiliares isolados sem contaminar Core;
- migrations versionadas, dry-run/tested/rollback-aware;
- review queue funcional;
- repo index incremental/branch-aware;
- local concurrency não corrompe trabalho paralelo;
- automation reutiliza Run safety + DLQ;
- Continuous Execution é explicitamente opt-in, Program Runner controla Goal→tests→repair→gate→próximo Goal e respeita approvals/budgets/Repository Governance;
- program mode sobrevive a crash e troca de worker via Portable Continuation;
- Harness Eval Suite executa corpus baseline e governa promotion de heurísticas.