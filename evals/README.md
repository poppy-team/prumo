# Behavioral Eval Corpus

> Authority: repository-canonical test corpus (W13). Each corpus encodes the
> failure modes one layer of the Documentation Control Plane must catch. A case
> whose expectation stops holding is a regression in the evaluator, so the
> corpora are the executable form of the design claims.

Every corpus runs under `go test ./...` — no model, no network and no API key is
required. Each case declares an explicit expected outcome; several pair a
violation with a **control** that must still verify, so a check cannot be
satisfied by rejecting everything.

| Corpus | Path | Runner | Covers |
|---|---|---|---|
| Documentation semantics | `evals/documentation/` | `TestDocumentationEvalCorpus` | requirement → claim → evidence binding, projection cycles, contradictions, staleness, supersession, authority |
| Context disclosure | `evals/context/` | `TestContextEvalCorpus` | L0–L4 disclosure levels, canonical-preference dedup, budget exclusion with a reason, sufficiency |
| UI specification | `evals/ui-specification/` | `TestUISpecificationEvalCorpus` | contract evidence, profile coverage, state vocabulary, duplicate ids, token/theme gaps, contrast, accessibility exception paths |
| Reconstruction | `evals/reconstruction/` | `TestReconstructionEvalCorpus` | byte-exact Markdown projection, route stability, agent surfaces never published as human pages, historical records excluded from AI retrieval |
| Interface map | `evals/interface-map/` | `TestInterfaceMapEvalCorpus` | mandatory position, conditional alternatives, state/token closure, symbol resolution, declared absence, edge endpoints, derivation scope |

## Case counts

- `evals/documentation/` — D001–D011 (13 files, several with controls).
- `evals/context/` — C001–C010.
- `evals/ui-specification/` — U001–U009 (10 files).
- `evals/reconstruction/` — R001–R004.
- `evals/interface-map/` — M001–M013 (several with controls).

## Adding a case

1. Add `cases/<ID>-<slug>.json` with an explicit expectation, and a control case
   when the failure mode could be detected by over-rejecting.
2. Add a row to the corpus README matrix (see `evals/documentation/README.md`
   for the format).
3. Run the corpus runner for that directory.

## What the corpora deliberately do not encode

Model-judged quality ("is this prose good?") is out of scope: a corpus that
needs a model to decide whether it passed cannot gate a build. Model-assisted
critique exists only as an optional gauntlet round that may never override a
deterministic failure (W19).
