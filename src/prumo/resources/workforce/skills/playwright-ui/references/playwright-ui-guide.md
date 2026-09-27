# Playwright UI Testing Reference Guide

## 1. Core Concepts

### 1.1 Semantic Locators
Playwright resolves `getByRole('button', { name: 'Place order' })` against the accessibility tree, so tests break only when user-visible semantics change — exactly when they should. Priority order: role, label, placeholder, text, test-id. Test-ids (`data-testid="order-confirm"`) are escape hatches for role-less custom widgets, not the default.

### 1.2 Auto-Waiting and Web-First Assertions
`expect(locator).toHaveText('Paid')` polls until the condition holds or the timeout (default 5 s, configurable per assertion) expires. This replaces sleep/retry spaghetti: the test describes the end state, Playwright handles timing. Fixed `waitForTimeout` calls are the leading cause of slow, flaky suites and are capped at 500 ms with justification.

### 1.3 Fixtures Over Shared State
`test.extend` fixtures build isolated worlds per test: an API client creates `SKU-8842 x2` in a fresh cart, the page loads with `storageState` for the buyer role, teardown deletes the order. Tests become order-independent and shard-safe. The one shared-setup exception: authenticating once per role and reusing `storageState` files.

### 1.4 Sharding, Retries, and Flake Math
Flake rate = `flaky_results / total_results` over the nightly 50-run job. Retries mask flakes: cap at 2, allow only for network-class failures (DNS, ECONNRESET, 5xx from staging), and report retry counts. A suite at 3% flake with 2 retries looks green while hiding real breakage — hence the 2% gate on unretried outcomes.

### 1.5 Trace Viewer Debugging
`trace.zip` records DOM snapshots, network, console, and actions per step. Opening a CI trace replays the failure click-by-click with time travel. Require traces on first retry so the first red run is already debuggable; videos cover what traces cannot (native dialogs, file pickers).

## 2. Patterns and Anti-Patterns

| Pattern (do) | Anti-pattern (do not) |
|---|---|
| `getByRole` + web-first assertions + API fixtures | XPath copied from DevTools + `sleep(3000)` + shared staging cart |
| `storageState` per role, one login setup each | Logging in through UI in all 200 tests |
| Trace/screenshot/video on failure, keyed by test + shard | "Failed in CI, passes locally", no artifacts |
| Quarantine with owner + expiry, fix within 5 days | `test.skip` with no ticket and no date |
| `eslint-plugin-playwright` in lint gate | Floating promises and missing awaits in test code |

## 3. Minimal Example: Isolated Checkout Test (TypeScript)

```ts
// tests/checkout.spec.ts
import { test, expect } from './fixtures/api-fixtures';

test('buyer completes checkout for SKU-8842', async ({ page, api, buyer }) => {
  const order = await api.createCart(buyer, [{ sku: 'SKU-8842', qty: 2 }]);
  await page.goto(`/checkout?cart=${order.cartId}`);

  await page.getByRole('button', { name: 'Place order' }).click();
  await expect(page).toHaveURL(/confirmation/);
  await expect(page.getByRole('heading', { name: 'Order confirmed' })).toBeVisible();
  await expect(page.getByTestId('order-total')).toHaveText('$149.98');

  await api.deleteOrder(buyer, order.id); // teardown even on failure via fixture
});
```

The API fixture seeds and cleans state, semantic locators survive CSS refactors, and URL plus heading assertions synchronize without sleeps. On failure the shared config attaches `trace.zip`, a screenshot, and a retry video to the CI artifact bundle.
