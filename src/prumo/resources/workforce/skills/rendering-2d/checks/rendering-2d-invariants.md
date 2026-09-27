# 2D Rendering Pipeline — Engineering Invariants & Quality Checklist

Every 2D rendering subsystem, sprite batcher, and vector canvas in Prumo must conform strictly to these architectural and performance invariants before shipping.

## 1. Memory & Buffer Allocation Invariants
- [ ] **Zero Per-Frame Heap Allocations**: Memory for vertex and index buffers must be pre-allocated once during initialization. Calling `malloc`, `realloc`, or `std::vector::push_back` beyond capacity in `begin()`, `draw()`, `flush()`, or `end()` is strictly prohibited.
- [ ] **CPU-Mapped Ring Buffering**: Dynamic 2D vertex data must use persistently mapped ring buffers with dynamic offset alignment to prevent CPU-GPU pipeline stalls.
- [ ] **Index Buffer Reuse**: The index buffer for quads ($0, 1, 2, 2, 3, 0$, offset by $4 \times \text{quadIndex}$) is static and immutable; it must be pre-populated once at startup rather than regenerated per frame.

## 2. Batching & Draw Call Invariants
- [ ] **Draw Call Minimization**: Scenes containing up to 100,000 sprites must render in $\le 25$ GPU draw calls.
- [ ] **Multi-Texture Batching**: The batcher must support binding at least 8 to 16 texture slots simultaneously, encoding the texture slot index directly into the vertex attributes (`float tex_index` or `uint8_t tex_slot`).
- [ ] **Deterministic Flush Triggers**: A GPU draw call flush must only occur when:
  1. Vertex buffer reaches maximum batch capacity ($V \ge V_{\text{max}}$).
  2. All texture slots are occupied and an unmapped texture is requested.
  3. The active hardware scissor rectangle changes.
  4. The blend mode changes (e.g. from Premultiplied Alpha to Additive).

## 3. Texture Atlas & Bleed Prevention Invariants
- [ ] **2-Pixel Gutter Padding**: Every sprite packed into an atlas must include at least 1 (preferably 2) gutter pixels replicated from its edges to prevent bilinear filtering color leakage at fractional camera coordinates.
- [ ] **Power-of-Two Dimensions**: Atlases must adhere to power-of-two resolutions ($1024 \times 1024$, $2048 \times 2048$, $4096 \times 4096$) with mipmapping configured with clamp-to-edge.

## 4. Blending & Color Space Invariants
- [ ] **Premultiplied Alpha Mandatory**: All transparent sprite textures must be encoded in Premultiplied Alpha format ($R' = R \cdot A$, $G' = G \cdot A$, $B' = B \cdot A$), with blending set to `GL_ONE, GL_ONE_MINUS_SRC_ALPHA` (or WebGPU equivalent). Straight alpha blending with dark edge fringe artifacts is rejected.
- [ ] **Color Space Consistency**: Color uniforms and vertex tints must be transformed to linear space prior to fragment shading and blended in linear color space before swapchain gamma encoding.

## 5. Typography & Vector Rendering Invariants
- [ ] **MSDF Antialiasing**: Vector typography and scalable UI glyphs must utilize Multi-channel Signed Distance Fields (MSDF) with screen-space derivative antialiasing (`fwidth()` or screen pixel range), ensuring sharp contours at any scaling factor.
