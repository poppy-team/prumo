# Trust & Security Model

Prumo operates under a model of **least privilege and strict isolation**. Autonomous agents receive only the permissions explicitly granted by the project profile.

## Trust Levels

1. **Canonical Repository (Highest Level)**: Files versioned in Git with intact hashes. No AI can mutate these files without passing gate validation.
2. **Local Development Environment**: The human user holds supreme authority and can revoke or abort any harness execution at any time.
3. **AI Agents (Restricted Level)**: Run in directive sandboxes, with no access to secrets or production environment variables, and with commands restricted to the profile's allowlist.
4. **External Sources & Web (Untrusted)**: Any data retrieved from the internet or from third-party packages is considered untrusted until explicitly validated for compliance.
