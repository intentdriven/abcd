package dashboard

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// The dashboard is the only listener abcd has (adr-2610032150581128). The
// scanner below holds the tree to it, in the shape of the home resolver's
// TestOnlyTheHomeResolverNamesTheHome: it parses every non-test Go file under
// internal/ and cmd/ (testdata skipped) and reports every call that opens a
// listener. Only this package may make one.

// listenerFuncs are the package functions that open a listener, by import
// path: a call through any import name of the package is judged.
var listenerFuncs = map[string][]string{
	"net":                   {"Listen", "ListenTCP", "ListenUDP", "ListenUnix", "ListenUnixgram", "ListenPacket", "ListenIP", "ListenMulticastUDP", "FileListener", "FilePacketConn", "ListenConfig"},
	"crypto/tls":            {"Listen", "NewListener"},
	"net/http":              {"ListenAndServe", "ListenAndServeTLS", "Serve", "ServeTLS"},
	"net/http/httptest":     {"NewServer", "NewTLSServer", "NewUnstartedServer"},
	"net/http/fcgi":         {"Serve"},
	"syscall":               {"Listen", "Bind"},
	"golang.org/x/sys/unix": {"Listen", "Bind"},
}

// listenerMethods are method names that open a listener whatever value they
// are called on: (*net.ListenConfig).Listen, (*http.Server).ListenAndServe.
var listenerMethods = []string{"Listen", "ListenPacket", "ListenAndServe", "ListenAndServeTLS", "ServeTLS"}

// listenerPackage is the one package allowed to open a listener.
const listenerPackage = "internal/surface/dashboard"

// listenerTestDoubles are non-test files outside the package that open a
// listener for tests alone, each with its reason; the test also proves no
// non-test file imports their package, so none reaches the binary.
var listenerTestDoubles = map[string]string{
	"internal/adapter/hosting/cloudflare/cloudflaretest/fake.go": "the Cloudflare API fake the hosting adapter's tests serve with httptest; imported by tests only",
}

// listenerCall is one finding.
type listenerCall struct {
	line int
	call string
}

// listenerCalls reports every call in f that opens a listener.
func listenerCalls(fset *token.FileSet, f *ast.File) []listenerCall {
	names := map[string]string{}
	for _, imp := range f.Imports {
		path, _ := strconv.Unquote(imp.Path.Value)
		name := path[strings.LastIndex(path, "/")+1:]
		if imp.Name != nil {
			name = imp.Name.Name
		}
		names[name] = path
	}
	var out []listenerCall
	ast.Inspect(f, func(n ast.Node) bool {
		sel, ok := n.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if id, ok := sel.X.(*ast.Ident); ok && id.Obj == nil {
			if path, ok := names[id.Name]; ok {
				for _, fn := range listenerFuncs[path] {
					if sel.Sel.Name == fn {
						out = append(out, listenerCall{line: fset.Position(sel.Pos()).Line, call: path + "." + fn})
						return true
					}
				}
				return true
			}
		}
		for _, m := range listenerMethods {
			if sel.Sel.Name == m {
				out = append(out, listenerCall{line: fset.Position(sel.Pos()).Line, call: "." + m})
			}
		}
		return true
	})
	return out
}

// sourceFiles parses every non-test Go file under internal/ and cmd/.
func sourceFiles(t *testing.T, fset *token.FileSet, root string) map[string]*ast.File {
	t.Helper()
	files := map[string]*ast.File{}
	for _, top := range []string{"internal", "cmd"} {
		err := filepath.WalkDir(filepath.Join(root, top), func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if d.Name() == "testdata" {
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
			rel, _ := filepath.Rel(root, p)
			files[filepath.ToSlash(rel)] = f
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	return files
}

func TestOnlyTheDashboardOpensAListener(t *testing.T) {
	root := repoRoot(t)
	fset := token.NewFileSet()
	files := sourceFiles(t, fset, root)
	if len(files) < 500 {
		t.Fatalf("the scan read %d files; it is not looking at the tree", len(files))
	}
	var bad []string
	judged := 0
	for rel, f := range files {
		calls := listenerCalls(fset, f)
		if strings.HasPrefix(rel, listenerPackage+"/") {
			judged += len(calls)
			continue
		}
		if _, ok := listenerTestDoubles[rel]; ok {
			continue
		}
		for _, c := range calls {
			bad = append(bad, fmt.Sprintf("%s:%d: %s", rel, c.line, c.call))
		}
	}
	if judged == 0 {
		t.Fatal("the scan found no listener in the dashboard package; it is not looking at the calls")
	}
	sort.Strings(bad)
	for _, b := range bad {
		t.Errorf("%s opens a listener outside %s: the dashboard is the only listener abcd has (adr-2610032150581128)", b, listenerPackage)
	}
	// A test double's package reaches the binary only through an import from
	// a non-test file; there must be none.
	for double := range listenerTestDoubles {
		pkg := "github.com/intentdriven/abcd/" + filepath.ToSlash(filepath.Dir(double))
		for rel, f := range files {
			if filepath.ToSlash(filepath.Dir(rel)) == filepath.Dir(double) {
				continue
			}
			for _, imp := range f.Imports {
				if p, _ := strconv.Unquote(imp.Path.Value); p == pkg {
					t.Errorf("%s imports the listener test double %s: it would reach the binary", rel, pkg)
				}
			}
		}
	}
}

func TestListenerScannerIsArmed(t *testing.T) {
	for _, c := range []struct {
		src  string
		want int
	}{
		{`import "net"; func f() { net.Listen("tcp", ":0") }`, 1},
		{`import n "net"; func f() { n.ListenTCP("tcp", nil) }`, 1},
		{`import "net"; func f() { var lc net.ListenConfig; lc.Listen(nil, "tcp", ":0") }`, 2},
		{`import "net"; func f() { net.FileListener(nil) }`, 1},
		{`import "net/http"; func f() { http.ListenAndServe(":0", nil) }`, 1},
		{`import "net/http"; func f() { s := &http.Server{}; s.ListenAndServe() }`, 1},
		{`import "net/http"; func f() { http.Serve(nil, nil) }`, 1},
		{`import "crypto/tls"; func f() { tls.Listen("tcp", ":0", nil) }`, 1},
		{`import "net/http/httptest"; func f() { httptest.NewServer(nil) }`, 1},
		{`import "syscall"; func f() { syscall.Bind(0, nil) }`, 1},
		{`import "golang.org/x/sys/unix"; func f() { unix.Listen(0, 1) }`, 1},
		{`import "net"; func f() { net.Dial("tcp", "192.0.2.1:80") }`, 0},
		{`import "net/http"; func f() { http.Get("http://example.com") }`, 0},
		{`func f() { var x struct{ Listening bool }; _ = x.Listening }`, 0},
	} {
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, "x.go", "package x\n"+c.src, parser.SkipObjectResolution)
		if err != nil {
			t.Fatalf("%s: %v", c.src, err)
		}
		if got := len(listenerCalls(fset, f)); got != c.want {
			t.Errorf("%s: %d listener calls, want %d", c.src, got, c.want)
		}
	}
}
