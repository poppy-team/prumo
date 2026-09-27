#!/usr/bin/env bash
# Verifies the typography system files

set -euo pipefail

echo "Verifying Typography System Package..."

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
SKILL_ROOT="$( dirname "$DIR" )"

# Check if JSON scale is valid
if [ -f "$SKILL_ROOT/examples/type-scale.json" ]; then
    if jq empty "$SKILL_ROOT/examples/type-scale.json" >/dev/null 2>&1; then
        echo "✅ type-scale.json is valid JSON"
    else
        echo "❌ type-scale.json is invalid JSON"
        exit 1
    fi
else
    echo "❌ Missing type-scale.json example"
    exit 1
fi

# Check for presence of clamp() in the spec
if grep -q "clamp(" "$SKILL_ROOT/templates/typography-spec.md"; then
    echo "✅ Fluid typography 'clamp()' referenced in template"
else
    echo "❌ Missing 'clamp()' reference in template"
    exit 1
fi

echo "Typography System verification complete."
