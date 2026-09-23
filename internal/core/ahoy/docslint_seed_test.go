package ahoy

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/banlist"
	"github.com/intentdriven/abcd/internal/core/lint"
	"github.com/intentdriven/abcd/internal/gittest"
)

// TestScaffoldedDocsLintRefusesAViolation is iss-2609150805167646's acceptance at
// the install seam: a repository abcd prepared runs a docs lint that CHECKS
// something. The shipped seed carried no rules, so a present-tense violation
// written inside a configured root drew "0 finding(s), 0 blocker(s)" — a green
// that meant nothing. The guard is proven both ways (guards-prove-themselves):
// the violation is refused, and the clean tree the scaffold itself wrote
// (CLAUDE.md and AGENTS.md included) raises no blocker.
func TestScaffoldedDocsLintRefusesAViolation(t *testing.T) {
	setupHermetic(t)
	repo := gittest.NewRepo(t).Root()
	if err := os.MkdirAll(filepath.Join(repo, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(rel, body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(repo, filepath.FromSlash(rel)), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("README.md", "# Example\n\nThe tool reads the configuration and reports what it finds.\n")
	write("docs/guide.md", "# Guide\n\nRun the check before you push.\n")

	res, err := Install(repo, installOpts(), RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != "clean" {
		t.Fatalf("install status = %q (remaining=%v), want clean", res.Status, res.Remaining)
	}
	cfgPath := filepath.Join(repo, filepath.FromSlash(banlist.PublicConfigRelPath))
	cfg, err := lint.LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("scaffolded docs-lint config does not load: %v", err)
	}

	// The ok side: the tree as the scaffold left it passes.
	findings, err := lint.Lint(cfg, repo)
	if err != nil {
		t.Fatalf("lint over the freshly prepared repo: %v", err)
	}
	for _, f := range findings {
		if f.Severity == "blocker" {
			t.Errorf("the scaffold's own tree raises a blocker: %s:%d [%s] %s", f.File, f.Line, f.RuleID, f.Message)
		}
	}

	// The refusal side: the report's own probe, inside a configured root.
	write("docs/guide.md", "# Guide\n\nPreviously this was different.\n")
	findings, err = lint.Lint(cfg, repo)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, f := range findings {
		if f.File == "docs/guide.md" && strings.HasPrefix(f.RuleID, "present_tense/") && f.Severity == "blocker" {
			found = true
		}
	}
	if !found {
		t.Fatalf("a present-tense violation inside a configured root drew no present_tense blocker; findings=%+v", findings)
	}
}

// TestDocsLintSeedCarriesTheWritingGuideRulesAndNoNames pins what the seed holds
// and what it withholds. abcd's own Writing-Guide currency families are seeded;
// the banned-names family is the repository's own and stays empty, since abcd
// cannot know which names a repository may not publish; and the harness family
// and the em-dash house-style token are left for the repository to declare
// (deliberateSeedOmissions carries the reasons).
func TestDocsLintSeedCarriesTheWritingGuideRulesAndNoNames(t *testing.T) {
	var seed struct {
		BannedTokens []lint.BannedToken         `json:"banned_tokens"`
		Rules        map[string]lint.RuleConfig `json:"rules"`
	}
	if err := json.Unmarshal([]byte(publicFamilySeed), &seed); err != nil {
		t.Fatalf("seed is not JSON: %v", err)
	}
	families := map[string]int{}
	for _, tok := range seed.BannedTokens {
		families[strings.SplitN(tok.ID, "/", 2)[0]]++
	}
	for _, want := range seededTokenFamilies {
		if families[want] == 0 {
			t.Errorf("seed carries no %s/ token; the Writing-Guide family must be seeded", want)
		}
	}
	for fam := range families {
		if !isSeededFamily(fam) {
			t.Errorf("seed carries a %s/ token; only %v are abcd's to seed", fam, seededTokenFamilies)
		}
	}
	for _, rule := range []string{"links_resolve", "stray_root_docs", "harness_leak"} {
		if rc, ok := seed.Rules[rule]; !ok || !rc.Enabled {
			t.Errorf("seed does not arm the %s rule", rule)
		}
	}
}

// deliberateSeedOmissions names every entry of abcd's own .abcd/docs-lint.json
// that the seed withholds ON PURPOSE, with the reason. A key ending in "/" names a
// whole token family; any other key names one token id or one rule. The parity
// test below reads this list, so an omission is structural: a canonical entry is
// either seeded unchanged or named here, and a new canonical entry fails the test
// until someone decides which.
var deliberateSeedOmissions = map[string]string{
	// The repository's own banned names: abcd cannot know which names a
	// repository may not publish, and a ban nobody declared fails a build over a
	// word the repository never chose.
	"names/": "the repository's own banned names",
	// A per-repository fit decision: refusing to name a specific agent tool is
	// right for abcd's published surface and wrong for a repository whose content
	// teaches those tools (112 false positives in the reporting repository,
	// iss-2609150805167646).
	"harness/": "per-repository fit: a repository may teach the tools it names",
	// abcd's house style, not a currency rule: the colon-at-the-pivot convention
	// is abcd's Writing Guide, and the reporting repository measured 419 of its 545
	// findings on it. Its fit for other repositories is the product thinker's
	// ruling, still owed, so the seed withholds it until that ruling lands.
	"punctuation/em-dash-in-list-item": "abcd house style; its fit elsewhere awaits the product thinker's ruling",
	// The citation rules police the research-citation apparatus abcd's own record
	// keeps (footnote and crosswalk conventions, and a baseline file the scaffold
	// does not write); a prepared repository has none of it to check.
	"citation_footnotes":      "abcd's citation apparatus",
	"citation_crosswalk_rows": "abcd's citation apparatus",
	"citation_url_syntax":     "abcd's citation apparatus",
	"citation_source_policy":  "abcd's citation apparatus",
	"citation_baseline":       "abcd's citation apparatus",
}

// deliberatelyOmitted reports whether a canonical token id or rule name is named
// in deliberateSeedOmissions, by itself or by its family.
func deliberatelyOmitted(id string) bool {
	if _, ok := deliberateSeedOmissions[id]; ok {
		return true
	}
	if fam, _, found := strings.Cut(id, "/"); found {
		_, ok := deliberateSeedOmissions[fam+"/"]
		return ok
	}
	return false
}

// seededTokenFamilies are the banned-token families the docs-lint seed carries:
// abcd's own Writing-Guide currency rules, as opposed to the repository's banned
// names and the families deliberateSeedOmissions withholds.
var seededTokenFamilies = []string{"present_tense", "spelling"}

// isSeededFamily reports whether a banned-token family is one the seed carries.
func isSeededFamily(family string) bool {
	for _, f := range seededTokenFamilies {
		if f == family {
			return true
		}
	}
	return false
}

// TestDocsLintSeedMatchesTheCanonicalRuleSet holds the seed to the set abcd runs
// on itself, so the two cannot drift: every canonical token and rule in this
// repository's .abcd/docs-lint.json is either seeded unchanged or named in
// deliberateSeedOmissions, nothing omitted is seeded, and nothing is seeded that
// has no canonical entry. A token matches on pattern, severity, successor and
// allow context; a rule on enabled and severity. The token message may differ,
// because the seed's message points a managed repository at a guide it does not
// host, and so may a rule's allowlist, since the scaffold writes CLAUDE.md.
func TestDocsLintSeedMatchesTheCanonicalRuleSet(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "..", ".abcd", "docs-lint.json"))
	if err != nil {
		t.Fatal(err)
	}
	var canon, seed struct {
		BannedTokens []lint.BannedToken         `json:"banned_tokens"`
		Rules        map[string]lint.RuleConfig `json:"rules"`
	}
	if err := json.Unmarshal(data, &canon); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(publicFamilySeed), &seed); err != nil {
		t.Fatal(err)
	}
	seeded := map[string]lint.BannedToken{}
	for _, tok := range seed.BannedTokens {
		seeded[tok.ID] = tok
	}
	used := map[string]bool{}
	for _, c := range canon.BannedTokens {
		s, ok := seeded[c.ID]
		delete(seeded, c.ID)
		if deliberatelyOmitted(c.ID) {
			used[c.ID] = true
			if fam, _, found := strings.Cut(c.ID, "/"); found {
				used[fam+"/"] = true
			}
			if ok {
				t.Errorf("seeded %s is named a deliberate omission (%s)", c.ID, omissionReason(c.ID))
			}
			continue
		}
		if !ok {
			t.Errorf("canonical %s is neither seeded nor named in deliberateSeedOmissions", c.ID)
			continue
		}
		if s.Pattern != c.Pattern || s.Severity != c.Severity || s.Successor != c.Successor ||
			strings.Join(s.AllowContext, "\x00") != strings.Join(c.AllowContext, "\x00") {
			t.Errorf("seeded %s drifted from the canonical entry:\n seed  %+v\n canon %+v", c.ID, s, c)
		}
	}
	for id := range seeded {
		t.Errorf("seeded %s has no canonical entry in .abcd/docs-lint.json", id)
	}

	for name, c := range canon.Rules {
		s, ok := seed.Rules[name]
		if deliberatelyOmitted(name) {
			used[name] = true
			if ok {
				t.Errorf("seeded rule %s is named a deliberate omission (%s)", name, omissionReason(name))
			}
			continue
		}
		if !ok {
			t.Errorf("canonical rule %s is neither seeded nor named in deliberateSeedOmissions", name)
			continue
		}
		if s.Enabled != c.Enabled || s.Severity != c.Severity {
			t.Errorf("seeded rule %s drifted: seed enabled=%v severity=%q, canon enabled=%v severity=%q",
				name, s.Enabled, s.Severity, c.Enabled, c.Severity)
		}
	}
	for name := range seed.Rules {
		if _, ok := canon.Rules[name]; !ok {
			t.Errorf("seeded rule %s has no canonical entry in .abcd/docs-lint.json", name)
		}
	}

	// A stale omission names nothing the canonical set holds, and would
	// silently pre-approve withholding an entry nobody has ruled on yet.
	for key := range deliberateSeedOmissions {
		if !used[key] {
			t.Errorf("deliberateSeedOmissions names %s, which the canonical set does not carry", key)
		}
	}
}

func omissionReason(id string) string {
	if r, ok := deliberateSeedOmissions[id]; ok {
		return r
	}
	fam, _, _ := strings.Cut(id, "/")
	return deliberateSeedOmissions[fam+"/"]
}
