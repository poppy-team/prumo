# Game Runtime Reference Guide

## Fixed Simulation Step

Gameplay and physics advance with a fixed step. The runtime adds a clamped frame delta to an accumulator and executes at most `MAX_STEPS` updates per frame. It drops or slows excess time explicitly rather than entering a death spiral.

```text
alpha = accumulator / fixed_delta
while accumulator >= fixed_delta and steps < max_steps:
    simulate(fixed_delta)
    accumulator -= fixed_delta
render(lerp(previous, current, alpha))
```

## Interpolation

Keep the previous and current simulation snapshots required for rendering. Interpolate display transforms only; do not mutate simulation state from presentation. Document which fields are render-only and which remain discrete.

## Scene Stack

Scenes are explicit states with `enter`, `exit`, `pause`, and `resume`. Load the next scene to a verified checkpoint before swapping it into the active stack. Commit the transition atomically, then release the old scene and its owned resources.

## Hostile Environment Events

Focus loss, suspend, resize, display change, and device loss are state transitions, not ad hoc exceptions. Pause at a tick boundary, record the accumulator and interpolation state, recreate resources through the documented recovery path, and resume without reseeding or silently skipping ticks.

## Determinism

Simulation code cannot use wall-clock time, unseeded randomness, unordered iteration, or platform-dependent floating behavior when determinism is required. Verification compares state hashes at fixed tick boundaries across repeated runs and worker counts.

## Patterns and Anti-Patterns

| Do | Avoid |
|---|---|
| Cap catch-up work | `while accumulator >= dt` without a bound |
| Interpolate presentation only | Feed render interpolation back into physics |
| Load then swap scenes | Expose a half-initialized scene |
| Resume from a saved tick boundary | Recreate state from menu defaults |
| Report dropped simulation time | Hide a sustained frame-rate collapse |

## Short Example

A 250 ms hitch with a 16.67 ms fixed step and a five-step cap performs five updates, retains a bounded remainder, and reports degraded time. The next frames resume without unbounded catch-up.
