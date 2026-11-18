package autonomy

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"
)

// TestTheOwedReadTakesTheRenewalDuePointFromClaim: the owed-read's next
// renewal and wake time are the instant a reservation is half through its
// term, which is claim's rule (claim.HalfThroughAt, which claim.HalfThrough
// also answers by), so a change to claim's due point moves the wake time with
// it (iss-2611141342176969). The implement loop's source test holds this
// package's sources to the same rule with its fuller matcher.
//
// What it measures: reservationItems in owedread.go calls claim.HalfThroughAt,
// and no non-test source of this package calls Sub on a selector named
// EffectiveExpiry or ClaimedAt with the other as its argument. What it cannot
// see: a term reached through locals, helpers, other field names or Unix
// stamps (the loop's matcher follows locals), nor whether the instant called
// for is the one the wake time is set from.
func TestTheOwedReadTakesTheRenewalDuePointFromClaim(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	var found []string
	calls := false
	for _, p := range files {
		if strings.HasSuffix(p, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, p, nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatal(err)
		}
		for _, d := range f.Decls {
			fd, ok := d.(*ast.FuncDecl)
			if !ok || fd.Body == nil {
				continue
			}
			ast.Inspect(fd.Body, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				fn, ok := call.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				if pkg, ok := fn.X.(*ast.Ident); ok && pkg.Name == "claim" && fn.Sel.Name == "HalfThroughAt" &&
					p == "owedread.go" && fd.Name.Name == "reservationItems" {
					calls = true
				}
				if fn.Sel.Name != "Sub" || len(call.Args) != 1 {
					return true
				}
				a, b := selName(fn.X), selName(call.Args[0])
				if a == "EffectiveExpiry" && b == "ClaimedAt" || a == "ClaimedAt" && b == "EffectiveExpiry" {
					found = append(found, fmt.Sprintf("%s:%d: reckons a reservation's term itself; call claim.HalfThroughAt",
						p, fset.Position(call.Pos()).Line))
				}
				return true
			})
		}
	}
	if !calls {
		found = append(found, "owedread.go: reservationItems does not call claim.HalfThroughAt for the renewal due point")
	}
	if len(found) > 0 {
		t.Fatalf("the owed-read keeps its own copy of claim's renewal due point:\n%s", strings.Join(found, "\n"))
	}
}

// selName is the field a selector expression names, "" for anything else.
func selName(e ast.Expr) string {
	if s, ok := e.(*ast.SelectorExpr); ok {
		return s.Sel.Name
	}
	return ""
}
