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

// auditIsMet reads the intent's `## Audit Notes` rollup — the audit's own
// machine-readable verdict — and nothing else. Two things it used to read
// besides: a rollup line quoted inside a fenced block, which is an example of
// the shape rather than a verdict about this intent, and a negative count,
// which no audit writes and which lets one rollup line cancel another. Either
// one alone can put an intent whose acceptance is NOT met on the homepage as
// the record's own evidence that the process works.
func TestAuditIsMetReadsOnlyTheUnfencedAuditNotes(t *testing.T) {
	const fence = "```"
	cases := []struct {
		name string
		doc  string
		want bool
	}{
		{"a met rollup features", auditedIntent(strings.Join([]string{
			"## Audit Notes",
			"",
			"Acceptance rollup: MET 1 · MET_WITH_CONCERNS 0 · NOT_MET 0 · INCONCLUSIVE 0",
		}, "\n")), true},

		{"a not-met rollup does not", auditedIntent(strings.Join([]string{
			"## Audit Notes",
			"",
			"Acceptance rollup: MET 2 · MET_WITH_CONCERNS 0 · NOT_MET 1 · INCONCLUSIVE 0",
		}, "\n")), false},

		{"a fenced negative cannot cancel a real NOT_MET", auditedIntent(strings.Join([]string{
			"## Audit Notes",
			"",
			"Acceptance rollup: MET 2 · MET_WITH_CONCERNS 0 · NOT_MET 1 · INCONCLUSIVE 0",
			"",
			"The line the auditor writes, for reference:",
			"",
			fence,
			"Acceptance rollup: MET 0 · MET_WITH_CONCERNS 0 · NOT_MET -1 · INCONCLUSIVE 0",
			fence,
		}, "\n")), false},

		// The half a negative-refusal alone does not reach: no negative
		// anywhere, and the fenced line is additive rather than cancelling.
		{"a fenced MET cannot lift a concerns-only rollup", auditedIntent(strings.Join([]string{
			"## Audit Notes",
			"",
			"Acceptance rollup: MET 0 · MET_WITH_CONCERNS 3 · NOT_MET 0 · INCONCLUSIVE 0",
			"",
			"The template this was filled in from:",
			"",
			fence,
			"Acceptance rollup: MET 1",
			fence,
		}, "\n")), false},

		// The half fence-awareness alone does not reach: both lines are real
		// Audit Notes prose, and the second cancels the first.
		{"a negative count cancels nothing", auditedIntent(strings.Join([]string{
			"## Audit Notes",
			"",
			"Acceptance rollup: MET 1 · MET_WITH_CONCERNS 0 · NOT_MET 1 · INCONCLUSIVE 0",
			"",
			"A correction nobody should be able to write:",
			"",
			"Acceptance rollup: MET 0 · MET_WITH_CONCERNS 0 · NOT_MET -1 · INCONCLUSIVE 0",
		}, "\n")), false},

		// The half fence-awareness and the negative refusal both miss: two
		// sections with the SAME title, the honest one concerns-only and the
		// second reading MET. Accumulating across every matching section lifts
		// the intent onto the homepage on the strength of the duplicate, which
		// is no harder to write than the fenced line was
		// (iss-2609090951277880).
		{"a second Audit Notes section cannot lift a concerns-only rollup", auditedIntent(strings.Join([]string{
			"## Audit Notes",
			"",
			"Acceptance rollup: MET 0 · MET_WITH_CONCERNS 3 · NOT_MET 0 · INCONCLUSIVE 0",
			"",
			"## Audit Notes",
			"",
			"Acceptance rollup: MET 1 · MET_WITH_CONCERNS 0 · NOT_MET 0 · INCONCLUSIVE 0",
		}, "\n")), false},

		// At ANY heading level: the section walk reads a title, not a depth.
		{"a deeper second Audit Notes section cannot lift it either", auditedIntent(strings.Join([]string{
			"## Audit Notes",
			"",
			"Acceptance rollup: MET 0 · MET_WITH_CONCERNS 3 · NOT_MET 0 · INCONCLUSIVE 0",
			"",
			"### Audit Notes",
			"",
			"Acceptance rollup: MET 1 · MET_WITH_CONCERNS 0 · NOT_MET 0 · INCONCLUSIVE 0",
		}, "\n")), false},

		// A document with two of them is malformed whichever way they read: an
		// intent has one audit, so a second section of that title is not a
		// second verdict but evidence that this file is not what it claims. The
		// negative count is refused on the same ground.
		{"a duplicate Audit Notes section is malformed even when both pass", auditedIntent(strings.Join([]string{
			"## Audit Notes",
			"",
			"Acceptance rollup: MET 1 · MET_WITH_CONCERNS 0 · NOT_MET 0 · INCONCLUSIVE 0",
			"",
			"## Audit Notes",
			"",
			"Acceptance rollup: MET 2 · MET_WITH_CONCERNS 0 · NOT_MET 0 · INCONCLUSIVE 0",
		}, "\n")), false},

		// And malformed at any DEPTH, which is the case that separates refusing
		// a duplicate from merely ignoring one: read-the-first alone would
		// answer true here off the honest section above.
		{"a duplicate at another heading level is malformed too", auditedIntent(strings.Join([]string{
			"## Audit Notes",
			"",
			"Acceptance rollup: MET 1 · MET_WITH_CONCERNS 0 · NOT_MET 0 · INCONCLUSIVE 0",
			"",
			"### Audit Notes",
			"",
			"Acceptance rollup: MET 2 · MET_WITH_CONCERNS 0 · NOT_MET 0 · INCONCLUSIVE 0",
		}, "\n")), false},

		// The anti-vacuity half of the rule above, in the same table: a title
		// that merely CONTAINS the words is a different section, and the one
		// honest rollup still features.
		{"a similarly titled section is not a duplicate", auditedIntent(strings.Join([]string{
			"## Audit Notes",
			"",
			"Acceptance rollup: MET 1 · MET_WITH_CONCERNS 0 · NOT_MET 0 · INCONCLUSIVE 0",
			"",
			"## Audit Notes for the reader",
			"",
			"Acceptance rollup: MET 0 · MET_WITH_CONCERNS 0 · NOT_MET 1 · INCONCLUSIVE 0",
		}, "\n")), true},

		{"a rollup in another section is not the audit", auditedIntent(strings.Join([]string{
			"## Notes for the auditor",
			"",
			"Acceptance rollup: MET 1 · MET_WITH_CONCERNS 0 · NOT_MET 0 · INCONCLUSIVE 0",
		}, "\n")), false},

		{"a rollup in the frontmatter is not the audit", strings.Join([]string{
			"---",
			"id: itd-7",
			"slug: an-audited-intent",
			"note: \"Acceptance rollup: MET 1 · MET_WITH_CONCERNS 0 · NOT_MET 0 · INCONCLUSIVE 0\"",
			"---",
			"",
			"# An Audited Intent",
			"",
			"## Audit Notes",
			"",
			"The audit has not run yet.",
			"",
		}, "\n"), false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			writeSourceFile(t, dir, "itd-7.md", tc.doc)
			c := &composer{root: mustOpenRoot(t, dir)}
			if got := c.auditIsMet("itd-7.md"); got != tc.want {
				t.Errorf("auditIsMet = %v, want %v, for:\n%s", got, tc.want, tc.doc)
			}
		})
	}
}

// auditedIntent wraps an Audit Notes section (or whatever stands in for one) in
// the rest of a shipped intent, so each case differs only in the part the
// rollup scan reads.
func auditedIntent(tail string) string {
	return strings.Join([]string{
		"---",
		"id: itd-7",
		"slug: an-audited-intent",
		"---",
		"",
		"# An Audited Intent",
		"",
		"## Press Release",
		"",
		"> **Somebody wrote this.** It is prose, not the mint placeholder.",
		"",
		"## Acceptance Criteria",
		"",
		"- Given the audited intent, when the audit runs, then it writes a rollup.",
		"",
		tail,
		"",
	}, "\n")
}

// The anti-vacuity guard: a rollup scan narrowed until it reads nothing would
// satisfy every refusal above and leave the homepage with no feature block at
// all, because there would be no shipped intent left whose audit reads MET.
// This repository's own record must still supply them.
func TestAuditIsMetOnTheCommittedIntents(t *testing.T) {
	const shipped = ".abcd/development/intents/shipped"
	entries, err := os.ReadDir(filepath.Join(repoRoot(), filepath.FromSlash(shipped)))
	if err != nil {
		t.Fatal(err)
	}
	c := &composer{root: mustOpenRoot(t, repoRoot())}
	var met []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		if c.auditIsMet(shipped + "/" + e.Name()) {
			met = append(met, e.Name())
		}
	}
	if len(met) == 0 {
		t.Fatalf("no shipped intent under %s reads MET; the feature block has nothing to quote", shipped)
	}
	t.Logf("%d of %d shipped intents read MET:\n%s", len(met), len(entries), strings.Join(met, "\n"))
}
