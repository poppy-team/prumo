# Playwright UI Testing — Verification Checklist

## 1. Selectors & Synchronization
- [ ] New tests use `getByRole`/`getByLabel`/`getByPlaceholder`/`getByText`/`data-testid` in priority order; zero positional CSS or XPath selectors added.
- [ ] Strict mode clean: every locator resolves to exactly one element in the projects under test.
- [ ] No `waitForTimeout` above 500 ms without a comment naming the external dependency; network waits use `waitForResponse` or URL assertions.
- [ ] Auto-waiting assertions (`toBeVisible`, `toHaveText`, `toHaveURL`) used instead of manual polling loops.

## 2. Fixtures & Test Isolation
- [ ] Each test seeds its own users, carts, and orders via API fixtures; no reliance on shared staging leftovers.
- [ ] Test data cleaned up in `afterEach`, even on failure (try/finally or fixture teardown).
- [ ] Auth state stored per role (`admin.json`, `buyer.json`) via `storageState`; no login-through-UI in more than one setup test per role.
- [ ] Secrets and PII excluded from fixtures committed to the repo; staging-only values injected via environment.

## 3. Browsers, Sharding & Stability
- [ ] Suite runs on Chromium, Firefox, and WebKit projects with per-project pass rates reported.
- [ ] CI shards at least 4 ways; total suite wall time under 12 minutes on the reference runner.
- [ ] Retries capped at 2 and restricted to network-identified flakes; retry counts reported per test.
- [ ] Nightly 50-run stability job shows flake rate under 2%; chronic failures quarantined with owner and expiry.

## 4. Failure Evidence & Regression Proof
- [ ] Trace-on-first-retry, screenshot-on-failure, and video-on-retry enabled in shared config.
- [ ] Every failure in the verification window links a trace viewable in Trace Viewer.
- [ ] `eslint-plugin-playwright` passes with no awaited-expect or missing-playwright-await violations.
- [ ] Automated verification script `scripts/verify.sh` executes with exit code 0.
