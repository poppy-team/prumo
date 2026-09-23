#!/usr/bin/env bash
# Verification script for ast-transformation (AST Transformation & Automated Codemods)
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
SKILL_DIR="$(cd -- "${SCRIPT_DIR}/.." && pwd)"
cd "${SKILL_DIR}"

echo "[Prumo Skill: ast-transformation] Starting verification routine..."

# 1. Run Python AST transformation idempotency test
echo "[1/2] Running AST transformer self-test and idempotency verification..."
if command -v python3 >/dev/null 2>&1; then
    if [ -f "examples/ast_transformer.py" ]; then
        python3 examples/ast_transformer.py
    fi
else
    echo "NOTICE: python3 not found; skipping AST transform execution."
fi

# 2. Check documentation assets
echo "[2/2] Verifying specification and reference documentation..."
if [ ! -f "references/ast-transformation-guide.md" ]; then
    echo "ERROR: Missing ast-transformation-guide.md"
    exit 1
fi

echo "[Prumo Skill: ast-transformation] Verification completed successfully."
exit 0
