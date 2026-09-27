# M7 Adoption Engine — Exit Gate

Status: **COMPLETE (M7 Exit Gate Passed — ADOPTION READY)**

## Exit gate criteria

Milestone M7 exit gate (per `docs/development/phases.md` and `docs/runtime/adoption-engine.md`):
an arbitrary existing repository can be audited without mutation; observed facts and inferences
are kept separate; confidence and evidence accompany all inferences in the Confidence Ledger;
M5 documentation contracts evaluate candidate bindings; ambiguity enters the Living Plan via
uncertainty questions; migration proposals are formal, non-destructive, and review governed;
the scanner is incremental and revision/branch-aware; malicious content and secret leaks are prevented.

---

### 1. Arbitrary existing repo can be audited without mutation (Zero Forced Layout)

- Proven by automated regression tests across 6 brownfield fixture repositories (`internal/adoption/evals_test.go`):
  - `TestBrownfieldCorpusGoCLI`: Go CLI with non-Prumo layout (`cmd/`, `internal/`, standard `README.md`, `docs/architecture.md`).
  - `TestBrownfieldCorpusWebMonorepo`: Multi-package web application (React, Vite, Express, PostgreSQL).
  - `TestBrownfieldCorpusMaliciousInjection`: Untrusted repository containing prompt injection directives.
  - `TestBrownfieldCorpusSecretsEnv`: Repository containing `.env` and sensitive API keys.
  - `TestBrownfieldCorpusStaleConflictingDocs`: Repository with conflicting historical documentation.
  - `TestBrownfieldCorpusNoDocs`: Project without any documentation files.
- Invariant: `hashDirectory` verifies that before and after `RunAdoptionAudit`, the repository tree and all file contents remain 100% byte-for-byte identical. No files are moved, renamed, or deleted.

---

### 2. Observed facts and inferences are kept strictly separate

- Factual ground truth is captured exclusively in `ObservedFact` (`internal/adoption/fact.go`) with `ConfidenceFactual` and concrete extraction sources (filenames, directory structures, parsed manifest fields).
- Derived assumptions are isolated as `ClassificationSignal`, `ProfileCandidate`, `CapabilityProposal`, and `CandidateBinding`.
- Neither scanner nor classifier elevates inferences to facts.

---

### 3. Confidence and evidence accompany all inferences (Confidence Ledger)

- Canonical `ConfidenceLedger` (`internal/adoption/ledger.go`, schema `schemas/confidence-ledger.schema.json`) registers all factual and inferred statements.
- Invariant enforced by `LedgerEntry.Validate()`: Every low and medium confidence entry requires human confirmation (`RequiresConfirmation: true`) and links back to concrete observed fact IDs in `Evidence`.
- `prumo-agent adopt --strict` rejects execution if unconfirmed inferences or unresolved contradictions remain.

---

### 4. M5 evaluates candidate bindings correctly

- Non-Prumo documentation layouts are mapped to Prumo canonical documentation contracts (`internal/adoption/mapping.go`, schema `schemas/mapping-candidate.schema.json`).
- High-confidence bindings are proposed for contracts required by detected profiles (e.g. `product.vision`, `system.architecture`, `testing.strategy`).
- Missing required contracts are reported in `DocCoverageSummary` without failing the audit.

---

### 5. Ambiguity enters the Living Plan / Open Questions

- `GenerateQuestions` (`internal/adoption/uncertainty.go`, schema `schemas/adoption-resolution.schema.json`) transforms unconfirmed inferences into interactive interview questions.
- Unresolved questions block strict adoption until an explicit decision is recorded by a human operator or authorized actor.
- Non-interactive pass (`ResolveNonInteractive`) deterministically applies default choices and records provenance.

---

### 6. Migration proposals are dry-run and review governed

- Adoption Engine does not apply ad-hoc changes. Confirmed findings produce formal `AdoptionMigrationProposal` items (`internal/adoption/migration.go`, schema `schemas/adoption-migration-proposal.schema.json`).
- Every proposal requires Review Queue approval before application (`review_required: true`).
- `DryRun` evaluates preconditions (e.g., `manifest_absent:prumo.json`) and produces unified diff previews without disk mutations.
- `Apply` enforces approval invariant (`ProposalStatusApproved`), executes atomic reversible changes, and records audit evidence in `migrations.JournalEntry` with SHA-256 integrity hash.
- CLI flags implemented in `cmd/prumo/adopt_commands.go`:
  - `prumo-agent adopt --propose-migration`
  - `prumo-agent adopt --dry-run`
  - `prumo-agent adopt --apply`

---

### 7. The scanner is incremental and revision/branch-aware

- Uses D3 repository index and `runtime.InspectRepository` (`internal/adoption/scanner.go`).
- Captures `branch`, `revision`, and `dirty` working tree state.
- Supports `ScanBudget` (`MaxFiles`, `MaxBytes`) with graceful continuation markers.
- Evaluates SHA-256 file hashes for change tracking across revisions.

---

### 8. Malicious content and secrets security invariants hold

- **No secret ingestion**: `.env` and credential files are detected (`env-presence:.env`) for risk disclosure, but secret values are never read, indexed, or persisted in facts or ledger (`content: [REDACTED_BY_SECURITY_POLICY]`). Tested in `TestBrownfieldCorpusSecretsEnv`.
- **Zero false canonical promotion**: Prompt injection in third-party README/AGENTS files (`TestBrownfieldCorpusMaliciousInjection`) gains zero policy authority. All candidate bindings remain `inferred-state` with zero promotion to canonical specifications.

---

## Evidence (Dogfood Audit on Prumo)

- Full audit executed against the Prumo repository itself (`TestDogfoodPrumoSelfAudit`):
  - Detected languages: Go, Python, Shell.
  - Detected app-type: CLI.
  - Detected frameworks and toolchains: standard Go toolchain, pytest, JSON Schema.
  - Discovered existing Prumo artifacts: `prumo.json`, canonical docs, contracts, schemas.
  - Output rendered via `prumo-agent adopt` and `RenderHumanReport` with zero file modifications.

---

## Test Verification

- `go test -race ./...` — 100% pass across all packages.
- `go vet ./...` — clean.
- `gofmt -l .` — clean.
- `python -m pytest -q` — 136 pass (Python oracle preserved).
- Adoption Engine suite (`internal/adoption`): 48 tests passing (100%).
- CLI suite (`cmd/prumo`): all tests passing (100%).

---

## Conclusion

Milestone M7 (Adoption Engine) is **COMPLETE**. All acceptance criteria and exit gate invariants are satisfied. The framework is declared **ADOPTION READY**.
