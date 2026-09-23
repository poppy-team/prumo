#!/usr/bin/env sh
# Verification script for state-management (State Management Architecture)
set -e

echo "[Prumo Skill: state-management] Starting verification routine..."

# 1. Execute TypeScript State Machine Invariant Test
echo "[1/2] Running state machine self-test..."
if [ -f "examples/state_machine.ts" ] && command -v node >/dev/null 2>&1; then
    node --experimental-strip-types examples/state_machine.ts || {
        echo "ERROR: State machine verification test failed."
        exit 1
    }
fi

# 2. Static scan for boolean flag explosions in state stores
echo "[2/2] Scanning for boolean soup anti-patterns in stores/components..."
BOOLEAN_SOUP=$(grep -rnE 'isLoading\s*:\s*boolean.*isError\s*:\s*boolean' . \
  --exclude-dir=.git --exclude-dir=node_modules --exclude-dir=target --exclude-dir=build --exclude-dir=dist \
  --exclude="*.md" \
  --include="*.ts" --include="*.tsx" --include="*.js" 2>/dev/null || true)

if [ -n "$BOOLEAN_SOUP" ]; then
    echo "NOTICE: Consider replacing multiple boolean status flags with discriminated union states:"
    echo "$BOOLEAN_SOUP" | head -n 5
fi

echo "[Prumo Skill: state-management] Verification completed successfully."
exit 0
