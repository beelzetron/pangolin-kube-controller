package testutil

import (
	"encoding/json"
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	traefikconfig "pangolin-kube-controller/internal/transform/config"
)

// ToUnstructuredFromMap marshals the provided map and unmarshals it into an Unstructured.
// If the map doesn't contain apiVersion/kind (minimal test objects), the Object
// field is filled directly as a graceful fallback to preserve older test behavior.
func ToUnstructuredFromMap(m map[string]interface{}) (*unstructured.Unstructured, error) {
	b, err := json.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("marshal map: %w", err)
	}
	var u unstructured.Unstructured
	if err := u.UnmarshalJSON(b); err != nil {
		// If missing apiVersion/kind in the map, fall back to setting Object directly.
		if _, hasAP := m["apiVersion"]; !hasAP {
			u.Object = m
			return &u, nil
		}
		if _, hasKind := m["kind"]; !hasKind {
			u.Object = m
			return &u, nil
		}
		return nil, fmt.Errorf("unmarshal JSON: %w; raw: %s", err, string(b))
	}
	return &u, nil
}

// ToUnstructuredFromRaw unmarshals raw JSON bytes into an Unstructured.
func ToUnstructuredFromRaw(raw []byte) (*unstructured.Unstructured, error) {
	var u unstructured.Unstructured
	if err := u.UnmarshalJSON(raw); err != nil {
		return nil, fmt.Errorf("unmarshal raw JSON: %w; raw: %s", err, string(raw))
	}
	return &u, nil
}

// BuildTraefikObject builds a Traefik CRD-like unstructured object from a spec raw message.
func BuildTraefikObject(kind, name, ns string, specRaw json.RawMessage) (*unstructured.Unstructured, error) {
	var spec map[string]interface{}
	if err := json.Unmarshal(specRaw, &spec); err != nil {
		return nil, fmt.Errorf("unmarshal spec raw: %w; raw: %s", err, string(specRaw))
	}
	obj := map[string]interface{}{
		"apiVersion": traefikconfig.GroupVersion,
		"kind":       kind,
		"metadata": map[string]interface{}{
			"name":      name,
			"namespace": ns,
		},
		"spec": spec,
	}
	return ToUnstructuredFromMap(obj)
}
