package e2e

import (
	"os"
	"path/filepath"
	"strings"
)

// TraefikVersion returns the CRD version to use, falling back to defaultTag when env is unset.
func TraefikVersion(defaultTag string) string {
	// Read the environment and trim whitespace. If the trimmed value is empty
	// treat it as unset and return the provided defaultTag. Returning raw
	// whitespace is unsafe and surprising.
	if v := strings.TrimSpace(os.Getenv("TRAEFIK_CRD_VERSION")); v != "" {
		return v
	}
	return defaultTag
}

// UpdateGoldenEnabled indicates whether golden files should be updated based on env flag.
func UpdateGoldenEnabled() bool { return os.Getenv("UPDATE_GOLDEN") == "1" }

// GoldenPath returns the path to the golden YAML for the given tag.
func GoldenPath(tag string) string {
	return filepath.Join("test", "golden", tag, "extended_e2e_all.yaml")
}
