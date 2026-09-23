# Modern JavaScript Event Loop, Security & Memory Architecture Reference

## 1. Prototype Pollution Vulnerabilities & Mitigation
JavaScript's prototypical inheritance allows properties of `Object.prototype` to be inherited by nearly all objects. If an application performs a recursive deep-merge or property assignment using untrusted keys (`__proto__`, `constructor`, `prototype`), an attacker can pollute the global object prototype:

```javascript
// DANGEROUS: Recursive merge without prototype key guard
function unsafeMerge(target, source) {
  for (const key in source) {
    if (typeof source[key] === "object") {
      if (!target[key]) target[key] = {};
      unsafeMerge(target[key], source[key]); // Attacker passes key = "__proto__"
    } else {
      target[key] = source[key];
    }
  }
}
```

### Mitigation Strategies:
1. **Key Filtering**: Reject keys named `"__proto__"`, `"constructor"`, or `"prototype"`.
2. **Null-Prototype Objects**: Use `Object.create(null)` for dictionary lookups; objects without prototypes cannot be polluted.
3. **`Map` Collections**: Use native `Map` for arbitrary user-keyed collections.
4. **Prototype Freezing**: Execute `Object.freeze(Object.prototype)` at application startup to make all prototypes immutable.

---

## 2. Event Loop Architecture: Microtasks vs Macrotasks

```
+-------------------------------------------------------------+
| 1. Execute Synchronous Script on Call Stack                 |
+-------------------------------------------------------------+
                              |
                              v
+-------------------------------------------------------------+
| 2. Drain Entire Microtask Queue (Promises, queueMicrotask)  |
|    * CAUTION: Microtasks added during this drain are also   |
|      executed immediately! Can lead to event loop starvation|
+-------------------------------------------------------------+
                              |
                              v
+-------------------------------------------------------------+
| 3. Browser UI Render / Animation Frame (if applicable)      |
+-------------------------------------------------------------+
                              |
                              v
+-------------------------------------------------------------+
| 4. Execute EXACTLY ONE Macrotask (setTimeout, I/O, IPC)     |
+-------------------------------------------------------------+
                              |
                              +----> Loop back to Step 2
```

### Microtask Starvation
If a microtask continuously schedules another microtask:
```javascript
function starve() {
  queueMicrotask(starve); // STALLS EVENT LOOP COMPLETELY!
}
```
Macrotasks (I/O, network responses, timers, rendering) will never execute. Always yield to macrotasks (`setTimeout(resolve, 0)`) for long computations.

---

## 3. Memory Leak Patterns in V8 / JavaScript Engines

### 3.1 Uncleared Event Listeners
```javascript
// LEAK: `largeContext` remains retained in memory as long as `window` exists
const largeContext = new Array(1000000).fill("heavy data");
window.addEventListener("resize", () => {
  console.log(largeContext.length);
});
```
**Fix**: Use `AbortController`:
```javascript
const controller = new AbortController();
window.addEventListener("resize", () => { ... }, { signal: controller.signal });
// When unmounting:
controller.abort(); // Event listener is cleaned up immediately
```

### 3.2 Accidental Closure Retention
When an inner function closes over a scope, the entire lexical environment of that scope is retained in memory by the engine if any inner closure survives.

### 3.3 Weak Collections for Ephemeral Metadata
Use `WeakMap` to associate auxiliary metadata with objects without preventing garbage collection:
```javascript
const sessionCache = new WeakMap();

export function setMetadata(userObj, meta) {
  sessionCache.set(userObj, meta);
}
// When userObj is de-referenced anywhere in the app, its sessionCache entry is automatically GC'd
```
