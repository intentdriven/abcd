package site

// The project route (itd-2610091010348892, the product thinker's ruling of
// 2026-10-15 that a project's site renders only what abcd already writes).
// A project site's `.abcd/site.json` holds settings only; its about page is
// derived from the identity block, the brief's opening, the shipped intents'
// press releases and CHANGELOG.md; its interface words, styles and scripts come
// from the binary; and the gate refuses any span from a file abcd does not
// write.

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// projectSettingsJSON is a project site's settings file as setup writes it.
const projectSettingsJSON = `{
  "schema_version": 1,
  "mode": "project",
  "purpose": "The fixture's project-site settings.",
  "hosting": {
    "provider": "github-pages",
    "name": "fixture",
    "publish": true
  }
}
`

// Words that sit only in a person's own files: the README and the docs. A
// project site renders none of them.
const (
	readmeOnlyWords = "Hand-written readme words the site never shows."
	docsOnlyWords   = "The fixture's documentation index."
)

// newProjectFixture is the fixture repository switched to the project route:
// its composition replaced by project settings, its site sources removed, an
// identity pointer and a brief opening added, and a README beside its docs.
func newProjectFixture(t *testing.T) *fixture {
	t.Helper()
	f := newFixture(t)
	f.write(ManifestRelPath, projectSettingsJSON)
	if err := os.RemoveAll(filepath.Join(f.Root(), "site-src")); err != nil {
		t.Fatal(err)
	}
	f.write(".abcd/positioning.json", `{"schema_version": 1, "block": {"file": ".abcd/development/brief/01-product/README.md", "heading": "Identity (canonical)"}, "severity": "warn", "surfaces": []}`+"\n")
	f.write(BriefOpeningRelPath, "# Press Release\n\n## Press Release\n\n"+
		"> **The fixture brief opens here.** It says what the fixture is for, in the brief's own words.\n\n"+
		"## Why This Matters\n\nA later section the opening does not reach.\n")
	f.write("README.md", "# Fixture\n\n"+readmeOnlyWords+"\n")
	f.commitAt("2026-03-07T09:00:00+00:00", "chore: the project route", "None")
	// A managed repository on GitHub: the origin remote is where a project
	// site's repository address and name come from.
	f.git("2026-03-07T09:00:00+00:00", "remote", "add", "origin", "https://github.com/"+projectOrigin+".git")
	return f
}

// projectOrigin is the project fixture's GitHub repository, from its origin
// remote.
const (
	projectOrigin     = "fixture-owner/fixture-repo"
	projectOriginURL  = "https://github.com/" + projectOrigin
	projectOriginName = "fixture-repo"
)

func buildProject(t *testing.T, f *fixture) (string, map[string][]byte) {
	t.Helper()
	out := t.TempDir()
	if _, err := Build(Request{RepoRoot: f.Root(), OutDir: out, Stamp: fixtureStamp, Project: true}); err != nil {
		t.Fatalf("project build: %v", err)
	}
	return out, siteFiles(t, out)
}

var dataSrcRe = regexp.MustCompile(`data-src="([^"#]*)`)

func TestAProjectBuildRendersItsAboutPageFromWhatAbcdWrites(t *testing.T) {
	f := newProjectFixture(t)
	out, files := buildProject(t, f)
	about := string(files["index.html"])
	for _, want := range []string{
		// The identity block, as the hero.
		`<p class="eyebrow" data-src=".abcd/development/brief/01-product/README.md#identity-canonical">fixture — A Fixture Product</p>`,
		`<h1 class="tagline" data-src=".abcd/development/brief/01-product/README.md#identity-canonical">A fixture tagline for a fixture product.</h1>`,
		// The brief's opening, and nothing after it.
		`data-src="` + BriefOpeningRelPath + `#press-release">Press Release</h2>`,
		"The fixture brief opens here.",
		// The newest shipped intent the record shows delivered.
		"A fixture user opens the page and sees the record.",
		// The release list.
		`>v0.2.0</a> · 2026-02-11</li>`, `>v0.1.0</a> · 2026-01-06</li>`,
	} {
		if !strings.Contains(about, want) {
			t.Errorf("the about page lacks %q", want)
		}
	}
	if strings.Contains(about, "A later section the opening does not reach.") {
		t.Error("the about page renders the brief past its opening")
	}
	for rel, data := range files {
		for _, words := range []string{readmeOnlyWords, docsOnlyWords, "Who this is for"} {
			if strings.Contains(string(data), words) {
				t.Errorf("%s renders a person's own words %q", rel, words)
			}
		}
		for _, m := range dataSrcRe.FindAllSubmatch(data, -1) {
			if !abcdWrittenSource(string(m[1])) {
				t.Errorf("%s carries a span from %q, which abcd does not write", rel, m[1])
			}
		}
	}
	// The interface words, styles and scripts are the binary's; the
	// repository carries no site sources, and nothing of abcd's own (the
	// redirect map, the installer) is served.
	for src, dst := range map[string]string{"site.css": "site.css", "site.js": "site.js", "record.js": "record.js", "headers": "_headers"} {
		want, err := setupSources.ReadFile("setupsrc/" + src)
		if err != nil {
			t.Fatal(err)
		}
		if string(files[dst]) != string(want) {
			t.Errorf("%s is not the binary's %s", dst, src)
		}
	}
	for _, absent := range []string{"_redirects", "install.sh"} {
		if _, ok := files[absent]; ok {
			t.Errorf("a project site serves %s", absent)
		}
	}
	// The pages that quote a person's files are left out.
	if _, ok := files["references/index.html"]; ok {
		t.Error("a project site renders the references page, which quotes ACKNOWLEDGEMENTS.md")
	}
	if strings.Contains(about, `href="/#`) {
		t.Error("the header links a composed chapter the about page does not have")
	}

	res, err := Check(CheckRequest{RepoRoot: f.Root(), OutDir: out, Project: true})
	if err != nil {
		t.Fatal(err)
	}
	if !res.OK() {
		for _, fd := range res.Findings {
			t.Errorf("finding: %+v", fd)
		}
	}
}

// Anything absent is left out, never filled: a project with a changelog and
// nothing else gets the release list alone.
func TestAnAboutPageLeavesOutWhatTheRepositoryDoesNotHold(t *testing.T) {
	f := newProjectFixture(t)
	for _, rel := range []string{".abcd/positioning.json", BriefOpeningRelPath} {
		if err := os.Remove(filepath.Join(f.Root(), filepath.FromSlash(rel))); err != nil {
			t.Fatal(err)
		}
	}
	_, files := buildProject(t, f)
	about := string(files["index.html"])
	if strings.Contains(about, `class="hero"`) || strings.Contains(about, "The fixture brief opens here.") {
		t.Error("the about page renders a part the repository does not hold")
	}
	if !strings.Contains(about, `id="releases"`) {
		t.Error("the about page lacks the release list the changelog holds")
	}
}

// The gate's project half: a span from docs/ or the README fails, on the about
// page and on a record page alike.
func TestTheProjectGateRefusesASpanFromDocsOrTheReadme(t *testing.T) {
	f := newProjectFixture(t)
	out, files := buildProject(t, f)
	inject := `<p data-src="docs/README.md">x</p><p data-src="README.md#fixture">y</p></main>`
	var record string
	for rel := range files {
		if strings.HasPrefix(rel, "record/adr/") && strings.HasSuffix(rel, "index.html") {
			record = rel
			break
		}
	}
	if record == "" {
		t.Fatal("the build wrote no ADR page")
	}
	for _, rel := range []string{"index.html", record} {
		page := strings.Replace(string(files[rel]), "</main>", inject, 1)
		if err := os.WriteFile(filepath.Join(out, filepath.FromSlash(rel)), []byte(page), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	res, err := Check(CheckRequest{RepoRoot: f.Root(), OutDir: out, Project: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{"index.html", record} {
		for _, src := range []string{"docs/README.md", "README.md#fixture"} {
			found := false
			for _, fd := range res.Findings {
				if fd.Check == CheckProvenance && fd.Where == rel && fd.Source == src && strings.Contains(fd.Detail, "abcd does not write") {
					found = true
				}
			}
			if !found {
				t.Errorf("no provenance finding on %s for the span from %s; findings: %+v", rel, src, res.Findings)
			}
		}
	}
}

// The route is named, never inferred: a build whose flag and settings file
// disagree is refused, both ways.
func TestABuildAndItsSettingsAgreeOnTheRoute(t *testing.T) {
	project := newProjectFixture(t)
	_, err := Build(Request{RepoRoot: project.Root(), OutDir: t.TempDir(), Stamp: fixtureStamp})
	if err == nil || !strings.Contains(err.Error(), "--project") {
		t.Fatalf("a composed build of project settings = %v, want a refusal naming --project", err)
	}
	if _, err := Check(CheckRequest{RepoRoot: project.Root(), OutDir: t.TempDir()}); err == nil || !strings.Contains(err.Error(), "--project") {
		t.Fatalf("a composed gate over project settings = %v, want a refusal naming --project", err)
	}
	composed := newFixture(t)
	_, err = Build(Request{RepoRoot: composed.Root(), OutDir: t.TempDir(), Stamp: fixtureStamp, Project: true})
	if err == nil || !strings.Contains(err.Error(), "composition") {
		t.Fatalf("a project build of a composition = %v, want a refusal", err)
	}
}

// A project site's settings carry settings and nothing else.
func TestProjectSettingsCarryNoComposition(t *testing.T) {
	for name, c := range map[string]struct{ body, want string }{
		"a composition key": {strings.Replace(projectSettingsJSON, `"hosting"`, `"home": {"hero": {"page": "docs/README.md"}}, "hosting"`, 1), "home is a composition key"},
		"no hosting block":  {`{"schema_version": 1, "mode": "project", "purpose": "x"}`, "need a hosting block"},
		"a composed host":   {strings.Replace(projectSettingsJSON, `"publish": true`, `"domain": "example.com"`, 1), ""},
		"an unknown mode":   {strings.Replace(projectSettingsJSON, `"project"`, `"other"`, 1), `mode "other"`},
	} {
		t.Run(name, func(t *testing.T) {
			f := newProjectFixture(t)
			body := c.body
			if name == "a composed host" {
				body = strings.Replace(body, `"github-pages"`, `"cloudflare"`, 1)
				c.want = "publishes a composed site"
			}
			f.write(ManifestRelPath, body)
			_, err := LoadManifest(f.Root())
			if !errors.Is(err, ErrManifestInvalid) || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("LoadManifest = %v, want a refusal naming %q", err, c.want)
			}
		})
	}
	f := newProjectFixture(t)
	m, err := LoadManifest(f.Root())
	if err != nil || !m.Project() || !m.Hosting.On() {
		t.Fatalf("project settings = %+v, %v", m, err)
	}
}

// Every page of a project site opens with the managed-repository line, in the
// binary's words, linking the repository.
func TestEveryProjectPageOpensWithTheManagedRepositoryLine(t *testing.T) {
	f := newProjectFixture(t)
	_, files := buildProject(t, f)
	ui, err := projectUI()
	if err != nil {
		t.Fatal(err)
	}
	want := managedLine(projectOriginURL, escapeText(ui.ManagedRecord))
	pages := 0
	for rel, data := range files {
		if !strings.HasSuffix(rel, ".html") {
			continue
		}
		pages++
		if !strings.Contains(string(data), want) {
			t.Errorf("%s does not open with the managed-repository line", rel)
		}
	}
	if pages < 5 {
		t.Fatalf("only %d pages were built", pages)
	}
}

// A project site below its GitHub Pages path links nothing outside it, and the
// gate holds it there.
func TestAProjectSiteBuildsBelowItsBasePath(t *testing.T) {
	f := newProjectFixture(t)
	out := t.TempDir()
	if _, err := Build(Request{RepoRoot: f.Root(), OutDir: out, Stamp: fixtureStamp, Project: true, BasePath: "/fixture/"}); err != nil {
		t.Fatal(err)
	}
	res, err := Check(CheckRequest{RepoRoot: f.Root(), OutDir: out, Project: true, BasePath: "/fixture/"})
	if err != nil {
		t.Fatal(err)
	}
	if !res.OK() {
		t.Fatalf("findings: %+v", res.Findings)
	}
}

// A project site's repository facts are the origin remote's alone. The
// package manifest is a file a person writes, so its name, author, licence and
// repository address reach no page and pass no gate; on a fork the managed
// line links the fork, never the upstream the manifest names.
func TestAProjectSiteReadsNothingFromThePackageManifest(t *testing.T) {
	f := newProjectFixture(t)
	f.write(".claude-plugin/plugin.json", `{
  "name": "Quillfeather",
  "description": "A hostile manifest.",
  "author": {"name": "Ottoline Brackenbury"},
  "repository": "https://github.com/upstream-elsewhere/quillfeather",
  "license": "Licensed under the Brackenbury Terms of Gentle Reuse"
}
`)
	f.commitAt("2026-03-08T09:00:00+00:00", "chore: a manifest of its own", "None")
	out, files := buildProject(t, f)
	for rel, data := range files {
		for _, words := range []string{"Quillfeather", "quillfeather", "Ottoline", "Brackenbury", "Gentle Reuse", "upstream-elsewhere"} {
			if strings.Contains(string(data), words) {
				t.Errorf("%s carries %q from the package manifest", rel, words)
			}
		}
		if strings.HasSuffix(rel, ".html") && !strings.Contains(string(data), `<p class="managed"><small><a href="`+projectOriginURL+`">`) {
			t.Errorf("%s: the managed line does not link the origin remote", rel)
		}
	}
	res, err := Check(CheckRequest{RepoRoot: f.Root(), OutDir: out, Project: true})
	if err != nil {
		t.Fatal(err)
	}
	if !res.OK() {
		t.Fatalf("findings: %+v", res.Findings)
	}
}
