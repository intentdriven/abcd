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
		// locations outside docs/, each inside its own closed set
		// (TestManifestRefusesQuoteSourcesOutsideTheirRoots). The .git refusal
		// still reaches them first: a heading extract out of .git/config is a
		// quote out of .git/config.
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

// TestManifestRefusesQuoteSourcesOutsideTheirRoots is the same boundary for the
// QUOTE-type path fields — the ones that select a span of a file rather than
// inlining the whole of it.
//
// They used to pass the relative-path check and then only the .git refusal,
// which is the denylist the page sources were closed against precisely because
// it cannot anticipate the next reachable file: it leaves the gitignored local
// tier, the private record and every file a future contributor adds still
// nameable. The contributors policy is the one that renders worst — policyQuote
// publishes the ENTIRE matched section verbatim whenever `part` is anything but
// first-bullet — so a manifest pointing it at a local scratch file with a
// matching heading published that file to the public site
// (iss-2609090951279243).
func TestManifestRefusesQuoteSourcesOutsideTheirRoots(t *testing.T) {
	const (
		identity  = `"file": ".abcd/development/brief/01-product/README.md"`
		uiStrings = `"ui_strings": "site-src/ui.json"`
		docsIndex = `"index": "docs/README.md"`
		policy    = `"file": "CONTRIBUTING.md"`
		baseline  = `"unresolved_reference_baseline": ".abcd/site-baseline.json"`
	)
	refused := []struct{ name, from, to, says string }{
		// The record's own detector: the policy file outside the roots.
		{"contributors policy in the local tier", policy,
			`"file": ".abcd/.work.local/scratch/notes.md"`, ".abcd/.work.local/scratch/notes.md"},
		{"contributors policy in the working tier", policy,
			`"file": ".abcd/work/CONTEXT.md"`, ".abcd/work/CONTEXT.md"},
		{"contributors policy at a repository-root dotfile", policy, `"file": ".env"`, ".env"},
		{"identity block in the local tier", identity,
			`"file": ".abcd/.work.local/scratch/identity.md"`, ".abcd/.work.local/scratch/identity.md"},
		{"identity block at a repository-root dotfile", identity, `"file": ".env"`, ".env"},
		{"ui strings in the local tier", uiStrings,
			`"ui_strings": ".abcd/.work.local/scratch/ui.json"`, ".abcd/.work.local/scratch/ui.json"},
		{"docs index outside the documentation root", docsIndex,
			`"index": ".abcd/work/CONTEXT.md"`, "docs.index"},
		{"baseline in the local tier", baseline,
			`"unresolved_reference_baseline": ".abcd/.work.local/scratch/b.json"`, ".abcd/.work.local/scratch/b.json"},
	}
	for _, c := range refused {
		t.Run(c.name, func(t *testing.T) {
			f := newFixture(t)
			repointManifest(t, f, c.from, c.to)
			_, err := LoadManifest(f.Root())
			if err == nil {
				t.Fatalf("the manifest accepted %s", c.name)
			}
			if !strings.Contains(err.Error(), c.says) {
				t.Errorf("the refusal does not name %q: %v", c.says, err)
			}
		})
	}

	// The anti-vacuity half, per field: each set is wider than the page roots
	// for a stated reason, and the location that reason names must still load.
	// A gate that refused these would refuse this repository's own manifest.
	accepted := []struct{ name, from, to string }{
		{"the identity block in the durable record", identity,
			`"file": ".abcd/development/brief/01-product/README.md"`},
		{"the contributors policy at the repository root", policy, `"file": "CONTRIBUTING.md"`},
		{"a documentation page as the policy source", policy, `"file": "docs/README.md"`},
		// The baseline is held to the manifest's own directory and no deeper,
		// so a repository may name a baseline it has yet to write.
		{"a baseline beside the manifest", baseline,
			`"unresolved_reference_baseline": ".abcd/other-baseline.json"`},
	}
	for _, c := range accepted {
		t.Run(c.name, func(t *testing.T) {
			f := newFixture(t)
			repointManifest(t, f, c.from, c.to)
			if _, err := LoadManifest(f.Root()); err != nil {
				t.Fatalf("the manifest refused %s: %v", c.name, err)
			}
		})
	}
}

// repointManifest rewrites one declaration in a fixture's committed manifest,
// failing the test if the string it is asked to replace is not there — a case
// that silently matched nothing would assert about the unedited manifest.
// Repointing a declaration at what it already says leaves the file alone: that
// is the case asserting the fixture's own value still loads.
func repointManifest(t *testing.T, f *fixture, from, to string) {
	t.Helper()
	manifest := filepath.Join(f.Root(), ".abcd", "site.json")
	original, err := os.ReadFile(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(original), from) {
		t.Fatalf("the fixture manifest does not contain %q", from)
	}
	if from == to {
		return
	}
	if err := os.WriteFile(manifest, []byte(strings.Replace(string(original), from, to, 1)), 0o644); err != nil {
		t.Fatal(err)
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
