# Web Application Security Audit Report

## Scope
- Component: `account.settings` web route
- Assessor: web-security-agent
- Review date: 2026-09-23
- Evidence: response-header scan, CSP report, and request tests

## Findings
- **HIGH**: A user-supplied avatar URL could resolve to `169.254.169.254`; the fetcher now rejects private, loopback, and link-local addresses.
- **MEDIUM**: The session cookie lacked `SameSite`; it is now `__Host-session; Secure; HttpOnly; SameSite=Lax`.
- **LOW**: A legacy redirect accepted absolute URLs; redirects now require a relative path or trusted host.

## Controls Verified
- CSP uses a per-request nonce and blocks `unsafe-eval`.
- State-changing requests require a session-bound CSRF token.
- CORS uses an explicit origin list and never combines wildcard origins with credentials.
- Uploads are stored outside the web root and served as attachments.

## Evidence
- XSS, CSRF, SSRF, CORS, and redirect regression suites pass.
- The header scanner reports the expected CSP, frame, and cookie attributes.
