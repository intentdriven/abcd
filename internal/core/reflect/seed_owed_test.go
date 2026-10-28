package reflect

import (
	stdreflect "reflect"
	"testing"

	"github.com/intentdriven/abcd/internal/core/changelog"
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

// legacyNotes is a hand-written audit from before the reviewer existed: prose
// and no review marker.
const legacyNotes = "## Audit Notes\n\nAudited by hand on 2026-07-01: every criterion holds.\n"

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
