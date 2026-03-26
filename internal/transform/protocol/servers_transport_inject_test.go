package protocol

import (
	"testing"

	"pangolin-kube-controller/internal/transform/sanitize"
)

// TestInjectServersTransport ensures serversTransport field is added when structure present.
func TestInjectServersTransport(t *testing.T) {
	u := map[string]interface{}{
		"spec": map[string]interface{}{
			"routes": []interface{}{map[string]interface{}{
				"services": []interface{}{map[string]interface{}{"name": "svc", "port": float64(1234)}},
			}},
		},
	}
	injectServersTransport(u, "my-transport")
	specRaw, ok := u["spec"]
	if !ok {
		t.Fatalf("spec key missing root=%#v", u)
	}
	spec, ok := specRaw.(map[string]interface{})
	if !ok {
		t.Fatalf("spec value not map, got %T %#v", specRaw, specRaw)
	}
	routesRaw, ok := spec["routes"]
	if !ok {
		t.Fatalf("routes key missing spec=%#v", spec)
	}
	routes, ok := routesRaw.([]interface{})
	if !ok {
		t.Fatalf("routes not []interface{}, got %T %#v", routesRaw, routesRaw)
	}
	if len(routes) == 0 {
		t.Fatalf("routes empty")
	}
	firstRaw := routes[0]
	first, ok := firstRaw.(map[string]interface{})
	if !ok {
		t.Fatalf("first route not map, got %T %#v", firstRaw, firstRaw)
	}
	svcsRaw, ok := first["services"]
	if !ok {
		t.Fatalf("services key missing first=%#v", first)
	}
	svcs, ok := svcsRaw.([]interface{})
	if !ok {
		t.Fatalf("services not []interface{}, got %T %#v", svcsRaw, svcsRaw)
	}
	if len(svcs) == 0 {
		t.Fatalf("services empty")
	}
	entryRaw := svcs[0]
	entry, ok := entryRaw.(map[string]interface{})
	if !ok {
		t.Fatalf("first service entry not map, got %T %#v", entryRaw, entryRaw)
	}
	vRaw, ok := entry["serversTransport"]
	if !ok {
		t.Fatalf("serversTransport key missing entry=%#v", entry)
	}
	v, ok := vRaw.(string)
	if !ok || v == "" {
		t.Fatalf("serversTransport not string/non-empty: %T %#v", vRaw, vRaw)
	}
	expected := sanitize.SanitizeResourceName("my-transport")
	if v != expected {
		t.Fatalf("serversTransport value mismatch: got %q want %q", v, expected)
	}
}
