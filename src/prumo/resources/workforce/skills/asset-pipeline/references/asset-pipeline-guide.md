# Asset Pipeline Reference Guide

## Deterministic Cooking

A cooker is a pure transformation of source bytes, normalized metadata, explicit settings, target profile, and cooker version. Timestamps, absolute paths, iteration order, and random seeds must not affect output. The artifact key is a digest of those inputs.

```text
artifact_key = sha256(source_digest || normalized_settings || cooker_version || target_profile)
```

Store metadata beside the artifact: source digest, tool versions, selected format, byte size, memory estimate, warnings, and build timestamp. The timestamp records when the cook ran; it does not participate in identity.

## Import and Validation

Reject malformed or over-budget sources during import. Validate dimensions, channel layout, codec, color space, compression settings, and platform limits before expensive conversion. Error messages identify the asset, violated rule, measured value, and allowed range.

## Caching and Concurrency

Key cache entries by artifact identity, not source path alone. Multiple workers may request the same key, but exactly one publishes the completed artifact through an atomic rename. Failed cooks never replace a valid cached result.

## Compression and Budgets

Choose platform formats from measured support, not preference. Generate complete mip or LOD chains when runtime sampling requires them. Enforce per-asset, per-class, level, and resident-memory ceilings during cook.

## Hot Reload and Streaming

Cook and decode away from the frame thread. Publish immutable resource revisions through handles; retire old revisions only after in-flight readers release them. Streaming keeps a low-detail fallback resident and applies bandwidth plus memory limits to prefetch.

## Patterns and Anti-Patterns

| Do | Avoid |
|---|---|
| Hash canonical inputs and tool versions | Hash only the filename |
| Reject over-budget assets at cook time | Discover a 4K texture budget failure on device |
| Swap immutable resource revisions | Mutate a live GPU texture in place |
| Make cache publication atomic | Let workers overwrite shared output files |
| Record toolchain provenance | Assume two developer machines match |

## Short Example

A source PNG, ASTC profile, mip setting, and cooker `2.7.1` produce digest `91f2...c0a4`. A second clean worker computes the same digest, reuses the artifact, and records the source revision without changing the packaged bytes.
