# Contributing

Thanks for your interest in the Pangolin Kubernetes Controller! This document
covers the workflow used by the maintainers, how to get your changes reviewed,
and how releases are produced.

## Branching Strategy

* `dev` is the default branch. All pull requests should target `dev`.
* `main` is protected and only receives fast-forward merges from tagged release
  branches.
* Feature work happens on topic branches prefixed with `feat/`, `fix/`, or
  similar descriptive names.

## Branch Protections

These rules are enforced on GitHub and should not be bypassed:

* **main**
  * Require pull requests with at least one approval
  * Require branches to be up to date with `main`
  * Require status checks: `ci`, `codeql`, `scorecard`
  * Disallow force pushes and deletions
* **dev**
  * Require pull requests with squash merges
  * Require status checks: `ci`, `commitlint`
  * Force pushes are restricted to maintainers only

## Commit Messages

We use [Conventional Commits](https://www.conventionalcommits.org/) for all
commits. This keeps the history clean and enables automated changelog
generation. Examples:

```text
feat: add readiness probe metrics
fix: handle 404 from Pangolin API
chore: update dependencies
```

## Pull Requests

1. Fork the repository and clone your fork.
2. Create a feature branch from `dev`.
3. Run the required checks locally: `task ci`.
4. Commit your changes with Conventional Commit messages.
5. Push the branch and open a pull request targeting `dev`.
6. Fill out the PR template and link to any relevant issues.
7. Address review feedback promptly.

## Release Process

1. Ensure `dev` is green and contains all desired changes.
2. Create a release branch: `release/vX.Y.Z` from `dev`.
3. Bump version numbers and update the changelog.
4. Open a PR from the release branch to `main`. Obtain approval.
5. After merging, tag the release (`vX.Y.Z`) from `main` and push the tag.
6. The release workflow builds and signs images, publishes the SBOM, and opens a
   PR in the Helm charts repository with the new image tag.
7. Merge the Helm chart PR once CI passes.

## Local Tooling

The `Taskfile.yml` is the source of truth for common tasks:

* `task lint` – run linters
* `task test` – run unit tests
* `task build` – build the controller binary
* `task tidy` – tidy Go modules
* `task ci` – run the full CI verification locally

### Go version and CI checks

* Go version: this repo follows the Go version declared in `go.mod` (currently 1.26.1). CI uses `actions/setup-go` with `go-version-file: go.mod` to ensure the same toolchain.
* Required CI checks (set as required on PRs where possible):
  * `go vet ./...`
  * `go test -race ./...`
  * `staticcheck ./...`
  * `gosec ./...` (security linter; configured to skip `internal/testschema` and with inline `// #nosec` where intentional)

For more details see [SUPPORT.md](SUPPORT.md) and [controller docs](docs/controller/controller.md).
