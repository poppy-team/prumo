# Container Image Specification Template

## 1. Image Identity & Base
- **Image**: `registry.example.com/billing-api:2.14.0`, digest `sha256:9c4e1a2b3d4e5f60718293a4b5c6d7e8f90a1b2c3d4e5f60718293a4b5c6d7e8f90a1`
- **Build stage base**: `golang:1.24.3-bookworm@sha256:2c5f8b1a` (toolchain, tests run here)
- **Final stage base**: `gcr.io/distroless/static-debian12@sha256:5f8d1e2a`
- **Build date**: 2026-09-12, `SOURCE_DATE_EPOCH=1726358400`
- **Final size**: 19.4 MB, 6 layers; budget was 25 MB

## 2. Build Reproducibility
- **Rebuild A digest**: `sha256:9c4e1a2b...7e8f90a1` (`docker build --no-cache`, 2026-09-12 14:02 UTC)
- **Rebuild B digest**: `sha256:9c4e1a2b...7e8f90a1` (`docker build --no-cache`, 2026-09-12 14:19 UTC)
- **Match**: identical; nondeterminism sources eliminated (timestamps pinned, deps vendored via go.sum)
- **Binary flags**: `CGO_ENABLED=0 go build -trimpath -ldflags="-s -w"`

## 3. Runtime Hardening
- **User**: `USER 65532:65532`; `docker run --rm image id` returns `uid=65532 gid=65532`
- **Capabilities**: `--cap-drop=ALL`, none added back
- **Filesystem**: read-only root; writable mounts at `/tmp` (scratch) and `/var/run/app` (sockets)
- **Exposed port**: 8080 only; no debug or metrics port in the ship image

## 4. Health, Signals & Shutdown
- **Health check**: `HEALTHCHECK --interval=15s --timeout=3s --retries=3 CMD ["/server", "-health-check"]`
- **Probe depth**: `-health-check` pings Postgres (2 s timeout) and verifies migration version 48
- **Shutdown**: exec-form ENTRYPOINT, SIGTERM drain 25 s within 30 s grace; `docker stop -t 30` exits 0 without SIGKILL
- **Liveness vs readiness**: liveness is process self-check; readiness requires DB ping plus warm caches

## 5. Vulnerability & Provenance
- **trivy image --severity HIGH,CRITICAL**: 0 HIGH, 0 CRITICAL (log 2026-09-12); 2 LOW (glibc locale data, accepted)
- **SBOM**: `sbom.spdx.json` (syft 1.27.0), 63 packages listed
- **Signature**: cosign keyless OIDC, identity `https://github.com/example/billing-api/.github/workflows/release.yaml@refs/tags/v2.14.0`
- **Waivers**: none open

## 6. Resource Envelope
- **Requests**: 256 MiB memory, 0.5 CPU; **limits**: 512 MiB memory, 1.0 CPU, 1 GiB ephemeral storage
- **Load test**: 1.5× peak (4,500 rps) for 10 min — p99 41 ms, zero OOM-kills, zero restarts
- **Startup**: cold start to ready in 2.8 s (migration check dominates)

## 7. Verification Evidence
- [ ] Rebuild digests A and B match (logs attached)
- [ ] trivy scan log with zero HIGH/CRITICAL attached
- [ ] Non-root `id` output and graceful-stop timing log attached
- [ ] `scripts/verify.sh` exits 0 (log attached)
