# Scene Graph & ECS — Technical Reference Guide

## 1. Core Concepts

**Archetype storage** packs entities sharing a component set into dense SoA tables, giving linear iteration over hot components (transform, velocity). **Sparse sets** back rarely-present tags with O(1) add/remove/has. **Generational handles** (index + generation) make stale references fail validation instead of aliasing recycled rows.

**Transform hierarchy** composes `world = parent_world * local` top-down once per tick, driven by dirty flags: a local change marks the subtree dirty; the sync pass recomputes breadth-first. Cycles are rejected at attach time.

**System scheduling** fixes execution order per phase (input → simulation → transform sync → render extract). Structural changes go through **command buffers** applied at phase boundaries so iteration never invalidates.

## 2. Patterns

- **SoA hot columns**: store `pos_x[], pos_y[], pos_z[]` contiguously; a simulation pass over 10k entities touches ~120 KB sequentially instead of chasing pointers.
- **Deferred attach/detach**: `cmd.attach(child, parent)` validated (cycle check, generation check) and applied at the next barrier; queries inside systems see a stable graph.
- **Level wavefront parallelization**: compute subtree depths once, then process all nodes at depth d in parallel jobs before depth d+1.
- **Render extract copy**: copy world matrices into a frame-local snapshot so rendering never races simulation writes.

## 3. Anti-Patterns

- Raw indices as entity IDs (use-after-recycle corruption).
- Adding/removing components inside a system loop (archetype moves invalidate iterators).
- Writing world matrices from gameplay code (two writers, order-dependent results).
- Depth-first recursive sync on deep hierarchies (stack overflow past ~10k depth); use iterative breadth-first.

## 4. Worked Example

A 12,000-entity stress scene (3-level hierarchy, churn of 200 spawns + 200 destroys per frame): archetype iteration over `LocalTransform + Velocity` runs in 0.31 ms; deferred command buffer applies 400 structural ops at the phase barrier in 0.09 ms; hierarchy sync over 4,000 parented entities completes in 0.22 ms. A stale handle from a destroyed entity fails generation validation, and attaching a node to its own descendant is rejected with `ERR_CYCLE`.

## 5. Verification Pointers

- Spawn/destroy churn while systems iterate; assert no crashes and stable frame time.
- Hold handles across a destroy+recycle cycle; assert stale validation failure.
- Record system execution order across 100 ticks; assert identical sequence every tick.
