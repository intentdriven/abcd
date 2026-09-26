package intent

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// owed_test.go pins the one reader of the review marker across shipped/
// (itd-2609150819445595, spc-2609202112205096 piece 1): one shipped intent per
// marker state, plus one whose Audit Notes carry two markers, where the first
// marker wins because existingMarker is the authority the emit already uses.

const (
	owedRcp      = "rcp-0000000000a1"
	ingestedRcp  = "rcp-0000000000b2"
	deadRcp      = "rcp-0000000000c3"
	firstDupRcp  = "rcp-0000000000d4"
	secondDupRcp = "rcp-0000000000e5"
	deadReason   = "verdict names criterion ac-9, which the intent does not carry"
)

// shippedWithNotes is a shipped intent whose Audit Notes hold notes verbatim.
func shippedWithNotes(id, slug, notes string) string {
	return "---\nid: " + id + "\nslug: " + slug + "\nspec_id: spc-1\nkind: standalone\nimpact: fix\n---\n" +
		"# " + slug + "\n\n## Acceptance Criteria\n\n- ok\n\n## Audit Notes\n\n" + notes + "\n"
}

// seedReviewStates writes one shipped intent per marker state, the duplicated
// marker, and a planned intent carrying an OWED marker (which is not shipped, so
// the reader never lists it).
func seedReviewStates(t *testing.T, root string) {
	t.Helper()
	writeFile(t, root, shippedDir+"/itd-11-owed.md", shippedWithNotes("itd-11", "owed", owedBlock(owedRcp)))
	writeFile(t, root, shippedDir+"/itd-12-ingested.md", shippedWithNotes("itd-12", "ingested",
		"<!-- abcd-review: INGESTED receipt="+ingestedRcp+" -->\nFidelity review — receipt "+ingestedRcp+" (verifier x y)."))
	writeFile(t, root, shippedDir+"/itd-13-dead.md", shippedWithNotes("itd-13", "dead",
		deadLetterBlock(deadRcp, deadReason, reviewsRelDir+"/"+deadRcp+".deadletter.json", nil, func(s string) string { return oneLine(s) })))
	writeFile(t, root, shippedDir+"/itd-14-bare.md", shippedWithNotes("itd-14", "bare", "_Empty. Populated by intent-auditor when intent moves to shipped/._"))
	writeFile(t, root, shippedDir+"/itd-15-dup.md", shippedWithNotes("itd-15", "dup",
		owedBlock(firstDupRcp)+"\n\n<!-- abcd-review: INGESTED receipt="+secondDupRcp+" -->\nFidelity review — receipt "+secondDupRcp+"."))
	writeFile(t, root, plannedDir+"/itd-16-planned.md", shippedWithNotes("itd-16", "planned", owedBlock("rcp-0000000000f6")))
}

func reviewsByID(t *testing.T, l ReviewListing) map[string]ReviewEntry {
	t.Helper()
	m := map[string]ReviewEntry{}
	for _, e := range l.Entries {
		if _, dup := m[e.IntentID]; dup {
			t.Fatalf("%s listed twice: %+v", e.IntentID, l.Entries)
		}
		m[e.IntentID] = e
	}
	return m
}

func TestReviewsReadsEveryShippedMarker(t *testing.T) {
	root := t.TempDir()
	seedReviewStates(t, root)

	l, err := Reviews(root)
	if err != nil {
		t.Fatal(err)
	}
	got := reviewsByID(t, l)
	if len(got) != 5 {
		t.Fatalf("want one entry per shipped intent (5), got %d: %+v", len(got), l.Entries)
	}
	if _, ok := got["itd-16"]; ok {
		t.Fatalf("a planned intent is not shipped and must not be listed: %+v", got["itd-16"])
	}

	cases := []struct {
		id, state, receipt, reEmit string
	}{
		{"itd-11", ReviewOwed, owedRcp, "abcd intent audit itd-11"},
		{"itd-12", ReviewIngested, ingestedRcp, ""},
		{"itd-13", ReviewDeadLetter, deadRcp, ""},
		{"itd-14", ReviewNone, "", "abcd intent audit itd-14"},
		// The first marker wins: the duplicated intent reads OWED with the first
		// receipt, never INGESTED with the second.
		{"itd-15", ReviewOwed, firstDupRcp, "abcd intent audit itd-15"},
	}
	for _, c := range cases {
		e := got[c.id]
		if e.State != c.state || e.ReceiptID != c.receipt || e.ReEmit != c.reEmit {
			t.Errorf("%s = %+v, want state %q receipt %q re_emit %q", c.id, e, c.state, c.receipt, c.reEmit)
		}
	}
	if r := got["itd-13"].Reason; r != deadReason {
		t.Errorf("dead-letter reason = %q, want %q", r, deadReason)
	}
	for _, id := range []string{"itd-11", "itd-12", "itd-14", "itd-15"} {
		if got[id].Reason != "" {
			t.Errorf("%s carries a reason but is not dead-lettered: %+v", id, got[id])
		}
	}
	if !got["itd-11"].IsOwed() || !got["itd-14"].IsOwed() || got["itd-12"].IsOwed() || got["itd-13"].IsOwed() {
		t.Errorf("owed set is OWED plus none: %+v", l.Entries)
	}

	// OWED plus none; DEAD_LETTER is counted apart and never as owed.
	if l.Owed != 3 || l.DeadLettered != 1 || l.Ingested != 1 {
		t.Fatalf("counts owed=%d dead_lettered=%d ingested=%d, want 3/1/1", l.Owed, l.DeadLettered, l.Ingested)
	}
}

// TestReviewsCarriesNoLocalTierPath: the dead-letter block names where its raw
// payload is retained, under the gitignored local tier; the listing reports the
// reason and never that path.
func TestReviewsCarriesNoLocalTierPath(t *testing.T) {
	root := t.TempDir()
	seedReviewStates(t, root)
	l, err := Reviews(root)
	if err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(l)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), ".work.local") || strings.Contains(string(b), "request.md") {
		t.Fatalf("listing carries a local-tier path:\n%s", b)
	}
}

// TestReviewsNeverWrites: the reader reads; it never re-emits a request or
// rewrites a record (decision 2).
func TestReviewsNeverWrites(t *testing.T) {
	root := t.TempDir()
	seedReviewStates(t, root)
	before := snapshotTree(t, root)
	if _, err := Reviews(root); err != nil {
		t.Fatal(err)
	}
	if _, err := Status(root); err != nil {
		t.Fatal(err)
	}
	if after := snapshotTree(t, root); after != before {
		t.Fatalf("the reader wrote:\nbefore:\n%s\nafter:\n%s", before, after)
	}
}

// TestStatusCountsOwedReviews: the lifecycle board carries the owed count, and it
// is the listing's owed total, from the same reader.
func TestStatusCountsOwedReviews(t *testing.T) {
	root := t.TempDir()
	seedReviewStates(t, root)
	v, err := Status(root)
	if err != nil {
		t.Fatal(err)
	}
	l, err := Reviews(root)
	if err != nil {
		t.Fatal(err)
	}
	if v.ReviewsOwed != l.Owed || v.ReviewsOwed != 3 {
		t.Fatalf("board owed = %d, listing owed = %d, want both 3", v.ReviewsOwed, l.Owed)
	}
}

// TestReviewOfOneIntent is the single-intent read the record dispatcher calls:
// the same reader, one record.
func TestReviewOfOneIntent(t *testing.T) {
	root := t.TempDir()
	seedReviewStates(t, root)
	corpus, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	it, ok := corpus.Lookup("itd-15")
	if !ok {
		t.Fatal("itd-15 not loaded")
	}
	e, err := ReviewOf(root, it)
	if err != nil {
		t.Fatal(err)
	}
	if e.State != ReviewOwed || e.ReceiptID != firstDupRcp {
		t.Fatalf("ReviewOf(itd-15) = %+v", e)
	}
}

// snapshotTree renders every file under root with its bytes, so a zero-write
// assertion compares one string.
func snapshotTree(t *testing.T, root string) string {
	t.Helper()
	var b strings.Builder
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			b.WriteString("dir " + p + "\n")
			return nil
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		b.WriteString("file " + p + "\n" + string(data) + "\n")
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return b.String()
}
