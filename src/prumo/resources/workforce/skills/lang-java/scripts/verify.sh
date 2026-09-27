#!/bin/sh
set -eu
echo "=== Verifying Java LTS Modern Engineering & Concurrency Discipline ==="
prumo tool check-escape-hatches src/prumo/resources/workforce/skills/lang-java
exit 0
