# Third-party notices

Prumo is distributed under the MIT license (see `LICENSE`). It also contains
code derived from the works below. Each entry names the upstream, its license,
the exact revision used, and what was derived from it.

## OpenCode (Go implementation)

- **Project:** OpenCode — a powerful AI coding agent, built for the terminal
- **Upstream:** https://github.com/opencode-ai/opencode
- **Revision:** `73ee493265acf15fcd8caab2bc8cd3bd375b63cb` (2025-09-18, the final
  commit; cryptographically signed by its author)
- **License:** MIT — Copyright (c) 2025 Kujtim Hoxha
- **Status upstream:** archived (read-only) on 2025-09-18 for provenance; the
  project continued under the name Crush, developed by the original author and
  the Charm team.

**What Prumo derived:** the terminal client's view layer — layout primitives,
chat surface, dialogs, command palette, theme set and styling — adapted into the
separate `prumo-agent tui` module, rebranded for Prumo and ported to the Charm v2
stack. Prumo did **not** take the upstream runtime: its provider integrations,
SQLite persistence, LSP client and agent loop are not part of this repository,
because the Prumo harness owns those concerns behind its own contracts.

**What Prumo explicitly did not take:** Crush, the actively maintained successor
project, ships under `FSL-1.1-MIT`, which is not an open-source license. It was
studied as a behavioural and UX reference only. No Crush code is present here,
and no part of Prumo is structured as a dependency on it.

The upstream `LICENSE` text is preserved verbatim inside the derived module at
`prumo-agent tui/LICENSE.opencode`.

## Go module dependencies

Direct and indirect Go dependencies of both modules are permissively licensed
(principally MIT and Apache-2.0) and are listed in each module's `go.mod` and
`go.sum`. This file records code that was **derived into** the repository, not
every library that is compiled against it.
