package judge

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// asked is pinned to the steps that ask through the router: every judgement
// type a file outside this package names, and only those, is in it. A step
// wired to the router without its type added to asked, or a type left in
// asked after its step stopped asking, fails here. A file that dot-imports
// the package fails it too: its unqualified names cannot be read as the
// package's, so the pin fails closed rather than miss them.
func TestAskedIsTheTypesTheCallersName(t *testing.T) {
	consts := map[string]Type{}
	fset := token.NewFileSet()
	jf, err := parser.ParseFile(fset, "judge.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range jf.Decls {
		gd, ok := d.(*ast.GenDecl)
		if !ok || gd.Tok != token.CONST {
			continue
		}
		for _, sp := range gd.Specs {
			vs := sp.(*ast.ValueSpec)
			if id, ok := vs.Type.(*ast.Ident); !ok || id.Name != "Type" {
				continue
			}
			for i, n := range vs.Names {
				v, err := strconv.Unquote(vs.Values[i].(*ast.BasicLit).Value)
				if err != nil {
					t.Fatal(err)
				}
				consts[n.Name] = Type(v)
			}
		}
	}
	if len(consts) != len(registry) {
		t.Fatalf("read %d type constants, the registry has %d", len(consts), len(registry))
	}
	named := map[Type]bool{}
	const importPath = "github.com/intentdriven/abcd/internal/core/decide/judge"
	err = filepath.WalkDir(filepath.Join("..", "..", ".."), func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == "testdata" || filepath.Base(path) == "judge" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		f, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
		if err != nil {
			return err
		}
		local := ""
		for _, im := range f.Imports {
			if p, _ := strconv.Unquote(im.Path.Value); p == importPath {
				local = "judge"
				if im.Name != nil {
					local = im.Name.Name
				}
			}
		}
		if local == "." {
			return fmt.Errorf("%s dot-imports the judge package, so the judgement types it names cannot be told apart from "+
				"its own identifiers and this pin of asked would miss them; import the package by name", path)
		}
		if local == "" {
			return nil
		}
		ast.Inspect(f, func(n ast.Node) bool {
			if sel, ok := n.(*ast.SelectorExpr); ok {
				if x, ok := sel.X.(*ast.Ident); ok && x.Name == local {
					if ty, ok := consts[sel.Sel.Name]; ok {
						named[ty] = true
					}
				}
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	var got []Type
	for ty := range named {
		got = append(got, ty)
	}
	slices.Sort(got)
	want := slices.Clone(asked)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Fatalf("the types the callers name are %v; asked is %v", got, want)
	}
}
