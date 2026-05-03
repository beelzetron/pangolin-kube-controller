package version

import (
	"fmt"
	"testing"
)

// TestVersionVarsDefaults verifies that the package-level version variables
// equal their documented source-code defaults when the binary is not built
// with -ldflags overrides (i.e. in the standard `go test` context).
// This catches accidental changes to the default values in version.go.
func TestVersionVarsDefaults(t *testing.T) {
	tests := []struct {
		name string
		got  string
		want string
	}{
		{"Version", Version, "dev"},
		{"Commit", Commit, "none"},
		{"Date", Date, "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("%s default = %q, want %q", tt.name, tt.got, tt.want)
			}
		})
	}
}

func TestGetAndString(t *testing.T) {
	info := Get()
	if info.Version != Version || info.Commit != Commit || info.Date != Date {
		t.Fatalf("Get mismatch: got %+v want Version=%s Commit=%s Date=%s", info, Version, Commit, Date)
	}
	s := info.String()
	// Use a test-local format so the test fails if the production format
	// string changes unexpectedly. Keep the expected layout explicit here.
	const expectedFormat = "Version=%s Commit=%s Date=%s"
	want := fmt.Sprintf(expectedFormat, Version, Commit, Date)
	if s != want {
		t.Fatalf("Info.String mismatch: got %q want %q", s, want)
	}
	if String() != want {
		t.Fatalf("package String mismatch: got %q want %q", String(), want)
	}
}
