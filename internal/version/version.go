package version

import "fmt"

var (
	// Version is the semantic version string of the build, set via -ldflags at
	// build time (defaults to "dev").
	Version = "dev"
	// Commit is the VCS commit SHA associated with the build (defaults to "none").
	Commit = "none"
	// Date is the build timestamp in RFC3339 or similar string form
	// (defaults to "unknown").
	Date = "unknown"
)

// buildStringFormat defines the canonical format used by Info.String and the
// package-level String() implementation. It documents the production contract
// for formatting build metadata so callers and tests rely on a single, well-
// defined representation.
const buildStringFormat = "Version=%s Commit=%s Date=%s"

// Info captures build metadata for the controller.
type Info struct {
	Version string
	Commit  string
	Date    string
}

// Get returns an Info populated with the package-level build metadata: Version, Commit, and Date.
func Get() Info {
	return Info{Version: Version, Commit: Commit, Date: Date}
}

// String returns a concise human-readable build string.
func (i Info) String() string {
	return fmt.Sprintf(buildStringFormat, i.Version, i.Commit, i.Date)
}

// String returns the package build information in a concise, human-readable form.
// The value is formatted like "Version=<version> Commit=<commit> Date=<date>".
func String() string { return Get().String() }
