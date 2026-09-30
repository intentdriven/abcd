package fsutil

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
)

// No production code outside this package calls flock: fsutil's two lock
// primitives are the one inter-process lock (iss-129, one-canonical-primitive).
// A test file may still flock directly, to stand in for another process or an
// older binary holding the lock.
func TestNoPackageOutsideFsutilCallsFlock(t *testing.T) {
	root, err := ModuleRoot(".")
	if err != nil {
		t.Fatal(err)
	}
	self := filepath.Join(root, "internal", "fsutil")
	fset := token.NewFileSet()
	var sites []string
	err = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if p == self || (p != root && (strings.HasPrefix(name, ".") || name == "testdata" || name == "vendor")) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
			return nil
		}
		f, err := parser.ParseFile(fset, p, nil, parser.SkipObjectResolution)
		if err != nil {
			return err
		}
		ast.Inspect(f, func(n ast.Node) bool {
			if sel, ok := n.(*ast.SelectorExpr); ok && (sel.Sel.Name == "Flock" || sel.Sel.Name == "FcntlFlock") {
				rel, _ := filepath.Rel(root, p)
				sites = append(sites, fmt.Sprintf("%s:%d", filepath.ToSlash(rel), fset.Position(sel.Pos()).Line))
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(sites) > 0 {
		t.Fatalf("flock called outside internal/fsutil; take fsutil.WithFileLock or fsutil.WithDirLock instead:\n%s", strings.Join(sites, "\n"))
	}
}
