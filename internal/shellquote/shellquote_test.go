package shellquote

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// TestSingleIsOneLiteralWord: whatever s holds, the shell reads Single(s) back
// as exactly one word equal to s. The proof is the shell's own reading, not a
// string comparison, so it holds the quoting to what a shell does with it.
func TestSingleIsOneLiteralWord(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("sh unavailable")
	}
	for _, s := range []string{
		"plain",
		"",
		"a b\tc",
		"it's",
		"two '' quotes",
		`a " double quote and a \ backslash`,
		"$HOME ${x} $(id) `date`",
		"history !word",
		"~/not/expanded",
		"a\nnewline",
	} {
		cmd := exec.Command("sh", "-c", "set -- "+Single(s)+`; printf '%s' "$#"; printf '\001%s' "$@"`)
		got, err := cmd.Output()
		if err != nil {
			t.Fatalf("sh -c with %s: %v", Single(s), err)
		}
		parts := strings.Split(string(got), "\001")
		if len(parts) != 2 || parts[0] != "1" || parts[1] != s {
			t.Errorf("Single(%q) = %s, which the shell reads as %q, want exactly one word %q", s, Single(s), parts, s)
		}
	}
	if got, want := Single("it's"), `'it'\''s'`; got != want {
		t.Errorf("Single(%q) = %s, want %s", "it's", got, want)
	}
	if got := Single(""); got != "''" {
		t.Errorf("Single(\"\") = %s, want ''", got)
	}
}

// quoteSpelling is the four bytes single quoting spells an embedded quote with.
const quoteSpelling = `'\''`

// quotingCopies reports each place f spells the single-quote escape as the
// REPLACEMENT of a strings call, which is the shape of a private copy of
// Single: strings.ReplaceAll(s, "'", `'\”`), strings.Replace(..., n), or a
// strings.NewReplacer pair. The inverse, a parser turning the four bytes back
// into a quote, passes the spelling as the text to find, and is not a copy.
func quotingCopies(fset *token.FileSet, f *ast.File) []int {
	var lines []int
	isSpelling := func(e ast.Expr) bool {
		lit, ok := e.(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			return false
		}
		v, err := strconv.Unquote(lit.Value)
		return err == nil && v == quoteSpelling
	}
	ast.Inspect(f, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if pkg, ok := sel.X.(*ast.Ident); !ok || pkg.Name != "strings" {
			return true
		}
		var replacements []ast.Expr
		switch sel.Sel.Name {
		case "ReplaceAll", "Replace":
			if len(call.Args) >= 3 {
				replacements = call.Args[2:3]
			}
		case "NewReplacer":
			for i := 1; i < len(call.Args); i += 2 {
				replacements = append(replacements, call.Args[i])
			}
		}
		for _, r := range replacements {
			if isSpelling(r) {
				lines = append(lines, fset.Position(r.Pos()).Line)
			}
		}
		return true
	})
	return lines
}

// TestOnlySingleQuotesForTheShell is the one-canonical-primitive check for
// shell quoting: outside this package, no non-test Go under internal/ or cmd/
// spells the single-quote escape as a replacement. A caller that needs a
// variant (a word left bare when it is plain, say) wraps Single; it does not
// restate it.
func TestOnlySingleQuotesForTheShell(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	thisPkg := filepath.Join("internal", "shellquote")
	var found []string
	walked := 0
	for _, tree := range []string{"internal", "cmd"} {
		err := filepath.WalkDir(filepath.Join(root, tree), func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if d.Name() == "testdata" {
					return fs.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(d.Name(), ".go") || strings.HasSuffix(d.Name(), "_test.go") {
				return nil
			}
			rel, err := filepath.Rel(root, p)
			if err != nil {
				return err
			}
			if filepath.Dir(rel) == thisPkg {
				return nil
			}
			src, err := os.ReadFile(p)
			if err != nil {
				return err
			}
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, rel, src, 0)
			if err != nil {
				return fmt.Errorf("parse %s: %w", rel, err)
			}
			walked++
			for _, line := range quotingCopies(fset, f) {
				found = append(found, fmt.Sprintf("%s:%d", filepath.ToSlash(rel), line))
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", tree, err)
		}
	}
	// A walk that read nothing would pass while holding nothing.
	if walked < 100 {
		t.Fatalf("the walk read only %d non-test Go files under internal/ and cmd/; it is not looking at the repository", walked)
	}
	sort.Strings(found)
	for _, l := range found {
		t.Errorf("%s spells the shell single-quote escape itself; call shellquote.Single, the one place it is written", l)
	}
}

// TestQuotingCopyScannerIsArmed proves the scanner reports each shape of a
// copy and leaves the inverse and a comment alone, so a pass above is the
// property holding rather than the scanner missing what it walked.
func TestQuotingCopyScannerIsArmed(t *testing.T) {
	hostile := map[string]string{
		"a raw-string ReplaceAll":          "func f(s string) string { return \"'\" + strings.ReplaceAll(s, \"'\", `'\\''`) + \"'\" }",
		"an interpreted-string ReplaceAll": `func f(s string) string { return strings.ReplaceAll(s, "'", "'\\''") }`,
		"a Replace with a count":           "func f(s string) string { return strings.Replace(s, \"'\", `'\\''`, -1) }",
		"a NewReplacer pair":               "var r = strings.NewReplacer(\"\\n\", \" \", \"'\", `'\\''`)",
	}
	for label, body := range hostile {
		if got := scan(t, body); len(got) != 1 {
			t.Errorf("%s: the scanner reports %v, want exactly one copy\n%s", label, got, body)
		}
	}
	benign := map[string]string{
		"the inverse, a parser": "func f(s string) string { return strings.ReplaceAll(s, `'\\''`, \"'\") }",
		"a prefix test":         "func f(s string) bool { return strings.HasPrefix(s, `'\\''`) }",
		"a comment":             "// a quote is spelled '\\'' inside single quotes\nvar x = 1",
		"another replacement":   `func f(s string) string { return strings.ReplaceAll(s, "'", "\\'") }`,
		"a call through Single": `func f(s string) string { return shellquote.Single(s) }`,
		"a NewReplacer inverse": "var r = strings.NewReplacer(`'\\''`, \"'\")",
	}
	for label, body := range benign {
		if got := scan(t, body); len(got) != 0 {
			t.Errorf("%s: the scanner reports %v from a source that holds no copy\n%s", label, got, body)
		}
	}
}

func scan(t *testing.T, body string) []int {
	t.Helper()
	src := "package p\n\n" + body + "\n"
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "hostile.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse: %v\n%s", err, src)
	}
	return quotingCopies(fset, f)
}
