package ahoy

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
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
// then the module's non-test Go files outside the package for a selector
// naming each (ahoy.Name). Tests do not count as callers: a function reached
// only by its own test is exactly the scaffolding the rule is about.
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
	selector := regexp.MustCompile(`\bahoy\.([A-Z][A-Za-z0-9_]*)`)
	err = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if p == self || name == ".git" || name == "node_modules" || (strings.HasPrefix(name, ".") && p != root) {
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
		for _, m := range selector.FindAllStringSubmatch(string(data), -1) {
			if _, ok := exported[m[1]]; ok {
				exported[m[1]] = true
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
