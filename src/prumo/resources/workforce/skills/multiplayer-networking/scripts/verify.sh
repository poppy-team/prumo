#!/usr/bin/env bash
# Verification script for Multiplayer Networking.
# Run from the repository root: bash src/prumo/resources/workforce/skills/multiplayer-networking/scripts/verify.sh
set -euo pipefail

SKILL_DIR="src/prumo/resources/workforce/skills/multiplayer-networking"

echo "===================================================="
echo " [Prumo Skill: multiplayer-networking] Multiplayer Networking Audit"
echo "===================================================="

VIOLATIONS=0
fail() { echo "[FAIL] $1"; VIOLATIONS=$((VIOLATIONS + 1)); }
pass() { echo "[PASS] $1"; }

# 1. Required skill assets exist and are non-empty
echo "--- 1. Verifying required skill assets ---"
for f in "SKILL.md" "manifest.json" "checks/multiplayer-networking-checklist.md" \
         "references/multiplayer-networking-guide.md" \
         "templates/multiplayer-networking-spec.md" "scripts/verify.sh"; do
  if [ -s "$SKILL_DIR/$f" ]; then
    pass "Found non-empty $f."
  else
    fail "Missing or empty $SKILL_DIR/$f."
  fi
done

# 2. Manifest validity and asset cross-references
echo "--- 2. Validating manifest and asset cross-references ---"
if command -v python3 >/dev/null 2>&1; then
  if ! python3 - "$SKILL_DIR" "multiplayer-networking" <<'EOF'
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

# 3. Content invariants: authority, prediction, bandwidth, compensation, reconnect
echo "--- 3. Checking multiplayer-networking content invariants ---"
check_grep() {
  if grep -qiE "$1" "$SKILL_DIR/$2" 2>/dev/null; then
    pass "$3."
  else
    fail "$3."
  fi
}
check_grep "server.authoritative|server authority|sole writer" "SKILL.md" "SKILL.md mandates server authority"
check_grep "prediction|reconciliation" "SKILL.md" "SKILL.md mandates prediction and reconciliation"
check_grep "delta compression|delta bitmask" "SKILL.md" "SKILL.md mandates delta compression"
check_grep "1200|MTU" "SKILL.md" "SKILL.md mandates the 1200-byte packet ceiling"
check_grep "lag compensation|rewind" "checks/multiplayer-networking-checklist.md" "Checklist covers lag compensation"
check_grep "reconnect|resync|baseline" "checks/multiplayer-networking-checklist.md" "Checklist covers reconnection"
check_grep "bandwidth|kbps" "checks/multiplayer-networking-checklist.md" "Checklist covers bandwidth budget"
check_grep "interpolation|extrapolation" "references/multiplayer-networking-guide.md" "Reference guide covers snapshot interpolation"
check_grep "quantization|bitmask" "references/multiplayer-networking-guide.md" "Reference guide covers quantization"
check_grep "tick|bandwidth|reconnect" "templates/multiplayer-networking-spec.md" "Netcode spec template covers tick, bandwidth, and reconnect"

echo "===================================================="
if [ "$VIOLATIONS" -eq 0 ]; then
  echo "[SUCCESS] multiplayer-networking verification passed cleanly."
  exit 0
else
  echo "[ERROR] multiplayer-networking verification failed with $VIOLATIONS violation(s)."
  exit 1
fi
