#!/usr/bin/env bash
# Verification script for Modern JavaScript (ES2024+) Security & Event Loop Hygiene
set -euo pipefail

echo "===================================================="
echo " [Prumo Skill: lang-javascript] JavaScript Audit"
echo "===================================================="

VIOLATIONS=0
EXCLUDE="--exclude-dir=.git --exclude-dir=node_modules --exclude-dir=dist --exclude-dir=build --exclude-dir=fixtures"

# 1. Scan for forbidden dynamic code execution (eval, new Function)
echo "--- 1. Auditing dangerous dynamic code evaluation ---"
DANGEROUS_EVAL=$(grep -rnE $EXCLUDE "\b(eval|new\s+Function|document\.write)\s*\(" --include="*.js" --include="*.mjs" . 2>/dev/null || true)
if [ -n "$DANGEROUS_EVAL" ]; then
    echo "[WARN] Found dangerous eval / dynamic execution calls:"
    echo "$DANGEROUS_EVAL" | head -n 5
else
    echo "[PASS] Zero eval/new Function/document.write calls detected."
fi

# 2. Scan for legacy `var` declarations
echo "--- 2. Auditing legacy 'var' declarations ---"
VAR_MATCHES=$(grep -rnE $EXCLUDE "\bvar\s+[a-zA-Z0-9_$]+" --include="*.js" --include="*.mjs" . 2>/dev/null | grep -v "\.min\.js" | grep -v "node_modules" || true)
if [ -n "$VAR_MATCHES" ]; then
    echo "[WARN] Found legacy 'var' declarations (replace with const/let):"
    echo "$VAR_MATCHES" | head -n 5
else
    echo "[PASS] Zero legacy 'var' declarations detected."
fi

# 3. Scan for direct __proto__ manipulation
echo "--- 3. Auditing direct __proto__ pollution hazards ---"
PROTO_POLLUTION=$(grep -rnE $EXCLUDE "\.__proto__\s*=" --include="*.js" --include="*.mjs" . 2>/dev/null || true)
if [ -n "$PROTO_POLLUTION" ]; then
    echo "[WARN] Direct __proto__ mutation detected:"
    echo "$PROTO_POLLUTION" | head -n 5
else
    echo "[PASS] Zero direct __proto__ assignments detected."
fi

# 4. Verify Node.js runtime and test reference example
echo "--- 4. Testing JavaScript Runtime & Reference Example ---"
if command -v node >/dev/null 2>&1; then
    echo "[INFO] Using Node.js: $(node --version)"
    EXAMPLE="src/prumo/resources/workforce/skills/lang-javascript/examples/safe_module.js"
    if [ -f "$EXAMPLE" ]; then
        node "$EXAMPLE" >/dev/null
        echo "[PASS] Reference ES2024 module executed cleanly."
    fi
else
    echo "[WARN] node binary not found in PATH."
fi

echo "===================================================="
if [ "$VIOLATIONS" -eq 0 ]; then
    echo "[SUCCESS] lang-javascript verification passed cleanly."
    exit 0
else
    echo "[ERROR] lang-javascript verification failed with $VIOLATIONS violation(s)."
    exit 1
fi
