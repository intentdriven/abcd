package refusal

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// TestNoTwoEntriesShareACode is the uniqueness test adr-2610301638558260
// names: two branches that each add a code collide only when they pick the
// same name, and this test, run in the merge queue like every test, is what
// catches that case before either branch lands.
func TestNoTwoEntriesShareACode(t *testing.T) {
	seen := map[Code]bool{}
	for _, e := range Entries() {
		if seen[e.Code] {
			t.Errorf("two registry entries share the code %q", e.Code)
		}
		seen[e.Code] = true
	}
}

// TestEveryCodeFollowsTheGrammar: a code is the verb, a slash and the reason,
// lower-case kebab-case (adr-2610301638558260).
func TestEveryCodeFollowsTheGrammar(t *testing.T) {
	for _, e := range Entries() {
		if !ValidName(string(e.Code)) {
			t.Errorf("code %q is not <verb>/<reason> in lower-case kebab-case", e.Code)
		}
	}
	for _, bad := range []string{"", "capture", "Capture/remedy-missing", "capture/remedy_missing",
		"capture/-remedy", "capture//remedy", "capture/remedy/missing", "capture/remedy-", "/remedy"} {
		if ValidName(bad) {
			t.Errorf("ValidName(%q) = true, want false", bad)
		}
	}
	for _, good := range []string{"capture/remedy-missing", "usage/unknown-flag", "explain/unknown-code"} {
		if !ValidName(good) {
			t.Errorf("ValidName(%q) = false, want true", good)
		}
	}
}

// TestEveryEntryIsComplete: each entry has its one-line summary, its
// explanation and a next command or the explicit word none, and a retired
// entry names a release tag.
func TestEveryEntryIsComplete(t *testing.T) {
	for _, e := range Entries() {
		if strings.TrimSpace(e.Summary) == "" || strings.Contains(e.Summary, "\n") {
			t.Errorf("%s: the summary must be one non-empty line, got %q", e.Code, e.Summary)
		}
		if strings.TrimSpace(e.Explanation) == "" {
			t.Errorf("%s: the explanation is empty", e.Code)
		}
		if e.Next != NoNext && !strings.HasPrefix(e.Next, "abcd ") {
			t.Errorf("%s: the next command must be an abcd command or %q, got %q", e.Code, NoNext, e.Next)
		}
		if e.Retired != "" && !strings.HasPrefix(e.Retired, "v") {
			t.Errorf("%s: retired must name a release tag, got %q", e.Code, e.Retired)
		}
		usage := strings.HasPrefix(string(e.Code), "usage/")
		if usage != (len(e.RaisedBy) == 0) {
			t.Errorf("%s: only a usage code is raised by every command and names no raiser; raised_by = %q", e.Code, e.RaisedBy)
		}
		for _, p := range e.RaisedBy {
			if !strings.HasPrefix(p, "abcd") {
				t.Errorf("%s: raised_by %q is not a command path", e.Code, p)
			}
		}
	}
}

// TestUncodedIsNeverRegistered: internal/uncoded is what a refusal built
// without a code carries, so that the gates fail on it; registering it would
// make it resolve and pass.
func TestUncodedIsNeverRegistered(t *testing.T) {
	if _, ok := Lookup(Uncoded); ok {
		t.Fatalf("%s is registered; it must never resolve", Uncoded)
	}
	if _, ok := Lookup(None); ok {
		t.Fatal("the empty code is registered")
	}
}

// TestWrapIsFoundThroughErrorsAs: a core package names the code where the
// refusal is decided, and the front door finds it however the error was
// wrapped on the way out.
func TestWrapIsFoundThroughErrorsAs(t *testing.T) {
	base := errors.New("the remedy is missing")
	wrapped := Wrap(CaptureRemedyMissing, base)
	if wrapped.Error() != base.Error() {
		t.Errorf("Wrap changed the message: %q", wrapped.Error())
	}
	if !errors.Is(wrapped, base) {
		t.Error("Wrap does not unwrap to the error it wraps")
	}
	outer := fmt.Errorf("abcd capture: %w", wrapped)
	if got := CodeOf(outer); got != CaptureRemedyMissing {
		t.Errorf("CodeOf through fmt.Errorf = %q, want %q", got, CaptureRemedyMissing)
	}
	joined := errors.Join(errors.New("other"), outer)
	if got := CodeOf(joined); got != CaptureRemedyMissing {
		t.Errorf("CodeOf through errors.Join = %q, want %q", got, CaptureRemedyMissing)
	}
	if got := CodeOf(base); got != None {
		t.Errorf("CodeOf of an uncoded error = %q, want none", got)
	}
	if Wrap(CaptureRemedyMissing, nil) != nil {
		t.Error("Wrap of a nil error must be nil")
	}
}

// TestFillNextUsesTheCommandPathOnly: the next command is filled from the
// refused call's command path, and from nothing else.
func TestFillNextUsesTheCommandPathOnly(t *testing.T) {
	cases := []struct{ next, path, want string }{
		{"abcd explain {command}", "abcd capture remedy", "abcd explain capture remedy"},
		{"abcd explain {command}", "abcd", "abcd explain abcd"},
		{`abcd capture "<text>" --remedy "<fix>"`, "abcd capture", `abcd capture "<text>" --remedy "<fix>"`},
		{NoNext, "abcd capture", NoNext},
	}
	for _, c := range cases {
		if got := FillNext(c.next, c.path); got != c.want {
			t.Errorf("FillNext(%q, %q) = %q, want %q", c.next, c.path, got, c.want)
		}
	}
}

// TestLedgerRoundTripsAndIsSorted: the ledger lists every registered code,
// retired ones included, sorted, and decodes to what was encoded.
func TestLedgerRoundTripsAndIsSorted(t *testing.T) {
	l := BuildLedger()
	if len(l.Codes) != len(Entries()) {
		t.Fatalf("the ledger lists %d codes, the registry %d", len(l.Codes), len(Entries()))
	}
	for i := 1; i < len(l.Codes); i++ {
		if l.Codes[i-1].Code >= l.Codes[i].Code {
			t.Fatalf("the ledger is not sorted at %q, %q", l.Codes[i-1].Code, l.Codes[i].Code)
		}
	}
	data, err := EncodeLedger(l)
	if err != nil {
		t.Fatal(err)
	}
	back, err := DecodeLedger(data)
	if err != nil {
		t.Fatal(err)
	}
	again, err := EncodeLedger(back)
	if err != nil {
		t.Fatal(err)
	}
	if string(again) != string(data) {
		t.Fatalf("the ledger does not round-trip:\n%s\n---\n%s", data, again)
	}
	if _, err := DecodeLedger([]byte(`{"format_version": 99, "codes": []}`)); err == nil {
		t.Error("a ledger of an unknown format version decoded")
	}
	if _, err := DecodeLedger([]byte(`{"format_version": 1, "codes": [{"code": "Not A Code"}]}`)); err == nil {
		t.Error("a ledger holding a malformed code decoded")
	}
}

// TestCommittedLedgerMatchesTheRegistry is the drift test: the ledger the
// release guard reads at a tag is written from the registry by
// cmd/refusal-codes-sync, and a registry change that is not written into it
// fails here.
func TestCommittedLedgerMatchesTheRegistry(t *testing.T) {
	path := filepath.Join("..", "..", "..", filepath.FromSlash(LedgerPath))
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read the committed ledger: %v (run `make refusal-codes-sync`)", err)
	}
	want, err := EncodeLedger(BuildLedger())
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Fatalf("%s is out of date with the registry; run `make refusal-codes-sync`.\ncommitted:\n%s\nwant:\n%s", LedgerPath, got, want)
	}
}

// TestEveryWrapInCoreNamesARegisteredConstant: a core package names a
// refusal's code with refusal.Wrap where the refusal is decided, and the code
// it names is one of this package's constants, registered here. A string, a
// conversion or a variable could name a code the registry never heard of.
func TestEveryWrapInCoreNamesARegisteredConstant(t *testing.T) {
	consts := constantCodes(t)
	coreDir := filepath.Join("..")
	err := filepath.WalkDir(coreDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == "testdata" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		if filepath.Base(filepath.Dir(path)) == "refusal" {
			return nil
		}
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return err
		}
		alias := importName(f, "github.com/intentdriven/abcd/internal/core/refusal")
		if alias == "" {
			return nil
		}
		ast.Inspect(f, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok || !isSelector(call.Fun, alias, "Wrap") || len(call.Args) == 0 {
				return true
			}
			sel, ok := call.Args[0].(*ast.SelectorExpr)
			if !ok || !isIdent(sel.X, alias) {
				t.Errorf("%s: refusal.Wrap must name a code constant of the refusal package", fset.Position(call.Pos()))
				return true
			}
			code, known := consts[sel.Sel.Name]
			if !known {
				t.Errorf("%s: refusal.%s is not a code constant", fset.Position(call.Pos()), sel.Sel.Name)
				return true
			}
			if _, ok := Lookup(code); !ok {
				t.Errorf("%s: refusal.%s (%s) is not registered", fset.Position(call.Pos()), sel.Sel.Name, code)
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// TestEveryConstantIsRegistered: a code constant the registry does not hold
// would be raised and resolve to nothing; internal/uncoded is the one
// exception, by design.
func TestEveryConstantIsRegistered(t *testing.T) {
	for name, code := range constantCodes(t) {
		if code == None || code == Uncoded {
			continue
		}
		if _, ok := Lookup(code); !ok {
			t.Errorf("constant %s (%s) is not registered", name, code)
		}
	}
}

// constantCodes reads this package's source for every constant declared as a
// Code, by name.
func constantCodes(t *testing.T) map[string]Code {
	t.Helper()
	fset := token.NewFileSet()
	out := map[string]Code{}
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, e.Name(), nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, decl := range f.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.CONST {
				continue
			}
			for _, spec := range gen.Specs {
				vs := spec.(*ast.ValueSpec)
				if !isIdent(vs.Type, "Code") {
					continue
				}
				for i, name := range vs.Names {
					lit, ok := vs.Values[i].(*ast.BasicLit)
					if !ok {
						t.Fatalf("constant %s is not a string literal", name.Name)
					}
					s, err := strconv.Unquote(lit.Value)
					if err != nil {
						t.Fatal(err)
					}
					out[name.Name] = Code(s)
				}
			}
		}
	}
	if len(out) == 0 {
		t.Fatal("no Code constants found")
	}
	return out
}

func importName(f *ast.File, path string) string {
	for _, imp := range f.Imports {
		if p, _ := strconv.Unquote(imp.Path.Value); p == path {
			if imp.Name != nil {
				return imp.Name.Name
			}
			return "refusal"
		}
	}
	return ""
}

func isSelector(e ast.Expr, x, sel string) bool {
	s, ok := e.(*ast.SelectorExpr)
	return ok && isIdent(s.X, x) && s.Sel.Name == sel
}

func isIdent(e ast.Expr, name string) bool {
	id, ok := e.(*ast.Ident)
	return ok && id.Name == name
}
