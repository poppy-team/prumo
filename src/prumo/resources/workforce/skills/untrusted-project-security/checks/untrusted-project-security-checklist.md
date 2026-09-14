# Untrusted Codebase & Repository Quarantine Checklist

- [ ] Foreign repository treated as untrusted; automatic builds and tests suspended
- [ ] Static analysis conducted with zero network egress permitted
- [ ] Hidden execution hooks (.git/hooks, VS Code tasks, devcontainers) scanned
- [ ] High-entropy blobs and obfuscated scripts flagged for inspection
- [ ] Exploration conducted inside ephemeral, read-only sandboxed containers
- [ ] Dependencies inspected statically without executing package managers
- [ ] Formal clearance report generated before repository is approved for work
