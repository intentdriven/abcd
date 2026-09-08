package site

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestManifestRefusesUnknownAndUnusableKeys pins the manifest's two promises:
// a key the binary does not know is refused, and a key it knows but does not
// act on YET is still validated. The second is the one that rots quietly — a
// path typo under a deferred key would otherwise sit in the file looking
// correct until the slice that consumes it lands, and then fail in a change
// that did not cause it.
func TestManifestRefusesUnknownAndUnusableKeys(t *testing.T) {
	f := newFixture(t)
	manifest := filepath.Join(f.Root(), ".abcd", "site.json")
	original, err := os.ReadFile(manifest)
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct{ name, from, to, says string }{
		{"unknown top-level key", `"schema_version": 1,`, `"schema_version": 1, "colour": "blue",`, "colour"},
		{"unknown nested key", `"letter": "a",`, `"letter": "a", "hue": 3,`, "hue"},
		{"wrong schema version", `"schema_version": 1,`, `"schema_version": 2,`, "schema_version"},
		{"unimplemented icon rule", `"icons": "image-before-lead-in"`, `"icons": "image-after-lead-in"`, "icons"},
		{"unimplemented tab arrangement", `"tabs": "left-h2s, then lead-h3s and remaining-h2s as a labelled group"`, `"tabs": "alphabetical"`, "tabs"},
		{"release read from elsewhere", `"from": "CHANGELOG.md"`, `"from": "RELEASES.md"`, "release.from"},
		// A figure on a layout that lifts none would be read and dropped —
		// including its deferred labels-from-page, which would ask the check
		// slice to compare a diagram nothing renders.
		{"figure on a layout that lifts none", `"layout": "prose",`, `"layout": "cards-from-h2",`, "figure"},
		// Deferred keys: no slice reads these yet, and a typo still fails now.
		{"deferred docs path", `"index": "docs/README.md"`, `"index": "/etc/passwd"`, "docs.index"},
		{"deferred contributors path", `"file": "CONTRIBUTING.md"`, `"file": "../outside.md"`, "policy.file"},
		{"deferred contributors heading", `"heading": "Attribution",`, `"heading": "",`, "policy.heading"},
		{"deferred baseline path", `"unresolved_reference_baseline": ".abcd/site-baseline.json"`, `"unresolved_reference_baseline": "../x.json"`, "baseline"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			body := strings.Replace(string(original), c.from, c.to, 1)
			if body == string(original) {
				t.Fatalf("the fixture manifest does not contain %q", c.from)
			}
			if err := os.WriteFile(manifest, []byte(body), 0o644); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { os.WriteFile(manifest, original, 0o644) })

			_, err := LoadManifest(f.Root())
			if err == nil {
				t.Fatalf("the manifest accepted %s", c.name)
			}
			if !strings.Contains(err.Error(), c.says) {
				t.Errorf("the refusal does not name %q: %v", c.says, err)
			}
		})
	}
}

// TestManifestRefusesPageSourcesOutsideTheDocRoots is the composer's half of the
// boundary TestAssetsRefuseAnythingOutsideTheAssetRoot holds for pictures.
//
// A page named by the manifest is INLINED into index.html. `.git/config` — which
// on a CI runner can carry the checkout token in an `http.…extraheader` line —
// is a clean, relative, in-root path, so fsutil.ValidRelPath accepts it and
// loadPage would compose it verbatim into a page served to the public. So would
// `.env`, and so would anything in the gitignored local tier. What may be read
// is therefore a closed set: the documentation the site is composed from, and
// the site's own static input. Twin of the positioning registry's .git refusal
// (iss-150).
func TestManifestRefusesPageSourcesOutsideTheDocRoots(t *testing.T) {
	const (
		heroPage    = `"page": "docs/explanation/rationale.md"`
		chapterPage = `"page": "docs/explanation/roles.md"`
		identity    = `"file": ".abcd/development/brief/01-product/README.md"`
		policy      = `"file": "CONTRIBUTING.md"`
	)
	cases := []struct{ name, from, to, says string }{
		{"hero page inside the git directory", heroPage, `"page": ".git/config"`, ".git/config"},
		{"hero page inside a case-folded git directory", heroPage, `"page": ".GIT/config"`, ".GIT/config"},
		{"hero page at a repository-root dotfile", heroPage, `"page": ".env"`, ".env"},
		{"hero page in the local tier", heroPage, `"page": ".abcd/.work.local/scratch/n.txt"`, ".abcd/.work.local/scratch/n.txt"},
		{"hero page at a root prose file", heroPage, `"page": "CONTRIBUTING.md"`, "CONTRIBUTING.md"},
		{"chapter page inside the git directory", chapterPage, `"page": ".git/config"`, ".git/config"},
		{"chapter page outside the page roots", chapterPage, `"page": ".abcd/work/CONTEXT.md"`, ".abcd/work/CONTEXT.md"},
		// identity.file and policy.file are not whole-file page sources — a
		// heading extract and a one-bullet quote — so they keep their legitimate
		// locations outside docs/. The .git refusal still reaches them: a
		// heading extract out of .git/config is a quote out of .git/config.
		{"identity block inside the git directory", identity, `"file": ".git/config"`, ".git/config"},
		{"contributors policy inside the git directory", policy, `"file": ".git/config"`, ".git/config"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := newFixture(t)
			manifest := filepath.Join(f.Root(), ".abcd", "site.json")
			original, err := os.ReadFile(manifest)
			if err != nil {
				t.Fatal(err)
			}
			body := strings.Replace(string(original), c.from, c.to, 1)
			if body == string(original) {
				t.Fatalf("the fixture manifest does not contain %q", c.from)
			}
			if err := os.WriteFile(manifest, []byte(body), 0o644); err != nil {
				t.Fatal(err)
			}

			_, err = LoadManifest(f.Root())
			if err == nil {
				t.Fatalf("the manifest accepted %s", c.name)
			}
			if !strings.Contains(err.Error(), c.says) {
				t.Errorf("the refusal does not name %q: %v", c.says, err)
			}
		})
	}
}

// TestCommittedManifestLoads is the anti-vacuity guard on the refusals above: a
// gate that refuses everything passes every one of them. This repository's own
// `.abcd/site.json` is the manifest the site build runs on, so it must still
// load — every path it names inside the roots the composer now allows.
func TestCommittedManifestLoads(t *testing.T) {
	m, err := LoadManifest(repoRoot())
	if err != nil {
		t.Fatalf("the committed %s no longer loads: %v", ManifestRelPath, err)
	}
	if m.Home.Hero.Page == "" || len(m.Home.Chapters) == 0 {
		t.Fatalf("the committed manifest loaded empty: %+v", m.Home)
	}
}
