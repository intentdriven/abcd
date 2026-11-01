package cli

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// The question limits a repository sets reach every reader through one
// resolver (itd-2610301914350541 decision 6). question.Default stays the one
// statement of the defaults and the bundled layer the resolver reads last, so
// a caller that passes it by name to a function that judges, states or
// renders the limits, or reads one of its three configurable figures, would
// hold a question to abcd's defaults in a repository that set its own.

// defaultLimitsAllowed are the files still allowed to pass question.Default by
// name: the resolver, the asking-sync command (the pages ship with the plugin
// and state the defaults, decision 12), and the rules package's one
// bundled-layer statement of the GRILL domain, which rules.Load delegates
// through. Tests are allowed everywhere and are not walked.
var defaultLimitsAllowed = []string{
	"internal/core/layered/question.go",
	"cmd/asking-sync/main.go",
	"internal/core/rules/grill.go",
}

// limitsTakers are the functions that judge, state or render the limits:
// question's check, the shared decision the hook and the verb call, the chip
// grammar, row split and asking rules, and the asking-sync renderers.
var limitsTakers = map[string]bool{
	"CheckLimits": true, "Decide": true, "AskingRules": true, "AskingRulesFrom": true, "ChipRole": true,
	"IsAbcds": true, "Rows": true, "render": true, "renderDrafter": true, "syncTarget": true,
}

// configurableFields are the three Limits fields a repository sets.
var configurableFields = map[string]bool{"Rows": true, "LabelWords": true, "MeaningSentences": true}

// defaultLimitsUses names each place in the Go file f that
// passes question.Default to a limits taker or reads one of its configurable
// fields. Inside package question itself Default is unqualified.
func defaultLimitsUses(fset *token.FileSet, f *ast.File) []string {
	inQuestion := f.Name.Name == "question"
	isDefault := func(e ast.Expr) bool {
		switch x := e.(type) {
		case *ast.SelectorExpr:
			id, ok := x.X.(*ast.Ident)
			return ok && id.Name == "question" && x.Sel.Name == "Default"
		case *ast.Ident:
			return inQuestion && x.Name == "Default"
		}
		return false
	}
	var out []string
	ast.Inspect(f, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.CallExpr:
			name := ""
			switch fn := x.Fun.(type) {
			case *ast.SelectorExpr:
				name = fn.Sel.Name
			case *ast.Ident:
				name = fn.Name
			}
			if !limitsTakers[name] {
				return true
			}
			for _, a := range x.Args {
				if isDefault(a) {
					out = append(out, fset.Position(x.Pos()).String()+": question.Default passed to "+name)
				}
			}
		case *ast.SelectorExpr:
			if configurableFields[x.Sel.Name] && isDefault(x.X) {
				out = append(out, fset.Position(x.Pos()).String()+": question.Default."+x.Sel.Name+" read")
			}
		}
		return true
	})
	return out
}

// TestOnlyTheResolverPassesTheDefaultLimits walks every non-test Go file of
// the module's internal/ and cmd/ trees and fails on any use of
// question.Default as the limits outside defaultLimitsAllowed.
func TestOnlyTheResolverPassesTheDefaultLimits(t *testing.T) {
	root := repoRootForTest(t)
	var bad []string
	for _, top := range []string{"internal", "cmd"} {
		err := filepath.WalkDir(filepath.Join(root, top), func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if d.Name() == "testdata" {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
				return nil
			}
			rel, err := filepath.Rel(root, p)
			if err != nil {
				return err
			}
			if slices.Contains(defaultLimitsAllowed, filepath.ToSlash(rel)) {
				return nil
			}
			src, err := os.ReadFile(p)
			if err != nil {
				return err
			}
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, filepath.ToSlash(rel), src, 0)
			if err != nil {
				return err
			}
			bad = append(bad, defaultLimitsUses(fset, f)...)
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	for _, b := range bad {
		t.Errorf("%s; pass the limits the resolver returns (layered.QuestionLimits) instead", b)
	}
	for _, a := range defaultLimitsAllowed {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(a))); err != nil {
			t.Errorf("the allowlist names %s, which is not in the tree: %v", a, err)
		}
	}
}

// TestDefaultLimitsUsesFindsEachShape is the walk's own control: each shape
// it must catch is caught, and the uses that read only the binary's fixed
// figures are not.
func TestDefaultLimitsUsesFindsEachShape(t *testing.T) {
	src := `package x

func f() {
	question.CheckLimits(fields, question.Default, who)
	_ = question.Decide(fields, state, question.Default)
	question.ChipRole(h, question.Default)
	_ = question.AskingRules(question.Default)
	_ = question.Default.Rows
	_ = question.Default.LabelWords
	_ = question.Default.MeaningSentences
	_ = question.Default.LaterLabels[0]
	_ = question.Default.Columns
	question.CheckLimits(fields, limits, who)
	_ = question.Decide(fields, state, state.Limits)
}
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "x.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	got := defaultLimitsUses(fset, f)
	if len(got) != 7 {
		t.Fatalf("want the seven uses of the default limits found, and the fixed figures left alone; got %d:\n%s", len(got), strings.Join(got, "\n"))
	}
	inPkg := `package question

func g() { _ = CheckLimits(f, Default, who); _ = Default.Rows }
`
	f, err = parser.ParseFile(fset, "q.go", inPkg, 0)
	if err != nil {
		t.Fatal(err)
	}
	if got := defaultLimitsUses(fset, f); len(got) != 2 {
		t.Fatalf("inside package question an unqualified Default must count; got %v", got)
	}
}
