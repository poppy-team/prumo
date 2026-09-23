# Grounded Implementation Verification Checklist

## Inspection Before Claim
- [ ] Every cited file, symbol, command, flag, config key, and test was inspected in the current workspace.
- [ ] Claims include a navigable source reference or recorded command output.
- [ ] Repository reality takes precedence over memory, naming guesses, and generic conventions.

## Scope Boundaries
- [ ] IN, OUT, and INCIDENTAL sets are explicit before mutation.
- [ ] No file outside IN is changed, including opportunistic cleanup or refactoring.
- [ ] Locked contracts, canonical decisions, and generated artifacts are not silently changed.

## Authority and Assumptions
- [ ] Missing requirements move through search, resolution, explicit assumption, or blocking.
- [ ] Assumptions name an owner, rationale, scope, and expiry or review date.
- [ ] Destructive, security-sensitive, and public-contract work fails closed without authority.

## Grounded Change
- [ ] New symbols follow inspected repository naming and integration patterns.
- [ ] Dependencies and APIs are verified at their actual definitions and call sites.
- [ ] Unverified slices are omitted or recorded as blocked rather than invented.

## Evidence and Stop Conditions
- [ ] Tests and verification commands validate the changed behavior, not only file presence.
- [ ] Remaining uncertainty and follow-up work are explicit.
- [ ] `scripts/verify.sh` exits with code 0 from the repository root.
