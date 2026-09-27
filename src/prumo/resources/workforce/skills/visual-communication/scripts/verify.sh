#!/bin/bash
set -euo pipefail

SKILL_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

echo "Verifying Visual Communication skill package..."

# 1. Check required files exist
REQUIRED_FILES=(
    "manifest.json"
    "SKILL.md"
    "checks/visual-communication-checklist.md"
    "templates/visual-strategy-spec.md"
    "references/visual-communication-guide.md"
    "examples/visual-hierarchy-audit.md"
)

for file in "${REQUIRED_FILES[@]}"; do
    if [ ! -f "$SKILL_DIR/$file" ]; then
        echo "❌ Error: Required file missing: $file"
        exit 1
    fi
done

# 2. Verify JSON validity of manifest
if ! command -v jq &> /dev/null; then
    echo "⚠️ jq not installed, skipping JSON validation"
else
    if ! jq empty "$SKILL_DIR/manifest.json"; then
        echo "❌ Error: manifest.json is invalid JSON"
        exit 1
    fi
fi

# 3. Check SKILL.md for required sections
REQUIRED_SECTIONS=(
    "1. Purpose"
    "2. Use When"
    "3. Do Not Use When"
    "4. Required Context"
    "5. Procedure"
    "6. Decision Rules"
    "7. Evidence Required"
    "8. Output Contract"
    "9. Stop Conditions"
    "10. Escalation Rules"
)

for section in "${REQUIRED_SECTIONS[@]}"; do
    if ! grep -q "$section" "$SKILL_DIR/SKILL.md"; then
        echo "❌ Error: SKILL.md is missing required section: $section"
        exit 1
    fi
done

# 4. Check that templates are actually templates (contain placeholders)
if ! grep -q "\[.*\]" "$SKILL_DIR/templates/visual-strategy-spec.md"; then
    echo "❌ Error: Template does not contain bracketed placeholders [like this]"
    exit 1
fi

echo "✅ Visual Communication skill verification passed!"
exit 0
