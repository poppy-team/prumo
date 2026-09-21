# 84 — Total Assurance Constitution: Gauntlet Loop, Evidence e Anti-False-Green

> Authority: canonical specification.
> Logical ID: 84
> Source: Notion Living Book (3e29bb7d023f817ca077d37f62038fd1)
> Status: Constituição 84: Perfis do Gauntlet / Total Assurance (84 — Total Assurance Constitution Gauntlet Loop, E).


<aside>
🛡️

**Status:** norma transversal obrigatória do Prumo. Este programa endurece Documentation Contracts, Quality Orchestrator, Workforce, Control Plane e release governance para impedir que uma run seja aprovada por compilação, aparência ou autoavaliação do agente. **Evidence before green; proof of use before release.**

</aside>

# Propósito

Transformar o rigor aplicado aos Gauntlet Loops de produto em uma capacidade nativa do Prumo. O framework deve cercar sistematicamente os pontos em que LLMs e code agents costumam falhar: implementação parcial, UI cenográfica, botões mortos, testes superficiais, falso sucesso, drift documental, regressões cruzadas, refactors frágeis, segurança declarativa, performance sem baseline, evidência stale, validação feita pelo próprio implementador e encerramento prematuro.

Esta página complementa e endurece [03 — Documentation Contracts, Profiles e Qualidade Documental](../framework/specs/ch03.md), [04 — UI/UX Documentation Pack Completo](../framework/specs/ch04.md), [13 — Testes, Conformance, Evals e Quality Gates](../framework/specs/ch13.md), [32 — Test Provider Contract, Quality Orchestrator e Evidence Normalization](../framework/specs/ch32.md), [55 — Observability Plane, Run Explainability e Incident Bundles](../framework/specs/ch55.md), [79 — Programa de Aprimoramento do Workforce: Agents, Skills, Recipes e Quality Contracts](../skills/specs/programa-de-aprimoramento-do-workforce-agents.md) e [63 — Programa de Implementação v0.4: do Go Core ao Connector SDK](../development/phases/phase-63.md).

# Nova regra constitucional

**DONE não é um texto produzido por um agent. DONE é um estado derivado de cobertura + evidência válida + ausência de blockers + verificação independente adequada ao risco.**

O Prumo deve distinguir:

- **implemented**: existe código real;
- **reachable**: existe caminho real para o usuário/sistema acessar;
- **exercised**: o caminho foi realmente executado;
- **evidenced**: há artefato suficiente para provar o comportamento;
- **verified**: evidence foi validada por gate/verifier apropriado;
- **accepted**: acceptance contract satisfeito;
- **released**: todos os release gates aplicáveis foram satisfeitos.

Nenhum desses estados implica automaticamente o seguinte.

# Princípio anti-falso-verde

Compilação prova apenas que uma configuração compilou.

Teste unitário prova apenas o comportamento coberto pelo teste.

Screenshot prova apenas um frame.

UI presente não prova interação.

Command presente não prova reachability.

Benchmark isolado não prova regressão.

LLM afirmar “implementado” não é evidence.

Retry até passar não transforma flaky em stable.

Ausência de crash não prova correção.

Cobertura de linhas não prova cobertura de requisitos.

Média alta não pode esconder um gate crítico reprovado.

# Universal Assurance Graph

Toda capability relevante deve ser rastreável por um grafo:

Requirement

→ Acceptance Criterion

→ Interaction/API Surface

→ Implementation Owner

→ Invariant

→ Test/Probe

→ Evidence Artifact

→ Verification

→ Regression Fixture

→ Documentation

→ Release Gate

Qualquer nó required sem ligação suficiente gera **coverage_gap**.

# Interaction/Surface Manifest universal

Projetos com superfícies interativas ou públicas devem possuir inventário machine-readable proporcional ao domínio. Entradas possíveis:

- UI controls, menus, modals, commands, shortcuts e gestures;
- CLI commands, flags, exit codes e prompts;
- API endpoints, RPCs, events e webhooks;
- file formats, importers/exporters e migrations;
- plugin/extension hooks;
- schemas e config keys;
- user-visible errors;
- background jobs e automations;
- security boundaries;
- performance-critical flows.

Cada item deve declarar, quando aplicável: ID estável, owner, entry point, preconditions, expected behavior, side effects, persistence, error behavior, permissions, undo/recovery, tests, evidence e current status.

# Gauntlet Loop como protocolo

O Prumo deve oferecer um loop explícito:

Discover

→ Inventory

→ Risk classify

→ Plan evidence

→ Implement

→ Build

→ Static checks

→ Unit/property tests

→ Integration/contract

→ Adversarial/negative

→ System/E2E

→ Performance/resource

→ Security

→ Accessibility/UX/visual quando aplicável

→ Manual/proof-of-use

→ Independent verification

→ Regression

→ Documentation delta

→ Score/gate

→ Repair

A quantidade de loops NÃO é fixa. A saída é governada por gates.

# Hard gates versus score

O sistema deve separar:

- **hard gate:** binário; não pode ser compensado;
- **quality score:** informativo/comparativo;
- **risk budget:** tolerância explícita;
- **waiver:** exceção temporária governada;
- **inconclusive:** evidência insuficiente.

Para critical scope, 9.9/10 não é sinônimo de release. Um único hard gate falho mantém o Goal incompleto.

# Evidence freshness e binding

Evidence precisa estar vinculada a:

- source revision/commit;
- config/policy hash;
- environment/toolchain fingerprint;
- relevant artifact revision;
- test selection;
- provider/model version quando probabilístico;
- timestamp;
- limitations/skips.

Mudança material invalida ou torna stale evidence dependente. O Prumo deve construir um **Evidence Dependency Graph**, em vez de tratar “teste passou ontem” como verdade eterna.

# Independent verification

O mesmo agent pode realizar checks locais de baixo risco, mas não deve autoaprovar mudanças high/critical sem policy explícita. Separar:

- implementer;
- verifier;
- security reviewer;
- UX/visual reviewer;
- performance reviewer;
- release authority.

Least Workforce continua válido: independência é acionada por risco, não por burocracia universal.

# Blind-spot hunting

Cada Gauntlet deve incluir uma fase que pergunta não apenas “o que está especificado?”, mas “que classe de falha razoável ficou fora da especificação?”. Fontes:

- diff arquitetural;
- threat model;
- state machine;
- public surface manifest;
- dependency graph;
- platform matrix;
- persistence model;
- concurrency model;
- resource model;
- import/export boundaries;
- human interaction flows;
- upgrade/downgrade lifecycle.

Descobertas viram requirement amendment, risk register, test fixture ou **not-applicable** justificado.

# Oracle problem

O Prumo deve registrar **quem/como sabe que o resultado está correto**. Para cada classe:

- exact oracle: schema, golden, deterministic expected state;
- relational oracle: invariants/metamorphic properties;
- differential oracle: comparação com implementação/reference;
- human oracle: review estruturado;
- probabilistic oracle: eval rubric + repeated runs + confidence;
- no trusted oracle: **inconclusive**, nunca auto-pass.

# Test quality gate

Um teste também pode estar errado. Revisar:

- assertion strength;
- mock excessivo;
- tautological tests;
- snapshot cego;
- tests que reproduzem a implementação em vez do contrato;
- fixtures irreais;
- retries mascarando falha;
- tests sem failure sensitivity;
- mutation testing quando custo/risco justificar.

# Negative-space validation

Validar ausência de comportamentos indevidos:

- side effect não autorizado;
- arquivo não deveria ser alterado;
- secret não deveria aparecer;
- command disabled não deveria executar;
- cancel não deveria sujar history;
- read-only não deveria mutar;
- failed save não deveria marcar clean;
- UI-only state não deveria sujar document;
- denied action não deveria ter feito mutation parcial.

# State-transition assurance

Para stateful software, documentar e testar:

actor + precondition + previous revision + mutation + side effects + invariants + rollback/cancel + failure state.

Abordagens de state machine devem ser usadas onde bugs de sequência importam.

# Fault model

Não testar só happy path. Modelar falhas relevantes:

- timeout/cancellation;
- dependency unavailable;
- disk full;
- permission denied;
- corrupted cache;
- malformed/corrupt input;
- partial write;
- process crash;
- OOM/resource exhaustion;
- concurrent mutation;
- provider 429/5xx;
- network partition;
- stale state;
- clock/time skew quando aplicável;
- GPU/device reset em workloads gráficos;
- unsupported platform/driver.

# Recovery as a first-class requirement

Para mudança relevante, perguntar:

- pode cancelar?
- pode resumir?
- pode retry sem duplicar side effect?
- pode rollback?
- pode restaurar de backup/checkpoint?
- comportamento após crash é conhecido?
- partial state é detectável?

Recovery evidence pode bloquear release.

# Performance correctness

Performance é parte da correção quando existe user/resource budget. O Prumo deve suportar budgets para:

- latency;
- throughput;
- startup/shutdown;
- frame time/frame pacing;
- CPU;
- RAM;
- VRAM;
- allocations;
- I/O;
- disk/cache;
- network;
- binary/bundle size;
- thermal/power quando aplicável;
- long-session degradation.

Toda otimização exige baseline, workload, ambiente, repetitions/variance quando relevante e comparação antes/depois.

# Degradation e endurance

Adicionar classes esquecidas por benchmarks curtos:

- memory leak;
- fragmentation;
- cache growth;
- handle/file descriptor leak;
- GPU resource leak;
- queue growth;
- log growth;
- repeated open/close;
- workspace/context switching;
- long-running session;
- repeated Undo/Redo;
- cancellation storms;
- project size scaling.

# Security beyond SAST

Security gate pode compor:

threat model + code review + SAST + SCA + secret scan + malformed/adversarial fixtures + fuzz + sandbox + permission checks + filesystem/path tests + supply-chain provenance + release signing, conforme profile.

# Human-facing quality

Para user-facing software, o Gauntlet deve poder exigir:

- task completion;
- discoverability;
- interaction-state coverage;
- keyboard/focus;
- accessibility;
- localization/pseudo-locale;
- responsive/DPI;
- visual QA;
- visual regression;
- error recovery;
- first-run flow;
- no dead control;
- no misleading affordance;
- performance perception.

# Documentation assurance

Documentation Contract passa a avaliar também:

- behavioral completeness;
- negative behavior;
- failure/recovery;
- NFR budgets;
- security assumptions;
- performance baselines;
- state ownership;
- lifecycle;
- compatibility/migration;
- observability;
- debugging/support;
- test oracles;
- evidence pointers;
- known unknowns;
- post-release signals.

# Scope closure

O Gauntlet deve impedir dois erros opostos:

1. encerrar cedo com requisitos incompletos;
2. aumentar escopo indefinidamente tentando “perfeição”.

10/10 é relativo ao **escopo aceito**, não significa feature creep. Recursos explicitamente pós-MVP permanecem fora.

# Waiver Contract

Waiver nunca é “ignorar”. Deve possuir:

ID, requirement/gate, rationale, risk, compensating controls, owner, approver, created_at, expiry, affected releases e remediation link.

Waiver expirada bloqueia novo release quando policy determinar.

# Stopping rule

Uma run pode encerrar somente quando:

- todos os required acceptance criteria têm evidence válida;
- nenhum hard gate requerido falha;
- nenhum required item está untested/unknown;
- blockers conhecidos foram resolvidos ou formalmente blocked_external;
- waivers são válidas e autorizadas;
- regression set aplicável passa;
- documentation delta foi resolvido;
- release/readiness policy permite.

# Relação com o restante do Livro Vivo

Esta constituição deve ser consumida por Documentation Contracts, UI/UX Documentation Pack, Test Provider Contract, Quality Orchestrator, Harness Eval Suite, Workforce Quality, Architecture Quality, Repository Governance, Observability, Security e Programa de Implementação. Ela não duplica esses módulos; fornece o **metacontrato de completude e prova** que os une.

# Diretriz de implementação

Antes de adicionar heurísticas de LLM:

1. schema;
2. manifest;
3. deterministic resolver;
4. evidence normalization;
5. hard gates;
6. fixtures;
7. only then probabilistic assistance.

O agente pode descobrir, explicar e propor. O Core decide invariantes verificáveis.

# Definition of Done desta norma

- existe modelo universal de surface/interaction coverage;
- Goal completion consome acceptance/evidence graph;
- Quality Orchestrator conhece pass, fail, flaky, inconclusive, blocked_external e waived;
- evidence possui freshness/dependency semantics;
- Gauntlet Loop é reexecutável e resumível;
- regression selection é impact-aware;
- blind-spot review é parte do protocolo;
- proof-of-use pode ser exigido por profile;
- hard gates não são compensados por score;
- release report é derivado de evidence, não de autoavaliação do agent.

[84.A — Universal Surface Coverage, Evidence Graph e Proof-of-Use](gauntlet-84_a.md)

[84.B — Blind-Spot Hunting, Oracle Quality, Negative Space e Adversarial Assurance](gauntlet-84_b.md)

[84.C — Performance, Resource Budgets, Scalability, Endurance e Efficiency Assurance](gauntlet-84_c.md)

[84.D — Security, Resilience, Data Integrity, Recovery e Failure Containment Assurance](gauntlet-84_d.md)

[84.E — UI/UX, Interaction Fidelity, Accessibility e Human-Factor Assurance](gauntlet-84_e.md)

[84.F — Documentation Refactoring, Drift, NFR Coverage e Living Truth Assurance](gauntlet-84_f.md)

[84.G — Release Candidate Exit Gates, 10/10 Sem Média e Continuous Assurance](gauntlet-84_g.md)

[84.H — Native Graphics/DCC Assurance Profile: GPU, Geometry, Assets e Long Sessions](gauntlet-84_h.md)