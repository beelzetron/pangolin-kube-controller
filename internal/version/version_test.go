package version

import (
	"fmt"
	"testing"
)

func TestVersionVarsAreDefined(t *testing.T) {
	if Version == "" || Commit == "" || Date == "" {
		t.Fatalf("version variables should be defined: Version=%q Commit=%q Date=%q", Version, Commit, Date)
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
