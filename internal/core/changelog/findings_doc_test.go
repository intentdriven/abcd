package changelog

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

// TestFindingGuardFailedDocNamesEveryFailingList pins iss-2610112036089151.
// FindingGuardFailed's doc entry once named one of the three lists GuardFindings
// fails on (an open blocking finding) and left out the deleted records and the
// uncommitted open records, so a reader of the verdict type learnt a third of
// what the verdict means.
//
// The lists are read from FindingGuard itself rather than written out here: every
// slice field is a list the guard reports, and every one but Waived — populated
// on a PASS, by design — is a list whose non-emptiness fails the guard. So a
// fourth failing list added to the struct fails this test until the doc entry
// names it too.
func TestFindingGuardFailedDocNamesEveryFailingList(t *testing.T) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "findings.go", nil, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse findings.go: %v", err)
	}

	var statusDoc string
	var lists []string
	for _, decl := range f.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok || gd.Tok != token.TYPE {
			continue
		}
		for _, spec := range gd.Specs {
			ts := spec.(*ast.TypeSpec)
			switch ts.Name.Name {
			case "FindingGuardStatus":
				statusDoc = gd.Doc.Text()
			case "FindingGuard":
				st, ok := ts.Type.(*ast.StructType)
				if !ok {
					t.Fatalf("FindingGuard is not a struct")
				}
				for _, field := range st.Fields.List {
					if _, isSlice := field.Type.(*ast.ArrayType); !isSlice {
						continue
					}
					for _, name := range field.Names {
						if name.Name != "Waived" {
							lists = append(lists, name.Name)
						}
					}
				}
			}
		}
	}
	if statusDoc == "" {
		t.Fatal("FindingGuardStatus has no doc comment")
	}
	if len(lists) == 0 {
		t.Fatal("FindingGuard declares no slice fields; the test no longer reads the struct it means to")
	}

	// The FindingGuardFailed entry runs from its name to the next blank line.
	start := strings.Index(statusDoc, "FindingGuardFailed")
	if start < 0 {
		t.Fatalf("FindingGuardStatus's doc has no FindingGuardFailed entry:\n%s", statusDoc)
	}
	entry := statusDoc[start:]
	if end := strings.Index(entry, "\n\n"); end >= 0 {
		entry = entry[:end]
	}
	for _, name := range lists {
		if !strings.Contains(entry, name) {
			t.Errorf("the FindingGuardFailed doc entry does not name %s, a list GuardFindings fails on:\n%s", name, entry)
		}
	}
}
