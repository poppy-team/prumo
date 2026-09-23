# Physics Test Specification — Starfall Drift Skiff Handling (Rapier 0.22)

## 1. Simulation Contract
- **Engine**: Rapier 0.22 (Rust), gravity (0, -980) px/s², velocity iterations 8, position iterations 3.
- **Timestep**: fixed 60 Hz, accumulator with 4-substep clamp and 100 ms frame clamp; render interpolation alpha documented in `render/interp.rs`.
- **RNG**: single seeded stream, seed 90210 for replays; no wall-clock or pointer-hash inputs to the solver.
- **Test date and owner**: 2026-09-09, gameplay pod (Tomas Lindqvist), build 0.9.2-rc2.

## 2. Collision Matrix (8 Layers, 64 Ordered Pairs)

| Pair | Expected | Observed over 64-case sweep | Verdict |
|---|---|---|---|
| Player x World | contact + slide | 8 of 8 contacts resolved | Pass |
| Player x OneWay | land from above only | 4 of 4 directional cases correct | Pass |
| Projectile x World | contact + despawn | 12 of 12 despawned | Pass |
| Projectile x Trigger | trigger only, no impulse | 6 of 6 overlap-only | Pass |
| Pickup x Player | trigger collect | 5 of 5 collected | Pass |
| Debris x Pickup | silent (masked) | 0 callbacks across 50 contacts | Pass |
| Enemy x OneWay | silent (walkers ignore) | 0 callbacks | Pass |
| All other 45 pairs | per matrix sheet rev 7 | match sheet | Pass |

## 3. Joints, CCD, and Stacking

| Case | Setup | Result | Verdict |
|---|---|---|---|
| Rope chain | 6 links, 40 kg load, 10 s | max constraint error 1.4 cm | Pass |
| Tow cable break | rated 120 N | broke at 114-122 N across 20 pulls | Pass |
| CCD soak | 1,000 shots, 900 px/s vs 8 px wall | zero tunneling | Pass |
| Crate stack | 12 crates, 3 s settle | residual jitter 0.6 mm/frame | Pass |
| Trigger at speed | capsule 300 px/s through gate | enter/exit ordered, no misses | Pass |

## 4. Determinism Replay

| Run | Trajectory SHA-256 (prefix) | Verdict |
|---|---|---|
| Headless run 1, seed 90210 | 9f2c…a41d | baseline |
| Headless run 2, seed 90210 | 9f2c…a41d | Pass, identical |
| Post-reload run, seed 90210 | 9f2c…a41d | Pass, identical |

## 5. Regression Evidence
- [x] 64-pair matrix suite green on rc2; contact-ordering sort by body id in place.
- [x] CCD and joint soaks logged with zero tunneling and sub-2 cm errors.
- [x] Replay hashes identical across runs and reload; CI gate `physics-determinism` green on commit 55bc10.
- [x] `scripts/verify.sh` exits 0 on the skill package.
