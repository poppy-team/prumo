---
name: rendering-2d
description: Dynamic sprite batching, texture atlas generation, ortho projection, 2D frustum culling, shader pipelines, and draw call minimization
---
# 2D Rendering & Graphics Pipeline

## 1. Dynamic Sprite Batching
Group consecutive 2D draw calls sharing the same texture and shader into a single contiguous vertex buffer. Minimize state changes to keep draw calls under 20 per frame.

## 2. Texture Atlas Architecture
Pack multiple sprite assets into unified texture atlases (power-of-two dimensions) with padding to eliminate texture bleed. Store UV coordinates in a fast lookup table.

## 3. Orthographic Camera & Screen Mapping
Implement a 2D orthographic camera supporting zoom, pan, rotation, and pixel-perfect integer scaling to eliminate sub-pixel artifact jitter.

## 4. 2D Frustum & Viewport Culling
Perform bounding box (AABB) intersection tests against the camera view rect to cull off-screen sprites before pushing vertex data to the GPU.

## 5. Z-Index & Sorting Layers
Sort sprites deterministically by sorting layer, Z-index, and material state to ensure correct alpha blending and minimize GPU pipeline state changes.

## 6. Alpha Blending & Blend Modes
Configure standard blend modes (premultiplied alpha, additive, multiply). Prefer premultiplied alpha to avoid dark fringing around transparent sprite edges.

## 7. 2D Shader Effects & Post-Processing
Implement customized 2D shaders for bloom, outlines, screen-shake distortion, and palette swapping using compact fragment shaders.

## 8. Tilemap Chunking & Optimization
Render large tilemaps in static chunks (e.g. 16x16 or 32x32 tiles) baked into vertex buffers, recalculating geometry only when tile modifications occur.

## 9. Particle System Batching
Render 2D particles using instanced rendering or dynamic point-sprite batches with CPU or compute-shader particle physics.

## 10. Frame Pacing & VSync Synchronization
Synchronize swapchain presentation with monitor refresh rates to avoid screen tearing while maintaining minimal input latency.
