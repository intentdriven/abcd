package repolint_test

import (
	"testing"

	"github.com/intentdriven/abcd/internal/core/repolint"
)

// projectSiteSettings is a project site's settings file as `abcd site setup
// --provider github-pages` writes it (itd-2610091010348892).
const projectSiteSettings = `{
  "schema_version": 1,
  "mode": "project",
  "purpose": "The fixture's project-site settings.",
  "hosting": {"provider": "github-pages", "name": "widgets", "publish": true}
}
`

// Bare `abcd lint` gates a project site on its own route: the settings file
// names it, so the site target is not refused for a missing --project.
func TestSiteGatesGateAProjectSiteOnItsOwnRoute(t *testing.T) {
	b := newFixtureRepo(t).conforming().
		file(".abcd/site.json", projectSiteSettings).
		file("CHANGELOG.md", "# Changelog\n\n## [Unreleased]\n\n## [0.1.0] - 2026-01-06\n\n### Added\n\n- The first cut.\n").
		commit()
	git(t, b.root, "remote", "add", "origin", "https://github.com/acme/widgets.git")
	res := b.run()
	for _, f := range res.Findings {
		if f.RuleID != "site-gates" {
			continue
		}
		if f.Severity == repolint.SeverityError {
			t.Errorf("the site target refused a project site: %s", f.Message)
			continue
		}
		t.Errorf("the project site fails a gate: %s", f.Message)
	}
	for _, s := range res.Skipped {
		if s == "site-gates" {
			t.Fatal("the site target was skipped for a repository that declares a site")
		}
	}
}
