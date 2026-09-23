#!/bin/sh
set -eu
echo "=== Verifying C# & .NET Engineering and Memory Efficiency ==="
prumo tool check-escape-hatches src/prumo/resources/workforce/skills/lang-csharp
exit 0
