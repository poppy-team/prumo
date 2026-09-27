# GitHub Issue Refinement — Technical Reference Guide

## 1. Core Concepts

**Intent preservation**: the reporter's problem statement survives refinement verbatim in meaning; everything else (criteria, labels, scope) sharpens around it. If intent is ambiguous, ask — never reinterpret.

**Testable criteria**: each checkbox names an observable verification ("rejects empty titles with `400` + error code `E-102`", "p95 latency < 200 ms on staging over 100 requests"). Adjectives without measurements are rejected back to draft.

**Dependency mapping**: `blocked by #N`, `blocks #M`, required ADRs, and area labels (`area/backend`) make the issue schedulable. Hidden second deliverables split into linked issues.

## 2. Patterns

- **One-sentence restatement**: open the refined body with "Problem: ..." quoting intent; get reporter acknowledgment for contested restatements.
- **Verification-method suffix**: append "(verify: ...)" to each criterion so implementers know how done is proven.
- **Non-goals section**: explicit out-of-scope line ("Non-goals: migration of historical data") stops scope creep at implementation.
- **Label hygiene pass**: exactly one type, one severity (P0–P3), current milestone, no contradictory pairs (`wontfix` + `P1`).

## 3. Anti-Patterns

- Rewriting the problem into a preferred feature ("actually what you want is...").
- Criteria like "works correctly" or "fast" with no observable test.
- Smuggling new scope during grooming without reporter/owner acknowledgment.
- Mega-issues bundling independent deliverables under one checkbox list.

## 4. Worked Example

Issue #412 "Search is slow" → refined: Problem: "prefix search p95 exceeds 800 ms on 2 M-record staging index." Criteria: "p95 < 200 ms over 100 seeded queries (verify: k6 script `search-load.js`)"; "no result regressions on fixture `search-golden.json` (verify: `go test ./search -run Golden`)"; Non-goals: "relevance ranking changes." Dependencies: blocked by #398 (index migration); labels `enhancement`, `P1`, `area/backend`; milestone v0.5. Reporter confirms intent in comment before implementation starts.

## 5. Verification Pointers

- Every checkbox pairs with a verification method; spot-check two by mentally executing them.
- Before/after summary recorded with intent-preservation note.
- No criterion references undefined fixtures or environments.
