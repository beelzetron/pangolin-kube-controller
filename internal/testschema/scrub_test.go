package testschema

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestScrubObjectMetaNilAndMissing(t *testing.T) {
	require.Nil(t, ScrubObjectMeta(nil), "nil input should return nil")
	obj := map[string]interface{}{"spec": map[string]interface{}{"x": 1}}
	// No metadata present => object returned unchanged
	got := ScrubObjectMeta(obj)
	if got == nil || got["metadata"] != nil {
		// metadata should remain absent
		if _, ok := got["metadata"]; ok && got["metadata"] != nil {
			require.Fail(t, "metadata should remain absent")
		}
	}
}

func TestScrubObjectMetaRemovesManagedFields(t *testing.T) {
	meta := map[string]interface{}{
		"resourceVersion":   "123",
		"uid":               "u-1",
		"generation":        7,
		"creationTimestamp": "2020-01-01T00:00:00Z",
		"managedFields":     []interface{}{map[string]interface{}{"a": 1}},
		"keep":              "ok",
	}
	obj := map[string]interface{}{"metadata": meta, "spec": map[string]interface{}{"x": 2}}
	got := ScrubObjectMeta(obj)
	m, ok := got["metadata"].(map[string]interface{})
	require.True(t, ok, "metadata should be a map")
	// Removed keys
	for _, k := range []string{"resourceVersion", "uid", "generation", "creationTimestamp", "managedFields"} {
		if _, exists := m[k]; exists {
			require.False(t, exists, "%s should have been removed", k)
		}
	}
	// Preserved other keys
	require.Equal(t, "ok", m["keep"])
}
