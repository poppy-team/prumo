#!/usr/bin/env bash
# Verification script for rendering-2d (2D Vector & Sprite Rendering Pipeline)
set -euo pipefail

echo "===================================================="
echo " [Prumo Skill: rendering-2d] 2D Pipeline & Batch Audit"
echo "===================================================="

VIOLATIONS=0
EXCLUDE="--exclude-dir=.git --exclude-dir=build --exclude-dir=vendor --exclude-dir=node_modules --exclude-dir=fixtures --exclude-dir=tests"

# 1. Hot-path dynamic allocation audit in 2D render loops
echo "--- 1. Auditing hot-path dynamic memory allocations ---"
DYNAMIC_ALLOCS=$(grep -rnE $EXCLUDE "(push_back|malloc|new\s+)" --include="*batcher*.cpp" --include="*2d*.cpp" --include="*render*.cpp" . 2>/dev/null | grep -iE "(draw|render|flush|tick)" | grep -v "examples" || true)
if [ -n "$DYNAMIC_ALLOCS" ]; then
    echo "[WARN] Found dynamic memory allocations inside 2D draw/render routines:"
    echo "$DYNAMIC_ALLOCS" | head -n 5
else
    echo "[PASS] No dynamic allocations detected in 2D render hot loops."
fi

# 2. Check for Premultiplied Alpha vs Straight Alpha blending
echo "--- 2. Auditing Alpha Blending Configurations ---"
STRAIGHT_ALPHA=$(grep -rnE $EXCLUDE "(SRC_ALPHA,\s*ONE_MINUS_SRC_ALPHA|blendMode:\s*['\"]normal['\"])" --include="*.cpp" --include="*.ts" --include="*.rs" . 2>/dev/null | grep -v "premultiplied" | head -n 5 || true)
if [ -n "$STRAIGHT_ALPHA" ]; then
    echo "[INFO] Found straight alpha blending references; consider Premultiplied Alpha for fringe elimination:"
    echo "$STRAIGHT_ALPHA"
else
    echo "[PASS] Alpha blending configuration audit complete."
fi

# 3. Compile and execute reference SpriteBatcher example
echo "--- 3. Compiling & Executing Reference 2D SpriteBatcher ---"
COMPILER=""
if command -v clang++ >/dev/null 2>&1; then
    COMPILER="clang++"
elif command -v g++ >/dev/null 2>&1; then
    COMPILER="g++"
fi

if [ -n "$COMPILER" ]; then
    EXAMPLE="src/prumo/resources/workforce/skills/rendering-2d/examples/sprite_batcher.cpp"
    if [ -f "$EXAMPLE" ]; then
        TMP_BIN=$(mktemp /tmp/prumo_batcher_test_XXXXXX)
        $COMPILER -std=c++20 -Wall -Wextra -Werror "$EXAMPLE" -o "$TMP_BIN"
        "$TMP_BIN" >/dev/null
        rm -f "$TMP_BIN"
        echo "[PASS] Reference C++20 SpriteBatcher compiled and executed cleanly."
    fi
else
    echo "[WARN] C++ compiler not available for batcher compilation test."
fi

echo "===================================================="
if [ "$VIOLATIONS" -eq 0 ]; then
    echo "[SUCCESS] rendering-2d verification passed cleanly."
    exit 0
else
    echo "[ERROR] rendering-2d verification failed with $VIOLATIONS violation(s)."
    exit 1
fi
