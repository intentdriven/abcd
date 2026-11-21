//go:build unix

package loop

import (
	"bytes"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/gittest"
)

// The drive over a run with members (spc-2611201455051897 step 1, criterion
// 5): its let-go and its release for a usage limit let every issue go
// plainly; its take-over and take-up hold every one again, comparing every
// issue's record first; and a take-up that meets a refusal part way releases
// what it held again before it refuses.

// bundleDrainLane is a drain's lane for the bundle's lead, opened on the
// fixture's origin, given the two members as the drain's start of a bundle
// will (step 4), each reserved on origin with its own page.
func bundleDrainLane(t *testing.T) (repo *gittest.Repo, bare, runID string, c *clock) {
	t.Helper()
	repo, bare = bundleRepo(t)
	c = &clock{t: time.Now().UTC().Truncate(time.Second)}
	// The members are held back from the drain's own choice, so the lane it
	// opens is the lead's.
	first := drainToLane(t, repo, Options{Now: c.now}, DrainOptions{Skip: []string{bundleTwo, bundleThree}, SkipReason: "the lead's bundle"})
	if first.Lane == nil {
		t.Fatalf("the drain opens a lane: %+v", first)
	}
	runID = first.Lane.RunID
	st, err := ReadState(repo.Root(), runID)
	if err != nil {
		t.Fatal(err)
	}
	if st.Key != bundleLead {
		t.Fatalf("premise: the drain's lane is the lead's: %s", st.Key)
	}
	for _, id := range []string{bundleTwo, bundleThree} {
		if _, err := claim.Reserve(repo.Root(), claim.Request{Record: id, Footprint: []string{bundleFiles[id]}, Drain: true}, clockAt(c.t)); err != nil {
			t.Fatalf("reserving %s: %v", id, err)
		}
	}
	if err := mutate(repo.Root(), runID, func(_ *os.Root, st *State) (bool, error) {
		for _, id := range []string{bundleTwo, bundleThree} {
			when := c.t
			st.Bundle = append(st.Bundle, Member{Issue: id, Footprint: []string{bundleFiles[id]}, ReservedAt: &when})
			st.Record = append(st.Record, Entry{At: c.t, Stage: StageReservation, Note: "reserved " + id, Issue: id})
		}
		return true, nil
	}); err != nil {
		t.Fatal(err)
	}
	return repo, bare, runID, c
}

// wantLetGoPlainly checks that every issue of the run was let go plainly by
// the drive, each line naming its issue.
func wantLetGoPlainly(t *testing.T, repo *gittest.Repo, bare, runID string) State {
	t.Helper()
	for _, id := range bundleIssues {
		if r := tipOf(t, bare, id); !r.released {
			t.Fatalf("the drive lets %s go: %+v", id, r)
		}
		if kind, _ := tombstoneTrailers(t, bare, id); kind != "" {
			t.Fatalf("the drive lets %s go plainly: %q", id, kind)
		}
	}
	st, err := ReadState(repo.Root(), runID)
	if err != nil {
		t.Fatal(err)
	}
	if got := issuesOf(linesAt(st, StageDriveReleased)); !slices.Equal(got, bundleIssues) {
		t.Fatalf("each let-go line names its issue: %+v", linesAt(st, StageDriveReleased))
	}
	if got := issuesOf(linesAt(st, StageReleasedOver)); !slices.Equal(got, bundleIssues) {
		t.Fatalf("each issue's record is noted as it is let go: %+v", linesAt(st, StageReleasedOver))
	}
	return st
}

// TestTheDrivesLetGoReleasesEveryMemberPlainly: a drive that ends lets every
// issue of its lane's run go, plainly.
func TestTheDrivesLetGoReleasesEveryMemberPlainly(t *testing.T) {
	repo, bare, runID, c := bundleDrainLane(t)
	tm := newTerminal(repo.Root(), TerminalOptions{Options: Options{Now: c.now}})
	tm.letGo()
	wantLetGoPlainly(t, repo, bare, runID)
}

// TestALimitsReleaseLetsEveryMemberGoPlainly: a usage limit that lasts lets
// every issue of the lane's run go, plainly.
func TestALimitsReleaseLetsEveryMemberGoPlainly(t *testing.T) {
	repo, bare, runID, c := bundleDrainLane(t)
	tm := newTerminal(repo.Root(), TerminalOptions{Options: Options{Now: c.now}})
	tm.letGoForLimit(runID)
	if st := wantLetGoPlainly(t, repo, bare, runID); !letGoByDrive(st) {
		t.Fatalf("the run reads as a drive's letting go: %+v", st.Record)
	}
}

// wantHeldAgain checks that every issue of the run is held again from this
// copy, live, and the run's record names each.
func wantHeldAgain(t *testing.T, repo *gittest.Repo, bare, runID string, now time.Time) {
	t.Helper()
	for _, id := range bundleIssues {
		r := reservationOfRoot(t, repo.Root(), id, now)
		if r.State != claim.StateLive || r.Copy != claim.CopyThis {
			t.Fatalf("%s is held again from this copy: %+v", id, r)
		}
		if tip := tipOf(t, bare, id); tip.released {
			t.Fatalf("%s's tip is a reservation again: %+v", id, tip)
		}
	}
	st, err := ReadState(repo.Root(), runID)
	if err != nil {
		t.Fatal(err)
	}
	var last []string
	for _, e := range st.Record {
		if e.Stage == StageReservation && e.At.Equal(now) {
			last = append(last, e.Issue)
		}
	}
	if !slices.Equal(last, bundleIssues) {
		t.Fatalf("each issue's reservation line names it, in Issues() order: %v", last)
	}
	if st.ReservedAt == nil || slices.ContainsFunc(st.Bundle, func(m Member) bool { return m.ReservedAt == nil }) {
		t.Fatalf("the run records a reservation on every issue: %v %+v", st.ReservedAt, st.Bundle)
	}
}

// reservationOfRoot is the reservation on id in root's view, fetched first.
func reservationOfRoot(t *testing.T, root, id string, now time.Time) claim.Reservation {
	t.Helper()
	if _, err := claim.Fetch(root, claim.Options{}); err != nil {
		t.Fatal(err)
	}
	snap, err := claim.View(root, now)
	if err != nil {
		t.Fatal(err)
	}
	r, ok := snap.Lookup(id)
	if !ok {
		t.Fatalf("no reservation of %s", id)
	}
	return r
}

// TestATakeUpHoldsEveryMemberAgain: a drive's take-up of a run it let go for
// a limit holds every issue again, each with its own page.
func TestATakeUpHoldsEveryMemberAgain(t *testing.T) {
	repo, bare, runID, c := bundleDrainLane(t)
	tm := newTerminal(repo.Root(), TerminalOptions{Options: Options{Now: c.now}})
	tm.letGoForLimit(runID)
	c.t = c.t.Add(time.Minute)
	if err := reserveResumed(repo.Root(), &StartResult{RunID: runID}, tm.takeUpOptions()); err != nil {
		t.Fatalf("the take-up holds every issue again: %v", err)
	}
	wantHeldAgain(t, repo, bare, runID, c.t)
	for _, id := range bundleIssues {
		if fp := footprintOf(t, bare, id); !slices.Equal(fp, []string{bundleFiles[id]}) {
			t.Fatalf("%s is held again with its own page: %v", id, fp)
		}
	}
}

// TestATakeOverHoldsEveryMemberAgain: a take-over of a run a stop let go holds
// every issue again.
func TestATakeOverHoldsEveryMemberAgain(t *testing.T) {
	repo, bare, runID, c := bundleDrainLane(t)
	o, _, _ := stopped(t)
	o.Now = c.now
	wantStopRefusal(t, checkOnce(repo.Root(), runID, o))
	for _, id := range bundleIssues {
		if !tipOf(t, bare, id).released {
			t.Fatalf("premise: the stop let %s go", id)
		}
	}
	c.t = c.t.Add(time.Minute)
	if err := reserveTaken(repo.Root(), &StartResult{RunID: runID}, Options{Now: c.now}, false); err != nil {
		t.Fatalf("the take-over holds every issue again: %v", err)
	}
	wantHeldAgain(t, repo, bare, runID, c.t)
}

// TestATakeUpWhoseMembersRecordMovedIsRefusedNamingIt: a member's record
// changed on origin's default branch while the run was let go, so the drive's
// take-up is refused naming the member, as the lead's would be, before
// anything is pushed.
func TestATakeUpWhoseMembersRecordMovedIsRefusedNamingIt(t *testing.T) {
	repo, bare, runID, c := bundleDrainLane(t)
	tm := newTerminal(repo.Root(), TerminalOptions{Options: Options{Now: c.now}})
	tm.letGoForLimit(runID)
	changeRecordOnOrigin(t, bare, bundleThree, "open")
	before := tipsOf(t, bare)
	c.t = c.t.Add(time.Minute)
	err := reserveResumed(repo.Root(), &StartResult{RunID: runID}, tm.takeUpOptions())
	r, ok := AsRefusal(err)
	if !ok || !r.recordMoved || !strings.Contains(r.Reason, bundleThree+"'s record on origin's default branch changed while the issue was let go") {
		t.Fatalf("the take-up is refused naming the member whose record moved: %v", err)
	}
	for id, tip := range tipsOf(t, bare) {
		if tip.commit != before[id].commit {
			t.Fatalf("the take-up pushed %s: %+v", id, tip)
		}
	}
}

// TestATakeUpThatBacksOffOnItsThirdMemberReleasesWhatItHeldAgain: another
// person reserved the third member while the run was let go, so the take-up,
// having held the lead and the first member again, backs off on the third:
// it releases plainly the two it held, then refuses naming the third, and the
// run's state is as it was, holding nothing.
func TestATakeUpThatBacksOffOnItsThirdMemberReleasesWhatItHeldAgain(t *testing.T) {
	repo, bare, runID, c := bundleDrainLane(t)
	tm := newTerminal(repo.Root(), TerminalOptions{Options: Options{Now: c.now}})
	tm.letGoForLimit(runID)
	q := otherCopy(t, bare, quinnEmail)
	if _, err := claim.Reserve(q, claim.Request{Record: bundleThree, Footprint: []string{"docs/c.md"}}, as(t, q, clockAt(c.t))); err != nil {
		t.Fatalf("Q reserves the third member: %v", err)
	}
	theirs := tipOf(t, bare, bundleThree)
	state := stateBytes(t, repo.Root(), runID)
	c.t = c.t.Add(time.Minute)
	err := reserveResumed(repo.Root(), &StartResult{RunID: runID}, tm.takeUpOptions())
	r, ok := AsRefusal(err)
	if !ok || !r.Contention || r.Stage != StageReservation || !strings.Contains(r.Reason, "stopped at "+bundleThree) ||
		!strings.Contains(r.Reason, bundleLead) || !strings.Contains(r.Reason, bundleTwo) {
		t.Fatalf("the take-up refuses naming the third member and what it released: %v", err)
	}
	for _, id := range []string{bundleLead, bundleTwo} {
		tip := tipOf(t, bare, id)
		held := reservationCommit(t, bare, tip.parent)
		if !tip.released || held.released || !held.claimed.Equal(c.t) {
			t.Fatalf("%s was held again and then released plainly: tip %+v, under it %+v", id, tip, held)
		}
		if kind, _ := tombstoneTrailers(t, bare, id); kind != "" {
			t.Fatalf("%s is released plainly: %q", id, kind)
		}
	}
	if got := tipOf(t, bare, bundleThree); got.commit != theirs.commit {
		t.Fatalf("Q's hold on the third member stands: %+v", got)
	}
	if after := stateBytes(t, repo.Root(), runID); !bytes.Equal(after, state) {
		t.Fatalf("the run's state is unchanged:\nbefore %s\nafter %s", state, after)
	}
}

// TestDropMovedRunTriesAgainOnlyTheIssueWhoseLetGoDidNotReachOrigin: the
// let-go of the first member did not reach origin while the lead's and the
// third's did, so dropping the run tries the first member's release again,
// over the tip it left, and leaves the others as they stand.
func TestDropMovedRunTriesAgainOnlyTheIssueWhoseLetGoDidNotReachOrigin(t *testing.T) {
	repo, bare, runID, c := bundleDrainLane(t)
	undo := refuseReservationPush(t, bare, bundleTwo)
	tm := newTerminal(repo.Root(), TerminalOptions{Options: Options{Now: c.now}})
	tm.letGoForLimit(runID)
	undo()
	run, err := ReadState(repo.Root(), runID)
	if err != nil {
		t.Fatal(err)
	}
	if failedReleaseTipFor(run, bundleTwo) == "" || failedReleaseTipFor(run, bundleLead) != "" || failedReleaseTipFor(run, bundleThree) != "" {
		t.Fatalf("premise: only the first member's let-go failed: %+v", run.Record)
	}
	before := tipsOf(t, bare)
	if before[bundleTwo].released {
		t.Fatalf("premise: the first member's hold stands: %+v", before[bundleTwo])
	}
	why, err := dropMovedRun(repo.Root(), run, "the record moved", c.t, Options{Now: c.now})
	if err != nil {
		t.Fatal(err)
	}
	after := tipsOf(t, bare)
	if !after[bundleTwo].released || after[bundleTwo].parent != before[bundleTwo].commit {
		t.Fatalf("the first member's release is tried again over its tip: %+v", after[bundleTwo])
	}
	for _, id := range []string{bundleLead, bundleThree} {
		if after[id].commit != before[id].commit {
			t.Fatalf("%s, already let go, is left as it stands: %+v", id, after[id])
		}
	}
	if !strings.Contains(why, bundleTwo) {
		t.Fatalf("the reason says what came of the first member's release: %q", why)
	}
}

// TestThePeerCheckAfterAWaitReadsEveryMember: the drive's check after a long
// wait finds a member's hold gone to another copy while the lead's still
// stands, and stops the lane with a refusal naming the member.
func TestThePeerCheckAfterAWaitReadsEveryMember(t *testing.T) {
	repo, bare := bundleRepo(t)
	now := time.Now().UTC().Truncate(time.Second)
	runID := bundleRun(t, repo, now.Add(-30*time.Minute))
	p2 := otherCopy(t, bare, patEmail)
	if res, err := claim.Reserve(p2, claim.Request{Record: bundleThree, Footprint: []string{"docs/c.md"}, TakeOver: true}, clockAt(now.Add(-time.Minute))); err != nil || res.Act != claim.ActTakenOver {
		t.Fatalf("P's other copy takes the third member over: %+v %v", res, err)
	}
	tm := newTerminal(repo.Root(), TerminalOptions{Options: Options{Now: func() time.Time { return now }}})
	tm.waited = 2 * time.Hour
	if !tm.peerTook(runID) {
		t.Fatal("a member's hold gone to another copy stops the lane")
	}
	run, err := ReadState(repo.Root(), runID)
	if err != nil {
		t.Fatal(err)
	}
	if i := run.current(); i < 0 || run.Lanes[i].Refused == nil || !strings.Contains(run.Lanes[i].Refused.Reason, bundleThree) {
		t.Fatalf("the refusal names the member: %+v", run.Lanes)
	}
}
