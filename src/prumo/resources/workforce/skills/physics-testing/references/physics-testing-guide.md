# Physics Testing Reference Guide

## 1. Core Concepts

### 1.1 Fixed Timestep With Accumulator
Variable frame deltas make integration order-dependent and replay impossible. The canonical loop accumulates wall time and steps the world in fixed quanta:
```
acc += min(frame_time, 100 ms); steps = 0
while acc >= DT and steps < 4: world.step(DT); acc -= DT; steps += 1
```
`DT = 1/60 s`. Leftover time renders via interpolation (`alpha = acc / DT`), never by stretching the physics step.

### 1.2 Collision Layers and Masks
Each body carries a layer bit and a mask of layers it scans: contact generation is `bodyA.layer & bodyB.mask != 0`. With 8 layers the full matrix is 64 ordered pairs; test all of them. Common defect: triggers on the wrong layer firing combat callbacks, or debris colliding with pickups and draining solver budget.

### 1.3 Triggers vs Contacts
Triggers report overlap (enter/stay/exit) without impulse response; contacts resolve momentum. Fast bodies can skip thin triggers between steps (the trigger analogue of tunneling): sweep shapes or expand trigger thickness along velocity axes for anything above 500 px/s.

### 1.4 Joints and Solver Iterations
Constraints solve iteratively: position and velocity iterations (typical 8/3) trade CPU for rigidity. Long chains (ropes, ragdolls) accumulate error per link; keep chains short, raise iterations locally, or switch to Featherstone-style articulated bodies. Break forces need hysteresis: break at 120 N, never re-form below 90 N, or joints chatter.

### 1.5 Continuous Collision Detection (CCD)
Discrete stepping tunnels when `speed * DT > wall_thickness` (900 px/s * 1/60 s = 15 px per step vs 8 px walls). CCD sweeps the shape along its motion vector and finds time-of-impact. Enable CCD selectively on fast small bodies; enabling it globally wastes the sweep budget on sleeping stacks.

### 1.6 Determinism Discipline
Bitwise replay needs: fixed DT, seeded RNG (single stream, logged seed), ordered contact iteration (sort by body id before solving), no wall-clock or pointer-hash inputs, and identical binary (same compiler flags; cross-platform float equality is not guaranteed on x87/SIMD mixes).

## 2. Patterns and Anti-Patterns

| Pattern (do) | Anti-pattern (do not) |
|---|---|
| Fixed DT + accumulator + interpolation alpha | `world.step(frame_delta)` and hope |
| Exhaustive 64-pair matrix test on every physics change | Spot-check player-vs-wall and ship |
| CCD on fast small bodies; thick triggers on fast paths | Global CCD "to be safe", then blame the frame budget |
| Sorted contact iteration, seeded single RNG stream | `rand()` calls inside contact callbacks |
| Debug-draw of normals/AABBs/anchors attached to every failure | "Cannot reproduce, closing" with no capture |

## 3. Minimal Example: Deterministic Replay Hash (Python Harness)

```python
# tools/physics_replay_check.py — run headless sim twice, compare hashes.
import hashlib, subprocess, sys

SEED = 90210
INPUT_LOG = "repro/inputs_30s.bin"  # 60 Hz recorded inputs, 1800 frames

def run_once():
    out = subprocess.run(
        ["./game_headless", "--physics-replay", INPUT_LOG,
         "--seed", str(SEED), "--dump-trajectory", "-"],
        capture_output=True, check=True, timeout=120,
    )
    return hashlib.sha256(out.stdout).hexdigest()

if __name__ == "__main__":
    h1, h2 = run_once(), run_once()
    print(f"run1={h1}\nrun2={h2}")
    sys.exit(0 if h1 == h2 else 1)
```

The harness dumps per-tick body positions to stdout, hashes the stream, and fails on any divergence. Gate CI on exit code 0; any engine, compiler-flag, or contact-ordering change that breaks bitwise equality shows up as a hash mismatch before it reaches players.
