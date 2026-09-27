# 84.A — Universal Surface Coverage, Evidence Graph e Proof-of-Use

> Authority: canonical specification.
> Logical ID: 84 A
> Source: Notion Living Book (3e29bb7d023f81a0a812c169ae1e5151)
> Status: Constituição 84: Perfis do Gauntlet / Total Assurance (84 A — Universal Surface Coverage, Evidence Graph).


<aside>
🧾

**Objetivo:** impedir que qualquer superfície pública ou interativa desapareça entre documentação, implementação e teste.

</aside>

# Surface Registry

Cada projeto/profile pode declarar um Surface Registry. Tipos iniciais: ui-control, ui-flow, cli-command, cli-flag, api-endpoint, rpc-method, event, file-format, importer, exporter, config-key, plugin-hook, background-job, migration, public-library-api, security-boundary e performance-critical-flow.

Campos mínimos por superfície:

- stable_id;
- semantic role;
- owner;
- source pointers;
- entry points;
- preconditions;
- permissions;
- valid states;
- invalid states;
- behavior;
- side effects;
- negative guarantees;
- persistence effect;
- cancel/recovery behavior;
- compatibility contract;
- accessibility/i18n quando aplicável;
- expected evidence;
- regression fixture;
- status.

# Coverage lattice

Estados independentes:

- documented;
- implemented;
- reachable;
- exercised;
- evidenced;
- independently-verified;
- regression-protected;
- release-accepted.

Não colapsar esses estados em um único boolean.

# Evidence classes

Evidence pode ser:

- build artifact;
- static-analysis finding;
- unit/property result;
- integration/contract result;
- conformance result;
- E2E trace;
- UI interaction trace;
- screenshot/visual diff;
- accessibility report;
- benchmark result;
- profiler trace;
- memory/leak report;
- security finding;
- fuzz corpus/seed;
- crash dump;
- round-trip artifact;
- manual structured review;
- release artifact manifest.

# Evidence validity

Toda evidence deve declarar provenance e freshness. Mudança em source, configuration, toolchain, environment, schema, provider/model ou dependency relevante pode torná-la stale. O Prumo deve conseguir explicar por que uma evidence ainda vale.

# Proof-of-use

Algumas classes exigem execução real, não apenas teste sintético. Exemplos:

- desktop GUI: launch + interaction real;
- installer: instalar e remover em ambiente limpo;
- file importer: abrir fixture real;
- save/load: round trip real;
- plugin: load/use/unload;
- CLI: processo real e exit code;
- release package: executar o artefato distribuído.

# Acceptance coverage

Cada acceptance criterion deve possuir coverage explícita. Estados: uncovered, planned, implemented, evidenced, verified, accepted, waived, blocked_external.

Goal DONE exige zero required criteria em uncovered/planned/implemented-only.

# False-green detectors

O Quality Orchestrator deve procurar padrões:

- teste que só chama função sem assertion relevante;
- screenshot sem interação precedente;
- mocks cobrindo todos os boundaries;
- command criado mas sem route;
- menu renderizado mas sem handler;
- disabled state inexistente;
- evidence de commit diferente do revisado;
- benchmark sem baseline;
- retry convertido silenciosamente em pass;
- ignored/skipped tests em gate required;
- coverage alta com acceptance uncovered.

# Review binding

Approval deve estar vinculada a revision exata. Novo commit material depois do review invalida approval ou exige impact-aware re-review.

# Output esperado

Comando conceitual futuro:

prumo assure coverage

prumo assure surface

prumo assure evidence

prumo assure explain <surface-id>

A saída machine-friendly deve indicar lacunas objetivas, não “parece completo”.