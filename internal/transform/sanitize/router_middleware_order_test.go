package sanitize

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

// Ensure multiple middlewares order is preserved when sanitizing router.
func TestRouterMiddlewareOrderPreserved(t *testing.T) {
	raw := json.RawMessage("{\"entryPoints\":[\"web\"],\"rule\":\"Path(``/``)\",\"service\":\"svc\",\"middlewares\":[\"mw1\",\"mw2\",\"mw3\"]}")
	mappings := &nameMappings{middlewares: map[string]string{"mw1": "mw1", "mw2": "mw2", "mw3": "mw3"}, services: map[string]string{"svc": "svc"}}
	out, err := sanitizeRouterRaw(raw, mappings)
	require.NoError(t, err)
	var obj map[string]interface{}
	require.NoError(t, json.Unmarshal(out, &obj))
	mws, ok := obj["middlewares"].([]interface{})
	if !ok || mws == nil {
		t.Fatalf("middlewares field missing or invalid: %#v", obj["middlewares"])
	}
	require.Equal(t, []interface{}{"mw1", "mw2", "mw3"}, mws)
}
