package site

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/changelog"
)

// The header and footer forge links are labelled with the forge's declared
// interface name, not the owner/repo handle — a reader who has never heard of
// the repository still knows where the link goes. A forge no name is declared
// for keeps the handle: the generic fallback the fixture forge exercises.
func TestForgeLabel(t *testing.T) {
	cases := []struct {
		name  string
		repo  string
		names map[string]string
		want  string
	}{
		{"declared forge name wins", "https://github.com/intentdriven/abcd",
			map[string]string{"github.com": "GitHub"}, "GitHub"},
		{"undeclared forge falls back to the handle", "https://example.invalid/fixture/repo",
			map[string]string{"github.com": "GitHub"}, "fixture/repo"},
		{"no map at all falls back to the handle", "https://github.com/intentdriven/abcd",
			nil, "intentdriven/abcd"},
		{"blank declared name falls back to the handle", "https://github.com/intentdriven/abcd",
			map[string]string{"github.com": "  "}, "intentdriven/abcd"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := &composer{}
			c.repo.Repository = tc.repo
			c.ui.ForgeNames = tc.names
			if got := c.forgeLabel(); got != tc.want {
				t.Fatalf("forgeLabel(%q) = %q, want %q", tc.repo, got, tc.want)
			}
		})
	}
}

// profileURL derives a forge profile only from a noreply address — both
// GitHub forms — and derives nothing from a real mailbox, which the export
// rule protects. A bot's bracketed name never matches.
func TestProfileURL(t *testing.T) {
	cases := map[string]string{
		"77722411+REPPL@users.noreply.github.com": "https://github.com/REPPL",
		"REPPL@users.noreply.github.com":          "https://github.com/REPPL",
		"someone@example.com":                     "",
		"":                                        "",
		"49699333+dependabot[bot]@users.noreply.github.com": "",
	}
	for in, want := range cases {
		if got := profileURL(in); got != want {
			t.Fatalf("profileURL(%q) = %q, want %q", in, got, want)
		}
	}
}

// uiStrings walks MAPS as well as fields. The provenance gate holds every
// rendered word to this list, so a family of declared interface strings the
// walk cannot see is refused on the page that renders it — which is how the
// forge names were refused when the walk knew only structs and strings.
func TestUIStringsIncludeMapValues(t *testing.T) {
	ui := UI{ForgeNames: map[string]string{"github.com": "GitHub"}}
	var found bool
	for _, s := range uiStrings(ui) {
		if s == "GitHub" {
			found = true
		}
		if s == "github.com" {
			t.Error("a map KEY reached the allowlist; only the values are rendered")
		}
	}
	if !found {
		t.Error("a declared forge name is not in the interface-string allowlist")
	}
}

// A key declared blank is named, so it cannot pass for a repository that
// simply chose not to declare it.
func TestUIMissingNamesABlankMapValue(t *testing.T) {
	ui := UI{ForgeNames: map[string]string{"github.com": "  "}}
	var named bool
	for _, m := range ui.missing() {
		if m == "forge_names.github.com" {
			named = true
		}
	}
	if !named {
		t.Errorf("a blank forge name went unnamed: %v", ui.missing())
	}
	if len(UI{ForgeNames: map[string]string{}}.missing()) != len(UI{}.missing()) {
		t.Error("an empty map was treated as a missing declaration")
	}
}

// forgeHost strips exactly the scheme and path; a bare or schemeless value
// still yields its host.
func TestForgeHost(t *testing.T) {
	cases := map[string]string{
		"https://github.com/owner/repo": "github.com",
		"http://example.invalid/f/r":    "example.invalid",
		"github.com/owner/repo":         "github.com",
		"":                              "",
	}
	for in, want := range cases {
		if got := forgeHost(in); got != want {
			t.Fatalf("forgeHost(%q) = %q, want %q", in, got, want)
		}
	}
}

// releaseOf stamps the featured record with the release that credits it, and a
// credit is the record's OWN handle: `itd-1990` in a newer section names a
// different record, not a longer spelling of `itd-199`. A substring match walks
// newest-first and returns the first line the id merely sits inside, so every
// short handle inherits the release of the first longer one above it — and a
// superstring landing in a future section silently restamps the featured id.
func TestReleaseOfMatchesTheHandleAtAWordBoundary(t *testing.T) {
	dir := t.TempDir()
	writeSourceFile(t, dir, "CHANGELOG.md", strings.Join([]string{
		"# Changelog",
		"",
		"## [Unreleased]",
		"",
		"## [0.9.0] - 2026-09-01",
		"",
		"### Added",
		"",
		"- A later promise, delivered. (itd-1990)",
		"",
		"## [0.3.0] - 2026-05-01",
		"",
		"### Added",
		"",
		"- The promise this release delivered. (itd-199)",
		"",
	}, "\n"))

	c := &composer{root: mustOpenRoot(t, dir)}
	cases := map[string]string{
		// Its own section, even though a NEWER one spells a superstring of it.
		"itd-199": "0.3.0",
		// The longer handle still finds the line that actually names it.
		"itd-1990": "0.9.0",
		// A handle nothing credits is stamped with nothing, rather than
		// borrowing the section of the first line it is a substring of.
		"itd-19": "",
		"itd-1":  "",
		// A different family sharing the number is a different record.
		"spc-199": "",
	}
	for id, want := range cases {
		if got := c.releaseOf(id); got != want {
			t.Errorf("releaseOf(%q) = %q, want %q", id, got, want)
		}
	}
}

// The anti-vacuity guard: the boundary rule run against the changelog this
// repository actually ships, where the short handles are the ones that
// inherited. Before the fix releaseOf("itd-1") returned 0.7.1 — the release
// that credits itd-130 — and releaseOf("itd-9") returned 0.4.1 off the itd-93
// credit, neither of which names its record at all.
func TestReleaseOfOnTheCommittedChangelog(t *testing.T) {
	c := &composer{root: mustOpenRoot(t, repoRoot())}
	short := c.releaseOf("itd-1")
	if short == "" {
		t.Fatal("the committed changelog credits no itd-1 at all, so this guard proves nothing")
	}
	if long := c.releaseOf("itd-130"); short == long {
		t.Errorf("itd-1 and itd-130 were both stamped %q; the short handle inherited the longer one's section", short)
	}
	if !committedChangelogSectionNames(t, short, "itd-1") {
		t.Errorf("itd-1 was stamped v%s, a release whose section never names itd-1", short)
	}
}

// committedChangelogSectionNames reports whether the dated section for version
// in the committed changelog names id at a word boundary. It splits the file
// on its own rather than through releaseOf, so the guard above is a second
// opinion and not a restatement of the thing under test.
func committedChangelogSectionNames(t *testing.T, version, id string) bool {
	t.Helper()
	re := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(id) + `\b`)
	in := false
	for _, line := range strings.Split(readFile(t, filepath.Join(repoRoot(), "CHANGELOG.md")), "\n") {
		if changelog.IsDatedHeading(line) {
			in = strings.Contains(line, "["+version+"]") || strings.Contains(line, "[v"+version+"]")
			continue
		}
		if in && re.MatchString(line) {
			return true
		}
	}
	return false
}

// writeSourceFile writes one composed source file into a bare directory, for a
// test that reads through a composer's containment root without building the
// whole fixture repository around it.
func writeSourceFile(t *testing.T, dir, rel, body string) {
	t.Helper()
	abs := filepath.Join(dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(abs, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
