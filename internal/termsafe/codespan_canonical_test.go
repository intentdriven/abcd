package termsafe

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// backtickScanRe matches a backtick sought in Go source, in the two forms a
// hand-rolled code-span pairer spells it: the rune literal a byte walk compares
// against, and a backtick string handed to a strings or bytes search.
var backtickScanRe = regexp.MustCompile("'`'|\\b(?:strings|bytes)\\.(?:Index|LastIndex|Count|Cut|Split)[A-Za-z]*\\([^)\\n]*\"`\"")

// backtickScanner is one allowlisted file: how many backtick scans it holds and
// why none of them is a second code-span pairer.
type backtickScanner struct {
	count  int
	reason string
}

// backtickScanners names every non-test Go file outside this package that
// seeks a backtick, with the number of scans it holds and the reason it is not
// a second pairer. The count is pinned, so a scan added to a file already on
// the list is a new claim too: a pairer written into an allowlisted file
// changes its count and fails until a reviewer reads the new reason. The
// default for a file this test names is to pair through PairCodeSpan. A writer
// that sizes a span or a fence one longer than the text it wraps counts the
// text's longest run through LongestFenceRun, so it seeks no backtick itself
// and has no entry here (iss-2610140739284100). It names the characters it
// wants through FenceBacktick and FenceTilde, and fenceConstantMisuses fails
// the test on any other use of those constants outside this package, so they
// are no way round the count.
var backtickScanners = map[string]backtickScanner{
	"internal/adapter/scanner/identity.go": {2, "two delimiter sets: a backtick is one of the characters that may end an identity token, and one of those that bounds the path token an owner slug is judged in; nothing is paired"},
	"internal/core/assistedby/body.go":     {2, "stripFencedBlocks is the Go copy of scripts/check-attribution.sh's strip_fenced_blocks, held to that gate by TestJudgeBodyMatchesTheGate: a backtick there is a fence opener's character, and an opener whose info string carries one is no fence, as the gate reads it; nothing is paired"},
	"internal/core/guard/tokenize.go":      {25, "the shell tokenizer: a backtick there is command substitution, a shell grammar, not markdown"},
	"internal/core/guard/payload.go":       {3, "targetsAnExpansion reads a shell line's raw text for an assignment target that holds an expansion: a backtick there opens or closes a command substitution, a shell grammar, not markdown; nothing is paired"},
	"internal/core/guard/varpath.go":       {1, "emptiedInside reads whether a default's text opens with a backquote, a command substitution the shell string's own shell runs and which can print nothing: a shell grammar, not markdown; nothing is paired"},
	"internal/core/guard/unknown.go":       {2, "spellWord spells a default's or an alternative's shell word, and readPattern reads a trim's or a replacement's pattern: a backtick in either opens a command substitution, whose output the spelling drops or the pattern reads as unknown text; nothing is paired"},
	"internal/core/lifeboat/mdrender.go":   {1, "escapeLeadingMarker asks whether a value opens with a backtick, then asks OpensBalancedCodeSpan, which pairs through PairCodeSpan"},
	"internal/core/mdrecord/inline.go":     {2, "walkLine, Read's walk of one line, hands each run it meets to one CodeSpanPairer, which pairs by PairCodeSpan's rule; the other scan asks whether the line holds a backtick at all before building one; nothing is paired by hand"},
	"internal/core/mdrecord/mdrecord.go":   {3, "MaskCodeSpans walks to each run and pairs it through PairCodeSpan, runEnd steps over an unpaired run, and the fence opener refuses an info string holding a backtick"},
	"internal/core/mdrender/render.go":     {2, "the site renderer's inline walk hands each run to PairCodeSpan; the other is the escapable-punctuation range, which names a backtick among the ASCII punctuation"},
	"internal/core/question/check.go":      {2, "the question check's register rule walks to each run and pairs it through PairCodeSpan, stepping over an unpaired run whole"},
	"internal/core/surface/appendix.go":    {2, "codeRegions walks to each run and pairs it through PairCodeSpan, stepping over an unpaired run whole"},
}

// TestNoSecondCodeSpanPairer is the one-canonical-primitive detector for the
// code-span rule, the counterpart of mdrecord's TestNoSecondFenceRule.
//
// PairCodeSpan is the tree's one code-span pairer. The site renderer, the prose
// cleaner, the record's comment and span readers and the block escapers each
// paired runs by their own walk, and the renderer's closed a span where the
// opening run's text first recurred: it refused a span the escapers had judged
// balanced and left unescaped, failing the page (iss-2609262322244502). The
// guard's own first sweep found two more walks pairing single backticks inside
// longer runs (iss-2609262350446885).
//
// The check is spelling-only, as the fence detector's is: every file that seeks
// a backtick is named with a reason, which is a short list because most of the
// tree never seeks one. A backtick assembled at run time, or spelled as a
// number, is outside its reach and is left to review. The one spelling this
// package itself hands out, the FenceBacktick and FenceTilde constants, is
// judged by fenceConstantMisuses: outside termsafe a constant may only be an
// argument of LongestFenceRun, so it cannot become a scan the regex misses.
func TestNoSecondCodeSpanPairer(t *testing.T) {
	root := filepath.Join("..", "..") // internal/termsafe -> repository root
	self := filepath.Join(root, "internal", "termsafe")
	var offenders []string
	seen := map[string]bool{}
	for _, dir := range []string{"internal", "cmd"} {
		err := filepath.WalkDir(filepath.Join(root, dir), func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if path == self {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(d.Name(), ".go") || strings.HasSuffix(d.Name(), "_test.go") {
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			rel, _ := filepath.Rel(root, path)
			rel = filepath.ToSlash(rel)
			misuses, err := fenceConstantMisuses(rel, data)
			if err != nil {
				return err
			}
			offenders = append(offenders, misuses...)
			n := len(backtickScanRe.FindAllIndex(data, -1))
			if n == 0 {
				return nil
			}
			seen[rel] = true
			w, ok := backtickScanners[rel]
			switch {
			case !ok:
				offenders = append(offenders, fmt.Sprintf("%s (seeks a backtick %d time(s) and is not allowlisted)", rel, n))
			case w.count != n:
				offenders = append(offenders, fmt.Sprintf("%s (seeks a backtick %d time(s); the allowlist names %d)", rel, n, w.count))
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", dir, err)
		}
	}
	if len(offenders) > 0 {
		sort.Strings(offenders)
		t.Errorf("backtick scans outside termsafe (pair through PairCodeSpan, count a run through LongestFenceRun, or allowlist with a reason and the count):\n  %s",
			strings.Join(offenders, "\n  "))
	}
	// An allowlist entry for a file that no longer seeks a backtick is a stale
	// claim.
	for rel := range backtickScanners {
		if !seen[rel] {
			t.Errorf("%s is allowlisted but no longer seeks a backtick; remove the entry", rel)
		}
	}
}

// termsafeImportPath is the import path fenceConstantMisuses resolves the
// constants' package by, whatever name a file imports it under.
const termsafeImportPath = "github.com/intentdriven/abcd/internal/termsafe"

// fenceConstantNames are the exported fence characters a file outside this
// package may hand to LongestFenceRun and to nothing else.
var fenceConstantNames = map[string]bool{"FenceBacktick": true, "FenceTilde": true}

// fenceConstantMisuses reports each use, in the Go file src read from rel, of
// termsafe.FenceBacktick or termsafe.FenceTilde that is not an argument
// written directly in a termsafe.LongestFenceRun call. The constants exist so
// a writer can ask for a run without spelling a backtick; compared against a
// byte or held in a variable they would give a hand-written scan a spelling
// backtickScanRe cannot see. The file is parsed rather than matched, because
// whether a constant sits in that call's argument list is a question about the
// syntax tree, and the package is found by its import path, so an alias is
// judged too. A dot import of termsafe is refused outright, since it would let
// a bare FenceBacktick stand for the selector this check looks for.
func fenceConstantMisuses(rel string, src []byte) ([]string, error) {
	if !strings.Contains(string(src), "FenceBacktick") && !strings.Contains(string(src), "FenceTilde") {
		return nil, nil
	}
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, rel, src, parser.SkipObjectResolution)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", rel, err)
	}
	var out []string
	name := ""
	for _, imp := range f.Imports {
		if strings.Trim(imp.Path.Value, `"`) != termsafeImportPath {
			continue
		}
		switch {
		case imp.Name == nil:
			name = "termsafe"
		case imp.Name.Name == ".":
			out = append(out, fmt.Sprintf("%s (dot-imports termsafe, which hides its fence constants from this check)", fset.Position(imp.Pos())))
		default:
			name = imp.Name.Name
		}
	}
	if name == "" || name == "_" {
		return out, nil
	}
	isTermsafe := func(e ast.Expr, sel func(string) bool) bool {
		s, ok := e.(*ast.SelectorExpr)
		if !ok {
			return false
		}
		x, ok := s.X.(*ast.Ident)
		return ok && x.Name == name && sel(s.Sel.Name)
	}
	isConstant := func(e ast.Expr) bool {
		return isTermsafe(e, func(n string) bool { return fenceConstantNames[n] })
	}
	allowed := map[ast.Expr]bool{}
	ast.Inspect(f, func(n ast.Node) bool {
		if call, ok := n.(*ast.CallExpr); ok && isTermsafe(call.Fun, func(n string) bool { return n == "LongestFenceRun" }) {
			for _, arg := range call.Args {
				if isConstant(arg) {
					allowed[arg] = true
				}
			}
		}
		return true
	})
	ast.Inspect(f, func(n ast.Node) bool {
		if e, ok := n.(ast.Expr); ok && isConstant(e) && !allowed[e] {
			sel := e.(*ast.SelectorExpr)
			out = append(out, fmt.Sprintf("%s (uses %s.%s outside a LongestFenceRun argument list)", fset.Position(sel.Pos()), name, sel.Sel.Name))
		}
		return true
	})
	return out, nil
}

// TestFenceConstantMisuses pins what fenceConstantMisuses lets through: the
// constants written directly as LongestFenceRun's arguments, under any import
// name, and nothing else.
func TestFenceConstantMisuses(t *testing.T) {
	const header = "package p\n\nimport %s \"" + termsafeImportPath + "\"\n\n"
	for _, c := range []struct {
		name  string
		alias string // the import's name, "" for the default
		body  string
		want  int
	}{
		{"passed to LongestFenceRun", "", "func f(s string) int { return termsafe.LongestFenceRun(s, termsafe.FenceBacktick) }", 0},
		{"both constants passed", "", "func f(s string) int { return termsafe.LongestFenceRun(s, termsafe.FenceTilde, termsafe.FenceBacktick) }", 0},
		{"comparison outside a call", "", "func f(s string, i int) bool { return s[i] == termsafe.FenceBacktick }", 1},
		{"assigned to a variable", "", "func f() byte { c := termsafe.FenceTilde; return c }", 1},
		{"package-level variable", "", "var tick = termsafe.FenceBacktick", 1},
		{"inside a slice handed to the call", "", "func f(s string) int { return termsafe.LongestFenceRun(s, []byte{termsafe.FenceBacktick}...) }", 1},
		{"argument of another call", "", "func f(s string) int { return strings.IndexByte(s, termsafe.FenceBacktick) }", 1},
		{"one clean use, one comparison", "", "func f(s string, i int) bool { return termsafe.LongestFenceRun(s, termsafe.FenceBacktick) > 0 && s[i] == termsafe.FenceTilde }", 1},
		{"imported under another name, passed", "ts", "func f(s string) int { return ts.LongestFenceRun(s, ts.FenceBacktick) }", 0},
		{"imported under another name, compared", "ts", "func f(s string, i int) bool { return s[i] == ts.FenceBacktick }", 1},
		{"another package's constant of the same name", "", "func f(s string, i int) bool { return s[i] == other.FenceBacktick || termsafe.CodeSpan(s) == \"\" }", 0},
		{"dot import", ".", "func f(s string, i int) bool { return s[i] == FenceBacktick }", 1},
	} {
		src := fmt.Sprintf(header, c.alias) + c.body + "\n"
		got, err := fenceConstantMisuses("fixture.go", []byte(src))
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if len(got) != c.want {
			t.Errorf("%s: %d misuse(s) %q, want %d", c.name, len(got), got, c.want)
		}
	}
}
