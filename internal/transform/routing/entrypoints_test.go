package routing

import (
	"reflect"
	"testing"
)

func TestExtractEntryPoints(t *testing.T) {
	// nil spec
	if extractEntryPoints(nil) != nil {
		t.Fatalf("expected nil for nil spec")
	}

	// []interface{}
	spec := map[string]interface{}{"entryPoints": []interface{}{" http ", "", "   ", "\t", "\n", "foo"}}
	if !reflect.DeepEqual(extractEntryPoints(spec), []string{"http", "foo"}) {
		t.Fatalf("unexpected result when filtering empty/whitespace: %#v", extractEntryPoints(spec))
	}

	// []string
	spec = map[string]interface{}{"entryPoints": []string{" a ", "b"}}
	if !reflect.DeepEqual(extractEntryPoints(spec), []string{"a", "b"}) {
		t.Fatalf("unexpected result for []string: %#v", extractEntryPoints(spec))
	}

	// wrong type
	spec = map[string]interface{}{"entryPoints": "notalist"}
	if len(extractEntryPoints(spec)) != 0 {
		t.Fatalf("expected empty for wrong type")
	}
	// nil slice ([]string(nil))
	spec = map[string]interface{}{"entryPoints": []string(nil)}
	if len(extractEntryPoints(spec)) != 0 {
		t.Fatalf("expected empty for nil []string slice")
	}

	// nil slice ([]interface{}(nil))
	spec = map[string]interface{}{"entryPoints": []interface{}(nil)}
	if len(extractEntryPoints(spec)) != 0 {
		t.Fatalf("expected empty for nil []interface{} slice")
	}

	// mixed valid/invalid types in []interface{}
	spec = map[string]interface{}{"entryPoints": []interface{}{"a", 123, nil, "b", true, "c"}}
	if !reflect.DeepEqual(extractEntryPoints(spec), []string{"a", "b", "c"}) {
		t.Fatalf("unexpected result for mixed types: %#v", extractEntryPoints(spec))
	}

	// all invalid types in []interface{}
	spec = map[string]interface{}{"entryPoints": []interface{}{nil, 42, false}}
	if len(extractEntryPoints(spec)) != 0 {
		t.Fatalf("expected empty for all invalid types in slice")
	}
}
