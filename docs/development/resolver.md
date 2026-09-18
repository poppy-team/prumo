# Resolver

The Resolver is the heart of Prumo, responsible for dynamically selecting the right capabilities for a given context.

## Resolution Algorithm

The resolver applies filters in the following order:
1. Core (always included)
2. Project-Type
3. Stack
4. Features
5. Risk level
6. Bundles
7. Risk-rules
8. Transitive dependencies (resolving `requires` blocks)

## Explainability Traces

Use `prumo-agent explain` to view exactly why a specific skill was included or excluded during resolution.

## Determinism Guarantees

The resolver guarantees deterministic output for a given project configuration and catalog version.

## Testing Resolution

Add tests in `tests/test_resolver.py` using mock catalogs to ensure filtering logic remains sound.

## Select Criteria

Skills, agents, and recipes use the `select` block in their manifests to declare when they apply (e.g., `{"stack": ["python"]}`).
