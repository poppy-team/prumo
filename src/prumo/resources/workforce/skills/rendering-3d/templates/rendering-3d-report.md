# 3D Rendering Pipeline — Architecture & Profiling Report

## 1. Pipeline Architecture & Target Profile
- **Subsystem**: [e.g., Forward+ Clustered / Deferred PBR Renderer]
- **Target Backend**: [e.g., WebGPU / Vulkan 1.3 / Metal]
- **Target Frame Budget**: [e.g., 60 FPS (16.6ms) / 120 FPS (8.33ms)]
- **Target Hardware Class**: [e.g., Desktop Discrete GPU / Apple Silicon / Mobile Tile-Based]

---

## 2. Render Pass Graph & Attachments
| Pass Index | Pass Name | Attachments / Inputs | LoadOp | StoreOp | Resolves MSAA |
|------------|-----------|----------------------|--------|---------|---------------|
| 01 | Depth Pre-Pass | Depth (D32_FLOAT) | Clear | Store | No |
| 02 | Cascaded Shadow Maps | Depth Atlas (D32_FLOAT) | Clear | Store | No |
| 03 | Clustered Light Culling | Compute SSBO (Grid + Light Indices) | - | - | - |
| 04 | Forward Opaque Pass | Color (RGBA16F), Depth (D32F ReadOnly) | Clear | Store | Yes |
| 05 | Transparent Forward | Color (RGBA16F), Depth (D32F ReadOnly) | Load | Store | No |
| 06 | Post-Process (Tonemap/TAA) | Target Swapchain (BGRA8_SRGB) | Discard| Store | No |

---

## 3. Geometry & Memory Footprint Breakdown
- **Static Vertex Buffers (VRAM)**: [MB]
- **Dynamic Ring Buffers (Uniforms/Transforms)**: [MB per frame $\times$ buffer count]
- **G-Buffer / Render Target Footprint**:
  - RT0: [Format, Resolution, MB]
  - RT1: [Format, Resolution, MB]
  - Depth Atlas: [Format, Resolution, MB]
  - **Total Transient Bandwidth**: [GB/s @ Target FPS]
- **Average Visible Triangles / Draw Calls**:
  - Total Scene Meshes: [count]
  - Frustum Culled Meshes: [count] (cull efficiency: [%])
  - Total Batched Draw Calls: [count]

---

## 4. Physically Based Shading & Lighting Verification
- [ ] Specular BRDF: Cook-Torrance (GGX + Smith + Fresnel-Schlick)
- [ ] Energy Conservation Confirmed: $k_d + k_s \le 1.0$ across all roughness/metallic ranges
- [ ] Dielectric $F_0 = 0.04$ / Metallic tinting correct
- [ ] Roughness clamp to $\alpha \ge 0.001$ prevents GGX division by zero
- [ ] Cascade shadow map texel snapping verified (no crawling artifacts on camera translation)

---

## 5. Performance Metrics & Frame Timing
- **CPU Render Thread Time**: [ms]
- **GPU Frame Time**: [ms]
  - Depth Pre-Pass: [ms]
  - Shadow Passes (All Cascades): [ms]
  - Light Culling Compute Pass: [ms]
  - Main Opaque Shading Pass: [ms]
  - Post-Processing & Tonemapping: [ms]
- **Memory Allocation Hot-Path Audit**: 0 bytes allocated during render loop.

---

## 6. Optimization Decisions & Deviations
- **Architectural Rationale**: [Explain why Forward+ vs Deferred was chosen given hardware targets]
- **Bandwidth Reduction Strategies**: [e.g., Octahedral normal packing, lazy allocation of transient buffers]
