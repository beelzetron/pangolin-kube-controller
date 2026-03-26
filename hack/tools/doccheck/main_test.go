package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestIsExportedAndVendor(t *testing.T) {
	if !isExported("Foo") || isExported("foo") {
		t.Fatalf("isExported unexpected")
	}
	if !isVendorPackage("vendor/foo") || isVendorPackage("foo/bar") {
		t.Fatalf("isVendorPackage unexpected")
	}
}

func TestIsWithinAndSafeJoinUnder(t *testing.T) {
	dir := t.TempDir()
	nested := filepath.Join(dir, "a", "b")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	if !isWithin(dir, nested) {
		t.Fatalf("expected %s to be within %s", nested, dir)
	}
	if _, err := safeJoinUnder(dir, "a/b/c"); err != nil {
		t.Fatalf("safeJoinUnder failed: %v", err)
	}
	if _, err := safeJoinUnder(dir, "../etc/passwd"); err == nil {
		t.Fatalf("expected traversal rejection")
	}
}

func TestIsTrackedGenDeclAndRender(t *testing.T) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", "package foo\nconst A = 1\nvar B = 2\ntype C int", 0)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	var constDecl, varDecl, typeDecl *ast.GenDecl
	ast.Inspect(f, func(n ast.Node) bool {
		if d, ok := n.(*ast.GenDecl); ok {
			switch d.Tok {
			case token.CONST:
				constDecl = d
			case token.VAR:
				varDecl = d
			case token.TYPE:
				typeDecl = d
			}
		}
		return true
	})
	if !isTrackedGenDecl(constDecl) || !isTrackedGenDecl(varDecl) || !isTrackedGenDecl(typeDecl) {
		t.Fatalf("isTrackedGenDecl failed")
	}

	rep := report{Findings: []finding{{PackagePath: "foo", PackageName: "foo", MissingPkgDoc: true, MissingIdents: []string{"A"}}}}
	out := renderMarkdown(rep)
	if !strings.Contains(out, "foo") || !strings.Contains(out, "A") {
		t.Fatalf("renderMarkdown output unexpected")
	}

	empty := renderMarkdown(report{})
	if !strings.Contains(empty, "All good") {
		t.Fatalf("expected All good for empty report")
	}

	m := map[string]struct{}{"z": {}, "a": {}, "m": {}}
	got := sortedMissing(m)
	if !slices.Equal(got, []string{"a", "m", "z"}) {
		t.Fatalf("sortedMissing unexpected: %v", got)
	}
}
