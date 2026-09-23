# Audio Engineering Reference Guide

## Mix Contract

Fix sample rate, channel layout, buffer size, buses, channel ownership, and latency ceiling before implementing effects. A typical contract is 48 kHz, 256-frame buffers, buses for master/music/SFX/dialogue/voice, and no more than 40 ms input-to-output latency.

## Real-Time Boundary

The audio callback may only read prepared buffers, advance deterministic state, and mix samples. It must not allocate, lock, perform I/O, resolve resources, or call general-purpose logging. Move those operations to producer threads through bounded lock-free queues.

## Voice Lifecycle

A voice moves through `idle → queued → priming → playing → releasing → idle`. Admission respects polyphony, priority, distance, and voice limits. Reuse an explicit pool and report the steal policy. Apply short ramps to starts, stops, loop wraps, and discontinuous parameter changes.

## Spatialization

Convert world positions into listener coordinates, apply clamped distance attenuation, then orientation and HRTF or panning. Occlusion combines gain and low-pass filtering. Avoid a single infinite inverse-square curve at zero distance; define near and far clamps and a unit-test table.

## Streaming and Mixing

Decode and resample on worker threads. Ring buffers must communicate underrun and overrun state, and streaming must crossfade loop seams. Preserve bus headroom: use conservative gain staging and reserve the master limiter for true peaks.

## Patterns and Anti-Patterns

| Do | Avoid |
|---|---|
| Precompute DSP parameters off the callback | Call an allocator from the callback |
| Bound every queue and report drops | Assume disk and network keep up |
| Store expected attenuation values | Tune spatialization only by ear |
| Separate music, dialogue, and effects buses | Mix every source directly at full scale |
| Soak-test target devices | Declare success from desktop speakers only |

## Short Example

For a listener distance of 8 m, configured attenuation is `-18.6 dB` and full wall occlusion adds `-12 dB`, producing `-30.6 dB`. A boundary test asserts the measured gain within `0.2 dB` of that table.
