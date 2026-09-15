# Documentation Eval Corpus

> Authority: repository-canonical test corpus (W13.2). Runner:
> `internal/documentation/evals_test.go`.

This corpus encodes the failure modes the Documentation Control Plane must
catch. Every case has an **expected outcome**; a case whose expectation stops
holding is a regression in the evaluator.

Cases live in `cases/*.json` and are executed by `go test ./internal/documentation/`.

## Case matrix

| ID | Case | Kind | Status |
|----|------|------|--------|
| D001 / D001b | Required knowledge must not be satisfied by a single generic keyword (with control) | `knowledge-match` | ✅ enforced |
| D002 / D002b | Evidence must bind to the claim it proves (with control) | `evidence-binding` | ✅ enforced |
| D003 | A lexical-only binding can never assert readiness | `evidence-binding` | ✅ enforced (W15) |
| D004 | Required knowledge without a requirement mapping is unbound | `evidence-binding` | ✅ enforced (W15) |
| D005 | A requirement with no claim is unbound | `evidence-binding` | ✅ enforced (W15) |
| D006 | Evidence proving an older revision is stale | `evidence-binding` | ✅ enforced (W15) |
| D007 | Evidence whose artifact is missing does not verify | `evidence-binding` | ✅ enforced (W15) |
| D008 | A low-authority claim cannot promote readiness | `evidence-binding` | ✅ enforced (W15) |
| D009 | A projection whose canonical source is another projection is a cycle | `projection-cycle` | ✅ enforced |
| D010 | An unresolved contradiction blocks readiness | `evidence-binding` | ✅ enforced (W15) |
| D011 | A superseded claim cannot promote readiness | `evidence-binding` | ✅ enforced (W15) |

The other corpora live beside this one — `evals/context/` (C001–C010, context
disclosure and dedup, W4), `evals/ui-specification/` (U001–U009, UI contracts,
W13.3) and `evals/reconstruction/` (R001–R004, publishing projections, W13.4).
See `evals/README.md`.

The failure modes that landed in later waves are covered by their own gates
rather than by cases here: translation and media drift are checked by content
digest in `internal/doclifecycle`, curated-region integrity by the
managed-region check inside `prumo docs verify`, and a documented CLI command
that no longer exists by the W16 context-rot guard.

## Case format

`knowledge-match`

```json
{ "id": "D001", "kind": "knowledge-match", "requirement": "...", "content": "...", "expect_satisfied": false }
```

`evidence-binding` without `requirement_map`/`claims` is a **lexical** binding and
can only ever report `unverified` (W15.5):

```json
{
  "id": "D003", "kind": "evidence-binding",
  "contract": { "id": "x", "required_knowledge": ["..."], "evidence_requirements": ["a"] },
  "sources": [ { "path": "docs/x.md", "content": "..." } ],
  "evidence": ["a"],
  "expect_state": "unverified"
}
```

`evidence-binding` (semantic, W15)

```json
{
  "id": "D006", "kind": "evidence-binding",
  "contract": { "id": "x", "required_knowledge": ["recovery guarantees"] },
  "sources": [ { "path": "docs/x.md", "content": "..." } ],
  "requirement_map": { "recovery guarantees": "REQ-RECOVERY" },
  "claims": [ { "id": "CLAIM-RECOVERY", "statement": "...", "status": "accepted",
    "authority": "canonical-documentation", "revision": 4, "requires": ["REQ-RECOVERY"],
    "evidence": [ { "id": "EV-1", "type": "test-suite", "target": "CLAIM-RECOVERY", "revision": "2", "verified": true } ] } ],
  "expect_state": "unverified",
  "expect_finding": "stale-evidence"
}
```

`projection-cycle`

```json
{ "id": "D009", "kind": "projection-cycle", "documents": [ { "path": "docs/a.md", "role": "projection", "canonical_source": "docs/b.md" }, { "path": "docs/b.md", "role": "projection" } ], "expect_findings": 1 }
```

## Adding a case

1. Add `cases/<ID>-<slug>.json` with an explicit expectation.
2. Add a row to the matrix above with its kind and status.
3. Run `go test ./internal/documentation/ -run TestDocumentationEvalCorpus`.
