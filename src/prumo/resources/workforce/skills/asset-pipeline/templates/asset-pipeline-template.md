# Asset Pipeline Report — Season 2 Environment Pack

## 1. Metadata
- **Skill**: asset-pipeline (Asset Pipeline & Cooking)
- **Date**: 2026-09-14
- **Author / Agent**: engine-tools-team
- **Target Goal / Phase**: GOAL-118 season2-environment-cook

## 2. Executive Summary
Cooked the 1,240-asset Season 2 environment pack (textures, static meshes, ambient audio banks) through the deterministic cooker. All outputs content-hashed and within the tier-2 GPU budget of 512 MB. Fixed one nondeterminism source (embedded timestamps in mesh optimizer output) and one over-budget texture set (4K normals downsampled to 2K with ASTC 6x6). Hot-reload swap verified hitch-free.

## 3. Inputs & Scope
- **Inputs Evaluated**: `assets/season2/**` sources (PNG 4K, FBX, WAV 48 kHz), tier-2 GPU caps (Adreno 730 class), cooker v2.7.1 settings
- **Artifacts Modified**: `cooked/season2/**` (1,240 outputs), `tools/cooker/mesh_opt.py` (timestamp strip), `assets/season2/cliffs/normals/*.png` (resized)

## 4. Key Findings & Implementation Details
- **Determinism**: Two clean cooks produce identical SHA-256 manifest (`manifest.sha256` matches byte-for-byte). Root cause of prior drift: `mtime` embedded by mesh optimizer; now zeroed at write.
- **Compression**: Color textures → ASTC 6x6 + full mip chains (avg 5.8:1); normals → ASTC 5x5 2K (was 4K, saved 96 MB); ambient beds → Opus 96 kbps (was WAV, saved 310 MB).
- **Budgets**: Cooked total 468 MB ≤ 512 MB ceiling. Largest single asset: `canyon_albedo` 22 MB. Cook-time rejection rule added: any texture cooking above 32 MB fails with remediation hint.
- **Hot reload**: Source edit → re-cook off hot path (avg 1.8 s) → atomic handle swap; frame-time p99 delta +0.4 ms during swap on test device.
- **Streaming**: Chunked LOD tiers with 8 MB/frame bandwidth cap; scripted flythrough shows zero missing-LOD frames; low-LOD fallback resident for all 1,240 assets.

## 5. Verification & Evidence
- **Evidence Type**: test, benchmark
- **Test Results**: Passed — determinism diff clean; 50 hot-reload swaps, 0 hitches > 2 ms; flythrough 3/3 runs hitch-free
- **Static Analysis Status**: Pass — cooker lint clean; manifest schema validated

## 6. Next Steps & Handoff
- Hand cooked manifest to QA perf pass (GOAL-119); owner: engine-tools-team.
- Schedule tier-1 GPU downscale review for 4K hero assets; owner: art-lead.
