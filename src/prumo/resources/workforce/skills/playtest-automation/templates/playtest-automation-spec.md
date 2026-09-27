# Playtest Automation Specification — Harbor Siege Milestone (Build 0.9.2-rc3)

## 1. Scope and Harness
- **Levels**: Harbor Approach, Dock Firefight, Lighthouse Ascent (quest chains Q201-Q214).
- **Harness flags**: `--playtest-headless --timescale 4`, state query `/state` at 10 Hz, input log format `inputs v6`.
- **Golden recording**: `inputs_harbor_v6.bin`, 12-minute designer run by Priya Nair, seed 4417, accepted 2026-09-12.
- **Test date and owner**: 2026-09-13, gameplay QA pod (Dario Fontana).

## 2. Bot Cohort and Routes

| Archetype | Route policy | Runs | Median completion | Deaths (median) | Coins (median) |
|---|---|---|---|---|---|
| Speeder | optimal, skips optionals | 10 | 8 min 41 s | 1 | 96 |
| Explorer | all waypoints, all doors | 10 | 14 min 02 s | 2 | 214 |
| Completionist | all 24 collectibles | 10 | 17 min 35 s | 3 | 238 |
| Combat-heavy | engages all 31 spawns | 10 | 13 min 20 s | 5 | 187 |
| Idle | zero input, 15 min | 3 | n/a, no progress, no soft-lock | 0 | 0 |

## 3. Telemetry Diff vs Build 0.9.2-rc2 (5% Tolerance)

| Metric | rc2 baseline | rc3 median | Delta | Verdict |
|---|---|---|---|---|
| Explorer completion | 13 min 48 s | 14 min 02 s | +1.7% | Pass |
| Combat deaths/run | 4.7 | 5.0 | +6.4% | Waived by design (WAIVER-311, spawn rebalance intentional) |
| Completionist coins | 236 | 238 | +0.8% | Pass |
| Speeder completion | 8 min 52 s | 8 min 41 s | -2.1% | Pass |

## 4. Soak Results (4 Hours, Timescale 4x)

| Item | Value |
|---|---|
| Heartbeats recorded | 2,880 of 2,880, zero gaps |
| Stuck incidents | 1 (explorer wedged at lighthouse stairwell tile L-19, bundle INC-2041, bug filed PHX-882) |
| Hangs / crashes | 0 |
| RSS growth | +22 MB over 4 h, inside 50 MB ceiling |

## 5. Regression Evidence
- [x] Golden replay matches quest timeline within 2 s on seed 4417.
- [x] Cohort telemetry CSV (53 runs) with diff report archived under `playtest/2026-09-13/`.
- [x] Incident bundle INC-2041 holds screenshot, state dump, and trailing 60 s of inputs.
- [x] `scripts/verify.sh` exits 0 on the skill package; playtest gate green on commit 77c1be.
