# Repository Settings Change Record

## Repository
- Name: `acme/api`
- Default branch: `main`
- Change date: 2026-09-23
- Owner: platform-security

## Applied Policy
- `main` requires 2 reviews and up-to-date branches.
- Required checks: `lint`, `test (ubuntu-22.04, node 20)`.
- Force pushes and branch deletion are blocked.
- `production` requires two named reviewers.
- `NPM_TOKEN` moves to the production environment scope.

## Verification
- Before payload: `artifacts/repo-main-before.json`
- After payload: `artifacts/repo-main-after.json`
- Direct push to `main` rejected with `GH006`.
- Fork pull request environment contained no repository secrets.
