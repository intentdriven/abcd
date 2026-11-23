package history

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gitutil"
)

// TestNoPrivateRootSHAPattern pins iss-2610140813542607: the store's root-commit
// key is checked by gitutil.IsFullSHA, the one canonical primitive, never by a
// regexp this package compiles for itself. A private copy can drift from the
// shared check — the drift that once made every history verb refuse a SHA-256
// repository. The test parses every Go file in the package and fails on any
// regexp literal that agrees with IsFullSHA on every probe below, whatever its
// spelling, so a reworded copy is caught as well as a verbatim one.
func TestNoPrivateRootSHAPattern(t *testing.T) {
	probes := []string{
		strings.Repeat("a", 40),
		strings.Repeat("0", 64),
		strings.Repeat("a", 39),
		strings.Repeat("a", 41),
		strings.Repeat("a", 63),
		strings.Repeat("a", 65),
		strings.Repeat("A", 40),
		strings.Repeat("g", 40),
		strings.Repeat("a", 40) + "\n",
		"",
	}
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	for _, name := range files {
		f, err := parser.ParseFile(fset, name, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		ast.Inspect(f, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok || len(call.Args) != 1 {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || (sel.Sel.Name != "MustCompile" && sel.Sel.Name != "Compile") {
				return true
			}
			if pkg, ok := sel.X.(*ast.Ident); !ok || pkg.Name != "regexp" {
				return true
			}
			lit, ok := call.Args[0].(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				return true
			}
			pattern, err := strconv.Unquote(lit.Value)
			if err != nil {
				return true
			}
			re, err := regexp.Compile(pattern)
			if err != nil {
				return true
			}
			for _, p := range probes {
				if re.MatchString(p) != gitutil.IsFullSHA(p) {
					return true
				}
			}
			t.Errorf("%s: regexp %q is a private copy of gitutil.IsFullSHA; call gitutil.IsFullSHA instead", fset.Position(lit.Pos()), pattern)
			return true
		})
	}
}
