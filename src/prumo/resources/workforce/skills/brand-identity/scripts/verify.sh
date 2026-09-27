#!/usr/bin/env bash
set -euo pipefail

# Verify Brand Identity Skill Package Invariants

SKILL_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CHECK_FAILED=0

echo "Verifying Brand Identity skill package..."

# 1. Check required files
REQUIRED_FILES=(
    "manifest.json"
    "SKILL.md"
    "checks/brand-identity-checklist.md"
    "templates/brand-guidelines-spec.md"
    "references/brand-identity-guide.md"
    "examples/brand-audit-example.md"
)

for file in "${REQUIRED_FILES[@]}"; do
    if [ ! -f "$SKILL_DIR/$file" ]; then
        echo "❌ Missing file: $file"
        CHECK_FAILED=1
    else
        echo "✅ Found $file"
    fi
done

# 2. Check SKILL.md sections
if [ -f "$SKILL_DIR/SKILL.md" ]; then
    SECTIONS=("Purpose" "Use When" "Do Not Use When" "Required Context" "Procedure" "Decision Rules" "Evidence Required" "Output Contract" "Stop Conditions" "Escalation Rules")
    for section in "${SECTIONS[@]}"; do
        if ! grep -q "# .*$section" "$SKILL_DIR/SKILL.md"; then
            echo "❌ SKILL.md missing section: $section"
            CHECK_FAILED=1
        fi
    done
    echo "✅ SKILL.md contains all 10 required sections"
fi

# 3. Check for OKLCH mention in templates
if [ -f "$SKILL_DIR/templates/brand-guidelines-spec.md" ]; then
    if ! grep -q "OKLCH" "$SKILL_DIR/templates/brand-guidelines-spec.md"; then
        echo "❌ templates/brand-guidelines-spec.md must include OKLCH references"
        CHECK_FAILED=1
    else
        echo "✅ templates/brand-guidelines-spec.md includes OKLCH references"
    fi
fi

if [ $CHECK_FAILED -eq 1 ]; then
    echo "❌ Verification failed!"
    exit 1
else
    echo "✅ Verification passed!"
    exit 0
fi
