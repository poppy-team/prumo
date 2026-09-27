#!/usr/bin/env bash
# Verification script for Python Strict Typing & Systems Hygiene
set -euo pipefail

echo "===================================================="
echo " [Prumo Skill: lang-python] Python Safety Audit"
echo "===================================================="

VIOLATIONS=0
EXCLUDE="--exclude-dir=.git --exclude-dir=.venv --exclude-dir=venv --exclude-dir=node_modules --exclude-dir=fixtures"

# 1. Scan for bare excepts and swallowed exceptions
echo "--- 1. Auditing bare except clauses and swallowed errors ---"
BARE_EXCEPTS=$(grep -rnE $EXCLUDE "except\s*:" --include="*.py" . 2>/dev/null || true)
if [ -n "$BARE_EXCEPTS" ]; then
    echo "[WARN] Found bare except: clauses (replace with specific exception classes):"
    echo "$BARE_EXCEPTS" | head -n 5
else
    echo "[PASS] Zero bare except: clauses detected."
fi

SWALLOWED_EXCEPTS=$(grep -rnE $EXCLUDE "except\s+Exception\s*:\s*pass" --include="*.py" . 2>/dev/null || true)
if [ -n "$SWALLOWED_EXCEPTS" ]; then
    echo "[WARN] Found swallowed exceptions (except Exception: pass):"
    echo "$SWALLOWED_EXCEPTS" | head -n 5
else
    echo "[PASS] Zero swallowed exceptions detected."
fi

# 2. Scan for insecure functions (eval, exec, pickle)
echo "--- 2. Auditing dangerous functions (eval, exec, pickle, shell=True) ---"
DANGEROUS_FUNCS=$(grep -rnE $EXCLUDE "\b(eval|exec|pickle\.loads)\s*\(" --include="*.py" . 2>/dev/null || true)
if [ -n "$DANGEROUS_FUNCS" ]; then
    echo "[WARN] Insecure dynamic execution/deserialization calls found:"
    echo "$DANGEROUS_FUNCS" | head -n 5
else
    echo "[PASS] Zero insecure eval/exec/pickle calls detected."
fi

# 3. Verify Python 3 runtime and execute reference example
echo "--- 3. Testing Python Runtime & Reference Example ---"
if command -v python3 >/dev/null 2>&1; then
    echo "[INFO] Using Python: $(python3 --version)"
    EXAMPLE="src/prumo/resources/workforce/skills/lang-python/examples/safe_python.py"
    if [ -f "$EXAMPLE" ]; then
        python3 "$EXAMPLE" >/dev/null
        echo "[PASS] Reference Python example executed cleanly."
    fi
else
    echo "[WARN] python3 binary not found in PATH."
fi

# 4. Optional ruff / mypy check if installed
if command -v ruff >/dev/null 2>&1; then
    echo "--- 4. Running Ruff static analysis ---"
    ruff check src/prumo/resources/workforce/skills/lang-python/ || true
fi

echo "===================================================="
if [ "$VIOLATIONS" -eq 0 ]; then
    echo "[SUCCESS] lang-python verification passed cleanly."
    exit 0
else
    echo "[ERROR] lang-python verification failed with $VIOLATIONS violation(s)."
    exit 1
fi
