package judge

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	gotypes "go/types"
	"io/fs"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// asked is pinned to the steps that ask through the router: every judgement
// type a file outside this package names by its constant, and only those, is
// in it. A step wired to the router without its type added to asked, or a type
// left in asked after its step stopped asking, fails here. Three spellings of a
// type that name no constant fail it too, rather than pass unseen: a dot
// import of the package, a conversion to its Type (parenthesised or not), and
// a string literal as the Type field of a composite literal of one of its
// types, the literal's type spelled or elided. The pin reads syntax, not
// types, so a string literal that reaches a Type any other way (an argument
// to a parameter of that type, an assignment to a Type field, a typed
// variable's initial value) is not seen; reading those needs go/types.
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
		var built error
		ours := func(e ast.Expr) bool {
			if st, ok := e.(*ast.StarExpr); ok {
				e = st.X
			}
			sel, ok := e.(*ast.SelectorExpr)
			if !ok {
				return false
			}
			x, ok := sel.X.(*ast.Ident)
			return ok && x.Name == local
		}
		var lit func(cl *ast.CompositeLit, typ ast.Expr)
		lit = func(cl *ast.CompositeLit, typ ast.Expr) {
			if ours(typ) {
				for _, el := range cl.Elts {
					kv, ok := el.(*ast.KeyValueExpr)
					if !ok {
						continue
					}
					if k, ok := kv.Key.(*ast.Ident); ok && k.Name == "Type" && literal(kv.Value) && built == nil {
						built = fmt.Errorf("%s gives a judgement type as a string literal (Type: %s), which this pin of asked "+
							"cannot read; name the type's constant instead", fset.Position(kv.Value.Pos()), gotypes.ExprString(kv.Value))
					}
				}
				return
			}
			var elem ast.Expr
			switch t := typ.(type) {
			case *ast.ArrayType:
				elem = t.Elt
			case *ast.MapType:
				elem = t.Value
			default:
				return
			}
			for _, el := range cl.Elts {
				if kv, ok := el.(*ast.KeyValueExpr); ok {
					el = kv.Value
				}
				if u, ok := el.(*ast.UnaryExpr); ok && u.Op == token.AND {
					el = u.X
				}
				if inner, ok := el.(*ast.CompositeLit); ok && inner.Type == nil {
					lit(inner, elem)
				}
			}
		}
		ast.Inspect(f, func(n ast.Node) bool {
			if cl, ok := n.(*ast.CompositeLit); ok && cl.Type != nil {
				lit(cl, cl.Type)
			}
			if call, ok := n.(*ast.CallExpr); ok {
				fun := ast.Unparen(call.Fun)
				if sel, ok := fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "Type" {
					if x, ok := sel.X.(*ast.Ident); ok && x.Name == local && built == nil {
						built = fmt.Errorf("%s builds a judgement type by conversion (%s.Type(...)), which this pin of asked "+
							"cannot read; name the type's constant instead", fset.Position(call.Pos()), local)
					}
				}
			}
			if sel, ok := n.(*ast.SelectorExpr); ok {
				if x, ok := sel.X.(*ast.Ident); ok && x.Name == local {
					if ty, ok := consts[sel.Sel.Name]; ok {
						named[ty] = true
					}
				}
			}
			return true
		})
		return built
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

// literal is an expression built from literals alone ("remedy-read", or
// "remedy" + "-read"): a constant naming no constant of the package.
func literal(e ast.Expr) bool {
	switch v := ast.Unparen(e).(type) {
	case *ast.BasicLit:
		return true
	case *ast.BinaryExpr:
		return literal(v.X) && literal(v.Y)
	default:
		return false
	}
}
