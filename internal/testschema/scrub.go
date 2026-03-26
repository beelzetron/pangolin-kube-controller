package testschema

// Utility to scrub nondeterministic metadata from unstructured objects before golden comparison.

// ScrubObjectMeta removes non-deterministic metadata fields (such as resourceVersion, uid,
// generation, creationTimestamp, and managedFields) from the "metadata" of the given
// unstructured Kubernetes object represented as a map. This is intended for testing purposes
// to ensure that object comparisons are not affected by automatically managed fields.
// If m or m["metadata"] is nil, it returns the input map unchanged.
func ScrubObjectMeta(obj map[string]interface{}) map[string]interface{} {
	if obj == nil {
		return nil
	}
	meta, ok := obj["metadata"].(map[string]interface{})
	if !ok || meta == nil {
		return obj
	}
	delete(meta, "resourceVersion")
	delete(meta, "uid")
	delete(meta, "generation")
	delete(meta, "creationTimestamp")
	delete(meta, "managedFields")
	obj["metadata"] = meta
	return obj
}
