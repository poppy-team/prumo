#!/usr/bin/env bash
# ==============================================================================
# verify.sh: Invariant and Validation Suite for Generative SVG Art Skill
# ==============================================================================
set -euo pipefail

SKILL_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
echo "🎨 Verifying generative-svg-art skill invariants in: ${SKILL_DIR}"

ERRORS=0

# 1. Verify Manifest Integrity
if [ ! -f "${SKILL_DIR}/manifest.json" ]; then
  echo "❌ Missing manifest.json"
  ERRORS=$((ERRORS + 1))
else
  echo "✅ manifest.json exists"
fi

# 2. Check Reference Guide Depth
if [ ! -f "${SKILL_DIR}/references/guide.md" ] || [ $(wc -l < "${SKILL_DIR}/references/guide.md") -lt 30 ]; then
  echo "❌ references/guide.md must contain comprehensive mathematical and filter theory (>= 30 lines)"
  ERRORS=$((ERRORS + 1))
else
  echo "✅ references/guide.md is comprehensive"
fi

# 3. Check Quality Checklist
if [ ! -f "${SKILL_DIR}/checks/checklist.md" ] || [ $(wc -l < "${SKILL_DIR}/checks/checklist.md") -lt 15 ]; then
  echo "❌ checks/checklist.md must contain detailed verification gates (>= 15 lines)"
  ERRORS=$((ERRORS + 1))
else
  echo "✅ checks/checklist.md is robust"
fi

# 4. Validate Example SVG
EXAMPLE_SVG="${SKILL_DIR}/examples/example.svg"
if [ ! -f "${EXAMPLE_SVG}" ]; then
  echo "❌ Missing examples/example.svg"
  ERRORS=$((ERRORS + 1))
else
  # Check for viewBox
  if ! grep -q 'viewBox=' "${EXAMPLE_SVG}"; then
    echo "❌ example.svg missing viewBox attribute"
    ERRORS=$((ERRORS + 1))
  fi
  # Check for XML closing tag
  if ! grep -q '</svg>' "${EXAMPLE_SVG}"; then
    echo "❌ example.svg is not well-formed (missing </svg>)"
    ERRORS=$((ERRORS + 1))
  fi
  # Check for accessibility tags (role="img" or aria-hidden)
  if ! grep -E -q 'role="img"|aria-hidden="true"' "${EXAMPLE_SVG}"; then
    echo "❌ example.svg lacks accessibility declaration (role='img' or aria-hidden='true')"
    ERRORS=$((ERRORS + 1))
  fi
  echo "✅ example.svg is well-formed, responsive and accessible"
fi

if [ "${ERRORS}" -gt 0 ]; then
  echo "❌ Verification failed with ${ERRORS} errors."
  exit 1
fi

echo "✨ generative-svg-art invariants verified successfully."
exit 0
