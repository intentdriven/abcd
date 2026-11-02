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
// is a write the person never asked for, made hundreds of times an hour. The
// one exception is the session's fill file (the product thinker's ruling of
// 2026-10-11, "Allow one fill file"; spc-2610111627213138 step 5): exactly one
// call of the fill writer, rules.WriteFill, from one function of the front
// door, recordStatusFill, is admitted, and every other write primitive, and
// any second call of the fill writer, is refused by name.
// TestStatuslineWritesOnlyTheFillFile (internal/surface/cli) proves that
// dynamically for the paths it drives; this is the structural half, which
// holds for the paths no fixture reaches.
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
	// The rules loader's writers: the session ledger's, and the fill's. The
	// fill writer is the one admitted (admittedFillWrite); the rest are the
	// prompt hook's and never the status line's.
	"rules": {
		"WriteFill": true, "ResetFill": true,
		"SaveState": true, "ResetState": true, "PruneState": true,
	},
}

// admittedFillWrite is the one write the status verb's path may make: the
// fill writer, called from this function of this front-door file, once.
var admittedFillWrite = struct{ file, fn, call string }{"statusline.go", "recordStatusFill", "rules.WriteFill"}

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
	case "github.com/intentdriven/abcd/internal/core/rules":
		return "rules"
	}
	return ""
}

// writeSite is one write-primitive call: its line, the call as written, and
// the top-level function it is in ("" outside every function).
type writeSite struct {
	line int
	call string
	fn   string
}

// writeCalls reports every write-primitive call in f, as "line: call".
func writeCalls(fset *token.FileSet, f *ast.File) []string {
	var out []string
	for _, s := range writeSites(fset, f) {
		out = append(out, fmt.Sprintf("%d: %s", s.line, s.call))
	}
	return out
}

// writeSites reports every write-primitive call in f with the top-level
// function it is in, so the one admitted fill write is admitted by place and
// not by name alone.
func writeSites(fset *token.FileSet, f *ast.File) []writeSite {
	var out []writeSite
	for _, decl := range f.Decls {
		fn := ""
		if fd, ok := decl.(*ast.FuncDecl); ok {
			fn = fd.Name.Name
		}
		for _, s := range writeCallsIn(fset, f, decl) {
			s.fn = fn
			out = append(out, s)
		}
	}
	return out
}

// writeCallsIn reports every write-primitive call below node.
func writeCallsIn(fset *token.FileSet, f *ast.File, node ast.Node) []writeSite {
	imports := importNames(f)
	var out []writeSite
	report := func(n ast.Node, what string) {
		out = append(out, writeSite{line: fset.Position(n.Pos()).Line, call: what})
	}
	ast.Inspect(node, func(n ast.Node) bool {
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

// judgeStatusWrites is the guard's judgement over the write sites of the
// scanned files, keyed by file base name: every site is refused but the
// admitted fill write, and that one is refused too when it is not the only
// call of the fill writer. It returns one finding per refused site, sorted.
func judgeStatusWrites(sites map[string][]writeSite) []string {
	var found []string
	admitted := 0
	for file, ss := range sites {
		for _, s := range ss {
			where := fmt.Sprintf("%s:%d: %s", file, s.line, s.call)
			switch {
			case file == admittedFillWrite.file && s.fn == admittedFillWrite.fn && s.call == admittedFillWrite.call:
				admitted++
				if admitted > 1 {
					found = append(found, where+": a second call of the fill writer; the status line writes the session's fill file once, from "+admittedFillWrite.fn+" alone")
				}
			case s.call == admittedFillWrite.call:
				found = append(found, where+" (in "+orNone(s.fn)+"): a second call site of the fill writer; the status line writes the session's fill file from "+admittedFillWrite.fn+" alone")
			default:
				found = append(found, where+" (in "+orNone(s.fn)+"): the status verb writes only the session's fill file (iss-2610050556383525; the ruling of 2026-10-11); any other write on its path runs on every refresh of every session — move it out of the status line's path")
			}
		}
	}
	sort.Strings(found)
	return found
}

func orNone(fn string) string {
	if fn == "" {
		return "no function"
	}
	return fn
}

// scanSites parses each file and returns its write sites by base name.
func scanSites(t *testing.T, files []string) map[string][]writeSite {
	t.Helper()
	sites := map[string][]writeSite{}
	for _, p := range files {
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, p, nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatal(err)
		}
		sites[filepath.Base(p)] = append(sites[filepath.Base(p)], writeSites(fset, f)...)
	}
	return sites
}

// TestStatusVerbReachesNoWritePrimitive is the structural guard: no file on
// the status verb's own path calls a write primitive, but for the one call of
// the fill writer from recordStatusFill in the front door, which is there
// exactly once.
func TestStatusVerbReachesNoWritePrimitive(t *testing.T) {
	files := statusVerbSources(t)
	if len(files) < statusVerbFileFloor {
		t.Fatalf("the scan found %d files, fewer than the %d the verb had; it is not looking at the verb", len(files), statusVerbFileFloor)
	}
	sites := scanSites(t, files)
	for _, w := range judgeStatusWrites(sites) {
		t.Error(w)
	}
	admitted := 0
	for _, s := range sites[admittedFillWrite.file] {
		if s.fn == admittedFillWrite.fn && s.call == admittedFillWrite.call {
			admitted++
		}
	}
	if admitted != 1 {
		t.Errorf("%s calls %s %d time(s) in %s, want exactly once: the guard admits that one call and is not looking at it",
			admittedFillWrite.file, admittedFillWrite.call, admitted, admittedFillWrite.fn)
	}
}

// TestStatusVerbGuardRefusesASecondWrite is the guard watched failing: a
// front door that writes the fill file twice, calls the fill writer from a
// second function, or makes any other write beside the admitted one, is
// refused, each by name; the front door as it is passes.
func TestStatusVerbGuardRefusesASecondWrite(t *testing.T) {
	const head = "package cli\n\nimport (\n\t\"os\"\n\n" +
		"\t\"github.com/intentdriven/abcd/internal/core/rules\"\n)\n\n" +
		"var _ = os.Getenv\n\n"
	const admittedFn = "func recordStatusFill(s string) {\n\t_ = rules.WriteFill(\"\", s, 1, now())\n}\n"
	for _, c := range []struct {
		name string
		src  string
		want []string
	}{
		{"the admitted write alone", head + admittedFn, nil},
		{"the fill written twice", head + "func recordStatusFill(s string) {\n\t_ = rules.WriteFill(\"\", s, 1, now())\n\t_ = rules.WriteFill(\"\", s, 2, now())\n}\n",
			[]string{"a second call of the fill writer"}},
		{"a second call site", head + admittedFn + "func decideStatus(s string) {\n\t_ = rules.WriteFill(\"\", s, 1, now())\n}\n",
			[]string{"(in decideStatus): a second call site of the fill writer"}},
		{"another write beside it", head + "func recordStatusFill(s string) {\n\t_ = rules.WriteFill(\"\", s, 1, now())\n\t_ = os.WriteFile(s, nil, 0o600)\n}\n",
			[]string{"os.WriteFile (in recordStatusFill): the status verb writes only the session's fill file"}},
		{"the ledger saved", head + admittedFn + "func decideStatus(s string) {\n\t_ = rules.SaveState(s, rules.SessionState{})\n}\n",
			[]string{"rules.SaveState (in decideStatus)"}},
		{"the fill directory made", head + admittedFn + "func decideStatus(s string) {\n\t_ = rules.ResetFill(\"\", s)\n}\n",
			[]string{"rules.ResetFill (in decideStatus)"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, "statusline.go", c.src, parser.SkipObjectResolution)
			if err != nil {
				t.Fatal(err)
			}
			got := judgeStatusWrites(map[string][]writeSite{"statusline.go": writeSites(fset, f)})
			if len(got) != len(c.want) {
				t.Fatalf("findings = %q, want %d naming %q", got, len(c.want), c.want)
			}
			for i, w := range c.want {
				if !strings.Contains(got[i], w) {
					t.Errorf("finding %q does not name %q", got[i], w)
				}
			}
		})
	}
	// The admitted call is admitted only in the front door: the same
	// function in another file of the verb's path is a second call site.
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "compose.go", head+admittedFn, parser.SkipObjectResolution)
	if err != nil {
		t.Fatal(err)
	}
	if got := judgeStatusWrites(map[string][]writeSite{"compose.go": writeSites(fset, f)}); len(got) != 1 {
		t.Errorf("the fill writer called from compose.go: findings = %q, want one", got)
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
