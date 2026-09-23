# Implementation Reality Verification — Reference Guide

## 1. Core Concepts

### 1.1 The Terminal State Hierarchy
Completion is a ladder, never a feeling: `implemented → reachable → exercised → evidenced → verified → accepted → released`. Each rung demands distinct proof. Code can be implemented (exists on disk) yet unreachable (no call path); reachable yet unexercised (no test hits it); exercised yet unevidenced (no recorded oracle). Agents advance one rung per proof and never declare DONE from prose output.

### 1.2 Presence ≠ Reachability ≠ Exercise
Finding a file proves presence. Reachability requires a traced call path from a real entrypoint (CLI command, handler, scheduled job) to the code, confirmed by static call-graph or runtime trace. Exercise requires execution under test or production traffic with inputs that traverse the claimed branches — import-time side effects and no-op invocations do not count.

### 1.3 False-Green Oracles
Tests pass for bad reasons: assertions that cannot fail (`assert True`), mocks replacing the very capability under test (mocked filesystem "proving" atomic writes), swallowed errors (`except: pass` around the critical call), and fixtures that never trigger the defect path. Verification actively hunts these: mutate the code (mutation testing) and confirm the test suite bleeds red; delete the mock and confirm the real integration still holds.

### 1.4 Call-Site Inventory and Dead-Code Detection
Every claimed feature ships with a call-site inventory: entrypoints, wiring (DI, routes, flags), lifecycle hooks, and error paths. Code with zero inbound references after inventory is dead code — it is deleted or explicitly quarantined, never counted as "implemented."

## 2. Patterns and Anti-Patterns

| Pattern (do) | Anti-pattern (avoid) |
|---|---|
| Trace entrypoint → handler → feature with `grep`/call-graph evidence | "The file exists, so the feature works" |
| Kill the mock; test the real fs/db/socket once | 100% mocked suite presented as integration proof |
| Mutation-test critical paths; require kills | Coverage % quoted as correctness proof |
| Inventory every call site, including error paths | Happy-path-only verification |
| Record the oracle (exact command + expected output) | "Tests pass" with no named suite or assertion |

## 3. Worked Example

Claim: "Atomic file replacement is implemented and tested."

- Presence: `fs/atomic_write.py` exists. Insufficient.
- Reachability: `grep` shows `api/upload.py:112` calls `atomic_write()` on the upload path — reachable. The CLI flag `--no-atomic` bypasses it — noted as an unwired path.
- Exercise: the test `test_atomic_replace` uses a fake in-memory FS — the real `os.replace` path is never executed. False-green suspected.
- Verification fix: add `test_atomic_replace_real_tmp` using `tmp_path` on disk, inject `ENOSPC` via full-volume fixture to exercise the error path, and mutation-test (remove the `fsync` call → test must fail). Only then does the rung advance to verified.
