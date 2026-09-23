# Realtime Synchronization — Verification Checklist

## 1. Tick Timeline & Determinism
- [ ] Fixed tick rate declared (20 Hz shooter, 10 Hz strategy) with accumulator decoupling render frames from ticks.
- [ ] All gameplay reads integer tick IDs as time; wall-clock or frame-count timing is absent from simulation.
- [ ] Per-tick state hash (FNV-1a over positions and health) matches across peers; desync detected within 5 ticks.
- [ ] Simulation avoids raw float nondeterminism via fixed-point or quantized floats with cross-platform proof.

## 2. Sync Model & Prediction
- [ ] Sync model documented: lockstep with input delay, rollback with prediction window, or snapshot interpolation.
- [ ] Rollback implementations cap prediction at 8 frames with mispredict rate under 5 percent in soak tests.
- [ ] Reconciliation against authoritative snapshots completes within 100 ms of receipt.
- [ ] Remote entities interpolate over a 100 ms jitter buffer; extrapolation never exceeds 250 ms.

## 3. Bandwidth & Snapshot Discipline
- [ ] Snapshots delta-compressed at 10–20 Hz cadence; full snapshots every 30 ticks for resync.
- [ ] Per-client downstream stays under 64 kbps; detail shedding (LOD) precedes player shedding.
- [ ] Input packets carry tick ID, checksum, and redundancy for the last 3 inputs against single loss.
- [ ] Clock sync uses NTP-offset estimates with monotonic timers; drift alarms fire past 50 ms skew.

## 4. Reconnect, Resync & Evidence
- [ ] 10-second reconnect grace with snapshot plus input-log catch-up; late join downloads full state first.
- [ ] Host migration elects a new authority within 5 seconds with zero score or inventory loss.
- [ ] 4-player 30-minute soak at 80 ms RTT, 15 ms jitter, 1 percent loss shows zero desyncs.
- [ ] scripts/verify.sh executes with exit code 0.
