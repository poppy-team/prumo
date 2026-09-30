# Installing Prumo

Prumo is distributed as a single static executable compiled in Go, with no external runtime dependencies (it does not require Python, Node.js, or compilers to be installed).

## Quick Installation (Recommended)

### Linux & macOS

```bash
# Download the official v0.6.0 executable
curl -fsSL https://github.com/poppy-team/prumo/releases/latest/download/prumo-linux-amd64 -o /usr/local/bin/prumo
chmod +x /usr/local/bin/prumo

# Verify the installation
prumo version
```

### Global Environment Initialization

On first install, run `setup` to register the system manifest and detect AI tools on your `$PATH`:

```bash
prumo setup
```

The command will initialize the global `~/.prumo` directory, which holds the connector registry and the update cache.

---

## Integrity Verification and Diagnostics

To audit the host environment, the detected AI tools, and the executable's integrity:

```bash
prumo doctor
```

When run outside a project, `prumo doctor` analyzes the global environment; inside a project, it also analyzes the conformance of the repository's schemas and goals.

---

## Autoupdate (Continuous Updates)

Prumo includes a built-in self-update subsystem with SHA-256 cryptographic verification, pulling directly from the official GitHub releases:

```bash
# Check whether new versions are available
prumo upgrade --check

# Update to the latest version
prumo upgrade

# Update to a specific version
prumo upgrade --version v0.6.1
```

---

## Reversible and Safe Uninstallation

If you need to remove the global installation or reset caches:

```bash
# Remove global connectors and binaries
prumo uninstall

# Clear caches without affecting project data
prumo uninstall --purge-cache
```

> [!IMPORTANT]
> Uninstalling **never** mutates or removes local repository directories (`prumo.json`, `.ai/`, `docs/`, and goal files remain 100% intact).
