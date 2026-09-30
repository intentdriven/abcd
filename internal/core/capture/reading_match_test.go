package capture

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/issueschema"
	"github.com/intentdriven/abcd/internal/core/record/match"
)

// reading_match_test.go proves ruling DQ2b (2026-09-30) on the reading
// family: a reading finding is matched when it is stored, its likely repeats
// are written onto it as `duplicates:` / `refines:` links and reported, and
// the promote step that mints an intent draft from it matches again.

// A finding the reading returns in its own words, doubling plantedFinding.
var readingDouble = ReadingItem{
	Pattern: "ledger reader skips records with duplicated keys",
	Body: map[string]string{
		"tension":            plantedDouble,
		"constraint_in_play": "every finding appears in every listing of the capture ledger",
		"why_a_tension":      "a record whose frontmatter carries a duplicated key disappears from every listing silently",
	},
}

// An unrelated finding, so a run has a second item and the corpus has other
// words to weigh against.
var readingOther = ReadingItem{
	Pattern: "site builder anchors go stale",
	Body: map[string]string{
		"tension":            "The site builder renders a stale anchor for a heading renamed since the last build.",
		"constraint_in_play": "every rendered anchor resolves to a heading on the current page",
		"why_a_tension":      "a renamed heading leaves the anchor pointing at nothing after the rebuild",
	},
}

func ingestDetection(t *testing.T, repo, ir, run string, m *match.Config, items ...ReadingItem) IngestReadingResult {
	t.Helper()
	res, err := IngestReading(IngestReadingRequest{
		RepoRoot: repo, IssuesRoot: ir,
		Run: run, Manifest: "sha256:" + strings.Repeat("a", 64),
		Position: "detection", Regime: issueschema.ReadingRegime("detection"),
		Items: items, Match: m,
	})
	if err != nil {
		t.Fatalf("IngestReading: %v", err)
	}
	if len(res.Records) != len(items) {
		t.Fatalf("IngestReading wrote %d records, want %d", len(res.Records), len(items))
	}
	return res
}

// matchOf returns the outcome the ingest reported for record id.
func matchOf(t *testing.T, res IngestReadingResult, id string) *match.Outcome {
	t.Helper()
	for _, m := range res.Matches {
		if m.ID == id {
			return m.Match
		}
	}
	t.Fatalf("the ingest reported no match for %s: %+v", id, res.Matches)
	return nil
}

// (a) and (b): a stored reading finding that doubles an open issue carries the
// typed link naming it, the ingest reports the match, and the linked record is
// still a record the family's own validator reads.
func TestReadingIngestLinksAFindingAnOpenRecordHolds(t *testing.T) {
	repo, ir := ledger(t)
	held := captureText(t, repo, ir, plantedFinding, nil)
	captureText(t, repo, ir, matchFiller1, nil)
	captureText(t, repo, ir, matchFiller2, nil)

	res := ingestDetection(t, repo, ir, "rdg-2609300000000001", bundled(), readingDouble)
	o := matchOf(t, res, res.Records[0].ID)
	if o == nil || len(o.Matches) == 0 {
		t.Fatalf("no match reported: %+v", o)
	}
	m := o.Matches[0]
	if m.ID != held.ID || !m.Linked || (m.Relation != match.Duplicates && m.Relation != match.Refines) {
		t.Fatalf("match = %+v, want %s linked", m, held.ID)
	}
	content := readRecord(t, repo, res.Records[0].Path)
	if !strings.Contains(content, "\n"+string(m.Relation)+": ["+held.ID+"]\n") {
		t.Fatalf("the reading record carries no %s link naming %s:\n%s", m.Relation, held.ID, content)
	}
	if _, err := ValidateReadingRecord(content); err != nil {
		t.Fatalf("the linked reading record does not validate: %v", err)
	}
}

// The reversal itself: a later reading that returns the same finding again is
// linked to the earlier reading's item, mechanically, at storing time.
func TestReadingIngestLinksARecurrenceOfAnEarlierReadingItem(t *testing.T) {
	repo, ir := ledger(t)
	captureText(t, repo, ir, matchFiller1, nil)
	captureText(t, repo, ir, matchFiller2, nil)
	first := ingestDetection(t, repo, ir, "rdg-2609300000000001", nil, readingDouble, readingOther)

	again := ingestDetection(t, repo, ir, "rdg-2609300000000002", bundled(), readingDouble)
	o := matchOf(t, again, again.Records[0].ID)
	if o == nil || len(o.Matches) == 0 || o.Matches[0].ID != first.Records[0].ID ||
		o.Matches[0].Relation != match.Duplicates || !o.Matches[0].Linked {
		t.Fatalf("the recurrence was not linked to %s: %+v", first.Records[0].ID, o)
	}
	content := readRecord(t, repo, again.Records[0].Path)
	if !strings.Contains(content, "\nduplicates: ["+first.Records[0].ID+"]\n") {
		t.Fatalf("no duplicates link on the recurrence:\n%s", content)
	}
}

// Two findings of one reading are two findings (the consistency pass's rule):
// the items one ingest files are never candidates for each other.
func TestReadingIngestNeverLinksTwoFindingsOfOneRun(t *testing.T) {
	repo, ir := ledger(t)
	captureText(t, repo, ir, matchFiller1, nil)
	captureText(t, repo, ir, matchFiller2, nil)
	twin := readingDouble
	twin.Pattern = "the ledger reader skips records carrying duplicated keys"
	res := ingestDetection(t, repo, ir, "rdg-2609300000000001", bundled(), readingDouble, twin)
	for _, r := range res.Records {
		o := matchOf(t, res, r.ID)
		for _, m := range o.Matches {
			for _, same := range res.Records {
				if m.ID == same.ID {
					t.Fatalf("%s was matched against %s, filed by the same ingest: %+v", r.ID, same.ID, o)
				}
			}
		}
		if c := readRecord(t, repo, r.Path); strings.Contains(c, "duplicates:") || strings.Contains(c, "refines:") {
			t.Fatalf("%s carries a link:\n%s", r.ID, c)
		}
	}
}

// Without a match configuration the ingest files unmatched and reports none,
// so every existing caller keeps its behaviour.
func TestReadingIngestWithoutAMatchReportsNone(t *testing.T) {
	repo, ir := ledger(t)
	captureText(t, repo, ir, plantedFinding, nil)
	res := ingestDetection(t, repo, ir, "rdg-2609300000000001", nil, readingDouble)
	if len(res.Matches) != 0 {
		t.Fatalf("an unmatched ingest reported matches: %+v", res.Matches)
	}
	if c := readRecord(t, repo, res.Records[0].Path); strings.Contains(c, "duplicates:") {
		t.Fatalf("an unmatched ingest wrote a link:\n%s", c)
	}
}

// The schema widening is exactly the two typed links, each a list of record
// ids the match can name; anything else stays refused.
func TestReadingRecordTypedLinksAreRecordIDs(t *testing.T) {
	base := "---\nschema_version: 1\nid: rdi-2609300000000001\nrun: rdg-2609300000000001\n" +
		"manifest: sha256:beef\nposition: detection\nregime: registrative\npattern: p\n" +
		"tension: t\nconstraint_in_play: c\nwhy_a_tension: w\n"
	for _, ok := range []string{"duplicates: [iss-1]", "refines: [itd-2]", "duplicates: [rdi-2609300000000002]"} {
		if _, err := ValidateReadingRecord(base + ok + "\n---\n"); err != nil {
			t.Errorf("%q refused: %v", ok, err)
		}
	}
	for _, bad := range []string{"duplicates: [dsp-1]", "refines: iss-1", "supersedes: [rdi-1]"} {
		if _, err := ValidateReadingRecord(base + bad + "\n---\n"); !errors.Is(err, ErrMalformedFrontmatter) {
			t.Errorf("%q not refused as malformed: %v", bad, err)
		}
	}
}

// (c): promoting an accepted reading item mints a draft that is matched again,
// on the finding's own words, and linked as capture links.
func TestPromoteReadingItemMatchesTheDraft(t *testing.T) {
	repo, ir := ledger(t)
	held := captureText(t, repo, ir, plantedFinding, nil)
	captureText(t, repo, ir, matchFiller1, nil)
	captureText(t, repo, ir, matchFiller2, nil)
	res := ingestDetection(t, repo, ir, "rdg-2609300000000001", nil, readingDouble)
	item := res.Records[0].ID
	if _, err := Disposition(DispositionRequest{
		RepoRoot: repo, IssuesRoot: ir, Item: item,
		State: issueschema.DispositionAccepted, Grounds: "the tension is real and worth acting on",
	}); err != nil {
		t.Fatal(err)
	}

	p, err := Promote(PromoteRequest{RepoRoot: repo, IssuesRoot: ir, ID: item, Match: bundled()})
	if err != nil {
		t.Fatalf("Promote: %v", err)
	}
	if p.Match == nil || len(p.Match.Matches) == 0 || p.Match.Matches[0].ID != held.ID || !p.Match.Matches[0].Linked {
		t.Fatalf("the promoted draft was not matched to %s: %+v", held.ID, p.Match)
	}
	rel := string(p.Match.Matches[0].Relation)
	draft, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(p.IntentPath)))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(draft), "\n"+rel+": ["+held.ID+"]\n") {
		t.Fatalf("the draft carries no %s link naming %s:\n%s", rel, held.ID, draft)
	}
}

// A promote without a match configuration mints as before and reports none.
func TestPromoteReadingItemWithoutAMatchReportsNone(t *testing.T) {
	repo, ir, item := dispositionedReadingFixture(t)
	p, err := Promote(PromoteRequest{RepoRoot: repo, IssuesRoot: ir, ID: item})
	if err != nil {
		t.Fatal(err)
	}
	if p.Match != nil {
		t.Fatalf("an unmatched promote reported a match: %+v", p.Match)
	}
}
