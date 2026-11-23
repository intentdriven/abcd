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
)

// The receipt of a run with members (itd-2611201450052326,
// spc-2611201455051897 step 2, criterion 6): it names every issue the run
// still holds once, in `resolves` or in `handbacks`; an issue handed back
// alone loses its resolution and is released with its own tombstone while
// the lane goes on; and a lane whose every issue is handed back is handed back
// with each issue released once, by its own kind.

// bundleAwaiting is a run of the three bundle issues, each reserved on the
// fixture's origin, driven to its implement await. It returns the repository,
// its bare origin, the run, the lane and the lane's directory.
func bundleAwaiting(t *testing.T) (*gittest.Repo, string, string, Lane, string) {
	t.Helper()
	repo, bare := bundleRepo(t)
	runID := bundleRun(t, repo, time.Now().UTC().Truncate(time.Second))
	advanceTo(t, repo, runID, StageImplement)
	if res, err := advance(repo.Root(), runID, DefaultStages(), Options{}); err != nil || res.Awaiting == nil {
		t.Fatalf("the implement stage awaits an implementer: %+v %v", res, err)
	}
	l := currentLane(t, repo, runID)
	return repo, bare, runID, l, filepath.Join(repo.Root(), filepath.FromSlash(RunRelDir), runID, l.ID)
}

// bundleFix commits a fix to id's page in the lane's worktree and returns the
// commit.
func bundleFix(t *testing.T, repo *gittest.Repo, l Lane, id string) string {
	t.Helper()
	return laneCommitAt(t, repo, l, bundleFiles[id], "corrected for "+id+"\n")
}

// laneCommitAt commits body at rel in the lane's worktree and returns the
// commit.
func laneCommitAt(t *testing.T, repo *gittest.Repo, l Lane, rel, body string) string {
	t.Helper()
	p := filepath.Join(l.Worktree, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	repo.Git("-C", l.Worktree, "add", "--", rel)
	repo.Git("-C", l.Worktree, "commit", "-q", "-m", "lane: "+rel)
	return strings.TrimSpace(repo.Git("-C", l.Worktree, "rev-parse", "HEAD"))
}

// resolving is a resolution of id by commit.
func resolving(id, commit string) Resolution {
	return Resolution{Issue: id, Commit: commit, Note: "the sentence is corrected", Impact: "fix",
		Grounds: "pursued: the page reads true; shown wrong if the test still finds the old sentence"}
}

// handingBack is a hand-back of id as a user-visible change.
func handingBack(id string) IssueHandBack {
	return IssueHandBack{Issue: id, Kind: HandBackUserVisible, Reason: "the correction changes what a reader of the page is told"}
}

// TestAReceiptResolvingTwoAndHandingOneBackIsTaken: a receipt that resolves
// the lead and one member and hands the other back is taken; the member
// handed back is recorded on the run and is no issue of it, its reservation
// is released with a tombstone of its kind and a line naming it, and the two
// resolved stay held for the lane, which goes on to its validators.
func TestAReceiptResolvingTwoAndHandingOneBackIsTaken(t *testing.T) {
	repo, bare, runID, l, dir := bundleAwaiting(t)
	a, b := bundleFix(t, repo, l, bundleLead), bundleFix(t, repo, l, bundleTwo)
	rc := goodReceipt(t, runID, l, dir, a, b)
	rc.Resolves = []Resolution{resolving(bundleLead, a), resolving(bundleTwo, b)}
	rc.HandBacks = []IssueHandBack{handingBack(bundleThree)}
	res, err := Receipt(repo.Root(), runID, writeReceipt(t, dir, rc), DefaultStages(), Options{})
	if err != nil {
		t.Fatalf("the receipt is taken: %v", err)
	}
	st, _ := ReadState(repo.Root(), runID)
	if got := st.Issues(); !slices.Equal(got, []string{bundleLead, bundleTwo}) {
		t.Fatalf("the member handed back is no issue of the run: %v", got)
	}
	if m := st.Bundle[st.member(bundleThree)]; m.HandBack == nil || m.HandBack.Kind != HandBackUserVisible || !strings.Contains(m.HandBack.Reason, "reader of the page") {
		t.Fatalf("the member's hand-back is recorded on it: %+v", m)
	}
	if lane := st.Lanes[0]; lane.Stage != StageValidate || len(lane.Resolves) != 2 || lane.HandBack != nil {
		t.Fatalf("the lane goes on to its validators with the two resolutions: %+v", lane)
	}
	if kind, _ := tombstoneTrailers(t, bare, bundleThree); kind != HandBackUserVisible {
		t.Fatalf("the member is tombstoned with its own kind: %q", kind)
	}
	for _, id := range []string{bundleLead, bundleTwo} {
		if r := tipOf(t, bare, id); r.released {
			t.Fatalf("%s is still held for the lane: %+v", id, r)
		}
	}
	if got := issuesOf(linesAt(st, StageReservationReleased)); !slices.Equal(got, []string{bundleThree}) {
		t.Fatalf("one release line, naming the member: %v", got)
	}
	if !strings.Contains(res.Next, bundleThree+" is handed back alone") {
		t.Fatalf("the answer says the member is handed back alone: %q", res.Next)
	}
}

// TestABundleReceiptsAccountOfItsIssuesIsChecked: in a run with members a
// receipt that leaves an issue out, names one in both lists, names an issue
// outside the run, or carries a top-level handback is refused naming the gap;
// in a run without members `handbacks` is refused, and a receipt still
// resolves another capture beside the lane's issue.
func TestABundleReceiptsAccountOfItsIssuesIsChecked(t *testing.T) {
	repo, _, runID, l, dir := bundleAwaiting(t)
	a, b, c := bundleFix(t, repo, l, bundleLead), bundleFix(t, repo, l, bundleTwo), bundleFix(t, repo, l, bundleThree)
	const outside = "iss-2611201500000009"
	for name, tc := range map[string]struct {
		mut  func(*LaneReceipt)
		want string
	}{
		"in neither list": {func(rc *LaneReceipt) { rc.Resolves = rc.Resolves[:2] },
			bundleThree + ", an issue of the run, in resolves or in handbacks"},
		"in both lists": {func(rc *LaneReceipt) { rc.HandBacks = []IssueHandBack{handingBack(bundleThree)} },
			bundleThree + " in resolves or in handbacks, not both"},
		"not a member": {func(rc *LaneReceipt) { rc.Resolves = append(rc.Resolves, resolving(outside, c)) },
			outside + " in neither resolves nor handbacks: it is not an issue of the run"},
		"a top-level handback": {func(rc *LaneReceipt) {
			rc.HandBack = &LaneHandBack{Kind: HandBackUserVisible, Reason: "the change shows"}
		}, "no handback in a run with members"},
		"a hand-back without its home": {func(rc *LaneReceipt) {
			rc.Resolves = rc.Resolves[:2]
			rc.HandBacks = []IssueHandBack{{Issue: bundleThree, Kind: HandBackDesignFinding, Reason: "a design call"}}
		}, "handbacks[0]: " + bundleThree + "'s home for kind design-finding"},
	} {
		t.Run(name, func(t *testing.T) {
			rc := goodReceipt(t, runID, l, dir, a, b, c)
			rc.Resolves = []Resolution{resolving(bundleLead, a), resolving(bundleTwo, b), resolving(bundleThree, c)}
			tc.mut(&rc)
			_, err := Receipt(repo.Root(), runID, writeReceipt(t, dir, rc), DefaultStages(), Options{})
			if r := mustRefusal(t, err); !strings.Contains(r.Reason, tc.want) {
				t.Fatalf("refused naming %q: %s", tc.want, r.Reason)
			}
		})
	}
	if st, _ := ReadState(repo.Root(), runID); st.Lanes[0].Stage != StageImplement || len(st.Issues()) != 3 {
		t.Fatalf("a refused receipt changes nothing: %+v", st.Lanes[0])
	}

	// A run without members takes no handbacks, and still resolves another
	// capture beside its issue.
	repo2, runID2, l2, dir2 := issueAwaiting(t)
	sha := laneCommit(t, repo2, l2, "fix.txt")
	rc := goodReceipt(t, runID2, l2, dir2, sha)
	rc.Resolves = []Resolution{resolving(eligibleIssue, sha)}
	rc.HandBacks = []IssueHandBack{handingBack(majorIssue)}
	_, err := Receipt(repo2.Root(), runID2, writeReceipt(t, dir2, rc), DefaultStages(), Options{})
	if r := mustRefusal(t, err); !strings.Contains(r.Reason, "no handbacks in a run without members") {
		t.Fatalf("a run without members refuses handbacks: %s", r.Reason)
	}
	rc.HandBacks = nil
	rc.Resolves = append(rc.Resolves, resolving(majorIssue, sha))
	if _, err := Receipt(repo2.Root(), runID2, writeReceipt(t, dir2, rc), DefaultStages(), Options{}); err != nil {
		t.Fatalf("a single lane's receipt still resolves another capture: %v", err)
	}
}

// TestAMemberHandedBackAfterAnEarlierResolutionLeavesTheBranchFirst: a
// receipt that hands back a member an earlier receipt resolved (a fix
// round's, as the lane's resolutions stand after round 1) is refused while
// that member's earlier commit is reachable from the lane's branch, naming
// it, and taken once the branch no longer carries it, the member's
// resolution taken off the lane so the landing writes no `Resolves:` for it.
func TestAMemberHandedBackAfterAnEarlierResolutionLeavesTheBranchFirst(t *testing.T) {
	repo, _, runID, l, dir := bundleAwaiting(t)
	a, b := bundleFix(t, repo, l, bundleLead), bundleFix(t, repo, l, bundleTwo)
	c := bundleFix(t, repo, l, bundleThree)
	// Round 1 resolved all three.
	if err := mutate(repo.Root(), runID, func(_ *os.Root, st *State) (bool, error) {
		st.Lanes[0].Resolves = []Resolution{resolving(bundleLead, a), resolving(bundleTwo, b), resolving(bundleThree, c)}
		return true, nil
	}); err != nil {
		t.Fatal(err)
	}
	rc := goodReceipt(t, runID, l, dir, a, b)
	rc.Resolves = []Resolution{resolving(bundleLead, a), resolving(bundleTwo, b)}
	rc.HandBacks = []IssueHandBack{handingBack(bundleThree)}
	_, err := Receipt(repo.Root(), runID, writeReceipt(t, dir, rc), DefaultStages(), Options{})
	if r := mustRefusal(t, err); !strings.Contains(r.Reason, bundleThree+"'s change off the lane's branch") || !strings.Contains(r.Reason, shortSHA(c)) {
		t.Fatalf("refused naming the member's earlier commit: %s", r.Reason)
	}
	// The implementer takes the member's commit out of the branch.
	repo.Git("-C", l.Worktree, "reset", "-q", "--hard", b)
	if _, err := Receipt(repo.Root(), runID, writeReceipt(t, dir, rc), DefaultStages(), Options{}); err != nil {
		t.Fatalf("taken once the commit is off the branch: %v", err)
	}
	st, _ := ReadState(repo.Root(), runID)
	if slices.ContainsFunc(st.Lanes[0].Resolves, func(r Resolution) bool { return r.Issue == bundleThree }) || len(st.Lanes[0].Resolves) != 2 {
		t.Fatalf("the member's resolution is taken off the lane: %+v", st.Lanes[0].Resolves)
	}
	if got := resolutionTrailers(st, st.Lanes[0]); slices.Contains(got, "Resolves: "+bundleThree) || !slices.Contains(got, "Refs: "+bundleThree) {
		t.Fatalf("the landing writes no Resolves: for it, and a Refs: in its place: %v", got)
	}
}

// TestTheLeadHandedBackAloneMovesItsWidenedPathsOntoTheNextMember: the lead
// handed back alone while its hold carries a path the lane widened to has that
// path widened onto the next member's hold before the lead's tombstone, and
// the receipt's own widening of what the branch changes beyond the holds goes
// onto that member's hold, not the lead's.
func TestTheLeadHandedBackAloneMovesItsWidenedPathsOntoTheNextMember(t *testing.T) {
	repo, bare, runID, l, dir := bundleAwaiting(t)
	test := laneCommitAt(t, repo, l, "docs/pin_test.md", "pins the corrected sentences\n")
	// The loop widened the lead's hold to the pinning test.
	c := bundleContext(t, repo, runID, time.Now().UTC())
	c.State.Lanes[0] = l
	h := widenLaneChanges(c, l)
	if h.err != nil || h.key != bundleLead || !slices.Equal(h.paths, []string{"docs/pin_test.md"}) {
		t.Fatalf("premise: the lead's hold is widened to the test: %+v", h)
	}
	if err := mutate(repo.Root(), runID, func(_ *os.Root, st *State) (bool, error) {
		st.Record = append(st.Record, h.lines...)
		return true, nil
	}); err != nil {
		t.Fatal(err)
	}
	b, d := bundleFix(t, repo, l, bundleTwo), bundleFix(t, repo, l, bundleThree)
	more := laneCommitAt(t, repo, l, "docs/more_test.md", "pins one more\n")
	rc := goodReceipt(t, runID, l, dir, test, b, d, more)
	rc.Resolves = []Resolution{resolving(bundleTwo, b), resolving(bundleThree, d)}
	rc.HandBacks = []IssueHandBack{handingBack(bundleLead)}
	if _, err := Receipt(repo.Root(), runID, writeReceipt(t, dir, rc), DefaultStages(), Options{}); err != nil {
		t.Fatalf("the receipt handing the lead back is taken: %v", err)
	}
	if fp := footprintOf(t, bare, bundleTwo); !slices.Contains(fp, "docs/pin_test.md") || !slices.Contains(fp, "docs/more_test.md") {
		t.Fatalf("the widened path moves onto the next member, and the later widening goes onto its hold: %v", fp)
	}
	if kind, _ := tombstoneTrailers(t, bare, bundleLead); kind != HandBackUserVisible {
		t.Fatalf("the lead is tombstoned with its own kind: %q", kind)
	}
	st, _ := ReadState(repo.Root(), runID)
	moved, released := -1, -1
	for i, e := range st.Record {
		if e.Stage == StageWidenedTo && e.Note == "docs/pin_test.md" && e.Issue == bundleTwo && moved < 0 {
			moved = i
		}
		if e.Stage == StageReservationReleased && e.Issue == bundleLead {
			released = i
		}
	}
	if moved < 0 || released < 0 || moved > released {
		t.Fatalf("the path is moved before the lead's release (%d, %d): %+v", moved, released, st.Record)
	}
	if st.LeadHandBack == nil || widenKey(st) != bundleTwo {
		t.Fatalf("the lead is handed back alone, and the next widening widens %s's hold: %+v", bundleTwo, st.LeadHandBack)
	}
}

// TestAMemberReleasedAloneIsNotReleasedAgain: a member released alone is not
// released again when the lane is later handed back whole, its release line
// stops no other issue's release, and each other issue is released once.
func TestAMemberReleasedAloneIsNotReleasedAgain(t *testing.T) {
	repo, bare, runID, l, dir := bundleAwaiting(t)
	a, b := bundleFix(t, repo, l, bundleLead), bundleFix(t, repo, l, bundleTwo)
	rc := goodReceipt(t, runID, l, dir, a, b)
	rc.Resolves = []Resolution{resolving(bundleLead, a), resolving(bundleTwo, b)}
	rc.HandBacks = []IssueHandBack{handingBack(bundleThree)}
	if _, err := Receipt(repo.Root(), runID, writeReceipt(t, dir, rc), DefaultStages(), Options{}); err != nil {
		t.Fatal(err)
	}
	st, _ := ReadState(repo.Root(), runID)
	if !releasedThisHoldFor(st, bundleThree) || releasedThisHoldFor(st, bundleLead) || releasedThisHoldFor(st, bundleTwo) || releasedThisHold(st) {
		t.Fatal("the member's release is its own: the run's hold is not released while the lead and the other member are held")
	}
	releaseOnHandBack(repo.Root(), &st, HandBack{Kind: HandBackStuck}, time.Now().UTC())
	got := issuesOf(linesAt(st, StageReservationReleased))
	if !slices.Equal(got, []string{bundleThree, bundleLead, bundleTwo}) {
		t.Fatalf("each issue is released once, the member never again: %v", got)
	}
	for _, id := range []string{bundleLead, bundleTwo} {
		if kind, _ := tombstoneTrailers(t, bare, id); kind != HandBackStuck {
			t.Fatalf("%s is tombstoned with the lane's kind: %q", id, kind)
		}
	}
	if kind, _ := tombstoneTrailers(t, bare, bundleThree); kind != HandBackUserVisible {
		t.Fatalf("the member keeps its own tombstone: %q", kind)
	}
	if !releasedThisHold(st) {
		t.Fatal("every issue's hold is released")
	}
}

// TestEveryIssueHandedBackHandsTheLaneBack: a receipt that hands back every
// issue of the run hands the lane back, its work discarded, and each issue is
// tombstoned once, by its own kind.
func TestEveryIssueHandedBackHandsTheLaneBack(t *testing.T) {
	repo, bare, runID, l, dir := bundleAwaiting(t)
	a := bundleFix(t, repo, l, bundleLead)
	rc := goodReceipt(t, runID, l, dir, a)
	rc.DefinitionOfDone = nil
	kinds := map[string]IssueHandBack{
		bundleLead:  handingBack(bundleLead),
		bundleTwo:   {Issue: bundleTwo, Kind: HandBackDesignFinding, Reason: "a design call", Home: "the brief"},
		bundleThree: {Issue: bundleThree, Kind: HandBackSecondPackage, Reason: "it reaches another package", Home: "an intent"},
	}
	rc.HandBacks = []IssueHandBack{kinds[bundleTwo], kinds[bundleLead], kinds[bundleThree]}
	res, err := Receipt(repo.Root(), runID, writeReceipt(t, dir, rc), DefaultStages(), Options{})
	if err != nil {
		t.Fatalf("a receipt handing back every issue is taken: %v", err)
	}
	st, _ := ReadState(repo.Root(), runID)
	lane := st.Lanes[0]
	if lane.Stage != StageHandedBack || lane.HandBack == nil || lane.HandBack.Kind != HandBackUserVisible || lane.HandBack.Discarded != a || res.HandBack == nil {
		t.Fatalf("the lane is handed back with the lead's hand-back, its work discarded: %+v", lane.HandBack)
	}
	if len(st.Issues()) != 0 {
		t.Fatalf("no issue is left to the run: %v", st.Issues())
	}
	for id, hb := range kinds {
		if kind, _ := tombstoneTrailers(t, bare, id); kind != hb.Kind {
			t.Fatalf("%s is tombstoned by its own kind %s: %q", id, hb.Kind, kind)
		}
	}
	if got := issuesOf(linesAt(st, StageReservationReleased)); !slices.Equal(got, bundleIssues) {
		t.Fatalf("each issue is released once: %v", got)
	}
}

// leadAloneUnrecorded is a bundle run reserved at at whose lead was handed
// back alone and whose own reserved_at is not set, as a take-up after the
// lead's hand-back leaves it: its two members are still held.
func leadAloneUnrecorded(t *testing.T, repo *gittest.Repo, at time.Time) string {
	t.Helper()
	runID := bundleRun(t, repo, at)
	if err := mutate(repo.Root(), runID, func(_ *os.Root, st *State) (bool, error) {
		st.LeadHandBack = &HandBack{At: at, Kind: HandBackUserVisible, Reason: "a reader is told something new"}
		st.ReservedAt = nil
		return true, nil
	}); err != nil {
		t.Fatal(err)
	}
	return runID
}

// TestTheGatesReadTheMembersWithTheLeadHandedBackAlone: with the lead handed
// back alone and its reserved_at not set, the run still holds its members, so
// the renewal renews each, the tidy-up on landing tombstones each, the
// release on hand-back releases each, and the parked renewal reads them; none
// of them reads the lead's hold.
func TestTheGatesReadTheMembersWithTheLeadHandedBackAlone(t *testing.T) {
	repo, bare := bundleRepo(t)
	now := time.Now().UTC().Truncate(time.Second)
	runID := leadAloneUnrecorded(t, repo, now.Add(-70*time.Minute))
	st, err := ReadState(repo.Root(), runID)
	if err != nil {
		t.Fatal(err)
	}
	if !st.holdsAny() || !slices.Equal(st.reservedIssues(), []string{bundleTwo, bundleThree}) {
		t.Fatalf("the run holds its members: %v", st.reservedIssues())
	}
	snap := viewAt(t, repo, now)
	if !parkedDue(st, snap, now) {
		t.Fatal("a parked renewal finds the members due")
	}
	renewReservation(repo.Root(), &st, now, nil)
	if got := issuesOf(linesAt(st, StageReservationRenewed)); !slices.Equal(got, []string{bundleTwo, bundleThree}) {
		t.Fatalf("each member is renewed, the lead not: %v", got)
	}
	lead := tipOf(t, bare, bundleLead)
	c := Context{RepoRoot: repo.Root(), RunDir: runRel(runID), State: st, Now: now}
	if clause := tidyReservation(c, st.Lanes[0]); !strings.Contains(clause, bundleTwo) || !strings.Contains(clause, bundleThree) {
		t.Fatalf("the tidy-up names each member: %q", clause)
	}
	for _, id := range []string{bundleTwo, bundleThree} {
		if !tipOf(t, bare, id).released {
			t.Fatalf("%s is tombstoned on landing", id)
		}
	}
	if got := tipOf(t, bare, bundleLead); got.commit != lead.commit {
		t.Fatalf("the lead's hold is not read: %+v", got)
	}

	repo2, bare2 := bundleRepo(t)
	st2, err := ReadState(repo2.Root(), leadAloneUnrecorded(t, repo2, now))
	if err != nil {
		t.Fatal(err)
	}
	releaseOnHandBack(repo2.Root(), &st2, HandBack{Kind: HandBackStuck}, now)
	if got := issuesOf(linesAt(st2, StageReservationReleased)); !slices.Equal(got, []string{bundleTwo, bundleThree}) {
		t.Fatalf("the release on hand-back releases each member: %v", got)
	}
	for _, id := range []string{bundleTwo, bundleThree} {
		if kind, _ := tombstoneTrailers(t, bare2, id); kind != HandBackStuck {
			t.Fatalf("%s is tombstoned with the lane's kind: %q", id, kind)
		}
	}
}

// viewAt is the reservations as the checkout last fetched them, at now.
func viewAt(t *testing.T, repo *gittest.Repo, now time.Time) claim.Snapshot {
	t.Helper()
	snap, err := claim.View(repo.Root(), now)
	if err != nil {
		t.Fatal(err)
	}
	return snap
}

// TestAStandingBundleReceiptHandingIssuesBackIsNotTakenByAStep: a receipt that
// stood at its path, collected by a step or a stop rather than handed in, is
// not taken when its `handbacks` names any issue, every issue or one: a
// hand-back discards work or releases an issue, which is a decision a step
// does not make on its own (receipt_standing.go). The lane, its worktree and
// every reservation stay as they were.
func TestAStandingBundleReceiptHandingIssuesBackIsNotTakenByAStep(t *testing.T) {
	for name, mut := range map[string]func(*LaneReceipt, string, string){
		"every issue": func(rc *LaneReceipt, _, _ string) {
			rc.Resolves = nil
			rc.HandBacks = []IssueHandBack{handingBack(bundleLead), handingBack(bundleTwo), handingBack(bundleThree)}
		},
		"one issue": func(rc *LaneReceipt, a, b string) {
			rc.Resolves = []Resolution{resolving(bundleLead, a), resolving(bundleTwo, b)}
			rc.HandBacks = []IssueHandBack{handingBack(bundleThree)}
		},
	} {
		t.Run(name, func(t *testing.T) {
			repo, bare, runID, l, dir := bundleAwaiting(t)
			a, b := bundleFix(t, repo, l, bundleLead), bundleFix(t, repo, l, bundleTwo)
			rc := goodReceipt(t, runID, l, dir, a, b)
			mut(&rc, a, b)
			path := writeReceipt(t, dir, rc)
			_, err := handIn(repo.Root(), runID, path, DefaultStages(), Options{}, nil, true)
			r := mustRefusal(t, err)
			if !r.handBack || !strings.Contains(r.Reason, "a step does not do on its own") {
				t.Fatalf("the standing receipt is refused as a hand-back a step does not make: %+v", r)
			}
			st, _ := ReadState(repo.Root(), runID)
			if lane := st.Lanes[0]; lane.Stage != StageImplement || lane.HandBack != nil || len(st.Issues()) != 3 {
				t.Fatalf("the lane and its issues stand as they were: %+v %v", lane, st.Issues())
			}
			if _, err := os.Stat(l.Worktree); err != nil {
				t.Fatalf("the lane's worktree is kept: %v", err)
			}
			for _, id := range bundleIssues {
				if r := tipOf(t, bare, id); r.released {
					t.Fatalf("%s is still held: %+v", id, r)
				}
			}
		})
	}
}

// TestABundlesFixBriefAsksForEveryIssueInTheReceipt: a fix round's brief in a
// run with members names each issue the run still holds and asks for every
// one of them in `resolves` or `handbacks`, with each resolution's commit in
// `commits`; a receipt that follows it, naming each issue's commit, passes the
// receipt's account of the run's issues.
func TestABundlesFixBriefAsksForEveryIssueInTheReceipt(t *testing.T) {
	repo, _, runID, l, _ := bundleAwaiting(t)
	a, b, c := bundleFix(t, repo, l, bundleLead), bundleFix(t, repo, l, bundleTwo), bundleFix(t, repo, l, bundleThree)
	st, err := ReadState(repo.Root(), runID)
	if err != nil {
		t.Fatal(err)
	}
	ctx := Context{RepoRoot: repo.Root(), RunDir: runRel(runID), State: st, Now: time.Now()}
	brief, _, err := writeFixBrief(ctx, l, ValidationRound{Round: 1, HeadSHA: c}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(abs(repo.Root(), brief))
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, w := range []string{bundleLead, bundleTwo, bundleThree, "exactly once in this receipt", "`resolves`", "`handbacks`",
		"the commit that fixes it", "list that commit in `commits`"} {
		if !strings.Contains(text, w) {
			t.Fatalf("the fix brief says %q:\n%s", w, text)
		}
	}
	rc := LaneReceipt{Commits: []string{a, b, c},
		Resolves: []Resolution{resolving(bundleLead, a), resolving(bundleTwo, b), resolving(bundleThree, c)}}
	if gaps := bundleReceiptGaps(repo.Root(), st, l, rc, false); len(gaps) != 0 {
		t.Fatalf("a fix receipt following the brief accounts for every issue: %v", gaps)
	}

	// A run without members is briefed as before, with no such paragraph.
	st.Bundle = nil
	ctx.State = st
	brief, _, err = writeFixBrief(ctx, l, ValidationRound{Round: 2, HeadSHA: c}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if data, _ := os.ReadFile(abs(repo.Root(), brief)); strings.Contains(string(data), "exactly once in this receipt") {
		t.Fatalf("a run without members gets no bundle paragraph:\n%s", data)
	}
}

// TestARunWithOnlyItsMembersHeldIsRenewedWhereItWaits: a run whose lead was
// handed back alone, its members still held, is renewed where its agent
// waits, as a run holding its lead is.
func TestARunWithOnlyItsMembersHeldIsRenewedWhereItWaits(t *testing.T) {
	now := time.Now()
	st := State{Key: bundleLead, LeadHandBack: &HandBack{At: now, Kind: HandBackUserVisible, Reason: "a person's call"},
		Bundle: []Member{{Issue: bundleTwo, ReservedAt: &now}, {Issue: bundleThree, ReservedAt: &now}},
		Lanes:  []Lane{{ID: "lane-1", Key: bundleLead, Stage: StageImplement}}}
	if !renewsWhereItWaits(st) {
		t.Fatal("a run holding its members is renewed where it waits")
	}
	st.Bundle[0].ReservedAt, st.Bundle[1].ReservedAt = nil, nil
	if renewsWhereItWaits(st) {
		t.Fatal("a run holding nothing is not")
	}
}
