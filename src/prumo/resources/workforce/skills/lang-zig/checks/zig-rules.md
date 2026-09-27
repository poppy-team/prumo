# Zig Systems Programming & Memory Verification Checklist

## 1. Explicit Memory & Allocator Hygiene
- [ ] **Explicit Allocators**: All functions performing heap operations accept an explicit `allocator: std.mem.Allocator`.
- [ ] **Deterministic Teardown**: Allocations pair with `defer allocator.free(slice)` or `defer instance.deinit()`.
- [ ] **Rollback Safety**: Multi-step allocations pair with `errdefer allocator.free(...)` to prevent partial leaks on error exit.
- [ ] **Leak Detection in Tests**: All unit tests run under `std.testing.allocator` with 0 reported leaks.
- [ ] **Appropriate Allocator Topology**: Short-lived request scopes use `std.heap.ArenaAllocator`; stack buffers use `std.heap.FixedBufferAllocator`.

## 2. Error Union & Control Flow Discipline
- [ ] **Zero Ignored Errors**: No bare `_ = expr catch {};` or silent discards without documented justification.
- [ ] **Error Propagation**: Use `try` for unrecoverable errors; handle specific errors with `catch |err| switch (err)`.
- [ ] **Zero Hidden Control Flow**: No hidden operator overloading or implicit constructor invocations.

## 3. Comptime & Type Invariants
- [ ] **Compile-Time Invariants**: Generic types and buffer bounds are asserted via `@compileError` at build time.
- [ ] **Safe Type Reflection**: Type introspection utilizes `@typeInfo` safely without assumptions about internal layout unless marked `extern struct`.

## 4. Pointer & Slice Safety
- [ ] **Fat Slices Over Multi-Pointers**: APIs accept `[]const T` or `[]T` instead of raw `[*]T`.
- [ ] **Sentinel Checks**: C string conversions use `[:0]const u8` with validated zero-terminator.
- [ ] **Escape Hatch Registry**: Any `@ptrCast`, `@alignCast`, or `@intCast` is documented in `.prumo/escape-hatches.json`.
- [ ] **ReleaseSafe Default**: Binaries compile under `ReleaseSafe` unless formally audited and benchmarked for `ReleaseFast`.
