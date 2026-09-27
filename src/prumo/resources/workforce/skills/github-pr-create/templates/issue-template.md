# [PR Draft] Add idempotency keys to checkout requests

## Objective
Prevent duplicate payment intents when a browser retries `POST /checkout`.

## Changes
- Added a validated `Idempotency-Key` header at the HTTP boundary.
- Stored key, request hash, and response reference in the payment adapter.
- Added duplicate, conflict, and timeout tests.

## Verification
- `go test ./checkout/...` — pass
- `go test -race ./checkout/...` — pass
- Staging seeded retry run — one payment intent for 100 requests

## Linked Issues
- Fixes #824
- Blocked by #812

## Risk and Rollback
- Risk: a client that reuses a key with a different body receives `409` instead of a second charge.
- Rollback: disable the header check while retaining request correlation.

## Review State
- Draft until security and payments owners approve the new boundary.
