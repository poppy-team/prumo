# Backend API Development — Verification Checklist

## 1. Contract & Versioning
- [ ] OpenAPI 3.1 document (or .proto / GraphQL SDL) is committed and generates server stubs without warnings.
- [ ] oasdiff (or buf breaking) report shows zero undeclared breaking changes against the previous released spec.
- [ ] Routes are versioned with a /v1 path prefix; deprecated routes return Sunset and Deprecation headers.
- [ ] Consumer migration notes exist for every renamed, removed, or retyped field.

## 2. Validation & Error Handling
- [ ] Every request body, query string, and path parameter is validated by a strict schema; unknown fields are rejected on strict endpoints.
- [ ] JSON bodies are capped at 1 MB with a 413 response beyond the limit.
- [ ] Errors follow RFC 9457 problem-details with stable machine-readable codes such as order_not_found.
- [ ] Status codes are deterministic: 400 validation, 401 unauthenticated, 403 unauthorized, 404 missing, 409 conflict, 422 semantic, 429 rate limited, 5xx server-only.

## 3. Authentication, Authorization & Rate Limiting
- [ ] JWT signature, expiry, audience, and scopes are verified on every request; opaque tokens are introspected with caching under 60 s TTL.
- [ ] Tenant isolation is enforced at the query layer; cross-tenant access attempts return 403 and are logged.
- [ ] Per-client token-bucket rate limits (600 req/min, burst 120) return 429 with a Retry-After header.
- [ ] Service-to-service calls use mutual TLS or signed service tokens with 5-minute expiry.

## 4. Pagination, Filtering & Idempotency
- [ ] Collection endpoints use opaque cursor pagination with default page size 25 and max 100; offset pagination is absent on large tables.
- [ ] Filter, sort, and field-selection parameters are allow-listed; unknown parameters return 400.
- [ ] POST and PUT mutations honor Idempotency-Key with 24-hour replay storage returning the original response.
- [ ] Webhook deliveries sign payloads with HMAC-SHA256 and support at-least-once redelivery with deduplication IDs.

## 5. Observability, Tests & Load Evidence
- [ ] Structured logs, OpenTelemetry traces, and RED metrics (rate, errors, duration) exist per endpoint.
- [ ] Contract and handler tests reach 90 percent branch coverage; k6 load at 2x peak RPS shows p95 under 300 ms with zero 5xx.
- [ ] scripts/verify.sh executes with exit code 0.
