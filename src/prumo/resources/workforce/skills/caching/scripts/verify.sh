#!/usr/bin/env bash
# Verification script for Caching Architecture, Invalidation & Resilience.
# Run from the repository root: bash src/prumo/resources/workforce/skills/caching/scripts/verify.sh
set -euo pipefail

SKILL_DIR="src/prumo/resources/workforce/skills/caching"
CATALOG="src/prumo/resources/catalog/skills.json"

echo "===================================================="
echo " [Prumo Skill: caching] Caching Architecture Audit"
echo "===================================================="

VIOLATIONS=0
fail() { echo "[FAIL] $1"; VIOLATIONS=$((VIOLATIONS + 1)); }
pass() { echo "[PASS] $1"; }

# 1. Required skill assets exist and are non-empty
echo "--- 1. Verifying required skill assets ---"
for f in "SKILL.md" "manifest.json" "checks/caching-checklist.md" \
         "references/caching-architecture-guide.md" \
         "templates/caching-strategy-spec.md" "scripts/verify.sh"; do
  if [ -s "$SKILL_DIR/$f" ]; then
    pass "Found non-empty $f."
  else
    fail "Missing or empty $SKILL_DIR/$f."
  fi
done

# 2. Stale superseded artifacts must be gone
echo "--- 2. Checking for stale superseded artifacts ---"
for f in "references/caching-guide.md" "templates/caching-template.md"; do
  if [ -e "$SKILL_DIR/$f" ]; then
    fail "Stale artifact still present: $SKILL_DIR/$f."
  else
    pass "Stale artifact absent: $f."
  fi
done

# 3. Manifest validity, asset cross-references, and catalog consistency
echo "--- 3. Validating manifest and catalog consistency ---"
if command -v python3 >/dev/null 2>&1; then
  if ! python3 - "$SKILL_DIR" "$CATALOG" <<'EOF'
import json
import os
import sys

skill_dir, catalog_path = sys.argv[1], sys.argv[2]
errors = []

with open(os.path.join(skill_dir, "manifest.json"), encoding="utf-8") as fh:
    manifest = json.load(fh)

if manifest.get("id") != "caching":
    errors.append("manifest id must be 'caching'")
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

with open(catalog_path, encoding="utf-8") as fh:
    catalog = json.load(fh)
entries = [s for s in catalog.get("skills", []) if s.get("id") == "caching"]
if len(entries) != 1:
    errors.append("catalog must contain exactly one caching entry")
else:
    entry = dict(entries[0])
    entry.pop("instructions", None)
    if entry != manifest:
        errors.append("catalog caching entry diverges from workforce manifest.json")

if errors:
    for err in errors:
        print(f"[FAIL] {err}")
    sys.exit(1)
print("[PASS] Manifest is valid and catalog entry mirrors it.")
EOF
  then
    fail "Manifest/catalog validation failed (see details above)."
  fi
else
  echo "[SKIP] python3 not found; skipping manifest/catalog validation."
fi

# 4. Content invariants: the four failure modes and core mechanisms
echo "--- 4. Checking caching content invariants ---"
check_grep() {
  if grep -qiE "$1" "$SKILL_DIR/$2" 2>/dev/null; then
    pass "$3."
  else
    fail "$3."
  fi
}
check_grep "singleflight" "SKILL.md" "SKILL.md mandates SingleFlight coalescing"
check_grep "ttl" "SKILL.md" "SKILL.md mandates explicit TTLs"
check_grep "commit" "SKILL.md" "SKILL.md mandates post-commit invalidation"
check_grep "bound" "SKILL.md" "SKILL.md mandates bounded capacity"
check_grep "stampede" "checks/caching-checklist.md" "Checklist covers stampede"
check_grep "avalanche" "checks/caching-checklist.md" "Checklist covers avalanche"
check_grep "penetration" "checks/caching-checklist.md" "Checklist covers penetration"
check_grep "breakdown" "checks/caching-checklist.md" "Checklist covers breakdown"
check_grep "eviction|LRU|W-TinyLFU" "references/caching-architecture-guide.md" "Reference guide covers eviction models"
check_grep "invalidation" "templates/caching-strategy-spec.md" "Strategy template covers invalidation"

echo "===================================================="
if [ "$VIOLATIONS" -eq 0 ]; then
  echo "[SUCCESS] caching verification passed cleanly."
  exit 0
else
  echo "[ERROR] caching verification failed with $VIOLATIONS violation(s)."
  exit 1
fi
