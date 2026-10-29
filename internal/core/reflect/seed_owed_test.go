package reflect

import (
	stdreflect "reflect"
	"testing"

	"github.com/intentdriven/abcd/internal/core/changelog"
	"github.com/intentdriven/abcd/internal/core/intent"
	"github.com/intentdriven/abcd/internal/core/launch"
	"github.com/intentdriven/abcd/internal/gittest"
)

// amendedOwedNotes is the shape the v0.12.0 audit found on six records: an
// amendment note a ruling wrote into Audit Notes above the OWED block a ship
// parked (iss-2610241431221889, itd-24 ac-2).
const amendedOwedNotes = "## Audit Notes\n\n" +
	"- 2026-10-02 — Amendment to decision 8, recorded by lane rd2 of autonomous run A.\n\n" +
	"<!-- abcd-review: OWED receipt=rcp-bbbbbbbbbbbb -->\n" +
	"Fidelity review OWED (receipt rcp-bbbbbbbbbbbb).\n" +
	"<!-- abcd-review-end receipt=rcp-bbbbbbbbbbbb -->\n"

// legacyNotes is a hand-written audit from before the reviewer existed: no
// review marker, and a paragraph that names the criteria and gives the
// auditor's verdict on each (intent.HandAudited).
const legacyNotes = "## Audit Notes\n\nAudited by hand on 2026-07-01 against the merged code: criteria 1 and 2\n" +
	"are MET, and criterion 3 is NOT_MET because the refusal names no tag.\n"

// strayNotes is an Audit Notes section with no review marker that holds only a
// ruling's amendment line: it names no criterion and gives no verdict, so it is
// not an audit (iss-2610241431221889).
const strayNotes = "## Audit Notes\n\n" +
	"- 2026-10-02 — Amendment to decision 8, recorded by lane rd2 of autonomous run A.\n"

// Criterion 2: an owed review is not audit notes whatever other prose the
// section carries, so the intent is named and offered the audit. A hand-written
// audit with no marker still counts.
func TestSeedOffersTheAuditToAnOwedReviewBesideAnAmendmentNote(t *testing.T) {
	r := gittest.NewRepo(t)
	r.Write(shippedDir+"itd-1-first.md", intentDoc("itd-1", "additive", "", "The first promise", auditedNotes))
	r.Commit("first release")
	r.Git("tag", "v0.1.0")
	r.Write(shippedDir+"itd-2-amended.md", intentDoc("itd-2", "fix", "", "An amended promise", amendedOwedNotes))
	r.Write(shippedDir+"itd-3-legacy.md", intentDoc("itd-3", "fix", "", "An old promise", legacyNotes))
	r.Commit("second release")
	r.Git("tag", "v0.2.0")

	s, err := BuildSeed(r.Root(), "v0.2.0")
	if err != nil {
		t.Fatalf("BuildSeed: %v", err)
	}
	want := []AuditOffer{{IntentID: "itd-2", Command: "abcd intent audit itd-2"}}
	if !stdreflect.DeepEqual(s.Unaudited, want) {
		t.Fatalf("unaudited = %+v, want %+v", s.Unaudited, want)
	}
	if s.Metrics.Audited != 1 || s.Metrics.Unaudited != 1 {
		t.Errorf("metrics audited/unaudited = %d/%d, want 1/1", s.Metrics.Audited, s.Metrics.Unaudited)
	}
}

// Criterion 2, read by the one rule the owed listing reads: a section with no
// review marker is audit notes only when it names the criteria and gives a
// verdict on them. A stray amendment line is not, so the intent is offered the
// audit; a hand-written paragraph with verdicts is. The seed and the owed
// listing agree on both records (iss-2610241431221889).
func TestSeedJudgesAnUnmarkedSectionByTheOwedListingsRule(t *testing.T) {
	r := gittest.NewRepo(t)
	r.Write(shippedDir+"itd-1-first.md", intentDoc("itd-1", "additive", "", "The first promise", auditedNotes))
	r.Commit("first release")
	r.Git("tag", "v0.1.0")
	r.Write(shippedDir+"itd-2-stray.md", intentDoc("itd-2", "fix", "", "A promise with a stray note", strayNotes))
	r.Write(shippedDir+"itd-3-legacy.md", intentDoc("itd-3", "fix", "", "An old promise", legacyNotes))
	r.Commit("second release")
	r.Git("tag", "v0.2.0")

	s, err := BuildSeed(r.Root(), "v0.2.0")
	if err != nil {
		t.Fatalf("BuildSeed: %v", err)
	}
	want := []AuditOffer{{IntentID: "itd-2", Command: "abcd intent audit itd-2"}}
	if !stdreflect.DeepEqual(s.Unaudited, want) {
		t.Fatalf("unaudited = %+v, want %+v", s.Unaudited, want)
	}

	corpus, err := intent.Load(r.Root())
	if err != nil {
		t.Fatalf("intent.Load: %v", err)
	}
	for _, si := range s.Intents {
		it, ok := corpus.Lookup(si.ID)
		if !ok {
			t.Fatalf("%s not in the intent store", si.ID)
		}
		e, err := intent.ReviewOf(r.Root(), it)
		if err != nil {
			t.Fatalf("ReviewOf(%s): %v", si.ID, err)
		}
		if si.Audited == e.IsOwed() {
			t.Errorf("%s: seed audited=%v but the owed listing says owed=%v (%+v)", si.ID, si.Audited, e.IsOwed(), e)
		}
	}
}

// Criterion 7: a release cut by launch ship rewrites each target it passes to
// `next` before the tag exists, so no record names the tag by the time reflect
// runs. The cut's own note in the release's changelog section is what still
// says those intents were targeted at it and did not ship; another release's
// note is not. The list is "still unshipped", so a noted intent that shipped
// after the tag, or that the store no longer holds, is not named.
func TestSeedListsTheTargetsTheCutMovedPastTheRelease(t *testing.T) {
	r := gittest.NewRepo(t)
	r.Write(shippedDir+"itd-1-first.md", intentDoc("itd-1", "additive", "", "The first promise", auditedNotes))
	r.Write(plannedDir+"itd-4-older.md", "---\nid: itd-4\ntarget_release: next\n---\n\n# Missed an earlier release\n")
	r.Write("CHANGELOG.md", "# Changelog\n\n## [Unreleased]\n\n## [0.1.0] - 2026-09-01\n\n"+
		changelog.TargetMoveNote([]launch.TargetMove{{ID: "itd-4", From: "v0.1.0"}})+"\n\n### Added\n\n- The first promise (itd-1).\n")
	r.Commit("first release")
	r.Git("tag", "v0.1.0")

	r.Write(shippedDir+"itd-2-second.md", intentDoc("itd-2", "additive", "", "The second promise", auditedNotes))
	r.Write(plannedDir+"itd-5-late.md", "---\nid: itd-5\ntarget_release: next\n---\n\n# A promise that missed its release\n")
	r.Write(plannedDir+"itd-6-later.md", "---\nid: itd-6\ntarget_release: next\n---\n\n# A promise that shipped a release late\n")
	r.Write("CHANGELOG.md", "# Changelog\n\n## [Unreleased]\n\n## [0.2.0] - 2026-09-20\n\n"+
		changelog.TargetMoveNote([]launch.TargetMove{
			{ID: "itd-5", From: launch.TargetNext},
			{ID: "itd-6", From: launch.TargetNext},
			{ID: "itd-7", From: launch.TargetNext},
		})+"\n\n### Added\n\n- The second promise (itd-2).\n\n"+
		"## [0.1.0] - 2026-09-01\n\n"+
		changelog.TargetMoveNote([]launch.TargetMove{{ID: "itd-4", From: "v0.1.0"}})+"\n\n### Added\n\n- The first promise (itd-1).\n")
	r.Commit("second release")
	r.Git("tag", "v0.2.0")

	r.Git("mv", plannedDir+"itd-6-later.md", shippedDir+"itd-6-later.md")
	r.Commit("itd-6 ships in the next release")

	s, err := BuildSeed(r.Root(), "v0.2.0")
	if err != nil {
		t.Fatalf("BuildSeed: %v", err)
	}
	want := []TargetedIntent{{ID: "itd-5", Path: plannedDir + "itd-5-late.md"}}
	if !stdreflect.DeepEqual(s.Unshipped, want) {
		t.Fatalf("unshipped = %+v, want %+v", s.Unshipped, want)
	}
}
