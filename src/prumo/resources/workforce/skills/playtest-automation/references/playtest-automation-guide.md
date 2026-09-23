# Playtest Automation Reference Guide

## 1. Core Concepts

### 1.1 Deterministic Input Replay
Record timestamped inputs at the simulation tick (60 Hz), not wall-clock time, alongside the RNG seed. Replay feeds the same stream into a fresh fixed-timestep run. Divergences come from exactly four sources: unseeded RNG, variable timesteps, unordered iteration (hash maps, thread races), and external input (network, clock). Eliminate all four or the replay is theater.

### 1.2 Oracles: What "Correct" Means for a Playthrough
Bots cannot judge fun, but they assert facts: quest flags reached in order, completion within a time window, death counts within tolerance, economy totals balanced. Write oracles as inequalities (`completions <= 15 min`, `coins in [180, 240]`), never exact values, except for golden replays where exact quest timelines are the point.

### 1.3 Bot Archetypes and Waypoint Graphs
One bot proves nothing; a cohort proves robustness. Standard five: speeder (optimal route, skips optionals), explorer (visits every waypoint, opens every door), completionist (all collectibles), combat-heavy (engages every spawn), idle (no input, the soft-lock hunter). Waypoints form a graph with stuck-detection radii; bots repath when progress stalls 15 s.

### 1.4 Timescale and Fidelity
Running at 4x finds crashes four times faster but only if simulation fidelity is identical. Physics substeps, AI tick rates, and animation-driven events must scale with simulation time, not wall time. Re-prove determinism at each timescale used; cap at the highest scale where golden replays still match.

### 1.5 Stuck, Hang, and Crash Taxonomy
- **Stuck**: frames flow, position variance zero for 90 s (bot trapped by geometry or AI deadlock).
- **Hang**: no frames for 10 s (infinite loop, deadlock, GPU stall).
- **Crash**: process exit with fault (null deref, assert, OOM). Each class needs different capture: stuck wants navmesh location plus AI blackboard, hangs want thread stacks, crashes want minidumps.

## 2. Patterns and Anti-Patterns

| Pattern (do) | Anti-pattern (do not) |
|---|---|
| Seed every RNG stream and log seeds per run | One global `rand()` shared by gameplay and particles |
| Version input logs; reject stale versions loudly | Silently replay v4 logs into a v6 build |
| Watchdog heartbeats plus per-incident bundles | "Soak ran overnight, looked fine" with no artifacts |
| 5% telemetry tolerance with signed waivers | Re-baselining every run until green |
| Idle-bot and pause-menu-bot coverage | Only the golden path, only at 1x |

## 3. Minimal Example: Replay Runner With Watchdog (Bash)

```bash
#!/usr/bin/env bash
# tools/run_golden_replay.sh — replay, watchdog, incident bundle.
set -euo pipefail
SEED=4417
INPUTS="repro/inputs_harbor_v6.bin"
OUT="playtest/2026-09-13"

./game_headless --playtest-headless --timescale 4 \
  --replay "$INPUTS" --seed "$SEED" \
  --state-log "$OUT/state.jsonl" --telemetry "$OUT/telemetry.csv" &
GAME_PID=$!

# Watchdog: heartbeat + stuck detection (90 s without progress).
python3 tools/watchdog.py --pid "$GAME_PID" --state-log "$OUT/state.jsonl" \
  --stuck-seconds 90 --screenshot-dir "$OUT/shots" --bundle-dir "$OUT/incidents"

wait "$GAME_PID"
python3 tools/diff_quest_timeline.py --expected repro/golden_timeline.json \
  --actual "$OUT/state.jsonl" --tolerance-seconds 2
```

The watchdog samples the state log every 5 s, screenshots and bundles state plus trailing inputs on stuck/hang, and the diff step enforces the 2-second golden timeline tolerance. CI fails the build on any nonzero exit.
