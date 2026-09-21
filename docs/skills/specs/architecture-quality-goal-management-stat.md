# 79.S — Architecture Quality, Goal Management, State e Plugin Architecture

> Authority: canonical specification.
> Logical ID: 79 S
> Source: Notion Living Book (3d89bb7d023f81de9cdfff52196d2354)
> Status: Skill Package de referência para 79 S — Architecture Quality, Goal Management, Stat.


<aside>
🏛️

Esta página cobre capacidades arquiteturais que atravessam várias famílias e não devem ficar implícitas em agents ou recipes.

</aside>

| Skill | Pri. | Lacuna | Aprimoramento | Aceite |
| --- | --- | --- | --- | --- |
| **architecture-quality** | P1 | Já cobre direction/cycles/boundaries/globals/fitness; falta aprofundar arquitetura evolutiva e NFRs. | Coupling/cohesion metrics, NFR budgets, architecture diff, modularity drift, deprecation/migration, stability boundaries e fitness functions executáveis. | Invariantes críticas são verificáveis e architecture change produz delta mensurável. |
| **goal-management** | P0 | Goal lock precisa estar ligado à coverage real de acceptance/evidence. | Goal→acceptance→plan→task→evidence→gate graph, amendments, supersession, scope drift detection, completion coverage. | DONE é impossível sem evidence coverage dos critérios required. |
| **state-management** | P1 | State bugs derivam de ownership, concorrência, persistence e async semantics. | Single source of truth, normalized/derived state, transactional transitions, persistence/hydration, concurrency, conflict rules, debugging/time-travel when applicable. | Critical state transitions possuem state machine/invariants testáveis. |
| **plugin-architecture** | P0 | Plugins precisam ser tratados como API/lifecycle, não apenas extensões carregáveis. | API versioning, capability manifest, sandbox boundary, load/unload/update, dependency isolation, compatibility/deprecation, crash containment e discovery. | Plugin incompatível/malformado falha isoladamente e não corrompe host. |

# Architecture quality — separação de responsabilidades

`architect` decide boundaries/trade-offs para uma mudança. `architecture-quality` fornece regras/fitness checks reutilizáveis. `architecture-change` recipe governa o fluxo de alteração. Nenhum dos três substitui os outros.

# Architecture Diff

Mudanças arquiteturais relevantes devem poder produzir um delta contendo:

- modules/boundaries added/removed;
- dependency direction changes;
- public contract changes;
- trust boundaries;
- data ownership;
- runtime/deployment topology when relevant;
- NFR budget change;
- compatibility/migration impact;
- new architectural debt/exception.

# Goal Coverage Contract

Cada acceptance criterion required deve possuir estado: `uncovered`, `planned`, `implemented`, `evidenced`, `accepted` ou `blocked`. Goal DONE exige todos os required criteria em estado permitido pela policy e evidence válida/não stale.

# Scope drift

Detectar quando task/implementation produz mudanças sem ligação a Goal/Plan. Drift pode ser:

- necessary incidental change;
- newly discovered requirement;
- opportunistic refactor;
- accidental scope expansion.

Cada classe possui policy de amendment/review diferente.

# State management model

State transitions devem declarar actor, preconditions, expected previous revision/version, mutation, invariants, side effects e failure behavior. Para distributed/realtime state, delegar detalhes a `realtime-synchronization`/`rpc-protocols` em vez de duplicar regras.

# Plugin Architecture vs Plugin Security

`plugin-architecture` é owner de lifecycle/API/compatibility/isolation design. `plugin-security` é owner de trust, permission, malicious behavior e signing/revocation. Recipes/agents devem compor ambas quando third-party plugin risk existir.

# Exit Gate

Architecture/Goal/state/plugin concerns possuem owners explícitos; Goal completion é evidence-driven; architecture changes geram delta/fitness evidence; plugin lifecycle e plugin security são separados e composáveis.

# Amendment 2026-09-21 — Architecture Assurance Graph

[84 — Total Assurance Constitution: Gauntlet Loop, Evidence e Anti-False-Green](../../quality/gauntlet-84.md) amplia Architecture Diff com evidence dependency, resource ownership, recovery boundaries, public-surface coverage e negative guarantees. Goal Coverage Contract passa a impedir DONE quando acceptance required está apenas implemented sem exercised/evidenced/verified conforme policy.