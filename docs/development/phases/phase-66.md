# 66 — Fase C: Documentation System v2 (M5)

> Authority: canonical specification.
> Logical ID: PHASE-66
> Source: Notion Living Book (3d69bb7d023f8106a544fd0fd1dbb7cb)
> Status: Fase/Gate de implementação (66 — Fase C Documentation System v2 (M5)).


## Papel no programa

Transformar documentação em um subsistema de engenharia mensurável e governado. A partir daqui o Atlas passa a saber **qual conhecimento é necessário**, não apenas a gerar Markdown.

## Fontes

- [03 — Documentation Contracts, Profiles e Qualidade Documental](../../framework/specs/ch03.md)
- [62 — Repository Change Governance, GitHub Policy e Agent SCM Safety](phase-62.md)
- [17 — Estrutura Canônica de Repositório Alvo](../../framework/specs/ch17.md)

## Objetivo

Implementar em Go:

- Documentation Contract;
- Documentation Profile;
- applicability;
- binding;
- coverage;
- general/Goal readiness;
- impact;
- Documentation Delta;
- staleness;
- contradiction findings;
- UI/UX documentation pack de referência.

## Regra central

O **Core** decide deterministically o que é requerido/aplicável/bloqueante. LLM pode propor conteúdo e semantic findings, mas não é autoridade para invariants, readiness crítico ou promoção canônica.

## Coverage states canônicos

- `missing`;
- `partial`;
- `implementation-ready`;
- `verified`;
- `stale`;
- `not-applicable`.

`implementation-ready` é scoped ao conhecimento necessário; não significa “documentação perfeita”.

## Documentation Contract

Campos semânticos mínimos:

- id/version;
- semantic role;
- applicability;
- required knowledge items;
- blocking questions;
- evidence requirements;
- update triggers;
- eligible owners;
- staleness policy;
- canonical source pointers/binding expectations.

Contract não é filename. O gate avalia informação.

## Profiles

Profiles compõem Contracts conforme capacidades do projeto. Famílias aceitas incluem Core Software, CLI, Desktop GUI, Web, Mobile, API/Service, Data Platform, Compiler, Programming Language, Game Engine, Graphics, Plugin Host, Library e Distributed System.

Preferir composição a inheritance profunda.

## Binding

Representar Contract ↔ canonical source(s), permitindo:

- um documento satisfazer vários contracts;
- um contract usar múltiplos documentos;
- ownership/authority explícita;
- sources externas como evidence/reference, não canonical por padrão.

## Coverage Engine

Avalia knowledge item por knowledge item. Percentual pode existir como resumo, mas blockers semânticos têm prioridade.

## Readiness Engine

Separar:

- **general readiness**: maturidade fundacional do projeto;
- **Goal readiness**: documentação suficiente para um Goal específico.

Uma ausência irrelevante ao Goal não pode bloqueá-lo.

## Goal gate

Readiness deve alimentar Gate/Evidence existente, sem redesenhar Goal lifecycle desnecessariamente.

## Impact Engine

Inputs:

- Goal;
- changed paths/schemas/capabilities;
- events;
- contract update triggers.

Output:

- contracts/documents possivelmente afetados;
- reason;
- severity/confidence quando aplicável.

## Documentation Delta

Objeto/proposta estruturada: source Goal/Run, affected contracts/docs, reasons, required changes, evidence e lifecycle mínimo (`proposed`, `reviewed`, `accepted`, `rejected`, `applied`, ou equivalente consistente com Review Queue).

Nunca promover patch gerado por model diretamente a canonical.

## Contradictions

Classes:

- syntactic;
- structural;
- semantic;
- authority conflict;
- version conflict.

Deterministic contradictions são Core. Semantic prose contradictions podem usar provider futuro, sempre como finding com confidence.

Resolution considera authority; “arquivo mais recente ganha” é proibido.

## Staleness

Preferir causal staleness por update trigger/dependency a `idade > N dias`. Time-based pode complementar.

Políticas mínimas: never, trigger-based, time-assisted/manual-verification ou equivalentes simples.

## UI/UX Pack

Contracts condicionais para IA/navigation/flows/screens/tokens/typography/colors/spacing/icons/components/layout/states/interactions/accessibility/localization/themes/visual regression. Structured JSON pode complementar Markdown quando melhora tooling.

Não exigir pack completo em todo GUI indiscriminadamente.

## CLI

- `atlas docs contracts [list|show]`;
- `atlas docs profiles [list|show]`;
- `atlas docs audit`;
- `atlas docs readiness [--goal]`;
- `atlas docs impact`;
- `atlas docs delta`;
- `atlas docs contradictions`.

Inspection é read-only; mutation/proposal é explícita. Machine output segue envelope Atlas.

## Schemas/protocol objects

Persisted/interchange candidates:

- DocumentationContract;
- DocumentationProfile;
- DocumentationBinding;
- CoverageReport;
- ReadinessReport;
- DocumentationDelta;
- DocumentationFinding.

Não criar schema para Go struct puramente interna.

## Resources

Built-in contracts/profiles devem ser data-driven e embeddable. Começar com conjunto pequeno de alta qualidade em vez de centenas de contracts.

Reference contracts iniciais: vision, scope/non-goals, glossary/domain, architecture, component contracts, data/state, runtime/error, dependencies, compatibility, performance, security, testing, risks/assumptions/limitations/debt, migration e installation.

Reference profiles iniciais suficientes para provar composição: core-software, cli, web, desktop-gui, api-service, compiler, library.

## Goal decomposition

- C-G01: Contract model/schema/registry.
- C-G02: Profile composition + applicability.
- C-G03: Binding + coverage engine.
- C-G04: general + Goal readiness.
- C-G05: impact + update triggers.
- C-G06: Documentation Delta.
- C-G07: contradiction + authority.
- C-G08: staleness.
- C-G09: UI pack + Atlas dogfood.

## Tests

Contract/profile/version/duplicate/conflict; all coverage states; applicability; unrelated missing docs not blocking Goal; blocker question; impact dedupe; delta lifecycle; deterministic contradictions; causal staleness; machine output ordering; integration fixtures para CLI/GUI/compiler scenarios.

Live LLM não é permitido em unit/conformance tests.

## Security

Repository prose é untrusted data. Textos contendo comandos/prompts não viram policy. Somente structured Atlas contracts/config controlam deterministic enforcement.

## Dogfood

Atlas recebe seu próprio profile/bindings e executa `atlas docs audit` + `atlas docs readiness`. O resultado deve reportar gaps honestamente; não marcar docs complete só para passar o Gate.

## Exit Gate — M5 READY

- contracts/profiles versionados;
- applicability/binding/coverage deterministicamente funcionais;
- Goal readiness bloqueia apenas conhecimento relevante;
- impact/delta/staleness/contradiction ativos;
- UI pack representado;
- conformance + tests verdes;
- Repository Governance governa patches documentais;
- Atlas dogfood coerente com estado real.