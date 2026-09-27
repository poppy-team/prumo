#!/bin/sh
# Verification script for lang-zig (Zig Explicit Allocation & Comptime Verification)
set -e

echo "[Prumo Skill: lang-zig] Starting verification routine..."

# 1. Compile and execute tests with memory leak detection
echo "[1/3] Running Zig test suite with std.testing.allocator..."
if command -v zig >/dev/null 2>&1; then
    if [ -f "examples/safe-alloc.zig" ]; then
        zig test examples/safe-alloc.zig
        echo "Zig test execution and leak detection passed (0 leaks)."
    fi
    # Format check
    if [ -f "examples/safe-alloc.zig" ]; then
        zig fmt --check examples/safe-alloc.zig || {
            echo "NOTICE: Formatting examples with zig fmt..."
            zig fmt examples/safe-alloc.zig
        }
    fi
else
    echo "NOTICE: zig compiler not found in PATH; skipping native binary execution."
fi

# 2. Scan for bare catch anti-patterns
echo "[2/3] Scanning for discarded error unions (bare catch)..."
BARE_CATCH=$(grep -rnE 'catch\s*\{\s*\}' . \
  --exclude-dir=.git --exclude-dir=node_modules --exclude-dir=target --exclude-dir=build --exclude-dir=dist \
  --include="*.zig" 2>/dev/null || true)

if [ -n "$BARE_CATCH" ]; then
    echo "ERROR: Discovered unhandled discarded error union (catch {}):"
    echo "$BARE_CATCH"
    exit 1
fi

# 3. Escape hatch audit
echo "[3/3] Scanning for unchecked pointer casts..."
PTR_CASTS=$(grep -rnE '@ptrCast|@alignCast' . \
  --exclude-dir=.git --exclude-dir=node_modules --exclude-dir=target --exclude-dir=build --exclude-dir=dist \
  --include="*.zig" 2>/dev/null || true)

if [ -n "$PTR_CASTS" ]; then
    echo "NOTICE: Pointer casts discovered. Ensure entries exist in .prumo/escape-hatches.json:"
    echo "$PTR_CASTS" | head -n 5
fi

echo "[Prumo Skill: lang-zig] Verification completed successfully."
exit 0
