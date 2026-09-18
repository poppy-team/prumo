# Python ↔ Go Conformance Strategy

## Purpose

Prove that Go v0.4 is a compatible implementation of the v0.3 protocol before new v0.4 subsystems become canonical. The Python runtime remains the oracle; this strategy does not replace the existing Python conformance suite.

## Comparison Contract

For identical fixture input, compare:

1. **Exit code** — success, validation failure, policy denied, configuration error, unavailable capability, internal error
2. **stdout** — exact for deterministic human output where compatibility requires; parsed structure for JSON
3. **stderr** — relevant diagnostics, excluding environment-specific noise
4. **JSON** — envelope/data/error codes, normalized only by explicitly approved rules
5. **Filesystem effects** — created, modified, deleted paths; file bytes where canonical
6. **Canonical artifacts** — `prumo.json`, Goals, plans, generated adapters and other protocol outputs
7. **Error behavior** — stable codes and preserved context/cause

## Fixture Layout

```text
conformance/
├── V03_BASELINE.json
├── fixtures/
│   ├── cli/
│   ├── goals/
│   ├── plans/
│   ├── schemas/
│   └── projects/
├── golden/
│   ├── stdout/
│   ├── stderr/
│   ├── json/
│   └── filesystem/
└── README.md
```

Existing conformance directories remain in place. New subdirectories are added only when fixtures exist.

## Fixture Categories

- Valid and invalid `prumo.json` / project profiles
- Goal lock, amendment, transition, illegal mutation
- Valid and cyclic Plan DAG
- Context pack and budget behavior
- Evidence, Gates, waivers, and Event stream
- FakeRuntime happy path, retry, fallback
- Resolver/workforce/model policy
- All v0.3 CLI commands and important error paths
- Compiler target output
- Migration and snapshot behavior
- Doctor diagnostics

## Golden Generation

Python v0.3 generates the reference:

```bash
python -m pytest tests/test_conformance.py
prumo-agent <command> <args> --json > conformance/golden/json/<case>.json
```

Go executes the same case:

```bash
go run ./cmd/prumo-agent <command> <args> --json
```

Golden updates require an intentional protocol decision, an updated schema/ADR, and review. Never regenerate all outputs as a convenience.

## Determinism

- Sort paths and collections where protocol specifies deterministic output
- Normalize timestamps only when the contract defines logical time
- Use isolated temporary directories
- Do not use LLMs for deterministic contract tests
- Do not compare environment-specific absolute paths without a normalization rule
- Record tool/runtime versions and fixture hash

## Gate Progression

| Gate | Requirement |
|------|-------------|
| M0 | Inventory and fixture matrix reviewed |
| M1 | Go CLI and test infrastructure green |
| M2 | Protocol + CLI critical contracts at 100% parity |
| M3 | Compiler outputs at 100% parity |
| M4+ | Python oracle runs in CI during transition |
| Python removal | Go critical conformance 100%, target release builds available, one critical-regression-free release cycle |

## Differential Runner

Do not build a general framework before cases justify it. Start with a small table-driven runner that invokes both binaries, captures exit/stdout/stderr, and compares fixture-specific expectations. Add filesystem snapshots only for commands with side effects.

**OPEN QUESTION**: Exact runner language and normalization library. Standard-library Go subprocess + JSON is preferred unless the fixture model proves insufficient.

## Non-Destructive Migration

Each migration fixture uses a copied temporary project. Before material changes, capture backup/checkpoint and verify post-state. `--dry-run` output is compared separately from applied migration output.

## Failure Reporting

A failure records:
- case ID;
- Python and Go commands;
- exit codes;
- stdout/stderr diff;
- JSON path diff;
- filesystem diff;
- runtime/tool versions;
- fixture hash.

A conformance failure blocks promotion; it is not silently accepted as a known difference.