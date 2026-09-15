# ADR 005: CLI Layer as Composition Root (Import Policy)

# Status

Accepted

# Context

- An earlier doc rule stated `cmd/` may import only `app` and `protocol`.
- Recorded as a non-P0 contradiction in the promotion report (2026-09-11):
  the shipped CLI imports ~43 internal packages (app, protocol, harness
  tree, connectors, install, planning, documentation, tooling, and others).
- The CLI is the user-facing composition root: flag parsing, service wiring,
  envelope output, and command routing. Domain logic lives in `internal/`;
  `cmd/prumo` contains no domain rules beyond dispatch and presentation.
- Boundary tests pin what matters: `cmd` never imports vendor packages, and
  `internal/` never imports `cmd` (checked by SDK boundary conformance).

# Decision

Formalize the existing reality as policy:

1. `cmd/prumo` is the composition root. It may import any `internal/`
   package to wire commands.
2. `cmd/prumo` must not contain domain logic: decisions, algorithms, and
   invariants belong in `internal/` packages behind typed APIs.
3. `internal/` packages must never import `cmd/`.
4. New user-visible behavior ships with a command in `cmd/prumo` plus a
   tested implementation in `internal/`; the wiring layer stays thin.

# Consequences

- The documented rule and the code agree; the promotion-report contradiction
  is resolved by ADR instead of a silent rewrite.
- Future extraction (e.g., a `prumo-code` repo building on the public SDK)
  targets `internal/` and `sdk/`, never `cmd/`.
- Reviewers judge `cmd/prumo` diffs by wiring-only expectations.
