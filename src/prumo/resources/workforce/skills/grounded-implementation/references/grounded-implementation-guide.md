# Grounded Implementation & Anti-Invention — Reference Guide

## 1. Core Concepts

### 1.1 Inspect Before Claim
No statement about repository reality — file existence, symbol definitions, CLI flags, test coverage, config values — is made without a tool inspection (read, grep, glob, bash) in the current turn. Memory of past states is treated as a hypothesis, re-verified on use. A claim without a cited inspection is indistinguishable from invention.

### 1.2 The UNKNOWN Protocol
Missing information moves through an explicit pipeline: `UNKNOWN → search authoritative source (repo → canonical docs → Living Book → maintainers) → resolve, record an explicit assumption with owner and expiry, or block the dependent slice`. Skipping straight from UNKNOWN to a plausible invention is the defining failure this skill prevents.

### 1.3 Scope Boundaries (IN / OUT / INCIDENTAL)
Every task materializes three sets: IN (files and behaviors to change), OUT (explicitly forbidden, e.g., locked contracts, unrelated modules), and INCIDENTAL (read-only context). Mutations outside IN stop immediately, even "obvious" drive-by fixes — those become separate tracked follow-ups.

### 1.4 Fail Closed on Missing Authority
Destructive, security-sensitive, or contract-mutating actions without canonical authority do not proceed on best judgment. The agent states what authority is missing, what it blocks, and the exact question or document that unblocks it — then stops that slice.

## 2. Patterns and Anti-Patterns

| Pattern (do) | Anti-pattern (avoid) |
|---|---|
| `grep` for the symbol, cite file:line, then use it | "The API probably supports `filter=`…" |
| Record `ASSUMPTION(owner, expiry)` when forced to proceed | Silent defaults baked into code without trace |
| Split the task: ship grounded slices, block invented ones | One big diff mixing verified and guessed changes |
| Quote the ADR/spec section that authorizes a contract change | "Best practice" cited as authority for locked decisions |
| Stop and ask for the missing credential/flag/contract | Guessing a secret format or endpoint from naming vibes |

## 3. Worked Example

Task: "Add `retry` support to the checkout client."

- Grounded slice: `grep -rn "checkout.NewClient" --include="*.go"` shows the constructor at `checkout/client.go:41` with an `Options` struct — extend `Options` with `MaxRetries`, wire it, test. Ship it.
- Blocked slice: no spec defines retry budgets for PSP calls. Record `ASSUMPTION: max 4 attempts, exponential backoff — owner: payments-team, expires 2026-10-07` and proceed only for the mechanical wiring, or block the budget constants until the payments team confirms. Never silently pick "3 retries" and bury it in the diff.
