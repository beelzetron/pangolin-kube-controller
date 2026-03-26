package sanitize

import (
	"strings"
	"testing"
)

func TestSanitizeReferenceMappingAndFallback(t *testing.T) {
	m := map[string]string{"old": "new"}
	if got := sanitizeReference("old", m); got != "new" {
		t.Fatalf("expected mapped value, got %s", got)
	}
	if got := sanitizeReference("X", m); got != "x" { // fallback is SanitizeResourceName("X")
		t.Fatalf("expected fallback sanitized value, got %s", got)
	}
}

func TestFinalizeSanitizedNameHashSuffix(t *testing.T) {
	orig := strings.Repeat("A", 400)
	san := strings.Repeat("a", 400)
	got := finalizeSanitizedName(san, orig)
	if len(got) > maxK8sNameLength {
		t.Fatalf("expected length <= %d, got %d", maxK8sNameLength, len(got))
	}
	suffix := shortHash(orig)
	if !strings.HasSuffix(got, "-"+suffix) {
		t.Fatalf("expected hash suffix, got %s", got)
	}
}
