# Scene Graph & ECS — Verification Checklist

## 1. Entity Identity & Storage
- [ ] Entity references are generational handles (index + generation); raw indices prohibited.
- [ ] Hot components packed in archetype SoA tables (≤ ~64 bytes/row target); sparse sets for rare tags.
- [ ] Stale handles from destroy+recycle cycles fail validation in tests.

## 2. Transform Hierarchy Correctness
- [ ] World matrices written only by the hierarchy sync pass (`world = parent_world * local`, breadth-first).
- [ ] Dirty-flag propagation recomputes exactly the dirty subtrees once per tick.
- [ ] Parent cycles rejected at attach time with an explicit error code.

## 3. System Scheduling & Determinism
- [ ] Fixed system order per phase (input → simulation → transform sync → render extract) recorded as configuration.
- [ ] Structural changes deferred through command buffers applied at phase boundaries; none mid-iteration.
- [ ] Execution order identical across 100 ticks on the determinism fixture.

## 4. Parallelism & Memory Safety
- [ ] Parallel jobs partition disjoint component sets or level wavefronts; no data races under thread sanitizer.
- [ ] Render path consumes a frame-local snapshot, never live simulation writes.
- [ ] Scene update meets the frame-time budget on all benchmark fixtures including churn scenes.

## 5. Evidence & Sign-Off
- [ ] Churn benchmark log (spawn/destroy rates, per-system timings, hierarchy depth) recorded.
- [ ] Cycle-rejection and stale-handle test results attached.
- [ ] `scripts/verify.sh` exits 0 from the repository root.
