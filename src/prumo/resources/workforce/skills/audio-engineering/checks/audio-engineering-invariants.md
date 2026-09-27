# Audio Engineering Verification Checklist

## Mix Contract
- [ ] Sample rate, buffer size, bus layout, channel count, and latency ceiling are explicit.
- [ ] Effects order, sends, headroom, and master limiting follow the mix design.
- [ ] Target output devices and platform mixer constraints are included in evidence.

## Real-Time Safety
- [ ] The audio callback performs no allocation, lock, filesystem/network I/O, or unbounded logging.
- [ ] Producer queues and ring buffers are bounded and expose underrun or overrun state.
- [ ] Voice pools, polyphony limits, and steal priority are deterministic.

## Spatial Audio
- [ ] Listener coordinates, attenuation clamps, HRTF or panning, and occlusion have unit tests.
- [ ] Start, stop, loop wrap, and discontinuous parameter changes use ramps.
- [ ] Expected gain and filter values are measured at near, far, and occlusion boundaries.

## Streaming and Playback
- [ ] Decoder and resampler work runs off the callback and pre-buffers before start.
- [ ] Loop transitions and crossfades are gapless across repeated playback.
- [ ] Device loss, format changes, disconnect, and reconnect recover without stale voices.

## Benchmarks and Evidence
- [ ] A full-polyphony soak records zero underruns and zero clipping events.
- [ ] Output latency and callback CPU remain within target-device budgets.
- [ ] `scripts/verify.sh` exits with code 0 from the repository root.
