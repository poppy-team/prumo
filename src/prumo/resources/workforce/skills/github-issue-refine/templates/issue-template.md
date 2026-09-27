# [Enhancement] Make checkout retries idempotent

## Problem
A repeated checkout request can create two payment intents for the same cart.

## Acceptance Criteria
- [ ] The same `Idempotency-Key` returns the original payment intent on retry.
- [ ] A conflicting key returns `409 IDEMPOTENCY_CONFLICT`.
- [ ] `test_duplicate_checkout_is_idempotent` passes with a deterministic fake gateway.
- [ ] p95 checkout latency remains below 400 ms with 100 seeded requests.

## Dependencies
- Blocked by #812: payment adapter request correlation.
- Related ADR: ADR-014 Payment boundary.

## Non-Goals
- Changing payment-provider selection or adding a new checkout currency.
