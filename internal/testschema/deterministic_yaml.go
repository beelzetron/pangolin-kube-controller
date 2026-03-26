package testschema

import (
	"bytes"
	"sort"
	"strings"

	"sigs.k8s.io/yaml"
)

// DeterministicYAML sorts top-level documents by Kind/name ordering and returns a multi-doc YAML.
// Not used yet by unit tests; kept for future offline E2E snapshotting.
func DeterministicYAML(objs []map[string]interface{}) ([]byte, error) {
	// Shallow sort by kind/name if present
	sort.SliceStable(objs, func(i, j int) bool {
		ki := kindNameKey(objs[i])
		kj := kindNameKey(objs[j])
		return ki < kj
	})
	var out bytes.Buffer
	for idx, m := range objs {
		b, err := yaml.Marshal(m)
		if err != nil {
			return nil, err
		}
		if idx > 0 {
			out.WriteString("---\n")
		}
		out.Write(b)
	}
	return out.Bytes(), nil
}

func kindNameKey(m map[string]interface{}) string {
	kind, _ := m["kind"].(string)
	var name string
	if md, ok := m["metadata"].(map[string]interface{}); ok {
		name, _ = md["name"].(string)
	}
	return strings.ToLower(kind) + "/" + strings.ToLower(name)
}
