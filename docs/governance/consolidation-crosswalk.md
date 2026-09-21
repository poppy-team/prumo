# 90 — Crosswalk de Consolidação: Prumo Framework ↔ Code Agent, Ownership e Supersession

> Authority: canonical specification.
> Logical ID: CONST-90
> Source: Notion Living Book (3e29bb7d023f81458af2f245c5a3b23d)
> Status: Constituição 90: Crosswalk de Consolidação, Ownership e Supersession.


<aside>
🗺️

**Status: crosswalk de consolidação.** Esta página registra como cada capítulo do antigo Prumo Code Agent se relaciona com o Livro Vivo Unificado. O objetivo é preservar todo o conteúdo, impedir duas fontes concorrentes e orientar retrieval, refactoring e futura promoção para o repositório.

</aside>

# Regra de consolidação

Nenhuma página é descartada apenas porque seu tema aparece em outra parte do Livro Vivo.

Cada página recebe um papel:

- **CANONICAL OWNER** — fonte principal vigente do conceito;
- **SPECIALIZED DETAIL** — detalhamento do domínio Code Agent que permanece válido sob o owner unificado;
- **RESEARCH/REFERENCE** — corpus de comparação, não decisão por si só;
- **HISTORICAL** — descreve estágio anterior;
- **SUPERSEDED** — decisão substituída, preservada apenas por provenance;
- **MIGRATION INPUT** — material usado para reconciliar/prometer estado futuro.

Retrieval operacional deve resolver owner/status antes de usar conteúdo.

# Crosswalk completo — antigo Prumo Code Agent 00–41

| Página Parte II | Papel após fusão | Owner/constituição atual | Regra de uso |
| --- | --- | --- | --- |
| 00 — Visão, Escopo e Relação | SPECIALIZED DETAIL | 86 — Arquitetura Unificada | Usar para visão específica do Code Agent; 86 vence em boundaries atuais. |
| 01 — Baseline Herdado | HISTORICAL + reality input | 89 — Programa de Consolidação | Não assumir que baseline descrito continua implementado sem repo verification. |
| 02 — Referências de Harnesses | RESEARCH/REFERENCE | 35 + Research governance | Patterns externos exigem Finding/Decision antes de virar contract. |
| 03 — Arquitetura Alvo do Agent Harness | SPECIALIZED DETAIL | 48 + 86 | Detalha execution plane; 86 define ownership final. |
| 04 — Model Runtime, Providers, Routing e Budgets | SPECIALIZED DETAIL | 53 + 87 | Usar contratos técnicos; portfolio/custo atual vem de 87. |
| 05 — Agent Loop, Conversation, Checkpoints | SPECIALIZED DETAIL | 51 + 85.A | State machine e resume permanecem; execução deve obedecer 85.A. |
| 06 — Coding ACI, Native Tools, MCP, ACP e A2A | SPECIALIZED DETAIL | 52 + 86 | Coding tools são specialization do Tool Gateway. |
| 07 — Permissions, Sandbox, Egress e Security Runtime | SPECIALIZED DETAIL | 12 + 54 + 60 | Não criar security policy paralela no Code Agent. |
| 08 — Context Compiler v2, Repo Map, LSP | SPECIALIZED DETAIL | 50 + 88 | Repo/code intelligence alimenta o Context Compiler único. |
| 09 — Prumo Code Desktop | SPECIALIZED PRODUCT DETAIL | 82 + 86 | UX/produto; decisão de base atual é Oxi. |
| 10 — Native Desktop UI Matrix | RESEARCH/HISTORICAL | 82 | Matriz preservada como pesquisa; não seleciona toolkit vigente. |
| 11 — Terminal Engine Nativo | SPECIALIZED PRODUCT DETAIL | 86 + product layer | Terminal é capability/client; process semantics permanecem no Harness. |
| 12 — Editor Engine | SPECIALIZED PRODUCT DETAIL | 82 | Aplicar escopo Workspace Viewer antes de IDE-grade expansion. |
| 13 — Prumo Code TUI | SPECIALIZED PRODUCT DETAIL CURRENT | 86 | Bubble Tea v2 permanece direção vigente. |
| 14 — TUI Technology Matrix | RESEARCH/REFERENCE | 86 | Bubble Tea v2 é current; challengers são research unless promoted. |
| 15 — Daemon, IPC, Remote e Escalabilidade | SPECIALIZED DETAIL | 86 | Daemon/protocol são platform-owned, não client-owned. |
| 16 — Performance Budgets e Hardware Modesto | SPECIALIZED DETAIL | 84.C + 82 | Budgets entram em Quality/Performance contracts. |
| 17 — Plugin, Extension Architecture e Public SDK | SPECIALIZED DETAIL | 10 + 86 | SDK/protocol pertence à Platform; extensions específicas podem ficar no produto. |
| 18 — Roadmap H0–H18 | MIGRATION INPUT | 89 | Usar milestones como histórico/dependência; 89 governa consolidação atual. |
| 19 — Registro de Decisões/Benchmarks/Open Questions | HISTORICAL LEDGER | 85 + 88 | Decisões current precisam status/supersession formal. |
| 20 — Panes, Paseo e Oride | RESEARCH/REFERENCE | 82/product research | Inspiração de produto, nunca requirement automática. |
| 21 — Stack H Floem + Bubble Tea | SUPERSEDED PARCIALMENTE | 82 + 86 | Floem superseded; Go Core e Bubble Tea v2 permanecem vigentes. |
| 22 — Agent Connectivity Runtime | SPECIALIZED DETAIL | 10 + 52 + 86 | Provider auth/connectivity atrás de contracts únicos. |
| 23 — Shared Product Contract | SPECIALIZED DETAIL | 86 / Prumo Protocol | Entities/commands públicos pertencem ao Protocol. |
| 24 — Workforce Runtime, Model Gateway, Fallback | SPECIALIZED DETAIL | 49 + 53 + 79 + 87 | Workforce, routing e budget seguem owners separados mas coordenados. |
| 25 — Planning & Knowledge Runtime | SPECIALIZED DETAIL | 05 + 08 + 88 | Planejamento e knowledge pertencem à Platform. |
| 26 — Deterministic Documentation Compiler | CANONICAL TECHNICAL DETAIL | 88 | Permanece implementação de referência do Documentation Runtime. |
| 27 — Gap Analysis Knowledge/Documentation | RESEARCH/MIGRATION INPUT | 88 | Usar para backlog, não como runtime contract isolado. |
| 28 — Local Knowledge Pack | SPECIALIZED/OPTIONAL | 50 + 81 | Local inference/knowledge é provider/profile, não condição do Core. |
| 29 — Local Intelligence Profiles | SPECIALIZED/OPTIONAL | 50 + 53 + 87 | Resource/model routing deve permanecer provider-neutral. |
| 30 — Local Inference Runtime | SPECIALIZED/OPTIONAL | 53 | Workers GGUF/ONNX são adapters, não canonical model semantics. |
| 31 — Context Compiler v2 Hybrid Retrieval | CANONICAL TECHNICAL DETAIL | 50 + 88 | Authority/freshness da constituição 85 têm precedência. |
| 32 — Handoff v2 e Knowledge API | CANONICAL TECHNICAL DETAIL | 08 + 51 + 85.A | Typed continuation sem dependência de transcript. |
| 33 — Knowledge Runtime Semantic Model | CANONICAL TECHNICAL DETAIL | 88 | Deve incorporar Claim/Unknown/Assumption/Supersession atuais. |
| 34 — Contradiction, Coverage & Readiness | CANONICAL TECHNICAL DETAIL | 03 + 84 + 85 | Readiness inclui authority, proof e anti-false-green. |
| 35 — Harness Build Strategy | CANONICAL TECHNICAL DETAIL | 86 + 89 | Prumo-native Go Core; external agents via providers/adapters. |
| 36 — Human Documentation Runtime | CANONICAL TECHNICAL DETAIL | 88 | Compartilha Knowledge IR sem misturar audiences/context. |
| 37 — Documentation Planner | CANONICAL TECHNICAL DETAIL | 88 | Page Graph antes de tree; no hallucinated pages. |
| 38 — Human Docs Quality/Publishing/i18n | SPECIALIZED DETAIL | 88 | Publishing é downstream de canonical docs. |
| 39 — Development Chronicle/Changelog | SPECIALIZED DETAIL | 07 + 29 + 88 | Derivar de records/events onde possível. |
| 40 — Repository Boundary & Product Split | CANONICAL DETAIL | 86 | Prumo Code futuro consome public protocol, nunca internal/. |
| 41 — Notebook-to-Repository Promotion | CANONICAL TECHNICAL DETAIL | 88 | Workflow oficial de reconciliation/promotion. |

# Overlaps principais consolidados

## Runtime/Control

Framework 48–61 e Parte II 03–08/15/22–24/32/35 descrevem ângulos complementares do mesmo runtime.

**Owner:** Platform/Harness; Parte II fornece specialization e research.

## Documentation/Knowledge

Framework 03/05/07/08/50/81/84 + Parte II 25–27/31–39/41.

**Owner:** 85 + 88 como constituições; páginas técnicas permanecem detalhamento.

## Clients

Parte II 09–14 + Framework 80/82.

**Owner:** 86; TUI Bubble Tea v2 e Native Oxi são decisões atuais.

## Budget/Models

Framework 49/53 + Parte II 04/24/29/30.

**Owner:** 49/53 para engines; 87 para portfolio/policy operacional.

## Quality

Framework 13/32/55/61/84 + Parte II 16/34.

**Owner:** 84 Total Assurance.

# Regra para futura refatoração de páginas

Não fazer merge físico destrutivo de páginas detalhadas até que:

1. seus claims estejam normalizados;
2. source maps existam;
3. owner atual esteja definido;
4. conteúdo único tenha destino;
5. links/backlinks tenham migration;
6. retrieval tests provem ausência de regressão.

Depois disso, uma página pode ser:

- mantida;
- renomeada;
- movida;
- dividida;
- transformada em histórico;
- superseded;
- arquivada.

Nunca deletar apenas para “ficar organizado”.

# Retrieval routing

Ao recuperar uma página da Parte II:

1. consultar status;
2. consultar owner;
3. carregar constituição aplicável;
4. usar conteúdo especializado;
5. descartar claims superseded/stale;
6. registrar source no Context Manifest.

# Definition of Done

- 00–41 possuem papel explícito;
- nenhum capítulo foi perdido;
- overlap não implica dual authority;
- Floem está claramente superseded;
- TUI/Oxi/Protocol owners estão claros;
- docs/knowledge possuem owner único;
- future refactor pode ocorrer sem reinterpretação por LLM.