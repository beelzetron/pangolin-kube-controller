// Command doccheck scans packages to report missing package docs and exported
// identifier documentation, emitting a JSON report and Markdown summary.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

type finding struct {
	PackagePath   string   `json:"package_path"`
	PackageName   string   `json:"package_name"`
	MissingPkgDoc bool     `json:"missing_package_doc"`
	MissingIdents []string `json:"missing_identifiers"`
}

type report struct {
	Findings []finding `json:"findings"`
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	outDir := flag.String("outdir", "dist/doccheck", "output directory for reports")
	flag.Parse()

	modRoot, err := findModuleRoot()
	if err != nil {
		return err
	}
	pkgs, err := goList(modRoot)
	if err != nil {
		return err
	}
	outBase, err := safeJoinUnder(modRoot, *outDir)
	if err != nil {
		return err
	}
	rep := collectFindings(modRoot, pkgs)
	if err := writeReport(outBase, rep); err != nil {
		return err
	}
	return nil
}

func collectFindings(modRoot string, pkgs []string) report {
	var rep report
	for _, pkg := range pkgs {
		fd, ok := collectPackageFinding(modRoot, pkg)
		if !ok {
			continue
		}
		rep.Findings = append(rep.Findings, fd)
	}
	return rep
}

func collectPackageFinding(modRoot, pkg string) (finding, bool) {
	if isVendorPackage(pkg) {
		return finding{}, false
	}
	fp := filepath.Join(modRoot, filepath.FromSlash(pkg))
	if !isWithin(modRoot, fp) {
		return finding{}, false
	}
	astFiles, missingPkgDoc := parsePackageFiles(fp)
	if len(astFiles) == 0 {
		return finding{}, false
	}
	return finding{
		PackagePath:   pkg,
		PackageName:   astFiles[0].Name.Name,
		MissingPkgDoc: missingPkgDoc,
		MissingIdents: findMissingIdentifiers(astFiles),
	}, true
}

func parsePackageFiles(pkgDir string) ([]*ast.File, bool) {
	fset := token.NewFileSet()
	files, err := os.ReadDir(pkgDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to read package directory %s: %v\n", pkgDir, err)
		// Do not treat filesystem errors as missing package documentation.
		// Skip this package by returning no files and missingPkgDoc=false.
		return nil, false
	}
	var astFiles []*ast.File
	missingPkgDoc := true
	for _, e := range files {
		name := e.Name()
		if !shouldParseFile(name) {
			continue
		}
		path := filepath.Join(pkgDir, name)
		f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			continue
		}
		astFiles = append(astFiles, f)
		if hasPackageDoc(f) {
			missingPkgDoc = false
		}
	}
	return astFiles, missingPkgDoc
}

func shouldParseFile(name string) bool {
	return strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, "_test.go")
}

func hasPackageDoc(f *ast.File) bool {
	if f.Doc == nil {
		return false
	}
	text := f.Doc.Text()
	text = strings.TrimSpace(text)
	if text == "" {
		return false
	}
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "go:build") || strings.HasPrefix(line, "+build") {
			continue
		}
		return strings.HasPrefix(line, "Package "+f.Name.Name)
	}
	return false
}

func writeReport(outBase string, rep report) error {
	if err := os.MkdirAll(outBase, 0o750); err != nil {
		return err
	}
	jsonPath := filepath.Join(outBase, "report.json")
	b, err := json.MarshalIndent(rep, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal report: %w", err)
	}
	if err := os.WriteFile(jsonPath, b, 0o644); err != nil {
		return err
	}
	mdPath := filepath.Join(outBase, "summary.md")
	if err := os.WriteFile(mdPath, []byte(renderMarkdown(rep)), 0o644); err != nil {
		return err
	}
	return nil
}

func findMissingIdentifiers(files []*ast.File) []string {
	missing := make(map[string]struct{})
	for _, f := range files {
		collectMissingInFile(f, missing)
	}
	return sortedMissing(missing)
}

func collectMissingInFile(file *ast.File, missing map[string]struct{}) {
	ast.Inspect(file, func(n ast.Node) bool {
		addMissingFromNode(n, missing)
		return true
	})
}

func addMissingFromNode(n ast.Node, missing map[string]struct{}) {
	switch d := n.(type) {
	case *ast.GenDecl:
		addMissingFromGenDecl(d, missing)
	case *ast.FuncDecl:
		addMissingFuncDecl(d, missing)
	}
}

func addMissingFromGenDecl(decl *ast.GenDecl, missing map[string]struct{}) {
	if !isTrackedGenDecl(decl) {
		return
	}
	for _, spec := range decl.Specs {
		addMissingFromSpec(decl, spec, missing)
	}
}

func isTrackedGenDecl(decl *ast.GenDecl) bool {
	return decl.Tok == token.TYPE || decl.Tok == token.CONST || decl.Tok == token.VAR
}

func addMissingFromSpec(decl *ast.GenDecl, spec ast.Spec, missing map[string]struct{}) {
	switch s := spec.(type) {
	case *ast.TypeSpec:
		addMissingTypeSpec(decl, s, missing)
	case *ast.ValueSpec:
		addMissingValueSpec(decl, s, missing)
	}
}

func addMissingTypeSpec(decl *ast.GenDecl, spec *ast.TypeSpec, missing map[string]struct{}) {
	if isExported(spec.Name.Name) && decl.Doc == nil && spec.Doc == nil {
		missing[spec.Name.Name] = struct{}{}
	}
}

func addMissingValueSpec(decl *ast.GenDecl, spec *ast.ValueSpec, missing map[string]struct{}) {
	if decl.Doc != nil || spec.Doc != nil {
		return
	}
	for _, name := range spec.Names {
		if isExported(name.Name) {
			missing[name.Name] = struct{}{}
		}
	}
}

func addMissingFuncDecl(decl *ast.FuncDecl, missing map[string]struct{}) {
	if decl.Recv == nil && isExported(decl.Name.Name) && decl.Doc == nil {
		missing[decl.Name.Name] = struct{}{}
	}
}

func sortedMissing(missing map[string]struct{}) []string {
	var out []string
	for k := range missing {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func renderMarkdown(rep report) string {
	var buf bytes.Buffer
	buf.WriteString("# Doccheck Summary\n\n")
	wroteFinding := false
	for _, f := range rep.Findings {
		if !f.MissingPkgDoc && len(f.MissingIdents) == 0 {
			continue
		}
		wroteFinding = true
		buf.WriteString(fmt.Sprintf("## %s (%s)\n", f.PackagePath, f.PackageName))
		if f.MissingPkgDoc {
			buf.WriteString("- Missing package doc\n")
		}
		if len(f.MissingIdents) > 0 {
			buf.WriteString("- Missing exported docs: ")
			buf.WriteString(strings.Join(f.MissingIdents, ", "))
			buf.WriteString("\n")
		}
		buf.WriteString("\n")
	}
	if !wroteFinding {
		return "# Doccheck Summary\n\nAll good.\n"
	}
	return buf.String()
}

func goList(modRoot string) ([]string, error) {
	cmd := exec.Command("go", "list", "./...")
	cmd.Dir = modRoot
	cmd.Env = sanitizedEnv()
	b, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("go list: %w\n%s", err, string(b))
	}
	s := strings.Split(strings.TrimSpace(string(b)), "\n")
	var pkgs []string
	for _, line := range s {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Convert module path to directory path: assume it maps to relative path inside module
		// Use go list -f '{{.Dir}}' in a more advanced version; for now infer by replacing module root path prefix
		// Simpler: after go list, run go list -f '{{.Dir}}' per pkg to get absolute directories.
		dcmd := exec.Command("go", "list", "-f", "{{.Dir}}", line)
		dcmd.Dir = modRoot
		dcmd.Env = sanitizedEnv()
		db, derr := dcmd.Output()
		if derr != nil {
			continue
		}
		d := strings.TrimSpace(string(db))
		if d == "" {
			continue
		}
		rel, rerr := filepath.Rel(modRoot, d)
		if rerr != nil {
			fmt.Fprintf(os.Stderr, "warning: filepath.Rel failed for modRoot=%s dir=%s: %v\n", modRoot, d, rerr)
			continue
		}
		rel = filepath.ToSlash(rel)
		pkgs = append(pkgs, rel)
	}
	return pkgs, nil
}

func safeJoinUnder(base, rel string) (string, error) {
	if filepath.IsAbs(rel) {
		return "", fmt.Errorf("outdir must be relative: %s", rel)
	}
	relSlash := filepath.ToSlash(rel)
	if relSlash == ".." || strings.HasPrefix(relSlash, "../") || strings.Contains(relSlash, "/../") {
		return "", fmt.Errorf("outdir must not traverse outside module root: %s", rel)
	}
	clean := filepath.Clean(rel)
	if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("outdir must not traverse outside module root: %s", rel)
	}
	joined := filepath.Join(base, clean)
	if !isWithin(base, joined) {
		return "", fmt.Errorf("outdir must be within module root: %s", rel)
	}
	return joined, nil
}

func isWithin(base, target string) bool {
	rel, err := filepath.Rel(base, target)
	if err != nil {
		return false
	}
	rel = filepath.Clean(rel)
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

// sanitizedEnv returns the current environment with PATH replaced by a
// conservative, fixed set of system directories to avoid executing binaries
// from writable or user-controlled locations. This mitigates SAST warnings
// about searching OS commands in an attacker-controlled PATH.
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

func isVendorPackage(pkg string) bool {
	pkg = filepath.ToSlash(pkg)
	return pkg == "vendor" || strings.HasPrefix(pkg, "vendor/") || strings.Contains(pkg, "/vendor/")
}

func findModuleRoot() (string, error) {
	d := "."
	for {
		if _, err := os.Stat(filepath.Join(d, "go.mod")); err == nil {
			abs, err := filepath.Abs(d)
			if err != nil {
				return "", err
			}
			return abs, nil
		}
		nd := filepath.Dir(d)
		if nd == d {
			return "", fmt.Errorf("go.mod not found")
		}
		d = nd
	}
}

func isExported(name string) bool {
	if name == "" {
		return false
	}
	r, _ := utf8.DecodeRuneInString(name)
	return unicode.IsUpper(r)
}
