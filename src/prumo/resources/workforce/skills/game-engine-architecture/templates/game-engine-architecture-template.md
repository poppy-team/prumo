# Game Engine Architecture Review — ECS Scheduler Pass

## 1. Metadata
- **Skill**: game-engine-architecture (Game Engine Architecture)
- **Date**: 2026-09-13
- **Author / Agent**: engine-core-team
- **Target Goal / Phase**: GOAL-125 ecs-scheduler-pass

## 2. Executive Summary
Refactored the ECS scheduler to a phased task graph (input → AI → physics → animation → render-extract) over a work-stealing pool (8 workers). Frame-time p95 dropped 11.2 ms → 7.8 ms on the target console profile; zero per-frame allocations in the hot loop (arena + pools verified by allocation counter). Deterministic replay confirmed: 10,000-tick script hashes match across 5 runs.

## 3. Inputs & Scope
- **Inputs Evaluated**: subsystem coupling map (render, physics, audio, input, animation), console frame budget 16.6 ms, replay determinism contract (fixed 60 Hz tick)
- **Artifacts Modified**: `engine/scheduler/task_graph.cpp`, `engine/memory/arena.cpp`, `engine/ecs/component_store.h`

## 4. Key Findings & Implementation Details
- **Task graph**: 5 phases with explicit read/write component sets; race detector (TSan) clean over 1,000 frames; render-extract isolated from simulation state (no gameplay/render data races).
- **Memory**: Per-frame scratch via bump arena (2 MB, reset per tick); projectiles/particles via pool allocators (4,096 slots each); allocation counter reads 0 mallocs in steady-state frames.
- **ECS layout**: Components stored in SoA chunks of 64 (cache-line friendly); entity IDs are 32-bit indices + 16-bit generations (stale-handle safe).
- **Determinism**: Fixed 60 Hz tick with integer-accumulator; replay of scripted 10,000-tick session produces identical state hash (`blake3:7a41…`) on all 5 runs.
- **Spatial queries**: Switched broadphase to spatial hash (cell 4 m); frustum-cull query p99 0.3 ms (was 1.1 ms with linear scan).

## 5. Verification & Evidence
- **Evidence Type**: test, benchmark
- **Test Results**: Passed — TSan 0 races; replay hashes 5/5 identical; allocation counter 0 in hot loop
- **Static Analysis Status**: Pass — scheduler lint clean; coupling map shows no new edges to drivers

## 6. Next Steps & Handoff
- GPU-driven culling spike (GOAL-126); owner: engine-core-team.
- Port arena sizing to handheld tier (1 MB scratch); owner: platform-team.
