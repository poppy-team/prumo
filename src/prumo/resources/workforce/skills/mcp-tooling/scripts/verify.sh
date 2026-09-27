#!/usr/bin/env bash
# Verification script for MCP Tooling & Integrations.
# Run from the repository root: bash src/prumo/resources/workforce/skills/mcp-tooling/scripts/verify.sh
set -euo pipefail

SKILL_DIR="src/prumo/resources/workforce/skills/mcp-tooling"

echo "===================================================="
echo " [Prumo Skill: mcp-tooling] MCP Tooling Audit"
echo "===================================================="

VIOLATIONS=0
fail() { echo "[FAIL] $1"; VIOLATIONS=$((VIOLATIONS + 1)); }
pass() { echo "[PASS] $1"; }

# 1. Required skill assets exist and are non-empty
echo "--- 1. Verifying required skill assets ---"
for f in "SKILL.md" "manifest.json" "checks/mcp-tooling-checklist.md" \
         "references/mcp-tooling-guide.md" \
         "templates/mcp-tooling-spec.md" "scripts/verify.sh"; do
  if [ -s "$SKILL_DIR/$f" ]; then
    pass "Found non-empty $f."
  else
    fail "Missing or empty $SKILL_DIR/$f."
  fi
done

# 2. Manifest validity and asset cross-references
echo "--- 2. Validating manifest and asset cross-references ---"
if command -v python3 >/dev/null 2>&1; then
  if ! python3 - "$SKILL_DIR" "mcp-tooling" <<'EOF'
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

# 3. Content invariants: SDK, transports, sessions, conformance
echo "--- 3. Checking mcp-tooling content invariants ---"
check_grep() {
  if grep -qiE "$1" "$SKILL_DIR/$2" 2>/dev/null; then
    pass "$3."
  else
    fail "$3."
  fi
}
check_grep "SDK|software development kit" "SKILL.md" "SKILL.md mandates official SDK use"
check_grep "stdio|Streamable HTTP" "SKILL.md" "SKILL.md mandates both transports"
check_grep "conformance|interop" "SKILL.md" "SKILL.md mandates conformance and interop"
check_grep "timeout|heartbeat|backoff" "checks/mcp-tooling-checklist.md" "Checklist covers timeouts and heartbeats"
check_grep "initialize|capabilit|negotiat" "checks/mcp-tooling-checklist.md" "Checklist covers version negotiation"
check_grep "session|resume|idempoten" "references/mcp-tooling-guide.md" "Reference guide covers sessions and idempotency"
check_grep "sampling|elicitation" "references/mcp-tooling-guide.md" "Reference guide covers sampling and elicitation"
check_grep "McpServer|StdioServerTransport|StreamableHTTP" "references/mcp-tooling-guide.md" "Reference guide shows server bootstrap code"
check_grep "transport|conform|interop" "templates/mcp-tooling-spec.md" "Tooling spec template records transports and interop"
check_grep "2025-06-18|1\\.12\\.0|214" "templates/mcp-tooling-spec.md" "Tooling spec template pins protocol, SDK, and battery"

echo "===================================================="
if [ "$VIOLATIONS" -eq 0 ]; then
  echo "[SUCCESS] mcp-tooling verification passed cleanly."
  exit 0
else
  echo "[ERROR] mcp-tooling verification failed with $VIOLATIONS violation(s)."
  exit 1
fi
