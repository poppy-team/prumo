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
- **Measured 2026-09-17 (Arch, podman 6.1.2, rootless, no daemon):** podman runs
  a container (`alpine:latest`, `--network none`, limits applied) and
  `TestContainerLive` passes against it in 3.66 s. The measurement also found a
  defect: the availability probe asked every runtime
  `info --format {{.ServerVersion}}`, and `ServerVersion` is a **Docker** field —
  `podman info` fails on it. So installing podman turned a PASS into a SKIP: the
  sandbox disappeared from a machine that could still run containers, because
  detection preferred the first runtime *present* rather than the first one
  *usable*. Both are fixed: each runtime is asked in its own language
  (`{{.Version.Version}}` for podman) and detection now walks the preference
  order until one answers.

# Decision

1. Keep podman-preferred detection: when podman is present, it is used
   first; docker is the tested fallback.
2. Docker is the reference runtime for CI-style verification; a rootless Podman
   environment is measured (2026-09-17, above) and passes the same live test, so
   both are reference runtimes and neither is assumed.
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
