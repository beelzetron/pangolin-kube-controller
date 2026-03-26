package testutil

import (
	"encoding/json"
	"testing"

	traefikconfig "pangolin-kube-controller/internal/transform/config"
)

const errUnexpected = "unexpected error: %v"

func TestToUnstructuredFromRawSuccessAndError(t *testing.T) {
	raw := []byte(`{"apiVersion":"v1","kind":"ConfigMap","metadata":{"name":"x"}}`)
	u, err := ToUnstructuredFromRaw(raw)
	if err != nil {
		t.Fatalf(errUnexpected, err)
	}
	if u.GetKind() != "ConfigMap" || u.GetAPIVersion() != "v1" || u.GetName() != "x" {
		t.Fatalf("unexpected object: kind=%s apiVersion=%s name=%s", u.GetKind(), u.GetAPIVersion(), u.GetName())
	}

	// Invalid JSON should error
	if _, err := ToUnstructuredFromRaw([]byte("{")); err == nil {
		t.Fatalf("expected error for invalid JSON")
	}
}

func TestToUnstructuredFromMapFallbackMissingAPIVersionKind(t *testing.T) {
	m := map[string]interface{}{
		"metadata": map[string]interface{}{"name": "y"},
		"spec":     map[string]interface{}{"foo": "bar"},
	}
	u, err := ToUnstructuredFromMap(m)
	if err != nil {
		t.Fatalf(errUnexpected, err)
	}
	// Fallback sets Object directly when apiVersion/kind are missing
	if u.Object == nil {
		t.Fatalf("fallback Object not set correctly: %#v", u.Object)
	}
	specAny, ok := u.Object["spec"]
	if !ok {
		t.Fatalf("fallback Object missing spec: %#v", u.Object)
	}
	spec, ok := specAny.(map[string]interface{})
	if !ok {
		t.Fatalf("fallback Object spec not an object: %#v", specAny)
	}
	fooAny, ok := spec["foo"]
	if !ok {
		t.Fatalf("fallback Object spec missing foo: %#v", spec)
	}
	foo, ok := fooAny.(string)
	if !ok {
		t.Fatalf("fallback Object spec.foo not a string: %#v", fooAny)
	}
	if foo != "bar" {
		t.Fatalf("fallback Object not set correctly: %#v", u.Object)
	}

	// Fully specified map should populate structured fields
	obj := map[string]interface{}{
		"apiVersion": traefikconfig.GroupVersion,
		"kind":       "Middleware",
		"metadata":   map[string]interface{}{"name": "m1"},
		"spec":       map[string]interface{}{"headers": map[string]interface{}{"customRequestHeaders": map[string]interface{}{"X-Env": "dev"}}},
	}
	u, err = ToUnstructuredFromMap(obj)
	if err != nil {
		t.Fatalf(errUnexpected, err)
	}
	if u.GetKind() != "Middleware" || u.GetAPIVersion() != traefikconfig.GroupVersion || u.GetName() != "m1" {
		t.Fatalf("unexpected object: kind=%s apiVersion=%s name=%s", u.GetKind(), u.GetAPIVersion(), u.GetName())
	}
}

func TestBuildTraefikObject(t *testing.T) {
	spec := json.RawMessage(`{"headers":{"customRequestHeaders":{"X-Env":"prod"}}}`)
	u, err := BuildTraefikObject("Middleware", "m2", "default", spec)
	if err != nil {
		t.Fatalf(errUnexpected, err)
	}
	if u.GetKind() != "Middleware" || u.GetAPIVersion() != traefikconfig.GroupVersion || u.GetName() != "m2" || u.GetNamespace() != "default" {
		t.Fatalf("unexpected object fields: kind=%s apiVersion=%s name=%s ns=%s", u.GetKind(), u.GetAPIVersion(), u.GetName(), u.GetNamespace())
	}
}
