# 79.G — Context, Agentic Workflow, Orchestration, MCP e Indexing

> Authority: canonical specification.
> Logical ID: 79 G
> Source: Notion Living Book (3d89bb7d023f81d9b143cd2d4b14ffa2)
> Status: Skill Package de referência para 79 G — Context, Agentic Workflow, Orchestration, M.


<aside>
🧠

Esta família controla **o que o modelo sabe, quem trabalha, como agentes cooperam e como ferramentas externas entram na execução**. Erros aqui amplificam todos os outros domínios; por isso várias melhorias são P0.

</aside>

# Camadas de responsabilidade

```
project-intelligence → fatos/provenance do projeto
prumo-navigation/indexers → descoberta e mapa estrutural
lean-progressive-context/context-optimization → composição de contexto
prompt-engineering → instrução probabilística versionada
agentic-workflow-design → semântica do workflow
orchestration-multi-agent → coordenação runtime
traycer-orchestration → adapter/provider específico
mcp-integration/mcp-tooling → integração e execução de ferramentas MCP
```

| Skill | Pri. | Lacuna | Aprimoramento | Aceite |
| --- | --- | --- | --- | --- |
| **agentic-workflow-design** | P0 | Workflow precisa ser state machine resistente. | States/transitions, checkpoints, compensation, retryability, idempotency, dead ends, human escalation. | Workflow simulável antes de executar. |
| **orchestration-multi-agent** | P0 | Concorrência/falha/custo precisam de contrato. | Task ownership, leases, handoff, dead-letter, retry, parallelism, conflict resolution, budget e specialist routing. | Simulation cobre crash e conflicting output. |
| **prompt-engineering** | P0 | Prompt é código probabilístico sem disciplina equivalente. | Versioning, schema outputs, injection boundaries, model variance, examples, regression evals, context minimization. | Prompt change compara baseline. |
| **project-intelligence** | P0 | Facts podem ficar stale ou misturar inferência. | Origin, confidence, freshness, generated/vendor policy, contradictions e incremental invalidation. | Cada fact tem provenance+confidence+freshness. |
| **lean-progressive-context** | P0 | Conceito central precisa de thresholds. | Context tiers, escalation conditions, token ceilings, caching, invalidation, summary-loss evals, provenance. | Menos contexto sem queda de correctness no baseline. |
| **context-optimization** | P1 | Compressão pode remover evidência essencial. | Value scoring, duplication, freshness, provenance preservation, compression-loss tests, cache invalidation. | Quality/token Pareto medido. |
| **prumo-navigation** | P0 | Navigation index pode ficar stale. | Generated index, rename/move detection, fallback search, scope/provenance, monorepo strategy. | Move/rename não deixa referência silenciosamente inválida. |
| **traycer-orchestration** | P1 | Princípios genéricos e provider-specific podem se misturar. | Assumptions, API/version compatibility, task-state mapping, provider failure handling; delegar orchestration geral. | Zero duplicação normativa com multi-agent core. |
| **mcp-integration** | P1 | Lifecycle/negociação precisam aprofundar. | Discovery, capability negotiation, auth, reconnect, timeout, pagination, provenance, version compatibility. | Connector contract fixtures. |
| **mcp-tooling** | P1 | Semântica de tool execution heterogênea. | Schema discovery, arg validation, cancellation, idempotency, partial results, file handles, typed errors. | Provider-neutral tool contract suite. |
| **clang-context-indexing** | P1 | C/C++ depende de configuração de TU. | compile_commands, macros, conditional compilation, headers/modules, generated/vendor policy, invalidation. | Symbol result informa configuração/TU. |
| **roslyn-context-indexing** | P1 | .NET depende de solution/project/TFM. | Projects, TFMs, symbols, source generators, analyzers, partial types, generated code, invalidation. | Result informa project+TFM. |
| **rust-analyzer-context-indexing** | P1 | Cargo features/cfg alteram código ativo. | Workspace/features/cfg/target/build scripts/proc macros, offline mode, stale handling. | Result registra active features/target. |
| **tree-sitter-context-indexing** | P1 | Parse tolerante pode mascarar ERROR nodes. | Grammar/version, queries, ERROR reporting, embedded languages, incremental edits, fallback. | Index quality report sinaliza parse uncertainty. |
| **ts-morph-context-indexing** | P1 | TS depende de tsconfig/module resolution. | Project refs, aliases, mixed JS/TS, decorators, generated code, incremental invalidation, compiler version. | Result aponta tsconfig/project. |
| **language-tooling** | P1 | Pode sobrepor indexers e editor tooling. | Tornar orchestrator de LSP/compiler/indexer/formatter/debugger; delegar detalhes por linguagem/provider. | Seleciona apenas adapters aplicáveis. |

# Context Indexer Contract compartilhado

Todos os indexers devem emitir envelope comum: provider, language, config/target, files indexed, symbols/edges, confidence, parse/compiler errors, generated/vendor exclusions, cache key, freshness, partial-result marker.

# Activation metrics

- precision/recall de skill selection;
- context recall: facts necessários presentes;
- context precision: informação irrelevante evitada;
- token overhead;
- stale context rate;
- wrong-provider/indexer selection;
- resume continuity without transcript.

# Multi-agent conflict policy

- leitura paralela livre quando não houver efeito;
- escrita requer ownership/lease ou optimistic check;
- output conflict deve gerar reconciliation step, não “última resposta vence”;
- base revision/commit é parte do handoff;
- retry de external mutation requer idempotency key/existence check.

# Exit Gate

O resolver consegue explicar seleção e rejeição; indexers usam envelope homogêneo; context budgets são medidos; multi-agent execution resiste a crash/conflict; prompt/workflow changes passam evals antes de promoção.