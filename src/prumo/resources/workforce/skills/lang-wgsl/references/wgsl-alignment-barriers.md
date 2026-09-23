# WGSL Memory Layout, Alignment & Synchronization Reference

## 1. Complete WGSL Type Alignment & Size Table
The W3C WebGPU specification mandates deterministic alignment rules based on the Vulkan Memory Model:

| Type | AlignOf(T) | SizeOf(T) | Notes |
|---|---|---|---|
| `bool` | 4 bytes | 4 bytes | Only in host-shareable memory if converted |
| `i32`, `u32`, `f32` | 4 bytes | 4 bytes | 32-bit scalar |
| `f16` | 2 bytes | 2 bytes | Requires `enable f16;` |
| `atomic<u32>`, `atomic<i32>` | 4 bytes | 4 bytes | Only valid in storage and workgroup |
| `vec2<T>` | 2 * AlignOf(T) | 2 * AlignOf(T) | 8 bytes for 32-bit floats |
| `vec3<T>` | **4 * AlignOf(T)** | **3 * AlignOf(T)** | **Align = 16B, Size = 12B (4B hole!)** |
| `vec4<T>` | 4 * AlignOf(T) | 4 * AlignOf(T) | 16 bytes for 32-bit floats |
| `mat2x2<f32>` | 8 bytes | 16 bytes | 2 column vectors of vec2 |
| `mat3x3<f32>` | 16 bytes | 48 bytes | 3 column vectors of vec3 (padded to 16B each) |
| `mat4x4<f32>` | 16 bytes | 64 bytes | 4 column vectors of vec4 |
| `array<T, N>` | AlignOf(T) | N * roundUp(AlignOf(T), SizeOf(T)) | Array stride must be aligned to AlignOf(T) |
| `struct S` | max(AlignOf(Member_i)) | roundUp(AlignOf(S), Offset_last + Size_last) | Struct size rounds up to its own alignment |

## 2. The `vec3` Trap & Host Memory Crosswalk
In C++ and Rust, a `struct` with a `[f32; 3]` and a `f32` may pack tightly into 16 bytes:
```rust
// Rust: 16 bytes total
#[repr(C)]
struct HostData {
    pos: [f32; 3], // offset 0
    time: f32,     // offset 12
}
```
In WGSL, `vec3<f32>` requires 16-byte alignment, meaning `pos` will occupy offsets 0..12, leaving bytes 12..15 as padding. If `time` is declared next, WGSL might place it at offset 12 or force alignment. To prevent undefined behavior across compilers, **always write explicit attributes**:
```wgsl
struct HostData {
    @align(16) pos: vec3<f32>,
    time: f32,
};
```
Better yet: use `vec4<f32>` for all 3D vectors in Uniform buffers, storing `time` in `pos.w`.

## 3. Workgroup Barrier Memory Synchronization
```wgsl
var<workgroup> tile: array<f32, 256>;

@compute @workgroup_size(256)
fn computeFilter(@builtin(local_invocation_id) lid: vec3<u32>) {
    // 1. Unsynchronized independent write
    tile[lid.x] = loadSample(lid.x);

    // 2. Memory Barrier:
    // Flushes all pending writes to workgroup memory and guarantees
    // memory visibility to all other invocations in this workgroup.
    workgroupBarrier();

    // 3. Safe read from any index written by any sibling invocation
    let blur = (tile[lid.x] + tile[(lid.x + 1u) % 256u]) * 0.5;
}
```
The barrier enforces an Acquire/Release semantics:
- **Release**: Invocations finish all prior writes to workgroup memory.
- **Acquire**: Subsequent reads observe all writes executed before the barrier.