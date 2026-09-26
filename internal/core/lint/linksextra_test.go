package lint

import (
	"path/filepath"
	"testing"
)

// links_resolve reaches the working tier through extra_roots without arming any
// other rule there: a dead relative link in a ledger record is a finding, an
// exempt file is skipped, and a banned token in the same tree is not judged
// (iss-2608230752354927).
func TestLinksResolveWalksExtraRoots(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "rec/a.md", "# a\n")
	writeFile(t, root, "work/issues/open/iss-1-x.md", "see [gone](../../../rec/gone.md) and [a](../../../rec/a.md), fn-9\n")
	writeFile(t, root, "work/reviews/old.md", "[dead](nowhere.md)\n")
	cfg := Config{
		Roots:        []string{"rec"},
		BannedTokens: []BannedToken{{ID: "t", Pattern: `\bfn-`, Message: "m", Severity: "blocker", Successor: "spc-", AllowContext: []string{"historical"}}},
		Rules: map[string]RuleConfig{"links_resolve": {
			Enabled: true, Severity: "blocker",
			ExtraRoots: []string{"work"}, Exempt: []string{"work/reviews/*"},
		}},
	}
	fs, err := Lint(cfg, root)
	if err != nil {
		t.Fatal(err)
	}
	if len(fs) != 1 || fs[0].RuleID != "links_resolve" || fs[0].File != filepath.Join("work", "issues", "open", "iss-1-x.md") {
		t.Fatalf("want exactly the one dead ledger link, got %+v", fs)
	}
}

// The shipped config walks .abcd/work for links, so a dead link in the ledger
// fails record-lint.
func TestLinksResolveCoversTheWorkTierInRealConfig(t *testing.T) {
	cfg, err := LoadConfig(filepath.Join("..", "..", "..", ".abcd", "record-lint.json"))
	if err != nil {
		t.Fatal(err)
	}
	rc := cfg.Rules["links_resolve"]
	for _, r := range rc.ExtraRoots {
		if r == ".abcd/work" {
			return
		}
	}
	t.Fatalf("links_resolve must walk .abcd/work: extra_roots = %v", rc.ExtraRoots)
}
