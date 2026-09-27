#!/usr/bin/env sh
# Verification script for WGSL WebGPU Soundness
set -eu

echo "=== [lang-wgsl] Starting WGSL Shader Soundness Routine ==="

SHADERS=$(find . -name "*.wgsl" -not -path "*/node_modules/*" 2>/dev/null || true)
if [ -z "$SHADERS" ]; then
    echo "INFO: No .wgsl shader files found in current directory."
    exit 0
fi

echo "Found WGSL shader files:"
echo "$SHADERS"

# 1. Check for vec3 padding hazard
echo "[1/3] Auditing struct definitions for unpadded vec3 fields..."
for s in $SHADERS; do
    if grep -n "vec3<" "$s" | grep -v "//" | grep -v "@align"; then
        echo "WARNING: Found unannotated 'vec3<' in $s. Verify explicit 16-byte alignment or convert to vec4."
    fi
done

# 2. Check for missing workgroupBarrier when using workgroup variables
echo "[2/3] Checking compute synchronization invariants..."
for s in $SHADERS; do
    if grep -q "var<workgroup>" "$s"; then
        if ! grep -q "workgroupBarrier()" "$s"; then
            echo "WARNING: $s declares workgroup memory but no workgroupBarrier() was detected!"
        fi
    fi
done

# 3. Naga compiler validation if available
echo "[3/3] Running static compiler validation..."
if command -v naga >/dev/null 2>&1; then
    for s in $SHADERS; do
        echo "Validating $s with Naga..."
        naga "$s"
    done
    echo "Naga validation: PASS"
else
    echo "INFO: 'naga' CLI not installed in PATH; skipping binary translation check."
fi

echo "=== [lang-wgsl] WGSL Verification Completed ==="
exit 0
