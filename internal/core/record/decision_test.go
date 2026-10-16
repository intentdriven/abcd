package record

import (
	"strings"
	"testing"
)

// TestRecordDispatchDescribesADecision pins `abcd dcn-<id>` (spc-2610110931381316,
// step 1): the root command routes a decision handle here, and the description
// says what the record is — the decision's own words, its date, where it sits —
// the way `abcd iss-<id>` does, and writes nothing. A padded spelling is the
// same handle; an absent one faults naming it; a record the folder's reader
// refuses is reported as skipped on read, not as not found.
func TestRecordDispatchDescribesADecision(t *testing.T) {
	repo := t.TempDir()
	const rel = ".abcd/work/decisions/dcn-2610141200001234-ship-the-folder.md"
	write(t, repo, rel,
		"---\nid: dcn-2610141200001234\nslug: ship-the-folder\ndate: 2026-10-14\n---\n"+
			"2026-10-14 — ship the folder before the history moves\n")
	write(t, repo, ".abcd/work/decisions/dcn-2610141200005678-a-bad-date.md",
		"---\nid: dcn-2610141200005678\nslug: a-bad-date\ndate: 2026-10-13\n---\n2026-10-13 — a bad date\n")
	before := treeSnapshot(t, repo)

	if !IDRe.MatchString("dcn-2610141200001234") {
		t.Fatal("the root command's id gate must admit a dcn handle")
	}
	for _, asked := range []string{"dcn-2610141200001234", "dcn-02610141200001234"} {
		d, err := Describe(repo, asked)
		if err != nil {
			t.Fatalf("Describe(%s): %v", asked, err)
		}
		if d.ID != "dcn-2610141200001234" || d.Family != "decision" || d.Status != "recorded" ||
			d.Title != "2026-10-14 — ship the folder before the history moves" || d.Path != rel {
			t.Errorf("Describe(%s) = %+v", asked, d)
		}
		if d.Links["date"] != "2026-10-14" {
			t.Errorf("Describe(%s) links = %v; want the date", asked, d.Links)
		}
		if len(d.NextMoves) != 1 || !strings.Contains(d.NextMoves[0], "a correction is a new decision") {
			t.Errorf("Describe(%s) next moves = %v", asked, d.NextMoves)
		}
	}
	if _, err := Describe(repo, "dcn-2610141200009999"); err == nil || !strings.Contains(err.Error(), "dcn-2610141200009999") {
		t.Errorf("an absent decision must fault naming it; got %v", err)
	}
	if _, err := Describe(repo, "dcn-2610141200005678"); err == nil || !strings.Contains(err.Error(), ErrSkippedRecord.Error()) {
		t.Errorf("a refused decision must be reported as skipped on read; got %v", err)
	}
	assertZeroWrites(t, repo, before)
}
