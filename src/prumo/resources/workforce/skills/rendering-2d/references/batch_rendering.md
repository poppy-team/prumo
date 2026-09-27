# High-Performance 2D Graphics & Batch Rendering — Engineering Reference

## 1. Batch Rendering Architecture

```
Individual Sprites:  [Sprite A (Tex 1)] [Sprite B (Tex 2)] [Sprite C (Tex 1)] ...
                             |
                             v  (Collected into pre-allocated memory buffer)
Dynamic Quad Batch:  [V0..V3 (TexSlot 0)] [V4..V7 (TexSlot 1)] [V8..V11 (TexSlot 0)]
                             |
                             v  (Single DrawIndexed call to GPU)
GPU Draw Call:       DrawElements(GL_TRIANGLES, quad_count * 6, GL_UNSIGNED_INT, 0)
                     With 16 textures bound to Sampler2DArray or Sampler2D[16]
```

### 1.1 Quad Vertex Structure (32 bytes per vertex)
```cpp
struct alignas(16) Vertex2D {
    float position[2];      // 8 bytes (X, Y in world or screen coords)
    float uv[2];            // 8 bytes (U, V in atlas)
    uint32_t color;         // 4 bytes (RGBA8 packed)
    float texture_slot;     // 4 bytes (0.0 to 15.0 for sampler indexing)
    float flags;            // 4 bytes (e.g. SDF mode, rounded corner flags)
    float _pad0;            // 4 bytes (aligns to 32 bytes)
};
```

---

## 2. Multi-Texture Batching
Standard batchers flush every time a new texture is requested. Modern multi-texture batchers bind an array of textures (typically 16 or 32 slots):
```wgsl
@group(0) @binding(0) var sprite_textures: binding_array<texture_2d<f32>, 16>;
@group(0) @binding(1) var sprite_sampler: sampler;

@fragment
fn fs_main(in: VertexOutput) -> @location(0) vec4<f32> {
    let slot = u32(round(in.texture_slot));
    let tex_color = textureSample(sprite_textures[slot], sprite_sampler, in.uv);
    return tex_color * in.color;
}
```
**Benefits**: Reduces draw calls by up to 95% compared to single-texture batching.

---

## 3. Premultiplied Alpha Math & The Dark Fringe Problem

### Straight Alpha vs Premultiplied Alpha
- In **Straight Alpha**, RGB and Alpha are stored independently: $(R, G, B, A)$.
  - Interpolation between a fully transparent pixel $(0, 0, 0, 0)$ and opaque white $(1, 1, 1, 1)$ results in $(0.5, 0.5, 0.5, 0.5)$.
  - The blended color has gray $R=0.5, G=0.5, B=0.5$, producing an ugly dark/dirty halo along transparent sprite borders.
- In **Premultiplied Alpha**, RGB is pre-scaled by Alpha: $(R \cdot A, G \cdot A, B \cdot A, A)$.
  - Fully transparent pixel is $(0, 0, 0, 0)$.
  - Hardware blending formula:
    $$\mathbf{C}_{\text{result}} = \mathbf{C}_{\text{src}} \times 1 + \mathbf{C}_{\text{dest}} \times (1 - A_{\text{src}})$$
  - Blending is linear and associative, completely eliminating dark fringe artifacts and correctly supporting both regular transparency and additive glow in a single pass.

---

## 4. Texture Atlas Packing & Bilinear Bleed Protection

### 4.1 MaxRects Bin Packing
MaxRects maintains a list of free rectangular spaces. When a sprite is placed:
1. Select best-fitting free rectangle using Best-Short-Side-Fit (BSSF) or Contact-Point heuristic.
2. Split overlapping free rectangles into smaller sub-rectangles.
3. Remove completely contained sub-rectangles.

### 4.2 Gutter Math
Bilinear texture filtering samples a $2 \times 2$ texel footprint. If a sprite rect is at $[X_0, Y_0, X_1, Y_1]$:
- Sampling at coordinate $(X_1 - \epsilon)$ interpolates texels outside $X_1$ (which belongs to a neighboring sprite in the atlas).
- **Solution**: Inject a 2-pixel gutter around every sprite:
  - Copy border row $Y_0$ into $Y_0 - 1$ and $Y_0 - 2$.
  - Copy border row $Y_1$ into $Y_1 + 1$ and $Y_1 + 2$.
  - Set UV coordinates to sample strictly within $[X_0, Y_0, X_1, Y_1]$, leaving gutters to absorb filter sampling without color bleeding.

---

## 5. Multi-channel Signed Distance Field (MSDF) Typography

Unlike standard single-channel SDF which rounds sharp corners, MSDF stores distance fields for 3 distinct edge channels (Red, Green, Blue):
$$\text{median}(r, g, b) = \max(\min(r, g), \min(\max(r, g), b))$$
Corners are preserved sharply at any resolution.
Screen-space antialiasing formula:
$$\text{screen\_px} = \text{pixel\_range} \times (\text{median}(r,g,b) - 0.5)$$
$$\text{opacity} = \text{clamp}(\text{screen\_px} + 0.5, 0.0, 1.0)$$
