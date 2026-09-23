#!/bin/sh
set -eu
echo "=== Verifying Dart Sound Null-Safety & Isolate Discipline ==="
prumo tool check-escape-hatches src/prumo/resources/workforce/skills/lang-dart
exit 0
