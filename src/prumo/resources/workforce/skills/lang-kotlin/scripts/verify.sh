#!/bin/sh
set -eu
echo "=== Verifying Kotlin Coroutines & Sound Null-Safety ==="
prumo tool check-escape-hatches src/prumo/resources/workforce/skills/lang-kotlin
exit 0
