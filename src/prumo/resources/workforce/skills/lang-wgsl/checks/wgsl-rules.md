# WGSL WebGPU Soundness Checklist

## 1. Struct Alignment & Padding Invariants
- [ ] All Uniform buffer structs have sizes that are clean multiples of 16 bytes.
- [ ] Any `vec3<T>` fields are either expanded to `vec4<T>` or have verified trailing padding fields.
- [ ] Struct member offsets match host-side memory representations (e.g. `bytemuck`, `encase`, C++ `alignas(16)`).
- [ ] Matrices (`mat4x4<f32>`) are aligned to 16 bytes and occupy 64 bytes.

## 2. Address Spaces & Bindings
- [ ] Every global variable specifies an address space (`var<uniform>`, `var<storage>`, `var<workgroup>`).
- [ ] Every pipeline resource has distinct `@group(G) @binding(B)` attributes without collisions.
- [ ] Read-only storage buffers use `var<storage, read>` to allow GPU hardware caching.

## 3. Concurrency & Compute Correctness
- [ ] Workgroup size `@workgroup_size(X, Y, Z)` total product is <= 256.
- [ ] Workgroup shared memory (`var<workgroup>`) is synchronized with `workgroupBarrier()` before reading.
- [ ] Concurrent shared modifications use `atomic<u32>` operations rather than bare arithmetic.
- [ ] Out-of-bounds access is guarded against dynamic buffer lengths via `arrayLength(&buf)`.

## 4. Control Flow & Texture Sampling
- [ ] Implicit derivative operations (`textureSample`, `dpdx`, `dpdy`) are strictly confined to uniform control flow.
- [ ] Divergent branch sampling uses `textureSampleLevel` with explicit mip levels.
- [ ] Shaders compile cleanly without warnings or errors under `naga` or `tint`.