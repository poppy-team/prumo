# Odin Systems Safety & Verification Report: {{Target.Component}}

## 1. Metadata
- **Component ID**: `{{component_id}}`
- **Compiler Version**: `odin` (Tested with `{{odin_version}}`)
- **Build Mode**: [Debug | Release | Speed | Size]
- **Date**: {{YYYY-MM-DD}}
- **Author / Agent**: {{agent_id}}

---

## 2. Memory & Allocator Safety Verification
- **Context Configuration**:
  - [ ] `context.allocator` properly scoped and documented.
  - [ ] `context.temp_allocator` reset intervals verified (`free_all`).
- **Tracking Allocator Results**:
  - Total Allocations: {{total_allocations}}
  - Leaked Allocations (`len(track.allocation_map)`): 0
  - Bad Frees (`len(track.bad_free_array)`): 0
  - Memory Leak Audit: [PASS / 0 bytes leaked]

---

## 3. Data-Oriented Architecture & Type Safety
- **Distinct Domain Types**: Verified (Entity IDs, counters, currencies declared as `distinct`).
- **Data Layout**: `#soa` employed for large collections where cache line efficiency is required.
- **Escape Hatches (`cast(rawptr)`)**: {{escape_hatches_count}} registered in `.prumo/escape-hatches.json`.

---

## 4. Test Execution & Output Log
- **Total Tests**: {{total_tests}}
- **Passed**: {{passed_tests}}
- **Failed**: 0
- **Command Executed**: `odin test {{test_directory}} -all-packages`

```text
{{odin_test_log}}
```

---

## 5. Architecture Sign-Off
- [ ] No unmanaged global memory allocations.
- [ ] All `make` and `new` paired with `defer delete` or `defer free`.
- [ ] Zero unhandled errors on fallible paths.
