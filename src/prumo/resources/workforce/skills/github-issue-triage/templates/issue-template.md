# Triage Decision Record

## Issue
- Number: #824
- Title: Profile upload intermittently returns 413
- Reporter: external customer
- Component: `web/media`
- Canonical duplicate: none

## Decision
- Classification: bug
- Priority: P1 — production checkout media path is degraded for 18% of uploads.
- Route: `area/backend`, `verified`
- Milestone: v0.5.0
- Owner: media team through CODEOWNERS

## Evidence
- Reproduced in the staging container with a 12 MiB fixture.
- Similar reports #801 and #809 were linked; no canonical duplicate exists.
- No credential, customer name, or internal hostname is included in the public comment.

## Follow-Up
- Reproducer and fixture are linked to issue #824.
- Reporter receives an update within the 24-hour SLA.
