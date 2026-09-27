# API Security Audit — Merchant Invoice Endpoints

## Scope
- **Component**: `/v1/merchants/{merchantId}/invoices`
- **Review date**: 2026-09-23
- **Assessor**: api-security-agent
- **Contract**: OpenAPI 3.1 `merchant-billing.yaml`

## Finding API-01
- **Severity**: High
- **Category**: BOLA / broken object-level authorization
- **Observed**: The invoice lookup validated the API key but not merchant ownership.
- **Evidence**: `GET /v1/merchants/other/invices/481` returned another tenant's invoice.
- **Remediation**: Scope the repository query by authenticated merchant ID and test cross-tenant IDs.

## Finding API-02
- **Severity**: Medium
- **Category**: Resource consumption
- **Observed**: `limit` accepted arbitrary 64-bit values.
- **Remediation**: Enforce integer schema, default `20`, maximum `100`, and reject unknown fields.

## Control Verification
- Authorization matrix: pass for 12 cross-tenant cases.
- Contract fuzzing: pass; malformed requests return 400/422, never 500.
- Quotas: 100 requests/minute per merchant; 429 includes `Retry-After`.
- Errors: RFC 7807; no stack trace or internal ID exposed.

## Release Decision
API-01 is release-blocking. Re-run the authorization suite after remediation.
