# GitHub PR Feedback Handling — Technical Reference Guide

## 1. Core Concepts

**Thread inventory**: every unresolved comment classified as fix (code change), test (coverage to add), question (needs an answer), or dispute (needs discussion). Unclassified threads are how concerns slip through.

**Minimal targeted diffs**: each fix addresses exactly the reviewer's ask; feedback pushes never bundle opportunistic refactoring. Reviewer-found bugs land with a failing-first regression test.

**Reply-before-resolve**: each thread gets a reply (what changed + commit SHA, or why not with rationale). Threads resolve when the reviewer is satisfied or the diff auto-obsoletes them — never by bulk-resolve.

## 2. Patterns

- **Blocking-first ordering**: `request-changes` threads before nits; CI-blocking concerns before style.
- **Suggestion-block apply**: `gh pr checkout <n>` + commit reviewer `suggestion` blocks verbatim where intent matches, crediting the reviewer in the reply.
- **Dispute template**: "Keeping current approach because <technical reason + evidence>; alternative <X> rejected due to <trade-off>. Happy to revisit if <condition>." — stays in-thread, stays technical.
- **Re-review packet**: summary comment mapping threads → commits + CI green + explicit re-request to each blocking reviewer (`gh pr review --request-reviewer` or UI).

## 3. Anti-Patterns

- Force-pushing over review history without noting it (destroys thread-to-diff mapping).
- "Fixed" replies with no commit reference.
- Bulk-resolving 20 threads to fake readiness.
- Arguing tone instead of technicals; Kennedy-style "per my last comment" replies.

## 4. Worked Example

PR #233 has 11 threads: 2 blocking (null-guards on `user.email`, missing index on `orders(user_id)`), 5 nits, 3 questions, 1 dispute (reviewer wants a new abstraction; author shows it would add a third query per request with benchmark numbers). Fixes land in 2 commits (`a1b2c3d`, `e4f5a6b`) plus a regression test failing-first on the null-email path. Every thread answered; dispute stays open-but-documented; CI green; re-request sent to both blocking reviewers with the thread→commit map. Approved and merged same day.

## 5. Verification Pointers

- Count unresolved threads via `gh pr view <n> --comments` — expect zero.
- Each fix commit maps to ≥1 thread; each thread has a reply.
- CI green on the final push; no force-push without a mapping note.
