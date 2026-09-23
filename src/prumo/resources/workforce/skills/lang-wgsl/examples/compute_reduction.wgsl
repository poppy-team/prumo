// Parallel Sum Reduction in WebGPU Shading Language (WGSL)
// Demonstrates explicit 16-byte aligned structs, workgroupBarrier synchronization,
// workgroup shared memory usage, and global-to-local memory indexing.

struct ComputeParams {
    @align(16) base_multiplier: vec4<f32>,
    @align(16) total_elements: u32,
    _padding0: u32,
    _padding1: u32,
    _padding2: u32,
};

@group(0) @binding(0) var<uniform> params: ComputeParams;
@group(0) @binding(1) var<storage, read> input_buffer: array<f32>;
@group(0) @binding(2) var<storage, read_write> output_reduced: array<f32>;

// Workgroup shared memory (256 floats = 1024 bytes, well within 16KB hardware limits)
const WORKGROUP_SIZE: u32 = 256u;
var<workgroup> s_scratch: array<f32, WORKGROUP_SIZE>;

@compute @workgroup_size(256, 1, 1)
fn main(
    @builtin(global_invocation_id) global_id: vec3<u32>,
    @builtin(local_invocation_id) local_id: vec3<u32>,
    @builtin(workgroup_id) group_id: vec3<u32>
) {
    let tid = local_id.x;
    let gid = global_id.x;

    // 1. Cooperative load into workgroup cache with boundary check
    if (gid < params.total_elements) {
        s_scratch[tid] = input_buffer[gid] * params.base_multiplier.x;
    } else {
        s_scratch[tid] = 0.0;
    }

    // 2. Barrier ensures all 256 threads finished their initial load into shared memory
    workgroupBarrier();

    // 3. Tree-based parallel reduction with logarithmic depth (log2(256) = 8 steps)
    for (var stride: u32 = WORKGROUP_SIZE / 2u; stride > 0u; stride = stride >> 1u) {
        if (tid < stride) {
            s_scratch[tid] += s_scratch[tid + stride];
        }
        // Essential barrier after every reduction step to synchronize memory visibility
        workgroupBarrier();
    }

    // 4. Thread 0 of each workgroup writes the reduced sum for this block
    if (tid == 0u) {
        output_reduced[group_id.x] = s_scratch[0];
    }
}