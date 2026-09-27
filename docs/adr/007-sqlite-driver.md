# ADR 007: SQLite Driver for Derived Runtime State

# Status

Accepted (2026-09-17). The measurement below is a follow-up, not a precondition: the driver lives behind a port, so swapping it is one file, and the decision was blocking ADR 009.

# Context

- FRAMEWORK.md permits SQLite only for derived/runtime state (Working
  Context, indexes, retrieval traces); canonical state stays Markdown/JSON/Git.
- GAP-034 asked for a driver decision before the first SQLite-backed
  runtime component lands (`internal/storage/sqlite` per dependency rules).
- Hard requirements: single-binary distribution (zero cgo by default),
  cross-platform builds (linux/macOS/windows, amd64/arm64), no external
  service, testability in hermetic CI.
- Go driver candidates: `modernc.org/sqlite` (pure Go, no cgo, larger
  binary, slightly slower), `mattn/go-sqlite3` (cgo, fast, complicates
  cross-compilation), `crawshaw.io/sqlite` (cgo, batch APIs).

# Decision

Adopt `modernc.org/sqlite` as the default driver for derived runtime state,
behind the `internal/storage.DerivedIndex` port:

1. Pure-Go keeps the single-binary, cross-platform distribution invariant
   (ADR 001 direction) with zero toolchain requirements for contributors.
2. All SQLite access goes through the storage port; the driver is an
   implementation detail replaceable without touching call sites.
3. If measured workloads (WCC volume, index builds) show cgo-driver
   advantage beyond tolerance on target hardware, an ADR amendment may
   swap the driver behind the same port — measured, not assumed.

# Consequences

- No cgo in default builds; CGO stays opt-in per build tag, never required
  for Prumo core features.
- Slight binary size increase is accepted as the cost of portability.
- The driver is not vendored into domain packages; only
  `internal/storage/sqlite` imports it.
