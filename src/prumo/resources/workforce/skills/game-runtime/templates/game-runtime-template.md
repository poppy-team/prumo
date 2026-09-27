# Game Runtime Report — Fixed-Timestep Cutover

## 1. Metadata
- **Skill**: game-runtime (Game Runtime Engineering)
- **Date**: 2026-09-08
- **Author / Agent**: engine-core-team
- **Target Goal / Phase**: GOAL-120 fixed-timestep-cutover

## 2. Executive Summary
Cut the runtime over to a fixed 60 Hz timestep with render interpolation (`alpha = acc / dt`), capped catch-up (max 5 steps, frame clamp 100 ms), and atomic scene transitions. Determinism proven (identical state hash at tick 36,000 across runs); injected 250 ms hitches cause bounded slowdown with no spiral; 50 scene-transition cycles leak zero resources.

## 3. Inputs & Scope
- **Inputs Evaluated**: legacy variable-dt loop (`main_loop.cpp`), scene stack (menu/lobby/match/results), 16.6 ms frame budget, suspend/resume contract
- **Artifacts Modified**: `runtime/main_loop.cpp`, `runtime/scene_stack.cpp`, `runtime/interpolation.cpp` (new)

## 4. Key Findings & Implementation Details
- **Accumulator**: `dt = 1/60 s`, `MAX_STEPS = 5`, frame clamp 100 ms. Hitch injection (250 ms × 20): catch-up bounded at 5 steps, remainder dropped as slow-motion (1 event logged per incident, no spiral).
- **Interpolation**: Previous + current state retained per interpolated transform; render blends at `alpha`; presentation smooth at 120/144 Hz displays (verified with high-speed capture, no judder above baseline).
- **Scenes**: Pushdown stack with `enter/exit/pause/resume`; transition loads next scene to checkpoint then swaps atomically; old scene releases post-commit (resource audit: 0 leaks / 50 cycles).
- **Budgets**: Per-scene debug asserts (AI ≤ 2 ms, physics ≤ 4 ms); match scene p95: AI 1.1 ms, physics 2.8 ms.
- **Suspend/resume**: Pause stores exact tick + accumulator; resume reproduces bit-identical state (hash match 10/10 trials); device-loss path recreates GPU resources via documented sequence.

## 5. Verification & Evidence
- **Evidence Type**: test, benchmark
- **Test Results**: Passed — determinism hashes match 5/5; hitch test no spiral 20/20; scene cycles 0 leaks; resume hashes 10/10
- **Static Analysis Status**: Pass — runtime lint clean

## 6. Next Steps & Handoff
- Netcode tick alignment with new fixed step (GOAL-124); owner: net-team.
- Tune slow-motion drop policy with design (currently 5-step cap); owner: game-design.
