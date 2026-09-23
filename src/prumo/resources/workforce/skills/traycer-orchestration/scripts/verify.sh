#!/usr/bin/env bash
# Verification script for Traycer Orchestration.
# Run from the repository root: bash src/prumo/resources/workforce/skills/traycer-orchestration/scripts/verify.sh
set -euo pipefail

SKILL_DIR="src/prumo/resources/workforce/skills/traycer-orchestration"
SKILL_ID="traycer-orchestration"

echo "===================================================="
echo " [Prumo Skill: $SKILL_ID] Traycer Orchestration Audit"
echo "===================================================="

VIOLATIONS=0
fail() { echo "[FAIL] $1"; VIOLATIONS=$((VIOLATIONS + 1)); }
pass() { echo "[PASS] $1"; }

# 1. Required skill assets exist and are non-empty
echo "--- 1. Verifying required skill assets ---"
for f in "SKILL.md" "manifest.json" "checks/traycer-orchestration-checklist.md" \
         "references/traycer-orchestration-guide.md" \
         "templates/traycer-orchestration-spec.md" "scripts/verify.sh"; do
  if [ -s "$SKILL_DIR/$f" ]; then
    pass "Found non-empty $f."
  else
    fail "Missing or empty $SKILL_DIR/$f."
  fi
done
if [ -x "$SKILL_DIR/scripts/verify.sh" ]; then
  pass "verify.sh is executable."
else
  fail "verify.sh is not executable."
fi

# 2. Stale superseded artifacts must be gone
echo "--- 2. Checking for stale superseded artifacts ---"
for f in "templates/traycer-orchestration-template.md"; do
  if [ -e "$SKILL_DIR/$f" ]; then
    fail "Stale artifact still present: $SKILL_DIR/$f."
  else
    pass "Stale artifact absent: $f."
  fi
done

# 3. Manifest validity and asset cross-references
echo "--- 3. Validating manifest and asset wiring ---"
if command -v python3 >/dev/null 2>&1; then
  if ! python3 - "$SKILL_DIR" "$SKILL_ID" <<'EOF'
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
print("[PASS] Manifest is valid and all wired assets exist.")
EOF
  then
    fail "Manifest validation failed (see details above)."
  fi
else
  echo "[SKIP] python3 not found; skipping manifest validation."
fi

# 4. Content invariants for traycer orchestration
echo "--- 4. Checking traycer-orchestration content invariants ---"
check_grep() {
  if grep -qiE "$1" "$SKILL_DIR/$2" 2>/dev/null; then
    pass "$3."
  else
    fail "$3."
  fi
}
check_grep "replaceable.*adapter|adapter.*replaceable" "SKILL.md" "SKILL.md mandates replaceable-adapter boundary"
check_grep "canonical|Goals?.*canonical" "SKILL.md" "SKILL.md mandates canonical repository Goals"
check_grep "plan graph|plan hash" "SKILL.md" "SKILL.md mandates versioned plan graphs"
check_grep "ledger|append-only" "SKILL.md" "SKILL.md mandates append-only step ledger"
check_grep "checkpoint|resume|replay" "SKILL.md" "SKILL.md mandates checkpoints with resume and replay"
check_grep "3 (retries|gates|attempts)|retry" "SKILL.md" "SKILL.md mandates bounded retry discipline"
check_grep "ledger\.jsonl|gate verdict|artifact hash" "references/traycer-orchestration-guide.md" "Reference guide covers ledger mechanics"
check_grep "resume drill|success rate|rework" "templates/traycer-orchestration-spec.md" "Spec template records drill and success metrics"
check_grep "trace completeness|resume|halt" "checks/traycer-orchestration-checklist.md" "Checklist covers trace, resume, and halt rules"

echo "===================================================="
if [ "$VIOLATIONS" -eq 0 ]; then
  echo "[SUCCESS] $SKILL_ID verification passed cleanly."
  exit 0
else
  echo "[ERROR] $SKILL_ID verification failed with $VIOLATIONS violation(s)."
  exit 1
fi
