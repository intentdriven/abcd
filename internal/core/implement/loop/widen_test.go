package loop

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// An issue lane widens its hold before it edits a path beyond it
// (iss-2610311224444576; dcn-2610311251002713, dcn-2610311544325397): the
// implementer runs `abcd implement widen` naming each further path before its
// first edit there, which reserves the issue again with the held footprint
// and those paths, judged for overlap, and backs off (exit 3, nothing
// changed) when another copy holds one live. An implementer's receipt is
// refused while the lane's branch changes a path its reservation does not
// cover, and a lane leaving its landing for a fresh round keeps the paths it
// widened to.

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

// TestAReceiptIsRefusedWhileTheBranchChangesAPathItsHoldDoesNotCover: the
// implementer changed README.txt without widening; the receipt is refused,
// naming the path and the widen step, the lane stays at its implement stage,
// and once the hold is widened the same receipt verifies.
func TestAReceiptIsRefusedWhileTheBranchChangesAPathItsHoldDoesNotCover(t *testing.T) {
	f := newNarrowFixture(t)
	err := f.handBack(t, Options{}, "docs/page.md", "README.txt")
	r, ok := AsRefusal(err)
	if !ok || r.Contention {
		t.Fatalf("the receipt is refused: %v", err)
	}
	if !strings.Contains(r.Reason, "README.txt") || !strings.Contains(r.Reason, "implement widen") || strings.Contains(r.Reason, "docs/page.md") {
		t.Fatalf("the refusal names the unheld path alone and the widen step: %s", r.Reason)
	}
	l := currentLane(t, f.repo, f.runID)
	if l.Stage != StageImplement {
		t.Fatalf("the lane stays at its implement stage: %s", l.Stage)
	}
	if _, err := f.widen(t, Options{}, "README.txt"); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(f.repo.Root(), filepath.FromSlash(RunRelDir), f.runID, l.ID)
	if _, err := Receipt(f.repo.Root(), f.runID, filepath.Join(dir, ReceiptFileName), f.stages, Options{}); err != nil {
		t.Fatalf("once widened, the receipt verifies: %v", err)
	}
	if l := currentLane(t, f.repo, f.runID); l.Stage != StageValidate {
		t.Fatalf("the lane moves to its validators: %s", l.Stage)
	}
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

// TestAnIssueBriefNamesTheWideningStep: the lane's brief tells the
// implementer to widen the hold before editing a file beyond the found_at,
// with the exact command for its run and lane, from the run's checkout, and
// what each exit means.
func TestAnIssueBriefNamesTheWideningStep(t *testing.T) {
	f := newNarrowFixture(t)
	f.awaitImplementerOnce(t)
	l := currentLane(t, f.repo, f.runID)
	brief, err := os.ReadFile(filepath.Join(f.repo.Root(), filepath.FromSlash(l.Brief)))
	if err != nil {
		t.Fatal(err)
	}
	want := " implement widen --run " + f.runID + " --lane " + l.ID + " <path>...)"
	for _, s := range []string{"## Files beyond the issue's place", "(cd '" + f.repo.Root() + "' && ", want, "Exit 3 naming a reservation",
		"write no receipt", "no `widen` sub-verb"} {
		if !strings.Contains(string(brief), s) {
			t.Fatalf("the brief carries %q:\n%s", s, brief)
		}
	}
}

// widenAsImplementer widens, as the lane's brief tells its implementer to,
// the hold of the issue run whose current lane is l to names before they are
// edited: a fixture's stand-in for that step. It does nothing for an intent's
// run, a run holding no reservation, or a lane at no editing stage.
func widenAsImplementer(t *testing.T, repo *gittest.Repo, l Lane, names ...string) {
	t.Helper()
	runs, err := Runs(repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	for _, st := range runs {
		i := st.current()
		if st.Issue() == "" || st.ReservedAt == nil || i < 0 || st.Lanes[i].ID != l.ID || st.Lanes[i].Worktree != l.Worktree ||
			!slices.Contains(widenStages, st.Lanes[i].Stage) {
			continue
		}
		if _, err := Widen(repo.Root(), st.RunID, l.ID, names, Options{}); err != nil {
			t.Fatalf("widening the hold to %q before the edit: %v", names, err)
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
// released while the implementer worked; the receipt names no hold to widen,
// since the widen step would refuse it, and the land fence settles the loss.
func TestTheBackstopIgnoresAReleasedHold(t *testing.T) {
	f := newNarrowFixture(t)
	f.awaitImplementerOnce(t)
	releaseKey(f.repo.Root(), secondIssue, time.Now())
	err := f.handBack(t, Options{}, "docs/page.md", "README.txt")
	if r, ok := AsRefusal(err); ok && strings.Contains(r.Reason, "implement widen") {
		t.Fatalf("a released hold sends the implementer to no widen step: %s", r.Reason)
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

// TestTheBackstopLeavesOutTheLandingsOwnRecords: the landing's records commit
// on the lane's branch moves the issue's record, a path the hold never names,
// and a lane leaving its landing after it (a pull request that started to
// conflict) is handed to an implementer again; that implementer's receipt is
// not refused for the loop's own records, while an edit of its own beyond
// the hold still is.
func TestTheBackstopLeavesOutTheLandingsOwnRecords(t *testing.T) {
	f := newNarrowFixture(t)
	if err := f.handBack(t, Options{}, "docs/page.md"); err != nil {
		t.Fatal(err)
	}
	l := currentLane(t, f.repo, f.runID)
	record := ".abcd/work/issues/resolved/" + secondIssue + "-the-record.md"
	records := commitIn(t, f.repo, l, record)
	gaps := func() []string {
		t.Helper()
		runs, err := Runs(f.repo.Root())
		if err != nil {
			t.Fatal(err)
		}
		for _, st := range runs {
			if st.RunID != f.runID {
				continue
			}
			lane := st.Lanes[st.current()]
			lane.Landing = &Landing{Records: records, RecordsDone: true}
			return unheldGaps(Context{RepoRoot: f.repo.Root(), State: st, Now: time.Now()}, lane, time.Now())
		}
		t.Fatalf("no run %s", f.runID)
		return nil
	}
	if g := gaps(); len(g) != 0 {
		t.Fatalf("the landing's own records are not the implementer's edits: %q", g)
	}
	commitIn(t, f.repo, l, "README.txt")
	g := gaps()
	if len(g) != 1 || !strings.Contains(g[0], "README.txt") || strings.Contains(g[0], record) {
		t.Fatalf("an edit beyond the hold after the records is still named, the records left out: %q", g)
	}
}

// holdNamesTheFoundAtAlone implements the lane with README.txt widened to, then
// makes the reservation name the found_at alone, as a reservation made again
// over a deleted tip by an earlier abcd did, leaving the lane at its validators.
func holdNamesTheFoundAtAlone(t *testing.T, f *narrowFixture) {
	t.Helper()
	if _, err := f.widen(t, Options{}, "README.txt"); err != nil {
		t.Fatal(err)
	}
	if err := f.handBack(t, Options{}, "docs/page.md", "README.txt"); err != nil {
		t.Fatal(err)
	}
	if l := currentLane(t, f.repo, f.runID); l.Stage != StageValidate {
		t.Fatalf("the lane is at its validators: %s", l.Stage)
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
	setup      func(t *testing.T, f *narrowFixture)
	before     func(t *testing.T, f *narrowFixture) func(Lane)
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
		{name: "left uncommitted before its records", path: "NOTES.md", setup: onlyThePage,
			before: func(t *testing.T, _ *narrowFixture) func(Lane) { return leftInTheWorktree(t, "NOTES.md") }},
		{name: "in a commit under the landing's subject", path: "NOTES.txt", setup: onlyThePage,
			before: func(t *testing.T, f *narrowFixture) func(Lane) {
				return committedUnderTheLandingsSubject(t, f, "NOTES.txt")
			}},
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
// lane stops at the fence before its push, naming the hold, with nothing pushed
// and its reservation as it was.
func TestALaneDoesNotLandAPathAPeerHolds(t *testing.T) {
	for _, c := range fenceCases() {
		t.Run(c.name, func(t *testing.T) {
			f := newNarrowFixture(t)
			c.setup(t, f)
			f.reserveAs(t, "iss-99", "quinn", c.path, time.Now().Add(-time.Minute))
			l, err := stepToThePush(t, f, c.hook(t, f))
			rf, ok := AsRefusal(err)
			if !ok || !rf.Contention {
				t.Fatalf("the lane stops as contention: %v (%+v)", err, l.Landing)
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
// left out as the move the landing makes.
func TestALaneReservesAPathNoReservationCoversBeforeItLands(t *testing.T) {
	for _, c := range fenceCases() {
		t.Run(c.name, func(t *testing.T) {
			f := newNarrowFixture(t)
			c.setup(t, f)
			l, err := stepToThePush(t, f, c.hook(t, f))
			if err != nil {
				t.Fatal(err)
			}
			if got := gitIn(t, f.bare, "for-each-ref", "refs/heads/"+l.Branch); got == "" {
				t.Fatalf("the lane's branch is pushed: %+v", l.Landing)
			}
			want := []string{c.path, "docs"}
			slices.Sort(want)
			if fp := sortedFootprint(t, f); !slices.Equal(fp, want) {
				t.Fatalf("the reservation names the path before the push: %q", fp)
			}
		})
	}
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
