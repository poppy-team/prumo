# Odin Systems Safety & Memory Verification Checklist

## 1. Context & Allocator Management
- [ ] **Tracking Allocator in Tests**: Unit tests employ `mem.Tracking_Allocator` and assert `len(track.allocation_map) == 0` upon exit.
- [ ] **Deterministic Cleanup**: All `make(...)`, `new(...)`, and slice clones pair immediately with `defer delete(...)` or `defer free(...)`.
- [ ] **Temp Allocator Hygiene**: Temporary scratch allocations utilize `context.temp_allocator` and are cleared periodically with `free_all(context.temp_allocator)`.
- [ ] **Zero Global Heap State**: Procedures do not allocate into global variables without synchronized, explicit thread-safe lifecycles.

## 2. Type System & Domain Modeling
- [ ] **Strong Primitives**: Domain entity identifiers, quantities, and units are declared as `distinct` types to prevent type confusion.
- [ ] **Tagged Unions**: Variant data structures use typed `union` and are processed using exhaustive `switch in` checks.
- [ ] **No Implicit Overloading**: Code maintains explicit control flow with zero operator overloading.

## 3. Data-Oriented Architecture (#soa)
- [ ] **Cache Coherence**: High-frequency entity collections exceeding 64 elements use `#soa` where vectorization and cache line efficiency are required.
- [ ] **Alignment & Padding**: Structs intended for GPU buffers or SIMD match required alignment boundaries (`#align`).

## 4. Error Propagation & Safety Hatches
- [ ] **Error Propagation**: Fallible calls use `or_return` or explicit multi-value error unwrapping (`val, ok := ...`).
- [ ] **Escape Hatch Audit**: All `cast(rawptr)` raw pointer conversions and FFI pointer casts are recorded in `.prumo/escape-hatches.json`.
