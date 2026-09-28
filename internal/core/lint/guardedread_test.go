package lint

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// readRepoFile refuses a path that leaves the repository lexically or through a
// link, reads an in-repo link through the path containment judged, and keeps a
// missing file's os.IsNotExist error for callers that treat absence as a state.
func TestReadRepoFile(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "real.md", "real")
	if err := os.Symlink(filepath.Join(root, "real.md"), filepath.Join(root, "bridge.md")); err != nil {
		t.Fatal(err)
	}
	symlinkOut(t, root, "leak.md", "secret")
	if err := os.Symlink("/dev/zero", filepath.Join(root, "zero.md")); err != nil {
		t.Fatal(err)
	}
	if b, err := readRepoFile(root, "bridge.md", 64); err != nil || string(b) != "real" {
		t.Errorf("in-repo link: got %q, %v", b, err)
	}
	for _, rel := range []string{"leak.md", "../x.md", "/etc/hosts"} {
		if _, err := readRepoFile(root, rel, 64); err == nil || !strings.Contains(err.Error(), "inside the repository") {
			t.Errorf("%s: want a containment refusal, got %v", rel, err)
		}
	}
	if _, err := readRepoFile(root, "zero.md", 64); err == nil {
		t.Error("a link to a device outside the repository must be refused")
	}
	if _, err := readRepoFile(root, "absent.md", 64); !os.IsNotExist(err) {
		t.Errorf("absent: want an IsNotExist error, got %v", err)
	}
}

// No read in the lint package goes around the guard: every production file reads
// through readRepoFile, readRepoAbs, readRepoLeaf or an fsutil guarded read, so a
// new rule that reaches for os.ReadFile is refused here rather than found by the
// next sweep. The check parses the source rather than matching text, so a read
// spelled through an import alias, a dot import, fs.ReadFile over os.DirFS, or a
// method on a handle (an os.Root's Open, an fs.FS's ReadFile) is caught too, and
// so is a file's content read out of git through the unbounded gitutil.Run (a
// `cat-file blob` or a `show`), which reads a committed record whole as surely as
// os.ReadFile reads a working-tree one (iss-2609261726015043); the bounded
// gitutil.RunCapped and RunCappedBytes pass. What it cannot see is a raw read
// done inside a callee package; the guard is this package's own source.
func TestLintReadsNothingUnguarded(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		data, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		hits, err := unguardedReads(name, data)
		if err != nil {
			t.Fatal(err)
		}
		for _, h := range hits {
			t.Errorf("%s reads without the guard (use readRepoFile, readRepoAbs, readRepoLeaf or an fsutil guarded read; out of git, gitutil.RunCapped)", h)
		}
	}
}

// The scanner itself, over the shapes a regular expression over `os.ReadFile(`
// let through: each planted read is named, and the guarded forms are not.
func TestUnguardedReadsSeesEverySpelling(t *testing.T) {
	const src = `package p

import (
	"io/fs"
	"io/ioutil"
	sys "os"
	. "os"

	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
)

func planted(root *sys.Root, fsys fs.FS) {
	sys.ReadFile("a")
	sys.OpenFile("b", 0, 0)
	fs.ReadFile(sys.DirFS("."), "c")
	root.Open("d")
	root.ReadFile("e")
	fsys.Open("f")
	ioutil.ReadFile("g")
	ReadFile("h")
	sys.OpenInRoot(".", "i")
	gitutil.Run(".", "cat-file", "blob", "HEAD:j")
	gitutil.Run(".", "show", "HEAD:k")
}

func guarded(root *sys.Root) {
	fsutil.ReadGuarded("a", 1)
	fsutil.ReadGuardedInRoot(root, "b", 1)
	sys.OpenRoot(".")
	sys.ReadDir(".")
	sys.Lstat("c")
	gitutil.RunCapped(".", 1, "cat-file", "blob", "HEAD:d")
	gitutil.Run(".", "cat-file", "-e", "HEAD:e")
	gitutil.Run(".", "rev-parse", "HEAD")
}
`
	hits, err := unguardedReads("planted.go", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"planted.go:14 sys.ReadFile", "planted.go:15 sys.OpenFile", "planted.go:16 fs.ReadFile",
		"planted.go:16 sys.DirFS", "planted.go:17 root.Open", "planted.go:18 root.ReadFile",
		"planted.go:19 fsys.Open", "planted.go:20 ioutil.ReadFile", "planted.go:21 ReadFile",
		"planted.go:22 sys.OpenInRoot", "planted.go:23 gitutil.Run cat-file blob",
		"planted.go:24 gitutil.Run show",
	}
	if strings.Join(hits, "\n") != strings.Join(want, "\n") {
		t.Errorf("got:\n%s\nwant:\n%s", strings.Join(hits, "\n"), strings.Join(want, "\n"))
	}
}

// unguardedReads names every call in one Go source file that opens or reads a
// file around the guard, as "file:line callee".
func unguardedReads(name string, src []byte) ([]string, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, name, src, parser.SkipObjectResolution)
	if err != nil {
		return nil, err
	}
	// The package-level functions that read or open, keyed by import path.
	banned := map[string]map[string]bool{
		"os":        {"ReadFile": true, "Open": true, "OpenFile": true, "DirFS": true, "OpenInRoot": true},
		"io/fs":     {"ReadFile": true},
		"io/ioutil": {"ReadFile": true},
	}
	// Any other receiver: a method with a reading name on a handle (an os.Root, an
	// fs.FS, an *os.File's directory) is a read the guard never saw.
	methods := map[string]bool{"Open": true, "OpenFile": true, "ReadFile": true}
	// gitutil.Run returns git's whole output: asked for a file's content at a
	// revision (`cat-file blob`, `show`), it is a read with no bound.
	const gitutilPath = "github.com/intentdriven/abcd/internal/gitutil"

	pkgOf := map[string]string{} // local name -> import path, for every import
	dot := map[string]bool{}     // import paths imported with "."
	for _, imp := range f.Imports {
		path, err := strconv.Unquote(imp.Path.Value)
		if err != nil {
			return nil, err
		}
		local := path[strings.LastIndex(path, "/")+1:]
		if imp.Name != nil {
			local = imp.Name.Name
		}
		switch local {
		case "_":
		case ".":
			dot[path] = true
		default:
			pkgOf[local] = path
		}
	}

	var hits []string
	hit := func(pos token.Pos, callee string) {
		hits = append(hits, name+":"+strconv.Itoa(fset.Position(pos).Line)+" "+callee)
	}
	ast.Inspect(f, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		switch fn := call.Fun.(type) {
		case *ast.SelectorExpr:
			x, isIdent := fn.X.(*ast.Ident)
			if isIdent {
				if path, isPkg := pkgOf[x.Name]; isPkg {
					if banned[path][fn.Sel.Name] {
						hit(call.Pos(), x.Name+"."+fn.Sel.Name)
					}
					if path == gitutilPath && fn.Sel.Name == "Run" {
						if read := gitContentRead(call.Args); read != "" {
							hit(call.Pos(), x.Name+".Run "+read)
						}
					}
					return true
				}
			}
			if methods[fn.Sel.Name] {
				recv := "(expr)"
				if isIdent {
					recv = x.Name
				}
				hit(call.Pos(), recv+"."+fn.Sel.Name)
			}
		case *ast.Ident:
			for path := range dot {
				if banned[path][fn.Name] {
					hit(call.Pos(), fn.Name)
				}
			}
		}
		return true
	})
	return hits, nil
}

// gitContentRead names the file-content read a git invocation's literal
// arguments ask for — "cat-file blob" or "show" — or "" when they ask for none.
func gitContentRead(args []ast.Expr) string {
	var lits []string
	for _, a := range args {
		if b, ok := a.(*ast.BasicLit); ok && b.Kind == token.STRING {
			if v, err := strconv.Unquote(b.Value); err == nil {
				lits = append(lits, v)
			}
		}
	}
	for i, v := range lits {
		switch {
		case v == "show":
			return "show"
		case v == "cat-file" && i+1 < len(lits) && lits[i+1] == "blob":
			return "cat-file blob"
		}
	}
	return ""
}
