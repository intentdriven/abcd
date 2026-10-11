package memory

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"strconv"
	"strings"
	"testing"
)

// TestRendersNameTheBinaryNotAFrontDoor is the iss-44 detector for the third
// parity instance: plain-CLI `abcd library ask` output headed as a plugin slash
// command. internal/core is transport-agnostic — the same bytes are rendered
// through the plugin surface, the bare CLI, and any later transport — so a
// render that spells a slash command tells two of those three callers something
// untrue about how they invoked it.
//
// It inspects this package's string literals rather than one render's output,
// because the defect is a habit of writing (three headings and six remedy lines
// carried it) and pinning one call site would leave the next one free. Literals
// only: prose in a comment may name the surface it is explaining.
func TestRendersNameTheBinaryNotAFrontDoor(t *testing.T) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, ".", func(fi fs.FileInfo) bool {
		return !strings.HasSuffix(fi.Name(), "_test.go")
	}, 0)
	if err != nil {
		t.Fatalf("parsing the package: %v", err)
	}
	if len(pkgs) == 0 {
		t.Fatal("no package parsed; the detector would pass vacuously")
	}
	for _, pkg := range pkgs {
		for name, file := range pkg.Files {
			ast.Inspect(file, func(n ast.Node) bool {
				lit, ok := n.(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING {
					return true
				}
				text, err := strconv.Unquote(lit.Value)
				if err != nil {
					text = lit.Value
				}
				if !strings.Contains(text, "/abcd:") {
					return true
				}
				t.Errorf("%s:%d names the `/abcd:` plugin surface in a rendered string, from the "+
					"transport-agnostic core:\n\t%s\nRenders spell the binary invocation "+
					"(`abcd library ask`), so the same bytes stay true whichever front door produced "+
					"them (iss-44).", name, fset.Position(lit.Pos()).Line, text)
				return true
			})
		}
	}
}
