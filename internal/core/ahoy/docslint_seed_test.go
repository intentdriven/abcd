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
// and what it withholds. abcd's own Writing-Guide token families are seeded; the
// banned-names family is the repository's own and stays empty, since abcd cannot
// know which names a repository may not publish; and the harness family is a
// per-repository fit decision (a repository that teaches those tools names them
// as its content), so it is left for the repository to declare.
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

// TestDocsLintSeedMatchesTheCanonicalRuleSet holds the seed's token rules to the
// set abcd runs on itself, so the two cannot drift: every seeded token has the
// same pattern, severity and allow context as the entry of that id in this
// repository's .abcd/docs-lint.json, and every entry of a seeded family there is
// seeded. The message may differ, because the seed's message points a managed
// repository at a guide it does not host.
func TestDocsLintSeedMatchesTheCanonicalRuleSet(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "..", ".abcd", "docs-lint.json"))
	if err != nil {
		t.Fatal(err)
	}
	var canon, seed struct {
		BannedTokens []lint.BannedToken `json:"banned_tokens"`
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
	for _, c := range canon.BannedTokens {
		if !isSeededFamily(strings.SplitN(c.ID, "/", 2)[0]) {
			continue
		}
		s, ok := seeded[c.ID]
		if !ok {
			t.Errorf("canonical %s is not seeded", c.ID)
			continue
		}
		delete(seeded, c.ID)
		if s.Pattern != c.Pattern || s.Severity != c.Severity || s.Successor != c.Successor ||
			strings.Join(s.AllowContext, "\x00") != strings.Join(c.AllowContext, "\x00") {
			t.Errorf("seeded %s drifted from the canonical entry:\n seed  %+v\n canon %+v", c.ID, s, c)
		}
	}
	for id := range seeded {
		t.Errorf("seeded %s has no canonical entry in .abcd/docs-lint.json", id)
	}
}
