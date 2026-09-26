package ahoy

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// TestEveryExportedAhoyFunctionHasAFrontDoor is the caller audit iss-33 asked
// for, scoped to this package: an exported function exists to be reached from
// outside it, so one that no production code outside the package names is
// silent scaffolding — it compiles, it can rot, and nothing would notice. The
// loud way to stage a function is to leave it unexported until a front door
// calls it (loud-staging.md); an exported one nothing reaches is refused here.
//
// It reads the package's own non-test files for exported top-level functions,
// then parses the module's non-test Go files outside the package (testdata/
// excluded) for a selector on the imported ahoy package naming each. Tests do
// not count as callers: a function reached only by its own test is exactly the
// scaffolding the rule is about. Nor do comments or string literals, which a
// text match would have counted. The ForTest suffix exempts a declared
// cross-package test seam by its name alone.
func TestEveryExportedAhoyFunctionHasAFrontDoor(t *testing.T) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, ".", func(fi fs.FileInfo) bool {
		return !strings.HasSuffix(fi.Name(), "_test.go")
	}, 0)
	if err != nil {
		t.Fatal(err)
	}
	exported := map[string]bool{}
	for _, pkg := range pkgs {
		for _, f := range pkg.Files {
			for _, d := range f.Decls {
				fn, ok := d.(*ast.FuncDecl)
				if !ok || fn.Recv != nil || !fn.Name.IsExported() {
					continue
				}
				// A cross-package test seam says so in its name: it exists for
				// front-door test packages that cannot reach an unexported seam,
				// and production never calls it by construction.
				if strings.HasSuffix(fn.Name.Name, "ForTest") {
					continue
				}
				exported[fn.Name.Name] = false
			}
		}
	}
	if len(exported) == 0 {
		t.Fatal("found no exported functions; the audit is reading the wrong directory")
	}

	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		t.Fatalf("module root not found at %s: %v", root, err)
	}
	self, _ := filepath.Abs(".")
	err = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			// testdata/ holds fixtures the go tool never builds, so a .go file
			// there is not production code and names no caller.
			if p == self || name == ".git" || name == "node_modules" || name == "testdata" || (strings.HasPrefix(name, ".") && p != root) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
			return nil
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		names, err := ahoySelectors(data)
		if err != nil {
			return fmt.Errorf("%s: %w", p, err)
		}
		for _, name := range names {
			if _, ok := exported[name]; ok {
				exported[name] = true
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	var unreached []string
	for name, reached := range exported {
		if !reached {
			unreached = append(unreached, name)
		}
	}
	sort.Strings(unreached)
	if len(unreached) > 0 {
		t.Fatalf("exported ahoy functions no production code outside the package calls — wire a front door or unexport/delete them: %v", unreached)
	}
}

// ahoyImportPath is the package whose exported functions the audit holds to a
// front door.
const ahoyImportPath = "github.com/intentdriven/abcd/internal/core/ahoy"

// ahoySelectors names every identifier src selects from the imported ahoy
// package, under whatever name the file imports it as. It parses the file
// rather than matching text, so a comment or a string literal naming ahoy.X is
// not a caller, and a file that does not import the package has none. A file
// that does not parse returns the error, so the audit fails loudly rather than
// counting nothing for it.
func ahoySelectors(src []byte) ([]string, error) {
	f, err := parser.ParseFile(token.NewFileSet(), "", src, parser.SkipObjectResolution)
	if err != nil {
		return nil, err
	}
	local := ""
	for _, imp := range f.Imports {
		if strings.Trim(imp.Path.Value, "`\"") != ahoyImportPath {
			continue
		}
		local = "ahoy"
		if imp.Name != nil {
			local = imp.Name.Name
		}
	}
	if local == "" || local == "_" || local == "." {
		return nil, nil
	}
	var out []string
	ast.Inspect(f, func(n ast.Node) bool {
		sel, ok := n.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if id, ok := sel.X.(*ast.Ident); ok && id.Name == local {
			out = append(out, sel.Sel.Name)
		}
		return true
	})
	return out, nil
}

// TestAhoySelectorsCountsOnlyCode is the audit's own detector: a comment or a
// string literal naming ahoy.X is not a caller, and neither is a selector on
// some other package that happens to be spelt ahoy; only a selector on the
// imported ahoy package, under whatever name it is imported, is.
func TestAhoySelectorsCountsOnlyCode(t *testing.T) {
	src := []byte(`package p

import (
	"fmt"

	abcdahoy "github.com/intentdriven/abcd/internal/core/ahoy"
)

// ahoy.InComment is named only here.
func f() {
	fmt.Println("ahoy.InString")
	_ = abcdahoy.Called
}
`)
	got, err := ahoySelectors(src)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != "Called" {
		t.Fatalf("ahoySelectors = %v; want only [Called]", got)
	}
	unimported := []byte("package p\n\nvar ahoy struct{ Shadow int }\n\nvar _ = ahoy.Shadow\n")
	if got, _ := ahoySelectors(unimported); len(got) != 0 {
		t.Fatalf("a selector on a local value named ahoy counted as a caller: %v", got)
	}
}
