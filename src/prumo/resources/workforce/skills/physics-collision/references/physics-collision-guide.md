# Physics & Collision Systems — Technical Reference Guide

## 1. Core Concepts

**Broadphase** culls the O(n²) pair space to a small candidate set. Choose by scene profile:
- **Uniform spatial hash**: cell size ≈ 2× mean body diameter; best for evenly distributed bodies; O(n) insert + query.
- **Sweep-and-prune (SAP)**: sort AABB endpoints per axis each tick; best for temporally coherent motion; nearly O(n + k) with insertion sort on warm arrays.
- **Dynamic AABB tree (BVH)**: best for highly non-uniform scenes; refit bottom-up, rotate for balance when tree height exceeds ~2× log₂(n).

**Narrowphase** turns candidate pairs into contact manifolds: AABB pre-test → shape-pair dispatch (circle/polygon, sphere/box, convex-convex via GJK for closest points + EPA for penetration depth and normal).

**Solver** applies sequential impulses: accumulate normal + friction impulses per contact, iterate 4–8 times, warm-start from the previous tick. Split position correction out of the velocity loop (split impulse) to remove resting jitter without adding energy.

## 2. Patterns

- **Fixed timestep with accumulator**: `acc += clamp(frame_dt, 0, 0.1); while (acc >= DT && steps < 4) { step(DT); acc -= DT; }` with interpolation alpha `acc / DT` for rendering.
- **Contact slop + restitution gating**: allow 0.5 mm penetration slop; apply restitution only when impact velocity exceeds ~1 m/s so stacks don't bounce.
- **Sleep islands**: bodies with linear velocity < 0.05 m/s and angular velocity < 0.1 rad/s for 30 ticks sleep; any new contact or explicit impulse wakes the island.
- **Swept CCD for fast movers**: if displacement per tick > 0.5 × smallest extent, sweep the shape (or cast a ray fallback) instead of discrete testing.

## 3. Anti-Patterns

- Integrating with variable frame delta (energy growth, tunneling, nondeterminism).
- Heap-allocating pair/contact buffers per tick (GC pauses, frame spikes); pre-size to 4× expected pair count and assert on overflow.
- Mutating the body list inside contact callbacks (iterator invalidation); queue creations/destructions for phase boundaries.
- Scaling the whole world to tiny units (float precision loss); keep 1 unit = 1 m and tune slop/thresholds in meters.

## 4. Worked Example

A 500-box stack scene at DT = 1/60 s: SAP broadphase yields ~2,400 candidate pairs from 124,750 theoretical pairs; narrowphase confirms ~1,900 contacts; 6 solver iterations converge penetration under slop within 1.1 ms on reference hardware. Two seeded runs produce byte-identical body transforms, proving determinism. A 40 m/s projectile (0.2 m radius) tunnels discretely at 60 Hz (0.67 m per tick) but is caught by swept CCD.

## 5. Verification Pointers

- Assert broadphase pair recall = 100% against brute-force O(n²) on a 200-body fixture.
- Assert zero allocations in the tick loop (language allocator stats or instrumented counters).
- Assert tunneling regression scene passes with CCD on and fails with CCD off.
