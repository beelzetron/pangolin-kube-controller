<!-- markdownlint-disable MD024 -->
# Changelog

All notable changes to this project are documented in this file.

The format is inspired by Keep a Changelog and based on Conventional Commits from the repository history.

## [Unreleased]

### Added

- Added certificate secret configuration parsing to support certificate-driven runtime behavior.
- Added a certificates endpoint and handler.

### Changed

- Simplified controller leadership, garbage-collection, and readiness handling.
- Simplified Kubernetes client construction and related tests.
- Improved CI artifact metadata handling and Go tool caching.

### Documentation

- Refined skill documentation and near-completion checklist guidance.
- Updated the Go files overview with missing entries.

## [0.1.0-alpha.1] - 2026-04-13

### Added

- Initial project scaffold and baseline controller implementation.
- Added a read-only check in applyDesiredObjects to improve reconciliation safety.

### Fixed

- Fixed Trivy GitHub Action execution in CI.
- Improved linting behavior and release signing flow.
- Added conditional SBOM upload behavior in CI based on job output.
- Updated Go image digests in Docker images.
- Updated Go version and OpenTelemetry dependencies.

### Changed

- Prepared release workflow and project metadata for 0.1.0-alpha.1.
