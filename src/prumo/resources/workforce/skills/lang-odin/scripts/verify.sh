#!/bin/sh
# Verification script for lang-odin (Odin Context Management & Memory Control)
set -e

echo "[Prumo Skill: lang-odin] Starting verification routine..."

# 1. Compiler check if odin is installed
echo "[1/3] Checking Odin compiler availability..."
if command -v odin >/dev/null 2>&1; then
    if [ -f "examples/safe-odin.odin" ]; then
        echo "Running odin check on example..."
        odin check examples/safe-odin.odin -file
        echo "Odin check passed successfully."
    fi
else
    echo "NOTICE: odin compiler not found in PATH; skipping native compilation."
fi

# 2. Check for unregistered escape hatches
echo "[2/3] Scanning for raw pointer casts (cast(rawptr))..."
RAW_CASTS=$(grep -rnE 'cast\s*\(\s*rawptr\s*\)' . \
  --exclude-dir=.git --exclude-dir=node_modules --exclude-dir=target --exclude-dir=build --exclude-dir=dist \
  --include="*.odin" 2>/dev/null || true)

if [ -n "$RAW_CASTS" ]; then
    echo "NOTICE: Raw pointer casts discovered. Ensure entries exist in .prumo/escape-hatches.json:"
    echo "$RAW_CASTS" | head -n 5
fi

# 3. Dynamic array cleanup audit
echo "[3/3] Auditing dynamic allocations for defer delete pairings..."
if [ -f "examples/safe-odin.odin" ]; then
    grep -q "defer delete" "examples/safe-odin.odin" || {
        echo "ERROR: examples/safe-odin.odin missing defer delete statement."
        exit 1
    }
fi

echo "[Prumo Skill: lang-odin] Verification completed successfully."
exit 0
