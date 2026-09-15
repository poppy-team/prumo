# Validation and Quality

Framework quality gates:

- JSON schemas validate generated contracts;
- canonical catalog loads and IDs are unique;
- resolver produces deterministic manifests;
- Goal transitions reject illegal state changes;
- `DONE` requires evidence;
- project bootstrap generates Markdown/JSON canonical state;
- YAML is not generated;
- legacy YAML input remains readable during migration support;
- platform compilation uses only selected workforce entries;
- runtime/generated context is not treated as canonical;
- documentation entrypoints remain synchronized;
- context/output policies have explicit budgets/stopping rules.

## LPC regression quality

Context optimization is only a success if task quality remains acceptable.

Benchmarks should compare legacy vs LPC/PCA by:

- task success/correctness;
- tests/evidence;
- input tokens;
- output/intermediate tokens;
- cost;
- latency;
- number of permanent files/docs created.

Do not optimize token count in isolation.

## Documentation quality

Validate:

- links/entrypoints;
- required audience coverage where applicable;
- documentation impact in behavior changes;
- no unnecessary task-context/summary proliferation;
- generated site is rebuildable from canonical sources.

## CI

GitHub Actions runs Python compile/tests, framework validation and CLI smoke tests. Projects using Prumo must add domain-specific gates.
