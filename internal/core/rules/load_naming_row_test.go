package rules

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// load_naming_row_test.go: the `LOAD` row of 02-constraints/04-naming.md is
// written from the bundled domain, so it names the recall words and aliases the
// domain carries, no others, and every rule it injects (iss-2611040554044986).

// backticked finds each code span of a clause.
var backticked = regexp.MustCompile("`([^`]+)`")

// namingRowList returns the code spans of the parenthesised list that follows
// opener in row, in the order the row gives them.
func namingRowList(t *testing.T, row, opener string) []string {
	t.Helper()
	start := strings.Index(row, opener)
	if start < 0 {
		t.Fatalf("the `LOAD` row has no %q list:\n%s", opener, row)
	}
	rest := row[start+len(opener):]
	end := strings.Index(rest, ")")
	if end < 0 {
		t.Fatalf("the `LOAD` row's %q list never closes:\n%s", opener, row)
	}
	var out []string
	for _, m := range backticked.FindAllStringSubmatch(rest[:end], -1) {
		out = append(out, m[1])
	}
	return out
}

// TestLoadNamingRowIsWrittenFromTheBundledDomain reads the row and fails when
// its recall words or aliases are not the bundled domain's own, when a phrase it
// gives as recalling the domain does not recall it, or when it leaves out one of
// the domain's rules: the row once gave "pegging or saturating every core",
// which matches nothing, and stopped at the trust rule's three parts, short of
// the load gate.
func TestLoadNamingRowIsWrittenFromTheBundledDomain(t *testing.T) {
	root, err := fsutil.ModuleRoot(".")
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, ".abcd", "development", "brief", "02-constraints", "04-naming.md"))
	if err != nil {
		t.Fatal(err)
	}
	var row string
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "| `LOAD` |") {
			row = line
			break
		}
	}
	if row == "" {
		t.Fatal("04-naming.md has no `LOAD` row")
	}

	d, ok := Defaults().Domains["LOAD"]
	if !ok {
		t.Fatal("the bundled defaults carry no LOAD domain")
	}
	if got := namingRowList(t, row, "Recall keywords ("); !slices.Equal(got, d.Recall) {
		t.Errorf("the `LOAD` row's recall keywords are\n  %q\nand the bundled domain's are\n  %q", got, d.Recall)
	}
	aliases := namingRowList(t, row, "aliases (")
	if !slices.Equal(aliases, d.Aliases) {
		t.Errorf("the `LOAD` row's aliases are\n  %q\nand the bundled domain's are\n  %q", aliases, d.Aliases)
	}

	// Each phrase the row gives recalls the domain as a prompt of its own, which
	// is what a reader takes the row to say.
	rs := Defaults()
	for _, phrase := range append(namingRowList(t, row, "Recall keywords ("), aliases...) {
		if !has(rs.Match(phrase), "LOAD") {
			t.Errorf("the `LOAD` row gives %q as recalling the domain, and a prompt of it does not", phrase)
		}
	}
	for _, stale := range []string{"pegging or saturating every core"} {
		if strings.Contains(row, stale) {
			t.Errorf("the `LOAD` row still says %q, which recalls nothing", stale)
		}
	}

	// All four rules, each by words only its own summary carries.
	if len(d.Rules) != 4 {
		t.Fatalf("the bundled LOAD domain carries %d rules, and the row and this test are written for 4: rewrite both", len(d.Rules))
	}
	if !strings.Contains(row, "its four rules") {
		t.Errorf("the `LOAD` row does not say the domain injects four rules:\n%s", row)
	}
	for i, want := range [][]string{
		{"one owned process group", "never by pattern"},
		{"clean proven by what is running"},
		{"consent", "a cap below the core count"},
		{"the load gate", "`abcd implement load --wait`", "`abcd implement load --race-gate`", "`--narrow`", "`load.preflights`"},
	} {
		for _, w := range want {
			if !strings.Contains(row, w) {
				t.Errorf("the `LOAD` row leaves %q out of rule %d:\n%s", w, i+1, row)
			}
		}
	}
}
