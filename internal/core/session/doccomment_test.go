package session

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

// TestAddressKeyCommentsAreTheirOwn: AddressKey and addressKey each carry
// their own doc comment, which begins with the function's name, so neither
// reads as the other's.
func TestAddressKeyCommentsAreTheirOwn(t *testing.T) {
	f, err := parser.ParseFile(token.NewFileSet(), "codes.go", nil, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, d := range f.Decls {
		fn, ok := d.(*ast.FuncDecl)
		if !ok || (fn.Name.Name != "AddressKey" && fn.Name.Name != "addressKey") {
			continue
		}
		seen[fn.Name.Name] = true
		doc := ""
		if fn.Doc != nil {
			doc = fn.Doc.Text()
		}
		if !strings.HasPrefix(doc, fn.Name.Name+" ") {
			t.Errorf("%s's doc comment does not begin with its name:\n%s", fn.Name.Name, doc)
		}
		if fn.Name.Name == "AddressKey" && strings.Contains(doc, "addressKey is") {
			t.Errorf("AddressKey's doc comment carries addressKey's:\n%s", doc)
		}
	}
	if !seen["AddressKey"] || !seen["addressKey"] {
		t.Fatalf("codes.go declares %v, want AddressKey and addressKey", seen)
	}
}
