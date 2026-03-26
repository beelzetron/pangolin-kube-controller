package routing

import "testing"

func TestAnnotateRouterEntryPointsIfPresentNoSpec(t *testing.T) {
	u := map[string]interface{}{"apiVersion": TraefikAPIVersion, "kind": KindIngressRoute, "metadata": map[string]interface{}{"name": "n"}}
	meta := map[string]interface{}{}
	AnnotateRouterEntryPointsIfPresent(u, meta)
	if _, ok := meta["annotations"].(map[string]interface{}); ok {
		t.Fatalf("expected no annotations when entryPoints absent")
	}
}
