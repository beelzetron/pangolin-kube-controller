package testschema

import (
	"strings"
	"testing"
)

func TestKindNameKey(t *testing.T) {
	k := kindNameKey(map[string]interface{}{
		"kind":     "IngressRoute",
		"metadata": map[string]interface{}{"name": "MyRoute"},
	})
	if k != "ingressroute/myroute" {
		t.Fatalf("unexpected key: %s", k)
	}
	// Missing metadata/name should still work
	k = kindNameKey(map[string]interface{}{"kind": "Service"})
	if k != "service/" {
		t.Fatalf("unexpected key for missing name: %s", k)
	}
}

func TestDeterministicYAMLSortsAndJoins(t *testing.T) {
	objs := []map[string]interface{}{
		{"kind": "Middleware", "metadata": map[string]interface{}{"name": "zzz"}},
		{"kind": "IngressRoute", "metadata": map[string]interface{}{"name": "aaa"}},
		{"kind": "IngressRoute", "metadata": map[string]interface{}{"name": "bbb"}},
	}
	out, err := DeterministicYAML(objs)
	if err != nil {
		t.Fatalf("DeterministicYAML error: %v", err)
	}
	s := string(out)
	// Expect three documents separated by '---' with sorted order by kind/name (ingressroute/aaa, ingressroute/bbb, middleware/zzz)
	if strings.Count(s, "---\n") != 2 {
		t.Fatalf("expected 2 document separators, got %d", strings.Count(s, "---\n"))
	}
	// Order assertions
	firstDocEnd := strings.Index(s, "---\n")
	if firstDocEnd == -1 {
		t.Fatalf("no first doc separator found")
	}
	first := s[:firstDocEnd]
	if !strings.Contains(first, "IngressRoute") || !strings.Contains(first, "aaa") {
		t.Fatalf("first doc should be IngressRoute/aaa, got: %s", first)
	}
	// Second doc starts after separator
	rest := s[firstDocEnd+4:]
	secondDocEnd := strings.Index(rest, "---\n")
	if secondDocEnd == -1 {
		t.Fatalf("no second doc separator found")
	}
	second := rest[:secondDocEnd]
	if !strings.Contains(second, "IngressRoute") || !strings.Contains(second, "bbb") {
		t.Fatalf("second doc should be IngressRoute/bbb, got: %s", second)
	}
	third := rest[secondDocEnd+4:]
	if !strings.Contains(third, "Middleware") || !strings.Contains(third, "zzz") {
		t.Fatalf("third doc should be Middleware/zzz, got: %s", third)
	}
}
