#!/usr/bin/env node
/**
 * Modern JavaScript (ES2024+) Security, Event Loop & Memory Hygiene Reference
 */

// 1. WeakMap for Leak-Free Metadata Caching
const entityMetadataCache = new WeakMap();

// 2. Safe Dictionary (Null Prototype Defense against Prototype Pollution)
export function createSafeDictionary() {
  return Object.create(null);
}

export function safeSetProperty(dict, key, value) {
  // Reject prototype pollution attack vectors
  if (key === "__proto__" || key === "constructor" || key === "prototype") {
    throw new TypeError(`Forbidden property key: ${key}`);
  }
  dict[key] = value;
  return dict;
}

// 3. Event Loop Yielding (Macrotask scheduling to prevent UI/IO starvation)
export function yieldToEventLoop() {
  return new Promise((resolve) => setTimeout(resolve, 0));
}

// 4. Asynchronous Task with Cooperative AbortController Cancellation
export async function fetchWithTimeout(url, { signal, timeoutMs = 2000 } = {}) {
  const timeoutController = new AbortController();
  const timer = setTimeout(() => {
    timeoutController.abort(new Error(`Operation timed out after ${timeoutMs}ms`));
  }, timeoutMs);

  try {
    const combinedSignal = signal
      ? AbortSignal.any([signal, timeoutController.signal])
      : timeoutController.signal;

    // Simulate async network request
    if (combinedSignal.aborted) {
      throw combinedSignal.reason;
    }

    await yieldToEventLoop();
    return { ok: true, url, timestamp: Date.now() };
  } finally {
    clearTimeout(timer);
  }
}

// 5. Immutable Array Manipulations (ES2023)
export function processScores(scores, bonusIndex, bonusValue) {
  // Use non-mutating methods (.toSorted, .with) to preserve input array integrity
  const withBonus = scores.with(bonusIndex, scores[bonusIndex] + bonusValue);
  const sorted = withBonus.toSorted((a, b) => b - a);
  return { withBonus, sorted };
}

// 6. Verification Runner
async function main() {
  console.log("=== Modern JavaScript (ES2024+) Systems Hygiene Demonstration ===");

  // Test Prototype Pollution Defense
  const dict = createSafeDictionary();
  safeSetProperty(dict, "user_id", "usr_1001");
  console.log("Safe dictionary property set:", dict["user_id"]);
  console.log("Is Null Prototype (Immune to Pollution):", Object.getPrototypeOf(dict) === null);

  try {
    safeSetProperty(dict, "__proto__", { evil: true });
  } catch (err) {
    console.log("Blocked prototype pollution attempt:", err.message);
  }

  // Test Non-mutating Array Methods
  const originalScores = [10, 50, 30, 20];
  const { withBonus, sorted } = processScores(originalScores, 0, 15);
  console.log("Original scores (unmutated):", originalScores);
  console.log("Updated with bonus (with):", withBonus);
  console.log("Sorted descending (toSorted):", sorted);

  // Test AbortController Timeout Pipeline
  const response = await fetchWithTimeout("https://api.prumo.dev/health", { timeoutMs: 1000 });
  console.log("Async response with cooperative cancellation:", response);
}

if (process.argv[1] && process.argv[1].endsWith("safe_module.js")) {
  main();
}
