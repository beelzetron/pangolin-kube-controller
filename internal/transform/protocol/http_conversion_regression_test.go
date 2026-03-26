package protocol

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	testschema "pangolin-kube-controller/internal/testschema"
	traefikconfig "pangolin-kube-controller/internal/transform/config"
	"pangolin-kube-controller/internal/transform/sanitize"
)

// TestOlderHTTPScenario ensures the smaller/older fixture still converts for core HTTP kinds.
func TestOlderHTTPScenario(t *testing.T) {
	b, err := os.ReadFile(testschema.TestDataPath("traefik-configs", "v3.5.0", "older.json"))
	require.NoError(t, err)

	var cfg traefikconfig.Config
	require.NoError(t, json.Unmarshal(b, &cfg))
	_, err = sanitize.SanitizeTraefikConfig(&cfg)
	require.NoError(t, err)
}
