# Game Runtime Verification Checklist

## Tick Contract
- [ ] Fixed simulation step, accumulator precision, frame clamp, and maximum catch-up steps are explicit.
- [ ] Gameplay and physics never use undeclared variable delta time.
- [ ] Excess time is handled with a documented policy and observable metric.

## Rendering and Scenes
- [ ] Previous/current snapshots interpolate presentation without mutating simulation state.
- [ ] Scene lifecycle hooks and transitions are explicit, atomic, and reversible on failure.
- [ ] Old scene resources release only after the new scene commits.

## Host Events
- [ ] Focus loss, suspend/resume, resize, display change, and device loss pause at tick boundaries.
- [ ] Resume restores the exact tick, accumulator, and required simulation state.
- [ ] GPU resource recreation follows a documented order and failure policy.

## Determinism and Performance
- [ ] Repeated input scripts produce identical state hashes at fixed tick boundaries.
- [ ] Per-scene and per-system update budgets produce attributable overruns.
- [ ] Injected long frames cannot create a tick spiral or unbounded memory growth.

## Evidence
- [ ] Scene-transition cycles show no resource leak across at least 50 runs.
- [ ] Target-device frame-time and recovery evidence is recorded.
- [ ] `scripts/verify.sh` exits with code 0 from the repository root.
