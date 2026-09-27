#!/usr/bin/env bash
# Verification script for Playtest Automation.
# Run from the repository root: bash src/prumo/resources/workforce/skills/playtest-automation/scripts/verify.sh
set -euo pipefail

SKILL_DIR="src/prumo/resources/workforce/skills/playtest-automation"

echo "===================================================="
echo " [Prumo Skill: playtest-automation] Playtest Automation Audit"
echo "===================================================="

VIOLATIONS=0
fail() { echo "[FAIL] $1"; VIOLATIONS=$((VIOLATIONS + 1)); }
pass() { echo "[PASS] $1"; }

# 1. Required skill assets exist and are non-empty
echo "--- 1. Verifying required skill assets ---"
for f in "SKILL.md" "manifest.json" "checks/playtest-automation-checklist.md" \
         "references/playtest-automation-guide.md" \
         "templates/playtest-automation-spec.md" "scripts/verify.sh"; do
  if [ -s "$SKILL_DIR/$f" ]; then
    pass "Found non-empty $f."
  else
    fail "Missing or empty $SKILL_DIR/$f."
  fi
done

# 2. Manifest validity and asset cross-references
echo "--- 2. Validating manifest and asset cross-references ---"
if command -v python3 >/dev/null 2>&1; then
  if ! python3 - "$SKILL_DIR" "playtest-automation" <<'EOF'
import json
import os
import sys

skill_dir, skill_id = sys.argv[1], sys.argv[2]
errors = []

with open(os.path.join(skill_dir, "manifest.json"), encoding="utf-8") as fh:
    manifest = json.load(fh)

if manifest.get("id") != skill_id:
    errors.append(f"manifest id must be {skill_id!r}")
if manifest.get("version", 0) < 2:
    errors.append("manifest version must be >= 2")
if manifest.get("schema_version", 0) < 3:
    errors.append("manifest schema_version must be >= 3")

valid_modes = {"implementation", "review", "audit", "research",
               "design", "testing", "documentation", "release"}
for mode in manifest.get("modes", []):
    if mode not in valid_modes:
        errors.append(f"invalid mode {mode!r} in manifest")

for key in ("references", "templates", "checks", "scripts"):
    for rel in manifest.get(key, []):
        if not os.path.isfile(os.path.join(skill_dir, rel)):
            errors.append(f"manifest {key} entry missing on disk: {rel}")

if errors:
    for err in errors:
        print(f"[FAIL] {err}")
    sys.exit(1)
print("[PASS] Manifest is valid and all listed assets exist on disk.")
EOF
  then
    fail "Manifest validation failed (see details above)."
  fi
else
  echo "[SKIP] python3 not found; skipping manifest validation."
fi

# 3. Content invariants: replay, bots, soak, telemetry
echo "--- 3. Checking playtest-automation content invariants ---"
check_grep() {
  if grep -qiE "$1" "$SKILL_DIR/$2" 2>/dev/null; then
    pass "$3."
  else
    fail "$3."
  fi
}
check_grep "deterministic|replay" "SKILL.md" "SKILL.md mandates deterministic replay"
check_grep "seed" "SKILL.md" "SKILL.md mandates seeded RNG control"
check_grep "stuck|watchdog|heartbeat" "SKILL.md" "SKILL.md mandates stuck detection"
check_grep "telemetry|tolerance" "SKILL.md" "SKILL.md mandates telemetry diffing"
check_grep "golden|archetype" "checks/playtest-automation-checklist.md" "Checklist covers golden replays and bot archetypes"
check_grep "soak|incident" "checks/playtest-automation-checklist.md" "Checklist covers soak and incidents"
check_grep "oracle|waypoint|timescale" "references/playtest-automation-guide.md" "Reference guide covers oracles and waypoints"
check_grep "taxonomy|stuck|hang|crash" "references/playtest-automation-guide.md" "Reference guide covers failure taxonomy"
check_grep "cohort|soak|telemetry" "templates/playtest-automation-spec.md" "Playtest spec template records cohort and soak"
check_grep "seed 4417|4417" "templates/playtest-automation-spec.md" "Playtest spec template pins a concrete seed"

echo "===================================================="
if [ "$VIOLATIONS" -eq 0 ]; then
  echo "[SUCCESS] playtest-automation verification passed cleanly."
  exit 0
else
  echo "[ERROR] playtest-automation verification failed with $VIOLATIONS violation(s)."
  exit 1
fi
