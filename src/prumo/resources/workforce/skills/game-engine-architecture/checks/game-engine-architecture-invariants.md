# Game Engine Architecture Verification Checklist

## Data and Dependency Boundaries
- [ ] Entities, components, systems, resources, and subsystem ownership are explicit.
- [ ] Core simulation does not depend on drivers, UI, storage formats, or platform SDKs.
- [ ] Task graphs declare read/write sets and contain no unproven cycle or race.

## ECS and Memory Layout
- [ ] Component layout, archetype or sparse-set strategy, and migration costs are documented.
- [ ] Handles include generation or equivalent stale-reference protection.
- [ ] Hot paths use bounded pools, arenas, and containers with measured allocation counts.

## Scheduling and Determinism
- [ ] System phases, dependencies, worker ownership, and job lifetimes are deterministic.
- [ ] Fixed-step simulation excludes wall-clock time and unseeded randomness from authoritative state.
- [ ] Identical input scripts produce identical state hashes across repeated runs.

## Resource and Spatial Lifecycle
- [ ] Resource handles, streaming states, fallback tiers, and retirement rules are explicit.
- [ ] Device loss and world teardown release every owned resource deterministically.
- [ ] Spatial structures, coordinate conventions, and acceleration bounds are tested.

## Performance Evidence
- [ ] Target CPU, memory, load, and frame budgets are measured on representative scenes.
- [ ] Profiling captures allocation, race, cache, and task-system counters.
- [ ] `scripts/verify.sh` exits with code 0 from the repository root.
