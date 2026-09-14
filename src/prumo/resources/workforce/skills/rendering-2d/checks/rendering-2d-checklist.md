# 2D Rendering & Graphics Pipeline Checklist

- [ ] Sprites batched into single draw calls sharing texture and shader state
- [ ] Texture atlases generated with edge padding to prevent UV bleeding
- [ ] Orthographic camera supports smooth zoom and pixel-perfect scaling
- [ ] Off-screen sprites culled via AABB view frustum checks
- [ ] Z-ordering and sorting layers deterministic with premultiplied alpha
- [ ] Tilemaps grouped into cached chunk meshes rather than per-tile draws
- [ ] Draw call count and frame rendering time tracked in real-time
