# Zig Quality & Safety Verification Report: {{Target.Component}}

## 1. Metadata
- **Target Module**: `{{target_module}}`
- **Compiler Version**: `zig 0.13+` (Tested on `{{zig_version}}`)
- **Build Mode**: [Debug | ReleaseSafe | ReleaseFast | ReleaseSmall]
- **Date**: {{YYYY-MM-DD}}
- **Author / Agent**: {{agent_id}}

---

## 2. Memory & Allocator Safety Audit
- **Allocators Used**:
  - [ ] `std.mem.Allocator` passed explicitly to all allocating functions.
  - [ ] `std.heap.ArenaAllocator` utilized for bounded lifecycles / scratch arenas.
  - [ ] `std.heap.FixedBufferAllocator` used for stack allocations.
- **Leak Detection**:
  - `std.testing.allocator` result: [PASS / 0 bytes leaked]
  - GPA leak check result: [CLEAN / No leaks detected]
- **Resource Teardown**:
  - Total `defer` cleanups: {{defer_count}}
  - Total `errdefer` rollback cleanups: {{errdefer_count}}

---

## 3. Comptime & Static Analysis
- **Comptime Invariants**: {{comptime_checks_description}}
- **Escape Hatch Count (`@ptrCast`, `@alignCast`, `@intCast`)**: {{escape_hatches_count}}
- **Registered Escape Hatches in `.prumo/escape-hatches.json`**: [YES / NONE NEEDED]

---

## 4. Test & Verification Evidence
- **Total Tests Executed**: {{total_tests}}
- **Passed**: {{passed_tests}}
- **Failed**: 0
- **Command Executed**: `zig test {{test_file}}`

```text
{{test_output_log}}
```

---

## 5. Architectural Compliance Sign-Off
- [ ] No global hidden allocators.
- [ ] Zero unhandled or discarded error unions.
- [ ] Code formatted with `zig fmt`.
