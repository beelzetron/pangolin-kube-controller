<!-- markdownlint-disable MD013 MD033 -->
# Build and Release Notes

This document explains the pinned Go version, multi-architecture image builds, and the release tagging flow.

## Overview

- Primary image registry: `ghcr.io/fosrl/pangolin-kube-controller`
- Alternative image registry: `docker.io/fosrl/pangolin-kube-controller`
- Default target platforms: linux/amd64, linux/arm64
- Tags: builds always push versioned tags (e.g., `0.1.0-alpha.1`). The `:latest` tag is pushed only when the release process requests it (for example via the release script flag `--publish-latest`) — `image:buildx` examples that include `-t ...:latest` are conditional and used only when publishing the `latest` tag.

## Go Version

- The module requires Go 1.26.1 (`go 1.26.1` in `go.mod`).
- The Docker build uses the official golang image for multi-arch builds.
- Using a newer Go toolchain than the module's `go` directive is supported in CI via `GOTOOLCHAIN=auto`.

## Dockerfile Multi-Arch Support

- Build stage uses BuildKit cross-compilation variables:
  - `FROM --platform=$BUILDPLATFORM golang@sha256:...` with pinned digest
  - `ARG TARGETOS TARGETARCH` and `CGO_ENABLED=0`
  - `go build` with `GOOS=$TARGETOS`, `GOARCH=$TARGETARCH`
- Runtime stage uses `gcr.io/distroless/static-debian12` for security.
- Image label `org.opencontainers.image.version` is populated from a build arg `VERSION`; it is not hardcoded.

## Prerequisites

- Docker with buildx enabled
- Git
- Task (see <https://taskfile.dev>)
- For container signing: cosign

## Quick Start (Multi-Arch Build and Release)

1. Ensure buildx is available and bootstrap a builder:

   ```bash
   task buildx:setup
   ```

2. Build and push multi-arch images:

   ```bash
   task image:buildx VERSION=0.1.0-alpha.1
   ```

3. Create and push a matching git tag:

   ```bash
   git tag -a v0.1.0-alpha.1 -m "Release v0.1.0-alpha.1"
   git push origin v0.1.0-alpha.1
   ```

   This triggers the [release workflow](../.github/workflows/release.yml) which builds and publishes images.

## What the Tasks Do

- `buildx:setup` — Uses or creates a builder named `pangolinx`
- `image:buildx` — `docker buildx build --platform <PLATFORMS> -t <IMAGE_DH>:<VERSION> [ -t <IMAGE_DH>:latest ] -t <IMAGE_GH>:<VERSION> [ -t <IMAGE_GH>:latest ] --push --build-arg VERSION=<VERSION>`
- Note: `-t ...:latest` entries are conditional; include them only when publishing the `latest` tag (e.g., release with `--publish-latest` or for non-prerelease releases).
- `image:local` — Builds a single-arch image locally without pushing

## Notes

- `VERSION` must be provided when running release tasks; nothing is hardcoded in the Taskfile.
- For alpha releases, use semantic version with pre-release suffix (e.g., `0.1.0-alpha.1`).
- The controller's security posture (non-root distroless runtime) is independent from other Pangolin-related services.
