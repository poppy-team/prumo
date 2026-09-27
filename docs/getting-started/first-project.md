# First Project Walkthrough

## 1. Initialize a Project
```bash
mkdir my-app && cd my-app
git init
prumo-agent init --non-interactive --profile /path/to/project-profile.json
```

## 2. Check Diagnostics
```bash
prumo-agent doctor
```

## 3. Create and Lock a Goal
```bash
prumo-agent goal new P01-G01 "Core Application Engine" --phase P01 --objective "Build resilient core application logic."
prumo-agent goal state P01-G01 PLANNED
prumo-agent goal state P01-G01 LOCKED
```

## 4. Compile Target Adapter
```bash
prumo-agent compile --target codex
```
