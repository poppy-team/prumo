# 79.AA — Grounding, Spec Compliance, Surface Conformance e Novas Skills do Prumo Unificado

> Authority: canonical specification.
> Logical ID: 79 AA
> Source: Notion Living Book (3e29bb7d023f811a9490f5c1ae91f9f2)
> Status: Skill Package P0: Grounding, Spec Compliance e Surface Conformance.


<aside>
🧩

**Status: extensão P0/P1 do Programa 79.** Esta página fecha lacunas reveladas pela consolidação Framework + Harness + Code Agent. O objetivo não é aumentar o catálogo por quantidade, mas garantir grounding, conformidade entre surfaces, promoção documental, recuperação e governança de custo sem transformar cada regra em prompt.

</aside>

# Princípio de classificação

Antes de criar qualquer pacote, decidir se a necessidade pertence a:

1. **Core invariant/check** — comportamento determinístico que não deve depender de LLM;
2. **Skill** — conhecimento + procedimento + checks reutilizáveis em múltiplas tasks;
3. **Agent role** — responsabilidade autônoma com input/output/handoff próprio;
4. **Recipe** — DAG/workflow recorrente envolvendo várias capabilities;
5. **Profile/bundle** — composição de skills existentes;
6. **Documentation Contract** — requisito de informação/coverage;
7. **Provider/adapter** — integração externa;
8. **DEFER/REJECT** — ganho insuficiente.

# Regras anti-duplicação

Uma capability nova só é criada se:

- possui owner semântico distinto;
- activation pode ser definida positivamente e negativamente;
- existe output/evidence próprio;
- composição existente é materialmente pior;
- contexto adicional é justificável;
- há eval capaz de mostrar ganho;
- lifecycle/freshness são claros.

Se a necessidade for uma invariante verificável, **não criar skill para convencer a LLM a obedecer**: implementar validator/gate.

# Matriz de novas lacunas

| Capacidade | Decisão inicial | Pri. | Owner |
| --- | --- | --- | --- |
| Canonical Truth / anti-invention | CORE CHECK + cross-cutting contract | P0 | Knowledge/Policy Runtime |
| Grounded implementation / spec compliance | CREATE Skill | P0 | Implementation workforce |
| Implementation reality verification | CREATE Skill | P0 | Review/Knowledge |
| Surface/protocol conformance | CREATE Skill + Recipe | P0 | Protocol/Quality |
| Documentation promotion & reconciliation | CREATE Skill + Recipe | P0/P1 | Documentation Runtime |
| Agent-aware UI/UX | CREATE Skill | P1 | UI/UX workforce |
| State-transition assurance | EXTEND testing/quality | P0 | Quality |
| Recovery/resume/idempotency | EXTEND resilience + testing | P0 | Runtime/Quality |
| Budget/model routing | CORE ENGINE + EXTEND orchestration skill | P1 | Budget/Workforce |
| Protocol evolution compatibility | EXTEND compatibility-migrations | P1 | Protocol/Release |
| Evidence freshness / invalidation | CORE CHECK + EXTEND evidence skill | P0 | Quality |
| Decision supersession | CORE WORKFLOW + Recipe | P0/P1 | Knowledge Governance |

# P0 Skill — grounded-implementation

## Missão

Impedir que implementers operem a partir de memória vaga, prompt genérico ou suposições sobre o repositório.

## Activation positiva

Ativar quando:

- task altera código/config/schema;
- há Decision/Requirement aplicável;
- mudança atravessa mais de um módulo;
- existe risco de drift planned-vs-implemented;
- agent precisa interpretar “implemente conforme documentação”.

## Negative triggers

Não ativar para:

- explicação puramente conceitual;
- transformação textual sem repo;
- operação trivial totalmente descrita por command determinístico.

## Input contract

- Goal/Task refs;
- Implementation Dossier;
- Context Manifest;
- accepted decisions;
- repository snapshot/revision;
- allowed mutation boundary;
- budget/policy.

## Procedimento obrigatório

1. resolver contracts aplicáveis;
2. inspecionar repo reality;
3. verificar paths/symbols antes de citá-los;
4. classificar drift;
5. listar unknowns/assumptions;
6. verificar baseline;
7. implementar menor delta coerente;
8. executar acceptance/evidence;
9. emitir Documentation Delta.

## Output contract

GroundedImplementationReport:

- task refs;
- sources used;
- repo facts verified;
- assumptions;
- unknowns;
- changes;
- tests/evidence;
- drift;
- docs delta;
- completion state.

## Proibições

- inventar symbol/path;
- completar requisito ausente por “best practice” sem proposal explícita;
- ampliar scope para “melhorar arquitetura” sem contract;
- afirmar test pass sem execution/evidence;
- alterar generated/managed region sem ownership.

## Evals

- stale docs vs current code;
- missing symbol;
- conflicting decision;
- partial implementation;
- ambiguous requirement;
- generated file;
- trivial task where skill should not overactivate.

# P0 Skill — implementation-reality-verification

## Missão

Comparar planejamento/documentação com realidade observável do repository sem transformar um lado em verdade do outro.

## Inputs

- claim/decision set;
- repository revision;
- code/schema/config;
- tests;
- evidence;
- public surfaces.

## Classes de resultado

- aligned_verified;
- aligned_unverified;
- planned_not_implemented;
- partially_implemented;
- implementation_ahead_of_docs;
- docs_stale;
- contradiction;
- unknown_insufficient_evidence;
- deprecated_or_dead_surface.

## Checks

- symbol/path existence;
- command registration;
- API route registration;
- feature reachability;
- UI control wiring;
- test oracle strength;
- config/schema presence;
- docs claim vs code;
- release/config capability.

## Output

ImplementationRealityReport com refs e evidence, nunca opinião global.

## Eval crítico

Distinguir corretamente “arquivo contém código” de “feature é reachable/exercised”.

# P0 Skill — surface-protocol-conformance

## Missão

Garantir que CLI, TUI, GUI, ACP/IDE e outros clients expressem a mesma semântica de Core/Protocol.

## Activation

- novo Command/Event/Projection;
- mudança de protocol;
- nova surface;
- alteração de Run/Permission/Budget/Evidence state;
- reconnect/replay;
- error model.

## Coverage

- command names/IDs;
- request/response schema;
- event order;
- state projection;
- cancellation;
- retry;
- permission;
- budget;
- evidence;
- errors;
- reconnect;
- replay cursors;
- version negotiation;
- unsupported capabilities.

## Negative space

Verificar que:

- GUI não bypassa Core;
- TUI não possui regra extra invisível;
- CLI machine output não mistura logs;
- client não inventa state local como canonical;
- unsupported capability falha explicitamente.

## Output

SurfaceConformanceMatrix + failures com protocol refs.

# Recipe — surface-conformance

DAG:

protocol change → affected commands/events → generate fixtures → run Core contract tests → run CLI → run TUI adapter → run GUI adapter → reconnect/replay → compare projections → evidence → gate.

Gate P0 falha se qualquer surface suportada divergir semanticamente.

# P0/P1 Skill — documentation-promotion-reconciliation

## Missão

Promover conhecimento de Notion, pesquisa ou caderno para repository canonical docs sem destruir implementation truth.

## Procedure

freeze snapshot → inventory → normalize claims → classify authority → compare implementation → detect contradictions → plan doc impact → generate candidates → validate → review → promote.

## Output

DocumentationMigrationReport.

## Must-not

- copiar corpus inteiro para docs/;
- declarar target architecture como implemented;
- reescrever código para fazer code “combinar” com Notion;
- apagar docs antigas antes de coverage proof;
- perder provenance.

# Recipe — notion-to-repository-promotion

Entradas:

- snapshot ID;
- target repo;
- authority policy;
- documentation profile.

Gates:

- secret/privacy scan;
- source manifest;
- contradiction classification;
- planned-vs-implemented;
- links/schema;
- docs readiness;
- review;
- promotion.

# P1 Skill — agent-aware-ui-ux

## Missão

Especializar design/implementação de interfaces cujo objeto principal é trabalho de agents e Runs.

Não substituir design-system, accessibility ou UI implementation; compor com elas.

## Domínio próprio

- Run/Thread/Agent status;
- streaming;
- tool events;
- approvals;
- Follow Agent;
- file/diff activity;
- evidence/gates;
- multi-agent ownership;
- budget/quota;
- reconnect/replay;
- conflicts human×agent;
- interruption/cancel/resume;
- progressive disclosure de reasoning summaries permitidos;
- uncertainty/UNKNOWN states.

## Regras

- não usar animação/cor como única fonte de state;
- distinguir queued/running/waiting/blocked/failed/completed;
- indicar side effects;
- approvals devem informar ação, scope e consequência;
- “working” não pode ocultar deadlock/stall indefinidamente;
- Follow Agent é opt-in/reversível;
- activity feed deriva de events estruturados;
- UI mostra provenance/evidence quando relevante;
- não exibir confidence como garantia.

## Evals

- discoverability;
- task completion;
- keyboard/focus;
- accessibility;
- reconnect;
- stale projection;
- concurrent agents;
- denied permission;
- partial failure;
- very long run;
- hardware modesto.

# EXTEND — testing/quality com state-transition-assurance

Adicionar ao contrato de testing:

actor + precondition + previous revision + mutation + side effects + invariants + cancel/rollback + failure state.

Aplicar state model/property testing quando sequence bugs forem relevantes.

Não criar skill separada inicialmente; promover somente se evals mostrarem activation/output próprios.

# EXTEND — resilience-chaos com recovery/resume/idempotency

Adicionar:

- checkpoint semantics;
- retry-after-side-effect;
- duplicate-effect detection;
- crash recovery;
- partial write;
- cancellation;
- handoff;
- provider outage;
- worktree recovery;
- replay consistency.

# Recipe — agent-handoff-recovery

failure/quota/outage → classify side effects → checkpoint → validate workspace → build typed handoff → choose replacement → resume → verify no duplicate effects → evidence.

# EXTEND — orchestration/context com budget-model-routing

Conhecimento de routing pode ser skill, mas decisões concretas pertencem ao Budget Manager/Model Gateway.

A skill deve ensinar/validar:

- task-to-route mapping;
- quality floor;
- reservation;
- review reserve;
- quota semantics;
- cost-aware escalation;
- fallback vs handoff;
- provider diversity.

Não permitir que texto da skill substitua hard budget enforcement.

# EXTEND — compatibility-migrations com protocol evolution

Adicionar:

- major/minor negotiation;
- capability discovery;
- generated bindings compatibility;
- old-client/new-server;
- new-client/old-server;
- deprecation window;
- migration fixtures;
- unknown field handling;
- event versioning;
- replay compatibility.

# CORE CHECK — llm-output-grounding

Não criar skill própria no primeiro corte.

Checks transversais:

- claim requiring repo fact possui source/evidence?
- output marcou assumption?
- completion claim tem evidence?
- path/symbol cited exists at source revision?
- decision status current?
- source superseded?
- unknown improperly converted into fact?

Esse checker pode gerar warnings/gates e alimentar evals.

# CORE CHECK — evidence freshness

Toda Evidence declara dependencies e source revision.

Mudança material invalida ou torna stale evidence afetada.

Gate não aceita stale evidence sem policy explícita.

# Recipe — decision-supersession

new decision → resolve overlapping claims → compare scope → classify replace/partial/parallel → impact analysis → migration delta → update old status → update current index → invalidate contexts/evidence/docs → verify retrieval.

# Agents novos candidatos

## conformance-reviewer — P0/P1 candidate

Responsabilidade:

- review independente de Protocol/Surface/Contract;
- não implementa a mesma mudança que aprova em high-risk profile;
- produz ConformanceReviewReport;
- busca divergence, dead surface, stale projection e bypass.

Gate CREATE:

skills + reviewer genérico não conseguem fornecer activation/output/eval suficientemente específicos.

## documentation-reconciler — P1 candidate

Responsabilidade:

- reconciliar planning snapshots, canonical docs e implementation reality;
- não editar código para “resolver” drift;
- emitir migration/documentation proposals.

Gate CREATE:

volume de promotions reais justificar role autônoma.

## recovery-verifier — P1 candidate

Responsabilidade:

- validar resume, idempotency e crash behavior independentemente do implementer.

Gate CREATE:

resilience/testing composition se mostrar insuficiente.

# Agents que NÃO devem ser criados por enquanto

## model-cost-governor

REJECT como agent inicial.

Budget e quota são melhores como deterministic runtime + policies + explainability. LLM pode aconselhar, não ser autoridade financeira.

## anti-hallucination-agent

REJECT.

Anti-invention é constituição/core contract, não um agent tentando vigiar outro prompt.

# Recipes novas candidatas

- surface-conformance — P0;
- notion-to-repository-promotion — P0/P1;
- agent-handoff-recovery — P1;
- decision-supersession — P1;
- budget-exhaustion-recovery — P1;
- protocol-breaking-change — P1;
- cross-client-release-verification — P1.

# Recipe — budget-exhaustion-recovery

soft threshold → reduce optional work → preserve verifier reserve → checkpoint → choose cheaper valid route if quality floor remains → hard threshold → stop safely → emit blocked_budget state → continuation instructions.

Nunca “terminar rápido” e marcar green.

# Required metadata para novas skills

Cada nova skill deve declarar no manifest:

- id/version/schema;
- purpose;
- non-goals;
- owner;
- maturity;
- activation positives;
- negative triggers;
- required capabilities;
- dependencies/conflicts;
- permissions;
- context budget;
- knowledge units;
- workflows;
- checks;
- output schemas;
- evidence types;
- freshness;
- examples good/bad;
- tests;
- eval corpus;
- failure modes;
- deprecation/replacement.

# Required eval dimensions

Para skills P0/P1 desta página:

- activation precision;
- activation recall;
- false positive cost;
- false negative cost;
- task success delta;
- regression rate;
- token/context overhead;
- latency;
- evidence completeness;
- prohibited behavior rate;
- stale-source sensitivity;
- ambiguous-input behavior;
- abstention/UNKNOWN correctness.

# Promotion gates

draft → experimental somente com manifest/schema.

experimental → recommended somente com positive/negative fixtures.

recommended → verified somente com eval baseline + real dogfood.

verified → stable/recommended distribution somente com freshness owner e incident history aceitável.

# Context composition

Skills não recebem o Livro Vivo.

Resolver carrega:

manifest → short procedure summary → exact workflow/knowledge blocks necessários → examples somente quando úteis.

# Definition of Done desta extensão

- anti-invention não depende de uma skill;
- grounded implementation possui procedimento/evals;
- repo reality pode ser verificada independentemente;
- surface protocol possui conformance workflow;
- documentation promotion é reproduzível;
- agent-aware UI/UX tem owner claro;
- recovery/state-transition entram em quality;
- budget routing permanece governado pelo Core;
- agents novos passam por anti-duplication gate;
- nenhuma capability nova entra sem negative triggers, output contract e eval.