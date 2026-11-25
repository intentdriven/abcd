package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/intentdriven/abcd/internal/core/refusal"
	"github.com/spf13/cobra"
)

// refusal_codes_test.go — the gates of spc-2610301431108175 steps 1 and 2:
// every refusal raised in this package names a registered code, every code
// resolves through `abcd explain`, every next command names a command the
// tree has, and every usage refusal on every command carries a usage code.

// refusalUncodedCeiling is the most files the exemption list may name. It is
// the list's length when the list was made, and it only falls: a step that
// codes a file's last refusal removes the file from the list and lowers this
// number with it, so a file cannot be added without one being taken off.
const refusalUncodedCeiling = 66

// refusalUncodedFiles reads the exemption list: the files, from the package
// directory, whose literals carry no code yet.
func refusalUncodedFiles(t *testing.T) []string {
	t.Helper()
	f, err := os.Open(filepath.Join("..", "..", "..", filepath.FromSlash(refusalUncodedPath)))
	if err != nil {
		t.Fatalf("read the exemption list: %v", err)
	}
	defer f.Close()
	var out []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		rel, ok := strings.CutPrefix(line, "internal/surface/cli/")
		if !ok || strings.Contains(rel, "/") {
			t.Fatalf("the exemption list names %q, which is not a file of internal/surface/cli", line)
		}
		out = append(out, rel)
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

// refusalConstants reads the refusal package's source for every Code
// constant, by name.
func refusalConstants(t *testing.T) map[string]refusal.Code {
	t.Helper()
	dir := filepath.Join("..", "..", "core", "refusal")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]refusal.Code{}
	fset := token.NewFileSet()
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, filepath.Join(dir, e.Name()), nil, 0)
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
				if id, ok := vs.Type.(*ast.Ident); !ok || id.Name != "Code" {
					continue
				}
				for i, name := range vs.Names {
					s, err := strconv.Unquote(vs.Values[i].(*ast.BasicLit).Value)
					if err != nil {
						t.Fatal(err)
					}
					out[name.Name] = refusal.Code(s)
				}
			}
		}
	}
	return out
}

// TestExitErrorIsBuiltOnlyThroughItsConstructors is the literal test: outside
// refused and exitWith, no file of this package builds an exitError literal,
// except the files the exemption list names while the migration runs. Each
// refused call names a registered code constant at exit 2 or 3. The list
// names no file that no longer needs it, and no more files than the ceiling.
func TestExitErrorIsBuiltOnlyThroughItsConstructors(t *testing.T) {
	exempt := refusalUncodedFiles(t)
	if len(exempt) > refusalUncodedCeiling {
		t.Errorf("the exemption list names %d files, more than its ceiling of %d: it only shrinks; "+
			"code the new file's refusals with refused or exitWith instead", len(exempt), refusalUncodedCeiling)
	}
	if !slices.IsSorted(exempt) {
		t.Errorf("the exemption list is not sorted")
	}
	consts := refusalConstants(t)
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	withLiterals := map[string]bool{}
	fset := token.NewFileSet()
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, name, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, decl := range f.Decls {
			fn, isFunc := decl.(*ast.FuncDecl)
			constructor := isFunc && fn.Recv == nil && (fn.Name.Name == "refused" || fn.Name.Name == "exitWith")
			ast.Inspect(decl, func(n ast.Node) bool {
				switch n := n.(type) {
				case *ast.CompositeLit:
					if id, ok := n.Type.(*ast.Ident); ok && id.Name == "exitError" && !constructor {
						withLiterals[name] = true
						if !slices.Contains(exempt, name) {
							t.Errorf("%s: an exitError literal outside refused and exitWith; build the refusal with refused(exit, refusal.<Code>, msg)",
								fset.Position(n.Pos()))
						}
					}
				case *ast.CallExpr:
					id, ok := n.Fun.(*ast.Ident)
					if !ok || id.Name != "refused" || len(n.Args) != 3 {
						return true
					}
					if lit, ok := n.Args[0].(*ast.BasicLit); !ok || (lit.Value != "2" && lit.Value != "3") {
						t.Errorf("%s: refused takes exit 2 or 3 as a literal", fset.Position(n.Pos()))
					}
					sel, ok := n.Args[1].(*ast.SelectorExpr)
					if !ok {
						t.Errorf("%s: refused must name a code constant of the refusal package", fset.Position(n.Pos()))
						return true
					}
					code, known := consts[sel.Sel.Name]
					if !known {
						t.Errorf("%s: refusal.%s is not a code constant", fset.Position(n.Pos()), sel.Sel.Name)
					} else if _, ok := refusal.Lookup(code); !ok {
						t.Errorf("%s: refusal.%s (%s) is not registered", fset.Position(n.Pos()), sel.Sel.Name, code)
					}
				}
				return true
			})
		}
	}
	for _, name := range exempt {
		if !withLiterals[name] {
			t.Errorf("the exemption list names %s, which builds no exitError literal any more: remove it from %s and lower refusalUncodedCeiling",
				name, refusalUncodedPath)
		}
	}
}

// TestExitWithCarriesUncodedForAnUncodedRefusal: a refusal built with no code
// at exit 2 or 3 carries internal/uncoded, which no test lets pass; a fault
// carries none.
func TestExitWithCarriesUncodedForAnUncodedRefusal(t *testing.T) {
	for _, exit := range []int{2, 3} {
		if got := exitWith(exit, refusal.None, "x").ref; got != refusal.Uncoded {
			t.Errorf("exitWith(%d, None) carries %q, want %q", exit, got, refusal.Uncoded)
		}
	}
	if got := exitWith(1, refusal.None, "x").ref; got != refusal.None {
		t.Errorf("exitWith(1, None) carries %q, want none", got)
	}
	if got := exitWith(2, refusal.CaptureMissingRemedy, "x").ref; got != refusal.CaptureMissingRemedy {
		t.Errorf("exitWith kept %q, want the code given", got)
	}
}

// TestEveryRegisteredCodeResolvesThroughExplain: `abcd explain <code>`
// answers every registered code, its text with the summary, the explanation
// and the raisers, and its --json with the entry itself.
func TestEveryRegisteredCodeResolvesThroughExplain(t *testing.T) {
	for _, e := range refusal.Entries() {
		out, err := runCLIErr(t, "explain", string(e.Code))
		if err != nil {
			t.Errorf("explain %s: %v", e.Code, err)
			continue
		}
		text := string(out)
		for _, want := range []string{string(e.Code) + " — " + e.Summary, strings.Split(e.Explanation, "\n")[0], "Try next: ", "Raised by: "} {
			if !strings.Contains(text, want) {
				t.Errorf("explain %s does not say %q:\n%s", e.Code, want, text)
			}
		}
		raw, err := runCLIErr(t, "explain", string(e.Code), "--json")
		if err != nil {
			t.Errorf("explain %s --json: %v", e.Code, err)
			continue
		}
		var got refusal.Entry
		if err := json.Unmarshal(raw, &got); err != nil {
			t.Errorf("explain %s --json is not an entry: %v\n%s", e.Code, err, raw)
			continue
		}
		if got.Code != e.Code || got.Summary != e.Summary || got.Explanation != e.Explanation || got.Next != e.Next ||
			!slices.Equal(got.RaisedBy, e.RaisedBy) || got.Retired != e.Retired {
			t.Errorf("explain %s --json = %+v, want %+v", e.Code, got, e)
		}
	}
}

// TestExplainRefusesAnUnknownCodeNamingTheNearest: a word holding a slash is
// a code, an unknown one is refused under explain/unknown-code, naming the
// nearest registered code and never repeating the word.
func TestExplainRefusesAnUnknownCodeNamingTheNearest(t *testing.T) {
	var stdout, stderr strings.Builder
	code := Run([]string{"explain", "capture/missing-remdy", "--json"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("exit %d, want 2:\n%s%s", code, stdout.String(), stderr.String())
	}
	var env errorEnvelope
	if err := json.Unmarshal([]byte(stdout.String()), &env); err != nil {
		t.Fatalf("not one envelope: %v\n%s", err, stdout.String())
	}
	if env.Code != string(refusal.ExplainUnknownCode) || !strings.Contains(env.Error, "`capture/missing-remedy`") {
		t.Errorf("envelope = %+v, want explain/unknown-code naming capture/missing-remedy", env)
	}
	if strings.Contains(stdout.String(), "remdy") {
		t.Errorf("the refusal repeats the word:\n%s", stdout.String())
	}
	// A code with --input reads no input.
	if code := Run([]string{"explain", "capture/missing-remedy", "--input", "x"}, &stdout, &stderr); code != 2 {
		t.Errorf("explain <code> --input exited %d, want 2", code)
	}
}

// nextCommandWords is the command path a next template names: the words
// after `abcd` up to the first placeholder, quoted value or flag.
func nextCommandWords(next string) []string {
	words := strings.Fields(strings.TrimPrefix(next, "abcd "))
	for i, w := range words {
		if strings.ContainsAny(w[:1], `"<-{'[`) {
			return words[:i]
		}
	}
	return words
}

// TestEveryNextCommandAndRaiserIsACommandTheTreeHas: each next template names
// a command path the tree has, and so does each command named as raising a
// code. A slot is filled with a path the tree has, so it needs no check.
func TestEveryNextCommandAndRaiserIsACommandTheTreeHas(t *testing.T) {
	paths := map[string]bool{}
	for _, f := range CommandFacts() {
		paths[f.Path] = true
	}
	for _, e := range refusal.Entries() {
		if e.Next != refusal.NoNext {
			words := nextCommandWords(e.Next)
			switch {
			case len(words) == 0:
				t.Errorf("%s: the next command %q names no command", e.Code, e.Next)
			case words[0] == "explain" && len(words) > 1:
				// explain's own positionals are a command path, which it
				// must resolve.
				if _, err := explainResolve(CommandFacts(), words[1:]); err != nil {
					t.Errorf("%s: the next command %q explains a path the tree does not have: %v", e.Code, e.Next, err)
				}
			default:
				if path := "abcd " + strings.Join(words, " "); !paths[path] {
					t.Errorf("%s: the next command %q names %q, which the tree does not have", e.Code, e.Next, path)
				}
			}
		}
		for _, p := range e.RaisedBy {
			if !paths[p] {
				t.Errorf("%s: raised by %q, which the tree does not have", e.Code, p)
			}
		}
	}
}

// TestUsageWordingCodes holds the classifier to the command framework's own
// wordings, one per usage code, as cobra and pflag word them and as the
// requirement aggregate and the hook plane extend them.
func TestUsageWordingCodes(t *testing.T) {
	cases := []struct {
		msg  string
		want refusal.Code
	}{
		{`unknown command "zz" for "abcd capture"`, refusal.UsageUnknownCommand},
		{`unknown command "zz" for "abcd hook"` + hookPlaneSkewNote, refusal.UsageUnknownCommand},
		{"requires at least 1 arg(s), only received 0", refusal.UsageMissingArgument},
		{"accepts 2 arg(s), received 1", refusal.UsageMissingArgument},
		{"accepts 1 arg(s), received 3", refusal.UsageExtraArgument},
		{"accepts at most 1 arg(s), received 2", refusal.UsageExtraArgument},
		{"accepts between 1 and 2 arg(s), received 0", refusal.UsageMissingArgument},
		{"accepts between 1 and 2 arg(s), received 4", refusal.UsageExtraArgument},
		{"accepts 2 arg(s), received 1; --impact is not set — usage: abcd capture resolve <iss-N> <note> --impact <i> (nothing written)", refusal.UsageMissingArgument},
		{"--impact is not set; --grounds is not set — usage: abcd x --impact <i> --grounds <g> (nothing written)", refusal.UsageMissingRequiredFlag},
		{"if any flags in the group [check yes] are set none of the others can be; [check yes] were all set", refusal.UsageFlagConflict},
		{"if any flags in the group [a b] are set they must all be set; missing [b]", refusal.UsageMissingRequiredFlag},
		{"at least one of the flags in the group [a b] is required", refusal.UsageMissingRequiredFlag},
		{`required flag(s) "x" not set`, refusal.UsageMissingRequiredFlag},
		{"unknown flag: --zz", refusal.UsageUnknownFlag},
		{"unknown shorthand flag: 'z' in -z", refusal.UsageUnknownFlag},
		{"bad flag syntax: ---x", refusal.UsageUnknownFlag},
		{"flag needs an argument: --remedy", refusal.UsageInvalidFlagValue},
		{`invalid argument for "--limit" flag: it takes a value of type int`, refusal.UsageInvalidFlagValue},
		{"abcd capture: the issue text is a thing", refusal.None},
	}
	for _, c := range cases {
		if got := usageWordingCode(c.msg); got != c.want {
			t.Errorf("usageWordingCode(%q) = %q, want %q", c.msg, got, c.want)
		}
	}
}

// codedRefusal runs args through the front door with --json and returns
// the exit and the last document on stdout, the refusal.
func codedRefusal(t *testing.T, args ...string) (int, errorEnvelope, string) {
	t.Helper()
	var stdout, stderr strings.Builder
	code := Run(append(slices.Clone(args), "--json"), &stdout, &stderr)
	dec := json.NewDecoder(strings.NewReader(stdout.String()))
	var env errorEnvelope
	for dec.More() {
		var doc errorEnvelope
		if err := dec.Decode(&doc); err != nil {
			break
		}
		env = doc
	}
	return code, env, stdout.String() + stderr.String()
}

// everyCommandPath is every command of a fresh tree below the root, by path,
// hidden ones included and cobra's run-time verbs left out.
func everyCommandPath(root *cobra.Command) [][]string {
	var out [][]string
	var walk func(*cobra.Command, []string)
	walk = func(c *cobra.Command, prefix []string) {
		for _, sub := range c.Commands() {
			if sub.Name() == "help" || sub.Name() == "completion" {
				continue
			}
			path := append(slices.Clone(prefix), sub.Name())
			out = append(out, path)
			walk(sub, path)
		}
	}
	walk(root, nil)
	return out
}

// TestEveryCommandCodesAnUnknownFlag is step 2's tree-wide half for flags:
// every command, hidden ones included, refuses an unknown flag carrying
// usage/unknown-flag, the hook plane at exit 1 and every other at 2, with the
// next command naming `abcd explain` on the command refused. Runs in a scratch
// directory, as the parent sweep does.
func TestEveryCommandCodesAnUnknownFlag(t *testing.T) {
	t.Chdir(t.TempDir())
	paths := append([][]string{nil}, everyCommandPath(NewRootCommand())...)
	for _, path := range paths {
		name := strings.Join(path, " ")
		t.Run(strings.ReplaceAll("abcd "+name, " ", "_"), func(t *testing.T) {
			code, env, out := codedRefusal(t, append(slices.Clone(path), "--zzz-not-a-flag")...)
			if code != 1 && code != 2 {
				t.Fatalf("exit %d, want a usage refusal:\n%s", code, out)
			}
			if env.Code != string(refusal.UsageUnknownFlag) {
				t.Fatalf("code = %q, want %s:\n%s", env.Code, refusal.UsageUnknownFlag, out)
			}
			want := "abcd explain " + name
			if name == "" {
				want = "abcd explain abcd"
			}
			if env.Next != want {
				t.Errorf("next = %q, want %q", env.Next, want)
			}
		})
	}
}

// TestEveryParentCodesAnUnknownSubverb is step 2's tree-wide half for
// sub-verbs: every parent that refuses an unknown sub-verb with the command
// framework's own refusal carries usage/unknown-command, the hook plane's at
// exit 1. A parent whose own validator or run decides first (a verb that
// takes a positional of its own, or refuses outside a checkout) words its
// refusal itself, and its code is its verb's step's.
func TestEveryParentCodesAnUnknownSubverb(t *testing.T) {
	t.Chdir(t.TempDir())
	framework := 0
	for _, p := range parents(t, NewRootCommand()) {
		name := strings.Join(p.path, " ")
		code, env, out := codedRefusal(t, append(slices.Clone(p.path), "zzz-not-a-subverb")...)
		if code != 1 && code != 2 {
			t.Errorf("%s: exit %d, want a usage refusal:\n%s", name, code, out)
			continue
		}
		if !strings.Contains(env.Error, `unknown command for "abcd `) {
			continue
		}
		framework++
		if env.Code != string(refusal.UsageUnknownCommand) {
			t.Errorf("%s: code = %q, want %s:\n%s", name, env.Code, refusal.UsageUnknownCommand, out)
		}
	}
	if framework < 20 {
		t.Errorf("only %d parents refused with the framework's own wording; the sweep is not reaching them", framework)
	}
}

// TestUsageRefusalsCarryTheirCodes runs one refusal of each kind the
// framework raises through the front door.
func TestUsageRefusalsCarryTheirCodes(t *testing.T) {
	_ = captureLedgerRepo(t)
	cases := []struct {
		args []string
		exit int
		want string // the code as a reader matches it, so a path that switches code fails
	}{
		{[]string{"zzz-not-a-verb"}, 2, "usage/unknown-command"},
		{[]string{"update", "--check", "--yes"}, 2, "usage/flag-conflict"},
		{[]string{"capture", "list", "--open", "--all"}, 2, "usage/flag-conflict"},
		{[]string{"capture", "list"}, 2, "usage/missing-required-flag"},
		{[]string{"capture", "resolve"}, 2, "usage/missing-argument"},
		{[]string{"drain", "--max", "not-a-number"}, 2, "usage/invalid-flag-value"},
		{[]string{"capture", "a thing that happened", "--severity", "enormous", "--remedy", "fix it"}, 2, "usage/invalid-flag-value"},
		{[]string{"guard", "hook", "--zzz-not-a-flag"}, 1, "usage/unknown-flag"},
		{[]string{"hook", "zzz-not-a-subverb"}, 1, "usage/unknown-command"},
	}
	for _, c := range cases {
		code, env, out := codedRefusal(t, c.args...)
		if code != c.exit || env.Code != c.want {
			t.Errorf("abcd %s: exit %d code %q, want exit %d code %q:\n%s", strings.Join(c.args, " "), code, env.Code, c.exit, c.want, out)
		}
	}
}

// refusalCodeViolations collects, for TestMain, every refusal the package's
// tests saw Run render with a code no reader could resolve.
var refusalCodeViolations struct {
	sync.Mutex
	lines []string
}

// checkRefusalCodeForTest is refusalCodeCheck under the package's tests: a
// refusal leaving with internal/uncoded, or with a code the registry does not
// hold, is recorded, and so is a code on a fault other than a usage refusal
// the hook plane moved to exit 1. While the exemption list exists a refusal
// with no code at all is the migration's, and passes; once it is gone, an
// exit 2 or 3 with no code is recorded too.
func checkRefusalCodeForTest(strict bool, record func(string)) func(int, refusal.Code, []string) {
	return func(exit int, ref refusal.Code, args []string) {
		var why string
		_, registered := refusal.Lookup(ref)
		switch {
		case ref == refusal.Uncoded:
			why = "carries internal/uncoded: build it with refused or exitWith and a registered code"
		case ref != refusal.None && !registered:
			why = "carries " + string(ref) + ", which is not registered"
		case exit == 1 && ref != refusal.None && !strings.HasPrefix(string(ref), "usage/"):
			why = "is a fault (exit 1) carrying " + string(ref) + "; a fault carries no code"
		case strict && (exit == 2 || exit == 3) && ref == refusal.None:
			why = "left at exit " + strconv.Itoa(exit) + " with no code"
		}
		if why != "" {
			record(fmt.Sprintf("abcd %s (exit %d) %s", strings.Join(args, " "), exit, why))
		}
	}
}

// installRefusalCodeCheck sets the check for the whole test binary; the
// returned function reports what it recorded and says whether the binary
// must fail.
func installRefusalCodeCheck() (report func() bool) {
	_, err := os.Stat(filepath.Join("..", "..", "..", filepath.FromSlash(refusalUncodedPath)))
	refusalCodeCheck = checkRefusalCodeForTest(os.IsNotExist(err), func(line string) {
		refusalCodeViolations.Lock()
		defer refusalCodeViolations.Unlock()
		refusalCodeViolations.lines = append(refusalCodeViolations.lines, line)
	})
	return func() bool {
		refusalCodeViolations.Lock()
		defer refusalCodeViolations.Unlock()
		for _, line := range refusalCodeViolations.lines {
			fmt.Fprintln(os.Stderr, "refusal code check:", line)
		}
		return len(refusalCodeViolations.lines) > 0
	}
}

// TestTheRefusalCodeCheckRecordsAnUncodedRefusal: the check the test binary
// runs over every refusal records internal/uncoded and an unregistered code,
// and in strict mode a refusal with no code at all.
func TestTheRefusalCodeCheckRecordsAnUncodedRefusal(t *testing.T) {
	var got []string
	record := func(line string) { got = append(got, line) }
	lenient, strict := checkRefusalCodeForTest(false, record), checkRefusalCodeForTest(true, record)
	lenient(2, refusal.CaptureMissingRemedy, nil)
	lenient(2, refusal.None, nil)
	lenient(1, refusal.UsageUnknownFlag, nil)
	lenient(1, refusal.None, nil)
	if len(got) != 0 {
		t.Fatalf("the check recorded a refusal that carries a registered code, or a legacy one while the list exists: %q", got)
	}
	lenient(2, refusal.Uncoded, nil)
	lenient(2, "capture/never-registered", nil)
	lenient(1, refusal.CaptureMissingRemedy, nil)
	strict(3, refusal.None, nil)
	if len(got) != 4 {
		t.Fatalf("the check recorded %d violations, want 4: %q", len(got), got)
	}
}
