# C3 Safety & Contract Verification Report: {{Target.Component}}

## 1. Metadata
- **Target Component**: `{{component_id}}`
- **Compiler Version**: `c3c` (Tested on `{{c3c_version}}`)
- **Compilation Target**: `{{target_triple}}`
- **Date**: {{YYYY-MM-DD}}
- **Author / Agent**: {{agent_id}}

---

## 2. Resource Management & Defer Audit
- **Dynamic Allocations**:
  - Total Allocations: {{allocation_count}}
  - Verified Paired `defer` cleanups: {{defer_count}}
  - Conditional `defer (catch)` rollbacks: {{defer_catch_count}}
- **Resource Leak Audit**: [CLEAN / 0 leaks detected]

---

## 3. Slice Bounds & Memory Safety
- **Public API Signatures**:
  - Slices used for sequence inputs (`T[]`): [100% / PASS]
  - Raw pointer iterations eliminated: [YES]
- **Fault Handling & Nodiscard**:
  - Total Fault Handlers: {{fault_handlers_count}}
  - Ignored Fault Checks: 0
  - `@nodiscard` functions checked: [YES]

---

## 4. Contract & Semantic Attributes
- **`@pure` Annotated Procedures**: {{pure_proc_count}}
- **`distinct` Domain Types**: {{distinct_types_count}}
- **Escape Hatches (`(void*)`, `@unsafe`)**: {{escape_hatches_count}} logged in `.prumo/escape-hatches.json`.

---

## 5. Test Execution Evidence
- **Total Tests**: {{total_tests}}
- **Passed**: {{passed_tests}}
- **Failed**: 0
- **Command Executed**: `c3c test {{test_target}}`

```text
{{c3c_test_log}}
```

---

## 6. Architecture Compliance Sign-Off
- [ ] No naked pointer arithmetic in exported procedures.
- [ ] All heap allocations bounded by defer or arena lifecycles.
- [ ] Fault branches handled explicitly without silent swallowing.
