# ADR 012 — Interface map artifact, declared-versus-derived authority, and per-project projections

**Status:** Accepted
**Date:** 2026-09-15
**Context:** H10 (terminal client), W5 (UI contract decomposition), W22 (interface map)
**Supersedes:** none

## Context

Four repository contracts already oblige a project with an interface to document
it: `ui.component-contracts` ("component inventory, anatomy, component states,
component interactions, implementation trace"), `ui.information-architecture`
("navigation model, surface hierarchy, naming conventions, task findability"),
`ui.screen-inventory` ("screen inventory, screen purpose, entry points") and
`ui.layout` ("layout model, responsive behavior, size classes, overflow
handling").

`schemas/ui-component-contract.schema.json` has described a single component
since W5 — with `anatomy`, `states`, `interactions` and `implementation.symbol`,
and a task item ("W5.10 require component/state → implementation symbol
traceability edges") marked delivered. No producer ever existed. Every one of the
four contracts was satisfied by `applicability: not-applicable` with a reason
that said "owed by the first layout increment".

The lesson is not that the contracts were wrong. It is that an obligation whose
only consumer is a waiver is an obligation nobody implements, and the gate stays
green while the artifact is absent.

## Decision

### 1. One artifact answers all four obligations

`docs/ui-ux/interface-map.json`, validated by `schemas/ui-interface-map.schema.json`,
compiled by `internal/uimap`, exposed as `prumo ui`. Four partial documents about
one object is how they drift apart; one object described from four angles cannot.

### 2. Declared and derived, with declared always winning

- **Declared:** structure, labels, roles, intent, placement, states, tokens,
  interconnections. A compiler cannot infer what a button is for.
- **Derived:** implementation symbols, and the elements the declaration omits. A
  compiler can prove those.
- **Precedence:** the declaration wins for everything it states. Derivation fills
  what it omits. Where both describe the same element with different facts, that
  is an **error**, never a silent pick.

The third rule is the load-bearing one. Without it, derivation becomes an
override channel nobody reviews, and the artifact stops being authored.

Derivation modes, configured per project: `off`, `verify-only` (default;
report what the map omits without touching it) and `fill-gaps` (list it for
review in a dedicated region). The default is conservative because adding rows to
a reviewed artifact is a change the author should opt into, while reporting an
omission is always safe.

Derivation is a seam, not a rule set: `SymbolDeriver` is an interface, and
`GoSymbolDeriver` ships here because this repository is Go. Another stack adds an
adapter rather than teaching the core about its framework.

### 3. Position is semantic; geometry is optional and per platform

A terminal has no stable pixel geometry — coordinates change with every resize.
Every element therefore declares a region, an order within it and an alignment,
optionally a weight and a stacking direction. Geometry exists in the schema for
platforms where pixels are stable, tagged with its platform and unit.

Mutually exclusive siblings may share a position **only** by declaring a
`condition`. Two unconditional siblings may not: nothing would tell a renderer
which to draw. This rule came from the artifact's own first run — the TUI's four
stages are alternatives, and a layout model without conditions read them as four
panes stacked.

### 4. Composition is a tree; interconnection is a graph

The tree carries what contains what, which is what a reader looks for first.
Focus order (which can cycle), data flow between distant branches, events,
overlays, navigation and blocking relationships are typed edges. A tree alone
would make a focus cycle and an overlay either invisible or falsely hierarchical.

### 5. The map is canonical; every projection is derived

Projections live under `.prumo/runtime/interface-map/`, never in `docs/`: derived
summaries do not become canonical files. Each carries the map digest and the
configuration digest in its first line, and a stale projection is a gate failure —
a reference describing the previous interface misleads the reader who opens it.

### 6. Scope is a product decision, per project and per audience

This is a framework capability for any project Prumo builds, not a feature of
this repository. A project that declares a UI capability (`ui`, `tui`,
`desktop-gui`, `web-application`) or a UI `project.type` gets the map by default;
a CLI- or library-only project gets nothing, because an interface map with no
interface is empty ceremony.

`ui.interface_map` in `prumo.json` narrows it:

| Key | Meaning |
|-----|---------|
| `enabled` | Master switch. `false` produces nothing and states the reason. |
| `targets` | `developer`, `site`, `agent`. A map for the code agent and the developer but not a public site is a legitimate scope. |
| `derivation` | `off`, `verify-only`, `fill-gaps`. |
| `path` | Override the canonical location. |

An empty `targets` list is refused rather than producing nothing: enabled with no
output is a silent no-op, and silent no-ops are what this ADR exists to end.

## Consequences

- The four contracts are bound to a real artifact, and `ui.component-contracts`
  no longer carries a waiver.
- The gate fails when an element has no position, a state contradicts the state
  matrix, a token is undeclared, an edge endpoint is missing, a symbol does not
  resolve, a projection is stale, or two unconditional siblings share a position.
  Every one of those is a rule a renderer needs anyway.
- An element that is described but not built is expressible: `status:
  not-implemented` with a required `absent_reason`. The TUI's approval prompt uses
  it, which puts the protocol gap (GAP-046) in the interface itself and not only
  in the gap register. Without the escape hatch, authors would invent symbols to
  get a green gate — the failure mode this artifact was born from.
- Cost: the map is one more artifact to keep current. It is bounded by the gate
  rather than by discipline: changing the interface without changing the map fails
  the build, and the projections are regenerated by one command.
- A non-Go project needs a deriver adapter before its symbols can be proven. Until
  then it can declare the map with `derivation.sources` empty and elements marked
  `not-implemented`, which is honest but weaker.

## Alternatives rejected

- **Derive the map from the implementation.** Automatic and framework-specific: it
  violates the rule that platform behavior lives in adapters, and it cannot infer
  intent, labels or placement — the three things a reader of an interface map came
  for.
- **Declare only, without symbol verification.** Cheaper to write, and it ages
  into a description of an interface that has changed. This is precisely the state
  the four contracts were in.
- **A tree without edges.** Loses focus cycles, overlays and cross-branch data
  flow.
- **Four separate documents per contract.** Four descriptions of one object drift;
  the drift is invisible because each document is individually plausible.
- **Committed Markdown projections.** Reviewable in a pull request, but it makes a
  derived file look canonical and puts a second source of truth one commit away.
- **Always on for every project.** A map for a library with no interface is
  ceremony, and it would train reviewers to ignore the artifact.
