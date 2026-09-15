# M5 Documentation System v2 — Dogfood Report

> **Historical record.** This report predates semantic readiness v2 (W15). The
> "readiness: ready" result below was produced by the lexical evaluator and is
> **not** an authoritative verdict: `prumo docs readiness` now reports lexical
> contracts as `unverified`. Current readiness semantics: ADR 010 and
> `docs/architecture/documentation-control-plane.md`.

Run against the Prumo repository:

```bash
prumo --json docs audit --path .
prumo --json docs readiness --goal M5 --path .
```

Observed result:

- Applicable profiles: `core-software`, `cli`.
- Applicable contracts: architecture, CLI, installation, product vision, scope, security, testing.
- Readiness: ready.
- Blocking contracts: none.
- Blocking questions: none.

The engine still reports missing semantic knowledge when a binding lacks sources or required knowledge is absent; the current repository satisfies the applicable C-G06 contracts. Tracked deltas are stored locally under `.ai/docs/deltas/` and move through repository review before canonical documentation changes.

Status: **M5 Exit Gate Passed** — see `docs/governance/m5-exit-gate.md`.
