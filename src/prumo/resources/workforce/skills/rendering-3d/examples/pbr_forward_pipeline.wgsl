// Cook-Torrance Microfacet PBR Forward Pass (WGSL)
// Conforming to W3C WebGPU Shading Language & Energy Conservation Invariants

struct CameraUniforms {
    view_proj: mat4x4<f32>,
    camera_pos: vec3<f32>,
    _pad0: f32,
};

struct DirectionalLight {
    direction: vec3<f32>,
    _pad0: f32,
    color: vec3<f32>,
    intensity: f32,
};

@group(0) @binding(0) var<uniform> camera: CameraUniforms;
@group(0) @binding(1) var<uniform> sun_light: DirectionalLight;

@group(1) @binding(0) var albedo_texture: texture_2d<f32>;
@group(1) @binding(1) var normal_texture: texture_2d<f32>;
@group(1) @binding(2) var metallic_roughness_texture: texture_2d<f32>;
@group(1) @binding(3) var texture_sampler: sampler;

struct VertexInput {
    @location(0) position: vec3<f32>,
    @location(1) normal: vec3<f32>,
    @location(2) uv: vec2<f32>,
    @location(3) tangent: vec4<f32>,
};

struct VertexOutput {
    @builtin(position) clip_position: vec4<f32>,
    @location(0) world_pos: vec3<f32>,
    @location(1) uv: vec2<f32>,
    @location(2) normal: vec3<f32>,
    @location(3) tangent: vec3<f32>,
    @location(4) bitangent: vec3<f32>,
};

@vertex
fn vs_main(model: VertexInput) -> VertexOutput {
    var out: VertexOutput;
    out.world_pos = model.position;
    out.clip_position = camera.view_proj * vec4<f32>(model.position, 1.0);
    out.uv = model.uv;
    
    let normal = normalize(model.normal);
    let tangent = normalize(model.tangent.xyz);
    let bitangent = cross(normal, tangent) * model.tangent.w;
    
    out.normal = normal;
    out.tangent = tangent;
    out.bitangent = bitangent;
    return out;
}

const PI: f32 = 3.14159265358979323846;

// Trowbridge-Reitz GGX Normal Distribution Function
fn distribution_ggx(n_dot_h: f32, roughness: f32) -> f32 {
    let a = roughness * roughness;
    let a2 = a * a;
    let n_dot_h2 = n_dot_h * n_dot_h;
    let denom = (n_dot_h2 * (a2 - 1.0) + 1.0);
    return a2 / (PI * denom * denom);
}

// Fresnel-Schlick Approximation
fn fresnel_schlick(cos_theta: f32, f0: vec3<f32>) -> vec3<f32> {
    return f0 + (vec3<f32>(1.0) - f0) * pow(clamp(1.0 - cos_theta, 0.0, 1.0), 5.0);
}

// Smith Joint Masking-Shadowing Function (Height-Correlated Visibility)
fn visibility_smith(n_dot_v: f32, n_dot_l: f32, roughness: f32) -> f32 {
    let a = roughness * roughness;
    let a2 = a * a;
    let g_v = n_dot_l * sqrt(n_dot_v * n_dot_v * (1.0 - a2) + a2);
    let g_l = n_dot_v * sqrt(n_dot_l * n_dot_l * (1.0 - a2) + a2);
    return 0.5 / max(g_v + g_l, 0.0001);
}

@fragment
fn fs_main(in: VertexOutput) -> @location(0) vec4<f32> {
    // 1. Re-normalize interpolated vectors
    let t = normalize(in.tangent);
    let b = normalize(in.bitangent);
    let n_geom = normalize(in.normal);
    let tbn = mat3x3<f32>(t, b, n_geom);

    // 2. Normal mapping
    let normal_map_sample = textureSample(normal_texture, texture_sampler, in.uv).xyz * 2.0 - 1.0;
    let n = normalize(tbn * normal_map_sample);

    let v = normalize(camera.camera_pos - in.world_pos);
    let l = normalize(-sun_light.direction);
    let h = normalize(v + l);

    // 3. Sample Material maps
    let albedo = textureSample(albedo_texture, texture_sampler, in.uv).rgb;
    let mr = textureSample(metallic_roughness_texture, texture_sampler, in.uv);
    let metallic = mr.b;
    // Perceptual roughness squared, clamped to avoid GGX singularity
    let roughness = max(mr.g, 0.04);

    let n_dot_v = max(dot(n, v), 0.0001);
    let n_dot_l = max(dot(n, l), 0.0001);
    let n_dot_h = max(dot(n, h), 0.0);
    let v_dot_h = max(dot(v, h), 0.0);

    // 4. Base Reflectivity F0 (Dielectric 0.04 vs Metallic Albedo)
    let f0 = mix(vec3<f32>(0.04), albedo, metallic);

    // 5. Specular Evaluation
    let d = distribution_ggx(n_dot_h, roughness);
    let f = fresnel_schlick(v_dot_h, f0);
    let vis = visibility_smith(n_dot_v, n_dot_l, roughness);
    let specular = d * f * vis;

    // 6. Energy Conservation (kd + ks <= 1.0)
    let k_s = f;
    var k_d = (vec3<f32>(1.0) - k_s) * (1.0 - metallic);

    // 7. Lambertian Diffuse
    let diffuse = k_d * (albedo / PI);

    // 8. Accumulate Radiance
    let radiance = sun_light.color * sun_light.intensity;
    let direct_lighting = (diffuse + specular) * radiance * n_dot_l;

    // Ambient baseline (0.03 * albedo)
    let ambient = vec3<f32>(0.03) * albedo;
    let color = ambient + direct_lighting;

    return vec4<f32>(color, 1.0);
}
