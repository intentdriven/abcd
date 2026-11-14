package loop

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"
)

// TestTheLoopTakesTheRenewalRuleFromClaim: when a reservation is due for
// renewal and the term a renewal pushes are claim's rules (claim.HalfThrough,
// claim.RenewalTerm), which the load gate's renewal keeps too
// (iss-2610171047088861). Both are reckoned from a reservation's term, its
// effective expiry less its claimed-at, so no source of this package may
// compute that term itself: a copy kept here could drift from claim's.
func TestTheLoopTakesTheRenewalRuleFromClaim(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	var found []string
	for _, p := range files {
		if strings.HasSuffix(p, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, p, nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(f, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok || len(call.Args) != 1 {
				return true
			}
			fn, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || fn.Sel.Name != "Sub" {
				return true
			}
			from, ok1 := fn.X.(*ast.SelectorExpr)
			to, ok2 := call.Args[0].(*ast.SelectorExpr)
			if ok1 && ok2 && from.Sel.Name == "EffectiveExpiry" && to.Sel.Name == "ClaimedAt" {
				found = append(found, fmt.Sprintf("%s:%d: reckons a reservation's term itself; call claim.HalfThrough or claim.RenewalTerm",
					p, fset.Position(call.Pos()).Line))
			}
			return true
		})
	}
	if len(found) > 0 {
		t.Fatalf("the loop keeps its own copy of claim's renewal rule:\n%s", strings.Join(found, "\n"))
	}
}
