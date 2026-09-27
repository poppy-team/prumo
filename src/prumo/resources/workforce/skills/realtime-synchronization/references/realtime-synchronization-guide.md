# Realtime Synchronization Reference Guide

## 1. Core Concepts

### 1.1 Tick-Based Time
Real time is continuous and hostile; game time must be discrete. A 20 Hz tick advances the simulation
every 50 ms with an integer ID. Rendering interpolates between tick N-1 and tick N states, so 144 Hz
displays stay smooth on a 20 Hz simulation. Latency math follows: with 80 ms RTT, inputs take 40 ms up,
the server ticks, and snapshots take 40 ms down, so clients always render roughly 2–4 ticks behind
authority. Design for behind-by-default instead of fighting it.

### 1.2 Lockstep vs Rollback vs Snapshot Interpolation
Lockstep: all peers exchange inputs, then all simulate tick N identically. Deterministic and cheat-clean
but hostage to the slowest peer; input delay of 2–4 ticks masks RTT for strategy games. Rollback: each
peer predicts remote inputs, simulates immediately, and re-simulates on mispredict when true inputs arrive;
perfect feel for fighters at the cost of deterministic re-simulation and state-save every frame. Snapshot
interpolation: the authoritative server broadcasts compressed world snapshots at 10–20 Hz; clients render
100 ms in the past, interpolating smoothly. Scales to 64+ players but surrenders peer determinism.

### 1.3 Prediction, Mispredicts, and Reconciliation
Prediction hides latency by assuming remote inputs repeat the last known values. Correct predictions cost
nothing; mispredicts cost re-simulation plus visible snaps. Budget: mispredict rate under 5 percent on
typical links, correction snaps under 3 meters of game space. Reconciliation blends: snap tiny errors
instantly, interpolate medium errors over 100 ms, and only hard-resync past 250 ms divergence.

### 1.4 Jitter Buffers and Clocks
A 100 ms jitter buffer absorbs arrival variance so interpolation never starves; size it at p99 jitter plus
one tick. Clocks: NTP aligns wall time across machines within 10–30 ms, monotonic timers drive the tick
accumulator without leap-second jumps. Measure offset continuously and alarm past 50 ms skew, because a
drifting client silently converts interpolation into permanent extrapolation.

### 1.5 Bandwidth Arithmetic
Per-client downstream = snapshot Hz × average delta bytes × 8 bits. Example: 20 Hz × 400 bytes × 8 =
64 kbps, the standard cap. Exceeding it means smaller deltas (quantize positions to 12-bit grid cells,
send health only on change), lower snapshot rate for distant entities (network LOD), or fewer players per
room. Upstream is cheaper: 20 Hz × 24-byte inputs × 8 ≈ 4 kbps per player with triple redundancy.

## 2. Patterns and Anti-Patterns

| Area | Pattern (do) | Anti-Pattern (avoid) |
|---|---|---|
| Time | Integer tick IDs, accumulator loop | dt-scaled simulation with wall clock |
| State | Per-tick FNV-1a hash compared in CI | Hoping floats agree across compilers |
| Remote motion | 100 ms jitter buffer interpolation | Extrapolating 1 s into walls |
| Corrections | Snap small, blend medium, resync large | Teleporting on every mispredict |
| Bandwidth | Delta + quantization under 64 kbps | Full 8 KB world state at 20 Hz |
| Reconnect | Snapshot plus input-log catch-up | Full match restart on one dropout |
| Cheating | Server-authoritative hits and scores | Trusting client damage reports |

## 3. Code Example: Tick Accumulator with Snapshot Interpolation

```go
const tickHz = 20
const tickDt = 1.0 / tickHz          // 50 ms per tick
const interpDelay = 0.10             // render 100 ms behind authority

accumulator := 0.0
tick := 0
for frame := range renderLoop(144) {
    accumulator += frame.dt
    for accumulator >= tickDt {
        simulateTick(tick)           // fixed-step gameplay only
        tick++
        accumulator -= tickDt
    }
    // render between snapshot[tick-3] and snapshot[tick-2] for 100 ms delay
    alpha := accumulator / tickDt
    renderInterpolated(snapshot[tick-3], snapshot[tick-2], alpha)
}
```

The accumulator guarantees identical tick counts on 60 Hz and 144 Hz displays, and rendering two
snapshots back provides the 100 ms cushion that jitter and 80 ms RTT demand.
