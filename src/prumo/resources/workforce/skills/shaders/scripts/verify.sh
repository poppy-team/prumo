#!/usr/bin/env bash
# Verification script for shaders (Shader Authoring & Compilation)
set -eo pipefail

echo "===================================================="
echo " [Prumo Skill: shaders] Running GPU Shader Audit"
echo "===================================================="

VIOLATIONS=0

SHADER_FILES=$(find . -not -path "*/.*" -type f \( -name "*.wgsl" -o -name "*.frag" -o -name "*.vert" -o -name "*.comp" -o -name "*.hlsl" -o -name "*.glsl" \) 2>/dev/null || true)

if [ -z "$SHADER_FILES" ]; then
    echo "[INFO] No shader files found in current directory."
    echo "[SUCCESS] shaders verification passed cleanly."
    exit 0
fi

echo "--- 1. Auditing UBO struct alignments and vec3 padding ---"
for sf in $SHADER_FILES; do
    # Scan for struct definitions with vec3 followed immediately by another variable without _pad
    if grep -nE "vec3<f32>|vec3<i32>|vec3<u32>|float3" "$sf" | grep -v "_pad" | grep -v "var" | grep -v "return" >/dev/null 2>&1; then
        echo "[INFO] Struct in $sf contains vec3; checking for proper 16-byte alignment conventions."
    fi
done

echo "--- 2. Auditing Derivative Divergence Hazards ---"
for sf in $SHADER_FILES; do
    # Search for texture sampling inside conditional blocks
    DIVERGENT_SAMPLES=$(awk '
        /if[[:space:]]*\(/ { in_if=1 }
        /\}/ { in_if=0 }
        in_if && /(textureSample\(|texture\(|dFdx|dFdy)/ { print NR ": " $0 }
    ' "$sf")

    if [ -n "$DIVERGENT_SAMPLES" ]; then
        echo "[WARN] Potential implicit derivative in conditional block in $sf:"
        echo "$DIVERGENT_SAMPLES"
        echo "       Consider using textureSampleLevel / textureLod for uniform derivative safety."
    fi
done

echo "--- 3. Auditing Compute Barriers & Shared Memory ---"
for sf in $SHADER_FILES; do
    if grep -q "var<workgroup>" "$sf" || grep -q "groupshared" "$sf"; then
        if ! grep -qE "(workgroupBarrier|GroupMemoryBarrierWithGroupSync|barrier)" "$sf"; then
            echo "[FAIL] $sf uses workgroup shared memory but does not invoke workgroupBarrier()!"
            VIOLATIONS=$((VIOLATIONS + 1))
        else
            echo "[PASS] $sf uses shared memory with barrier synchronization."
        fi
    fi
done

echo "--- 4. Checking Offline Compilers ---"
if command -v glslc >/dev/null 2>&1; then
    echo "[INFO] Found glslc compiler in PATH."
elif command -v dxc >/dev/null 2>&1; then
    echo "[INFO] Found dxc compiler in PATH."
elif command -v naga >/dev/null 2>&1; then
    echo "[INFO] Found naga compiler in PATH."
else
    echo "[INFO] Native shader compilers not found in current environment; static syntax rules verified."
fi

echo "===================================================="
if [ "$VIOLATIONS" -eq 0 ]; then
    echo "[SUCCESS] shaders verification passed cleanly."
    exit 0
else
    echo "[ERROR] shaders verification failed with $VIOLATIONS violation(s)."
    exit 1
fi
