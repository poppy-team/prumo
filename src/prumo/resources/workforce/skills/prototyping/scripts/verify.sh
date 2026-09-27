#!/usr/bin/env bash
# ==============================================================================
# verify.sh: Invariant and Validation Suite for Prototyping Skill
# ==============================================================================
set -euo pipefail

SKILL_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
echo "🧪 Verifying prototyping skill invariants in: ${SKILL_DIR}"

ERRORS=0

# 1. Verify Manifest Integrity
if [ ! -f "${SKILL_DIR}/manifest.json" ]; then
  echo "❌ Missing manifest.json"
  ERRORS=$((ERRORS + 1))
else
  echo "✅ manifest.json exists"
fi

# 2. Check Reference Guide Depth
if [ ! -f "${SKILL_DIR}/references/design-sprint.md" ] || [ $(wc -l < "${SKILL_DIR}/references/design-sprint.md") -lt 30 ]; then
  echo "❌ references/design-sprint.md must contain comprehensive Design Sprint and FSM theory (>= 30 lines)"
  ERRORS=$((ERRORS + 1))
else
  echo "✅ references/design-sprint.md is comprehensive"
fi

# 3. Check Quality Checklist
if [ ! -f "${SKILL_DIR}/checks/fidelity-checklist.md" ] || [ $(wc -l < "${SKILL_DIR}/checks/fidelity-checklist.md") -lt 15 ]; then
  echo "❌ checks/fidelity-checklist.md must contain detailed verification gates (>= 15 lines)"
  ERRORS=$((ERRORS + 1))
else
  echo "✅ checks/fidelity-checklist.md is robust"
fi

# 4. Check Handoff Specification Template
if [ ! -f "${SKILL_DIR}/templates/handoff-spec.md" ] || [ $(wc -l < "${SKILL_DIR}/templates/handoff-spec.md") -lt 20 ]; then
  echo "❌ templates/handoff-spec.md must contain state matrices and scorecard (>= 20 lines)"
  ERRORS=$((ERRORS + 1))
else
  echo "✅ templates/handoff-spec.md is structured"
fi

# 5. Check User Flow Example
if [ ! -f "${SKILL_DIR}/examples/flow.md" ] || [ $(wc -l < "${SKILL_DIR}/examples/flow.md") -lt 20 ]; then
  echo "❌ examples/flow.md must contain cognitive walkthrough and state specs (>= 20 lines)"
  ERRORS=$((ERRORS + 1))
else
  echo "✅ examples/flow.md is detailed"
fi

if [ "${ERRORS}" -gt 0 ]; then
  echo "❌ Verification failed with ${ERRORS} errors."
  exit 1
fi

echo "✨ prototyping invariants verified successfully."
exit 0
