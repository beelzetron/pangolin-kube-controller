package logging

import (
	"encoding/json"
	"strings"
)

// sensitiveSubstrings defines case-insensitive substrings which, when present in a JSON key,
// cause the corresponding value to be replaced with a placeholder.
var sensitiveSubstrings = []string{
	"auth", "pass", "secret", "token", "key",
}

const redacted = "***redacted***"

// RedactJSONLike attempts to parse the provided bytes as JSON and returns a redacted
// JSON document where any value whose key name contains one of the sensitive substrings
// (case-insensitive) is replaced with a redaction placeholder. If parsing fails, the input
// is returned unchanged along with a non-nil error so callers may decide how to proceed.
func RedactJSONLike(in []byte) ([]byte, error) {
	var v interface{}
	if err := json.Unmarshal(in, &v); err != nil {
		return in, err
	}
	redactedValue := redactAny(v)
	b, err := json.Marshal(redactedValue)
	if err != nil {
		return in, err
	}
	return b, nil
}

func redactAny(v interface{}) interface{} {
	switch t := v.(type) {
	case map[string]interface{}:
		return redactMap(t)
	case []interface{}:
		for i := range t {
			t[i] = redactAny(t[i])
		}
		return t
	default:
		return v
	}
}

func redactMap(m map[string]interface{}) map[string]interface{} {
	for k, val := range m {
		if containsSensitive(k) {
			m[k] = redacted
			continue
		}
		m[k] = redactAny(val)
	}
	return m
}

func containsSensitive(key string) bool {
	lk := strings.ToLower(key)
	for _, sub := range sensitiveSubstrings {
		if strings.Contains(lk, sub) {
			return true
		}
	}
	return false
}
