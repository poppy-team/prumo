# Physics Testing — Verification Checklist

## 1. Timestep & Simulation Contract
- [ ] Fixed timestep enforced (60 Hz, `dt = 1/60 s`); render uses interpolation, never variable steps.
- [ ] Substep catch-up clamped (max 4 substeps); spiral-of-death clamps are logged, never silent.
- [ ] Gravity, linear damping, and solver iterations (velocity 8, position 3) pinned in config, not scattered in code.
- [ ] Scene reload reproduces identical initial state; no static-initialization order dependence.

## 2. Collision Layers, Triggers & Platforms
- [ ] All 64 layer-pair cases executed: callbacks fire exactly for intended pairs, zero for masked pairs.
- [ ] Trigger enter/exit ordering correct at 300 px/s with no missed exits or double enters.
- [ ] One-way platforms pass from below, land from above, and honor drop-through input within one frame.
- [ ] Sleeping bodies wake on contact, applied force, and teleport; no frozen corpses block gameplay.

## 3. Joints, Stacking & Continuous Collision
- [ ] 6-link rope chain under 40 kg holds constraint error below 2 cm over a 10 s soak.
- [ ] Joint break forces trigger within 10% of declared thresholds, both directions where applicable.
- [ ] 1,000 shots at 900 px/s against 8 px walls with CCD enabled show zero tunneling.
- [ ] 12-crate stack settles within 3 s with residual jitter below 1 mm/frame; no explosive separations.

## 4. Determinism & Debug Evidence
- [ ] 30 s input replay at RNG seed 90210 produces identical SHA-256 trajectory hashes across two headless runs.
- [ ] Post-reload replay hash matches; cross-platform equality scope documented (same-OS bitwise, cross-OS tolerance noted).
- [ ] Debug-draw captures (contacts, AABBs, joint anchors) archived for every investigated failure.
- [ ] Automated verification script `scripts/verify.sh` executes with exit code 0.
