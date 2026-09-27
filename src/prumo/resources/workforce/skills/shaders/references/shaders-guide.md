# GPU Shader Engineering — Architectural Reference Guide

## 1. Memory Layout Standards: std140 vs std430 vs WGSL

Understanding memory layouts is vital to guarantee that host languages (C++, Rust, Go) pack binary memory exactly as expected by GPU compute and raster units.

### Alignment Comparison Table
| Type | std140 (Uniform Buffer) Alignment | std140 Array Stride | std430 (Storage Buffer) Alignment | std430 Array Stride |
|------|-----------------------------------|---------------------|-----------------------------------|---------------------|
| `float`, `int`, `uint` | 4 bytes | 16 bytes (Padded!) | 4 bytes | 4 bytes |
| `vec2` | 8 bytes | 16 bytes (Padded!) | 8 bytes | 8 bytes |
| `vec3` | 16 bytes | 16 bytes | 16 bytes | 16 bytes |
| `vec4` | 16 bytes | 16 bytes | 16 bytes | 16 bytes |
| `mat4` | 16 bytes (as 4 $\times$ vec4) | 16 bytes per col/row | 16 bytes (as 4 $\times$ vec4) | 16 bytes per col/row |
| Struct | Round up to 16 bytes | 16 bytes multiple | Max member alignment | Match struct size |

### The `vec3` Hazard in Uniform Buffers
```wgsl
// PROBLEMATIC: CPU sizeof = 16 bytes, but GPU treats vec3 as 16-byte aligned.
// If CPU writes float3 followed by float, layout mismatch occurs on hardware.
struct BadUniforms {
    light_pos: vec3<f32>,   // offset 0, size 12, alignment 16
    intensity: f32,         // offset 12 -> On some drivers, expected at offset 16!
}

// CORRECT: Explicit padding field guarantees identical layout across CPU and GPU
struct GoodUniforms {
    light_pos: vec3<f32>,   // offset 0..12
    _pad0: f32,             // offset 12..16 (explicitly fills the 16-byte boundary)
    intensity: f32,         // offset 16..20
    _pad1: vec3<f32>,       // offset 20..32 (pads struct to 32 bytes)
}
```

---

## 2. GPU Hardware Execution Model & Warp Divergence

Modern GPUs do not execute threads independently. They schedule execution units in fixed batches:
- **NVIDIA**: Warp = 32 threads.
- **AMD**: Wavefront = 64 threads (or 32 in RDNA Wave32 mode).
- **Apple Silicon / Mobile**: SIMD group = 16 or 32 threads.

```
                    Warp Divergence Penalty
           +----------------------------------------+
           | Thread 0..15 take TRUE (Branch A)      |
           | Thread 16..31 take FALSE (Branch B)    |
           +----------------------------------------+
                               |
                               v
Step 1: Execute Branch A (Threads 16..31 idle/masked off)
Step 2: Execute Branch B (Threads 0..15 idle/masked off)
Total Time = Time(Branch A) + Time(Branch B)  <-- 50% ALU Efficiency!
```

### Branchless Optimization Patterns
```hlsl
// Bad: Dynamic branching inside tight loop
if (value > threshold) {
    color = albedo * 1.5;
} else {
    color = albedo * 0.5;
}

// Good: Branchless execution via select/mix
float condition = step(threshold, value); // 1.0 if value >= threshold, else 0.0
color = albedo * mix(0.5, 1.5, condition);
```

---

## 3. Derivative Operations & Uniform Control Flow

Pixel shaders calculate implicit level of detail (LoD) for mipmapped texture sampling using hardware quad derivatives:
$$\frac{\partial u}{\partial x} = \text{quad\_right}.u - \text{quad\_left}.u$$
$$\frac{\partial v}{\partial y} = \text{quad\_bottom}.v - \text{quad\_top}.v$$

If texture sampling instructions (`textureSample()` in WGSL, `tex2D()` in HLSL, `texture()` in GLSL) are placed inside divergent branches:
- Quads are broken (some pixels in the $2 \times 2$ stamp do not execute the instruction).
- Derivatives become undefined or return zero, resulting in catastrophic mipmapping artifacts or hardware crashes.

**Solution**: Always use explicit LoD sampling (`textureSampleLevel(tex, smp, uv, 0.0)`) when sampling inside non-uniform loops or conditional statements.

---

## 4. Workgroup Memory & Bank Conflicts in Compute

Workgroup shared memory is organized into 32 separate memory banks (4 bytes wide each):
- When multiple threads in a warp access different words in the same memory bank simultaneously, access is serialized (**Bank Conflict**).
- When all threads access the same word, a hardware **broadcast** occurs with zero serialization.

### Padding Shared Memory Arrays
```wgsl
// 16x16 tile with 1-element row padding to avoid 32-bank stride conflicts:
var<workgroup> tile: array<vec4<f32>, 16 * 17>; // 17 floats per row shifts banks
```

---

## 5. Cross-Compilation & Reflection Workflow

```
[WGSL / HLSL Source] 
         |
         v (dxc / naga / glslc)
[SPIR-V 1.5 Bytecode]
         |
    +----+---------------------------+
    |                                |
    v (SPIRV-Cross / Naga)           v (spirv-reflect / naga)
[Backend Shaders: MSL, HLSL, DXIL]   [CPU Header Generation: Rust / C++ structs]
```
Offline reflection extracts:
1. Exact struct byte sizes and member offsets.
2. Descriptor set, binding index, and resource types (`UniformBuffer`, `SampledTexture`, `StorageBuffer`).
3. Push constant / root constant memory ranges.
