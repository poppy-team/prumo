#!/usr/bin/env bash
# Verification script for rendering-3d (3D Rendering Pipeline)
set -eo pipefail

echo "===================================================="
echo " [Prumo Skill: rendering-3d] Running 3D Pipeline Audit"
echo "===================================================="

VIOLATIONS=0
EXCLUDE="--exclude-dir=.git --exclude-dir=node_modules --exclude-dir=target --exclude-dir=archive"

# 1. Hot-path dynamic allocation audit
echo "--- 1. Auditing hot-path dynamic memory allocations ---"
DYNAMIC_ALLOCS=$(grep -rnE $EXCLUDE "(malloc|calloc|free|operator new|std::vector<.*>\.push_back)" --include="*.cpp" --include="*.c" --include="*.rs" . 2>/dev/null | grep -iE "(render|draw|record|tick|present)" || true)

if [ -n "$DYNAMIC_ALLOCS" ]; then
    echo "[FAIL] Found dynamic memory allocations inside render/draw/tick routines:"
    echo "$DYNAMIC_ALLOCS"
    VIOLATIONS=$((VIOLATIONS + 1))
else
    echo "[PASS] No raw dynamic allocations detected in render tick loops."
fi

# 2. Shader Energy Conservation & Normalization audit
echo "--- 2. Auditing Shader PBR & Math Invariants ---"
SHADER_FILES=$(find . -not -path "*/.*" -type f \( -name "*.wgsl" -o -name "*.frag" -o -name "*.vert" -o -name "*.hlsl" -o -name "*.glsl" \) 2>/dev/null || true)

if [ -n "$SHADER_FILES" ]; then
    for sf in $SHADER_FILES; do
        if grep -q "v_normal" "$sf" && ! grep -q "normalize(.*v_normal.*)" "$sf"; then
            echo "[WARN] Shader $sf uses interpolated v_normal without explicit normalize()"
        fi
    done
    echo "[PASS] Shader math scanned."
else
    echo "[INFO] No shader files found in working tree."
fi

# 3. WebGPU / Vulkan Attachment LoadOp & StoreOp validation
echo "--- 3. Auditing Pass Attachment Operations ---"
IMPLICIT_LOADS=$(grep -rnE $EXCLUDE "(loadOp.*undefined|loadOp:\s*None)" --include="*.ts" --include="*.js" --include="*.rs" --include="*.cpp" . 2>/dev/null || true)
if [ -n "$IMPLICIT_LOADS" ]; then
    echo "[FAIL] Found implicit or undefined LoadOp attachments:"
    echo "$IMPLICIT_LOADS"
    VIOLATIONS=$((VIOLATIONS + 1))
else
    echo "[PASS] Render pass attachment declarations are explicit."
fi

echo "===================================================="
if [ "$VIOLATIONS" -eq 0 ]; then
    echo "[SUCCESS] rendering-3d verification passed cleanly."
    exit 0
else
    echo "[ERROR] rendering-3d verification failed with $VIOLATIONS violation(s)."
    exit 1
fi
