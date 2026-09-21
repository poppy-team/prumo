# 75 — Fase J: Connector SDK, Capability Contract e Multi-Harness (M11)

> Authority: canonical specification.
> Logical ID: PHASE-75
> Source: Notion Living Book (3d69bb7d023f81b3b21ef0170aeec0fd)
> Status: Fase/Gate de implementação (75 — Fase J Connector SDK, Capability Contract e M).


## Papel no programa

Connector SDK transforma as lições do OpenCode em contrato formal e testável para múltiplos harnesses. O objetivo não é criar wrappers diferentes para cada ferramenta, mas manter **um Atlas Core** com capabilities negociadas e enforcement explicitamente mensurado por connector.

## Fontes

- [10 — Connector SDK, Capability Contract e Roadmap Multi-Harness](../../framework/specs/ch10.md)
- [09 — OpenCode Native Harness](../../framework/specs/ch09.md)
- [11 — Instalação, Atualização, Desinstalação e Distribuição](../../framework/specs/ch11.md)
- [57 — Atlas Package & Runtime Manager, atlas.lock e Provider Isolation](../../framework/specs/ch57.md)
- [61 — Atlas Harness Eval Suite, Canary e Runtime Quality Metrics](../../framework/specs/ch61.md)

## Objetivo

Formalizar:

- Connector Contract versionado;
- host capability vocabulary;
- capability negotiation;
- enforcement levels/report;
- compile/install/update/uninstall lifecycle;
- Connector Test Kit;
- compatibility/migrations;
- package/runtime integration;
- generic fallback;
- prova com um **segundo connector** sem alterar Core semantics.

## Dependencies

OpenCode Native connector concluído e dogfoodado; D3 package/runtime/evals; Tool Gateway/Context/Run/Experience/Quality estáveis o suficiente para contract.

## Non-goals

- public Go SDK/library por princípio;
- suportar todos hosts de mercado em M11;
- esconder diferenças fundamentais entre harnesses;
- fork de hosts;
- reimplementar Control Plane em connectors;
- exigir native integration para usar Atlas.

# 1. Connector definition

Connector é adapter entre semantic Atlas capabilities e primitives concretas de um host/harness.

Ele declara **o que o host consegue fazer**, **como Atlas compila/instala a integração** e **qual enforcement real existe**.

Connector não é agent, Skill, model provider ou Test Provider.

# 2. Connector Contract

Campos mínimos:

- id/version;
- connector schema/protocol version;
- Atlas protocol compatibility;
- host identity/version ranges;
- host primitives/capabilities;
- context injection modes;
- primary/subagent support;
- skills/disclosure support;
- custom command support;
- pre/post tool hooks;
- blocking/approval capability;
- session lifecycle hooks;
- compaction/continuation hooks;
- MCP local/remote support;
- permission model mapping;
- configuration format/scope;
- install strategy;
- cleanup ownership;
- runtime requirements;
- generated artifact templates/compilers;
- capability degradation behavior;
- tests/fixtures;
- provenance/trust/package metadata.

# 3. Capability vocabulary

Stable semantic examples:

- `agents.primary`;
- `agents.subagents`;
- `skills.lazy`;
- `skills.static`;
- `commands.custom`;
- `hooks.session.start`;
- `hooks.session.end`;
- `hooks.tool.pre`;
- `hooks.tool.post`;
- `hooks.compaction`;
- `permissions.block`;
- `permissions.approval`;
- `context.dynamic`;
- `context.static`;
- `mcp.local`;
- `mcp.remote`;
- `events.structured`;
- `continuation.portable` — host consegue consumir/seguir ContinuationRecord/`atlas continue` por CLI/MCP/arquivo mesmo sem lifecycle hook nativo;
- `continuation.native` — connector consegue detectar/abrir `ExecutorSession`, reidratar contexto e registrar session lifecycle automaticamente;
- `handoff.native_or_bridge` — Handoff enriquecido sobre o Portable Continuation Protocol.

Não criar capability para cada detalhe de uma versão específica do OpenCode; vocabulary é semantic/stable, mapping é connector-specific.

# 4. Capability negotiation

Fluxo:

```
Atlas desired capabilities
+ Connector declared support
+ detected host version/runtime
+ project policy
→ Effective Connector Capability Set
→ Enforcement Report
```

Unsupported capability não deve causar silent false confidence. Policy decide se:

- fallback é permitido;
- lower enforcement é aceitável;
- feature fica disabled;
- execution é bloqueada.

# 5. Enforcement levels

Formalizar níveis após OpenCode validation. Modelo recomendado:

- **L0 — Guidance:** prompt/docs only; nenhuma observação/enforcement confiável.
- **L1 — Static Integration:** generated config/agents/commands/context estático.
- **L2 — Lifecycle Aware:** session/context hooks e state bridge.
- **L3 — Observed Execution:** tool/events observáveis; partial guards.
- **L4 — Governed Execution:** pre-action blocking/approval + lifecycle/Tool Gateway integration suficiente para enforcement forte.

Cada connector produz report por capability; connector geral pode ter nível agregado mínimo/summary, mas não esconder que uma capability específica é L2 enquanto outra é L4.

# 6. Connector package

Connector é Package type. Package manifest cuida de distribution/runtime/provenance; Connector manifest cuida de host semantics/capabilities.

`atlas.lock` pode pin connector version + generated protocol compatibility.

# 7. Compiler

`atlas connector compile <id>`/`atlas compile --target <id>` deve:

- resolver host/version/profile;
- render deterministic generated artifacts;
- incluir ownership metadata;
- produzir no-op quando unchanged;
- suportar dry-run/diff quando practical;
- nunca modificar global host config sem install/apply explícito.

# 8. Install lifecycle

```
inspect host
→ compatibility check
→ compile/plan
→ show permissions/config delta
→ explicit install/apply
→ validate/doctor
→ record installation + cleanup manifest
```

Update calcula diff/migration. Uninstall remove owned artifacts/fragments; project canonical state fica intacto.

# 9. Config merge safety

Connector deve conhecer sua propriedade de fragmentos e evitar full-file overwrite. Estratégias possíveis:

- dedicated generated file included pelo host;
- namespaced config section;
- structural JSON/JSONC/YAML merge com expected-current-state;
- backup + ownership markers.

Text replacement frágil só como fallback com conformance fixtures.

# 10. Generic fallback forever

Atlas deve manter caminho de baixo acoplamento mesmo para hosts sem native connector:

- generated instructions/context pack;
- machine CLI;
- `atlas continue --json|--prompt` e Portable Continuation Record;
- local MCP quando host suporta;
- generic LLM/compiler target.

Native connector é melhoria de UX/enforcement, não requisito para protocolo **nem para continuidade básica entre sessões/models/harnesses**.

Todo connector deve provar que seu caminho nativo produz a mesma semantic continuation do generic path: mesma Run, mesmos completed/current/pending facts, mesmas decisões/evidence relevantes e nenhuma duplicação de side effects. Referência: [77 — Portable Continuation Protocol: Continuidade Agnóstica de Session, Model e Harness](phase-77.md).

# 11. MCP universal bridge

Quando `atlas mcp serve` estiver disponível, expor subset estável como status/trace/docs/context/goals/validators/tools conforme security policy. MCP pode ser universal low-coupling bridge, mas connector native ainda é necessário para host lifecycle/hooks/permissions fortes.

Não expor arbitrary Core internals como MCP simplesmente por conveniência.

# 12. Connector Test Kit

Todo connector deve passar, conforme capabilities declaradas:

## Installation

- install idempotency;
- update compatibility;
- cleanup ownership;
- uninstall preservation;
- missing host/version mismatch.

## Compilation

- deterministic output;
- no user config clobber;
- generated ownership markers;
- stable diff.

## Context

- correct project/Goal context;
- context budget obeyed;
- no stale/forbidden sources;
- compaction/continuation behavior when declared.

## Permissions/tools

- blocked write/destructive operation when capability claims block;
- side-effect evidence;
- permission mapping correctness;
- unsupported guard reports lower enforcement.

## Lifecycle

- session start/end + `ExecutorSession` identity;
- Run resume;
- portable continuation across fresh session/model/harness;
- Handoff enrichment;
- connector/host crash recovery;
- version drift;
- conformance entre native continuation e generic `atlas continue` path.

## Compatibility

- min/max host versions;
- protocol mismatch;
- config format version;
- migration fixture.

# 13. Connector conformance manifest

Test results produzem machine-readable conformance report:

- connector/host/protocol versions;
- declared capabilities;
- tested capabilities;
- pass/fail/unsupported;
- enforcement level proven;
- test/evidence refs;
- platform matrix;
- known limitations.

Marketing/README claims não podem exceder proven conformance.

# 14. Second connector proof

Antes de chamar SDK estável, implementar/elevate **um segundo connector** usando contract sem mudar Core semantics.

Candidatos Tier A:

- Claude Code;
- Codex;
- Gemini CLI.

Escolher baseado em primitives/compatibility atuais no momento da implementação. O objetivo é provar generalidade, não necessariamente igualar OpenCode L4.

Se o segundo connector exige mudar uma capability semanticamente OpenCode-specific, revisar vocabulary/contract; não contaminar Core com conditionals de host.

# 15. Connector tiers

Roadmap conceitual:

## Tier A — strategic/native

OpenCode; Claude Code; Codex; Gemini CLI, conforme primitives reais.

Critérios:

- active maintenance;
- useful native primitives;
- lifecycle/tool/context integration;
- Connector Test Kit verde;
- documented enforcement.

## Tier B — important ecosystem

GitHub Copilot, Kiro, VS Code integrations conforme host surfaces.

## Tier C — community/specialized

Cursor, Continue, Cline/Roo, Windsurf, JetBrains e outros conforme demanda.

Tier não é qualidade moral; representa investment/coverage/support target.

# 16. Public Go SDK policy

**Não criar public Go SDK apenas porque a fase se chama Connector SDK.**

Primeiro entregar:

- schemas/contracts;
- manifest format;
- CLI compiler/install lifecycle;
- reference connector helpers internos onde reutilização é real;
- test kit;
- examples/templates.

Extrair pacote Go público somente quando segundo/terceiro connector mostra API estável consumida externamente. Interfaces prematuras são contra Clean Code do projeto.

# 17. Connector authoring UX

Documentar passo a passo:

1. inspect host primitives;
2. choose semantic capabilities;
3. create connector manifest;
4. implement compiler/templates/bridge;
5. declare permissions/runtime;
6. add fixtures;
7. run `atlas connector test <id>`;
8. inspect conformance report;
9. package/sign/distribute;
10. install/doctor/uninstall dogfood.

Criar minimal reference connector/example, não dezenas.

# 18. CLI

Target:

- `atlas connector list`;
- `atlas connector inspect <id>`;
- `atlas connector capabilities <id>`;
- `atlas connector compile <id>`;
- `atlas connector install <id>`;
- `atlas connector update <id>`;
- `atlas connector doctor <id>`;
- `atlas connector test <id>`;
- `atlas connector uninstall <id>`;
- `atlas connector explain <id>`.

Consolidar nomes com surface já existente; evitar aliases redundantes.

# 19. Security

- connector package untrusted até verification/policy;
- host config/tool permissions least privilege;
- connector não recebe secrets por default;
- TS/JS/Python host bridge roda sob declared runtime/environment;
- generated prompts não podem enfraquecer non-disableable Core policies;
- remote connectors/MCP respeitam egress/data classification;
- supply-chain metadata/signatures via Package Manager.

# 20. Compatibility/versioning

Versionar separadamente:

- Atlas protocol;
- Connector Contract schema;
- connector implementation;
- host version/API/config format.

Compatibility matrix e migration fixtures obrigatórias para supported host major changes.

# 21. Goal decomposition

- J-G01: formal Connector Contract schema/capability vocabulary.
- J-G02: capability negotiation + enforcement report.
- J-G03: compile/install/update/cleanup lifecycle common layer.
- J-G04: Connector Test Kit + conformance report schema.
- J-G05: authoring documentation/reference example.
- J-G06: generic fallback/MCP bridge contract review.
- J-G07: OpenCode migration from draft to final Connector Contract.
- J-G08: second connector implementation/elevation.
- J-G09: cross-host Harness Evals + compatibility fixtures.
- J-G10: SDK/API stability review and v0.4/v1 boundary decision.

# 22. Tests/Evals

- contract schema versions;
- unknown capability;
- negotiation/downgrade;
- install/compile determinism;
- config merge conflicts;
- cleanup;
- host missing/old/new unsupported;
- protocol mismatch;
- tool blocking claim verified;
- context injection budget;
- session/handoff/resume;
- connector crash;
- package lock/reproducibility;
- two-host scenario corpus.

Harness metrics compare connector levels and generic fallback: success, context tokens, policy violations, wrong-tool/unnecessary calls, evidence, recovery, installation friction.

# 23. Dogfood

Provar três caminhos no Atlas:

1. OpenCode native connector;
2. segundo connector feito pelo SDK;
3. generic fallback/MCP/`atlas continue` path sem connector nativo.

Todos devem operar sobre o mesmo canonical project/Goal/docs/runtime contracts sem migration específica por host. O dogfood deve incluir uma Run iniciada em um connector nativo e continuada pelo generic path — ou vice-versa — mantendo identidade e estado semântico.

# Exit Gate — CONNECTOR SDK READY

- Connector Contract versionado e host-agnostic;
- capabilities/enforcement são negotiated e honestamente reportadas;
- Test Kit prova claims do connector;
- package/install/cleanup lifecycle comum funciona;
- OpenCode passa final contract sem duplicar Core;
- segundo connector é implementado sem alterar semantic Core;
- generic fallback permanece funcional e suporta Portable Continuation sem connector nativo;
- compatibility/versioning/migration docs completas;
- não foi criado public Go SDK prematuro sem consumidor real;
- Atlas está pronto para expandir ecossistema multi-harness de forma incremental.