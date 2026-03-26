package protocol

import (
	"net/url"
	"testing"
)

func mustParse(t *testing.T, s string) *url.URL {
	t.Helper()
	u, err := url.Parse(s)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return u
}

func TestDerivePortExplicitAndDefaults(t *testing.T) {
	if port := derivePort(mustParse(t, "http://host:8081")); port != 8081 {
		t.Fatalf("explicit port got %d", port)
	}
	if port := derivePort(mustParse(t, "https://host")); port != 443 {
		t.Fatalf("https default got %d", port)
	}
	if port := derivePort(mustParse(t, "http://host")); port != 80 {
		t.Fatalf("http default got %d", port)
	}
}
