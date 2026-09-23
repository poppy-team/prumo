# Network Security Audit — Merchant API Edge

## Scope
- **Component**: public API load balancer and outbound payment client
- **Review date**: 2026-09-23
- **Assessor**: network-security-agent
- **Environment**: production edge and service mesh

## Finding NET-01
- **Severity**: High
- **Observed**: The payment client accepted TLS 1.0 when a compatibility flag was enabled.
- **Remediation**: Remove the downgrade path and enforce TLS 1.2 minimum with TLS 1.3 preferred.

## Finding NET-02
- **Severity**: High
- **Observed**: The outbound client had no connect timeout and could accumulate idle connections indefinitely.
- **Remediation**: Set a 3-second connect timeout, 10-second response-header timeout, bounded body reads, and finite per-host pools.

## Control Verification
- Certificate chain, hostname/SAN, EKU, and revocation policy are validated.
- Internal payment service requires mTLS with a short-lived client certificate.
- Egress is default-deny for `payments.internal` and approved public domains only.
- `Host` and forwarded-host values use an exact authorized-host set.
- WebSocket handshakes validate `Origin` and cap frame and message sizes.

## Release Decision
NET-01 and NET-02 block release. Capture negative handshake and exhaustion tests before approval.
