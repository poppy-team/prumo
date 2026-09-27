# 79.B — Contratos Transversais, Schemas, Permissions, Evidence, Handoff e Lifecycle

> Authority: canonical specification.
> Logical ID: 79 B
> Source: Notion Living Book (3d89bb7d023f818b9291f90a0600d096)
> Status: Skill Package de referência para 79 B — Contratos Transversais, Schemas, Permission.


<aside>
🧱

**P0 estrutural.** Esta página deve ser implementada antes da campanha de reescrita das skills. Sem estes contratos, conteúdo melhor continuaria difícil de validar, selecionar e evoluir.

</aside>

# 1. Workforce Relationship Contract

Criar representação machine-readable para as relações:

```
Goal/Task
→ Risk/Profile
→ Recipe
→ Step
→ Agent role
→ Required/optional skills
→ Capabilities/permissions
→ Context inputs
→ Output schema
→ Evidence producers
→ Gates
→ Handoff
```

Cada aresta deve informar `source`, `target`, `relation_type`, `required`, `reason`, `condition`, `version_constraint` quando aplicável.

## Validadores

- referência a ID inexistente;
- output sem consumer quando deveria alimentar próximo step;
- recipe requer skill não compatível com agent;
- capability de step não disponível ao agent/skill;
- dependência circular;
- conflito entre skills simultaneamente selected;
- gate sem evidence producer;
- specialist risk sem specialist reviewer quando policy exigir.

# 2. Skill Package v3 Contract

O contrato v3 definido em [21 — Skill Package v3, Resolver, Configuração e Evals](../../framework/specs/ch21.md) deve ser concretizado de forma homogênea.

## Campos mínimos

- `id`, `version`, `schema_version`;
- `description`, `maturity`, `activation`;
- `capabilities`, `requires`, `requires_any`, `conflicts`;
- `permissions` e elevation policy;
- `compatibility` e supported versions;
- `context_budget`;
- `outputs` e output schemas;
- `knowledge`, `workflows`, `checks`, `scripts`, `tests`, `evals`, `examples`;
- `provenance`, `last_reviewed`, `freshness_policy`;
- `supersedes`, `deprecated_by`, `migration_notes` quando necessário.

# 3. Agent Contract

Um agent estável deve declarar:

- role ID e propósito;
- modes em que pode atuar;
- required/optional skills;
- capabilities mínimas e eleváveis;
- input contracts;
- output contract/schema;
- procedure states;
- decision rules;
- evidence responsibilities;
- measurable stop conditions;
- escalation routing;
- failure/retry/resume behavior;
- handoff contract;
- prohibited self-approval paths;
- eval suite e maturity.

# 4. Recipe Contract

Cada recipe deve possuir:

- preconditions;
- typed required inputs;
- step DAG;
- dependencies explícitas;
- role + skill requirements por step;
- conditional steps/gates;
- expected outputs;
- evidence types/producers;
- mutation/irreversibility classification;
- retryability/idempotency;
- compensation/rollback/resume;
- completion criteria;
- recipe-level evals.

# 5. Evidence Envelope

Evidence nunca deve ser apenas texto “passou”. Envelope mínimo recomendado:

```json
{
  "id": "ev_...",
  "type": "test|build|lint|review|security_scan|benchmark|accessibility|visual|...",
  "producer": "provider/tool/agent",
  "command_or_action": "...",
  "target": "commit/artifact/path/scenario",
  "environment": {},
  "started_at": "...",
  "finished_at": "...",
  "result": "pass|fail|inconclusive|skipped",
  "exit_code": 0,
  "artifacts": [],
  "hashes": [],
  "summary": "...",
  "limitations": [],
  "provenance": {}
}
```

## Regras

- `skipped` nunca equivale a `pass`.
- `inconclusive` bloqueia gate quando a evidência é required.
- review deve referenciar commit/diff exato.
- benchmark deve apontar baseline e metodologia.
- evidence expirada/stale deve ser invalidada por mudança relevante no target/environment.

# 6. Gate Contract

Gate define **política**, evidence define **prova**.

Campos: `id`, `condition`, `required_evidence`, `thresholds`, `aggregation`, `failure_action`, `waiver_policy`, `independent_verification`, `staleness_rules`.

## Conditional gates

Risk/impact pode injetar:

- security;
- accessibility;
- performance;
- compatibility/migration;
- database migration;
- visual regression;
- API contract;
- release/supply chain;
- documentation/readiness.

# 7. Failure Taxonomy

| Classe | Retry? | Ação |
| --- | --- | --- |
| Transient provider/tool | Sim, bounded | Retry/backoff ou alternate provider. |
| Deterministic validation | Não | Corrigir input/implementation. |
| Missing context | Após recontextualizar | Progressive context escalation. |
| Policy denied | Não automaticamente | Escalation/permission request. |
| Flaky/non-deterministic | Investigação | Classify/quarantine; nunca mascarar com retries infinitos. |
| Security stop | Não | Abort mutation, preserve evidence, specialist escalation. |
| Irreversible/data-loss risk | Não | Require recovery proof/human authority. |

# 8. Idempotency & Resume Contract

Todo step deve declarar `idempotent`, `idempotency_key`, `side_effects`, `resume_strategy`, `compensation`.

Operações externas como create issue/PR/release/deploy precisam de “already-exists detection” antes de retry.

# 9. Concurrency Contract

Para execução multi-agent:

- resource ownership;
- path/semantic leases;
- optimistic conflict detection;
- merge policy;
- stale base detection;
- no silent overwrite;
- parallel-read / serialized-write quando necessário.

# 10. Permission Model

Separar:

- **required:** capability sem a qual a tarefa não pode executar;
- **optional:** melhora resultado, mas não é obrigatória;
- **elevated:** write/process/network/admin/production etc. ativada somente por política e etapa.

A skill nunca autoeleva permissão. O Core/harness continua sendo autoridade.

# 11. Handoff Artifact

Campos mínimos:

- source/target role;
- task/goal/plan IDs;
- base commit/state;
- outputs/artifacts IDs;
- evidence IDs;
- files/resources changed;
- assumptions;
- decisions;
- unresolved questions;
- risks/blockers;
- retry/resume state;
- exact next action;
- context references.

# 12. Lifecycle & Freshness

Maturity recomendada para unificação com documentação existente: `draft → experimental → recommended/verified → stable` deve ser normalizada em uma taxonomia única antes da implementação. Não manter duas máquinas de estado concorrentes.

Promotion exige threshold por tipo. Deprecation exige replacement/alias/migration note quando houver. Knowledge sujeito a standards/toolchains deve conter `last_reviewed` e policy de staleness.

# 13. Definition of Done transversal

Nenhum componente pode ser considerado `stable` enquanto faltar qualquer item obrigatório de seu contrato ou enquanto sua eval crítica estiver abaixo do threshold aprovado.