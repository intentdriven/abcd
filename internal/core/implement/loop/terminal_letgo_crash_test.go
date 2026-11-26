//go:build unix

package loop

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/gittest"
)

// A drive's let-go after a move killed between its run write and its drain
// write (iss-2611260523161723, dcn-2611260036312398): the lane a no opens
// starts on the drain's own hold with no second push, so the drain state the
// kill leaves still records as the drain's a hold the new run now carries.
// The let-go reads that run and records its lane, and releases none of its
// holds; a hold no run carries goes as before.

// killAfterRunWrite makes the next lane the drain opens die as a kill would
// leave it: the start writes the run, and the drain state stays as it was on
// disk at that moment. The returned func puts it back once the move is over,
// so the move's own write after the start is undone, and nothing else is.
// No defer runs at a kill, so the move's releases after the start are what a
// killed move never makes; the fixtures here leave it none to make.
func killAfterRunWrite(t *testing.T, repo *gittest.Repo) (restore func()) {
	t.Helper()
	path := filepath.Join(repo.Root(), filepath.FromSlash(DrainStateRel))
	var onDisk []byte
	snap := func() {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("reading the drain state at the kill: %v", err)
		}
		onDisk = data
	}
	wasLane, wasBundle := startLane, startBundle
	startLane = func(root, key string, o Options) (StartResult, error) {
		res, err := wasLane(root, key, o)
		if err == nil && onDisk == nil {
			snap()
		}
		return res, err
	}
	startBundle = func(root, lead string, o Options, members []laneMember) (StartResult, error) {
		res, err := wasBundle(root, lead, o, members)
		if err == nil && onDisk == nil {
			snap()
		}
		return res, err
	}
	t.Cleanup(func() { startLane, startBundle = wasLane, wasBundle })
	return func() {
		t.Helper()
		startLane, startBundle = wasLane, wasBundle
		if onDisk == nil {
			t.Fatal("no lane opened, so nothing was killed")
		}
		if err := os.WriteFile(path, onDisk, 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// killedBundle opens a bundle of a lead and two members with the move killed
// after the run write, and returns the run and each issue's hold on origin.
func killedBundle(t *testing.T) (repo *gittest.Repo, bare, runID, lead string, members []string, taken map[string]string) {
	t.Helper()
	repo, bare = bundleDrainRepo(t)
	bundleMachineConfig(t, `{"drain":{"bundle_max":3}}`)
	first := mustDrain(t, repo, Options{}, DrainOptions{})
	lead, m1, m2, _ := orderOf(t, first)
	second := judge(t, repo, Options{}, first, JudgementNo, "", noReason)
	third := judge(t, repo, Options{}, second, JudgementNo, "", noReason)
	restore := killAfterRunWrite(t, repo)
	fourth := judge(t, repo, Options{}, third, JudgementNo, "", noReason)
	restore()
	if fourth.Lane == nil || !slices.Equal(fourth.Lane.Bundle, []string{m1, m2}) {
		t.Fatalf("the bundle's lane opens with both members: %+v", fourth.Lane)
	}
	// The kill leaves the lead and the first member forming, and the last
	// member's judgement awaited on its hold, as the move before it wrote.
	st := formedState(t, repo)
	if st.Forming == nil || st.Judging == nil || st.Judging.Hold == nil || st.Judging.Issue != m2 || len(st.Lanes) != 0 {
		t.Fatalf("the kill leaves the bundle forming, %s awaited, and no lane recorded: %+v", m2, st)
	}
	members = []string{m1, m2}
	taken = map[string]string{}
	for _, id := range append([]string{lead}, members...) {
		taken[id] = refOf(t, bare, id)
	}
	for _, id := range []string{lead, m1} {
		if !st.Forming.holds(id, taken[id]) {
			t.Fatalf("the forming the kill leaves records %s's hold at origin's tip %s: %+v", id, taken[id], st.Forming)
		}
	}
	if st.Judging.Hold.Commit != taken[m2] {
		t.Fatalf("the judgement the kill leaves records %s's hold at origin's tip %s: %+v", m2, taken[m2], st.Judging.Hold)
	}
	return repo, bare, fourth.Lane.RunID, lead, members, taken
}

// recordedLane fails unless the drain state records runID's lane in progress
// on lead with members, and neither a judgement awaited nor a bundle forming.
func recordedLane(t *testing.T, repo *gittest.Repo, runID, lead string, members []string) {
	t.Helper()
	st, live, err := readDrain(repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	if !live || st.Judging != nil || st.Forming != nil || len(st.Lanes) != 1 {
		t.Fatalf("the let-go records the lane the kill left and clears what the drain held: live %v %+v", live, st)
	}
	l := st.Lanes[0]
	if l.RunID != runID || l.Issue != lead || l.Outcome != DrainLaneInProgress || !slices.Equal(l.Bundle, members) {
		t.Fatalf("the lane recorded = %+v; want run %s on %s with %v, in progress", l, runID, lead, members)
	}
}

// TestALetGoAfterAKilledBundleOpenLeavesTheLeadsHold: a move that opened a
// bundle's lane was killed after the run write, so the drain state still
// records the bundle forming; a drive's let-go after it leaves the lead's
// hold, which the run carries, held on origin, and records the lane. Watched
// failing before letGo read the runs: the lead's reservation was released
// plainly under the run that holds it.
func TestALetGoAfterAKilledBundleOpenLeavesTheLeadsHold(t *testing.T) {
	repo, bare, runID, lead, members, taken := killedBundle(t)
	var said []string
	newTerminal(repo.Root(), TerminalOptions{Say: sayInto(&said)}).letGo()
	heldStill(t, bare, lead, taken[lead])
	recordedLane(t, repo, runID, lead, members)
	if run, err := ReadState(repo.Root(), runID); err != nil || letGoByDrive(run) || !slices.Contains(run.reservedIssues(), lead) {
		t.Fatalf("the run still holds its lead, not let go: %v %+v", err, run.Record)
	}
}

// TestALetGoAfterAKilledBundleOpenLeavesEveryMembersHold: the same kill,
// for the members: each member's hold, which the run carries, stays held on
// origin through the let-go, and the let-go says the lane keeps them.
// Watched failing before letGo read the runs: each member was released.
func TestALetGoAfterAKilledBundleOpenLeavesEveryMembersHold(t *testing.T) {
	repo, bare, runID, lead, members, taken := killedBundle(t)
	var said []string
	newTerminal(repo.Root(), TerminalOptions{Say: sayInto(&said)}).letGo()
	for _, id := range members {
		heldStill(t, bare, id, taken[id])
	}
	all := strings.Join(said, "\n")
	if strings.Contains(all, "released the reservation") {
		t.Fatalf("the let-go releases nothing the run carries:\n%s", all)
	}
	if !strings.Contains(all, runID) || !strings.Contains(all, members[0]) || !strings.Contains(all, members[1]) {
		t.Fatalf("the let-go names the lane it recorded, with its members:\n%s", all)
	}
	recordedLane(t, repo, runID, lead, members)
}

// TestALetGoAfterAKilledSingleLaneOpenLeavesTheJudgementsHold: a no opened a
// single lane on the hold the judgement was asked on, and the move was
// killed after the run write, so the drain state still awaits the judgement
// with that hold; a drive's let-go leaves the hold, the run's now, held on
// origin and records the lane. Watched failing before letGo read the runs:
// the judgement's hold was released under the run.
func TestALetGoAfterAKilledSingleLaneOpenLeavesTheJudgementsHold(t *testing.T) {
	repo, bare := reservingDrainRepo(t, "commands/drain.md")
	first, taken := askFirstOnce(t, repo, bare, DrainOptions{})
	restore := killAfterRunWrite(t, repo)
	next := judge(t, repo, Options{}, first, JudgementNo, "", "the remedy changes neither")
	restore()
	if next.Lane == nil || next.Lane.Issue != secondIssue {
		t.Fatalf("the no opens %s's lane: %+v", secondIssue, next)
	}
	if st := formedState(t, repo); st.Judging == nil || st.Judging.Hold == nil || st.Judging.Hold.Commit != taken || len(st.Lanes) != 0 {
		t.Fatalf("the kill leaves the judgement awaited on its hold and no lane: %+v", st)
	}
	var said []string
	newTerminal(repo.Root(), TerminalOptions{Say: sayInto(&said)}).letGo()
	heldStill(t, bare, secondIssue, taken)
	recordedLane(t, repo, next.Lane.RunID, secondIssue, nil)
}

// TestALetGoAfterAnUnkilledSingleLaneOpenReleasesItThroughItsRun: with no
// kill, the lane a no opens is recorded, and a drive's let-go releases its
// hold through its run, which records the release, as it did before. It
// passes before and after the change: it guards the let-go every drive makes.
func TestALetGoAfterAnUnkilledSingleLaneOpenReleasesItThroughItsRun(t *testing.T) {
	repo, bare := reservingDrainRepo(t, "commands/drain.md")
	first, taken := askFirstOnce(t, repo, bare, DrainOptions{})
	next := judge(t, repo, Options{}, first, JudgementNo, "", "the remedy changes neither")
	if next.Lane == nil || next.Lane.Issue != secondIssue {
		t.Fatalf("the no opens %s's lane: %+v", secondIssue, next)
	}
	var said []string
	newTerminal(repo.Root(), TerminalOptions{Say: sayInto(&said)}).letGo()
	releasedOver(t, bare, secondIssue, taken)
	run, err := ReadState(repo.Root(), next.Lane.RunID)
	if err != nil || !letGoByDrive(run) {
		t.Fatalf("the run records the drive's let-go: %v %+v", err, run.Record)
	}
	if st := formedState(t, repo); len(st.Lanes) != 1 || st.Judging != nil {
		t.Fatalf("the let-go adds no lane and awaits nothing: %+v", st)
	}
	if !slices.ContainsFunc(said, func(s string) bool {
		return strings.HasPrefix(s, secondIssue+": released the reservation on "+secondIssue)
	}) {
		t.Fatalf("the let-go says it released %s:\n%s", secondIssue, strings.Join(said, "\n"))
	}
}

// TestAMovesFinishWritesItsStateBeforeItReleasesAHold: a no over X while an
// issue new on origin heads the order leaves X's hold to the move's finish;
// by the time that release is pushed, the drain state on disk already awaits
// the new issue rather than X, so a move killed during the push leaves no
// state recording a hold it was releasing. Watched failing before finish
// wrote first: the push found X still awaited on disk.
func TestAMovesFinishWritesItsStateBeforeItReleasesAHold(t *testing.T) {
	repo, bare := reservingDrainRepo(t, "commands/drain.md")
	first, taken := askFirstOnce(t, repo, bare, DrainOptions{})
	fileFirstOnOrigin(t, bare, originIssue)
	was := claimReserve
	t.Cleanup(func() { claimReserve = was })
	var onDisk []*DrainJudging
	claimReserve = func(dir string, req claim.Request, o claim.Options) (claim.Result, error) {
		if req.Release && req.Record == secondIssue {
			st, _, err := readDrain(dir)
			if err != nil {
				t.Errorf("reading the drain state at the release: %v", err)
			}
			onDisk = append(onDisk, st.Judging)
		}
		return was(dir, req, o)
	}
	next := judge(t, repo, Options{}, first, JudgementNo, "", "the remedy changes neither")
	claimReserve = was
	if next.Judging == nil || next.Judging.Issue != originIssue {
		t.Fatalf("the move asks over the issue now heading the order: %+v", next.Judging)
	}
	releasedOver(t, bare, secondIssue, taken)
	if len(onDisk) == 0 {
		t.Fatal("the move pushed no release of " + secondIssue)
	}
	for _, j := range onDisk {
		if j == nil || j.Issue != originIssue {
			t.Fatalf("at the release the drain state on disk awaits %+v; want %s, as the move leaves it", j, originIssue)
		}
	}
}
