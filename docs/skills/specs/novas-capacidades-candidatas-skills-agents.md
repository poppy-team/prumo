# 79.P — Novas Capacidades Candidatas: Skills, Agents e Recipes

> Authority: canonical specification.
> Logical ID: 79 P
> Source: Notion Living Book (3d89bb7d023f81f4aa6fc1eae57a1d0f)
> Status: Skill Package de referência para 79 P — Novas Capacidades Candidatas Skills, Agents.


<aside>
🌱

Estas capacidades são **candidatas**, não autorização automática para criar novos pacotes. Cada uma deve passar pelo teste anti-duplicação e demonstrar ganho sobre composição de skills existentes.

</aside>

# Critério para criar capacidade nova

Uma nova unidade só deve existir se:

1. o domínio aparece recorrentemente em Goals reais;
2. não existe owner atual claro;
3. compor skills existentes exige boilerplate ou perde regras importantes;
4. activation pode ser definida com precisão;
5. existem outputs/evidence próprios;
6. evals conseguem provar ganho;
7. custo de contexto/manutenção é justificável.

| Candidato | Tipo | Pri. | Escopo proposto | Gate para criação |
| --- | --- | --- | --- | --- |
| **privacy-engineering** | Skill | P1 | PII classification, minimization, consent/legal basis metadata quando aplicável, retention, deletion/export, telemetry redaction e privacy threat modeling. | Provar que security+observability não cobrem lifecycle de dados pessoais adequadamente. |
| **internationalization-localization** | Skill | P1 | Resource strings, fallback, pluralization, numbers/dates, RTL, text expansion, fonts e pseudo-localization. | Projetos multilíngues demonstram gaps recorrentes em frontend/design skills. |
| **fuzz-property-testing** | Skill | P1 | Generators, shrinking, seeds, corpus, sanitizers, stateful property testing e regression preservation. | Reuso comprovado em parser/serialization/security/network/geometry. |
| **compatibility-migrations** | Skill | P1 | Expand-contract, version negotiation, deprecation, upgrade/downgrade, file/API/config/schema compatibility e roll-forward/rollback. | Evitar duplicação em API/DB/release/compiler através de contrato central. |
| **resilience-chaos** | Skill | P1 | Timeout/retry/circuit breaker/bulkhead/backpressure, failure injection, partitions, exhaustion e recovery objectives. | Necessidade comprovada em distributed/realtime/backend projects. |
| **license-compliance** | Skill | P2 | SPDX normalization, dependency/asset/font/data license compatibility, notices e prohibited-license policy. | Supply-chain + asset research não bastam para release compliance. |
| **reliability-engineer / incident-responder** | Agent | P2 | Operational triage, mitigation, rollback, evidence timeline, recovery e postmortem. | Somente projetos com runtime operacional e incidents reais. |
| **incident-response** | Recipe | P2 | Detect→classify→contain→mitigate→verify→recover→postmortem→follow-ups. | Bug-fix não modela corretamente outage/degradation. |
| **schema-data-migration** | Recipe | P2 | Preflight, backup/restore proof, expand, backfill, cutover, contract, observe, rollback window. | Architecture-change é ampla demais para data-risk recorrente. |
| **performance-regression** | Recipe | P2 | Stable benchmark env, bisect, profile, minimal optimization, statistical comparison e review. | Regression investigations recorrentes justificam workflow próprio. |

# Anti-duplication RFC

Antes de criar cada candidato, produzir mini-RFC:

- problem statement;
- examples from real tasks;
- existing skills that partially cover it;
- why composition is insufficient;
- proposed owner/non-owner boundaries;
- activation triggers/negative triggers;
- permissions/context cost;
- output/evidence contract;
- eval design;
- maintenance/freshness cost.

# Decision outcomes

Cada RFC termina em uma destas decisões:

- **CREATE** — capacidade própria aprovada;
- **EXTEND** — ampliar skill existente;
- **COMPOSE** — usar bundle/recipe de skills existentes;
- **DEFER** — evidência insuficiente;
- **REJECT** — duplicação/custo maior que benefício.

# Exit Gate

Nenhuma capacidade nova entra no catálogo somente porque “parece útil”. Toda adição possui owner claro, negative triggers, eval baseline e justificativa de menor complexidade sistêmica.

# Candidatos promovidos pela auditoria 2026-09-16

Passam para design detalhado, sujeitos aos mesmos gates anti-duplicação: `graphics-webgpu`, `framework-threejs`, `platform-webassembly`, `wasi-component-model`, `advanced-linux`, `database-reliability-operations`, `programming-language-engineering`, `computer-science-foundations` e o novo **Teaching Module**. `lang-wat`/`format-wat`, packs por ISA e packs por banco podem permanecer knowledge/subskills se não demonstrarem activation/output/eval próprios.