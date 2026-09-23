# Odin Context System & Data-Oriented Memory: Technical Reference Guide

## 1. The Implicit Context Architecture

Every procedure in Odin receives an implicit thread-local `context: runtime.Context` parameter unless marked with `#no_context`.

### 1.1 Context Fields
```odin
Context :: struct {
    allocator:              Allocator,          // Standard dynamic allocator
    temp_allocator:         Allocator,          // Ring buffer / arena for scratch allocations
    assertion_failure_proc: Assertion_Failure_Proc,
    logger:                 Logger,
    user_index:             int,
    user_ptr:               rawptr,
}
```

### 1.2 Overriding Context Locally
A procedure or block can override the context for itself and all downstream callee procedures:

```odin
my_scoped_proc :: proc() {
    arena: mem.Arena
    mem.arena_init(&arena, make([]u8, 64 * mem.Megabyte, context.allocator))
    defer delete(arena.data)

    // Override context.allocator for the remainder of this scope
    context.allocator = mem.arena_allocator(&arena)

    // All downstream procedures now allocate from the arena
    process_batch()
}
```

---

## 2. Leak-Proof Testing with `mem.Tracking_Allocator`

Odin's standard library provides a built-in tracking allocator that logs memory allocations, reallocations, and frees:

```odin
package test_suite

import "core:fmt"
import "core:mem"
import "core:testing"

@(test)
test_component_lifecycle :: proc(t: ^testing.T) {
    track: mem.Tracking_Allocator
    mem.tracking_allocator_init(&track, context.allocator)
    defer mem.tracking_allocator_destroy(&track)
    context.allocator = mem.tracking_allocator(&track)

    // Run tests...
    run_domain_logic()

    // Assert zero memory leaks
    if len(track.allocation_map) > 0 {
        for _, leak in track.allocation_map {
            fmt.printf("%v leaked %m bytes\n", leak.location, leak.size)
        }
        testing.fail(t)
    }

    // Assert zero bad/double frees
    if len(track.bad_free_array) > 0 {
        for bad_free in track.bad_free_array {
            fmt.printf("%v bad free at %p\n", bad_free.location, bad_free.memory)
        }
        testing.fail(t)
    }
}
```

---

## 3. Data-Oriented Design: Structure of Arrays (`#soa`)

In traditional Array of Structures (AoS):
```
[Pos | Vel | Life | Color][Pos | Vel | Life | Color] ...
```
Iterating over only `Life` loads entire 48-byte structs into cache lines, wasting 80%+ of memory bandwidth.

Using Odin's `#soa`:
```odin
Particle :: struct {
    pos:   [3]f32, // 12 bytes
    vel:   [3]f32, // 12 bytes
    life:  f32,    // 4 bytes
    color: [4]u8,  // 4 bytes
}

particles := make(#soa[dynamic]Particle, 1000, context.allocator)
defer delete(particles)

// Iterate directly on slice of life values - 100% cache line utilization!
for &l in particles.life {
    l -= dt
}
```
The `#soa` attribute instructs the Odin compiler to decompose the dynamic array into parallel contiguous buffers behind an ergonomic, array-like indexing syntax.
