#!/bin/bash
# verify.sh - Verifies responsive design requirements in CSS files

TARGET_DIR="${1:-.}"
EXIT_CODE=0

echo "Running Responsive Design Validation on $TARGET_DIR..."

# Check for fixed pixel breakpoints
if grep -r -E "@media.*min-width:[[:space:]]*[0-9]+px" "$TARGET_DIR" | grep -E "(320px|768px|1024px)"; then
    echo "⚠️  WARNING: Hardcoded device-specific breakpoints detected. Use content-first breakpoints."
fi

# Check for container queries
if ! grep -r -q "container-type" "$TARGET_DIR"; then
    echo "❌ ERROR: No container queries found (container-type missing). Use container queries for component adaptivity."
    EXIT_CODE=1
fi

# Check for fluid typography
if ! grep -r -q "clamp(" "$TARGET_DIR"; then
    echo "❌ ERROR: No fluid sizing found (clamp missing). Use fluid typography and spacing."
    EXIT_CODE=1
fi

# Check for logical properties
if grep -r -E "(margin-left|margin-right|padding-left|padding-right)" "$TARGET_DIR" | grep -v "node_modules"; then
    echo "⚠️  WARNING: Physical properties detected. Prefer logical properties (e.g., margin-inline)."
fi

# Check for hover media query
if ! grep -r -q "@media.*hover.*hover" "$TARGET_DIR"; then
    echo "⚠️  WARNING: @media (hover: hover) not found. Ensure touch devices don't receive stuck hover states."
fi

if [ $EXIT_CODE -eq 0 ]; then
    echo "✅ Responsive Design validation passed."
else
    echo "❌ Responsive Design validation failed."
fi

exit $EXIT_CODE
