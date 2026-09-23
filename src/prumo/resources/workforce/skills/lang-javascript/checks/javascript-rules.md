# Modern JavaScript (ES2024+) Quality & Security Invariants Checklist

## 1. Modern Syntax & ESM Architecture
- [ ] Strictly native ECMAScript Modules (`import`/`export`), no CommonJS `require()` / `module.exports`.
- [ ] Zero usage of `var` across all files (`const` by default, `let` strictly when variable reassignment is needed).
- [ ] Non-mutating array operations utilized (`.toSorted()`, `.toReversed()`, `.toSpliced()`).
- [ ] Deep cloning utilizes platform-native `structuredClone()`.

## 2. Event Loop & Asynchronous Hygiene
- [ ] All Promise rejections are handled via `await` inside `try/catch` or chained `.catch()`.
- [ ] Zero recursive microtask scheduling that could cause event loop starvation (`queueMicrotask` recursion).
- [ ] Asynchronous I/O operations (HTTP, WebSocket, timers) accept and respect an `AbortSignal`.
- [ ] Timeouts and intervals (`setTimeout`, `setInterval`) have corresponding cleanup routines (`clearTimeout`, `clearInterval`).

## 3. Prototype Pollution & Security Defense
- [ ] Zero calls to `eval()` or `new Function()`.
- [ ] Zero direct assignments to `__proto__` or modification of built-in prototypes (`Object.prototype`, `Array.prototype`).
- [ ] Dictionaries accepting dynamic keys use `new Map()` or `Object.create(null)`.
- [ ] Unsanitized HTML string injection (`innerHTML`, `outerHTML`) is strictly forbidden (use `textContent` or sanitized DOM nodes).

## 4. Memory Leak Prevention
- [ ] Event listeners are registered with an `AbortSignal` option (`{ signal }`) or explicitly removed upon component unmount.
- [ ] Caches storing object references use `WeakMap` or `WeakSet` to avoid preventing garbage collection of referenced objects.
- [ ] Detached DOM nodes are not retained in global or long-lived module-level arrays.
