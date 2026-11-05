package lint_test

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// No Go file in the module finds a path through its own source file
// (iss-2611011343052330). The runtime package reports the file a call sits in,
// and under -trimpath that is the module path (github.com/intentdriven/abcd/...),
// not a place on disk, so a test that joins it to reach the repository fails
// there, and the test stages are to run under -trimpath so that one worktree's
// cached results serve another (itd-2611011325421304). A test finds the
// repository by walking up from its working directory to go.mod instead
// (fsutil.ModuleRoot("."), or a package's own helper doing the same).
//
// The rule is the shape, not the nine files the issue named: it reads every Go
// file, whatever its build tags, and not only test files, since a helper
// package a test imports reaches the same dead path.
func TestNoGoFileFindsAPathThroughItsOwnSource(t *testing.T) {
	root, err := fsutil.ModuleRoot(".")
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	var sites []string
	err = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if p != root && (strings.HasPrefix(name, ".") || name == "testdata" || name == "vendor" || name == "node_modules") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(p, ".go") {
			return nil
		}
		f, err := parser.ParseFile(fset, p, nil, parser.SkipObjectResolution)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		for _, line := range sourcePathLookups(fset, f) {
			sites = append(sites, filepath.ToSlash(rel)+":"+strconv.Itoa(line))
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(sites) > 0 {
		t.Fatalf("these read the path of their own source file from the runtime package, which -trimpath breaks; walk up from the working directory to go.mod instead (fsutil.ModuleRoot(\".\")):\n%s", strings.Join(sites, "\n"))
	}
}

// sourcePathLookups returns the line of every use in f of what can yield a
// source path: runtime.Caller, unless a four-way assignment discards its file
// (`pc, _, line, ok :=`, whose results survive -trimpath), whether it is called,
// passed on whole or kept as a value; runtime.CallersFrames, whose frames carry
// the file; and a FileLine method, which runtime.FuncForPC's result carries. A
// file that does not import the runtime package has none, and the package is
// matched by its import path, so an alias is caught too. A dot import is
// reported at the import itself, since its uses carry no qualifier to match.
func sourcePathLookups(fset *token.FileSet, f *ast.File) []int {
	local := ""
	for _, imp := range f.Imports {
		if path, err := strconv.Unquote(imp.Path.Value); err != nil || path != "runtime" {
			continue
		}
		local = "runtime"
		if imp.Name != nil {
			local = imp.Name.Name
		}
		if local == "." {
			return []int{fset.Position(imp.Pos()).Line}
		}
	}
	if local == "" || local == "_" {
		return nil
	}
	inRuntime := func(sel *ast.SelectorExpr, names ...string) bool {
		id, ok := sel.X.(*ast.Ident)
		return ok && id.Name == local && slices.Contains(names, sel.Sel.Name)
	}
	// The calls that discard the file, found first so the walk below can tell
	// them from every other use of runtime.Caller.
	discarded := map[*ast.SelectorExpr]bool{}
	ast.Inspect(f, func(n ast.Node) bool {
		var file ast.Expr
		var rhs []ast.Expr
		switch s := n.(type) {
		case *ast.AssignStmt:
			if len(s.Lhs) == 4 {
				file = s.Lhs[1]
			}
			rhs = s.Rhs
		case *ast.ValueSpec:
			if len(s.Names) == 4 {
				file = s.Names[1]
			}
			rhs = s.Values
		}
		id, ok := file.(*ast.Ident)
		if !ok || id.Name != "_" || len(rhs) != 1 {
			return true
		}
		if call, ok := rhs[0].(*ast.CallExpr); ok {
			if sel, ok := ast.Unparen(call.Fun).(*ast.SelectorExpr); ok && inRuntime(sel, "Caller") {
				discarded[sel] = true
			}
		}
		return true
	})
	var lines []int
	ast.Inspect(f, func(n ast.Node) bool {
		sel, ok := n.(*ast.SelectorExpr)
		if ok && !discarded[sel] && (inRuntime(sel, "Caller", "CallersFrames") || sel.Sel.Name == "FileLine") {
			lines = append(lines, fset.Position(sel.Pos()).Line)
		}
		return true
	})
	return lines
}

// The detector is watched on each shape it judges, so a rewrite that stops
// seeing one fails here rather than passing a lookup it no longer finds.
func TestSourcePathLookupsJudgesEachShape(t *testing.T) {
	cases := []struct {
		name string
		body string
		want []int
	}{
		{"the file kept", "import \"runtime\"\nfunc f() { _, file, _, _ := runtime.Caller(0); _ = file }", []int{4}},
		{"the file discarded", "import \"runtime\"\nfunc f() { pc, _, line, ok := runtime.Caller(0); _, _, _ = pc, line, ok }", nil},
		{"the file discarded in parentheses", "import \"runtime\"\nfunc f() { pc, _, _, _ := (runtime.Caller)(0); _ = pc }", nil},
		{"a var with the file kept", "import \"runtime\"\nvar _, file, _, _ = runtime.Caller(0)", []int{4}},
		{"a var with the file discarded", "import \"runtime\"\nvar pc, _, _, _ = runtime.Caller(0)", nil},
		{"a reassignment", "import \"runtime\"\nfunc f() { var file string; _, file, _, _ = runtime.Caller(0); _ = file }", []int{4}},
		{"the call passed on whole", "import \"runtime\"\nfunc g(uintptr, string, int, bool) {}\nfunc f() { g(runtime.Caller(0)) }", []int{5}},
		{"the function kept as a value", "import \"runtime\"\nvar caller = runtime.Caller\nfunc f() { _, file, _, _ := caller(0); _ = file }", []int{4}},
		{"an aliased import", "import rt \"runtime\"\nfunc f() { _, file, _, _ := rt.Caller(1); _ = file }", []int{4}},
		{"a dot import", "import . \"runtime\"\nfunc f() { _, file, _, _ := Caller(1); _ = file }", []int{3}},
		{"the frames of a stack", "import \"runtime\"\nfunc f(pcs []uintptr) { _ = runtime.CallersFrames(pcs) }", []int{4}},
		{"FileLine on a function", "import \"runtime\"\nfunc f(pc uintptr) { fn := runtime.FuncForPC(pc); file, _ := fn.FileLine(pc); _ = file }", []int{4}},
		{"a function's name alone", "import \"runtime\"\nfunc f(pc uintptr) string { return runtime.FuncForPC(pc).Name() }", nil},
		{"another package's Caller", "import \"example.invalid/runtime2\"\nfunc f() { _, file, _, _ := runtime2.Caller(0); _ = file }", nil},
		{"no runtime import", "func f() {}", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, "x.go", fmt.Sprintf("package x\n\n%s\n", tc.body), parser.SkipObjectResolution)
			if err != nil {
				t.Fatal(err)
			}
			if got := sourcePathLookups(fset, f); !slices.Equal(got, tc.want) {
				t.Errorf("sourcePathLookups = %v, want %v", got, tc.want)
			}
		})
	}
}
