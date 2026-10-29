package site

// Every page opens with a line saying it is the record of an abcd-managed
// repository, linking to that repository (itd-2610091010348892). Its words are
// an interface string from the closed allowlist, so the provenance gate still
// refuses any text a repository file does not account for.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The words sit in a <small>, so a repository whose own site-src/site.css
// predates the line's `.managed` rule (setup keeps a stylesheet the repository
// already has) still shows it as a small line at the top of the page, not as a
// paragraph of body text.
func managedLine(href, words string) string {
	return "<body>\n" + `<p class="managed"><small><a href="` + href + `">` + words + `</a></small></p>`
}

func TestEveryPageOpensWithTheManagedRepositoryLine(t *testing.T) {
	f := newFixture(t)
	_, files := buildFixtureAt(t, f, "")
	ui, err := LoadUI(f.Root(), "site-src/ui.json")
	if err != nil {
		t.Fatal(err)
	}
	if ui.ManagedRecord == "" {
		t.Fatal("the allowlist declares no words for the line")
	}
	want := managedLine("https://example.invalid/fixture/repo", escapeText(ui.ManagedRecord))
	pages := 0
	for rel, data := range files {
		if !strings.HasSuffix(rel, ".html") {
			continue
		}
		pages++
		if !strings.Contains(string(data), want) {
			t.Errorf("%s does not open with the managed-repository line %q", rel, want)
		}
	}
	if pages < 5 {
		t.Fatalf("only %d pages were built", pages)
	}
}

// A managed repository need not carry a package manifest. Its repository is
// then the origin remote on github.com, which is what a release workflow's
// checkout carries, so the line still links it.
func TestTheLineLinksTheOriginWithoutAPackageManifest(t *testing.T) {
	f := newFixture(t)
	if err := os.Remove(filepath.Join(f.Root(), ".claude-plugin", "plugin.json")); err != nil {
		t.Fatal(err)
	}
	f.gitOut("remote", "add", "origin", "git@github.com:acme/widgets.git")
	meta, err := LoadRepoMeta(f.Root())
	if err != nil {
		t.Fatal(err)
	}
	if meta.Repository != "https://github.com/acme/widgets" {
		t.Fatalf("repository = %q, want the origin's github.com address", meta.Repository)
	}
}

// With no repository address at all the line still says what the site is.
func TestTheLineStandsWithoutARepositoryAddress(t *testing.T) {
	c := &composer{ui: UI{ManagedRecord: "The record."}}
	if got := c.managedLine(); got != `<p class="managed"><small>The record.</small></p>` {
		t.Fatalf("line = %q", got)
	}
}

// The seed stylesheet's rule sizes the line itself, so the <small> inside it
// takes the rule's size rather than shrinking the words a second time.
func TestTheSeedStylesheetSizesTheLineOnce(t *testing.T) {
	css, err := setupSources.ReadFile("setupsrc/site.css")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(css), ".managed small{font-size:inherit}") {
		t.Fatal("the seed stylesheet does not size the managed line's <small> as the line")
	}
}
