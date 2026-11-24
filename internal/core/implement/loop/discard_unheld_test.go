package loop

// discard_unheld_test.go is iss-2611231404319245 and iss-2611231404326060:
// `implement step --discard` ends a lane at work whose run nothing drives, and
// a lane handed back for any reason, saving the lane's own commits to a bundle
// in the run's directory first; a lane whose run is live, or whose pull
// request is open, is refused, writing nothing. And an acceptance of a changed
// git configuration given with the discard is written only with the discard's
// own write: a refused discard leaves the run's state and fingerprint as they
// were.

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/core/lease"
	"github.com/intentdriven/abcd/internal/gittest"
)

// pastTheLease is a clock a lease and a minute past the oldest await of the
// run's lanes: a host's agent with no live driver is then gone.
func pastTheLease(t *testing.T, repo *gittest.Repo, runID string) Options {
	t.Helper()
	st, err := ReadState(repo.Root(), runID)
	if err != nil {
		t.Fatal(err)
	}
	at := time.Now()
	for _, l := range st.Lanes {
		for _, a := range l.Awaits {
			if a.Since.Before(at) {
				at = a.Since
			}
		}
	}
	later := at.Add(lease.Default + time.Minute)
	return Options{Now: func() time.Time { return later }}
}

// runDirFiles is every file directly in the run's directory.
func runDirFiles(t *testing.T, repo *gittest.Repo, runID string) []string {
	t.Helper()
	ents, err := os.ReadDir(filepath.Join(repo.Root(), filepath.FromSlash(runRel(runID))))
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, e := range ents {
		if !e.IsDir() {
			out = append(out, e.Name())
		}
	}
	return out
}

// bundlesIn is every bundle in the run's directory.
func bundlesIn(t *testing.T, repo *gittest.Repo, runID string) []string {
	t.Helper()
	var out []string
	for _, f := range runDirFiles(t, repo, runID) {
		if strings.HasSuffix(f, ".bundle") {
			out = append(out, f)
		}
	}
	return out
}

// fingerprintBytes is the run's recorded configuration fingerprint.
func fingerprintBytes(t *testing.T, repo *gittest.Repo, runID string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(repo.Root(), filepath.FromSlash(runRel(runID)), ConfigFingerprintFileName))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// wantBundled wants the discard's result to name a bundle in the run's
// directory that holds the lane's tip on its branch.
func wantBundled(t *testing.T, repo *gittest.Repo, runID string, res StepResult, branch, tip string) {
	t.Helper()
	if res.Bundle == "" {
		t.Fatalf("the result names no bundle: %+v", res)
	}
	path := filepath.Join(repo.Root(), filepath.FromSlash(res.Bundle))
	if filepath.Dir(path) != filepath.Join(repo.Root(), filepath.FromSlash(runRel(runID))) {
		t.Fatalf("the bundle is not in the run's own directory: %s", res.Bundle)
	}
	heads := repo.Git("bundle", "list-heads", path)
	if !strings.Contains(heads, tip) || !strings.Contains(heads, "refs/heads/"+branch) {
		t.Fatalf("the bundle holds the lane's branch at its tip %s: %s", tip, heads)
	}
}

// TestDiscardEndsALaneAtWorkThatNothingDrives: a lane at its implement stage,
// its implementer handed out longer ago than a lease with no live driver,
// is discarded: its two commits are saved to a bundle in the run's directory,
// then its worktree and branch go, and the issue's run is over.
func TestDiscardEndsALaneAtWorkThatNothingDrives(t *testing.T) {
	repo, runID, l, _ := issueAwaiting(t)
	laneCommit(t, repo, l, "one.txt")
	tip := laneCommit(t, repo, l, "two.txt")
	if l.Stage != StageImplement || len(l.Awaits) != 1 {
		t.Fatalf("premise: the lane is at work with its implementer out: %+v", l)
	}
	o := pastTheLease(t, repo, runID)
	res, err := Discard(repo.Root(), runID, l.ID, o)
	if err != nil {
		t.Fatalf("a lane at work that nothing drives is discarded: %v", err)
	}
	if res.Stage != StageDiscarded {
		t.Fatalf("the lane is discarded: %+v", res)
	}
	wantBundled(t, repo, runID, res, l.Branch, tip)
	if _, err := os.Stat(l.Worktree); !os.IsNotExist(err) {
		t.Fatalf("the lane's worktree is removed: %v", err)
	}
	if gitErr(repo, "rev-parse", "--verify", "--quiet", "refs/heads/"+l.Branch) == nil {
		t.Fatal("the lane's branch is deleted")
	}
	st, err := ReadState(repo.Root(), runID)
	if err != nil {
		t.Fatal(err)
	}
	if !st.Complete() {
		t.Fatalf("the issue's run is over: %+v", st.Lanes)
	}
	if !strings.Contains(recordText(st), res.Bundle) {
		t.Fatalf("the run record names the bundle:\n%s", recordText(st))
	}
}

// TestDiscardRefusesALaneWhoseAgentMayStillBeInItsCall: the same lane, its
// implementer handed out within a lease, is live: the discard is refused as
// contention, naming the agent, and nothing is removed or written.
func TestDiscardRefusesALaneWhoseAgentMayStillBeInItsCall(t *testing.T) {
	repo, runID, l, _ := issueAwaiting(t)
	laneCommit(t, repo, l, "one.txt")
	before := stateBytes(t, repo.Root(), runID)
	_, err := Discard(repo.Root(), runID, l.ID, Options{})
	r := mustRefusal(t, err)
	if !r.Contention || !strings.Contains(r.Reason, "implementer") || !strings.Contains(r.Reason, "live") {
		t.Fatalf("the discard backs off from a live run, naming its agent: %+v", r)
	}
	if !bytes.Equal(before, stateBytes(t, repo.Root(), runID)) {
		t.Fatal("a refused discard changed the run's state")
	}
	if _, err := os.Stat(l.Worktree); err != nil {
		t.Fatalf("a refused discard removes nothing: %v", err)
	}
	if b := bundlesIn(t, repo, runID); len(b) != 0 {
		t.Fatalf("a refused discard saves no bundle: %v", b)
	}
}

// TestDiscardRefusesALaneWhoseRunAnotherLiveProcessDrives: a run whose
// recorded driver is another process still alive is live, however old its
// awaits.
func TestDiscardRefusesALaneWhoseRunAnotherLiveProcessDrives(t *testing.T) {
	repo, runID, l, _ := issueAwaiting(t)
	home, _ := stopWorld(t, false)
	procsOf(t, hostTwo, hostOne, hostTwo)
	if err := ClaimDriver(repo.Root(), runID, Options{stopHome: home, Session: "session-two"}); err != nil {
		t.Fatal(err)
	}
	procsOf(t, hostOne, hostOne, hostTwo)
	o := pastTheLease(t, repo, runID)
	o.stopHome = home
	before := stateBytes(t, repo.Root(), runID)
	_, err := Discard(repo.Root(), runID, l.ID, o)
	r := mustRefusal(t, err)
	if !r.Contention || !strings.Contains(r.Reason, "session-two") {
		t.Fatalf("the discard backs off from the run's live driver, naming it: %+v", r)
	}
	if !bytes.Equal(before, stateBytes(t, repo.Root(), runID)) {
		t.Fatal("a refused discard changed the run's state")
	}
	if _, err := os.Stat(l.Worktree); err != nil {
		t.Fatalf("a refused discard removes nothing: %v", err)
	}
}

// TestDiscardEndsALaneHandedBackAsUnachievable: a lane handed back after its
// fix rounds keeps its worktree and branch; the discard takes it, saving its
// commit to a bundle first.
func TestDiscardEndsALaneHandedBackAsUnachievable(t *testing.T) {
	repo, runID, l, _ := issueAwaiting(t)
	tip := laneCommit(t, repo, l, "partial.txt")
	if err := mutate(repo.Root(), runID, func(_ *os.Root, st *State) (bool, error) {
		st.Lanes[0].Stage = StageHandedBack
		st.Lanes[0].Awaits = nil
		st.Lanes[0].HandBack = &HandBack{At: time.Now().UTC().Truncate(time.Second), Verdict: VerdictUnachievable, Round: 2, FixRounds: 1,
			Verdicts: "ruthless-reviewer: not passed"}
		return true, nil
	}); err != nil {
		t.Fatal(err)
	}
	res, err := Discard(repo.Root(), runID, l.ID, Options{})
	if err != nil {
		t.Fatalf("a lane handed back as unachievable is discarded: %v", err)
	}
	if res.Stage != StageDiscarded {
		t.Fatalf("the lane is discarded: %+v", res)
	}
	wantBundled(t, repo, runID, res, l.Branch, tip)
	if _, err := os.Stat(l.Worktree); !os.IsNotExist(err) {
		t.Fatalf("the lane's worktree is removed: %v", err)
	}
	if gitErr(repo, "rev-parse", "--verify", "--quiet", "refs/heads/"+l.Branch) == nil {
		t.Fatal("the lane's branch is deleted")
	}
}

// TestDiscardOfALaneWithNoCommitsOfItsOwnSavesNoBundle: a lane whose branch
// holds nothing past the default branch has nothing to save.
func TestDiscardOfALaneWithNoCommitsOfItsOwnSavesNoBundle(t *testing.T) {
	repo, runID, l, _ := issueAwaiting(t)
	res, err := Discard(repo.Root(), runID, l.ID, pastTheLease(t, repo, runID))
	if err != nil {
		t.Fatalf("a lane at work that nothing drives is discarded: %v", err)
	}
	if res.Bundle != "" || len(bundlesIn(t, repo, runID)) != 0 {
		t.Fatalf("a lane with no commits of its own saves no bundle: %q %v", res.Bundle, bundlesIn(t, repo, runID))
	}
}

// TestDiscardRefusesALaneAtWorkWithAnOpenPullRequest: a stale lane at land
// whose pull request the forge reports open (armed or not) is refused, naming
// it, and nothing is removed; once the forge reports it closed the discard
// goes through and closes nothing.
func TestDiscardRefusesALaneAtWorkWithAnOpenPullRequest(t *testing.T) {
	f := newLandFixture(t, queueRuleset("MERGE"))
	l := f.toArmOnce(t)
	if l.Stage != StageLand || l.PR != 7 {
		t.Fatalf("premise: the lane is at land with its pull request open: %+v", l)
	}
	before := stateBytes(t, f.repo.Root(), f.runID)
	_, err := Discard(f.repo.Root(), f.runID, l.ID, Options{})
	r := mustRefusal(t, err)
	if !strings.Contains(r.Reason, "#7") || !strings.Contains(r.Reason, "open") {
		t.Fatalf("the discard is refused, naming the open pull request: %+v", r)
	}
	if !bytes.Equal(before, stateBytes(t, f.repo.Root(), f.runID)) {
		t.Fatal("a refused discard changed the run's state")
	}
	if _, err := os.Stat(l.Worktree); err != nil {
		t.Fatalf("a refused discard removes nothing: %v", err)
	}
	if err := os.WriteFile(filepath.Join(f.gh, "state"), []byte("CLOSED\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := Discard(f.repo.Root(), f.runID, l.ID, Options{})
	if err != nil || res.Stage != StageDiscarded {
		t.Fatalf("a lane whose pull request is closed is discarded: %+v %v", res, err)
	}
	if strings.Contains(f.ghLog(t), "pr close") {
		t.Fatalf("the discard closes no pull request already closed:\n%s", f.ghLog(t))
	}
}

// TestARefusedDiscardWritesNoAcceptance (iss-2611231404326060): with a
// changed git configuration, a discard given the acceptance and then refused
// leaves the run's state and its fingerprint byte for byte as they were; the
// same discard once it goes through writes the acceptance in its own record.
func TestARefusedDiscardWritesNoAcceptance(t *testing.T) {
	repo, runID, l, _ := issueAwaiting(t)
	repo.Git("config", "filter.lfs.clean", "git-lfs clean -- %f")
	if _, err := Discard(repo.Root(), runID, l.ID, pastTheLease(t, repo, runID)); err == nil {
		t.Fatal("premise: the changed configuration refuses the discard")
	}
	state, fp := stateBytes(t, repo.Root(), runID), fingerprintBytes(t, repo, runID)
	o := Options{AcceptConfig: true}
	if _, err := Discard(repo.Root(), runID, l.ID, o); err == nil {
		t.Fatal("premise: the live lane refuses the discard")
	}
	if !bytes.Equal(state, stateBytes(t, repo.Root(), runID)) {
		t.Fatal("a refused discard wrote its acceptance into the run's state")
	}
	if !bytes.Equal(fp, fingerprintBytes(t, repo, runID)) {
		t.Fatal("a refused discard recorded the configuration as accepted")
	}
	o = pastTheLease(t, repo, runID)
	o.AcceptConfig = true
	res, err := Discard(repo.Root(), runID, l.ID, o)
	if err != nil {
		t.Fatalf("the discard goes through with the acceptance: %v", err)
	}
	if len(res.AcceptedConfigChange) != 1 || res.AcceptedConfigChange[0] != "filter.lfs.clean (added)" {
		t.Fatalf("the result names what was accepted: %q", res.AcceptedConfigChange)
	}
	st, err := ReadState(repo.Root(), runID)
	if err != nil {
		t.Fatal(err)
	}
	text := recordText(st)
	if !strings.Contains(text, "accepted a change to the repository's git configuration: filter.lfs.clean (added)") || !strings.Contains(text, "discarded") {
		t.Fatalf("the record carries the acceptance beside the discard:\n%s", text)
	}
	if bytes.Equal(fp, fingerprintBytes(t, repo, runID)) {
		t.Fatal("the discard that went through did not record the configuration")
	}
}

// TestARefusedReleaseWritesNoAcceptance: --release given the acceptance over a
// lane that is not held is refused, and writes nothing either.
func TestARefusedReleaseWritesNoAcceptance(t *testing.T) {
	repo, runID, l, _ := issueAwaiting(t)
	repo.Git("config", "filter.lfs.clean", "git-lfs clean -- %f")
	if err := configTripwire(repo.Root(), runID); err == nil {
		t.Fatal("premise: the changed configuration trips the tripwire")
	}
	state, fp := stateBytes(t, repo.Root(), runID), fingerprintBytes(t, repo, runID)
	if _, err := Release(repo.Root(), runID, l.ID, Options{AcceptConfig: true}); err == nil {
		t.Fatal("premise: a lane that is not held refuses the release")
	}
	if !bytes.Equal(state, stateBytes(t, repo.Root(), runID)) || !bytes.Equal(fp, fingerprintBytes(t, repo, runID)) {
		t.Fatal("a refused release wrote its acceptance")
	}
}

// wantNothingRemoved wants a refused discard to have left the lane's worktree
// and branch and the run's state as they were.
func wantNothingRemoved(t *testing.T, repo *gittest.Repo, runID string, l Lane, state []byte) {
	t.Helper()
	if !bytes.Equal(state, stateBytes(t, repo.Root(), runID)) {
		t.Fatal("a refused discard changed the run's state")
	}
	if _, err := os.Stat(l.Worktree); err != nil {
		t.Fatalf("a refused discard removes no worktree: %v", err)
	}
	if err := gitErr(repo, "rev-parse", "--verify", "--quiet", "refs/heads/"+l.Branch); err != nil {
		t.Fatalf("a refused discard deletes no branch: %v", err)
	}
}

// TestDiscardWritesNoBundleOverANameAlreadyTaken: a link standing at the
// bundle's name in the run's directory is never written through: the discard
// is refused before anything is removed, and the file the link names keeps
// what it held.
func TestDiscardWritesNoBundleOverANameAlreadyTaken(t *testing.T) {
	repo, runID, l, _ := issueAwaiting(t)
	tip := laneCommit(t, repo, l, "one.txt")
	target := filepath.Join(t.TempDir(), "target")
	if err := os.WriteFile(target, []byte("kept\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(repo.Root(), filepath.FromSlash(runRel(runID)), l.ID+"-"+tip[:12]+".bundle")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	state := stateBytes(t, repo.Root(), runID)
	_, err := Discard(repo.Root(), runID, l.ID, pastTheLease(t, repo, runID))
	r := mustRefusal(t, err)
	if !strings.Contains(r.Reason, "already exists") || !strings.Contains(r.Remedy, "nothing was removed") {
		t.Fatalf("the discard is refused, naming the name already taken: %+v", r)
	}
	if b, err := os.ReadFile(target); err != nil || string(b) != "kept\n" {
		t.Fatalf("the file the link names was written: %q %v", b, err)
	}
	if fi, err := os.Lstat(link); err != nil || fi.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("the link at the bundle's name was replaced: %v %v", fi, err)
	}
	wantNothingRemoved(t, repo, runID, l, state)
}

// TestDiscardDeletesTheBranchOnlyAtTheTipItSaved: a branch that moves after
// its commits were saved is not deleted: the discard is refused, the branch
// keeps its new commit, and the bundle holds the tip that was saved.
func TestDiscardDeletesTheBranchOnlyAtTheTipItSaved(t *testing.T) {
	repo, runID, l, _ := issueAwaiting(t)
	tip := laneCommit(t, repo, l, "one.txt")
	old := saveBundle
	t.Cleanup(func() { saveBundle = old })
	moved := ""
	saveBundle = func(root *os.Root, repoRoot, id string, lane Lane) (string, int, string, error) {
		rel, n, at, err := old(root, repoRoot, id, lane)
		moved = laneCommit(t, repo, l, "two.txt")
		return rel, n, at, err
	}
	_, err := Discard(repo.Root(), runID, l.ID, pastTheLease(t, repo, runID))
	mustRefusal(t, err)
	if got := repo.Git("rev-parse", "refs/heads/"+l.Branch); got != moved {
		t.Fatalf("the branch that moved is kept at its new tip %s: %q", moved, got)
	}
	b := bundlesIn(t, repo, runID)
	if len(b) != 1 {
		t.Fatalf("the saved tip's bundle stays in the run's directory: %v", b)
	}
	heads := repo.Git("bundle", "list-heads", filepath.Join(repo.Root(), filepath.FromSlash(runRel(runID)), b[0]))
	if !strings.Contains(heads, tip) {
		t.Fatalf("the bundle holds the tip that was saved, %s: %s", tip, heads)
	}
}

// reservedIssueAwaiting is a reserving issue run whose lane awaits its
// implementer.
func reservedIssueAwaiting(t *testing.T) (repo *gittest.Repo, bare, runID string, l Lane) {
	t.Helper()
	repo, bare = reservingDrainRepo(t, "commands/drain.md")
	res, err := startedOnce(t, repo, secondIssue)
	if err != nil {
		t.Fatal(err)
	}
	for range len(Sequence) {
		if currentLane(t, repo, res.RunID).awaiting() != nil {
			break
		}
		if _, err := advance(repo.Root(), res.RunID, DefaultStages(), Options{}); err != nil {
			t.Fatal(err)
		}
	}
	if tomb := originReservation(t, bare, claim.RefPrefix+secondIssue); tomb.commit == "" || tomb.released {
		t.Fatalf("premise: the run holds its reservation: %+v", tomb)
	}
	return repo, bare, res.RunID, currentLane(t, repo, res.RunID)
}

// TestDiscardLeavesAReservationMadeAgainSinceTheRunHeldIt: the run's
// reservation lapsed and this person reserved the issue again from this copy;
// the discard that ends the run leaves that newer reservation as it stands
// and says so.
func TestDiscardLeavesAReservationMadeAgainSinceTheRunHeldIt(t *testing.T) {
	repo, bare, runID, l := reservedIssueAwaiting(t)
	o := pastTheLease(t, repo, runID)
	if _, err := claim.Reserve(repo.Root(), claim.Request{Record: secondIssue, Footprint: []string{"commands/drain.md"}}, clockAt(o.Now().Add(-10*time.Second))); err != nil {
		t.Fatal(err)
	}
	mine := originReservation(t, bare, claim.RefPrefix+secondIssue)
	res, err := Discard(repo.Root(), runID, l.ID, o)
	if err != nil || res.Stage != StageDiscarded {
		t.Fatalf("the lane is discarded: %+v %v", res, err)
	}
	if now := originReservation(t, bare, claim.RefPrefix+secondIssue); now.commit != mine.commit || now.released {
		t.Fatalf("the reservation made since is left as it stands: %+v, was %+v", now, mine)
	}
	if note := recordNote(t, repo, runID, StageReservationReleased); !strings.Contains(note, "as it stands") {
		t.Fatalf("the run record says the reservation was left as it stands: %q", note)
	}
}

// TestDiscardReleasesTheRunsOwnReservation: the same discard, with nothing
// reserved since, releases the run's own reservation.
func TestDiscardReleasesTheRunsOwnReservation(t *testing.T) {
	repo, bare, runID, l := reservedIssueAwaiting(t)
	if _, err := Discard(repo.Root(), runID, l.ID, pastTheLease(t, repo, runID)); err != nil {
		t.Fatal(err)
	}
	if tomb := originReservation(t, bare, claim.RefPrefix+secondIssue); !tomb.released {
		t.Fatalf("the run's own reservation is released: %+v", tomb)
	}
}

// TestDiscardRefusesALaneAtLandWhosePullRequestItsStateNeverRecorded: a lane
// pushed at land whose state records no pull request (opened, then not read
// back) is refused while the forge lists one open for its branch, and while
// that listing cannot be read; once the forge lists none, it is discarded.
func TestDiscardRefusesALaneAtLandWhosePullRequestItsStateNeverRecorded(t *testing.T) {
	f := newLandFixture(t, queueRuleset("MERGE"))
	l := f.toArmOnce(t)
	if l.Landing == nil || l.Landing.Pushed == "" {
		t.Fatalf("premise: the lane is pushed: %+v", l)
	}
	if err := mutate(f.repo.Root(), f.runID, func(_ *os.Root, st *State) (bool, error) {
		st.Lanes[0].PR = 0
		return true, nil
	}); err != nil {
		t.Fatal(err)
	}
	state := stateBytes(t, f.repo.Root(), f.runID)
	_, err := Discard(f.repo.Root(), f.runID, l.ID, Options{})
	if r := mustRefusal(t, err); !strings.Contains(r.Reason, "#7") || !strings.Contains(r.Reason, "open") {
		t.Fatalf("the discard is refused, naming the pull request the forge lists: %+v", r)
	}
	wantNothingRemoved(t, f.repo, f.runID, l, state)
	if err := os.WriteFile(filepath.Join(f.gh, "refuse-list"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	_, err = Discard(f.repo.Root(), f.runID, l.ID, Options{})
	mustRefusal(t, err)
	wantNothingRemoved(t, f.repo, f.runID, l, state)
	if err := os.Remove(filepath.Join(f.gh, "refuse-list")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(f.gh, "pr.json"), []byte("[]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if res, err := Discard(f.repo.Root(), f.runID, l.ID, Options{}); err != nil || res.Stage != StageDiscarded {
		t.Fatalf("a lane whose branch has no pull request is discarded: %+v %v", res, err)
	}
}

// closedAtLand is the land fixture's lane at land, its run holding its
// reservation and its pull request closed on the forge, so a discard takes it.
func closedAtLand(t *testing.T) (*landFixture, Lane) {
	t.Helper()
	f := newLandFixture(t, queueRuleset("MERGE"))
	l := f.toArmOnce(t)
	reserveRun(t, f.repo, f.runID, "itd-10", time.Now().UTC().Truncate(time.Second), "internal/widget")
	if err := os.WriteFile(filepath.Join(f.gh, "state"), []byte("CLOSED\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return f, l
}

// TestDiscardOfTheRunsLastLaneReleasesItsReservation: with nothing pending
// and no lane owing anything, the discard of the run's last lane releases its
// reservation.
func TestDiscardOfTheRunsLastLaneReleasesItsReservation(t *testing.T) {
	f, l := closedAtLand(t)
	if _, err := Discard(f.repo.Root(), f.runID, l.ID, Options{}); err != nil {
		t.Fatal(err)
	}
	if tomb := originReservation(t, f.bare, reservedRef); !tomb.released {
		t.Fatalf("the run's reservation is released: %+v", tomb)
	}
}

// TestDiscardKeepsTheReservationWhileAStepIsPending: a step of the spec still
// waiting for a lane will be taken up by the next step, so the discard of the
// run's only lane releases nothing.
func TestDiscardKeepsTheReservationWhileAStepIsPending(t *testing.T) {
	f, l := closedAtLand(t)
	if err := mutate(f.repo.Root(), f.runID, func(_ *os.Root, st *State) (bool, error) {
		st.Pending = append(st.Pending, PendingStep{Number: 2, Title: "the second step", Needs: []int{1}})
		return true, nil
	}); err != nil {
		t.Fatal(err)
	}
	if res, err := Discard(f.repo.Root(), f.runID, l.ID, Options{}); err != nil || res.Stage != StageDiscarded {
		t.Fatalf("the lane is discarded: %+v %v", res, err)
	}
	if tomb := originReservation(t, f.bare, reservedRef); tomb.released {
		t.Fatalf("a run with a step pending keeps its reservation: %+v", tomb)
	}
	if note := recordNote(t, f.repo, f.runID, StageReservationReleased); note != "" {
		t.Fatalf("the run record logs no release: %q", note)
	}
}

// TestDiscardKeepsTheReservationWhileAHandedBackLaneIsStillArmed: a sibling
// handed back with its pull request still armed owes the arming's withdrawal,
// so the discard of the other lane releases nothing.
func TestDiscardKeepsTheReservationWhileAHandedBackLaneIsStillArmed(t *testing.T) {
	f, l := closedAtLand(t)
	if err := mutate(f.repo.Root(), f.runID, func(_ *os.Root, st *State) (bool, error) {
		sib := st.Lanes[0]
		sib.ID, sib.Branch, sib.Worktree, sib.Awaits, sib.Stage, sib.PR = "lane-2", "", "", nil, StageHandedBack, 8
		landing := *st.Lanes[0].Landing
		landing.Armed = true
		sib.Landing = &landing
		sib.HandBack = &HandBack{At: time.Now().UTC().Truncate(time.Second), Verdict: VerdictUnachievable, Round: 2, FixRounds: 1,
			Verdicts: "ruthless-reviewer: not passed"}
		if !armingOwed(sib) {
			t.Fatalf("premise: the sibling owes its arming's withdrawal: %+v", sib)
		}
		st.Lanes = append(st.Lanes, sib)
		return true, nil
	}); err != nil {
		t.Fatal(err)
	}
	if res, err := Discard(f.repo.Root(), f.runID, l.ID, Options{}); err != nil || res.Stage != StageDiscarded {
		t.Fatalf("the lane is discarded: %+v %v", res, err)
	}
	if tomb := originReservation(t, f.bare, reservedRef); tomb.released {
		t.Fatalf("a run whose handed-back lane is still armed keeps its reservation: %+v", tomb)
	}
}

// TestTheBundleHeaderMustListTheSavedTipAlone: a bundle is kept only when its
// header, read whole across the writes git makes, lists the branch at the
// tip that was read and no other ref.
func TestTheBundleHeaderMustListTheSavedTipAlone(t *testing.T) {
	tip := strings.Repeat("a", 40)
	line := tip + " refs/heads/build/x"
	for _, c := range []struct {
		name, header string
		want         bool
	}{
		{"the tip alone", "# v2 git bundle\n-" + strings.Repeat("b", 40) + " base\n" + line + "\n\nPACK", true},
		{"a moved tip", "# v2 git bundle\n" + strings.Repeat("c", 40) + " refs/heads/build/x\n\nPACK", false},
		{"a second ref", "# v3 git bundle\n@object-format=sha1\n" + line + "\n" + tip + " refs/heads/other\n\nPACK", false},
		{"no end to the header", "# v2 git bundle\n" + line + "\n", false},
	} {
		h := &bundleHeader{}
		for _, b := range []byte(c.header) {
			if _, err := h.Write([]byte{b}); err != nil {
				t.Fatal(err)
			}
		}
		if got := h.listsOnly(line); got != c.want {
			t.Errorf("%s: listsOnly = %v; want %v", c.name, got, c.want)
		}
	}
}

// wantLeftStanding wants the discard to have left the reservation on origin
// at the commit it held before, unreleased, and the run record to say so.
func wantLeftStanding(t *testing.T, repo *gittest.Repo, bare, runID string, before reservation) {
	t.Helper()
	if now := originReservation(t, bare, claim.RefPrefix+secondIssue); now.commit != before.commit || now.released {
		t.Fatalf("the reservation is left as it stands: %+v, was %+v", now, before)
	}
	if note := recordNote(t, repo, runID, StageReservationReleased); !strings.Contains(note, "as it stands") {
		t.Fatalf("the run record says the reservation was left as it stands: %q", note)
	}
}

// TestDiscardLeavesAReservationMadeBeforeAnUnrelatedStateWrite: this person
// released the run's reservation and reserved the issue again from this copy,
// and the run's state was written after that for a reason of its own (a
// driver's claim); the reservation is still not one the run took, so the
// discard leaves it standing.
func TestDiscardLeavesAReservationMadeBeforeAnUnrelatedStateWrite(t *testing.T) {
	repo, bare, runID, l := reservedIssueAwaiting(t)
	o := pastTheLease(t, repo, runID)
	if _, err := claim.Reserve(repo.Root(), claim.Request{Record: secondIssue, Release: true}, clockAt(o.Now().Add(-3*time.Minute))); err != nil {
		t.Fatal(err)
	}
	if _, err := claim.Reserve(repo.Root(), claim.Request{Record: secondIssue, Footprint: []string{"commands/drain.md"}}, clockAt(o.Now().Add(-2*time.Minute))); err != nil {
		t.Fatal(err)
	}
	written := o.Now().Add(-time.Minute)
	if err := mutate(repo.Root(), runID, func(_ *os.Root, st *State) (bool, error) {
		st.Record = append(st.Record, Entry{At: written, Stage: StageDriver, Note: "replaced the driver, which is gone"})
		st.UpdatedAt = written
		return true, nil
	}); err != nil {
		t.Fatal(err)
	}
	mine := originReservation(t, bare, claim.RefPrefix+secondIssue)
	if res, err := Discard(repo.Root(), runID, l.ID, o); err != nil || res.Stage != StageDiscarded {
		t.Fatalf("the lane is discarded: %+v %v", res, err)
	}
	wantLeftStanding(t, repo, bare, runID, mine)
}

// TestDiscardLeavesAReservationMadeAgainInTheSecondTheRunReserved: this
// person released the run's reservation and reserved the issue again from
// this copy within the same second the run reserved it, so the new tip's
// claimed-at is the run's logged act to the second. The tip is still not the
// run's: the run's own commit at that second lies below it, past the release,
// so the act's time names two commits and proves neither. The discard leaves
// the reservation standing.
func TestDiscardLeavesAReservationMadeAgainInTheSecondTheRunReserved(t *testing.T) {
	repo, bare, runID, l := reservedIssueAwaiting(t)
	own := originReservation(t, bare, claim.RefPrefix+secondIssue)
	if _, err := claim.Reserve(repo.Root(), claim.Request{Record: secondIssue, Release: true}, clockAt(own.claimed)); err != nil {
		t.Fatal(err)
	}
	if _, err := claim.Reserve(repo.Root(), claim.Request{Record: secondIssue, Footprint: []string{"commands/drain.md"}}, clockAt(own.claimed)); err != nil {
		t.Fatal(err)
	}
	mine := originReservation(t, bare, claim.RefPrefix+secondIssue)
	if mine.released || !mine.claimed.Equal(own.claimed) {
		t.Fatalf("premise: the reservation made again carries the run's claimed-at: %+v, run's %+v", mine, own)
	}
	if res, err := Discard(repo.Root(), runID, l.ID, pastTheLease(t, repo, runID)); err != nil || res.Stage != StageDiscarded {
		t.Fatalf("the lane is discarded: %+v %v", res, err)
	}
	wantLeftStanding(t, repo, bare, runID, mine)
}

// TestDiscardLeavesAReservationMadeAgainWhenTheRunWasCreated: a start's clock
// moves on between its reservation and the run's creation, as a slow start's
// does. The run records its reservation at the time the reservation was
// claimed, not at the run's creation, so this person's release and fresh
// reservation from this copy at the run's creation time carry no time the run
// logs as its own, and the discard leaves that reservation standing.
func TestDiscardLeavesAReservationMadeAgainWhenTheRunWasCreated(t *testing.T) {
	repo, bare := reservingDrainRepo(t, "commands/drain.md")
	base := time.Now().UTC().Truncate(time.Second).Add(-10 * time.Minute)
	var ticks atomic.Int64
	ticking := Options{Now: func() time.Time { return base.Add(time.Duration(ticks.Add(1)) * time.Second) }}
	res, err := Start(repo.Root(), secondIssue, ticking)
	if err != nil {
		t.Fatal(err)
	}
	for range len(Sequence) {
		if currentLane(t, repo, res.RunID).awaiting() != nil {
			break
		}
		if _, err := advance(repo.Root(), res.RunID, DefaultStages(), Options{}); err != nil {
			t.Fatal(err)
		}
	}
	own := originReservation(t, bare, claim.RefPrefix+secondIssue)
	st, err := ReadState(repo.Root(), res.RunID)
	if err != nil {
		t.Fatal(err)
	}
	var created time.Time
	for _, e := range st.Record {
		switch e.Stage {
		case "start":
			if created.IsZero() {
				created = e.At
			}
		case StageReservation:
			if !e.At.Equal(own.claimed) {
				t.Errorf("the run logs its reservation at %s, the time it was claimed at, %s", e.At, own.claimed)
			}
		}
	}
	if !created.After(own.claimed) {
		t.Fatalf("premise: the run was created after its reservation was claimed: %s, claimed %s", created, own.claimed)
	}
	if _, err := claim.Reserve(repo.Root(), claim.Request{Record: secondIssue, Release: true}, clockAt(created)); err != nil {
		t.Fatal(err)
	}
	if _, err := claim.Reserve(repo.Root(), claim.Request{Record: secondIssue, Footprint: []string{"commands/drain.md"}}, clockAt(created)); err != nil {
		t.Fatal(err)
	}
	mine := originReservation(t, bare, claim.RefPrefix+secondIssue)
	l := currentLane(t, repo, res.RunID)
	if res, err := Discard(repo.Root(), res.RunID, l.ID, pastTheLease(t, repo, res.RunID)); err != nil || res.Stage != StageDiscarded {
		t.Fatalf("the lane is discarded: %+v %v", res, err)
	}
	wantLeftStanding(t, repo, bare, res.RunID, mine)
}

// TestDiscardLeavesAReservationMadeAgainWhenAWideningBackedOff: a lane's
// widening of the run's hold backed off, which the record logs as a
// reservation line though nothing was pushed; this person then released the
// run's reservation and reserved the issue again from this copy in that same
// second. A lane's reservation line is no anchor of the run's hold, so the
// discard leaves the reservation standing.
func TestDiscardLeavesAReservationMadeAgainWhenAWideningBackedOff(t *testing.T) {
	repo, bare, runID, l := reservedIssueAwaiting(t)
	o := pastTheLease(t, repo, runID)
	backedOff := o.Now().Add(-2 * time.Minute).UTC().Truncate(time.Second)
	if err := mutate(repo.Root(), runID, func(_ *os.Root, st *State) (bool, error) {
		st.Record = append(st.Record, Entry{At: backedOff, Lane: l.ID, Stage: StageReservation,
			Note: "the widening of " + secondIssue + "'s hold to docs/x.md backed off: held by another; nothing was pushed, and the lane edits none of them"})
		st.UpdatedAt = backedOff
		return true, nil
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := claim.Reserve(repo.Root(), claim.Request{Record: secondIssue, Release: true}, clockAt(backedOff)); err != nil {
		t.Fatal(err)
	}
	if _, err := claim.Reserve(repo.Root(), claim.Request{Record: secondIssue, Footprint: []string{"commands/drain.md"}}, clockAt(backedOff)); err != nil {
		t.Fatal(err)
	}
	mine := originReservation(t, bare, claim.RefPrefix+secondIssue)
	if res, err := Discard(repo.Root(), runID, l.ID, o); err != nil || res.Stage != StageDiscarded {
		t.Fatalf("the lane is discarded: %+v %v", res, err)
	}
	wantLeftStanding(t, repo, bare, runID, mine)
}

// TestDiscardReleasesTheRunsReservationRenewedByAnotherWaiter: a load-gate
// waiter in this copy renewed the run's own reservation half through its
// lease, which the run's record never logs; the renewal is the same hold, so
// the discard releases it.
func TestDiscardReleasesTheRunsReservationRenewedByAnotherWaiter(t *testing.T) {
	repo, bare, runID, l := reservedIssueAwaiting(t)
	own := originReservation(t, bare, claim.RefPrefix+secondIssue)
	half := own.claimed.Add(own.expires.Sub(own.claimed)/2 + time.Minute)
	renewed, err := claim.RenewDue(repo.Root(), clockAt(half))
	if err != nil || len(renewed) != 1 {
		t.Fatalf("premise: the waiter renews the run's reservation: %+v %v", renewed, err)
	}
	if r := originReservation(t, bare, claim.RefPrefix+secondIssue); r.commit == own.commit {
		t.Fatalf("premise: the renewal is a commit of its own: %+v", r)
	}
	if res, err := Discard(repo.Root(), runID, l.ID, pastTheLease(t, repo, runID)); err != nil || res.Stage != StageDiscarded {
		t.Fatalf("the lane is discarded: %+v %v", res, err)
	}
	if tomb := originReservation(t, bare, claim.RefPrefix+secondIssue); !tomb.released {
		t.Fatalf("the run's renewed reservation is released: %+v", tomb)
	}
}

// TestDiscardReleasesNothingTheViewShowsReleased: a reservation this
// checkout's view shows released already is not released again: nothing is
// pushed, and the record says there was nothing of the run's to release.
func TestDiscardReleasesNothingTheViewShowsReleased(t *testing.T) {
	repo, bare, runID, l := reservedIssueAwaiting(t)
	o := pastTheLease(t, repo, runID)
	if _, err := claim.Reserve(repo.Root(), claim.Request{Record: secondIssue, Release: true}, clockAt(o.Now().Add(-time.Minute))); err != nil {
		t.Fatal(err)
	}
	tomb := originReservation(t, bare, claim.RefPrefix+secondIssue)
	if res, err := Discard(repo.Root(), runID, l.ID, o); err != nil || res.Stage != StageDiscarded {
		t.Fatalf("the lane is discarded: %+v %v", res, err)
	}
	if now := originReservation(t, bare, claim.RefPrefix+secondIssue); now.commit != tomb.commit {
		t.Fatalf("nothing is pushed over a reservation already released: %+v, was %+v", now, tomb)
	}
	if note := recordNote(t, repo, runID, StageReservationReleased); !strings.Contains(note, "nothing of this run's to release") {
		t.Fatalf("the run record says there was nothing to release: %q", note)
	}
}

// TestDiscardClearsAStaleBundleLeftPartWritten: a part-written bundle a
// discard cut short left in the run's directory, even a link, is removed, not
// written through, and the retry saves the bundle under its own name.
func TestDiscardClearsAStaleBundleLeftPartWritten(t *testing.T) {
	repo, runID, l, _ := issueAwaiting(t)
	tip := laneCommit(t, repo, l, "one.txt")
	target := filepath.Join(t.TempDir(), "target")
	if err := os.WriteFile(target, []byte("kept\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	partial := filepath.Join(repo.Root(), filepath.FromSlash(runRel(runID)), l.ID+"-"+tip[:12]+".bundle.partial")
	if err := os.Symlink(target, partial); err != nil {
		t.Fatal(err)
	}
	res, err := Discard(repo.Root(), runID, l.ID, pastTheLease(t, repo, runID))
	if err != nil || res.Stage != StageDiscarded {
		t.Fatalf("the discard goes through past a stale part-written bundle: %+v %v", res, err)
	}
	wantBundled(t, repo, runID, res, l.Branch, tip)
	if b, err := os.ReadFile(target); err != nil || string(b) != "kept\n" {
		t.Fatalf("the file the stale link names was written: %q %v", b, err)
	}
	if _, err := os.Lstat(partial); !os.IsNotExist(err) {
		t.Fatalf("the stale part-written bundle is removed: %v", err)
	}
}
