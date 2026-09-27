# Container Image Design & Hardening Reference Guide

## 1. Core Concepts

### 1.1 Layer Economics
Image size and pull time follow:

$$\text{Pull time} = \frac{\text{compressed bytes}}{\text{bandwidth}} + \text{decompression} + \text{per-layer overhead} \times N_{\text{layers}}$$

Every layer costs a round trip. Multi-stage builds collapse the toolchain out of the
shipped artifact: a Go service that needs 1.1 GB of build image ships a 19 MB final
image. Fewer, denser layers in the final stage beat many fine-grained ones.

### 1.2 Least Privilege in User Namespaces
A container root (UID 0) with a kernel escape becomes host root. A numeric non-root
user (`65532`) with `cap-drop=ALL` and a read-only root filesystem reduces a breakout
to an unprivileged foothold with nowhere to write. Numeric UIDs (not names) survive
any base image's `/etc/passwd` contents.

### 1.3 Health Semantics: Liveness vs Readiness
- **Liveness**: "is this process deadlocked?" Restart on failure. Cheap self-check.
- **Readiness**: "can this instance serve traffic?" Probe dependencies (DB ping,
  migration version). A pod that is alive but not ready must not receive traffic.
- A single endpoint returning unconditional 200 conflates both and hides outages.

### 1.4 Reproducible Builds
Two builds from identical inputs must yield identical digests. Nondeterminism creeps
in via timestamps (fix with `SOURCE_DATE_EPOCH`), network fetches at build time
(vendor or lock dependencies first), and unordered file copies (sort inputs, use
explicit `COPY` lists). Verify with two `--no-cache` builds and `diff` of digests.

## 2. Patterns and Anti-Patterns

**Do: static binary plus distroless final.**
`CGO_ENABLED=0 go build -trimpath -ldflags="-s -w"` into
`gcr.io/distroless/static-debian12` gives a 19 MB image with no shell to exploit.

**Do: exec-form ENTRYPOINT.**
`ENTRYPOINT ["/server"]` makes the app PID 1 so SIGTERM arrives directly and the
25-second drain actually runs. Shell form (`ENTRYPOINT /server`) orphans signals.

**Do not: `FROM ubuntu:latest` plus `apt-get install` in the final stage.**
That ships 80 MB of OS, a package manager, and monthly CVE churn for a static binary
that needs none of it.

**Do not: bake secrets with `ARG`/`ENV`.**
Build args persist in image history (`docker history` shows them). Inject runtime
secrets via mounted files or environment from the orchestrator, never into layers.

**Do not: `HEALTHCHECK CMD curl -f http://localhost:8080/` in a distroless image.**
There is no curl in distroless by design. Compile a `-health-check` subcommand into the binary instead.

## 3. Dockerfile Configuration Example

```dockerfile
# syntax=docker/dockerfile:1.9
# Stage 1 — toolchain, locked deps, tests before compile
FROM golang:1.24.3-bookworm@sha256:2c5f8b1a9e4d7c0a3f6b2e8d1c4a5b6e7f809102132435465768798a9b0c1d2e3 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download && go mod verify
COPY . .
RUN go vet ./... && go test ./... -count=1
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

# Stage 2 — minimal runtime, non-root, real health check
FROM gcr.io/distroless/static-debian12@sha256:5f8d1e2a3b4c5d6e7f809102132435465768798a9b0c1d2e3f4051627384950617
COPY --from=build /out/server /server
USER 65532:65532
EXPOSE 8080
HEALTHCHECK --interval=15s --timeout=3s --retries=3 CMD ["/server", "-health-check"]
ENTRYPOINT ["/server"]
```

Build, scan, and sign:

```bash
export SOURCE_DATE_EPOCH=1726358400
docker build --no-cache -t registry.example.com/billing-api:2.14.0 .
trivy image --severity HIGH,CRITICAL registry.example.com/billing-api:2.14.0
syft registry.example.com/billing-api:2.14.0 -o spdx-json > sbom.spdx.json
cosign sign --yes registry.example.com/billing-api:2.14.0
```
