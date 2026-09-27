#!/usr/bin/env bash
# Verification script for interaction-design (Interaction Design & Motion Engineering)
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
SKILL_DIR="$(cd -- "${SCRIPT_DIR}/.." && pwd)"
cd "${SKILL_DIR}"

echo "[Prumo Skill: interaction-design] Starting verification routine..."

# 1. Audit micro-interaction model in examples
echo "[1/4] Checking micro-interaction 4-part model in examples..."
python3 -c "
import os

sample_path = 'examples/microinteraction_spec.md'
if not os.path.exists(sample_path):
    raise FileNotFoundError('Missing ' + sample_path)

with open(sample_path, 'r', encoding='utf-8') as f:
    text = f.read()

required_stages = ['Trigger', 'Rules', 'Feedback', 'Loops & Modes']
for stage in required_stages:
    if stage not in text:
        raise ValueError('Example missing micro-interaction stage: ' + stage)

print('Micro-interaction model validated.')
"

# 2. Audit animation duration upper bounds (should not exceed 400ms)
echo "[2/4] Auditing transition durations for sluggishness (>400ms)..."
SLUGGISH_TRANSITIONS="$(
  find "${SKILL_DIR}" -type f \( -name "*.css" -o -name "*.scss" \) \
    -exec grep -HnE 'transition.*:\s*([5-9][0-9]{2}|[1-9][0-9]{3,})ms' {} + 2>/dev/null || true
)"

if [ -n "$SLUGGISH_TRANSITIONS" ]; then
    echo "WARNING: Animation durations exceeding 400ms discovered (ensure optimal interaction speed):"
    echo "$SLUGGISH_TRANSITIONS" | head -n 5
fi

# 3. Check for prefers-reduced-motion support
echo "[3/4] Checking for prefers-reduced-motion accessibility overrides..."
if [ -f "examples/microinteraction_spec.md" ]; then
    grep -q "prefers-reduced-motion" "examples/microinteraction_spec.md" || {
        echo "ERROR: Missing prefers-reduced-motion override in examples/microinteraction_spec.md"
        exit 1
    }
fi

# 4. Template verification
echo "[4/4] Checking template integrity..."
if [ ! -f "templates/interaction-spec.md" ]; then
    echo "ERROR: Missing templates/interaction-spec.md"
    exit 1
fi

echo "[Prumo Skill: interaction-design] Verification completed successfully."
exit 0
