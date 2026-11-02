package cli

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

// TestEveryLayerNoteIsPrintedThroughNoteLine pins the one spelling of a
// resolution note's stderr line. RootsFor hands the HOME refusal over already
// prefixed "abcd: ", so a site that adds a prefix of its own prints it twice
// ("abcd abcd: HOME …", "abcd land: abcd: HOME …"), and a site that adds none
// prints the rules root's refusal unattributed. Every loop over a variable
// whose name ends in notes (notes, tnotes) in this package prints each note as
// fmt.Fprintln(w, noteLine(n)), the note's line alone; a print site written
// another way fails here, naming its position.
func TestEveryLayerNoteIsPrintedThroughNoteLine(t *testing.T) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, ".", func(fi os.FileInfo) bool {
		return !strings.HasSuffix(fi.Name(), "_test.go")
	}, 0)
	if err != nil {
		t.Fatalf("parse the surface package: %v", err)
	}
	var offenders []string
	seen := 0
	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			ast.Inspect(file, func(n ast.Node) bool {
				rs, ok := n.(*ast.RangeStmt)
				if !ok {
					return true
				}
				if id, ok := rs.X.(*ast.Ident); !ok || !strings.HasSuffix(strings.ToLower(id.Name), "notes") {
					return true
				}
				ast.Inspect(rs.Body, func(m ast.Node) bool {
					call, ok := m.(*ast.CallExpr)
					if !ok {
						return true
					}
					sel, ok := call.Fun.(*ast.SelectorExpr)
					if !ok || !strings.HasPrefix(sel.Sel.Name, "Fprint") {
						return true
					}
					if pkgIdent, ok := sel.X.(*ast.Ident); !ok || pkgIdent.Name != "fmt" {
						return true
					}
					seen++
					through := false
					if sel.Sel.Name == "Fprintln" && len(call.Args) == 2 {
						if c, ok := call.Args[1].(*ast.CallExpr); ok {
							if f, ok := c.Fun.(*ast.Ident); ok && f.Name == "noteLine" {
								through = true
							}
						}
					}
					if !through {
						offenders = append(offenders, fset.Position(call.Pos()).String())
					}
					return true
				})
				return true
			})
		}
	}
	if seen < 15 {
		t.Fatalf("found %d note prints, want at least 15: the scan no longer sees the print sites", seen)
	}
	if len(offenders) > 0 {
		t.Fatalf("a resolution note is printed without noteLine at:\n%s", strings.Join(offenders, "\n"))
	}
}
