package statusline

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// The status verb is read-only by rule (iss-2610050556383525): the harness
// runs it on every refresh of every session, so a write anywhere on its path
// is a write the person never asked for, made hundreds of times an hour.
// TestStatuslineWritesNothing (internal/surface/cli) proves that dynamically
// for the paths it drives; this is the structural half, which holds for the
// paths no fixture reaches.
//
// WHY A FILE SCAN AND NOT A CALL-GRAPH WALK. The verb's own code is this
// package and one front-door file, internal/surface/cli/statusline.go, and
// this package exists for the status line alone: everything in it is on the
// verb's path or on the board's read of the same row, and nothing in it has
// any business writing (the setting is WRITTEN by ahoy's install step, never
// here — ReadSettingsFile is the one reader both share). So "every non-test
// file of the package, plus the front-door file" is the verb's call graph
// within its own packages, over-approximated, and a scan of it for the write
// primitives is the same answer a call-graph walk would give at a fraction of
// the machinery: a precise walk needs go/types over the whole module, or
// golang.org/x/tools, which is not a dependency. Calls OUT of these files —
// ahoy.Managed, gitutil, mode.ReadAt, fsutil's home reads — are outside the
// scan; the end-to-end test is what covers them, and a new call into a writer
// from here (mode.SetAt, say) is caught by name below.

// writeFuncs are the package-qualified calls that create, modify or remove a
// filesystem entry. A qualifier not listed here is not judged by this map.
var writeFuncs = map[string]map[string]bool{
	"os": {
		"WriteFile": true, "Create": true, "CreateTemp": true, "MkdirTemp": true,
		"Mkdir": true, "MkdirAll": true, "Remove": true, "RemoveAll": true,
		"Rename": true, "Symlink": true, "Link": true, "Truncate": true,
		"Chmod": true, "Chown": true, "Lchown": true, "Chtimes": true,
	},
	"ioutil": {"WriteFile": true, "TempFile": true, "TempDir": true},
	"syscall": {
		"Mkdir": true, "Unlink": true, "Rmdir": true, "Rename": true,
		"Creat": true, "Truncate": true, "Ftruncate": true, "Symlink": true, "Link": true,
	},
	// The mode store's writer: the badge READS the store and must never set it.
	"mode": {"Set": true, "SetAt": true},
}

// fsutilWritePrefixes are fsutil's write families. Every fsutil entry point
// that creates or replaces something is named for it — Write*, Ensure*,
// Create*, Append*, and the two lock helpers, which create their lock file.
var fsutilWritePrefixes = []string{"Write", "Ensure", "Create", "Append", "WithFileLock", "WithDirLock"}

// writeMethods are method names that write whatever their receiver is: an
// *os.Root, an *os.File, a store. A selector call by one of these names on a
// receiver that is not a package is reported, whatever the receiver's type,
// because a file scan cannot see types and none of these names has a
// read-only meaning anywhere in the standard library or in abcd.
var writeMethods = map[string]bool{
	"WriteFile": true, "Create": true, "Mkdir": true, "MkdirAll": true,
	"Remove": true, "RemoveAll": true, "Rename": true, "Symlink": true,
	"Link": true, "Truncate": true, "Chmod": true, "Chown": true, "Lchown": true,
	"Chtimes": true, "WriteString": true, "WriteAt": true,
}

// readOnlyOpenFlags are the only os.O_* / syscall.O_* names an os.OpenFile
// flag argument may combine. Anything else in the flag — O_WRONLY, O_RDWR,
// O_CREATE, O_TRUNC, O_APPEND, a variable, a literal — is a write or might be.
var readOnlyOpenFlags = map[string]bool{
	"O_RDONLY": true, "O_NOFOLLOW": true, "O_DIRECTORY": true,
	"O_CLOEXEC": true, "O_NONBLOCK": true,
}

// isReadOnlyFlag reports whether e is an |-combination of read-only open
// flags spelled through os or syscall, and nothing else.
func isReadOnlyFlag(e ast.Expr) bool {
	switch x := e.(type) {
	case *ast.ParenExpr:
		return isReadOnlyFlag(x.X)
	case *ast.BinaryExpr:
		return x.Op == token.OR && isReadOnlyFlag(x.X) && isReadOnlyFlag(x.Y)
	case *ast.SelectorExpr:
		pkg, ok := x.X.(*ast.Ident)
		return ok && (pkg.Name == "os" || pkg.Name == "syscall") && readOnlyOpenFlags[x.Sel.Name]
	}
	return false
}

// importNames maps each import's local name in f to its path, so a renamed
// import (`stdos "os"`) is judged by what it imports, not by how it is spelled.
func importNames(f *ast.File) map[string]string {
	out := map[string]string{}
	for _, imp := range f.Imports {
		p := strings.Trim(imp.Path.Value, `"`)
		name := filepath.Base(p)
		if imp.Name != nil {
			name = imp.Name.Name
		}
		out[name] = p
	}
	return out
}

// qualifierFor maps an import path to the key the tables above use.
func qualifierFor(path string) string {
	switch path {
	case "os", "syscall", "io/ioutil":
		return filepath.Base(path)
	case "github.com/intentdriven/abcd/internal/fsutil":
		return "fsutil"
	case "github.com/intentdriven/abcd/internal/core/mode":
		return "mode"
	}
	return ""
}

// writeCalls reports every write-primitive call in f, as "line: call".
func writeCalls(fset *token.FileSet, f *ast.File) []string {
	imports := importNames(f)
	var out []string
	report := func(n ast.Node, what string) {
		out = append(out, fmt.Sprintf("%d: %s", fset.Position(n.Pos()).Line, what))
	}
	ast.Inspect(f, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		name := sel.Sel.Name
		if id, ok := sel.X.(*ast.Ident); ok {
			if path, isPkg := imports[id.Name]; isPkg {
				q := qualifierFor(path)
				switch {
				case q == "os" && name == "OpenFile":
					if len(call.Args) < 2 || !isReadOnlyFlag(call.Args[1]) {
						report(call, types.ExprString(call.Fun)+" with flags "+exprOrNone(call.Args, 1))
					}
				case writeFuncs[q][name]:
					report(call, types.ExprString(call.Fun))
				case q == "fsutil" && hasAnyPrefix(name, fsutilWritePrefixes):
					report(call, types.ExprString(call.Fun))
				}
				return true
			}
		}
		if writeMethods[name] {
			report(call, types.ExprString(call.Fun))
		}
		return true
	})
	return out
}

func exprOrNone(args []ast.Expr, i int) string {
	if i < len(args) {
		return types.ExprString(args[i])
	}
	return "(none)"
}

func hasAnyPrefix(s string, prefixes []string) bool {
	for _, p := range prefixes {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}

// statusVerbSources are the files the scan reads: every non-test Go file of
// this package, and the verb's front-door file. The front door is named
// rather than globbed because the cli package holds every other verb too.
func statusVerbSources(t *testing.T) []string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	mod, err := fsutil.ModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	pkg, err := filepath.Glob(filepath.Join(mod, "internal", "core", "statusline", "*.go"))
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, p := range pkg {
		if !strings.HasSuffix(p, "_test.go") {
			out = append(out, p)
		}
	}
	front := filepath.Join(mod, "internal", "surface", "cli", "statusline.go")
	if _, err := os.Stat(front); err != nil {
		t.Fatalf("the status verb's front door is not where the scan looks: %v", err)
	}
	return append(out, front)
}

// statusVerbFileFloor is how many files the scan read when the rule was
// written (2026-10-05: the package's six non-test files and the front
// door). A scan reading fewer is not looking at the verb; a file removed on
// purpose lowers it in the same change.
const statusVerbFileFloor = 7

// TestStatusVerbReachesNoWritePrimitive is the structural guard: no file on
// the status verb's own path calls a write primitive.
func TestStatusVerbReachesNoWritePrimitive(t *testing.T) {
	files := statusVerbSources(t)
	if len(files) < statusVerbFileFloor {
		t.Fatalf("the scan found %d files, fewer than the %d the verb had; it is not looking at the verb", len(files), statusVerbFileFloor)
	}
	var found []string
	for _, p := range files {
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, p, nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatal(err)
		}
		for _, w := range writeCalls(fset, f) {
			found = append(found, filepath.Base(p)+":"+w)
		}
	}
	sort.Strings(found)
	for _, w := range found {
		t.Errorf("%s: the status verb is read-only (iss-2610050556383525); a write on its path runs on every refresh of every session — move it out of the status line's path", w)
	}
}

// TestWriteScannerIsArmed proves the scanner reports each write family, a
// renamed import and a write-flagged open included, and passes the reads the
// verb actually makes.
func TestWriteScannerIsArmed(t *testing.T) {
	for _, c := range []struct {
		body string
		want int
	}{
		{`_ = os.WriteFile(p, nil, 0o600)`, 1},
		{`_, _ = os.Create(p)`, 1},
		{`_ = os.MkdirAll(p, 0o700)`, 1},
		{`_ = os.Remove(p)`, 1},
		{`_ = os.Rename(p, q)`, 1},
		{`_, _ = os.OpenFile(p, os.O_WRONLY|os.O_CREATE, 0o600)`, 1},
		{`_, _ = os.OpenFile(p, os.O_RDWR, 0)`, 1},
		{`_, _ = os.OpenFile(p, flags, 0)`, 1},
		{`_ = fsutil.WriteFileAtomic(p, nil, 0o600)`, 1},
		{`_ = fsutil.EnsureRealDirAll(p, q, 0o700)`, 1},
		{`_, _ = fsutil.EnsureHomeScope(p, q, 0o700)`, 1},
		{`_ = mode.SetAt(p, s)`, 1},
		{`_ = root.WriteFile(p, nil, 0o600)`, 1},
		{`_ = root.Remove(p)`, 1},
		{`_, _ = f.WriteString(p)`, 1},
		{`_ = stdos.WriteFile(p, nil, 0o600)`, 1},
		{`_, _ = os.OpenFile(p, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_DIRECTORY, 0)`, 0},
		{`_, _ = os.ReadFile(p)`, 0},
		{`_, _ = os.Lstat(p)`, 0},
		{`_, _, _ = fsutil.ReadHomeDeclaration(p, q, 1)`, 0},
		{`_ = fsutil.IsRealDir(p)`, 0},
		{`_, _ = mode.ReadAt(p)`, 0},
		{`_ = os.Getenv(p)`, 0},
	} {
		src := "package x\n\nimport (\n\t\"os\"\n\tstdos \"os\"\n\t\"syscall\"\n\n" +
			"\t\"github.com/intentdriven/abcd/internal/core/mode\"\n" +
			"\t\"github.com/intentdriven/abcd/internal/fsutil\"\n)\n\n" +
			"var _, _, _, _, _ = os.Getenv, stdos.Getenv, syscall.Getpid, mode.ReadAt, fsutil.IsRealDir\n\n" +
			"func f() {\n\t" + c.body + "\n}\n"
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, "x.go", src, parser.SkipObjectResolution)
		if err != nil {
			t.Fatalf("%s: %v", c.body, err)
		}
		if got := writeCalls(fset, f); len(got) != c.want {
			t.Errorf("%s: reported %d (%v), want %d", c.body, len(got), got, c.want)
		}
	}
}
