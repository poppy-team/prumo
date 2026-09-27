# [Bug] Expired OAuth token returns 500 instead of 401

## Summary
The profile endpoint dereferences a null token when an OAuth access token expires during an active session.

## Context and Reproduction
- Version: `v2.4.1`
- Environment: Ubuntu 22.04, OpenJDK 17
- Steps: sign in with Google, expire the token, request `GET /api/v1/profile`.
- Actual: `500 Internal Server Error` at `AuthController.java:142`.
- Expected: `401 Unauthorized` with code `AUTH_TOKEN_EXPIRED`.

## Acceptance Criteria
- [ ] The endpoint returns `401` and `AUTH_TOKEN_EXPIRED` for an expired token.
- [ ] The regression test fails on the old controller and passes with the null guard.
- [ ] Logs contain no token or user secret.

## Technical Notes
- Related issue: #781
- Relevant files: `AuthController.java`, `AuthControllerTest.java`
- Evidence: sanitized exception excerpt in issue #779
