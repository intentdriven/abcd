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

// homeModeCreators are the calls that can create a folder: each is judged
// when the folder it creates is led by abcd's home.
var homeModeCreators = map[string]bool{
	"EnsureHomeScope": true, "EnsureRealDirAll": true, "EnsureRealDir": true,
	"CreateRunDir": true, "MkdirAll": true, "Mkdir": true,
}

// isResolverPath reports whether e is a call to this package's Path, the home
// itself or a folder in it.
func isResolverPath(e ast.Expr) bool {
	call, ok := e.(*ast.CallExpr)
	if !ok {
		return false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Path" {
		return false
	}
	x, ok := sel.X.(*ast.Ident)
	return ok && x.Name == "abcdhome"
}

// isDirMode reports whether e is DirMode spelled through this package.
func isDirMode(e ast.Expr) bool {
	m, ok := e.(*ast.SelectorExpr)
	if !ok || m.Sel.Name != "DirMode" {
		return false
	}
	x, ok := m.X.(*ast.Ident)
	return ok && x.Name == "abcdhome"
}

// dirModeNames returns the package-level constants and variables files
// declare as abcdhome.DirMode, which a writer may hand in its place.
func dirModeNames(files []*ast.File) map[string]bool {
	out := map[string]bool{}
	for _, f := range files {
		for _, d := range f.Decls {
			g, ok := d.(*ast.GenDecl)
			if !ok {
				continue
			}
			for _, sp := range g.Specs {
				vs, ok := sp.(*ast.ValueSpec)
				if !ok {
					continue
				}
				for i, n := range vs.Names {
					if i < len(vs.Values) && isDirMode(vs.Values[i]) {
						out[n.Name] = true
					}
				}
			}
		}
	}
	return out
}

// pathLocals returns the identifiers decl assigns from a call to this
// package's Path, so a folder held in a local is led by the home as the call
// is.
func pathLocals(decl ast.Node) map[string]bool {
	out := map[string]bool{}
	ast.Inspect(decl, func(n ast.Node) bool {
		if as, ok := n.(*ast.AssignStmt); ok && len(as.Lhs) == len(as.Rhs) {
			for i, r := range as.Rhs {
				if id, ok := as.Lhs[i].(*ast.Ident); ok && isResolverPath(r) {
					out[id.Name] = true
				}
			}
		}
		return true
	})
	return out
}

// homeModeCalls reports every call in f that can create abcd's home or a
// folder in it and hands a mode other than DirMode, and counts the calls it
// judged. A call is judged when it is fsutil.EnsureHomeScope, or a creator
// whose base is a home value (the D3 naming rule, locals included), a call to
// Path, or a local assigned from one. The mode passes when it is
// abcdhome.DirMode or a package name declared as it (allowed). The walk that
// creates the home creates it at the mode it is handed, so one writer handing
// 0o755 makes the home readable by every account whenever it runs first
// (iss-2610032205304585). A folder whose base is a variable walked down from
// the home (a loop over levels) is out of reach; its store's mode constant is
// declared as abcdhome.DirMode instead.
func homeModeCalls(fset *token.FileSet, f *ast.File, allowed map[string]bool) ([]homeSpelling, int) {
	var out []homeSpelling
	judged := 0
	for _, d := range f.Decls {
		locals := homeLocals(d)
		paths := pathLocals(d)
		ast.Inspect(d, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok || len(call.Args) < 2 {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || !homeModeCreators[sel.Sel.Name] {
				return true
			}
			if sel.Sel.Name != "EnsureHomeScope" {
				base := call.Args[0]
				id, isID := base.(*ast.Ident)
				if !isHomeValue(base, locals) && !isResolverPath(base) && !(isID && paths[id.Name]) {
					return true
				}
			}
			judged++
			mode := call.Args[len(call.Args)-1]
			if isDirMode(mode) {
				return true
			}
			if id, ok := mode.(*ast.Ident); ok && allowed[id.Name] {
				return true
			}
			out = append(out, homeSpelling{line: fset.Position(call.Pos()).Line, text: types.ExprString(mode)})
			return true
		})
	}
	return out, judged
}

// homeWriterFloor is how many home writers the tree held when the rule was
// written; a scan judging fewer is not looking at them (a renamed helper, a
// changed walk) and fails rather than passing on nothing.
const homeWriterFloor = 11

// TestEveryHomeWriterMakesTheHomePrivate is iss-2610032205304585: every
// writer that creates abcd's home, or a folder in it, hands
// fsutil.EnsureHomeScope abcdhome.DirMode, so the home is private to the
// account whichever command creates it first.
func TestEveryHomeWriterMakesTheHomePrivate(t *testing.T) {
	pkgs, _ := sourceTree(t)
	fsutilPkg := filepath.Join("internal", "fsutil")
	var lines []string
	judged := 0
	for dir, pp := range pkgs {
		if dir == fsutilPkg {
			continue
		}
		var files []*ast.File
		for _, f := range pp.files {
			files = append(files, f)
		}
		allowed := dirModeNames(files)
		for rel, f := range pp.files {
			found, n := homeModeCalls(pp.fset, f, allowed)
			judged += n
			for _, h := range found {
				lines = append(lines, fmt.Sprintf("%s:%d: %s", rel, h.line, h.text))
			}
		}
	}
	if judged < homeWriterFloor {
		t.Fatalf("the scan judged %d home writers, fewer than the %d the tree held; it is not looking at them", judged, homeWriterFloor)
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
		{`_, _ = fsutil.EnsureHomeScope(home, abcdhome.Rel(), 0o755)`, 1},
		{`_, _ = fsutil.EnsureHomeScope(home, abcdhome.Rel("history"), 0o700)`, 1},
		{`_, _ = fsutil.EnsureHomeScope(home, rel, perm)`, 1},
		{`_, _ = fsutil.EnsureHomeScope(home, abcdhome.Rel(), abcdhome.DirMode)`, 0},
		{`_ = fsutil.EnsureRealDirAll(home, abcdhome.Rel("runs"), 0o755)`, 1},
		{`_ = fsutil.EnsureRealDirAll(s.home, rel, storeDirPerm)`, 1},
		{`_ = fsutil.EnsureRealDirAll(home, abcdhome.Rel("lab"), abcdhome.DirMode)`, 0},
		{`_ = fsutil.EnsureRealDirAll(home, abcdhome.Rel("lab"), okMode)`, 0},
		{`_ = fsutil.EnsureRealDirAll(home, abcdhome.Rel("lab"), wideMode)`, 1},
		{`h, _ := os.UserHomeDir(); _ = fsutil.EnsureRealDirAll(h, abcdhome.Rel("x"), 0o755)`, 1},
		{`_ = os.MkdirAll(abcdhome.Path(home, "sources"), 0o755)`, 1},
		{`d := abcdhome.Path(home); _ = fsutil.EnsureRealDir(d, 0o755)`, 1},
		{`_, _ = fsutil.CreateRunDir(abcdhome.Path(home), "runs", "x", 0o755)`, 1},
		{`_ = fsutil.EnsureRealDirAll(repoRoot, ".abcd/work", 0o755)`, 0},
		{`_ = os.MkdirAll(filepath.Join(cwd, ".abcd"), 0o755)`, 0},
	} {
		src := "package p\n\nconst okMode = abcdhome.DirMode\n\nconst wideMode = 0o755\n\nfunc f() {\n\t" + c.call + "\n}\n"
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, "hostile.go", src, 0)
		if err != nil {
			t.Fatalf("parse: %v\n%s", err, src)
		}
		if got, _ := homeModeCalls(fset, f, dirModeNames([]*ast.File{f})); len(got) != c.want {
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
