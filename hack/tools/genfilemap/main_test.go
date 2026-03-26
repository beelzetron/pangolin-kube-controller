package main

import (
	"os"
	"runtime"
	"strings"
	"testing"
)

func TestHyphenToWords(t *testing.T) {
	got := hyphenToWords("foo_bar-baz")
	if got != "foo bar baz" {
		t.Fatalf("unexpected hyphenToWords: %q", got)
	}
	if hyphenToWords("") != "behavior" {
		t.Fatalf("empty input should return 'behavior'")
	}
}

func TestDeriveGroupAndIsTestFile(t *testing.T) {
	cases := map[string]string{
		"cmd/healthcheck/main.go":               "cmd",
		"internal/observability/metrics/foo.go": "internal/observability/metrics",
		"test/integration/foo.go":               "test",
		"foo/bar/baz.go":                        "foo",
	}
	for in, want := range cases {
		got := string(deriveGroup(in))
		if got != want {
			t.Fatalf("deriveGroup(%q) = %q, want %q", in, got, want)
		}
	}
	if !isTestFile("foo_test.go") || isTestFile("foo.go") {
		t.Fatalf("isTestFile behavior unexpected")
	}
}

func TestOneLineAndPathHeuristic(t *testing.T) {
	if oneLine("hello world") != "hello world" {
		t.Fatalf("oneLine failed")
	}
	if oneLine("   ") != "" {
		t.Fatalf("oneLine whitespace failed")
	}
	if s, ok := pathHeuristicSummary("cmd/healthcheck/main.go", "main.go"); !ok || !strings.Contains(s, "Health probe") {
		t.Fatalf("pathHeuristicSummary unexpected: %v %v", s, ok)
	}
}

func TestSortedGroupKeys(t *testing.T) {
	m := map[groupKey][]entry{"z": {}, "a": {}, "m": {}}
	got := sortedGroupKeys(m)
	if len(got) != 3 || string(got[0]) != "a" {
		t.Fatalf("sortedGroupKeys unexpected: %v", got)
	}
}

func TestSanitizedEnvProvidesSinglePath(t *testing.T) {
	env := sanitizedEnv()
	var pathCount int
	var pathVal string
	for _, e := range env {
		if strings.HasPrefix(e, "PATH=") || strings.HasPrefix(e, "Path=") {
			pathCount++
			pathVal = e
		}
	}
	if pathCount != 1 {
		t.Fatalf("expected one PATH, got %d", pathCount)
	}
	if runtime.GOOS == "windows" {
		if !strings.Contains(pathVal, "C:") {
			t.Fatalf("windows PATH unexpected: %s", pathVal)
		}
	} else {
		if !strings.Contains(pathVal, "/usr/") {
			t.Fatalf("unix PATH unexpected: %s", pathVal)
		}
	}
	// ensure sanitizedEnv returns an env slice
	if len(env) == 0 {
		t.Fatalf("sanitizedEnv returned empty env")
	}
	_ = os.Environ() // keep lint happy
}
