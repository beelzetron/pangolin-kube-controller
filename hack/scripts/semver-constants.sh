#!/usr/bin/env bash
# Shared SemVer regex pattern used across release automation tooling.
#
# IMPORTANT: This regex is used with Bash's `[[ string =~ regex ]]`.
# It intentionally contains capture groups relied upon by `hack/scripts/release.sh`:
#   1: major, 2: minor, 3: patch, 4: prerelease (including leading '-'), 7: build metadata (including leading '+')
#
# Supported examples:
#   1.2.3
#   1.2.3-rc.1
#   1.2.3+build.42
#   1.2.3-rc.1+build.42

export SEMVER_REGEX='^([0-9]+)\.([0-9]+)\.([0-9]+)(-([0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*))?(\+([0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*))?$'
