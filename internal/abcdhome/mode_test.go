package abcdhome

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"path/filepath"
	"sort"
	"testing"
)

// homeModeCalls reports every call in f that can create abcd's home and
// hands a mode other than DirMode spelled through this package: a call to
// fsutil.EnsureHomeScope, and a call to fsutil.EnsureRealDirAll or
// fsutil.EnsureRealDir whose base is a home value. The walk that creates the
// home creates it at the mode it is handed, so one writer handing 0o755 makes
// the home readable by every account whenever it runs first
// (iss-2610032205304585).
func homeModeCalls(fset *token.FileSet, f *ast.File) []homeSpelling {
	var out []homeSpelling
	ast.Inspect(f, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		switch sel.Sel.Name {
		case "EnsureHomeScope":
		case "EnsureRealDirAll", "EnsureRealDir":
			if len(call.Args) < 2 || !isHomeValue(call.Args[0], nil) {
				return true
			}
		default:
			return true
		}
		mode := call.Args[len(call.Args)-1]
		if m, ok := mode.(*ast.SelectorExpr); ok && m.Sel.Name == "DirMode" {
			if x, ok := m.X.(*ast.Ident); ok && x.Name == "abcdhome" {
				return true
			}
		}
		out = append(out, homeSpelling{line: fset.Position(call.Pos()).Line, text: types.ExprString(mode)})
		return true
	})
	return out
}

// TestEveryHomeWriterMakesTheHomePrivate is iss-2610032205304585: every
// writer that creates abcd's home, or a folder in it, hands
// fsutil.EnsureHomeScope abcdhome.DirMode, so the home is private to the
// account whichever command creates it first.
func TestEveryHomeWriterMakesTheHomePrivate(t *testing.T) {
	pkgs, _ := sourceTree(t)
	fsutilPkg := filepath.Join("internal", "fsutil")
	var lines []string
	for dir, pp := range pkgs {
		if dir == fsutilPkg {
			continue
		}
		for rel, f := range pp.files {
			for _, h := range homeModeCalls(pp.fset, f) {
				lines = append(lines, fmt.Sprintf("%s:%d: %s", rel, h.line, h.text))
			}
		}
	}
	sort.Strings(lines)
	for _, l := range lines {
		t.Errorf("%s creates abcd's home at a mode other than abcdhome.DirMode; pass abcdhome.DirMode", l)
	}
}

// TestHomeModeScannerIsArmed proves the scanner reports a writer handing any
// other mode, a private-looking literal included, and passes the one spelling.
func TestHomeModeScannerIsArmed(t *testing.T) {
	for _, c := range []struct {
		call string
		want int
	}{
		{`fsutil.EnsureHomeScope(home, abcdhome.Rel(), 0o755)`, 1},
		{`fsutil.EnsureHomeScope(home, abcdhome.Rel("history"), 0o700)`, 1},
		{`fsutil.EnsureHomeScope(home, rel, perm)`, 1},
		{`fsutil.EnsureHomeScope(home, abcdhome.Rel(), abcdhome.DirMode)`, 0},
		{`fsutil.EnsureRealDirAll(home, abcdhome.Rel("runs"), 0o755)`, 1},
		{`fsutil.EnsureRealDirAll(s.home, rel, storeDirPerm)`, 1},
		{`fsutil.EnsureRealDirAll(home, abcdhome.Rel("lab"), abcdhome.DirMode)`, 0},
		{`fsutil.EnsureRealDirAll(repoRoot, ".abcd/work", 0o755)`, 0},
	} {
		src := "package p\n\nfunc f() {\n\t_, _ = " + c.call + "\n}\n"
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, "hostile.go", src, 0)
		if err != nil {
			t.Fatalf("parse: %v\n%s", err, src)
		}
		if got := homeModeCalls(fset, f); len(got) != c.want {
			t.Errorf("%s: the scanner reports %v, want %d finding(s)", c.call, got, c.want)
		}
	}
}

// TestTheHomeModesArePrivate pins the two modes: the home and its folders
// for the account alone, and the records in it read and written by it alone.
func TestTheHomeModesArePrivate(t *testing.T) {
	if DirMode != 0o700 || FileMode != 0o600 {
		t.Errorf("DirMode = %o, FileMode = %o; want 700 and 600", DirMode, FileMode)
	}
}
