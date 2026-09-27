# Authentication Security Audit — Customer Session Service

## Scope
- **Component**: login, refresh, password reset, and session revocation
- **Review date**: 2026-09-23
- **Assessor**: identity-security-agent
- **Identity source**: customer database plus OIDC provider

## Finding AUTH-01
- **Severity**: Critical
- **Category**: Refresh-token replay
- **Observed**: Reusing a rotated refresh token returned a new access token.
- **Impact**: A stolen refresh token could create a parallel authenticated session.
- **Remediation**: Atomically consume each token and revoke the entire family on reuse.

## Finding AUTH-02
- **Severity**: High
- **Category**: Account enumeration
- **Observed**: Password-reset responses differed for known and unknown emails.
- **Remediation**: Return the same response and timing class; audit requests internally with normalized account IDs.

## Verified Controls
- Passwords use Argon2id with unique salts and a 64 MiB memory cost.
- Access tokens expire after 10 minutes; refresh tokens rotate on every use.
- Recovery links expire after 20 minutes and are single-use.
- MFA failures are rate-limited and written to the security audit stream without secrets.

## Release Decision
AUTH-01 blocks release. Require passing replay, revocation, and enumeration tests.
