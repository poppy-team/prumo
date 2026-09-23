#!/usr/bin/env bash
# Verification script for Network Resilience Testing.
# Run from the repository root: bash src/prumo/resources/workforce/skills/network-testing/scripts/verify.sh
set -euo pipefail

SKILL_DIR="src/prumo/resources/workforce/skills/network-testing"

echo "===================================================="
echo " [Prumo Skill: network-testing] Network Audit"
echo "===================================================="

VIOLATIONS=0
fail() { echo "[FAIL] $1"; VIOLATIONS=$((VIOLATIONS + 1)); }
pass() { echo "[PASS] $1"; }

# 1. Required skill assets exist and are non-empty
echo "--- 1. Verifying required skill assets ---"
for f in "SKILL.md" "manifest.json" "checks/network-testing-checklist.md" \
         "references/network-testing-guide.md" \
         "templates/network-testing-spec.md" "scripts/verify.sh"; do
  if [ -s "$SKILL_DIR/$f" ]; then
    pass "Found non-empty $f."
  else
    fail "Missing or empty $SKILL_DIR/$f."
  fi
done

# 2. Manifest validity and asset cross-references
echo "--- 2. Validating manifest and asset cross-references ---"
if command -v python3 >/dev/null 2>&1; then
  if ! python3 - "$SKILL_DIR" <<'EOF'
import json
import os
import sys

skill_dir = sys.argv[1]
errors = []

with open(os.path.join(skill_dir, "manifest.json"), encoding="utf-8") as fh:
    manifest = json.load(fh)

if manifest.get("id") != "network-testing":
    errors.append("manifest id must be 'network-testing'")
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
print("[PASS] Manifest is valid and all listed assets exist.")
EOF
  then
    fail "Manifest validation failed (see details above)."
  fi
else
  echo "[SKIP] python3 not found; skipping manifest validation."
fi

# 3. Content invariants: core network-testing mechanisms
echo "--- 3. Checking network-testing content invariants ---"
check_grep() {
  if grep -qiE "$1" "$SKILL_DIR/$2" 2>/dev/null; then
    pass "$3."
  else
    fail "$3."
  fi
}
check_grep "packet loss" "SKILL.md" "SKILL.md mandates packet loss testing"
check_grep "netem|toxiproxy" "SKILL.md" "SKILL.md mandates netem or Toxiproxy injection"
check_grep "backoff" "SKILL.md" "SKILL.md mandates exponential backoff"
check_grep "idempoten" "SKILL.md" "SKILL.md mandates idempotent retries"
check_grep "jitter|packet loss" "checks/network-testing-checklist.md" "Checklist covers jitter and packet loss"
check_grep "retry" "checks/network-testing-checklist.md" "Checklist covers retry discipline"
check_grep "partition|disconnect" "checks/network-testing-checklist.md" "Checklist covers partitions and disconnects"
check_grep "chaos|fault injection" "references/network-testing-guide.md" "Reference guide covers chaos and fault injection"
check_grep "exponential" "references/network-testing-guide.md" "Reference guide covers exponential backoff"
check_grep "recovery|p99|rto" "templates/network-testing-spec.md" "Spec template carries recovery time evidence"

echo "===================================================="
if [ "$VIOLATIONS" -eq 0 ]; then
  echo "[SUCCESS] network-testing verification passed cleanly."
  exit 0
else
  echo "[ERROR] network-testing verification failed with $VIOLATIONS violation(s)."
  exit 1
fi
