# Shader Authoring & Compilation — Quality & Performance Invariants

Every shader asset (WGSL, HLSL, GLSL, SPIR-V) authored or compiled in Prumo must satisfy these critical GPU execution and memory layout invariants before entering production pipelines.

## 1. Memory Layout & Alignment Invariants
- [ ] **16-Byte Struct Alignment**: All structs mapped to Uniform Buffers (`std140` or WGSL `var<uniform>`) must have an overall size that is an exact multiple of 16 bytes.
- [ ] **Vec3 Padding Field**: Every 3-component vector (`vec3<f32>` / `float3`) in a uniform buffer must be followed by an explicit 4-byte padding field (`_pad: f32`) unless placed at the very end of the struct.
- [ ] **Array Stride Awareness**:
  - In `std140`, array elements are rounded up to 16 bytes. Arrays of scalars (`float arr[4]`) must be avoided in uniform buffers or explicitly padded to `vec4`.
  - In `std430` / storage buffers, arrays match natural alignment, but struct member offsets must be verified against host ABI.
- [ ] **Matrix Majorness**: Matrix memory layout (Column-Major vs Row-Major) must match CPU math library conventions (GLM / DirectXMath / nalgebra) without runtime matrix transpositions on the CPU.

## 2. Warp / Wavefront Execution & Divergence Invariants
- [ ] **Uniform Control Flow for Derivatives**:
  - Implicit derivative operations (`textureSample()`, `dFdx()`, `dFdy()`, `fwidth()`) are strictly forbidden inside non-uniform dynamic branches (`if` or dynamic `for` loops).
  - Inside divergent branches, explicit LOD sampling (`textureSampleLevel()` / `textureLod()`) must be used instead.
- [ ] **Branchless Inner Loops**:
  - Inner loops in compute and fragment shaders must eliminate non-uniform branch divergence by using branchless ALU select instructions (`mix()`, `step()`, `clamp()`).
- [ ] **Loop Unrolling Pragma**:
  - Fixed-size short loops (kernel radius $\le 8$) must be annotated with unroll directives (`#pragma unroll` or unrolled explicitly) to eliminate loop counter overhead and register spilling.

## 3. Compute Shared Memory & Synchronization
- [ ] **Barrier Correctness**:
  - Every write to workgroup shared memory (`var<workgroup>` / `groupshared`) must be followed by `workgroupBarrier()` before any invocation reads the memory.
  - Barriers must never be placed inside conditionally divergent blocks where not all workgroup invocations reach the barrier (preventing GPU lockup / deadlock).
- [ ] **Bank Conflict Elimination**:
  - Access patterns to workgroup shared memory must avoid 32-bank stride conflicts (e.g., by adding 1 element padding per row in 2D shared tile arrays: `array<vec3<f32>, 16 * 17>`).
- [ ] **Occupancy Workgroup Sizing**:
  - Compute workgroup sizes must be exact multiples of hardware warp/wavefront size (64, 128, or 256 threads) to ensure maximum streaming multiprocessor ALU occupancy.

## 4. Resource Bindings & Pipeline States
- [ ] **Frequency-Based Descriptor Sets**:
  - Resource bindings must strictly follow the update frequency hierarchy (`Group 0: Frame`, `Group 1: Pass`, `Group 2: Material`, `Group 3: Draw`).
- [ ] **Static Compilation & Pipeline Pre-Warming**:
  - All shaders must be compiled offline during build/packaging.
  - Zero runtime shader compilation from source strings allowed during active interactive rendering frames.
