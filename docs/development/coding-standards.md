# Prumo Coding Standards (Go)

> Clean Code is permanent implementation policy — applied pragmatically, not dogmatically.

## Naming

- Names express **intent and domain**, not accidental implementation
- `FindRoot()`, not `GetDir()`; `VerifyGoalLock()`, not `CheckGoal()`
- Packages represent **real domain boundaries**: `protocol`, `project`, `resolver`, `compiler`
- Avoid generic suffixes: no `ManagerHelper`, `CommonUtil`, `DataProcessor`

## Functions

- Single cohesive responsibility, clear contract
- Small enough to understand at a glance (< 50 lines ideal, < 100 max)
- Explicit error returns — no silent failures, no ignored errors
- `context.Context` only where cancellation/deadline actually cross boundaries (I/O, network, long operations)

## Packages

- Represent **real domain boundaries**, not file organization
- Start consolidated; split when responsibility justifies
- No `utils`, `common`, `misc`, `helpers` — every package has domain semantics
- No circular dependencies (CI-enforced)

## Errors

```go
// Wrapped errors preserve cause
if err != nil {
    return fmt.Errorf("failed to resolve workforce for profile %q: %w", profileID, err)
}

// Sentinel errors for domain invariants
var ErrGoalLocked = errors.New("goal is locked and cannot be modified without amendment")

// Never panic for user/config errors
// panic only for programmer errors (unreachable branches, invariant violations)
```

## Interfaces

- Small, close to consumer, defined where needed
- No interface before real substitution/test/integration need
- Prefer structs + functions over premature abstractions
- Existing approved interfaces: `Repository`, `EventSink`, `DerivedIndex`, `HarnessTransport`, `Clock`, `EmbeddingProvider` (future)

## Dependencies

- **Standard library first** — `encoding/json`, `os`, `path/filepath`, `io/fs`, `sync`, `context`, `time`, `errors`, `fmt`
- **Zero CGO by default** — preserves single-binary distribution
- **`database/sql` without ORM** — first implementation
- **`embed.FS`** for schemas/templates/resources traveling with binary
- **No new external dependency without justification**:
  - What measurable problem does it solve?
  - Why is stdlib insufficient?
  - Can it be derived instead of canonical?
  - What's the removal cost?

## I/O Boundaries

- I/O at edges; protocol logic testable without real filesystem
- Use `io/fs.FS` abstraction for file access (enables `embed.FS` swap)
- CLI commands are **thin adapters** → delegate to `internal/app.Service`
- Storage implements ports defined by consumers

## Testing

- Table-driven tests for domain logic
- Pure functions preferred — no mocks when simple test doubles suffice
- Integration tests use `t.TempDir()`, synthetic Git repos
- Golden tests for compiler outputs: `testdata/` + `conformance/golden/`
- LLM never used where output should be deterministic

## Definition of Done

A change to Prumo Core is complete when:
1. Compiles and formats (`gofmt -l .` empty)
2. Passes unit + relevant integration tests (`go test -race ./...`)
3. Passes `go vet ./...` and `staticcheck ./...`
4. Passes conformance/golden tests if protocol-affecting
5. No circular package dependencies
6. Explicit error handling (no ignored errors)
7. Affected contracts/docs updated
8. Sufficient evidence for risk level
9. No critical TODO without tracking issue

## Anti-Overengineering Guardrail

Before adding new service, package, database, interface, or daemon:
- What measurable problem does it solve?
- Why is the current mechanism insufficient?
- Can the feature be derived instead of canonical?
- What's the removal cost?
- Does it belong in core or as provider/plugin?

## Portability Rule

> No behavior required to interpret an Prumo project may depend exclusively on a SaaS, LLM, IDE, plugin, or external database.

## No-Nos

- Singletons with global mutable state
- Reflection/metaprogramming where explicit structure suffices
- Duplicating Prumo rules in adapters
- `panic` for user errors
- Interfaces with one implementation and no test/integration justification
- Gigabytes of context in one function/package