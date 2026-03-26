package routing

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	tst "pangolin-kube-controller/internal/testutil"
)

func TestIngressRouteEntryPointsAnnotationSingle(t *testing.T) {
	raw := json.RawMessage([]byte(`{
		"entryPoints": ["websecure"],
		"rule": "Host(\"example.com\")",
		"service": "svc1"
	}`))
	u, err := TransformRouterToIngressRoute("router1", raw, RouterConfig{
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
	val, ok := anns[RouterEntryPointsAnnotation].(string)
	require.True(t, ok, "entryPoints annotation should be a string")
	require.Equal(t, "websecure", val)
}
