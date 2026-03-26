package routing

import (
	"encoding/json"
	"testing"

	tst "pangolin-kube-controller/internal/testutil"

	"github.com/stretchr/testify/require"
)

// If router has no priority, transform should not set a priority field.

func TestRouterPriorityDefaulting(t *testing.T) {
	// Rule uses standard Traefik syntax Path(`/`) with a single slash ("/")
	// instead of a double slash ("//") — the previous double-slash caused parsing issues.
	raw := json.RawMessage("{\"entryPoints\":[\"web\"],\"rule\":\"Path(`/`)\",\"service\":\"svc\"}")
	u, err := TransformRouterToIngressRoute("r1", raw, RouterConfig{
		Namespace:         tst.TestNamespace,
		ManagedLabelKey:   tst.ManagedLabelKey,
		ManagedLabelValue: tst.ManagedLabelValue,
		ManagedAnnoKey:    tst.ManagedAnnoKey,
		ManagedAnnoValue:  tst.ManagedAnnoValue,
		IngressClass:      tst.DefaultIngressClass,
	})
	require.NoError(t, err)
	spec, ok := u["spec"].(map[string]interface{})
	require.True(t, ok, "spec should be a map")
	routes, ok := spec["routes"].([]interface{})
	require.True(t, ok, "routes should be a slice")
	require.Len(t, routes, 1)
	r0, ok := routes[0].(map[string]interface{})
	require.True(t, ok, "route should be a map")
	_, has := r0["priority"]
	require.False(t, has)
}

func TestRouterTransformIncludesEntryPointsAnnotation(t *testing.T) {
	raw := json.RawMessage("{\"entryPoints\":[\"websecure\"],\"rule\":\"Path(`/`)\",\"service\":\"svc\"}")
	u, err := TransformRouterToIngressRoute("r2", raw, RouterConfig{
		Namespace:         tst.TestNamespace,
		ManagedLabelKey:   tst.ManagedLabelKey,
		ManagedLabelValue: tst.ManagedLabelValue,
		ManagedAnnoKey:    tst.ManagedAnnoKey,
		ManagedAnnoValue:  tst.ManagedAnnoValue,
		IngressClass:      tst.DefaultIngressClass,
	})
	require.NoError(t, err)
	meta, ok := u["metadata"].(map[string]interface{})
	require.True(t, ok, "metadata should be a map")
	anns, ok := meta["annotations"].(map[string]interface{})
	require.True(t, ok, "annotations should be a map")
	v, ok := anns[RouterEntryPointsAnnotation]
	require.True(t, ok)
	require.Equal(t, "websecure", v)
}
