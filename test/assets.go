//go:build integration

package testassets

import (
	"embed"
	"io/fs"
	"os"
	"path/filepath"
)

// Embedded CRDs for Traefik v3.5.1.
// The files live under test/crds/traefik/3.5.1.
// We embed them from this package located at test/ to satisfy //go:embed rules.
//
//go:embed crds/traefik/3.5.1/*.yaml
var crdsFS embed.FS

// WriteCRDsToTempDir writes the embedded CRDs to a temporary directory and returns its path.
func WriteCRDsToTempDir() (string, error) {
	dir, err := os.MkdirTemp("", "traefik-crds-")
	if err != nil {
		return "", err
	}
	// Walk embedded files and materialize them on disk.
	err = fs.WalkDir(crdsFS, ".", func(path string, d fs.DirEntry, werr error) error {
		if werr != nil {
			return werr
		}
		if d.IsDir() {
			return nil
		}
		// Only copy YAMLs.
		if filepath.Ext(path) != ".yaml" && filepath.Ext(path) != ".yml" {
			return nil
		}
		b, rerr := crdsFS.ReadFile(path)
		if rerr != nil {
			return rerr
		}
		out := filepath.Join(dir, filepath.Base(path))
		return os.WriteFile(out, b, 0o644)
	})
	if err != nil {
		_ = os.RemoveAll(dir)
		return "", err
	}
	return dir, nil
}
