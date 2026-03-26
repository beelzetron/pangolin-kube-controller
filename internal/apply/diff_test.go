package apply

import (
	"sort"
	"testing"

	"k8s.io/apimachinery/pkg/api/equality"
)

func TestDiffKeys(t *testing.T) {
	tests := []struct {
		name   string
		oldM   map[string]interface{}
		newM   map[string]interface{}
		expect []string
	}{
		{
			name:   "empty maps",
			oldM:   map[string]interface{}{},
			newM:   map[string]interface{}{},
			expect: nil,
		},
		{
			name:   "identical maps",
			oldM:   map[string]interface{}{"a": "b"},
			newM:   map[string]interface{}{"a": "b"},
			expect: nil,
		},
		{
			name:   "key removed",
			oldM:   map[string]interface{}{"a": "b", "c": "d"},
			newM:   map[string]interface{}{"a": "b"},
			expect: []string{"c"},
		},
		{
			name:   "key added",
			oldM:   map[string]interface{}{"a": "b"},
			newM:   map[string]interface{}{"a": "b", "c": "d"},
			expect: []string{"c"},
		},
		{
			name:   "value changed",
			oldM:   map[string]interface{}{"a": "b"},
			newM:   map[string]interface{}{"a": "c"},
			expect: []string{"a"},
		},
		{
			name:   "numeric equality - int vs float64",
			oldM:   map[string]interface{}{"count": int64(5)},
			newM:   map[string]interface{}{"count": float64(5)},
			expect: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DiffKeys(tt.oldM, tt.newM)
			expect := append([]string(nil), tt.expect...)
			sort.Strings(got)
			sort.Strings(expect)
			if !equality.Semantic.DeepEqual(got, expect) {
				t.Errorf("DiffKeys() = %v, want %v", got, expect)
			}
		})
	}
}

func TestValuesSemanticallyEqual(t *testing.T) {
	tests := []struct {
		name string
		a    interface{}
		b    interface{}
		want bool
	}{
		{"nil equal", nil, nil, true},
		{"string equal", "hello", "hello", true},
		{"string not equal", "hello", "world", false},
		{"int equal", int64(42), int64(42), true},
		{"int vs float64 equal", int64(42), float64(42), true},
		{"int vs float64 not equal", int64(42), float64(43), false},
		{"uint64 equal", uint64(100), uint64(100), true},
		{"uint64 vs int64 positive equal", uint64(100), int64(100), true},
		{"uint64 vs int64 negative not equal", uint64(100), int64(-1), false},
		{"int64 overflow vs float64", int64(1 << 61), float64(1 << 61), false},
		{"large uint64 not equal to float64", uint64(1 << 61), float64(1 << 61), false},
		{"bool equal", true, true, true},
		{"bool not equal", true, false, false},
		{"map equal", map[string]interface{}{"a": 1}, map[string]interface{}{"a": 1}, true},
		{"map not equal", map[string]interface{}{"a": 1}, map[string]interface{}{"a": 2}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValuesSemanticallyEqual(tt.a, tt.b); got != tt.want {
				t.Errorf("ValuesSemanticallyEqual(%v, %v) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestValuesSemanticallyEqualWithSemanticDeepEqual(t *testing.T) {
	mapA := map[string]interface{}{"nested": map[string]interface{}{"key": "value"}}
	mapB := map[string]interface{}{"nested": map[string]interface{}{"key": "value"}}

	if !equality.Semantic.DeepEqual(mapA, mapB) {
		t.Fatal("test setup: maps should be deeply equal")
	}

	if !ValuesSemanticallyEqual(mapA, mapB) {
		t.Error("ValuesSemanticallyEqual should return true for DeepEqual maps")
	}
}

func TestDiffKeysWithNestedMaps(t *testing.T) {
	oldM := map[string]interface{}{
		"spec": map[string]interface{}{
			"port": int64(8080),
		},
	}
	newM := map[string]interface{}{
		"spec": map[string]interface{}{
			"port": float64(8080),
		},
	}

	got := DiffKeys(oldM, newM)
	if len(got) != 0 {
		t.Errorf("DiffKeys() = %v, expected no differences for semantically equal nested values", got)
	}
}

func TestDiffKeysWithAllTypes(t *testing.T) {
	oldM := map[string]interface{}{
		"str":    "old",
		"num":    int64(5),
		"flag":   true,
		"nilVal": nil,
	}
	newM := map[string]interface{}{
		"str":    "new",
		"num":    float64(6),
		"flag":   false,
		"nilVal": "string",
	}

	got := DiffKeys(oldM, newM)
	expected := []string{"str", "num", "flag", "nilVal"}
	sort.Strings(got)
	sort.Strings(expected)
	if len(got) != len(expected) {
		t.Errorf("DiffKeys() returned %d keys, want %d; got=%v expected=%v", len(got), len(expected), got, expected)
		return
	}
	for i := range expected {
		if got[i] != expected[i] {
			t.Errorf("DiffKeys() mismatch at %d: got=%v expected=%v", i, got, expected)
			return
		}
	}
}

func TestComputeHash(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		want  string // expected hex-encoded sha256
	}{
		{
			name:  "empty input",
			input: []byte{},
			// sha256 of empty: e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
			want: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		},
		{
			name:  "hello",
			input: []byte("hello"),
			// sha256("hello") = 2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824
			want: "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ComputeHash(tt.input)
			if got != tt.want {
				t.Errorf("ComputeHash(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestComputeHashDeterministic(t *testing.T) {
	b := []byte("deterministic-test")
	h1 := ComputeHash(b)
	h2 := ComputeHash(b)
	if h1 != h2 {
		t.Errorf("ComputeHash not deterministic: %q != %q", h1, h2)
	}
}

func TestComputeHashDistinct(t *testing.T) {
	if ComputeHash([]byte("foo")) == ComputeHash([]byte("bar")) {
		t.Error("ComputeHash should return different values for different inputs")
	}
}
