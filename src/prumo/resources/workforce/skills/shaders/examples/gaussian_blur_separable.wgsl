// High-Performance Separable 1D Gaussian Blur Compute Shader (WGSL)
// Demonstrates workgroup shared memory caching, coalesced global reads,
// cooperative halo loading, workgroup barriers, and branchless convolution.

struct BlurUniforms {
    direction: vec2<i32>,   // (1, 0) for horizontal pass, (0, 1) for vertical pass
    radius: i32,            // Filter kernel radius (e.g. 4)
    dimensions: vec2<i32>,  // Image width, height
    _pad0: i32,             // Align to 16-byte boundary
};

@group(0) @binding(0) var<uniform> config: BlurUniforms;
@group(0) @binding(1) var input_texture: texture_2d<f32>;
@group(0) @binding(2) var output_texture: texture_storage_2d<rgba16float, write>;

// Gaussian Weights precomputed for radius 4 (stddev ~ 2.0)
const KERNEL: array<f32, 5> = array<f32, 5>(
    0.2270270270, // center (offset 0)
    0.1945945946, // offset +/- 1
    0.1216216216, // offset +/- 2
    0.0540540541, // offset +/- 3
    0.0162162162  // offset +/- 4
);

const WORKGROUP_SIZE_X: u32 = 128u;
const MAX_RADIUS: u32 = 4u;
// Tile capacity: 128 workgroup invocations + 2 * 4 halo margins = 136 elements
const TILE_SIZE: u32 = WORKGROUP_SIZE_X + 2u * MAX_RADIUS;

// Workgroup Shared Memory Tile
var<workgroup> shared_tile: array<vec4<f32>, TILE_SIZE>;

@compute @workgroup_size(128, 1, 1)
fn cs_blur_separable(
    @builtin(global_invocation_id) global_id: vec3<u32>,
    @builtin(local_invocation_id) local_id: vec3<u32>,
    @builtin(workgroup_id) group_id: vec3<u32>
) {
    let tid = local_id.x;
    let base_coord = vec2<i32>(global_id.xy);

    // 1. Cooperative Shared Tile Loading
    // Center load: each thread loads its primary pixel into shared memory
    let center_coord = clamp(
        base_coord,
        vec2<i32>(0, 0),
        config.dimensions - vec2<i32>(1, 1)
    );
    shared_tile[tid + MAX_RADIUS] = textureLoad(input_texture, center_coord, 0);

    // Left halo cooperative load: first 4 threads load left boundary pixels
    if (tid < MAX_RADIUS) {
        let left_coord = clamp(
            base_coord - config.direction * i32(MAX_RADIUS),
            vec2<i32>(0, 0),
            config.dimensions - vec2<i32>(1, 1)
        );
        shared_tile[tid] = textureLoad(input_texture, left_coord, 0);
    }

    // Right halo cooperative load: last 4 threads load right boundary pixels
    if (tid >= WORKGROUP_SIZE_X - MAX_RADIUS) {
        let right_coord = clamp(
            base_coord + config.direction * i32(MAX_RADIUS),
            vec2<i32>(0, 0),
            config.dimensions - vec2<i32>(1, 1)
        );
        shared_tile[tid + 2u * MAX_RADIUS] = textureLoad(input_texture, right_coord, 0);
    }

    // 2. Hardware Execution Barrier: Guarantee all shared writes are visible before convolution
    workgroupBarrier();

    // 3. Early out if thread is outside target image dimensions
    if (base_coord.x >= config.dimensions.x || base_coord.y >= config.dimensions.y) {
        return;
    }

    // 4. Branchless Convolution from fast on-chip shared SRAM
    let tile_center_idx = tid + MAX_RADIUS;
    var accumulated_color: vec4<f32> = shared_tile[tile_center_idx] * KERNEL[0];

    for (var r: u32 = 1u; r <= MAX_RADIUS; r = r + 1u) {
        let sample_pos = shared_tile[tile_center_idx + r];
        let sample_neg = shared_tile[tile_center_idx - r];
        accumulated_color += (sample_pos + sample_neg) * KERNEL[r];
    }

    // 5. Coalesced Global Write to Storage Texture
    textureStore(output_texture, base_coord, accumulated_color);
}
