# Backend API Endpoint Specification Template

## 1. Service Overview
- Service: Orders API owned by the Checkout team, contact checkout-api@acme.example
- Base URL: https://api.acme.example/v1
- Contract source: openapi/orders-1.4.0.yaml, diffed with oasdiff against 1.3.2, zero breaking changes
- Peak traffic: 800 RPS sustained, 1,600 RPS burst during flash sales
- SLO: p95 latency under 250 ms, error rate under 0.1 percent, availability 99.95 percent monthly

## 2. Endpoint Inventory

| Method & Path | Auth | Purpose | p95 Budget |
|---|---|---|---|
| POST /v1/orders | JWT scope orders:write | Create order, idempotent via Idempotency-Key | 250 ms |
| GET /v1/orders | JWT scope orders:read | Cursor-paginated order list, default 25 max 100 | 180 ms |
| GET /v1/orders/ord_9f31 | JWT scope orders:read | Retrieve single order by public ID | 120 ms |
| POST /v1/orders/ord_9f31/cancel | JWT scope orders:write | Cancel unpaid order, 409 if already paid | 200 ms |
| POST /v1/webhooks/acme-events | HMAC-SHA256 signature | Payment provider callbacks with dedup IDs | 300 ms |

## 3. Error Catalog
- order_not_found → 404, retryable false, message: Order ord_9f31 does not exist
- order_already_paid → 409, retryable false, message: Order ord_9f31 is paid and cannot be cancelled
- payment_declined → 422, retryable false, message: Card ending 4242 was declined, code card_declined
- rate_limited → 429 with Retry-After 12, retryable true with backoff
- upstream_timeout → 503, retryable true, message: Payment gateway timed out after 2 s

## 4. Rate Limits & Quotas
- Standard clients: token bucket 600 requests per minute, burst 120, headers RateLimit-Remaining exposed
- Webhook ingress: 2,000 events per minute per signing key, overflow queued to Kafka topic billing.events.retry
- Payload caps: 1 MB JSON bodies, 100 items per bulk request, 429 beyond quota

## 5. Verification Evidence
- Contract diff: oasdiff breaking count 0, changelog entry added for new cancel endpoint
- Tests: pytest 214 passed, handler branch coverage 92 percent, Pact consumer AcmeWeb verified green
- Load: k6 at 1,600 RPS for 10 minutes, p50 95 ms, p95 210 ms, p99 380 ms, zero 5xx
- verify.sh: exit code 0 on 2026-09-23 run by checkout-api pipeline job 4471
