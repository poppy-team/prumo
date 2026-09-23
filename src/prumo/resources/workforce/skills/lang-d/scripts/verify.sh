#!/bin/sh
set -eu

echo "=== Verifying Dlang Safety & Scope Pointers ==="

echo "[1/2] Scanning for unregistered escape hatches..."
prumo tool check-escape-hatches src/prumo/resources/workforce/skills/lang-d

echo "[2/2] Dlang verification complete."
exit 0
