# Uninstallation Manual

## One-Line Uninstall

Download and review the remover. `pure` mode removes only managed state and preserves all project files:

```bash
curl --fail --location https://raw.githubusercontent.com/raillen/prumo/main/scripts/uninstall.sh -o uninstall.sh
less uninstall.sh
sh uninstall.sh --mode pure --dry-run
sh uninstall.sh --mode pure
```

To also remove the executable found on `PATH`:

```bash
sh uninstall.sh --mode full --remove-binary --dry-run
sh uninstall.sh --mode full --remove-binary
```

## Safety Rule (Uninstall safety)

Uninstalling Prumo never removes project data.

The following remain untouched:

- `.ai/`;
- `prumo.json`;
- `docs/`;
- Goals, Plans, Tasks, Evidence;
- Git history;
- files created by the user.

Removing a project remains an explicit user decision, made with the operating system's and Git's normal tools.

## Basic Removal

```bash
prumo-agent uninstall
prumo-agent --home ~/.prumo uninstall
```

This command updates the global manifest and removes only installation state directly tracked by Prumo.

## Remove Connectors

```bash
prumo-agent uninstall --connectors
```

A connector is only removed when it has a cleanup manifest with paths created by Prumo itself. Paths outside the manifest, files modified by the user, or uncertain ownership are reported as leftovers instead of being deleted.

## Clear Cache and Derived State

```bash
prumo-agent uninstall --purge-cache
```

Removes the cache, local logs, derived databases, and temporary files under `PRUMO_HOME`. This command does not change the main manifest unless `--purge-global-config` is also used.

## Remove Global Configuration

```bash
prumo-agent uninstall --purge-global-config
```

Removes `PRUMO_HOME/config`. The next run creates a clean state through `prumo-agent setup`. Project data remains preserved.

## Complete Binary Removal

The CLI does not delete its own executable. Remove the installed binary using your system's normal procedure:

```bash
rm /usr/local/bin/prumo
```

Also remove the home directory, but only when it no longer contains anything you still need:

```bash
rm -rf ~/.prumo
```

First confirm that the directory does not contain secrets, keys, local backups, or runtime packages that are still needed.

## Post-Removal Verification

```bash
command -v prumo || echo "binary removed"
test ! -e ~/.prumo/config/installation.json && echo "global configuration removed"
git -C ./my-project status --short
```

The last command should show that the project remains functional and versioned, even without Prumo installed globally.

## Error Recovery

If a removal is interrupted:

1. Run `prumo-agent setup` in a new, temporary home.
2. Compare the generated manifests.
3. Restore only backups recorded in the cleanup manifest.
4. Never restore a backup over a file modified by the user without confirming its content and checksum.
