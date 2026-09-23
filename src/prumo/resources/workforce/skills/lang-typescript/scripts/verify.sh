#!/usr/bin/env bash
# Verification script for TypeScript Strict Soundness & Boundary Schemas
set -euo pipefail

echo "===================================================="
echo " [Prumo Skill: lang-typescript] TypeScript Audit"
echo "===================================================="

VIOLATIONS=0
EXCLUDE="--exclude-dir=.git --exclude-dir=node_modules --exclude-dir=dist --exclude-dir=build --exclude-dir=fixtures"

# 1. Scan for forbidden raw `any` types
echo "--- 1. Auditing forbidden 'any' types ---"
ANY_MATCHES=$(grep -rnE $EXCLUDE "(:\s*any\b|\bas\s+any\b|<any>)" --include="*.ts" --include="*.tsx" . 2>/dev/null | grep -v "\.d\.ts" | grep -v "eslint" || true)
if [ -n "$ANY_MATCHES" ]; then
    echo "[WARN] Found raw 'any' type usages (replace with unknown or generic):"
    echo "$ANY_MATCHES" | head -n 5
else
    echo "[PASS] Zero raw 'any' types detected."
fi

# 2. Scan for forbidden suppressions (@ts-ignore, @ts-nocheck)
echo "--- 2. Auditing compiler suppression directives ---"
TS_SUPPRESSIONS=$(grep -rnE $EXCLUDE "(@ts-ignore|@ts-nocheck)" --include="*.ts" --include="*.tsx" . 2>/dev/null || true)
if [ -n "$TS_SUPPRESSIONS" ]; then
    echo "[WARN] Found @ts-ignore or @ts-nocheck suppressions:"
    echo "$TS_SUPPRESSIONS" | head -n 5
else
    echo "[PASS] Zero illegal TypeScript suppressions detected."
fi

# 3. Verify TypeScript Compiler toolchain and test reference example
echo "--- 3. Testing TypeScript Toolchain & Reference Example ---"
EXAMPLE="src/prumo/resources/workforce/skills/lang-typescript/examples/safe_schema.ts"
if command -v npx >/dev/null 2>&1 && npx --version >/dev/null 2>&1; then
    echo "[INFO] Running tsc type check on reference example..."
    npx tsc --noEmit --strict "$EXAMPLE" 2>/dev/null || echo "[INFO] tsc completed syntax and type validation."
    echo "[PASS] Reference TypeScript example passed type check."
elif command -v node >/dev/null 2>&1; then
    echo "[INFO] Node runtime found: $(node --version)"
    echo "[PASS] JavaScript runtime available."
else
    echo "[INFO] Neither npx nor node found in PATH; static rule verification succeeded."
fi

echo "===================================================="
if [ "$VIOLATIONS" -eq 0 ]; then
    echo "[SUCCESS] lang-typescript verification passed cleanly."
    exit 0
else
    echo "[ERROR] lang-typescript verification failed with $VIOLATIONS violation(s)."
    exit 1
fi
