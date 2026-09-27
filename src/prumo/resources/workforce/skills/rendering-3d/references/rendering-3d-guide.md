# Modern 3D Graphics Pipeline — Technical Engineering Reference

## 1. Pipeline Architecture Taxonomy

```
+-----------------------------------------------------------------------------------+
| Modern Real-Time 3D Rendering Architectures                                       |
+-----------------------------------------------------------------------------------+
| Architecture | Geometry Pass Complexity | Light Pass Complexity | Memory Bandwidth|
+--------------+--------------------------+-----------------------+-----------------+
| Forward      | O(DrawCalls)             | O(Objects * Lights)   | Minimal (Tile)  |
| Forward+     | Depth Prepass + Main     | O(Tiles * CulledLights| Low-Moderate    |
| Deferred     | G-Buffer (Dense Write)   | O(Pixels * Lights)    | High (VRAM BW)  |
| Clustered    | Depth Prepass + Cluster  | O(Clusters * Lights)  | Low-Moderate    |
+-----------------------------------------------------------------------------------+
```

### Forward+ vs Deferred Shading
1. **Deferred Shading**:
   - Encodes material properties into a Multi-Render-Target (MRT) G-Buffer during a geometry pass.
   - Standard G-Buffer Layout (Typical 64-bit to 128-bit per pixel):
     - **RT0 (RGBA8_UNORM)**: `Albedo.rgb`, `PerceptualRoughness.a`
     - **RT1 (RGBA16_FLOAT)**: `OctahedralNormal.rg`, `Metallic.b`, `AO.a`
     - **RT2 (RG16_FLOAT)**: `ScreenMotionVectors.xy` (for TAA and motion blur)
     - **Depth Buffer (D32_FLOAT)**: Depth used for position reconstruction via inverse View-Projection matrix:
       $$\mathbf{P}_{\text{world}} = \text{Unproject}(\mathbf{x}_{\text{ndc}}, \mathbf{y}_{\text{ndc}}, \mathbf{z}_{\text{depth}})$$
   - *Limitation*: High memory bandwidth on mobile/tiled architectures; challenging native MSAA; cannot handle transparency in the deferred pass.

2. **Forward+ / Clustered Forward Shading**:
   - Retains a standard forward color pass while decoupling light counts via spatial indexing.
   - Screen space is divided into a 3D grid ($X \times Y \times Z$ clusters).
   - $Z$-slices are logarithmic (view-depth):
     $$S_z = \left\lfloor \frac{\log(-z_{\text{view}} / z_{\text{near}})}{\log(z_{\text{far}} / z_{\text{near}})} \times N_{\text{slices}} \right\rfloor$$
   - A Compute Shader executes light culling against each cluster AABB, storing active light indices in a flat SSBO buffer.
   - The fragment shader samples only the subset of lights intersecting its active cluster, achieving $O(\text{CulledLights})$ scaling with native MSAA and multiple BRDF support.

---

## 2. Physically Based Rendering (PBR) — Microfacet Cook-Torrance

### 2.1 The Specular BRDF
$$f_r(\mathbf{l}, \mathbf{v}) = k_d \frac{c}{\pi} + \frac{D(\mathbf{h}) F(\mathbf{v}, \mathbf{h}) G(\mathbf{l}, \mathbf{v}, \mathbf{h})}{4 (\mathbf{n} \cdot \mathbf{l}) (\mathbf{n} \cdot \mathbf{v})}$$

Where:
- $\mathbf{n}$ is the surface normal, $\mathbf{v}$ is the view vector, $\mathbf{l}$ is the light vector.
- $\mathbf{h} = \frac{\mathbf{l} + \mathbf{v}}{\|\mathbf{l} + \mathbf{v}\|}$ is the half-vector.
- $\alpha = \text{roughness}^2$ (Disney perceptual roughness remapping).

### 2.2 Normal Distribution Function (NDF): Trowbridge-Reitz GGX
Models microfacet normal alignment with half-vector $\mathbf{h}$:
$$D(\mathbf{h}) = \frac{\alpha^2}{\pi \left( (\mathbf{n} \cdot \mathbf{h})^2 (\alpha^2 - 1) + 1 \right)^2}$$

### 2.3 Fresnel Reflectance: Fresnel-Schlick Approximation
Models reflection percentage as a function of incident angle:
$$F(\mathbf{v}, \mathbf{h}) = F_0 + (1 - F_0) (1 - (\mathbf{v} \cdot \mathbf{h}))^5$$
- For dielectrics: $F_0 = 0.04$
- For metals: $F_0 = \text{albedo}$
- Final $F_0 = \text{mix}(0.04, \text{albedo}, \text{metallic})$

### 2.4 Geometric Shadowing: Smith Joint Masking-Shadowing Function
Height-correlated Smith geometric attenuation, canceling out the denominator $4(\mathbf{n}\cdot\mathbf{l})(\mathbf{n}\cdot\mathbf{v})$:
$$V(\mathbf{l}, \mathbf{v}) = \frac{G(\mathbf{l}, \mathbf{v}, \mathbf{h})}{4 (\mathbf{n} \cdot \mathbf{l}) (\mathbf{n} \cdot \mathbf{v})} = \frac{0.5}{(\mathbf{n} \cdot \mathbf{l}) \sqrt{(\mathbf{n} \cdot \mathbf{v})^2(1 - \alpha^2) + \alpha^2} + (\mathbf{n} \cdot \mathbf{v}) \sqrt{(\mathbf{n} \cdot \mathbf{l})^2(1 - \alpha^2) + \alpha^2}}$$

### 2.5 Energy Conservation
$$\mathbf{k}_s = F(\mathbf{v}, \mathbf{h})$$
$$\mathbf{k}_d = (1 - \mathbf{k}_s) \times (1 - \text{metallic})$$

---

## 3. Cascaded Shadow Maps (CSM) — Precision & Anti-Artifact Math

### 3.1 Practical Split Scheme
Combines logarithmic and linear partitioning:
$$z_i = \lambda z_{\text{near}} \left(\frac{z_{\text{far}}}{z_{\text{near}}}\right)^{\frac{i}{m}} + (1 - \lambda)\left(z_{\text{near}} + \frac{i}{m}(z_{\text{far}} - z_{\text{near}})\right)$$
Typically $\lambda \in [0.5, 0.85]$.

### 3.2 Texel Snapping Algorithm
To prevent shadow shimmering when the camera translates:
1. Compute the bounding sphere of the view-frustum slice: Center $\mathbf{C}$, Radius $R$.
2. Create light view-projection matrix using $\mathbf{C}$ as target with extent $[-R, +R]$.
3. Compute texel size in world coordinates:
   $$\text{texelSize} = \frac{2 \times R}{\text{shadowMapResolution}}$$
4. Transform center $\mathbf{C}$ to light view space: $\mathbf{C}_{\text{light}} = \mathbf{V}_{\text{light}} \times \mathbf{C}$.
5. Snap light-space coordinates to integer multiples of texel size:
   $$\mathbf{C}'_{x} = \lfloor \mathbf{C}_{\text{light}, x} / \text{texelSize} \rfloor \times \text{texelSize}$$
   $$\mathbf{C}'_{y} = \lfloor \mathbf{C}_{\text{light}, y} / \text{texelSize} \rfloor \times \text{texelSize}$$
6. Reconstruct view matrix using snapped center $\mathbf{C}'$.

---

## 4. Vulkan / WebGPU Synchronization Model

```
Frame N:     [CPU: Record Cmds] ---> [GPU: Queue Submit] ---> [GPU: Render Passes] ---> [Present]
Frame N+1:   [CPU: Wait Fence]  ---> [CPU: Record Cmds] ---> [GPU: Queue Submit]
             (Triple-buffered Ring Buffers guarantee zero CPU stalls)
```

1. **Double/Triple Buffering**:
   - `VkFence` or WebGPU CPU timeline ensures CPU does not overwrite uniform buffer ring slices currently read by GPU.
2. **Explicit Pass Dependencies**:
   - Depth Pre-Pass $\xrightarrow{\text{barrier}}$ Shadow Map $\xrightarrow{\text{barrier}}$ G-Buffer / Forward Lighting Pass $\xrightarrow{\text{barrier}}$ Post-Processing.
3. **Transient Memory Allocation**:
   - G-Buffer attachments with `VK_IMAGE_USAGE_TRANSIENT_ATTACHMENT_BIT` and `VK_MEMORY_PROPERTY_LAZILY_ALLOCATED_BIT` allow mobile tile-based GPUs (Apple Silicon, ARM Mali, Adreno) to keep G-Buffer completely inside fast on-chip tile memory without writing back to external DRAM.
