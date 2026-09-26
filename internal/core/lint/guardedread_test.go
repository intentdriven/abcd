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
// method on a handle (an os.Root's Open, an fs.FS's ReadFile) is caught too. What
// it cannot see is a raw read done inside a callee package; the guard is this
// package's own source.
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
			t.Errorf("%s reads without the guard (use readRepoFile, readRepoAbs, readRepoLeaf or an fsutil guarded read)", h)
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
}

func guarded(root *sys.Root) {
	fsutil.ReadGuarded("a", 1)
	fsutil.ReadGuardedInRoot(root, "b", 1)
	sys.OpenRoot(".")
	sys.ReadDir(".")
	sys.Lstat("c")
}
`
	hits, err := unguardedReads("planted.go", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"planted.go:13 sys.ReadFile", "planted.go:14 sys.OpenFile", "planted.go:15 fs.ReadFile",
		"planted.go:15 sys.DirFS", "planted.go:16 root.Open", "planted.go:17 root.ReadFile",
		"planted.go:18 fsys.Open", "planted.go:19 ioutil.ReadFile", "planted.go:20 ReadFile",
		"planted.go:21 sys.OpenInRoot",
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
