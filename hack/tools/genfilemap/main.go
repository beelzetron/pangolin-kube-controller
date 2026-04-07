// Command genfilemap generates a Markdown overview of tracked Go files grouped
// by folder with a one-line purpose per file.
package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

func main() {
	out := flag.String("out", "docs/GO_FILES_OVERVIEW.md", "output path relative to module root")
	flag.Parse()

	modRoot, err := findModuleRoot()
	fatalIf(err)
	files, err := gitTrackedGoFiles(modRoot)
	fatalIf(err)

	groups := groupFiles(modRoot, files)

	var buf bytes.Buffer
	buf.WriteString("# Go Files Overview\n\n")
	buf.WriteString("Purpose: quick, non-developer friendly guide to what each Go file does and why it exists. Grouped by folder.\n\n")
	buf.WriteString("## Notes\n\n- “Tests” entries are unit/integration tests that verify the behavior described; names indicate focus.\n- CRD = Custom Resource Definition (Traefik’s Kubernetes objects such as IngressRoute, Middleware, TraefikService).\n\n")

	for _, g := range sortedGroupKeys(groups) {
		buf.WriteString("## " + string(g) + "\n")
		entries := groups[g]
		sort.Slice(entries, func(i, j int) bool { return entries[i].Path < entries[j].Path })
		for _, e := range entries {
			fmt.Fprintf(&buf, "- %s — %s\n", e.Path, e.Summary)
		}
		buf.WriteString("\n")
	}

	outPath := filepath.Join(modRoot, filepath.FromSlash(*out))
	fatalIf(os.MkdirAll(filepath.Dir(outPath), 0o750))
	fatalIf(os.WriteFile(outPath, buf.Bytes(), 0o644))
}

type entry struct {
	Path    string
	Summary string
}

type groupKey string

// rule defines a path matching rule used to derive human summaries.
type rule struct {
	prefix   string
	contains string
	suffix   string
	baseEq   string
	summary  string
}

// pathRules is the ordered set of rules used by pathHeuristicSummary.
var pathRules = []rule{
	{prefix: "cmd/healthcheck/", summary: "Health probe binary. Runs liveness/readiness checks."},
	{suffix: "/main.go", summary: "Entrypoint binary. Loads configuration and starts orchestration."},
	{contains: "/internal/http/", baseEq: "server.go", summary: "HTTP server exposing metrics and health probes (and optional pprof)."},
	{contains: "/internal/reconcilers/core/", baseEq: "reconciler.go", summary: "Core reconciler: fetches remote config, reconciles Traefik CRDs (SSA), GC, metrics, readiness."},
	{contains: "/internal/observability/metrics/otel/", summary: "OpenTelemetry metrics instruments and exporter wiring."},
	{contains: "/internal/observability/log/", summary: "Helpers for safe logging (e.g., JSON redaction)."},
	{contains: "/internal/observability/metrics/", summary: "Prometheus metrics registry and handlers."},
	{contains: "/internal/traefik/protocol/", summary: "Protocol helpers: HTTP/TCP/UDP transforms for Services, EndpointSlices, and IngressRoute TCP/UDP."},
	{contains: "/internal/traefik/routing/", summary: "Router → IngressRoute transformation and annotations."},
	{contains: "/internal/traefik/sanitize/", summary: "Normalize names and references to Kubernetes-safe values."},
	{contains: "/internal/traefik/adapter/", summary: "Adapter for dynamic ResourceInterface (test-friendly)."},
	{contains: "/internal/traefik/config/", summary: "Lightweight Traefik dynamic config model and CRD group constants."},
	{contains: "/internal/config/", summary: "Configuration loading and normalization from environment (and optional file)."},
	{contains: "/internal/k8s/", summary: "Constructs in-cluster Kubernetes clients with rate limits."},
	{contains: "/internal/instancelabel/", summary: "Resolve and verify Traefik instance label used on managed resources."},
	{contains: "/internal/testschema/", summary: "Test helpers for CRD loading, scrubbing, and validation."},
	{contains: "/internal/version/", summary: "Build metadata (Version/Commit/Date) helpers."},
	{prefix: "test/", summary: "Integration/E2E test helpers and assets."},
}

func groupFiles(modRoot string, files []string) map[groupKey][]entry {
	res := make(map[groupKey][]entry)
	for _, p := range files {
		gk := deriveGroup(p)
		sum := summarizeFile(modRoot, p)
		res[gk] = append(res[gk], entry{Path: p, Summary: sum})
	}
	return res
}

func deriveGroup(p string) groupKey {
	parts := strings.Split(p, "/")
	if len(parts) == 0 {
		return groupKey(".")
	}
	// cmd/<name>
	if parts[0] == "cmd" && len(parts) >= 2 {
		return groupKey("cmd")
	}
	// internal/<pkg>[/subpkg]
	if parts[0] == "internal" {
		if len(parts) >= 3 && (parts[1] == "service" || parts[1] == "observability") {
			return groupKey(strings.Join(parts[:3], "/"))
		}
		if len(parts) >= 2 {
			return groupKey(strings.Join(parts[:2], "/"))
		}
	}
	// test/*
	if parts[0] == "test" {
		return groupKey("test")
	}
	return groupKey(parts[0])
}

func summarizeFile(modRoot, rel string) string {
	if s, ok := headerSummary(modRoot, rel); ok {
		return s
	}
	base := filepath.Base(rel)
	name := strings.TrimSuffix(base, ".go")
	if isTestFile(base) {
		return fmt.Sprintf("Tests %s.", hyphenToWords(strings.TrimSuffix(name, "_test")))
	}
	if s, ok := pathHeuristicSummary(rel, base); ok {
		return s
	}
	return "Go source file."
}

func headerSummary(modRoot, rel string) (string, bool) {
	srcPath := filepath.Join(modRoot, rel)
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, srcPath, nil, parser.ParseComments)
	if err == nil && f != nil && f.Doc != nil {
		if line := oneLine(f.Doc.Text()); line != "" {
			return line, true
		}
	}
	return "", false
}

func isTestFile(base string) bool {
	return strings.HasSuffix(base, "_test.go")
}

func pathHeuristicSummary(rel, base string) (string, bool) {
	// Order matters: more specific rules first.
	for _, r := range pathRules {
		if r.prefix != "" && !strings.HasPrefix(rel, r.prefix) {
			continue
		}
		if r.contains != "" && !strings.Contains(rel, r.contains) {
			continue
		}
		if r.suffix != "" && !strings.HasSuffix(rel, r.suffix) {
			continue
		}
		if r.baseEq != "" && base != r.baseEq {
			continue
		}
		return r.summary, true
	}
	return "", false
}

func oneLine(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	// Take first sentence-ish or first line
	if i := strings.IndexAny(s, "\n.!"); i >= 0 {
		line := strings.TrimSpace(s[:i])
		if line != "" {
			return line
		}
	}
	line := strings.SplitN(s, "\n", 2)[0]
	line = strings.Join(strings.Fields(line), " ")
	if len(line) > 180 {
		line = line[:177] + "..."
	}
	return line
}

func sortedGroupKeys(m map[groupKey][]entry) []groupKey {
	ks := make([]groupKey, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	sort.Slice(ks, func(i, j int) bool { return string(ks[i]) < string(ks[j]) })
	return ks
}

func findModuleRoot() (string, error) {
	d, err := filepath.Abs(".")
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(d, "go.mod")); err == nil {
			return d, nil
		}
		nd := filepath.Dir(d)
		if nd == d {
			return "", fmt.Errorf("go.mod not found from %s", d)
		}
		d = nd
	}
}

func gitTrackedGoFiles(modRoot string) ([]string, error) {
	// Ask git for all tracked files and filter for .go entries in Go to
	// avoid relying on shell/glob behavior differences between git versions.
	cmd := exec.Command("git", "--no-pager", "ls-files")
	cmd.Dir = modRoot
	cmd.Env = sanitizedEnv()
	b, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git ls-files: %w", err)
	}
	s := bufio.NewScanner(bytes.NewReader(b))
	var out []string
	for s.Scan() {
		p := filepath.ToSlash(strings.TrimSpace(s.Text()))
		// Only include Go source files
		if !strings.HasSuffix(p, ".go") {
			continue
		}
		if p == "" {
			continue
		}
		if strings.HasPrefix(p, "vendor/") || strings.Contains(p, "/vendor/") {
			continue
		}
		if strings.Contains(p, "/node_modules/") {
			continue
		}
		out = append(out, p)
	}
	return out, s.Err()
}

func hyphenToWords(s string) string {
	s = strings.ReplaceAll(s, "_", " ")
	s = strings.ReplaceAll(s, "-", " ")
	s = strings.TrimSpace(s)
	if s == "" {
		return "behavior"
	}
	return s
}

// fatalIf logs the error and exits the program when err is non-nil. Use for
// unrecoverable failures where continuing would leave the program in an
// undefined state.
func fatalIf(err error) {
	if err == nil {
		return
	}
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

// sanitizedEnv returns the current environment with PATH replaced by a
// conservative set of system directories so external commands aren't searched
// in user-writable locations. This addresses SAST concerns about PATH usage.
func sanitizedEnv() []string {
	var out []string
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, "PATH=") || strings.HasPrefix(e, "Path=") {
			continue
		}
		out = append(out, e)
	}
	var safePath string
	if runtime.GOOS == "windows" {
		safePath = `C:\Windows\System32;C:\Windows;C:\Windows\System32\Wbem`
	} else {
		safePath = "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"
	}
	out = append(out, "PATH="+safePath)
	return out
}
