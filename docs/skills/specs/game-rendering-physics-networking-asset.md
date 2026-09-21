# 79.O — Game, Rendering, Physics, Networking, Assets, Audio e Realtime Skills

> Authority: canonical specification.
> Logical ID: 79 O
> Source: Notion Living Book (3d89bb7d023f81bb9fdee94b104e9dd1)
> Status: Skill Package de referência para 79 O — Game, Rendering, Physics, Networking, Asset.


<aside>
🎮

O domínio de games/graphics possui forte dependência de **determinismo, frame budgets, hardware, input, asset provenance e simulação adversarial**. Skills devem evitar checklists web genéricos e produzir fixtures de engine/runtime.

</aside>

| Skill | Pri. | Hardening | Aceite |
| --- | --- | --- | --- |
| **asset-pipeline** | P1 | Versioned importers, content hashes, deterministic transforms, cache invalidation, provenance/license, malformed assets e rebuild tests. | Mesmo input+toolchain produz mesmo derivado. |
| **asset-reference-research** | P1 | Fonte, autor/licença, allowed usage, confidence, duplicate detection, technical extraction. | Asset externo possui provenance/licença. |
| **audio-engineering** | P1 | Latency budget, sample rates/resampling, clipping/mixing, spatial audio, streaming, realtime-thread safety, device matrix. | Tests detectam underrun/clipping/allocation em realtime path. |
| **game-engine-architecture** | P1 | Subsystem ownership, update order, ECS/scene boundaries, platform abstraction, threading, deterministic modes e profiling hooks. | Architecture fitness protege boundaries. |
| **game-runtime** | P1 | Fixed/variable timestep, pause/background, save/load, replay, crash recovery, deterministic simulation e resource lifecycle. | Replay determinístico onde contract exigir. |
| **game-ui-testing** | P1 | Gamepad/keyboard/mouse/touch, safe area, resolutions/aspect ratios, localization, a11y, frame timing e focus navigation. | Input×resolution×locale matrix. |
| **input-handling** | P1 | Remapping, chords, repeat/debounce, dead zones, hot-plug, focus/context stacks, accessibility e persistence. | Input mapping independente de renderer. |
| **multiplayer-networking** | P1 | Authority, prediction, reconciliation, interpolation, protocol evolution, determinism, bandwidth, cheat resistance e interest management. | Latency/loss/jitter/reorder simulations. |
| **network-testing** | P1 | Deterministic profiles para latency/loss/jitter/reorder/duplicate/partition/bandwidth/half-open/reconnect. | Failure reproduzível por seed/profile. |
| **realtime-synchronization** | P1 | Sequence IDs/clocks, conflict resolution, replay/resume, offline sync, duplication, backpressure e reconnect. | Partition/reconnect preserva invariantes. |
| **rendering-2d** | P1 | Batching, atlases, text/fonts, clipping, color spaces, HiDPI, resource lifetime, CPU/GPU budgets e golden images. | Representative scene image+perf baselines. |
| **rendering-3d** | P1 | Mesh/material/light/camera, depth/culling/LOD, HDR/color, resource lifetime, GPU capabilities, picking e perf. | Correctness visual+GPU budget evidence. |
| **scene-graph** | P1 | Parent cycles, lifetime, reparenting, dirty propagation, serialization, stable IDs, threading e huge-scene perf. | Cycle prevention/transform propagation tests. |
| **shaders** | P1 | Permutations, reflection, resource bindings, precision, cross-backend validation, hot reload, cache e golden rendering; delegar GLSL/HLSL/WGSL. | Shader compile/render matrix. |
| **physics-collision** | P1 | Broad/narrow phase, CCD/tunneling, contact stability, layers/masks, triggers, tolerances, determinism e perf. | Golden geometry scenes e edge cases. |
| **physics-testing** | P1 | Property tests, deterministic seeds, invariant checks, golden scenes, fuzz geometry e replay. | Tolerances explícitas; sem float equality ingênua. |
| **playtest-automation** | P1 | Scenario DSL, deterministic seeds, bots, checkpoints, telemetry, screenshots/replays e flaky classification. | Failure gera reproduction artifact. |

# Asset reference vs visual reference

`asset-reference-research` é owner de **fonte técnica/licença/asset reutilizável**. `visual-reference-research` é owner de **linguagem visual/padrões/estética e abstraction**. Resolver pode usar ambas, mas por razões diferentes.

# Deterministic simulation contract

Cenários automatizados devem poder registrar seed, simulation version, tick rate/timestep, initial state, inputs/events e expected invariants. Quando determinismo estrito não for possível, declarar tolerances e fontes conhecidas de variance.

# GPU/renderer evidence

Registrar backend/API, GPU/adapter class quando disponível, driver/runtime, resolution, frame count/warmup, shader set, scene fixture e CPU/GPU timing methodology.

# Performance budgets

Budgets devem ser profile-specific: editor vs game runtime, 2D vs 3D, desktop vs mobile, target FPS, memory constraints, network bandwidth, audio latency. Evitar thresholds universais sem projeto/target.

# Exit Gate

Game/graphics skills possuem fixtures representativas, determinism/perf where relevant, network adversity, resource lifetime checks e clear language/backend delegation.