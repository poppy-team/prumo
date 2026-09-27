# Physics & Collision Systems — Verification Checklist

## 1. Timestep & Determinism
- [ ] Fixed timestep with accumulator and substep clamp; no variable-dt integration.
- [ ] Two seeded runs produce bit-identical body transforms and contact counts.
- [ ] Render interpolation uses accumulator alpha, never simulation state mutation.

## 2. Broadphase Correctness & Budget
- [ ] Pair recall is 100% against brute-force O(n²) on the 200-body fixture.
- [ ] Broadphase consumes < 15% of the physics frame budget on all fixtures.
- [ ] Structure matches scene profile (spatial hash / sweep-and-prune / AABB tree justified).

## 3. Narrowphase & Solver Stability
- [ ] Contact slop (~0.5 mm) and restitution gating (~1 m/s threshold) configured.
- [ ] Sequential-impulse solver iterates 4–8 times with warm starting; stacks rest without jitter.
- [ ] Fast movers (displacement > 0.5 × smallest extent per tick) use swept CCD; tunneling regression scene passes.

## 4. Memory & Performance Invariants
- [ ] Zero heap allocations inside per-tick broadphase, narrowphase, and solver loops.
- [ ] Pair/contact buffers pre-sized with overflow assertion; total step < budget (e.g. 2 ms at 60 FPS).
- [ ] Sleeping islands active: quiescent bodies skip integration until woken.

## 5. Structural Safety & Evidence
- [ ] No body creation/destruction inside contact callbacks; deferred to phase boundaries.
- [ ] Benchmark log (per-phase timings, counts) and determinism diff recorded.
- [ ] `scripts/verify.sh` exits 0 from the repository root.
