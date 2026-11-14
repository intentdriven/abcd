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
// (iss-2610171047088861). Both start from a reservation's term, its effective
// expiry less its claimed-at, so a copy of either kept here starts with that
// subtraction.
//
// What it measures: no non-test source of this package calls Sub with one
// side an expiry and the other a claimed-at, in either order, where
// termReckonings says which spellings count as each. What it cannot see: a
// term reached any other way, such as a local under another name handed in
// from another function, a value read through a helper or a struct field of
// another name, or arithmetic on Unix stamps; nor a copy of the rule in
// another package.
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
		for _, pos := range termReckonings(f) {
			found = append(found, fmt.Sprintf("%s:%d: reckons a reservation's term itself; call claim.HalfThrough or claim.RenewalTerm",
				p, fset.Position(pos).Line))
		}
	}
	if len(found) > 0 {
		t.Fatalf("the loop keeps its own copy of claim's renewal rule:\n%s", strings.Join(found, "\n"))
	}
}

// TestTermReckoningsSeesEachSpelling holds the matcher to the spellings the
// audit above names, and to code beside them it must leave alone.
func TestTermReckoningsSeesEachSpelling(t *testing.T) {
	for _, c := range []struct {
		name, body string
		flagged    bool
	}{
		{"both fields", `_ = r.EffectiveExpiry.Sub(r.ClaimedAt)`, true},
		{"reversed", `_ = r.ClaimedAt.Sub(r.EffectiveExpiry)`, true},
		{"expiry held in a local", `exp := r.EffectiveExpiry; _ = exp.Sub(r.ClaimedAt)`, true},
		{"expiry assigned later", `var exp time.Time; exp = r.EffectiveExpiry; _ = exp.Sub(r.ClaimedAt)`, true},
		{"claimed-at held in a local", `from := r.ClaimedAt; _ = r.EffectiveExpiry.Sub(from)`, true},
		{"both held in locals", `exp, from := r.EffectiveExpiry, r.ClaimedAt; _ = exp.Sub(from)`, true},
		{"claimed-at by name", `claimedAt := stamp(); _ = r.EffectiveExpiry.Sub(claimedAt)`, true},
		{"expiry by name", `_ = effectiveExpiry.Sub(r.ClaimedAt)`, true},
		{"inside the half-point sum", `_ = r.ClaimedAt.Add(r.EffectiveExpiry.Sub(r.ClaimedAt) / 2)`, true},
		{"time left", `_ = time.Until(r.EffectiveExpiry)`, false},
		{"compared with now", `_ = now.Before(r.EffectiveExpiry) || r.EffectiveExpiry.After(now)`, false},
		{"expiry less now", `_ = r.EffectiveExpiry.Sub(now)`, false},
		{"age since the claim", `_ = now.Sub(r.ClaimedAt)`, false},
		{"claimed-at less another time", `exp := r.ExpiresAt; _ = exp.Sub(r.ClaimedAt)`, false},
		{"another local less the claimed-at", `exp := r.EffectiveExpiry; _ = exp; other := now; _ = other.Sub(r.ClaimedAt)`, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			src := "package p\nfunc f() {\n" + strings.ReplaceAll(c.body, "; ", "\n") + "\n}\n"
			f, err := parser.ParseFile(token.NewFileSet(), "p.go", src, parser.SkipObjectResolution)
			if err != nil {
				t.Fatal(err)
			}
			if got := len(termReckonings(f)) > 0; got != c.flagged {
				t.Errorf("termReckonings of %q flagged %v, want %v", c.body, got, c.flagged)
			}
		})
	}
}

// termReckonings is each Sub call in f, inside a function, whose receiver and
// argument are an expiry and a claimed-at, in either order. An expiry is a
// selector named EffectiveExpiry, an identifier named effectiveExpiry, or an
// identifier the same function assigns from such a selector anywhere in its
// body (the order of statements is not followed, so it errs towards flagging);
// a claimed-at is the same for ClaimedAt and claimedAt. Only a plain
// identifier assigned straight from the selector is followed: one assigned
// through any other expression, or in another function, is not.
func termReckonings(f *ast.File) []token.Pos {
	var out []token.Pos
	for _, d := range f.Decls {
		fd, ok := d.(*ast.FuncDecl)
		if !ok || fd.Body == nil {
			continue
		}
		held := map[string]string{} // local name -> the field it was assigned from
		note := func(lhs []ast.Expr, rhs []ast.Expr) {
			if len(lhs) != len(rhs) {
				return
			}
			for i, l := range lhs {
				id, ok := l.(*ast.Ident)
				if !ok {
					continue
				}
				if s, ok := rhs[i].(*ast.SelectorExpr); ok && (s.Sel.Name == "EffectiveExpiry" || s.Sel.Name == "ClaimedAt") {
					held[id.Name] = s.Sel.Name
				}
			}
		}
		ast.Inspect(fd.Body, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.AssignStmt:
				note(n.Lhs, n.Rhs)
			case *ast.ValueSpec:
				lhs := make([]ast.Expr, len(n.Names))
				for i, id := range n.Names {
					lhs[i] = id
				}
				note(lhs, n.Values)
			}
			return true
		})
		field := func(e ast.Expr) string {
			switch e := e.(type) {
			case *ast.SelectorExpr:
				return e.Sel.Name
			case *ast.Ident:
				switch e.Name {
				case "effectiveExpiry":
					return "EffectiveExpiry"
				case "claimedAt":
					return "ClaimedAt"
				}
				return held[e.Name]
			}
			return ""
		}
		ast.Inspect(fd.Body, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok || len(call.Args) != 1 {
				return true
			}
			fn, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || fn.Sel.Name != "Sub" {
				return true
			}
			a, b := field(fn.X), field(call.Args[0])
			if a == "EffectiveExpiry" && b == "ClaimedAt" || a == "ClaimedAt" && b == "EffectiveExpiry" {
				out = append(out, call.Pos())
			}
			return true
		})
	}
	return out
}
