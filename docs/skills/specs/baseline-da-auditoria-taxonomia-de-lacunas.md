# 79.A — Baseline da Auditoria, Taxonomia de Lacunas e Matriz Mestra

> Authority: canonical specification.
> Logical ID: 79 A
> Source: Notion Living Book (3d89bb7d023f81c69edde8f22dfd66e8)
> Status: Skill Package de referência para 79 A — Baseline da Auditoria, Taxonomia de Lacunas.


<aside>
📊

Esta página é o **baseline de auditoria** do programa. Ela registra o que deve ser medido antes de qualquer alteração e como cada lacuna deve ser classificada para impedir uma migração baseada apenas em opinião ou contagem de arquivos.

</aside>

# Objetivo

Criar uma fotografia reprodutível do workforce atual e uma matriz que permita comparar **antes → depois** após cada fase de hardening.

# Unidade de análise

A auditoria usa como unidade mínima um componente versionável:

- Agent: `agents/<id>/AGENT.md` e metadata associada.
- Skill: `skills/<id>/manifest.json`, `SKILL.md` e seus knowledge/workflows/templates/checks/scripts/schemas/tests/evals/examples.
- Recipe: `recipes/<id>/RECIPE.md` e DAG de steps/gates.

# Taxonomia de lacunas

| Classe | Definição | Exemplo | Ação padrão |
| --- | --- | --- | --- |
| **Internal Gap** | Componente existe, mas não cumpre seu propósito com profundidade suficiente. | `screen-reader` existe, porém executa passos genéricos de UX. | Reescrever conteúdo/contrato; não criar skill nova. |
| **Integration Gap** | Componentes existem, mas seus contratos e relações podem divergir. | Agent executa responsabilidade cuja skill não está em `required_skills`. | Adicionar validação/lint e dependência explícita. |
| **Coverage Gap** | Capacidade recorrente importante não pertence claramente a nenhum componente. | Privacy engineering transversal. | Propor novo componente após análise anti-duplicação. |
| **Evidence Gap** | A regra existe, mas não há prova objetiva suficiente. | Gate aceita “tests passed” sem comando/artefato/hash. | Normalizar evidence e exigir producer verificável. |
| **Activation Gap** | A skill existe, mas seleção automática/contextual não possui precisão demonstrada. | Ativação redundante de `clean-code`, `code-quality` e `refactoring`. | Corpus + precision/recall + ownership boundaries. |
| **Privilege Gap** | Capability declarada é maior que o necessário. | Skill read-only recebe filesystem.write/process.spawn. | Least privilege + elevation explícita. |
| **Lifecycle Gap** | Maturity/deprecation/freshness não está governada. | Skill stale continua `stable`. | Lifecycle + freshness + migration policy. |

# Métricas de baseline obrigatórias

1. **Inventory coverage:** número de agents, skills e recipes encontrados vs catálogo/resolver.
2. **Manifest coverage:** percentual de skills no contrato v3 atual.
3. **Machine-contract coverage:** outputs com schema, evidence tipada, handoffs estruturados.
4. **Specialization score:** percentual de skills cujo procedimento contém ações específicas do domínio e checks coerentes.
5. **Template similarity:** clusters de [SKILL.md](http://SKILL.md) com alta similaridade textual/estrutural.
6. **Activation precision/recall:** em corpus rotulado para skills automáticas/contextuais.
7. **Overlap rate:** média de skills semanticamente equivalentes ativadas para a mesma tarefa.
8. **Orphan rate:** skill sem agent/recipe/feature que a utilize; agent/recipe referenciando skill inexistente; outputs sem consumer.
9. **Permission excess rate:** capabilities concedidas sem uso necessário demonstrado.
10. **Evidence completeness:** proporção de gates com producer, comando, target, environment, result, artifact/hash e timestamp.
11. **Eval coverage:** percentual de componentes críticos com positive/negative/adversarial evals.
12. **Freshness coverage:** knowledge packs com source/version/last-reviewed.
13. **Failure recovery coverage:** steps com classe de falha, retryability, idempotência e resume/rollback.
14. **Context efficiency:** tokens adicionados por skill vs ganho mensurado.

# Matriz mestra de frentes

| Pri. | Frente | Lacuna dominante | Entrega | Esforço |
| --- | --- | --- | --- | --- |
| **P0** | Workforce Relationship Graph | Agents, skills e recipes não possuem visão canônica conjunta. | Agent → Skill → Recipe → Capability → Evidence → Gate → Handoff graph. | L |
| **P0** | Skill Package v3 migration | Formato e profundidade heterogêneos. | Contrato v3 + validator + migration tooling. | XL |
| **P0** | Specialization rewrite | Micro-skills nominalmente diferentes compartilham boilerplate. | Procedimentos/domain checks reais. | XL |
| **P0** | Agent↔Skill / Recipe↔Agent drift | Declaração manual pode divergir. | Static lint + resolver validation. | M |
| **P0** | Evidence Contract | Evidência textual insuficiente. | Evidence envelope verificável. | L |
| **P0** | Evals | Existência de arquivos não prova ganho. | Baseline/corpus/thresholds por risco. | XL |
| **P0** | Fallback/Recovery | Retry genérico com modelo alternativo. | Failure taxonomy + deterministic recovery. | L |
| **P0** | Least privilege | Capabilities amplas em tarefas simples. | Required/optional/elevated capabilities. | L |
| **P0** | Handoff | Continuidade depende demais de narrativa. | Structured handoff artifact. | M |
| **P1** | Dependencies/conflicts | Relacionamentos v3 ainda não representam todo o catálogo real. | DAG de composição + cycle/conflict detection. | L |
| **P1** | Semantic overlap | Famílias próximas competem por ownership. | owns / does-not-own / delegates-to. | M |
| **P1** | Context budget | Campo existe, enforcement precisa de métricas. | Progressive disclosure mensurado. | M |
| **P1** | Freshness/provenance | Toolchains e standards envelhecem. | version/source/last-reviewed/stale rules. | M |
| **P1** | Idempotency/concurrency | Multi-agent e resume podem duplicar ou conflitar mudanças. | leases, state guards, idempotency keys, conflict policy. | L |
| **P1** | Observability/Coverage | Difícil descobrir componentes caros ou pouco úteis. | scorecards + coverage reports. | L |
| **P2** | New capabilities | Privacy, i18n, fuzz/property, compatibility, resilience etc. | Novos pacotes somente após prova anti-duplicação. | L–XL |

# Invariantes do baseline

- O baseline deve ser produzido a partir do commit/árvore analisado e registrar SHA.
- Mudanças posteriores não podem reescrever o baseline histórico; devem gerar nova snapshot.
- Contagem total de skills **não** é KPI de maturidade.
- Uma skill que existe mas é boilerplate conta como **cobertura nominal**, não cobertura efetiva.
- Findings da auditoria devem distinguir **fato observado**, **inferência** e **recomendação**.

# Entregáveis de F0

- `workforce-inventory.json`.
- `workforce-relations.json` inicial.
- `coverage-report.json` + Markdown humano.
- similarity clusters.
- orphan/conflict report.
- capability/permission audit.
- baseline de activation/evals para amostra P0.
- hash do commit auditado.

# Exit Gate F0

F0 só fecha quando 100% dos componentes encontrados no filesystem e catálogo podem ser identificados por ID estável, nenhum resultado depende apenas de inspeção manual e a auditoria pode ser repetida com o mesmo commit produzindo o mesmo inventário.