#!/usr/bin/env bash
# Verify color-science skill package integrity

set -euo pipefail

SKILL_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

echo "Verifying Color Science Skill Package..."

# Check required files
for file in "manifest.json" "SKILL.md" "checks/color-science-checklist.md" "templates/color-audit-report.md" "references/color-science-guide.md" "examples/oklch-palette.json"; do
    if [ ! -f "$SKILL_DIR/$file" ]; then
        echo "❌ Missing required file: $file"
        exit 1
    fi
done

# Basic manifest validation
if ! command -v jq >/dev/null 2>&1; then
    echo "⚠️ jq not installed, skipping JSON validation"
else
    if ! jq -e '.id == "color-science"' "$SKILL_DIR/manifest.json" > /dev/null; then
        echo "❌ Invalid manifest.json: id must be 'color-science'"
        exit 1
    fi
    echo "✅ manifest.json is valid"
fi

# Check for OKLCH keywords in SKILL.md
if ! grep -iq "oklch" "$SKILL_DIR/SKILL.md"; then
    echo "❌ SKILL.md is missing OKLCH references"
    exit 1
fi

echo "✅ All checks passed."
