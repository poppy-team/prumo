# ADR 006: Container Runtime Default (Docker/Podman, Rootless)

# Status

Accepted

# Context

- The sandbox ladder exposes container execution with memory/CPU/PID limits
  and no network (`internal/harness/aci/container.go`).
- `DetectContainerRuntime` prefers podman and falls back to docker; bogus
  runtimes report unavailable instead of pretending (honest detection).
- GAP-037 closed 2026-09-14: `TestContainerLive` passed against a real
  Docker daemon (`alpine:latest`, `--network none`, limits applied).
- Podman was not installed on the dev host; rootless operation is the
  framework preference (least privilege), but no live measurement exists.

# Decision

1. Keep podman-preferred detection: when podman is present, it is used
   first; docker is the tested fallback.
2. Docker is the reference runtime for live proofs and CI-style verification
   until a rootless Podman environment is measured.
3. Rootless is mandated when the detected runtime supports it: the sandbox
   layer must not require a privileged daemon where rootless mode exists.
4. Runtime choice is an operator flag (`--sandbox-runtime`), never a silent
   fallback: an unavailable requested runtime fails fast.

# Consequences

- Live evidence exists for Docker; Podman parity gets proven when a rootless
  environment is available (already wired by detection order).
- Strong isolation (gVisor/runsc) remains a later rung on the ladder
  (GAP-024), not part of this default.
- No silent host fallback: container refusal stays a typed, visible error.
