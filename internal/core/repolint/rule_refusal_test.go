package repolint_test

import (
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/repolint"
)

// A target that refuses must not leave bare `abcd lint` reading clean
// (iss-2610100649479892): its refusal is an error finding naming the target
// and what it refused, so the aggregate exits 2 as the target alone does.

// docsLintRoots is a loadable docs-lint config with one rule armed over roots.
func docsLintRoots(roots ...string) string {
	return `{"roots": ["` + strings.Join(roots, `", "`) + `"], ` +
		`"rules": {"links_resolve": {"enabled": true, "severity": "blocker"}}}`
}

// requireRefusal fails unless res carries ruleID's refusal of target at error
// severity, mentioning each of want, and exits 2.
func requireRefusal(t *testing.T, res repolint.Result, ruleID, target string, want ...string) {
	t.Helper()
	f := findingFor(res, ruleID)
	if f == nil {
		t.Fatalf("no %s finding for the %s target's refusal: findings=%+v skipped=%v", ruleID, target, res.Findings, res.Skipped)
	}
	if f.Severity != repolint.SeverityError {
		t.Errorf("severity = %q, want error: a refusal checked nothing", f.Severity)
	}
	for _, w := range append([]string{"abcd lint " + target}, want...) {
		if !strings.Contains(f.Message, w) {
			t.Errorf("message does not mention %q: %s", w, f.Message)
		}
	}
	if res.ExitCode != 2 {
		t.Errorf("exit = %d, want 2", res.ExitCode)
	}
}

// The reporter's shape: CLAUDE.md retired, still in the roots, and no docs/.
func TestDocsCurrencyRootThatDoesNotResolveIsAnError(t *testing.T) {
	res := newFixtureRepo(t).conforming().
		file("README.md", "# readme\n").
		file(".abcd/docs-lint.json", docsLintRoots("CLAUDE.md", "README.md")).
		commit().run()
	requireRefusal(t, res, "docs-currency", "docs", "CLAUDE.md", "does not exist")
	for _, id := range res.Skipped {
		if id == "docs-currency" {
			t.Errorf("docs-currency skipped although .abcd/docs-lint.json configures roots: %v", res.Skipped)
		}
	}
}

// The refusal is one finding among the rest, never an aborted lint that takes
// the other rules' results down with it.
func TestDocsCurrencyRefusalKeepsTheOtherFindings(t *testing.T) {
	res := newFixtureRepo(t).
		file(".gitignore", ".abcd/.work.local/\n").
		file(".abcd/development/README.md", "# durable record\n").
		file(".abcd/work/README.md", "# shared\n"). // no DECISIONS.md: a decision-durability warn
		file(".abcd/.work.local/NEXT.md", "# local handoff\n").
		file("AGENTS.md", "# conventions\n").
		file("docs/a.md", "# a\n").
		file(".abcd/docs-lint.json", docsLintRoots("docs", "CLAUDE.md")).
		commit().run()
	requireRefusal(t, res, "docs-currency", "docs", "CLAUDE.md")
	if findingFor(res, "decision-durability") == nil {
		t.Errorf("the docs refusal took the other rules' findings with it: %+v", res.Findings)
	}
}

// A config the docs target cannot load is a refusal too.
func TestDocsCurrencyMalformedConfigIsAnError(t *testing.T) {
	res := newFixtureRepo(t).conforming().
		file("docs/a.md", "# a\n").
		file(".abcd/docs-lint.json", "{ not json").
		commit().run()
	requireRefusal(t, res, "docs-currency", "docs")
}

// A repository with neither docs/ nor a docs-lint config has no docs target.
func TestDocsCurrencyStillSkippedWithNoDocsAndNoConfig(t *testing.T) {
	res := newFixtureRepo(t).conforming().commit().run()
	if findingFor(res, "docs-currency") != nil {
		t.Fatalf("docs-currency ran with neither docs/ nor a config: %+v", res.Findings)
	}
}

// A site the site target cannot render is its refusal.
func TestSiteGatesRefusalIsAnError(t *testing.T) {
	res := newFixtureRepo(t).conforming().
		file(".abcd/site.json", "{").
		commit().run()
	requireRefusal(t, res, "site-gates", "site")
}

// A positioning registry the identity target cannot load is its refusal.
func TestPositioningMalformedConfigIsAnError(t *testing.T) {
	res := newFixtureRepo(t).conforming().withPositioning().
		file(".abcd/positioning.json", "{ not json").commit().run()
	requireRefusal(t, res, "identity-positioning", "identity")
}

// So is an identity block it cannot read, whatever severity the family takes.
func TestPositioningMissingBlockIsAnError(t *testing.T) {
	res := newFixtureRepo(t).conforming().withPositioning().
		file(".abcd/development/brief.md", "# no identity block here\n").commit().run()
	requireRefusal(t, res, "identity-positioning", "identity")
}
