# ADR 013 — OpenCode Go as the terminal client foundation, and the client/core module boundary

**Status:** Accepted
**Date:** 2026-09-17
**Context:** H10 (terminal client), W22 (interface map), GAP-046/GAP-047, `prumo-tui`
**Supersedes:** the from-scratch client approach recorded in `docs/product/tui-spike-h10.md`

## Context

The H10 spike built a terminal client from nothing: a Bubble Tea model, a
palette, a bounded timeline, and a daemon supervisor. It did its job — it forced
the protocol to grow the approval operations, it produced the redraw baseline,
and it made the client/core boundary a machine-checked invariant. Its five
acceptance criteria are met or measured.

It is also a small surface compared with what a terminal client of this product
has to carry: streaming markdown, syntax highlighting, a file picker, an
editor, diff rendering, dialog stacks, sessions and theming. Rebuilding all of
that from an empty `tea.Model` is a large cost for a part of the system that is
not where Prumo's value lives.

An archived Go implementation of exactly that surface exists under a license
that permits derivation. Its provenance was verified rather than assumed:

- repository `github.com/opencode-ai/opencode`, **MIT** (Copyright (c) 2025
  Kujtim Hoxha), archived read-only on 2025-09-18 at commit
  `73ee493265acf15fcd8caab2bc8cd3bd375b63cb`, signed by its author;
- the archive notice states the project continued as **Crush**, which is where
  the maintained successor lives. Crush ships under `FSL-1.1-MIT`, which is not
  an open-source license.

That distinction decides the legal shape of this decision: the archived MIT
snapshot may be adapted; the FSL successor may only be read.

The candidate import is large — roughly 135 Go files, a dependency tree
containing three cloud provider SDKs, a SQLite migration stack, an LSP client
with 370 KB of vendored protocol, and viper. Almost none of that belongs to a
client whose runtime is the Prumo harness.

## Decision

**1. Adopt the archived OpenCode Go snapshot as the terminal client's view-layer
foundation.** Prumo does not rename a fork; it takes a proven view layer and
replaces everything underneath it.

**2. The client is its own Go module.** `prumo-tui` (module
`github.com/raillen/prumo-tui`) depends on the Prumo core only through
`github.com/raillen/prumo/sdk/prumo`. The core does not depend on the client.

The direction document proposed placing the client at `internal/tui/`. That is
rejected here, and the reason is the document's own first principle: a client
that must not know about providers, quotas or orchestration must not live inside
the tree that grants that access. `internal/` would make the boundary a test
result instead of a structural fact, and it would put the client's heavy
dependency tree inside the core module.

**3. Import scope is the view layer only.** The upstream runtime packages are
deliberately not imported, because Prumo already owns each concern behind the
harness contracts:

| Upstream package | Prumo owner | Why not imported |
|---|---|---|
| `internal/llm` (providers, agents, tools, prompts) | `internal/harness/model`, `gateway`, `modelregistry`, `toolgateway` | Providers are the harness's. Importing it would pull `anthropic-sdk-go`, `openai-go`, Google `genai`, Azure and AWS SDKs into a client that must stay provider-neutral |
| `internal/db` (SQLite, sqlc, goose) | the daemon's store | Sessions and history belong to the daemon. Not importing it also removes any question of a second SQLite driver colliding with ADR 007 |
| `internal/lsp` | `internal/harness/lsp` | Already implemented behind the harness contracts |
| `internal/app`, `internal/session`, `internal/pubsub`, `internal/message`, `internal/config` | the daemon plus the SDK | The client's runtime is the harness, reached over the protocol |

Where the view layer needs one of those shapes, the client carries a **local
view-level type** and the adapter fills it from protocol events. The view model
is not the domain model, and keeping them separate is what makes the client
replaceable later.

**4. The from-scratch spike is archived, not deleted.** `tui/` moves to
`archive/tui-poc/`. What it produced outlives it: the protocol's approval
operations, `awaiting_approval` as an observable state, the redraw baseline in
ADR 009, and the boundary invariant this ADR now makes structural. Deleting the
code is a later, deliberate act.

**5. Sequence, and nothing in parallel.** Import and rebrand; then the Charm v2
port; then the harness integration; then the archive of the spike and the map
retarget. Framework migration, architectural change and new behaviour do not
share a step, so a regression has one candidate cause.

**6. Provenance is recorded, not remembered.** `LICENSE` (MIT) and
`THIRD_PARTY_NOTICES.md` land with the import, the notices naming the upstream,
its license, the exact revision and what was and was not derived. The upstream
`LICENSE` travels inside the derived module.

## Consequences

- **The core module loses its only external dependencies.** Today
  `charm.land/bubbletea/v2` and `charm.land/lipgloss/v2` are direct requires
  purely because the client lived inside the core. After this ADR the core is
  standard-library only, and the client's heavy tree is contained where it
  belongs.
- **CI must cover two modules.** `go test ./...` at the repository root silently
  skips a nested module. The workflow gains explicit steps for the client, and
  the Go version must be aligned — the workflow currently pins an older toolchain
  than `go.mod` requires, which the second module makes load-bearing.
- **The map moves with the client.** `docs/ui-ux/interface-map.json` declares its
  sources as the old package path and names its symbols. Archiving the spike
  without retargeting it fails `prumo ui verify` with unresolved symbols. The
  retarget is part of the work, not an afterthought.
- **The client polls, it is not pushed to.** Upstream was written against a live
  event broker; the harness protocol exposes replay plus status. The adapter
  bridges the two, and the difference is documented rather than hidden. A push
  channel is a candidate for its own increment.
- **One more artifact to keep current.** The imported view layer is now Prumo's
  to maintain. That cost is accepted deliberately: it buys a client surface that
  would otherwise take far longer to reach, and the boundary makes the decision
  reversible — the client can be replaced without touching the core.
