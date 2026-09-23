# PR Review: Session cookie hardening

## Scope
- PR: #842
- Base: `main`
- Changed paths: `src/auth/*`, `src/http/session.ts`, `test/auth/session.test.ts`
- Risk classification: High

## Review Checklist
- [x] Authorization and session ownership verified on logout and refresh.
- [x] Cookie uses `Secure`, `HttpOnly`, `SameSite`, and `__Host-` prefix.
- [x] SAST and dependency gates are clean.
- [ ] Blocking finding: refresh-token family revocation test is still missing.

## Findings
- **HIGH**: A refresh token remains usable after logout in the current fixture. Request changes until `logout_revokes_refresh_family` is added.

## Verdict
Request changes. Re-review after the failing-first test and the final CI run are linked.
