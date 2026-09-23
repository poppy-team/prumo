# C3 Safety & Contract Assertions Checklist

## 1. Resource Management & Defer Hygiene
- [ ] **Deterministic Defer**: All dynamic memory allocations (`mem::alloc`, `mem::new`) and OS handles pair immediately with `defer mem::free(ptr)` or handle closing.
- [ ] **Rollback Safety**: Fallible multi-step procedures use `defer (catch err)` for cleanup upon unexpected faults.
- [ ] **Scope LIFO Order**: Resource cleanup order matches inverse acquisition dependencies.

## 2. Slice & Memory Safety
- [ ] **Bounded Slices**: Procedure signatures handling buffers accept slices (`type[]`) rather than raw pointers (`type*`).
- [ ] **Length Verification**: Slice operations utilize built-in `.len` property; no external manual length bookkeeping.
- [ ] **Safe Sub-slicing**: Sub-range slices use validated indices (`slice[start..end]`).

## 3. Faults & Error Handling
- [ ] **Fault Types**: Fallible functions declare explicit fault return types (`!Type` or `Type!`).
- [ ] **Zero Swallowed Faults**: Every `catch` block contains recovery, logging, or rethrow logic; no empty catch blocks.
- [ ] **Nodiscard Checking**: Public procedures returning results or faults are annotated with `@nodiscard`.

## 4. Contracts & Type Safety
- [ ] **Distinct Domain Types**: IDs, currencies, and timestamps are declared as `distinct` types to prevent type confusion.
- [ ] **Pure Functions**: Referential transparent mathematical/transformation procedures are annotated with `@pure`.
- [ ] **Escape Hatch Audit**: All raw pointer conversions and unsafe blocks are recorded in `.prumo/escape-hatches.json`.
