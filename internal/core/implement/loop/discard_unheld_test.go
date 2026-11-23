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
	"testing"
	"time"

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
