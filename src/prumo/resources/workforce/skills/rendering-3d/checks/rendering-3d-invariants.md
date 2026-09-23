# 3D Rendering Pipeline — Engineering Invariants & Quality Checklist

Every 3D graphics pipeline subsystem (Forward+, Deferred, or Clustered) in Prumo must strictly conform to these architectural and hardware-level invariants before production deployment.

## 1. Frame Lifecycle & Memory Allocation Invariants
- [ ] **Zero Dynamic Allocations in Hot Paths**: `malloc`, `free`, `new`, `delete`, or dynamic container reallocation (`std::vector::push_back` beyond capacity) are strictly forbidden within `render()`, `update()`, or command recording loops.
- [ ] **Frame Ring Buffers**: Dynamic per-frame data (uniforms, transform matrices, light lists) must reside in pre-allocated ring buffers indexed by `frameIndex % FRAME_OVERLAP` (double/triple buffering).
- [ ] **Buffer Sub-allocation**: Uniform and storage buffers must use alignment-aware dynamic offsets (`minUniformBufferOffsetAlignment`, typically 256 bytes) instead of binding hundreds of distinct small buffers.

## 2. Geometry & Visibility Culling Invariants
- [ ] **Early Frustum Culling**: Hierarchical view-frustum culling (AABB vs 6 frustum planes) must filter draws on CPU (or via Compute shader indirect draws) before command buffer submission.
- [ ] **Occlusion Culling / Depth Pre-Pass**:
  - In Forward/Forward+ pipelines with dense geometry, an Early Depth Pre-Pass with depth-write enabled and zero color writes must precede the lighting pass.
  - In Deferred pipelines, Early-Z must be enabled (`Z-test LESS/EQUAL`, read-only depth during lighting evaluation).
- [ ] **Vertex Cache & Layout Optimization**:
  - Interleaved vertex attributes (Position, Normal, UV, Tangent) for general passes, or separated Position stream (`float3`) for Depth-only / Shadow passes to maximize vertex cache bandwidth.
  - Index buffers must be reordered using vertex cache optimization algorithms (e.g., Forsyth or Tom Forsyth / Meshoptimizer).

## 3. Lighting & PBR Material Invariants
- [ ] **Microfacet Model Conformance**: Shading must adhere to physical Cook-Torrance specular BRDF:
  $$f_r = k_d \frac{c}{\pi} + \frac{D(h) F(v,h) G(l,v,h)}{4 (n \cdot l) (n \cdot v)}$$
- [ ] **Energy Conservation**: Total outgoing reflectance cannot exceed incident light: $k_d + F \le 1.0$, with diffuse attenuated by Fresnel: $k_d = (1 - F) \times (1 - \text{metallic})$.
- [ ] **Dielectric Base Reflectance**: Non-metallic surfaces must use $F_0 = 0.04$ (or user IOR equivalent); pure metals must tint $F_0$ with albedo and set diffuse to zero.
- [ ] **Roughness & Normal Protection**:
  - Normal vectors must be re-normalized per-fragment after interpolation: `n = normalize(v_normal)`.
  - Roughness must be clamped ($\alpha = \max(\text{roughness}^2, 0.001)$) to prevent division by zero in GGX denominator.

## 4. Shadow Mapping & Precision Invariants
- [ ] **Texel Snapping**: Cascaded shadow map (CSM) orthographic projection centers must be snapped to world-space increments of $\frac{2 \times \text{radius}}{\text{textureResolution}}$ to eliminate texel crawling/swimming during camera translation.
- [ ] **Adaptive Depth Biasing**: Shadow depth testing must apply slope-scaled depth bias (`depthBias` + `slopeScaledDepthBias`) or normal-offset bias to prevent shadow acne and peter-panning.
- [ ] **Cascade Blending**: Smooth transitions between cascade splits (PCF filtering with transition band) to prevent abrupt shadow LOD seams.

## 5. Pass Synchronization & Resource Transitions
- [ ] **Explicit Attachment Operations**:
  - Render pass attachments must declare deterministic `LoadOp` (`Clear` for first write, `Load` for accumulative) and `StoreOp` (`Store` if sampled later, `Discard`/`DontCare` for transient depth/MSAAs).
- [ ] **Pipeline Barrier Validation**:
  - Transitions between Write (e.g., Shadow map / G-Buffer output) and Read (e.g., Light pass sampling) must be guarded by explicit barriers (or WebGPU pass dependencies) without introducing redundant pipeline flushes.
- [ ] **MSAA Resolve**: Multisampled color buffers must resolve directly in hardware during render pass store rather than through auxiliary copy passes.
