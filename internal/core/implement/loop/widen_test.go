package loop

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// An issue lane's hold follows what its branch changes (iss-2610311224444576,
// iss-2611110638266241, iss-2611101316537446; dcn-2610311251002713,
// dcn-2610311544325397): once an implementer hands its receipt back, the loop
// itself reserves the issue again with the held footprint and every path the
// branch changes beyond it, judged for overlap, before it judges the receipt;
// again before each step at the lane's validators and at the fence before its
// landing's push. Another copy's live hold on such a path is a wait by design.
// `abcd implement widen` stays for a host session's implementer, and backs off
// (exit 3, nothing changed) when another copy holds a path live. A lane leaving
// its landing for a fresh round keeps the paths it widened to.

// handBack commits names in the lane's worktree and hands the implementer's
// receipt back at o's clock, returning the receipt's error.
func (f *narrowFixture) handBack(t *testing.T, o Options, names ...string) error {
	t.Helper()
	f.awaitImplementer(t)
	l := currentLane(t, f.repo, f.runID)
	sha := commitIn(t, f.repo, l, names...)
	dir := filepath.Join(f.repo.Root(), filepath.FromSlash(RunRelDir), f.runID, l.ID)
	rc := goodReceipt(t, f.runID, l, dir, sha)
	rc.Model = "claude-test-5"
	rc.Resolves = []Resolution{{Issue: secondIssue, Commit: sha, Note: "the page names the flag", Impact: "fix",
		Grounds: "pursued: the page names the flag; shown wrong if it still does not"}}
	_, err := Receipt(f.repo.Root(), f.runID, writeReceipt(t, dir, rc), f.stages, o)
	return err
}

// widen widens the current lane's hold to paths at o's clock.
func (f *narrowFixture) widen(t *testing.T, o Options, paths ...string) (WidenResult, error) {
	t.Helper()
	f.awaitImplementer(t)
	return Widen(f.repo.Root(), f.runID, currentLane(t, f.repo, f.runID).ID, paths, o)
}

func sortedFootprint(t *testing.T, f *narrowFixture) []string {
	t.Helper()
	fp := slices.Clone(f.footprint(t))
	slices.Sort(fp)
	return fp
}

// TestAnIssueLaneWidensItsHoldBeforeItEditsAPathBeyondIt: the lane holds the
// directory its found_at names; widening to README.txt reserves the issue
// again with both, says so in the run record, and a widening to a path the
// hold already covers pushes nothing.
func TestAnIssueLaneWidensItsHoldBeforeItEditsAPathBeyondIt(t *testing.T) {
	f := newNarrowFixture(t)
	res, err := f.widen(t, Options{}, "README.txt")
	if err != nil {
		t.Fatalf("the widening is made: %v", err)
	}
	if fp := sortedFootprint(t, f); !slices.Equal(fp, []string{"README.txt", "docs"}) {
		t.Fatalf("the reservation holds the found_at and the further path: %q", fp)
	}
	if !slices.Equal(res.Added, []string{"README.txt"}) {
		t.Fatalf("the result names the path newly held: %+v", res)
	}
	st, err := ReadState(f.repo.Root(), f.runID)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.ContainsFunc(st.Record, func(e Entry) bool {
		return e.Stage == StageReservation && e.Lane == res.Lane && strings.Contains(e.Note, "widened") && strings.Contains(e.Note, "README.txt")
	}) {
		t.Fatalf("the run record says the hold widened: %+v", st.Record)
	}

	before := f.tip(t)
	again, err := f.widen(t, Options{}, "docs/page.md", "README.txt")
	if err != nil {
		t.Fatalf("a widening to held paths is no error: %v", err)
	}
	if len(again.Added) != 0 || f.tip(t).commit != before.commit {
		t.Fatalf("a widening to paths already held pushes nothing: %+v, tip %+v over %+v", again, f.tip(t), before)
	}
}

// TestAWideningBacksOffAnotherCopysLiveHold: another account holds
// README.txt live; the widening is contention (exit 3), names the hold, and
// leaves the lane's reservation as it was.
func TestAWideningBacksOffAnotherCopysLiveHold(t *testing.T) {
	f := newNarrowFixture(t)
	f.reserveAs(t, "iss-99", "quinn", "README.txt", time.Now().Add(-time.Minute))
	before := f.tip(t)
	_, err := f.widen(t, Options{}, "README.txt")
	r, ok := AsRefusal(err)
	if !ok || !r.Contention {
		t.Fatalf("the widening backs off as contention: %v", err)
	}
	if !strings.Contains(r.Reason, "iss-99") || !strings.Contains(r.Reason, "README.txt") {
		t.Fatalf("the back-off names the hold it met: %s", r.Reason)
	}
	if tip := f.tip(t); tip.commit != before.commit {
		t.Fatalf("nothing is pushed: %+v over %+v", tip, before)
	}
	if fp := f.footprint(t); !slices.Equal(fp, []string{"docs"}) {
		t.Fatalf("the reservation is left as it was: %q", fp)
	}
	st, err := ReadState(f.repo.Root(), f.runID)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.ContainsFunc(st.Record, func(e Entry) bool {
		return e.Stage == StageReservation && strings.Contains(e.Note, "backed off") && strings.Contains(e.Note, "README.txt")
	}) {
		t.Fatalf("the run record says the widening backed off: %+v", st.Record)
	}
}

// walledStage does an implementer's work as a drive's walled stage does it,
// with no way to reach origin (iss-2611110638266241): origin is moved away
// while names are committed in the lane's worktree and the receipt is
// written, and the widen step the brief once named fails there. The receipt's
// path is returned, and origin is put back for the loop's own hand-in.
func (f *narrowFixture) walledStage(t *testing.T, names ...string) string {
	t.Helper()
	f.awaitImplementer(t)
	l := currentLane(t, f.repo, f.runID)
	away := f.bare + ".walled"
	if err := os.Rename(f.bare, away); err != nil {
		t.Fatal(err)
	}
	back := func() {
		if _, err := os.Stat(away); err == nil {
			if err := os.Rename(away, f.bare); err != nil {
				t.Fatal(err)
			}
		}
	}
	t.Cleanup(back)
	if _, err := Widen(f.repo.Root(), f.runID, l.ID, names, Options{}); err == nil {
		t.Fatal("premise: a stage that cannot reach origin cannot widen the hold itself")
	}
	sha := commitIn(t, f.repo, l, names...)
	dir := filepath.Join(f.repo.Root(), filepath.FromSlash(RunRelDir), f.runID, l.ID)
	rc := goodReceipt(t, f.runID, l, dir, sha)
	rc.Model = "claude-test-5"
	rc.Resolves = []Resolution{{Issue: secondIssue, Commit: sha, Note: "the page names the flag", Impact: "fix",
		Grounds: "pursued: the page names the flag; shown wrong if it still does not"}}
	path := writeReceipt(t, dir, rc)
	back()
	return path
}

// TestTheLoopReservesWhatAWalledStageChangesBeforeItsValidators: a walled
// implementer, with no credential for origin, commits README.txt beyond the
// found_at and hands its receipt back; the loop's own hand-in reserves the
// path before it judges the receipt, the receipt verifies, the lane moves to
// its validators, and the run record names the path widened to.
func TestTheLoopReservesWhatAWalledStageChangesBeforeItsValidators(t *testing.T) {
	f := newNarrowFixture(t)
	path := f.walledStage(t, "docs/page.md", "README.txt")
	if fp := f.footprint(t); !slices.Equal(fp, []string{"docs"}) {
		t.Fatalf("premise: nothing reached origin while the stage worked: %q", fp)
	}
	if _, err := Receipt(f.repo.Root(), f.runID, path, f.stages, Options{}); err != nil {
		t.Fatalf("the receipt verifies once the loop has widened the hold: %v", err)
	}
	if fp := sortedFootprint(t, f); !slices.Equal(fp, []string{"README.txt", "docs"}) {
		t.Fatalf("the loop reserves the path the lane's branch changes beyond its found_at: %q", fp)
	}
	l := currentLane(t, f.repo, f.runID)
	if l.Stage != StageValidate {
		t.Fatalf("the lane moves to its validators: %s", l.Stage)
	}
	st, err := ReadState(f.repo.Root(), f.runID)
	if err != nil {
		t.Fatal(err)
	}
	widenedAt := slices.IndexFunc(st.Record, func(e Entry) bool {
		return e.Stage == StageWidenedTo && e.Lane == l.ID && e.Note == "README.txt"
	})
	verifiedAt := slices.IndexFunc(st.Record, func(e Entry) bool {
		return e.Stage == "receipt" && e.Lane == l.ID && strings.Contains(e.Note, "implement stage's receipt verified")
	})
	if widenedAt < 0 || verifiedAt < 0 || widenedAt > verifiedAt {
		t.Fatalf("the run record names the path widened to, ahead of the receipt it judged: %+v", st.Record)
	}
	if got := widenedPaths(st); !slices.Contains(got, "README.txt") {
		t.Fatalf("a reservation made again names the path widened to: %q", got)
	}
}

// TestAPeersHoldOnAChangedPathIsAWaitNotAFailedLaunch: another account holds
// README.txt live, and the walled implementer changes it; the receipt is taken
// all the same, with nothing pushed and the record naming the hold, and the
// lane waits before its validators, every step a wait by design that is never
// counted toward the refusals bound, until the hold lapses, when the next step
// reserves the path and hands the lane to its first validator.
func TestAPeersHoldOnAChangedPathIsAWaitNotAFailedLaunch(t *testing.T) {
	f := newNarrowFixture(t)
	f.reserveAs(t, "iss-99", "quinn", "README.txt", time.Now().Add(-time.Minute))
	path := f.walledStage(t, "docs/page.md", "README.txt")
	if _, err := Receipt(f.repo.Root(), f.runID, path, f.stages, Options{}); err != nil {
		t.Fatalf("a back-off over another's hold does not refuse the receipt, which a drive would count as a failed launch: %v", err)
	}
	if fp := f.footprint(t); !slices.Equal(fp, []string{"docs"}) {
		t.Fatalf("nothing is pushed over the other account's hold: %q", fp)
	}
	st, err := ReadState(f.repo.Root(), f.runID)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.ContainsFunc(st.Record, func(e Entry) bool {
		return e.Stage == StageReservation && strings.Contains(e.Note, "could not be made yet") &&
			strings.Contains(e.Note, "README.txt") && strings.Contains(e.Note, "iss-99") && strings.Contains(e.Note, "quinn")
	}) {
		t.Fatalf("the run record names the path and the hold it backed off: %+v", st.Record)
	}
	b := stuckBounds{refusals: PaceValue{Value: BundledStuckRefusals}, minutes: PaceValue{Value: BundledStuckMinutes}}
	for i := range BundledStuckRefusals + 2 {
		_, err := advance(f.repo.Root(), f.runID, f.stages, Options{})
		rf, ok := AsRefusal(err)
		if !ok || !rf.Contention || !waitByDesign(rf) {
			t.Fatalf("step %d: the lane waits by design before its validators: %v", i, err)
		}
		if !strings.Contains(rf.Reason, "README.txt") || !strings.Contains(rf.Reason, "iss-99") || !strings.Contains(rf.Reason, "quinn") {
			t.Fatalf("step %d: the wait names the path and its holder: %s", i, rf.Reason)
		}
		run, err := ReadState(f.repo.Root(), f.runID)
		if err != nil {
			t.Fatal(err)
		}
		l := run.Lanes[run.current()]
		if l.Refused == nil || l.Refused.Count != 0 || stuckReason(run, b) != "" || l.HandBack != nil || l.Stage != StageValidate || len(l.Awaits) != 0 {
			t.Fatalf("step %d: the wait is counted toward no refusals bound, the lane neither stuck nor handed back, and no validator starts: %+v", i, l)
		}
	}
	// The other account's hold lapses: it is reserved again an hour past its end.
	f.reserveAsOver(t, "iss-99", "quinn", "README.txt", time.Now().Add(-6*time.Hour))
	res, err := advance(f.repo.Root(), f.runID, f.stages, Options{})
	if err != nil {
		t.Fatalf("once the hold has ended the lane goes on: %v", err)
	}
	if res.Awaiting == nil || res.Awaiting.Role == RoleImplementer {
		t.Fatalf("the lane is handed to its first validator, not to an implementer again: %+v", res)
	}
	if fp := sortedFootprint(t, f); !slices.Equal(fp, []string{"README.txt", "docs"}) {
		t.Fatalf("the path is reserved before the validators judge it: %q", fp)
	}
}

// reserveAsOver is reserveAs over the reservation id already has on origin,
// which it replaces.
func (f *narrowFixture) reserveAsOver(t *testing.T, id, holder, footprint string, at time.Time) {
	t.Helper()
	gitIn(t, f.bare, "update-ref", "-d", claim.RefPrefix+id)
	f.reserveAs(t, id, holder, footprint, at)
}

// TestALaneLeavingItsLandingKeepsThePathsItWidenedTo: the lane widened to
// README.txt and changed it; the landing narrows to the changed files, and
// the sync that sends it back to a fresh round reserves the found_at again
// beside README.txt, never dropping the path the lane still changes.
func TestALaneLeavingItsLandingKeepsThePathsItWidenedTo(t *testing.T) {
	f := newNarrowFixture(t)
	if _, err := f.widen(t, Options{}, "README.txt"); err != nil {
		t.Fatal(err)
	}
	if err := f.handBack(t, Options{}, "docs/page.md", "README.txt"); err != nil {
		t.Fatal(err)
	}
	f.moveDefault(t, "docs/guide.md")
	f.toLand(t)
	due := f.tip(t).claimed.Add(61 * time.Minute)
	f.stepAt(t, due)
	l := currentLane(t, f.repo, f.runID)
	if l.Stage != StageValidate || len(l.Syncs) != 1 {
		t.Fatalf("the landing's sync sends the lane back to a fresh round: stage %s, syncs %d", l.Stage, len(l.Syncs))
	}
	if fp := sortedFootprint(t, f); !slices.Equal(fp, []string{"README.txt", "docs"}) {
		t.Fatalf("the lane leaving its landing holds its found_at and the path it widened to: %q", fp)
	}
}

// TestAWideningIsRefusedOutsideTheLanesEditingStages: a lane at its landing
// widens nothing, and a lane the run does not have is refused; neither is
// contention.
func TestAWideningIsRefusedOutsideTheLanesEditingStages(t *testing.T) {
	f := newNarrowFixture(t)
	_, err := Widen(f.repo.Root(), f.runID, "lane-9", []string{"README.txt"}, Options{})
	if r, ok := AsRefusal(err); !ok || r.Contention || !strings.Contains(r.Reason, "lane-9") {
		t.Fatalf("a lane the run does not have is refused: %v", err)
	}
	if err := f.handBack(t, Options{}, "docs/page.md"); err != nil {
		t.Fatal(err)
	}
	f.toLand(t)
	l := currentLane(t, f.repo, f.runID)
	_, err = Widen(f.repo.Root(), f.runID, l.ID, []string{"README.txt"}, Options{})
	if r, ok := AsRefusal(err); !ok || r.Contention || !strings.Contains(r.Reason, string(StageLand)) {
		t.Fatalf("a lane at its landing is refused, naming its stage: %v", err)
	}
	if fp := f.footprint(t); slices.Contains(fp, "README.txt") {
		t.Fatalf("nothing is widened: %q", fp)
	}
}

// TestAnIssueBriefSaysTheLoopReservesWhatTheBranchChanges: the lane's brief
// no longer sends the implementer to a widen step it cannot run behind a
// drive's wall: it says the loop reserves what the commits change once the
// receipt is handed back, and what is still refused or waited for.
func TestAnIssueBriefSaysTheLoopReservesWhatTheBranchChanges(t *testing.T) {
	f := newNarrowFixture(t)
	f.awaitImplementerOnce(t)
	l := currentLane(t, f.repo, f.runID)
	brief, err := os.ReadFile(filepath.Join(f.repo.Root(), filepath.FromSlash(l.Brief)))
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range []string{"## Files beyond the issue's place", "you run nothing to\nreserve it",
		"the loop reserves every path your commits change", "A path another copy holds", "the lane is handed back, naming the\n  path and its holder",
		"A path no reservation may carry", "A gate file", "person's own session"} {
		if !strings.Contains(string(brief), s) {
			t.Fatalf("the brief carries %q:\n%s", s, brief)
		}
	}
	for _, s := range []string{"implement widen", "write no receipt"} {
		if strings.Contains(string(brief), s) {
			t.Fatalf("the brief names no widen step for the implementer (%q):\n%s", s, brief)
		}
	}
	flat := strings.Join(strings.Fields(string(brief)), " ")
	for _, s := range []string{"a backslash", "a control, non-printing, bidirectional or zero-width character", "a leading or trailing space",
		"the hooks path the repository configures"} {
		if !strings.Contains(flat, s) {
			t.Fatalf("the brief says which names no reservation may carry, and every gate (%q):\n%s", s, brief)
		}
	}
	// The brief names every gate the drive refuses to land, from the list the
	// drive reads (DriveGates), so the two never drift apart.
	for _, g := range DriveGates {
		if !strings.Contains(flat, "`"+g+"`") || !DriveGate(g, "") {
			t.Fatalf("the brief names the gate %s the drive refuses:\n%s", g, brief)
		}
	}
}

// TestAResumeOverALapsedHoldKeepsThePathsTheLaneWidenedTo: the lane widened
// to README.txt, changed it and reached its landing, and its reservation
// lapsed while the run sat; `abcd build` of the issue reserves it again with
// the found_at and README.txt, never the found_at alone.
func TestAResumeOverALapsedHoldKeepsThePathsTheLaneWidenedTo(t *testing.T) {
	f := newNarrowFixture(t)
	if _, err := f.widen(t, Options{}, "README.txt"); err != nil {
		t.Fatal(err)
	}
	if err := f.handBack(t, Options{}, "docs/page.md", "README.txt"); err != nil {
		t.Fatal(err)
	}
	f.toLand(t)
	lapsed := f.tip(t).expires.Add(time.Hour)
	if _, err := Start(f.repo.Root(), secondIssue, Options{Now: func() time.Time { return lapsed }}); err != nil {
		t.Fatalf("the resume reserves the issue again: %v", err)
	}
	if fp := sortedFootprint(t, f); !slices.Contains(fp, "README.txt") || !covers(fp, []string{"docs"}) {
		t.Fatalf("the resume over a lapsed hold keeps the path the lane widened to beside its found_at: %q", fp)
	}
}

// TestTheBackstopIgnoresAReleasedHold: the reservation this copy held was
// released while the implementer worked; the loop widens nothing it does not
// hold, the receipt is not refused on the hold, and the land fence settles
// the loss.
func TestTheBackstopIgnoresAReleasedHold(t *testing.T) {
	f := newNarrowFixture(t)
	f.awaitImplementerOnce(t)
	releaseKey(f.repo.Root(), secondIssue, time.Now())
	err := f.handBack(t, Options{}, "docs/page.md", "README.txt")
	if r, ok := AsRefusal(err); ok && strings.Contains(r.Reason, "a hold on") {
		t.Fatalf("a released hold refuses no receipt on the hold: %s", r.Reason)
	}
	if fp := f.footprint(t); slices.Contains(fp, "README.txt") {
		t.Fatalf("a hold this copy no longer has is not widened: %q", fp)
	}
}

// TestALaneAtItsValidateStageWidensItsHold: a lane whose receipt verified is
// at its validate stage, where a fix round hands an implementer the lane
// again, so a widening there reserves the further path like one at the
// implement stage.
func TestALaneAtItsValidateStageWidensItsHold(t *testing.T) {
	f := newNarrowFixture(t)
	if err := f.handBack(t, Options{}, "docs/page.md"); err != nil {
		t.Fatal(err)
	}
	l := currentLane(t, f.repo, f.runID)
	if l.Stage != StageValidate {
		t.Fatalf("the lane is at its validate stage: %s", l.Stage)
	}
	res, err := Widen(f.repo.Root(), f.runID, l.ID, []string{"README.txt"}, Options{})
	if err != nil {
		t.Fatalf("a lane at its validate stage widens: %v", err)
	}
	if !slices.Equal(res.Added, []string{"README.txt"}) {
		t.Fatalf("the widening adds README.txt: %q", res.Added)
	}
	if fp := sortedFootprint(t, f); !slices.Equal(fp, []string{"README.txt", "docs"}) {
		t.Fatalf("the hold names the found_at and README.txt: %q", fp)
	}
}

// TestTheLoopReservesTheRunsOwnRecordsLikeAnyPath: the run's own records the
// lane's branch changes, a decision record the implementer wrote and the
// issue's record the landing's records commit moves, are reserved by the
// loop's widening like any other path, and refuse no receipt.
func TestTheLoopReservesTheRunsOwnRecordsLikeAnyPath(t *testing.T) {
	f := newNarrowFixture(t)
	decision := ".abcd/work/decisions/dcn-2611110000000001-a-judgement-call.md"
	record := ".abcd/work/issues/resolved/" + secondIssue + "-the-record.md"
	if err := f.handBack(t, Options{}, "docs/page.md", decision); err != nil {
		t.Fatalf("a decision record the lane wrote refuses no receipt: %v", err)
	}
	l := currentLane(t, f.repo, f.runID)
	commitIn(t, f.repo, l, record)
	st, err := ReadState(f.repo.Root(), f.runID)
	if err != nil {
		t.Fatal(err)
	}
	var lines []Entry
	if g := receiptHoldGaps(Context{RepoRoot: f.repo.Root(), State: st, Now: time.Now(), widened: &lines}, l); len(g) != 0 {
		t.Fatalf("the issue's own record is no gap: %q", g)
	}
	fp := f.footprint(t)
	for _, p := range []string{"docs", decision, record} {
		if !covers(fp, []string{p}) {
			t.Fatalf("the reservation covers %s: %q", p, fp)
		}
	}
	if !slices.ContainsFunc(lines, func(e Entry) bool { return e.Stage == StageWidenedTo && e.Note == record }) {
		t.Fatalf("the widening's lines name the record: %+v", lines)
	}
}

// holdNamesTheFoundAtAlone implements the lane with README.txt widened to and
// takes it past its validators, then makes the reservation name the found_at
// alone, as a reservation made again over a deleted tip by an earlier abcd
// did, leaving the lane at its landing.
func holdNamesTheFoundAtAlone(t *testing.T, f *narrowFixture) {
	t.Helper()
	if err := f.handBack(t, Options{}, "docs/page.md", "README.txt"); err != nil {
		t.Fatal(err)
	}
	f.toLand(t)
	if l := currentLane(t, f.repo, f.runID); l.Stage != StageLand {
		t.Fatalf("the lane is at its landing: %s", l.Stage)
	}
	if _, err := claim.Reserve(f.repo.Root(), claim.Request{Record: secondIssue, Footprint: []string{"docs"}}, claim.Options{}); err != nil {
		t.Fatal(err)
	}
	if fp := f.footprint(t); !slices.Equal(fp, []string{"docs"}) {
		t.Fatalf("the reservation names the found_at alone: %q", fp)
	}
}

// stepToThePush steps the lane from its validators to the fence before its
// landing's push, its head preflighted, returning the first contention met.
// before, when set, runs ahead of each step at the landing.
func stepToThePush(t *testing.T, f *narrowFixture, before func(Lane)) (Lane, error) {
	t.Helper()
	for range 4 * len(Sequence) {
		f.toLand(t)
		l := currentLane(t, f.repo, f.runID)
		if l.Landing != nil && l.Landing.Pushed != "" {
			return l, nil
		}
		if before != nil {
			before(l)
			l = currentLane(t, f.repo, f.runID)
		}
		if l.Landing != nil && l.Landing.RecordsDone {
			preflighted(t, l, l.HeadSHA)
		}
		if _, err := advance(f.repo.Root(), f.runID, f.stages, Options{}); err != nil {
			if rf, ok := AsRefusal(err); ok && rf.Contention {
				return currentLane(t, f.repo, f.runID), err
			}
			t.Fatalf("stepping to the push: %v", err)
		}
	}
	t.Fatal("the lane never reached its push")
	return Lane{}, nil
}

// leftInTheWorktree writes name in the lane's worktree once its landing has
// begun and before its records are committed.
func leftInTheWorktree(t *testing.T, name string) func(Lane) {
	return func(l Lane) {
		t.Helper()
		if l.Landing == nil || l.Landing.RecordsDone {
			return
		}
		if err := os.WriteFile(filepath.Join(l.Worktree, name), []byte("a note\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
}

// committedUnderTheLandingsSubject commits name on the judged head under the
// subject of the landing's records commit, once its landing has begun, as a
// call stopped after its commit leaves it for the next to find.
func committedUnderTheLandingsSubject(t *testing.T, f *narrowFixture, name string) func(Lane) {
	return func(l Lane) {
		t.Helper()
		if l.Landing == nil || l.Landing.RecordsDone {
			return
		}
		tip, err := gitutil.Run(l.Worktree, "rev-parse", "HEAD")
		if err != nil {
			t.Fatal(err)
		}
		if strings.TrimSpace(tip) != l.HeadSHA {
			return
		}
		st, err := ReadState(f.repo.Root(), f.runID)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(l.Worktree, name), []byte("a note\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := gitutil.LocalGit(l.Worktree, "add", name); err != nil {
			t.Fatal(err)
		}
		if out, err := gitutil.LocalGit(l.Worktree, "commit", "-q", "-m", landSubject(st, l)); err != nil {
			t.Fatalf("%v: %s", err, out)
		}
	}
}

// forgotAfterItsCommit puts the run back once, as a call killed after the landing's
// commit leaves it, so the next step finds that commit on the judged head.
func forgotAfterItsCommit(t *testing.T, f *narrowFixture) func(Lane) {
	done := false
	return func(l Lane) {
		t.Helper()
		if done || l.Landing == nil || !l.Landing.RecordsDone {
			return
		}
		done = true
		judged, err := gitutil.Run(l.Worktree, "rev-parse", "HEAD~1")
		if err != nil {
			t.Fatal(err)
		}
		err = mutate(f.repo.Root(), f.runID, func(_ *os.Root, st *State) (bool, error) {
			cur := &st.Lanes[st.current()]
			cur.HeadSHA, cur.Landing.Records, cur.Landing.RecordsDone = strings.TrimSpace(judged), "", false
			return true, nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}

// aHeadStandsIn leaves, once the landing's commit is made, a commit standing in for the
// branch's tip whose tree holds nothing the branch changed.
func aHeadStandsIn(t *testing.T, f *narrowFixture) func(Lane) {
	done := false
	return func(l Lane) {
		t.Helper()
		if done || l.Landing == nil || !l.Landing.RecordsDone {
			return
		}
		done = true
		main := strings.TrimSpace(f.repo.Git("rev-parse", "refs/remotes/origin/main"))
		stand := strings.TrimSpace(f.repo.Git("commit-tree", main+"^{tree}", "-p", main, "-m", "a stand-in"))
		f.repo.Git("replace", "-f", l.HeadSHA, stand)
	}
}

// heldByHand leaves the run with no reserved_at, a reservation made by hand
// naming the found_at alone standing for it; letGo says the run's record shows
// it let an earlier one go.
func heldByHand(t *testing.T, f *narrowFixture, letGo bool) {
	t.Helper()
	holdNamesTheFoundAtAlone(t, f)
	err := mutate(f.repo.Root(), f.runID, func(_ *os.Root, st *State) (bool, error) {
		st.ReservedAt = nil
		if letGo {
			st.Record = append(st.Record, Entry{At: time.Now(), Stage: StageReservationReleased, Note: "let go"})
		}
		st.UpdatedAt = time.Now()
		return true, nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// fenceCase is one way a path the lane's landing would carry comes to lie
// beyond its reservation.
type fenceCase struct {
	name, path string
	linked     bool
	setup      func(t *testing.T, f *narrowFixture)
	before     func(t *testing.T, f *narrowFixture) func(Lane)
}

// fixture is the case's narrow fixture, its path holding a link the landing repoints
// when the case says so.
func (c fenceCase) fixture(t *testing.T) *narrowFixture {
	t.Helper()
	if c.linked {
		return newNarrowFixtureLinking(t, "docs/", "docs", c.path)
	}
	return newNarrowFixture(t)
}

func fenceCases() []fenceCase {
	onlyThePage := func(t *testing.T, f *narrowFixture) {
		t.Helper()
		if err := f.handBack(t, Options{}, "docs/page.md"); err != nil {
			t.Fatal(err)
		}
	}
	return []fenceCase{
		{name: "changed on its branch", path: "README.txt", setup: holdNamesTheFoundAtAlone},
		{name: "repointed by the landing's commit", path: "NOTES.md", linked: true, setup: onlyThePage},
		{name: "repointed in the landing's commit found on the judged head", path: "NOTES.md", linked: true, setup: onlyThePage,
			before: forgotAfterItsCommit},
		{name: "changed on its branch, under a head standing in without it", path: "README.txt", setup: holdNamesTheFoundAtAlone,
			before: aHeadStandsIn},
		{name: "beyond a hand reservation in a run that never held one", path: "README.txt",
			setup: func(t *testing.T, f *narrowFixture) { heldByHand(t, f, false) }},
		{name: "beyond a hand reservation in a run that let its own go", path: "README.txt",
			setup: func(t *testing.T, f *narrowFixture) { heldByHand(t, f, true) }},
	}
}

// hook is the case's step hook for f, nil when it has none.
func (c fenceCase) hook(t *testing.T, f *narrowFixture) func(Lane) {
	if c.before == nil {
		return nil
	}
	return c.before(t, f)
}

// TestALaneDoesNotLandAPathAPeerHolds: the lane's landing would carry a path no
// reservation of its covers, and another account holds that path live; the
// lane waits at the fence before its push, by design and never counted toward
// the refusals bound, naming the hold, with nothing pushed and its reservation
// as it was.
func TestALaneDoesNotLandAPathAPeerHolds(t *testing.T) {
	for _, c := range fenceCases() {
		t.Run(c.name, func(t *testing.T) {
			f := c.fixture(t)
			c.setup(t, f)
			f.reserveAs(t, "iss-99", "quinn", c.path, time.Now().Add(-time.Minute))
			l, err := stepToThePush(t, f, c.hook(t, f))
			rf, ok := AsRefusal(err)
			if !ok || !rf.Contention || !waitByDesign(rf) {
				t.Fatalf("the lane waits by design: %v (%+v)", err, l.Landing)
			}
			if l.Refused == nil || l.Refused.Count != 0 {
				t.Fatalf("the wait is counted toward the minutes bound alone, never the refusals bound: %+v", l.Refused)
			}
			if !strings.Contains(rf.Reason, "iss-99") || !strings.Contains(rf.Reason, "quinn") || !strings.Contains(rf.Reason, c.path) {
				t.Fatalf("the stop names the hold it met and its holder: %s", rf.Reason)
			}
			if l.Stage != StageLand || l.Landing == nil || l.Landing.Pushed != "" {
				t.Fatalf("the lane stays at its landing, unpushed: %s %+v", l.Stage, l.Landing)
			}
			if got := gitIn(t, f.bare, "for-each-ref", "refs/heads/"+l.Branch); got != "" {
				t.Fatalf("the lane's branch reached origin: %s", got)
			}
			if fp := f.footprint(t); !slices.Equal(fp, []string{"docs"}) {
				t.Fatalf("the reservation is left as it was: %q", fp)
			}
		})
	}
}

// TestALaneReservesAPathNoReservationCoversBeforeItLands: with no other hold on
// the path, the fence reserves it, judged like any reservation, and the lane's
// push goes ahead holding every path its branch changes, the issue's own record
// the landing moves among them.
func TestALaneReservesAPathNoReservationCoversBeforeItLands(t *testing.T) {
	for _, c := range fenceCases() {
		t.Run(c.name, func(t *testing.T) {
			f := c.fixture(t)
			c.setup(t, f)
			l, err := stepToThePush(t, f, c.hook(t, f))
			if err != nil {
				t.Fatal(err)
			}
			if got := gitIn(t, f.bare, "for-each-ref", "refs/heads/"+l.Branch); got == "" {
				t.Fatalf("the lane's branch is pushed: %+v", l.Landing)
			}
			want := heldWith([]string{"docs", c.path}, landingMoved(t, f, l))
			slices.Sort(want)
			if fp := sortedFootprint(t, f); !slices.Equal(fp, want) {
				t.Fatalf("the reservation names the path before the push: %q", fp)
			}
		})
	}
}

// TestALandingStopsAPathItsMovesDoNotExplainBeforeTheFence: a path the landing's own
// moves did not repoint never reaches the fence: the landing refuses it by name, as
// no contention, with nothing pushed and the reservation as it was.
func TestALandingStopsAPathItsMovesDoNotExplainBeforeTheFence(t *testing.T) {
	for name, c := range map[string]struct {
		path   string
		before func(t *testing.T, f *narrowFixture) func(Lane)
	}{
		"left uncommitted before its records": {"NOTES.md", func(t *testing.T, _ *narrowFixture) func(Lane) {
			return leftInTheWorktree(t, "NOTES.md")
		}},
		"in a commit under the landing's subject": {"NOTES.txt", func(t *testing.T, f *narrowFixture) func(Lane) {
			return committedUnderTheLandingsSubject(t, f, "NOTES.txt")
		}},
	} {
		t.Run(name, func(t *testing.T) {
			f := newNarrowFixture(t)
			if err := f.handBack(t, Options{}, "docs/page.md"); err != nil {
				t.Fatal(err)
			}
			before := c.before(t, f)
			var rf *Refusal
			for range 4 * len(Sequence) {
				f.toLand(t)
				before(currentLane(t, f.repo, f.runID))
				_, err := advance(f.repo.Root(), f.runID, f.stages, Options{})
				if err != nil {
					var ok bool
					if rf, ok = AsRefusal(err); !ok {
						t.Fatal(err)
					}
					break
				}
			}
			if rf == nil || rf.Contention || !strings.Contains(rf.Reason, c.path) {
				t.Fatalf("the landing refuses the path by name: %+v", rf)
			}
			l := currentLane(t, f.repo, f.runID)
			if l.Stage != StageLand || l.Landing == nil || l.Landing.RecordsDone || l.Landing.Pushed != "" {
				t.Fatalf("the lane stays at its landing, unpushed: %s %+v", l.Stage, l.Landing)
			}
			if got := gitIn(t, f.bare, "for-each-ref", "refs/heads/"+l.Branch); got != "" {
				t.Fatalf("the lane's branch reached origin: %s", got)
			}
			if fp := f.footprint(t); !slices.Equal(fp, []string{"docs"}) {
				t.Fatalf("the reservation is left as it was: %q", fp)
			}
		})
	}
}

// TestWhatALaneChangedIsReadAsStored: the files a lane changed are read from the objects
// as stored, and a nested checkout counts whatever the repository asks git's diff to pass
// over.
func TestWhatALaneChangedIsReadAsStored(t *testing.T) {
	nested := func(t *testing.T, f *narrowFixture, l Lane) string {
		t.Helper()
		dir := filepath.Join(l.Worktree, "vendored")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		f.repo.Git("-C", dir, "init", "-q")
		f.repo.Git("-C", dir, "-c", "user.name=t", "-c", "user.email=t@example.invalid", "commit", "-q", "--allow-empty", "-m", "nested")
		f.repo.Git("-C", l.Worktree, "add", "-A")
		f.repo.Git("-C", l.Worktree, "commit", "-q", "-m", "a nested checkout")
		return strings.TrimSpace(f.repo.Git("-C", l.Worktree, "rev-parse", "HEAD"))
	}
	for name, ignore := range map[string]func(t *testing.T, f *narrowFixture, l Lane){
		"its list of submodules": func(t *testing.T, f *narrowFixture, _ Lane) {
			t.Helper()
			landWrite(t, f.repo.Root(), ".gitmodules", "[submodule \"vendored\"]\n\tpath = vendored\n\turl = ./vendored\n\tignore = all\n")
		},
		"its configuration": func(t *testing.T, f *narrowFixture, _ Lane) {
			t.Helper()
			f.repo.Git("config", "diff.ignoreSubmodules", "all")
		},
	} {
		t.Run("a nested checkout passed over by "+name, func(t *testing.T) {
			f := newNarrowFixture(t)
			f.awaitImplementer(t)
			l := currentLane(t, f.repo, f.runID)
			ignore(t, f, l)
			head := nested(t, f, l)
			got, declined := laneChangedFiles(f.repo.Root(), l, head)
			if declined != "" || !slices.Contains(got, "vendored") {
				t.Fatalf("the nested checkout is among what the lane changed: %q %s", got, declined)
			}
		})
	}
}

// landingMoved is each path the landing's records commit of l changed, read as
// stored, whatever stands in for it.
func landingMoved(t *testing.T, f *narrowFixture, l Lane) []string {
	t.Helper()
	if l.Landing == nil || !gitutil.IsFullSHA(l.Landing.Records) {
		t.Fatalf("premise: the landing made its records commit: %+v", l.Landing)
	}
	var out []string
	for _, p := range strings.Split(f.repo.Git("--no-replace-objects", "diff-tree", "--no-commit-id", "--no-renames", "--name-only", "-r", l.Landing.Records), "\n") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// TestTheFenceWaitsOutALostNetworkAtItsWidening: the network goes after the
// fence's renewal, so the path its widening reserves cannot be fetched for; the
// lane waits on the shared probe as at the fence's other reads, nothing pushed.
func TestTheFenceWaitsOutALostNetworkAtItsWidening(t *testing.T) {
	f := newNarrowFixture(t)
	holdNamesTheFoundAtAlone(t, f)
	dropped := false
	l, err := stepToThePush(t, f, func(l Lane) {
		if l.Landing != nil && l.Landing.RecordsDone && !dropped {
			dropped = true
			f.dropNetworkAfter(t, 1)
		}
	})
	rf, ok := AsRefusal(err)
	if !ok || !waitByDesign(rf) {
		t.Fatalf("the lane waits: %v", err)
	}
	if !strings.Contains(rf.Reason, "README.txt") {
		t.Fatalf("the wait names the path: %s", rf.Reason)
	}
	if l.Landing == nil || l.Landing.Pushed != "" {
		t.Fatalf("nothing is pushed: %+v", l.Landing)
	}
}
