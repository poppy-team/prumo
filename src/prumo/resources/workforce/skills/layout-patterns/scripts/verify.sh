#!/bin/bash
# Verify implementation of layout primitives
# Usage: ./verify.sh <path-to-css-file>

if [ -z "$1" ]; then
  echo "Usage: $0 <path-to-css-file>"
  exit 1
fi

CSS_FILE="$1"

if [ ! -f "$CSS_FILE" ]; then
  echo "Error: File not found - $CSS_FILE"
  exit 1
fi

echo "Verifying Layout Patterns in $CSS_FILE..."
FAILURES=0

# Check for Flexbox/Grid usage
if ! grep -Eq "display:\s*(flex|grid)" "$CSS_FILE"; then
  echo "[WARNING] No flex or grid layouts detected. Ensure layout primitives are used."
  ((FAILURES++))
else
  echo "[PASS] Flex/Grid systems detected."
fi

# Check for modern spacing (gap)
if ! grep -Eq "gap:\s*" "$CSS_FILE"; then
  echo "[WARNING] Missing 'gap' usage. Expected for Stack/Cluster/Grid primitives."
  ((FAILURES++))
else
  echo "[PASS] 'gap' detected."
fi

# Check for Fluid Typography/Spacing Constraints
if ! grep -Eq "clamp\(|min\(|max\(" "$CSS_FILE"; then
  echo "[WARNING] No fluid constraints (clamp/min/max) detected. Consider intrinsic sizing."
else
  echo "[PASS] Fluid constraints detected."
fi

# Check for bad practices: absolute widths
if grep -Eq "width:\s*[0-9]+px" "$CSS_FILE"; then
  echo "[WARNING] Fixed pixel widths detected. Rely on relative intrinsic sizing or max-width instead."
  ((FAILURES++))
fi

if [ $FAILURES -gt 0 ]; then
  echo "Verification complete with warnings/errors."
  exit 0 # Non-fatal for structural verification, but informs the user
else
  echo "Verification complete. All basic layout invariants pass."
  exit 0
fi
