#!/bin/sh
# Verification script for lang-c3 (C3 Semantic Safety & Contract Assertions)
set -e

echo "[Prumo Skill: lang-c3] Starting verification routine..."

# 1. Compiler check if c3c is installed
echo "[1/3] Checking C3 compiler availability..."
if command -v c3c >/dev/null 2>&1; then
    if [ -f "examples/safe-defer.c3" ]; then
        echo "Running c3c check on example..."
        c3c check examples/safe-defer.c3
        echo "C3 check passed successfully."
    fi
else
    echo "NOTICE: c3c compiler not found in PATH; skipping native compilation."
fi

# 2. Defer hygiene check on memory allocations
echo "[2/3] Auditing memory allocations for defer release..."
if [ -f "examples/safe-defer.c3" ]; then
    grep -q "defer" "examples/safe-defer.c3" || {
        echo "ERROR: examples/safe-defer.c3 missing defer statements."
        exit 1
    }
fi

# 3. Scan for unchecked pointer casts
echo "[3/3] Scanning for raw pointer casts..."
RAW_CASTS=$(grep -rnE '\(void\*\)' . \
  --exclude-dir=.git --exclude-dir=node_modules --exclude-dir=target --exclude-dir=build --exclude-dir=dist \
  --include="*.c3" 2>/dev/null || true)

if [ -n "$RAW_CASTS" ]; then
    echo "NOTICE: Raw pointer casts discovered. Ensure entries exist in .prumo/escape-hatches.json:"
    echo "$RAW_CASTS" | head -n 5
fi

echo "[Prumo Skill: lang-c3] Verification completed successfully."
exit 0
