# Container Image — Verification Checklist

## 1. Minimal Reproducible Build
- [ ] Base images pinned by digest, never by floating tag (`latest`, `alpine` alone are absent).
- [ ] Multi-stage build: toolchain confined to the build stage; final stage holds only the binary and required assets.
- [ ] Two consecutive `docker build --no-cache` runs from the same context produce identical image digests (`SOURCE_DATE_EPOCH=1726358400` set).
- [ ] Final image size is within budget (Go API at most 25 MB) with layer count justified in the spec.

## 2. Non-Root Execution & Filesystem Hardening
- [ ] Container runs as numeric non-root user (`USER 65532:65532`); `docker run --rm image id` shows `uid=65532`.
- [ ] No process inside the final image runs as UID 0; no `sudo`, `su`, or setuid binaries present.
- [ ] Root filesystem is read-only; writable mounts exist only at proven paths (`/tmp`, `/var/run/app`).
- [ ] All Linux capabilities dropped (`--cap-drop=ALL`); any added capability has a documented syscall justification.

## 3. Health Checks, Signals & Shutdown
- [ ] `HEALTHCHECK` uses exec form with 15 s interval, 3 s timeout, 3 retries, backed by a real dependency-touching probe.
- [ ] `ENTRYPOINT` uses exec form so PID 1 receives SIGTERM directly (no shell wrapper swallowing signals).
- [ ] `docker stop -t 30` completes without SIGKILL; in-flight requests drain within the 25-second handler budget.
- [ ] Liveness and readiness semantics are distinct: liveness restarts deadlocks, readiness gates traffic during warmup.

## 4. Vulnerability & Supply-Chain Gates
- [ ] `trivy image --severity HIGH,CRITICAL` reports zero findings at ship time.
- [ ] `syft` SBOM generated and attached to the image/artifact; cosign signature present (keyless OIDC or named key).
- [ ] No compiler, shell, curl, or package manager exists in the final stage (`docker run --rm image which sh` fails).
- [ ] CVE waivers, if any, carry expiry dates and named owners; none cover CRITICAL findings silently.

## 5. Resources & Evidence
- [ ] Memory (512 MiB), CPU (1.0), and ephemeral storage (1 GiB) requests/limits documented and load-tested.
- [ ] Image specification follows `templates/containers-spec.md` with digests, sizes, and scan results.
- [ ] OOM-kill and CPU-throttle behavior observed under 1.5× load without data corruption.
- [ ] Automated verification script `scripts/verify.sh` executes with exit code 0.
