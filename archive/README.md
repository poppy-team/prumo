# Archive

Code kept for the record, not for the build. Every Go file under this directory
carries `//go:build ignore`, so the module does not compile it and does not
carry its dependencies.

## `tui-poc` — the from-scratch terminal client (H10 spike)

The client that was built from nothing and ran the H10 spike to completion. It
is superseded as a client strategy by [ADR 013](../docs/adr/013-tui-foundation.md):
the shipping client is `prumo-tui`, built on the archived OpenCode Go view layer.

It is kept because deleting code is a separate, deliberate act, and because the
spike is the evidence for things that outlived it:

- the protocol's approval operations (`approve`/`deny`, `awaiting_approval`,
  `pending_permissions`), which the spike forced into existence;
- the redraw baseline recorded in ADR 009;
- `TestBoundaryNoInternalImports`, the shape of the client/core boundary that
  ADR 013 then made structural by moving the client into its own module.

It is excluded from the build, so its tests do not run. That is the cost of
archiving rather than maintaining, and it is why the new client carries its own.
