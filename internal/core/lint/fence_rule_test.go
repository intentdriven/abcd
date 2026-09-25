package lint

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// The rules that skip example text read fences by mdrecord's rule, the tree's
// one notion of a fence (iss-2609250955209513). A private toggle that flipped on
// every line starting with three backticks linted a tilde-fenced example as
// prose, and — the silent half — a backtick line inside a tilde block switched
// the mask on, so the live prose after the block was masked until the next
// backtick line and a real finding there was never reported.

func TestFenceMaskReadsTildeFencesAndTheirContents(t *testing.T) {
	for name, tc := range map[string]struct {
		body string
		want []bool
	}{
		"prose after a tilde block holding a backtick line is live": {
			"~~~\n```go\n~~~\n\nLIVE PROSE LINE", []bool{true, true, true, false, false}},
		"a tilde-fenced example is masked": {
			"~~~\nexample prose\n~~~\nlive", []bool{true, true, true, false}},
		"a four-backtick fence holds a quoted three-backtick line": {
			"````\n```\nquoted\n```\n````\nlive", []bool{true, true, true, true, true, false}},
	} {
		if got := fenceMask(strings.Split(tc.body, "\n")); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%s: fenceMask = %v, want %v", name, got, tc.want)
		}
	}
}

// TestFenceMaskLeavesCommentedTextToTheRules: the rules read commented text, and
// moving them onto the fence rule does not change that — mdrecord's comment
// mask is not theirs to take.
func TestFenceMaskLeavesCommentedTextToTheRules(t *testing.T) {
	if got := fenceMask([]string{"<!--", "parked", "-->"}); !reflect.DeepEqual(got, []bool{false, false, false}) {
		t.Fatalf("fenceMask masked a comment: %v", got)
	}
}

// TestLinksResolveAfterATildeBlockHoldingABacktickLine is the silent half at a
// rule: the broken link below the tilde block is live prose and is reported.
func TestLinksResolveAfterATildeBlockHoldingABacktickLine(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "rec/doc.md",
		"~~~\n```go\n[t](fenced-missing.md)\n~~~\n\nbroken: [t](missing.md)\n")
	cfg := Config{
		Roots: []string{"rec"},
		Rules: map[string]RuleConfig{"links_resolve": {Enabled: true, Severity: "blocker"}},
	}
	fs, err := Lint(cfg, root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, "links_resolve"); n != 1 || !hasFinding(fs, filepath.Join("rec", "doc.md"), "links_resolve", 6) {
		t.Fatalf("want exactly the live broken link on line 6, got %d: %+v", n, fs)
	}
}
