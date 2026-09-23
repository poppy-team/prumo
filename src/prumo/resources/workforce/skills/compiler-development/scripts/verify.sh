#!/usr/bin/env bash
# Verification script for compiler-development (Compiler Development & Language Engineering)
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
SKILL_DIR="$(cd -- "${SCRIPT_DIR}/.." && pwd)"
cd "${SKILL_DIR}"

echo "[Prumo Skill: compiler-development] Starting verification routine..."

# 1. Compile and execute Pratt Parser demonstration
echo "[1/2] Compiling and running Pratt parser precedence test..."
if command -v g++ >/dev/null 2>&1; then
    CXX="g++"
elif command -v clang++ >/dev/null 2>&1; then
    CXX="clang++"
else
    CXX=""
fi

if [ -n "$CXX" ]; then
    if [ -f "examples/pratt_parser.cpp" ]; then
        $CXX -std=c++20 -O2 examples/pratt_parser.cpp -o /tmp/pratt_verify
        /tmp/pratt_verify
        rm -f /tmp/pratt_verify
        echo "Pratt parser executed and verified mathematical precedence."
    fi

    if [ -f "examples/parser_combinator.cpp" ]; then
        $CXX -std=c++20 -O2 examples/parser_combinator.cpp -o /tmp/pc_verify
        /tmp/pc_verify >/dev/null
        rm -f /tmp/pc_verify
        echo "Parser combinator compiled and executed successfully."
    fi
else
    echo "NOTICE: C++ compiler not found in PATH; skipping native binary execution."
fi

# 2. Syntax & structural checks
echo "[2/2] Checking compiler reference documentation and specifications..."
if [ ! -f "references/compiler-architecture.md" ]; then
    echo "ERROR: Missing compiler architecture reference documentation."
    exit 1
fi

echo "[Prumo Skill: compiler-development] Verification completed successfully."
exit 0
