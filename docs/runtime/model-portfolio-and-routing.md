# 87 — Model Portfolio, Roteamento por Custo/Quota e Budget Mensal

> Authority: canonical specification.
> Logical ID: CONST-87
> Source: Notion Living Book (3e29bb7d023f8123bfbef4369c8b8da1)
> Status: Constituição 87: Roteamento por Custo/Quota, Budget e Fallback.


<aside>
💰

**Status: policy operacional versionável.** Esta página transforma o uso econômico de modelos em regra explícita do Prumo sem acoplar o Core a um fornecedor. O teto pessoal atual para este profile é **R$100/mês**, mas pricing, planos e quotas são metadata externas e mutáveis.

</aside>

# Princípio

Budget não é apenas token count.

O Prumo governa:

- dinheiro;
- input/output/context tokens;
- chamadas;
- tempo;
- concorrência;
- quota;
- tool output;
- external-source usage;
- retries;
- premium escalations.

# Profile atual: personal-low-cost

Valores:

- monthly_hard_currency_cap = BRL 100;
- quality_floor = dependente da task e do risco;
- silent_overage = forbidden;
- quota_unknown = never fabricated.

O teto monetário é hard. Ao aproximar-se dele:

1. reduzir trabalho opcional;
2. preferir routes de menor custo compatíveis;
3. reutilizar cache/context;
4. preservar capacidade para review/fix quando policy exigir;
5. checkpoint antes de hard stop.

Nunca degradar security/release/critical verification abaixo do quality floor só para economizar.

# Channels atuais

O ambiente pode usar:

- OpenCode;
- CommandCode;
- Freebuff;
- providers/gateways OpenAI-compatible;
- outros adapters permitidos.

Esses nomes são **ProviderProfiles/channels**, não entidades de domínio. Mudança de plano não altera Agent Runtime.

Múltiplas contas só podem entrar em AccountPool quando o uso for permitido pelos termos do serviço e pela policy de credenciais. O Prumo não deve automatizar evasão de limites contratuais.

# Portfólio atual para desenvolvimento do Prumo

As porcentagens abaixo são **share alvo de tarefas roteadas**, não share obrigatório de dinheiro.

| Modelo | Share alvo | Responsabilidade preferida |
| --- | --- | --- |
| DeepSeek V4.1 | 50% | implementação, CLI, adapters/providers, persistência, testes, migrações, Bubble Tea, debugging e volume |
| Muse Contributor | 25% | contracts, arquitetura, orchestration, Gauntlet, evidence, segurança e review de alto impacto |
| GLM-5.3 Flash | 15% | iterações rápidas, TUI components, documentação, testes e refactors mecânicos |
| Kimi K3 | 10% | UI/UX, interaction design, TUI/IDE/workspace viewer, discoverability e estados contextuais |

A execução real pode divergir conforme quota, qualidade e task mix. O Budget Report deve mostrar desvio e reason code.

# Routing classes

## ROUTE-CHEAP

Use para:

- transformações mecânicas;
- testes repetitivos;
- docs deriváveis;
- pequenas correções locais;
- classificação de baixo risco.

## ROUTE-PRIMARY

Use para implementação normal e debugging.

## ROUTE-ARCH

Use para mudança de boundary, protocol, schema, migration ou desenho transversal.

## ROUTE-UX

Use para interaction model, workflows, hierarchy, visual behavior e usability.

## ROUTE-VERIFY

Preferir modelo/provider diferente do implementer quando risco justificar.

# Escalation protocol

Um modelo mais caro/escasso só entra quando existe reason:

- bloqueio após tentativas limitadas;
- arquitetura transversal;
- review independente;
- UX complexa;
- regression difícil;
- high-risk migration;
- ambiguity cuja resolução barata falhou.

Registrar:

- escalation_reason;
- previous_attempts;
- expected_gain;
- budget_remaining;
- selected_route.

# De-escalation

Após resolver a parte difícil, retornar ao worker mais barato para:

- aplicar mudanças repetitivas;
- escrever fixtures;
- atualizar docs;
- executar refactors;
- completar testes.

# Quota-aware routing

QuotaState:

- known;
- estimated;
- unknown;
- cooldown;
- exhausted.

Nunca inferir saldo exato a partir de ausência de erro.

429/quota failure pode atualizar health/cooldown, não inventar “X tokens restantes”.

# Reservation

Antes de criar subagent:

parent budget → reserve child envelope → execute → return unused reservation.

Impede um subagent de consumir o resto da run.

# Review reserve

Quality policy pode exigir reservar capacidade suficiente para:

- verifier;
- regression;
- fix loop;
- documentation delta.

Uma Run não deve gastar 100% do envelope apenas na primeira implementação e ficar sem budget para provar o resultado.

# Cost metadata

Pricing é versionado por:

- provider;
- model;
- currency;
- input price;
- output price;
- cache price;
- effective_from;
- source;
- confidence/verification.

Sem metadata válida, exibir custo como unknown/estimated, nunca zero.

# Token efficiency

Obrigatório:

- progressive disclosure;
- role-specific context;
- Context Manifest;
- dedup de snippets;
- retrieval por stable IDs;
- cache quando provider suporta;
- compaction estruturada;
- não reenviar transcript completo;
- não carregar todas as skills;
- não pedir a múltiplos agents a mesma análise sem purpose.

# Budget × Workforce

Least Workforce e budget são integrados:

task risk → required capabilities → minimum workforce → route/model constraints → budget reservation.

Mais agents precisam justificar ganho esperado.

# Budget × Quality

Hard gate não pode ser compensado por economia.

Se budget insuficiente para required verification:

- run = blocked_budget;
- checkpoint;
- informar o que foi implementado e o que ainda não foi provado;
- nunca marcar accepted/released.

# CLI

Commands alvo:

- prumo budget show;
- prumo budget explain;
- prumo budget set;
- prumo cost estimate;
- prumo cost report;
- prumo route explain;
- prumo quota status.

# GUI/TUI

Exibir sem poluir:

- budget remaining;
- route ativa;
- model/provider;
- quota/cooldown;
- premium escalation;
- projected vs observed cost;
- warning antes de hard stop.

# Evidence

Cada Run registra:

- budget inicial;
- reservations;
- observed usage;
- pricing revision;
- cache hits reportados;
- routes;
- fallbacks/handoffs;
- escalations;
- stopped/downgraded reason.

# Anti-gaming

Não otimizar KPI “custo” causando:

- mais retrabalho;
- false-green;
- review ausente;
- contexto insuficiente;
- modelo inadequado à task.

Métricas recomendadas:

- cost per accepted outcome;
- quality per currency unit;
- tokens per verified requirement.

Não usar apenas custo bruto.

# Definition of Done

- monthly cap e run envelopes coexistem;
- model shares são policy, não hard-coded;
- provider plans são adapters;
- quota unknown permanece unknown;
- fallback respeita side effects;
- review reserve é possível;
- cost/quality são observáveis;
- hard cap nunca vira false-green.