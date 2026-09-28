package site

// The health page's tests, over the same in-process fixture repository the rest
// of the explorer's tests build.
//
// The fixture already carries one of each state the page has to render: a
// dangling supersession (a finding), a handful of records nothing links to
// (a finding), a single author with no forge profile (a family with nothing to
// report), one commit predating the trailer convention (a count), and no commit
// declaring two models (a second family with nothing to report). The two cases
// it does NOT carry — a list long enough to hit the cap, and two author names
// the evidence folds into one — are built on top of it here.

import (
	"encoding/json"
	"strconv"
	"strings"
	"testing"
)

// TestHealthPageRendersEveryFamily is the page's contract: five panels, each
// headed by its own interface label and noted with its own count, and every
// family present whether or not it has anything to report.
func TestHealthPageRendersEveryFamily(t *testing.T) {
	f := newFixture(t)
	out := t.TempDir()
	buildFixture(t, f, out)

	page := outFile(t, out, "record/health/index.html")

	for _, family := range []string{
		"References a record the tree does not hold",
		"Linked to nothing, and nothing links to it",
		"Two author names, one contributor",
		"Authored commits declaring nothing",
		"Commits declaring more than one model",
	} {
		if !strings.Contains(page, family) {
			t.Errorf("the health page renders no panel for %q", family)
		}
	}

	// The page is reachable from the sub-navigation it belongs to, and the
	// dashboard's own strip points at it.
	if !strings.Contains(page, `href="/record/health/" class="on" aria-current="page"`) {
		t.Error("the health tab is not marked current on the health page")
	}
	dash := outFile(t, out, "record/index.html")
	if !strings.Contains(dash, `href="/record/health/"`) {
		t.Error("the dashboard's sub-navigation does not reach the health page")
	}
}

// TestHealthUnresolvedListsTheDanglingReference pins the one finding the
// fixture's record carries: adr-2 supersedes an adr-9 that is not in the tree.
func TestHealthUnresolvedListsTheDanglingReference(t *testing.T) {
	f := newFixture(t)
	out := t.TempDir()
	buildFixture(t, f, out)

	page := outFile(t, out, "record/health/index.html")
	for _, want := range []string{
		`<a href="/record/adr/adr-2/">adr-2</a>`,
		`<b class="stub">adr-9</b>`,
		"not in the tree",
	} {
		if !strings.Contains(page, want) {
			t.Errorf("the unresolved panel does not carry %q", want)
		}
	}
}

// TestHealthCleanFamiliesSaySo is why an empty family is rendered at all: a
// reader must be able to tell a check that found nothing from a check that
// never ran. The fixture has one author with no forge profile and no commit
// declaring two models, so both of those families are clean.
func TestHealthCleanFamiliesSaySo(t *testing.T) {
	f := newFixture(t)
	out := t.TempDir()
	buildFixture(t, f, out)

	page := outFile(t, out, "record/health/index.html")
	// The identity-candidate panel: headed, noted zero, and saying so.
	if !strings.Contains(page, `<summary><h3>Two author names, one contributor<span>0</span></h3></summary>`+
		`<div class="health"><div class="hsum">Nothing to report</div></div>`) {
		t.Error("the identity-candidate family does not render its clean state")
	}
	// The multi-trailer panel is clean AND says it is not a fault either way.
	if !strings.Contains(page, "Not a fault: it is why the trailer count and the commit count differ") {
		t.Error("the multi-trailer panel does not say that the family is not a fault")
	}
	if strings.Count(page, "Nothing to report") != 2 {
		t.Errorf("the fixture has exactly two clean families; the page says %d",
			strings.Count(page, "Nothing to report"))
	}
}

// TestHealthUndeclaredRendersTheCount pins the family that is a number rather
// than a list: the fixture's first commit predates its own trailer convention.
func TestHealthUndeclaredRendersTheCount(t *testing.T) {
	f := newFixture(t)
	out := t.TempDir()
	buildFixture(t, f, out)

	page := outFile(t, out, "record/health/index.html")
	if !strings.Contains(page, `Authored commits declaring nothing<span>1</span>`) {
		t.Error("the undeclared panel does not note the fixture's one undeclared commit")
	}
	// The number and its word are separate elements, never one composed phrase.
	if !strings.Contains(page, `<b class="tnum">1</b> no declaration`) {
		t.Error("the undeclared count is not emitted in its own element")
	}
	// The merges the count sets aside are stated, not assumed.
	if !strings.Contains(page, "merge commits excluded") {
		t.Error("the undeclared panel does not say what it excluded")
	}
}

// TestHealthIsolatedCapStatesTheTrueTotal is the promise the cap makes: the
// panel note is the REAL number of isolated records, the list is bounded, and
// the entries it did not draw are counted rather than silently dropped.
//
// The fixture is grown past the cap here with frontmatter-free principle files,
// which is the cheapest record the format has: a heading and a body, no typed
// links in either direction, so every one of them is isolated by construction.
func TestHealthIsolatedCapStatesTheTrueTotal(t *testing.T) {
	f := newFixture(t)

	// Enough extra isolated records to overrun the cap on their own.
	const extra = 25
	for i := 0; i < extra; i++ {
		name := string(rune('a'+i%26)) + "-extra-fixture-principle-" + strconv.Itoa(i)
		f.write(".abcd/development/principles/"+name+".md",
			"# "+name+"\n\n**The rule.** One more record nothing links to.\n")
	}
	f.commitAt("2026-03-07T09:00:00+00:00", "docs: more principles", "None")

	out := t.TempDir()
	buildFixture(t, f, out)
	page := outFile(t, out, "record/health/index.html")

	// The truth, taken from the export the page was rendered from rather than
	// from a number written into this test.
	// Reached is reached: a typed reference OR a body mention. The page asks
	// the same question the chart's bubble size asks, so this test asks it too
	// rather than counting typed edges alone and disagreeing with the page.
	export := decodeExport(t, out)
	total := 0
	for _, n := range export.Nodes {
		linked := false
		for _, ed := range export.Edges {
			if ed.From == n.ID || ed.To == n.ID {
				linked = true
				break
			}
		}
		for _, m := range export.Mentions {
			if m.From == n.ID || m.To == n.ID {
				linked = true
				break
			}
		}
		if !linked {
			total++
		}
	}
	if total <= isolatedListCap {
		t.Fatalf("the fixture must overrun the cap for this test to mean anything: %d isolated", total)
	}

	if want := "<summary><h3>Linked to nothing, and nothing links to it<span>" + strconv.Itoa(total) + "</span>"; !strings.Contains(page, want) {
		t.Errorf("the isolated panel does not note the true total %d", total)
	}
	// The list itself is bounded, and the remainder is stated.
	drawn := strings.Count(sliceBetween(page, "<summary><h3>Linked to nothing", "</details>"), "<li>")
	if drawn != isolatedListCap {
		t.Errorf("the isolated list drew %d entries; the cap is %d", drawn, isolatedListCap)
	}
	if want := `<b class="tnum">` + strconv.Itoa(total-isolatedListCap) + `</b> more`; !strings.Contains(page, want) {
		t.Errorf("the isolated list does not count what it did not draw: want %q", want)
	}
}

// TestHealthSameAuthorSuggestsTheMailmapLine is the identity family with
// something to report. Two commit identities resolving to one forge profile are
// raised as a CANDIDATE, with the exact `.mailmap` line that would fold them —
// and that line names a forge noreply address only, because the export withholds
// a contributor's real address on purpose.
func TestHealthSameAuthorSuggestsTheMailmapLine(t *testing.T) {
	f := newFixture(t)

	// Two names, one forge profile. The invented username is the second name,
	// so this pair trips BOTH signals at once: the identical derived profile,
	// and one author's name standing as the other's profile username.
	const canonical = "Fixture Alias <77+fixturealias@users.noreply.github.com>"
	const alias = "fixturealias <fixturealias@users.noreply.github.com>"
	f.commitBy("2026-03-08T09:00:00+00:00", "docs: one", canonical)
	f.commitBy("2026-03-08T10:00:00+00:00", "docs: two", canonical)
	f.commitBy("2026-03-08T11:00:00+00:00", "docs: three", alias)

	out := t.TempDir()
	buildFixture(t, f, out)
	page := outFile(t, out, "record/health/index.html")

	if !strings.Contains(page, "Two author names, one contributor<span>1</span>") {
		t.Error("the identity family does not raise the fixture's one candidate")
	}
	// The pair, each side linked to the profile the evidence derived.
	for _, want := range []string{
		`<a href="https://github.com/fixturealias">Fixture Alias</a>`,
		`<a href="https://github.com/fixturealias">fixturealias</a>`,
	} {
		if !strings.Contains(page, want) {
			t.Errorf("the candidate pair does not carry %q", want)
		}
	}
	// The suggestion: the canonical name against the alias's commit address.
	want := `Suggested <code>Fixture Alias &lt;fixturealias@users.noreply.github.com&gt;</code>`
	if !strings.Contains(page, want) {
		t.Errorf("the candidate does not propose the mailmap line: want %q", want)
	}
}

// TestSameContributorReadsBothSignals covers the pairing rule directly, so the
// two signals are pinned apart from everything the build does around them.
func TestSameContributorReadsBothSignals(t *testing.T) {
	profiled := Author{Name: "A Contributor", Profile: "https://github.com/acontributor"}
	sameProfile := Author{Name: "acontributor2", Profile: "https://github.com/acontributor"}
	byName := Author{Name: "ACONTRIBUTOR"}
	unrelated := Author{Name: "Somebody Else", Profile: "https://github.com/somebodyelse"}

	for _, tc := range []struct {
		name string
		a, b Author
		want bool
	}{
		{"the same derived profile", profiled, sameProfile, true},
		{"a name that is the other's username", profiled, byName, true},
		{"two different profiles", profiled, unrelated, false},
		{"two names and no profile at all", byName, Author{Name: "Nobody"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := sameContributor(tc.a, tc.b); got != tc.want {
				t.Errorf("sameContributor = %v, want %v", got, tc.want)
			}
			if got := sameContributor(tc.b, tc.a); got != tc.want {
				t.Errorf("sameContributor (reversed) = %v, want %v", got, tc.want)
			}
		})
	}
}

// --- helpers ---------------------------------------------------------------

// commitBy makes one empty commit attributed to a named author, which is how a
// fixture grows a second identity without a second working tree.
func (f *fixture) commitBy(date, subject, author string) {
	f.t.Helper()
	f.git(date, "commit", "--allow-empty", "--author="+author, "-m", subject+"\n\nAssisted-by: None")
}

// decodeExport reads the record.json a build wrote.
func decodeExport(t *testing.T, out string) RecordExport {
	t.Helper()
	var export RecordExport
	if err := json.Unmarshal([]byte(outFile(t, out, "record.json")), &export); err != nil {
		t.Fatal(err)
	}
	return export
}

// sliceBetween is the part of a page between two markers, or "" where either is
// absent. It keeps a count of list items scoped to one panel.
func sliceBetween(s, from, to string) string {
	i := strings.Index(s, from)
	if i < 0 {
		return ""
	}
	rest := s[i:]
	j := strings.Index(rest, to)
	if j < 0 {
		return ""
	}
	return rest[:j]
}

// TestHealthCarriesTheDisclosureRate pins where the disclosure rate lives. The
// contributors page carries two things only — the authors of record and the
// Assisted-by trailers — and the share of authored commits that disclose AI
// assistance sits with the other findings, on the health page's row of counts,
// as COMMITS that disclose over commits a person wrote, with the merges set
// aside stated (iss-2608231008315498).
func TestHealthCarriesTheDisclosureRate(t *testing.T) {
	f := newFixture(t)
	out := t.TempDir()
	buildFixture(t, f, out)

	a := decodeExport(t, out).Authorship
	if a.Authored == 0 || a.AssistedCommits == 0 || a.Merges == 0 {
		t.Fatalf("the fixture must carry authored, assisted and merge commits: %+v", a)
	}
	share := strconv.Itoa(a.AssistedCommits*100/a.Authored) + "%"
	want := `<span class="n">` + share + `</span><span class="l">of authored commits disclose AI assistance</span>` +
		`<span class="s">` + strconv.Itoa(a.AssistedCommits) + ` / ` + strconv.Itoa(a.Authored) + `</span>` +
		`<span class="s"><b class="tnum">` + strconv.Itoa(a.Merges) + `</b> merge commits excluded</span>`

	health := outFile(t, out, "record/health/index.html")
	if !strings.Contains(health, want) {
		t.Errorf("the health page does not carry the disclosure rate %s", want)
	}
	contributors := outFile(t, out, "contributors/index.html")
	if strings.Contains(contributors, "disclose AI assistance") {
		t.Error("the contributors page still carries the disclosure rate")
	}
	if n := strings.Count(contributors, `class="panel fold c12"`); n != 2 {
		t.Errorf("the contributors page carries %d folded full-width panels, want the two: authors and trailers", n)
	}
	if strings.Contains(contributors, `class="tile`) {
		t.Error("the contributors page carries a stat tile")
	}
}
