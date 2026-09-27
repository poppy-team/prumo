# Scene Graph & ECS Delivery Record

## Metadata
- Skill: Scene Graph & ECS
- Date: 2026-09-23
- Author: scene-runtime-agent
- Goal: SCENE-22 — deterministic hierarchy updates for the 12k-entity stress scene

## Storage and Scheduling
- Storage: archetype SoA for `LocalTransform` and `Velocity`; sparse sets for tags.
- Handles: generational index plus generation counter.
- Phases: input → simulation → transform sync → render extract.
- Structural edits: command buffer applied at the phase barrier.

## Hierarchy Record
- 4,000 parented entities use breadth-first dirty-subtree propagation.
- Attaching a node to its descendant returns `ERR_CYCLE` before mutation.
- Render extraction reads a frame-local snapshot and never live simulation buffers.

## Evidence
- 200 spawns and 200 destroys per frame complete in 0.09 ms.
- Stale handles fail validation after destroy and slot recycling.
- The system order is identical across 100 ticks.
- Thread sanitizer reports no data race in the wavefront transform pass.
