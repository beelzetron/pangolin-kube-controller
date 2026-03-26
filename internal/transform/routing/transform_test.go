package routing

import (
	"encoding/json"
	"fmt"
	"testing"

	tst "pangolin-kube-controller/internal/testutil"

	"github.com/stretchr/testify/require"
)

func TestTransformRouterToIngressRouteBasic(t *testing.T) {
	raw := json.RawMessage([]byte(`{
		"entryPoints": ["web"],
		"rule": "Host(\"example.com\")",
		"service": "svc1",
		"middlewares": ["mw1"],
		"priority": 5,
		"tls": {"options":"default"}
	}`))
	u, err := TransformRouterToIngressRoute("r1", raw, RouterConfig{
		Namespace:         tst.TestNamespace,
		ManagedLabelKey:   tst.ManagedLabelKey,
		ManagedLabelValue: tst.ManagedLabelValue,
		ManagedAnnoKey:    tst.ManagedAnnoKey,
		ManagedAnnoValue:  tst.ManagedAnnoValue,
		IngressClass:      tst.DefaultIngressClass,
	})
	require.NoError(t, err)

	// Basic shape
	require.Equal(t, TraefikAPIVersion, u["apiVersion"])
	require.Equal(t, KindIngressRoute, u["kind"])
	meta, ok := u["metadata"].(map[string]interface{})
	require.True(t, ok, "metadata should be a map")
	require.Equal(t, "r1", meta["name"])
	require.Equal(t, tst.TestNamespace, meta["namespace"])

	spec, ok := u["spec"].(map[string]interface{})
	require.True(t, ok, "spec should be a map")
	// entryPoints may decode as []string; normalize to []interface{} for assertion
	var epsNorm []interface{}
	entryPointsRaw, ok := spec["entryPoints"]
	require.True(t, ok, "entryPoints should be present")
	switch v := entryPointsRaw.(type) {
	case []interface{}:
		epsNorm = v
	case []string:
		for _, s := range v {
			epsNorm = append(epsNorm, s)
		}
	default:
		t.Fatalf("entryPoints unexpected type %T", v)
	}
	require.Contains(t, epsNorm, "web")
	routes, ok := spec["routes"].([]interface{})
	require.True(t, ok, "routes should be a slice")
	require.Len(t, routes, 1)
	r, ok := routes[0].(map[string]interface{})
	require.True(t, ok, "route should be a map")
	require.Equal(t, "Rule", r["kind"])
	require.Equal(t, "Host(\"example.com\")", r["match"])

	refs, ok := r["services"].([]interface{})
	require.True(t, ok, "services should be a slice")
	require.Len(t, refs, 1)
	ref, ok := refs[0].(map[string]interface{})
	require.True(t, ok, "service ref should be a map")
	require.Equal(t, "TraefikService", ref["kind"])
	require.Equal(t, "svc1", ref["name"])

	mwRefs, ok := r["middlewares"].([]interface{})
	require.True(t, ok, "middlewares should be a slice")
	require.Contains(t, mwRefs, map[string]interface{}{"name": "mw1"})
}

func TestTransformRouterToIngressRouteErrors(t *testing.T) {
	cases := []json.RawMessage{
		[]byte(`{"service":"svc"}`),
		[]byte("{\"rule\":\"Path(`/`)\"}"),
	}
	for i, raw := range cases {
		name := fmt.Sprintf("case%d_%s", i, string(raw))
		t.Run(name, func(t *testing.T) {
			_, err := TransformRouterToIngressRoute("r1", raw, RouterConfig{Namespace: tst.TestNamespace})
			require.Error(t, err)
		})
	}
}
