#!/bin/sh
# Verification script for lang-asm (Assembly x86/x86-64 ABI & Calling Conventions)
set -e

echo "[Prumo Skill: lang-asm] Starting verification routine..."

# 1. Check for non-executable stack note
echo "[1/3] Checking for .note.GNU-stack directives in assembly files..."
MISSING_GNU_STACK=0
for f in $(find . -type f \( -name "*.s" -o -name "*.asm" \) \
  -not -path "*/.git/*" -not -path "*/node_modules/*" -not -path "*/target/*" -not -path "*/build/*"); do
    if ! grep -q ".note.GNU-stack" "$f"; then
        echo "ERROR: Missing .note.GNU-stack directive in $f"
        MISSING_GNU_STACK=1
    fi
done

if [ "$MISSING_GNU_STACK" -ne 0 ]; then
    echo "ERROR: Non-executable stack note missing from assembly sources."
    exit 1
fi

# 2. Check for DWARF CFI unwind directives
echo "[2/3] Checking for CFI unwinding directives..."
if [ -f "examples/safe-leaf.s" ]; then
    grep -q "\.cfi_startproc" "examples/safe-leaf.s" || {
        echo "ERROR: examples/safe-leaf.s missing .cfi_startproc"
        exit 1
    }
    grep -q "\.cfi_endproc" "examples/safe-leaf.s" || {
        echo "ERROR: examples/safe-leaf.s missing .cfi_endproc"
        exit 1
    }
fi

# 3. Assembler validation
echo "[3/3] Validating assembly syntax with GNU Assembler (as)..."
if command -v as >/dev/null 2>&1; then
    if [ -f "examples/safe-leaf.s" ]; then
        as examples/safe-leaf.s -o /tmp/asm_verify.o
        rm -f /tmp/asm_verify.o
        echo "Assembly assembled cleanly without errors."
    fi
else
    echo "NOTICE: GNU Assembler (as) not found; skipping compilation check."
fi

echo "[Prumo Skill: lang-asm] Verification completed successfully."
exit 0
