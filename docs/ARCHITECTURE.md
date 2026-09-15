# Architecture

Prumo separates durable truth from generated execution views and temporary agent context.

```text
Project repository
├── Canonical knowledge (Markdown)
├── prumo.json + Goals/manifests/intelligence (JSON)
├── Goals + evidence
├── Selected AI workforce
└── Project history
          │
          ▼
  Structural indexes / graph / caches
  (derived; SQLite/runtime)
          │
          ▼
 Progressive Context Engine contract
 ├── Context Packs / task maps
 ├── structural + symbol retrieval
 ├── Working Context Capsule
 ├── Context Compiler / Renderer
 ├── token & output budgets
 └── stopping + garbage collection
          │
          ▼
 Replaceable orchestrator / model adapters
          │
          ▼
 Implementation → tests → review → docs delta → intelligence → GC
```

## Canonical vs generated

Canonical sources are intentionally human-maintainable:

- Markdown: product/user/developer/operations/architecture/spec/ADR knowledge.
- JSON: `prumo.json`, Goals, workforce manifests, model policy and compact durable Project Intelligence.

Generated artifacts include indexes, summaries, context renderings, site output and platform adapters. They can be rebuilt and never outrank canonical sources.

## Three persistence classes

### Canonical Knowledge

Long-lived project truth in Git.

### Project History

Compact durable information needed to understand project evolution: Goals, evidence pointers, costs, task summaries and accepted debt.

### Working Context

Search results, WCC versions, temporary summaries, intermediate findings, child results and retrieval traces. This belongs in runtime state and should be garbage-collected.

## Capability resolution

The workforce resolver remains deterministic. Project type, stack, features and risks select the smallest useful set of Agents, Skills and Recipes. Skill dependencies are resolved transitively.

The same principle now applies to context: do not load the full workforce and do not load the full knowledge base.

## Context architecture

The preferred strategy is deterministic-first:

1. direct/local context;
2. structural or symbol retrieval;
3. known Context Pack/task map;
4. progressive targeted expansion;
5. single-level isolated delegation when it has clear value;
6. stop.

Deep recursive execution is not a framework default.

## Documentation architecture

A project exposes explicit user, developer, operations and agent surfaces. The PRUMO routes by intent. Documents may remain moderately sized and are chunked virtually by headings/symbols instead of being split into microfiles solely for LLM consumption.

## Project configuration

Projects use one canonical root configuration, `prumo.json`, instead of duplicating project profile + project manifest truth.

## Runtime

Derived indexes, knowledge graph caches and temporary context should use a runtime database such as `.prumo/runtime/prumo.db` and remain gitignored.

Language-specific structural analysis is an adapter concern. C# projects should prefer Roslyn for semantic/symbol analysis rather than treating source code only as text.

## Extensibility

Project-specific overrides remain in the project. Reusable patterns may become framework capabilities through an RFC/change proposal. New persistent file formats require an ADR and evidence that the added maintenance cost is justified.
