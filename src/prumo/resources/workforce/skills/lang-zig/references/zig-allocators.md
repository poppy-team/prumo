# Zig Memory Architecture & Allocators: Technical Reference Guide

## 1. The Explicit Allocation Paradigm

Zig has no hidden allocations in the language or standard library. If a function does not accept a `std.mem.Allocator`, it does not allocate memory on the heap.

### 1.1 The Allocator Interface (`std.mem.Allocator`)

`std.mem.Allocator` is a fat pointer struct containing a pointer to allocator state and a virtual table:

```zig
pub const Allocator = struct {
    ptr: *anyopaque,
    vtable: *const VTable,
    ...
};
```

Common methods:
- `alloc(comptime T: type, n: usize) ![]T`: Allocate slice of `n` items.
- `free(slice: anytype) void`: Free previously allocated slice.
- `create(comptime T: type) !*T`: Allocate a single item.
- `destroy(ptr: anytype) void`: Free a single item.
- `realloc(slice: anytype, new_len: usize) ![]T`: Resize slice in place or copy.

---

## 2. Allocator Selection Taxonomy

```
+-------------------------------------------------------------------------+
|                         Allocation Requirements                         |
+------------------------------------+------------------------------------+
                                     |
              +----------------------+----------------------+
              |                                             |
     [Dynamic / Unbounded]                           [Fixed / Bounded]
              |                                             |
      +-------+-------+                                     v
      |               |                       FixedBufferAllocator
[Long-Lived]    [Short-Lived / Scratch]       - Pre-allocated stack buffer
      |               |                       - Zero syscalls
      v               v                       - Constant O(1) alloc
GeneralPurpose   ArenaAllocator
  Allocator     - Wraps backing allocator
(GPA / Test)    - Free all memory in one deinit()
                - Eliminates micro-fragmentation
```

### 2.1 GeneralPurposeAllocator (GPA)
`std.heap.GeneralPurposeAllocator(.{})` provides safety-first dynamic allocation:
- Thread-safe by default.
- Detects double-free and use-after-free via memory poisoning.
- `gpa.deinit()` returns `Check.leak`, signaling leaked memory allocations.

### 2.2 ArenaAllocator
`std.heap.ArenaAllocator` wraps a child allocator and aggregates allocations:
```zig
var arena = std.heap.ArenaAllocator.init(std.heap.page_allocator);
defer arena.deinit();
const allocator = arena.allocator();
// Allocate multiple data structures freely; arena.deinit() frees everything
```

### 2.3 FixedBufferAllocator
Constructed on top of a fixed slice, eliminating heap calls completely:
```zig
var stack_buffer: [4096]u8 = undefined;
var fba = std.heap.FixedBufferAllocator.init(&stack_buffer);
const allocator = fba.allocator();
```

---

## 3. Error Handling and Resource Rollbacks

Zig error handling uses error sets and error unions:

```zig
pub fn createPipeline(allocator: std.mem.Allocator) !Pipeline {
    var stage1 = try Stage.init(allocator);
    errdefer stage1.deinit();

    var stage2 = try Stage.init(allocator);
    errdefer stage2.deinit();

    return Pipeline{ .s1 = stage1, .s2 = stage2 };
}
```

If `stage2.init()` fails with `error.OutOfMemory`, `errdefer stage1.deinit()` automatically executes, ensuring zero memory leakage on failure paths.
