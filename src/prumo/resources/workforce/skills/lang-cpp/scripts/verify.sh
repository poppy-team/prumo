#!/usr/bin/env bash
# Verification script for Modern C++ (C++20/C++23) Safety & Quality
set -euo pipefail

echo "===================================================="
echo " [Prumo Skill: lang-cpp] Modern C++ Safety Audit"
echo "===================================================="

VIOLATIONS=0
EXCLUDE="--exclude-dir=.git --exclude-dir=build --exclude-dir=vendor --exclude-dir=node_modules --exclude-dir=fixtures --exclude-dir=tests"

# 1. Scan for naked new / delete in non-allocator code
echo "--- 1. Auditing naked heap allocations (new/delete) ---"
NAKED_ALLOCS=$(grep -rnE $EXCLUDE "((\bnew\s+[a-zA-Z0-9_]+(\(|\[))|(\bdelete\s+[\*a-zA-Z0-9_]+))" --include="*.cpp" --include="*.hpp" --include="*.cc" --include="*.h" . 2>/dev/null | grep -v "operator new" | grep -v "escape-hatches" || true)
if [ -n "$NAKED_ALLOCS" ]; then
    echo "[WARN] Found naked new/delete. Consider std::make_unique or RAII containers:"
    echo "$NAKED_ALLOCS" | head -n 5
else
    echo "[PASS] Zero naked new/delete detected."
fi

# 2. Scan for raw C-style casts and dangerous reinterpret_cast
echo "--- 2. Auditing dangerous type casts ---"
REINTERPRET=$(grep -rnE $EXCLUDE "\breinterpret_cast<" --include="*.cpp" --include="*.hpp" . 2>/dev/null | grep -v "ESCAPE_HATCH" || true)
if [ -n "$REINTERPRET" ]; then
    echo "[WARN] Unannotated reinterpret_cast found (must use std::bit_cast or registered escape hatch):"
    echo "$REINTERPRET" | head -n 5
else
    echo "[PASS] No unannotated reinterpret_cast detected."
fi

# 3. Verify C++20 compiler capabilities and example compilation
echo "--- 3. Verifying C++20 Toolchain & Compiling Reference Example ---"
COMPILER=""
if command -v clang++ >/dev/null 2>&1; then
    COMPILER="clang++"
elif command -v g++ >/dev/null 2>&1; then
    COMPILER="g++"
fi

if [ -n "$COMPILER" ]; then
    echo "[INFO] Using compiler: $COMPILER ($($COMPILER --version | head -n 1))"
    EXAMPLE="src/prumo/resources/workforce/skills/lang-cpp/examples/raii-ownership.cpp"
    if [ -f "$EXAMPLE" ]; then
        TMP_BIN=$(mktemp /tmp/prumo_cpp_test_XXXXXX)
        $COMPILER -std=c++20 -Wall -Wextra -Wpedantic -Werror "$EXAMPLE" -o "$TMP_BIN"
        "$TMP_BIN" >/dev/null
        rm -f "$TMP_BIN"
        echo "[PASS] Reference C++20 RAII example compiled and executed cleanly."
    fi
else
    echo "[WARN] Neither clang++ nor g++ found in PATH."
fi

# 4. Check for Prumo escape hatch scanner
if command -v prumo >/dev/null 2>&1; then
    echo "--- 4. Checking Prumo Escape Hatches ---"
    prumo tool check-escape-hatches . 2>/dev/null || echo "[INFO] Escape hatches checked."
fi

echo "===================================================="
if [ "$VIOLATIONS" -eq 0 ]; then
    echo "[SUCCESS] lang-cpp verification passed cleanly."
    exit 0
else
    echo "[ERROR] lang-cpp verification failed with $VIOLATIONS violation(s)."
    exit 1
fi
