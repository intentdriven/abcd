package site

// The closed page set and its per-page switches (itd-2609061543533170,
// criterion 4): the same pages for every repository, switched off per
// repository, never on to something extra.

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// withPages writes a `pages` block into the fixture's manifest.
func withPages(t *testing.T, f *fixture, block string) {
	t.Helper()
	p := filepath.Join(f.Root(), ManifestRelPath)
	raw, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	s := strings.Replace(string(raw), `"schema_version": 1,`, `"schema_version": 1,
  "pages": `+block+`,`, 1)
	if s == string(raw) {
		t.Fatal("the fixture manifest has no schema_version line to anchor the pages block on")
	}
	if err := os.WriteFile(p, []byte(s), 0o644); err != nil {
		t.Fatal(err)
	}
}

// htmlPages returns every rendered HTML page, keyed by output-relative path.
func htmlPages(t *testing.T, out string) map[string]string {
	t.Helper()
	pages := map[string]string{}
	err := filepath.WalkDir(out, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".html") {
			return err
		}
		raw, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(out, p)
		pages[filepath.ToSlash(rel)] = string(raw)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return pages
}

var hrefRe = regexp.MustCompile(`href="(/[^"#?]*)`)

// linksInto names every page that links under route.
func linksInto(pages map[string]string, route string) []string {
	var from []string
	for name, html := range pages {
		for _, m := range hrefRe.FindAllStringSubmatch(html, -1) {
			if strings.HasPrefix(m[1], "/"+route) {
				from = append(from, name+" → "+m[1])
			}
		}
	}
	return from
}

// TestEveryPageOfTheSetRendersByDefault: with no pages block, the whole set
// renders — the landing page, the explorer, a record page, the graph, the
// timeline, the glossary and the status page.
func TestEveryPageOfTheSetRendersByDefault(t *testing.T) {
	f := newFixture(t)
	withGlossary(t, f)
	out := t.TempDir()
	buildFixture(t, f, out)
	pages := htmlPages(t, out)
	for _, want := range []string{
		"index.html", "record/index.html", "record/adr/adr-1/index.html", "record/graph/index.html",
		"record/glossary/index.html", "record/health/index.html",
	} {
		if _, ok := pages[want]; !ok {
			t.Errorf("default build lacks %s", want)
		}
	}
	if !strings.Contains(pages["record/index.html"], `class="panel tl"`) {
		t.Error("default dashboard lacks the timeline")
	}
}

// TestASwitchedOffPageIsGoneAndNothingLinksToIt: each switch removes its page
// and every link the renderer would have drawn to it, so a reader never meets
// a link to a page the site does not have.
func TestASwitchedOffPageIsGoneAndNothingLinksToIt(t *testing.T) {
	for _, tc := range []struct {
		key, route string
	}{
		{"graph", routeGraph},
		{"glossary", routeGlossary},
		{"status", routeHealth},
	} {
		t.Run(tc.key, func(t *testing.T) {
			f := newFixture(t)
			withGlossary(t, f)
			withPages(t, f, `{"`+tc.key+`": false}`)
			out := t.TempDir()
			buildFixture(t, f, out)
			pages := htmlPages(t, out)
			for name := range pages {
				if strings.HasPrefix(name, tc.route) {
					t.Errorf("%s is switched off and %s was rendered", tc.key, name)
				}
			}
			if links := linksInto(pages, tc.route); len(links) > 0 {
				t.Errorf("%s is switched off and pages still link to it: %v", tc.key, links)
			}
			if _, ok := pages["record/index.html"]; !ok {
				t.Error("switching one page off took the explorer with it")
			}
		})
	}
	t.Run("timeline", func(t *testing.T) {
		f := newFixture(t)
		withPages(t, f, `{"timeline": false}`)
		out := t.TempDir()
		buildFixture(t, f, out)
		if strings.Contains(htmlPages(t, out)["record/index.html"], `class="panel tl"`) {
			t.Error("the timeline is switched off and the dashboard still draws it")
		}
	})
}

// TestTheExplorerSwitchTakesEveryExplorerPage: with the explorer off, the
// landing page is the site, and its header no longer offers the record.
func TestTheExplorerSwitchTakesEveryExplorerPage(t *testing.T) {
	f := newFixture(t)
	withGlossary(t, f)
	withPages(t, f, `{"explorer": false}`)
	out := t.TempDir()
	buildFixture(t, f, out)
	pages := htmlPages(t, out)
	for name := range pages {
		if name != "index.html" {
			t.Errorf("the explorer is off and %s was rendered", name)
		}
	}
	for _, r := range []string{"record/", "contributors/", "references/"} {
		if links := linksInto(pages, r); len(links) > 0 {
			t.Errorf("the explorer is off and the landing page links into it: %v", links)
		}
	}
}

// TestSwitchesTheSetCannotHonourAreRefused: the landing page and the record
// pages carry the site, a sub-page cannot be on under an explorer that is off,
// and a page outside the closed set is a typo, not a request.
func TestSwitchesTheSetCannotHonourAreRefused(t *testing.T) {
	for _, block := range []string{
		`{"landing": false}`,
		`{"record_pages": false}`,
		`{"explorer": false, "graph": true}`,
		`{"blog": true}`,
	} {
		t.Run(block, func(t *testing.T) {
			f := newFixture(t)
			withPages(t, f, block)
			_, err := LoadManifest(f.Root())
			if !errors.Is(err, ErrManifestInvalid) {
				t.Fatalf("pages %s: err = %v, want ErrManifestInvalid", block, err)
			}
		})
	}
	f := newFixture(t)
	withPages(t, f, `{"explorer": false, "record_pages": false}`)
	if _, err := LoadManifest(f.Root()); err != nil {
		t.Fatalf("switching the record pages off with the explorer is consistent: %v", err)
	}
}
