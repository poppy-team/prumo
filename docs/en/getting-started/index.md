# Getting Started with Prumo

Welcome to the official documentation for the **Prumo Framework** (v0.6.0).

Prumo is a Git-native protocol and engineering harness built for reliable, evidence-driven software development with human engineers and autonomous AI agents.

## Quick Installation

### Linux & macOS

```bash
curl -fsSL https://raw.githubusercontent.com/poppy-team/prumo/main/scripts/install.sh | sh
```

### Windows (PowerShell)

```powershell
powershell -ExecutionPolicy ByPass -c "irm https://raw.githubusercontent.com/poppy-team/prumo/main/scripts/install.ps1 | iex"
```

## First Project Walkthrough

1. **Initialize the repository**:
   ```bash
   prumo init ./my-app --profile ./examples/brasa/project-profile.json
   ```

2. **Create your first Goal**:
   ```bash
   prumo goal new P01-G01 "Foundation" --phase P01 --objective "Set up core data structures"
   ```

3. **Lock the Goal**:
   ```bash
   prumo goal state P01-G01 LOCKED
   ```

4. **Run health diagnostics**:
   ```bash
   prumo doctor
   ```
