# Playtest Automation — Verification Checklist

## 1. Harness & Determinism Controls
- [ ] Build exposes synthetic input injection, time-scale control, and a state-query API (positions, quest flags, RNG state).
- [ ] Fixed timestep and seeded RNG streams active during automation; timescale capped at 4x with determinism re-proof above 1x.
- [ ] Input log format versioned (e.g., `inputs v6`); old logs rejected explicitly rather than misread.
- [ ] Injection latency below one tick; no dropped or duplicated synthetic inputs in the acceptance run.

## 2. Golden Replays & Bot Coverage
- [ ] Golden recording replays with identical quest-flag timeline and completion time within 2 s on the recorded seed.
- [ ] Bot cohort covers at least speeder, explorer, completionist, combat-heavy, and idle archetypes.
- [ ] Each archetype runs at least 10 times per build; waypoint graphs cover all required checkpoints and secrets under test.
- [ ] Idle-bot run included: 15 minutes of no input must neither progress quests nor soft-lock the game.

## 3. Soak, Stuck & Crash Detection
- [ ] Watchdog heartbeats recorded every 5 s for the full soak; gaps investigated, never ignored.
- [ ] Stuck defined numerically (zero position variance for 90 s or no frames for 10 s) and enforced, not eyeballed.
- [ ] Every incident bundles screenshot, full state dump, and trailing 60 s of inputs.
- [ ] Memory growth across the soak stays below 50 MB; crash dumps symbolicated with build id recorded.

## 4. Telemetry & Regression Evidence
- [ ] Telemetry CSV captures completion times, death positions, and economy totals per run with seed and build id columns.
- [ ] Baseline diff computed with the 5% tolerance rule; breaches carry design-signed waivers or bug IDs.
- [ ] Death-position heatmap regenerated and attached for level-design review.
- [ ] Automated verification script `scripts/verify.sh` executes with exit code 0.
