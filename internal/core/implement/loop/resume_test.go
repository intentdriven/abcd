package loop

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/core/machineload"
	"github.com/intentdriven/abcd/internal/core/occupancy"
	"github.com/intentdriven/abcd/internal/core/runner"
	"github.com/intentdriven/abcd/internal/core/runstop"
	"github.com/intentdriven/abcd/internal/core/sessionname"
	"github.com/intentdriven/abcd/internal/gittest"
	"strconv"
)

// A fixture process table: the walk starts at the shell, whose parent is a host.
const (
	hostOne = 4100
	hostTwo = 4300
	shellOf = 100 // added to a host's id for its shell
)

// procsOf stands a process table in, holding each host and its shell, so a host
// not named is one that has gone.
func procsOf(t *testing.T, here int, hosts ...int) {
	t.Helper()
	now := time.Now()
	tab := sessionname.Table{ReadAt: now}
	for _, h := range hosts {
		tab.Procs = append(tab.Procs,
			machineload.Proc{PID: h, PPID: 1, Name: "host", Age: time.Duration(h) * time.Second},
			machineload.Proc{PID: h + shellOf, PPID: h, Name: "zsh", Age: time.Second})
	}
	oldProcs, oldFrom, oldSelf := readProcs, hereFrom, hereSelf
	readProcs = func() (sessionname.Table, error) { return tab, nil }
	hereFrom = func() int { return here + shellOf }
	hereSelf = func() int { return here + shellOf }
	t.Cleanup(func() { readProcs, hereFrom, hereSelf = oldProcs, oldFrom, oldSelf })
}

// TestAClaimRecordsTheDriverAndRefusesASecond: the state of the run names the
// live driver; a second driver is refused while the first is alive, and replaces
// it, with a record entry, once it is gone.
func TestAClaimRecordsTheDriverAndRefusesASecond(t *testing.T) {
	repo, runID, _, _ := awaitingLane(t)
	home, _ := stopWorld(t, false)
	o := Options{stopHome: home, Session: "session-one"}

	procsOf(t, hostOne, hostOne, hostTwo)
	if err := ClaimDriver(repo.Root(), runID, o); err != nil {
		t.Fatal(err)
	}
	c := controlOf(t, repo.Root(), runID)
	if c.Driver == nil || c.Driver.PID != hostOne || c.Driver.Session != "session-one" || c.Driver.StartUnix == 0 {
		t.Fatalf("the driver: %+v", c.Driver)
	}
	// The same driver again is no change.
	if err := ClaimDriver(repo.Root(), runID, o); err != nil {
		t.Fatal(err)
	}

	procsOf(t, hostTwo, hostOne, hostTwo)
	err := ClaimDriver(repo.Root(), runID, Options{stopHome: home, Session: "session-two"})
	r, ok := AsRefusal(err)
	if !ok || r.Stage != StageDriver || !r.Contention || !strings.Contains(r.Reason, "session-one") {
		t.Fatalf("a second driver: %v", err)
	}

	procsOf(t, hostTwo, hostTwo)
	if err := ClaimDriver(repo.Root(), runID, Options{stopHome: home, Session: "session-two"}); err != nil {
		t.Fatal(err)
	}
	if c := controlOf(t, repo.Root(), runID); c.Driver.PID != hostTwo {
		t.Fatalf("the driver: %+v", c.Driver)
	}
	st, err := ReadState(repo.Root(), runID)
	if err != nil {
		t.Fatal(err)
	}
	if e := st.Record[len(st.Record)-1]; e.Stage != StageDriver || !strings.Contains(e.Note, "gone") {
		t.Fatalf("the record's last entry: %+v", e)
	}
	if got := runCheckouts(home); len(got) != 1 || got[0] != repo.Root() {
		t.Fatalf("the checkouts noted: %v", got)
	}
}

// TestALoopLeftBehindBacksOff: a step whose walk names no host, a loop its
// session left in the background, claims nothing and is refused while a driver is
// alive; with none, it is not stopped.
func TestALoopLeftBehindBacksOff(t *testing.T) {
	repo, runID, _, _ := awaitingLane(t)
	home, _ := stopWorld(t, false)
	o := Options{stopHome: home}
	procsOf(t, hostOne, hostOne)
	old := hereFrom
	orphan := func() int { return 1 }
	hereFrom = orphan
	t.Cleanup(func() { hereFrom = old })
	if err := ClaimDriver(repo.Root(), runID, o); err != nil {
		t.Fatalf("no driver is recorded: %v", err)
	}
	if c := controlOf(t, repo.Root(), runID); c.Driver != nil {
		t.Fatalf("a loop with no host claimed the run: %+v", c.Driver)
	}
	hereFrom = func() int { return hostOne + shellOf }
	if err := ClaimDriver(repo.Root(), runID, o); err != nil {
		t.Fatal(err)
	}
	hereFrom = orphan
	err := ClaimDriver(repo.Root(), runID, o)
	if r, ok := AsRefusal(err); !ok || r.Stage != StageDriver || !r.Contention {
		t.Fatalf("a loop beside a live driver: %v", err)
	}
}

// TestAnUnreadableTableClaimsNothing: a step is never stopped by the process
// listing.
func TestAnUnreadableTableClaimsNothing(t *testing.T) {
	repo, runID, _, _ := awaitingLane(t)
	home, _ := stopWorld(t, false)
	old := readProcs
	readProcs = func() (sessionname.Table, error) { return sessionname.Table{}, os.ErrPermission }
	t.Cleanup(func() { readProcs = old })
	if err := ClaimDriver(repo.Root(), runID, Options{stopHome: home}); err != nil {
		t.Fatal(err)
	}
	if c := controlOf(t, repo.Root(), runID); c.Driver != nil {
		t.Fatalf("the driver: %+v", c.Driver)
	}
}

// driverGone gives the run a driver whose process has gone.
func driverGone(t *testing.T, repoRoot, runID, home string) {
	t.Helper()
	procsOf(t, hostOne, hostOne)
	if err := ClaimDriver(repoRoot, runID, Options{stopHome: home, Session: "gone-session"}); err != nil {
		t.Fatal(err)
	}
	procsOf(t, hostTwo, hostTwo)
}

func theRun(t *testing.T, l Listing, runID string) ListedRun {
	t.Helper()
	for _, r := range l.Runs {
		if r.RunID == runID {
			return r
		}
	}
	t.Fatalf("the listing lacks %s: %+v", runID, l)
	return ListedRun{}
}

// TestTheListingIsWhatARunWroteDown: each lane's stage, an agent lost by its gone
// driver, the uncommitted work and the move, with nothing written.
func TestTheListingIsWhatARunWroteDown(t *testing.T) {
	repo, runID, _, dir, _ := deadLane(t)
	home, _ := stopWorld(t, false)
	driverGone(t, repo.Root(), runID, home)
	before, err := os.ReadFile(filepath.Join(repo.Root(), filepath.FromSlash(StateRelPath(runID))))
	if err != nil {
		t.Fatal(err)
	}

	l, err := ListRuns(repo.Root(), "", Options{stopHome: home})
	if err != nil {
		t.Fatal(err)
	}
	r := theRun(t, l, runID)
	if r.Driver == nil || r.Driver.State != "gone" || r.Driver.Session != "gone-session" {
		t.Fatalf("the driver: %+v", r.Driver)
	}
	if len(r.Lanes) != 1 || r.Lanes[0].Stage != StageImplement || r.Lanes[0].Uncommitted == nil || *r.Lanes[0].Uncommitted != 3 {
		t.Fatalf("the lane: %+v", r.Lanes)
	}
	a := r.Lanes[0].Agents
	if len(a) != 1 || a[0].State != AgentLost || !a[0].ReceiptArrived || a[0].Role != RoleImplementer {
		t.Fatalf("the agents: %+v", a)
	}
	if !strings.Contains(r.Next, "--take "+runID) {
		t.Fatalf("the move: %q", r.Next)
	}
	after, err := os.ReadFile(filepath.Join(repo.Root(), filepath.FromSlash(StateRelPath(runID))))
	if err != nil || string(after) != string(before) {
		t.Fatalf("the listing wrote the state: %v", err)
	}
	noAside(t, dir)
}

// TestATakeOverCollectsBeforeItRestarts: a receipt waiting at its path is
// collected and no fresh agent is told for it, whatever the driver left.
func TestATakeOverCollectsBeforeItRestarts(t *testing.T) {
	repo, runID, l, dir := awaitingLane(t)
	home, _ := stopWorld(t, false)
	c1 := laneCommit(t, repo, l, "one.txt")
	writeReceipt(t, dir, goodReceipt(t, runID, l, dir, c1))
	driverGone(t, repo.Root(), runID, home)

	res, err := TakeRun(repo.Root(), runID, Options{stopHome: home})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(res.Collected["lane-1"], "collected") || len(res.Restarted) != 0 {
		t.Fatalf("the take-over: %+v", res)
	}
	st, err := ReadState(repo.Root(), runID)
	if err != nil {
		t.Fatal(err)
	}
	if st.Lanes[0].Stage != StageValidate || len(st.Lanes[0].Awaits) != 0 {
		t.Fatalf("the lane: %+v", st.Lanes[0])
	}
	if d := controlOf(t, repo.Root(), runID).Driver; d == nil || d.PID != hostTwo {
		t.Fatalf("the driver is the taker: %+v", d)
	}
	noAside(t, dir)
}

// TestATakeOverSavesAsideBeforeAFreshAgent: an agent gone with work uncommitted
// and a partial receipt: the partial receipt does not verify and is named, the
// work is saved aside and named, the worktree is clean at its last commit, and
// the await is re-told once.
func TestATakeOverSavesAsideBeforeAFreshAgent(t *testing.T) {
	repo, runID, l, dir, head := deadLane(t)
	home, _ := stopWorld(t, false)
	driverGone(t, repo.Root(), runID, home)

	res, err := TakeRun(repo.Root(), runID, Options{stopHome: home})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Restarted) != 1 || res.Restarted[0].Aside.Head != head || len(res.Restarted[0].Aside.Files) != 3 {
		t.Fatalf("the restart: %+v", res.Restarted)
	}
	if res.Restarted[0].Awaiting == nil || res.Restarted[0].Awaiting.Role != RoleImplementer {
		t.Fatalf("the fresh agent is told: %+v", res.Restarted[0])
	}
	if !strings.Contains(res.Collected["lane-1"], "did not verify") {
		t.Fatalf("the partial receipt: %+v", res.Collected)
	}
	asideOf(t, dir)
	if status := repo.Git("-C", l.Worktree, "status", "--porcelain"); strings.TrimSpace(status) != "" {
		t.Fatalf("the worktree is clean:\n%s", status)
	}
	if res.Run == nil || len(res.Run.Lanes[0].Asides) != 1 {
		t.Fatalf("the listing names the aside: %+v", res.Run)
	}
}

// TestATakeOverRefusesOverWorkItCouldNotSave: a restart that cannot save the
// lane's work refuses, naming it, and names no step that would build on it.
func TestATakeOverRefusesOverWorkItCouldNotSave(t *testing.T) {
	repo, runID, l, dir, _ := deadLane(t)
	home, _ := stopWorld(t, false)
	driverGone(t, repo.Root(), runID, home)
	old := checkAsidePatch
	checkAsidePatch = func(string, string) error { return os.ErrPermission }
	t.Cleanup(func() { checkAsidePatch = old })
	res, err := TakeRun(repo.Root(), runID, Options{stopHome: home})
	if r, ok := AsRefusal(err); !ok || r.Stage != StageResume || !strings.Contains(r.Reason, "lane-1") {
		t.Fatalf("over work it could not save: %v", err)
	}
	if res.Next != "" {
		t.Fatalf("it names a move: %q", res.Next)
	}
	noAside(t, dir)
	editsIntact(t, repo, l)
}

// TestATakeOverIsRefusedWhileADriverIsAliveOrAStopHolds: nothing changes.
func TestATakeOverIsRefusedWhileADriverIsAliveOrAStopHolds(t *testing.T) {
	repo, runID, l, dir, _ := deadLane(t)
	home, _ := stopWorld(t, false)
	procsOf(t, hostOne, hostOne, hostTwo)
	if err := ClaimDriver(repo.Root(), runID, Options{stopHome: home}); err != nil {
		t.Fatal(err)
	}
	procsOf(t, hostTwo, hostOne, hostTwo)
	_, err := TakeRun(repo.Root(), runID, Options{stopHome: home})
	if r, ok := AsRefusal(err); !ok || r.Stage != StageDriver || !r.Contention {
		t.Fatalf("with its driver alive: %v", err)
	}
	if _, err := runstop.Set(home, time.Now()); err != nil {
		t.Fatal(err)
	}
	_, err = TakeRun(repo.Root(), runID, Options{stopHome: home})
	wantStopRefusal(t, err)
	noAside(t, dir)
	editsIntact(t, repo, l)
}

// onOriginAccepted gives the run's repository an origin, which the fixture adds
// after the run began, and accepts that change of its configuration.
func onOriginAccepted(t *testing.T, repo *gittest.Repo, runID string) string {
	t.Helper()
	bare := onOrigin(t, repo)
	if _, err := AcceptConfigChange(repo.Root(), runID, Options{}); err != nil {
		t.Fatal(err)
	}
	return bare
}

// specHasFootprint gives the fixture's spec the footprint a run reserves with.
func specHasFootprint(t *testing.T, repo *gittest.Repo) {
	t.Helper()
	specs, err := filepath.Glob(filepath.Join(repo.Root(), ".abcd", "development", "specs", "open", "spc-1*"))
	if err != nil || len(specs) != 1 {
		t.Fatalf("the fixture's spec: %v, %v", specs, err)
	}
	f, err := os.OpenFile(specs[0], os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if _, err := f.WriteString("\n## Footprint\n\n- packages: commands/x.md\n- tests: the widget\n"); err != nil {
		t.Fatal(err)
	}
}

// TestATakeOverAfterAStopReservesAgainFirst: the listing shows what the stop left
// and the reservation it released; the take-over reserves again, clears the mark,
// and the run steps on.
func TestATakeOverAfterAStopReservesAgainFirst(t *testing.T) {
	repo, runID, _, _, _ := deadLane(t)
	specHasFootprint(t, repo)
	bare := onOriginAccepted(t, repo, runID)
	now := time.Now().UTC().Truncate(time.Second)
	reserveRun(t, repo, runID, "itd-10", now, "commands/x.md")
	o, home, _ := stopped(t)
	later := now.Add(3 * time.Hour)
	o.Now = func() time.Time { return later }
	wantStopRefusal(t, checkOnce(repo.Root(), runID, o))
	if _, err := runstop.Lift(home, later); err != nil {
		t.Fatal(err)
	}

	l, err := ListRuns(repo.Root(), runID, o)
	if err != nil {
		t.Fatal(err)
	}
	r := theRun(t, l, runID)
	if r.Stop == nil || r.Stop.HaltedAt == nil || r.Reserved || r.Stop.Left[0].Aside == "" {
		t.Fatalf("the listing's account of the stop: %+v", r)
	}

	res, err := TakeRun(repo.Root(), runID, o)
	if err != nil {
		t.Fatal(err)
	}
	if res.Reservation == "" {
		t.Fatalf("the take-over reserves again: %+v", res)
	}
	if got := originReservation(t, bare, reservedRef); got.released || got.commit == "" {
		t.Fatalf("origin's reservation: %+v", got)
	}
	if c := controlOf(t, repo.Root(), runID); c.Stop != nil {
		t.Fatalf("the mark stands: %+v", c.Stop)
	}
	if err := checkOnce(repo.Root(), runID, o); err != nil {
		t.Fatalf("the run steps on: %v", err)
	}
}

// TestATakeOverBacksOffWhenAPeerTookTheWork: a stop released the claims so a peer
// may take them; the take-over then refuses at exit 3 with the run still marked,
// and starts nothing.
func TestATakeOverBacksOffWhenAPeerTookTheWork(t *testing.T) {
	repo, runID, _, _, _ := deadLane(t)
	specHasFootprint(t, repo)
	bare := onOriginAccepted(t, repo, runID)
	now := time.Now().UTC().Truncate(time.Second)
	reserveRun(t, repo, runID, "itd-10", now, "commands/x.md")
	o, home, _ := stopped(t)
	later := now.Add(3 * time.Hour)
	o.Now = func() time.Time { return later }
	wantStopRefusal(t, checkOnce(repo.Root(), runID, o))
	if _, err := runstop.Lift(home, later); err != nil {
		t.Fatal(err)
	}
	q := otherCopy(t, bare, quinnEmail)
	if _, err := claim.Reserve(q, claim.Request{Record: "itd-10", Footprint: []string{"commands/x.md"}}, as(t, q, clockAt(later))); err != nil {
		t.Fatal(err)
	}

	_, err := TakeRun(repo.Root(), runID, o)
	if r, ok := AsRefusal(err); !ok || !r.Contention {
		t.Fatalf("over a peer's reservation: %v", err)
	}
	if c := controlOf(t, repo.Root(), runID); c.Stop == nil {
		t.Fatal("the run's mark was cleared though the work is a peer's")
	}
	st, rerr := ReadState(repo.Root(), runID)
	if rerr != nil {
		t.Fatal(rerr)
	}
	if st.Lanes[0].Awaits[0].Kept == "" {
		t.Fatalf("the await was re-told: %+v", st.Lanes[0].Awaits[0])
	}
}

// TestATakeOverRefusedAtTheEndSaysWhatItDid: a lane it could not restart refuses after
// the driver was claimed, and says so.
func TestATakeOverRefusedAtTheEndSaysWhatItDid(t *testing.T) {
	repo, runID, _, _, _ := deadLane(t)
	home, _ := stopWorld(t, false)
	driverGone(t, repo.Root(), runID, home)
	old := checkAsidePatch
	checkAsidePatch = func(string, string) error { return errors.New("will not apply") }
	t.Cleanup(func() { checkAsidePatch = old })
	_, err := TakeRun(repo.Root(), runID, Options{stopHome: home})
	r, ok := AsRefusal(err)
	if !ok || !strings.Contains(r.Wrote, "driver") {
		t.Fatalf("the refusal: %+v, %v", r, err)
	}
}

// TestTheListingSaysHowFarALanesBaseHasMoved: the listing counts the commits
// origin's default branch, as this checkout last fetched it, holds past a lane's
// base (iss-2611040337228910), so a successor sees the drift before it steps; a
// default branch still at the base has moved none.
func TestTheListingSaysHowFarALanesBaseHasMoved(t *testing.T) {
	repo, runID, l, _ := awaitingLane(t)
	home, _ := stopWorld(t, false)
	procsOf(t, hostOne, hostOne)
	if len(l.BaseSHA) != 40 {
		t.Fatalf("the lane names no base: %+v", l)
	}
	moved := func() *int {
		t.Helper()
		got, err := ListRuns(repo.Root(), runID, Options{stopHome: home})
		if err != nil {
			t.Fatal(err)
		}
		r := theRun(t, got, runID)
		if len(r.Lanes) != 1 {
			t.Fatalf("the lanes: %+v", r.Lanes)
		}
		return r.Lanes[0].BaseMoved
	}

	repo.Git("update-ref", "refs/remotes/origin/main", l.BaseSHA)
	if n := moved(); n == nil || *n != 0 {
		t.Fatalf("a default branch at the lane's base moved %v commit(s)", n)
	}
	tip, tree := l.BaseSHA, strings.TrimSpace(repo.Git("rev-parse", l.BaseSHA+"^{tree}"))
	for i := range 3 {
		tip = strings.TrimSpace(repo.Git("commit-tree", tree, "-p", tip, "-m", "landed "+strconv.Itoa(i)))
	}
	repo.Git("update-ref", "refs/remotes/origin/main", tip)
	if n := moved(); n == nil || *n != 3 {
		t.Fatalf("three commits past the base read as %v", n)
	}
}

// TestTheListingNamesAJudgement: a drain awaiting a judgement is named with the
// move that gives it, so the entry a stopped session left is never a dead end.
func TestTheListingNamesAJudgement(t *testing.T) {
	repo, bare := reservingDrainRepo(t, "commands/drain.md")
	first, _ := askFirstOnce(t, repo, bare, DrainOptions{})
	home, _ := stopWorld(t, false)
	l, err := ListRuns(repo.Root(), "", Options{stopHome: home})
	if err != nil {
		t.Fatal(err)
	}
	if len(l.Drains) != 1 || l.Drains[0].Judging != first.Judging.Issue || !strings.Contains(l.Drains[0].Next, "--judgement") {
		t.Fatalf("the drains: %+v", l.Drains)
	}
}

// TestADriverIsNotRefusedByItsOwnLaterReading: a process's start time is measured
// back from each reading, so two readings of one driver differ by a second. A
// driver's next step is not a second driver.
func TestADriverIsNotRefusedByItsOwnLaterReading(t *testing.T) {
	repo, runID, _, _ := awaitingLane(t)
	home, _ := stopWorld(t, false)
	o := Options{stopHome: home}
	procsOf(t, hostOne, hostOne)
	if err := ClaimDriver(repo.Root(), runID, o); err != nil {
		t.Fatal(err)
	}
	first, err := readProcs()
	if err != nil {
		t.Fatal(err)
	}
	later := first
	later.ReadAt = first.ReadAt.Add(1100 * time.Millisecond)
	readProcs = func() (sessionname.Table, error) { return later, nil }
	if err := ClaimDriver(repo.Root(), runID, o); err != nil {
		t.Fatalf("the driver's own next step: %v", err)
	}
}

// TestAClaimJudgesTheDriverItFindsUnderTheLock: a driver that claimed after this
// call read the process table is not in that reading, and is not gone.
func TestAClaimJudgesTheDriverItFindsUnderTheLock(t *testing.T) {
	repo, runID, _, _ := awaitingLane(t)
	home, _ := stopWorld(t, false)
	procsOf(t, hostTwo, hostOne, hostTwo)
	if err := ClaimDriver(repo.Root(), runID, Options{stopHome: home}); err != nil {
		t.Fatal(err)
	}
	withBoth, err := readProcs()
	if err != nil {
		t.Fatal(err)
	}
	stale := withBoth
	stale.Procs = nil
	for _, p := range withBoth.Procs {
		if p.PID == hostOne || p.PID == hostOne+shellOf {
			stale.Procs = append(stale.Procs, p)
		}
	}
	reads := 0
	readProcs = func() (sessionname.Table, error) {
		reads++
		if reads == 1 {
			return stale, nil
		}
		return withBoth, nil
	}
	hereFrom = func() int { return hostOne + shellOf }
	err = ClaimDriver(repo.Root(), runID, Options{stopHome: home})
	if r, ok := AsRefusal(err); !ok || r.Stage != StageDriver || !r.Contention {
		t.Fatalf("over a driver the first reading could not hold: %v", err)
	}
	if c := controlOf(t, repo.Root(), runID); c.Driver.PID != hostTwo {
		t.Fatalf("the live driver was replaced: %+v", c.Driver)
	}
}

// TestATakeOverRefusesADriverItCannotJudge: a take-over resets worktrees, so a
// recorded driver the process table cannot judge is not taken for gone.
func TestATakeOverRefusesADriverItCannotJudge(t *testing.T) {
	repo, runID, l, dir, _ := deadLane(t)
	home, _ := stopWorld(t, false)
	procsOf(t, hostOne, hostOne)
	if err := ClaimDriver(repo.Root(), runID, Options{stopHome: home}); err != nil {
		t.Fatal(err)
	}
	readProcs = func() (sessionname.Table, error) { return sessionname.Table{}, errors.New("no process table") }
	_, err := TakeRun(repo.Root(), runID, Options{stopHome: home})
	if r, ok := AsRefusal(err); !ok || r.Stage != StageDriver || !r.Contention {
		t.Fatalf("with its driver unjudged: %v", err)
	}
	noAside(t, dir)
	editsIntact(t, repo, l)
}

// TestATakeOverTakesNoGitStepOverAChangedConfiguration: saving a lane runs git in
// its worktree, so a configuration changed during the run refuses it.
func TestATakeOverTakesNoGitStepOverAChangedConfiguration(t *testing.T) {
	repo, runID, l, dir, _ := deadLane(t)
	home, _ := stopWorld(t, false)
	if err := mutate(repo.Root(), runID, func(_ *os.Root, st *State) (bool, error) {
		st.Lanes[0].Awaits = nil
		return true, nil
	}); err != nil {
		t.Fatal(err)
	}
	repo.Git("config", "core.fsmonitor", "/tmp/planted")
	if _, err := TakeRun(repo.Root(), runID, Options{stopHome: home}); err == nil {
		t.Fatal("the take-over stepped over a changed configuration")
	}
	noAside(t, dir)
	editsIntact(t, repo, l)
}

// TestAFinishedRecordClearsTheMark: a stop let the claim go and the record then
// finished on the default branch, an armed pull request merging. There is nothing to
// reserve, so the take-over clears the mark and the run ends as it would have.
func TestAFinishedRecordClearsTheMark(t *testing.T) {
	repo, runID, _, _, _ := deadLane(t)
	specHasFootprint(t, repo)
	bare := onOriginAccepted(t, repo, runID)
	now := time.Now().UTC().Truncate(time.Second)
	reserveRun(t, repo, runID, "itd-10", now, "commands/x.md")
	o, home, _ := stopped(t)
	later := now.Add(3 * time.Hour)
	o.Now = func() time.Time { return later }
	wantStopRefusal(t, checkOnce(repo.Root(), runID, o))
	if _, err := runstop.Lift(home, later); err != nil {
		t.Fatal(err)
	}
	q := otherCopy(t, bare, patEmail)
	found, err := filepath.Glob(filepath.Join(q, ".abcd", "development", "intents", "*", "itd-10*"))
	if err != nil || len(found) != 1 {
		t.Fatalf("the fixture's intent: %v, %v", found, err)
	}
	shipped := filepath.Join(q, ".abcd", "development", "intents", "shipped")
	if err := os.MkdirAll(shipped, 0o755); err != nil {
		t.Fatal(err)
	}
	gitIn(t, q, "mv", found[0], filepath.Join(shipped, filepath.Base(found[0])))
	gitIn(t, q, "commit", "-q", "-m", "ship")
	gitIn(t, q, "push", "-q", "origin", "HEAD:main")

	if _, err := TakeRun(repo.Root(), runID, o); err != nil {
		t.Fatalf("the take-over of a finished record: %v", err)
	}
	if c := controlOf(t, repo.Root(), runID); c.Stop != nil {
		t.Fatalf("the mark stands over a finished record: %+v", c.Stop)
	}
}

// TestAStepSettlesAStopBeforeItClaims: a successor's step under a stop judges the
// agents by the driver they ran under, so a dead host's agent is lost, not taken to
// be in its call under the successor; and it claims nothing.
func TestAStepSettlesAStopBeforeItClaims(t *testing.T) {
	repo, runID, _, dir, _ := deadLane(t)
	home, _ := stopWorld(t, true)
	o := Options{stopHome: home}
	procsOf(t, hostOne, hostOne)
	if err := ClaimDriver(repo.Root(), runID, Options{stopHome: home, Session: "gone-session"}); err != nil {
		t.Fatal(err)
	}
	if _, err := runstop.Set(home, time.Now()); err != nil {
		t.Fatal(err)
	}
	procsOf(t, hostTwo, hostTwo)
	if err := ClaimDriver(repo.Root(), runID, o); err != nil {
		t.Fatal(err)
	}
	c := controlOf(t, repo.Root(), runID)
	if c.Stop == nil || c.Stop.HaltedAt == nil {
		t.Fatalf("the dead host's agent held the stop: %+v", c.Stop)
	}
	if c.Driver == nil || c.Driver.PID != hostOne {
		t.Fatalf("the step claimed the run under a stop: %+v", c.Driver)
	}
	asideOf(t, dir)
}

// TestARecordedDriverIsTheSessionNotItsWrapper: a one-shot program between a host and
// a step does not stand as the driver, so its exit does not read as the host gone.
func TestARecordedDriverIsTheSessionNotItsWrapper(t *testing.T) {
	repo, runID, _, _ := awaitingLane(t)
	home, _ := stopWorld(t, false)
	now := time.Now()
	const host, wrapper, shell = 4100, 4150, 4160
	tab := sessionname.Table{ReadAt: now, Procs: []machineload.Proc{
		{PID: host, PPID: 1, Name: "host", Age: time.Hour},
		{PID: wrapper, PPID: host, Name: "timeout", Age: time.Second},
		{PID: shell, PPID: wrapper, Name: "zsh", Age: time.Second},
	}}
	if err := occupancy.Note(home, "s-one", repo.Root(), sessionname.Host{PID: host, StartUnix: now.Add(-time.Hour).Unix()}, tab); err != nil {
		t.Fatal(err)
	}
	oldProcs, oldFrom := readProcs, hereFrom
	t.Cleanup(func() { readProcs, hereFrom = oldProcs, oldFrom })
	readProcs = func() (sessionname.Table, error) { return tab, nil }
	hereFrom = func() int { return shell }
	if err := ClaimDriver(repo.Root(), runID, Options{stopHome: home}); err != nil {
		t.Fatal(err)
	}
	if c := controlOf(t, repo.Root(), runID); c.Driver == nil || c.Driver.PID != host || c.Driver.Session != "s-one" {
		t.Fatalf("the driver: %+v", c.Driver)
	}
}

// TestATakeOverChecksTheConfigurationFirst: a take-over runs git from its first move, so
// a changed configuration refuses it before it collects anything.
func TestATakeOverChecksTheConfigurationFirst(t *testing.T) {
	repo, runID, l, dir := awaitingLane(t)
	c1 := laneCommit(t, repo, l, "one.txt")
	writeReceipt(t, dir, goodReceipt(t, runID, l, dir, c1))
	home, _ := stopWorld(t, false)
	repo.Git("config", "core.fsmonitor", "/tmp/planted")
	if _, err := TakeRun(repo.Root(), runID, Options{stopHome: home}); err == nil {
		t.Fatal("the take-over stepped over a changed configuration")
	}
	st, err := ReadState(repo.Root(), runID)
	if err != nil {
		t.Fatal(err)
	}
	if len(st.Lanes[0].Awaits) == 0 {
		t.Fatalf("it collected first: %+v", st.Lanes[0])
	}
	if c := controlOf(t, repo.Root(), runID); c.Driver != nil {
		t.Fatalf("it claimed first: %+v", c.Driver)
	}
}

// TestARefusedTakeOverSaysWhatItDid: a reservation refused after the driver was claimed
// and receipts collected says so.
func TestARefusedTakeOverSaysWhatItDid(t *testing.T) {
	repo, runID, _, _, _ := deadLane(t)
	specHasFootprint(t, repo)
	bare := onOriginAccepted(t, repo, runID)
	now := time.Now().UTC().Truncate(time.Second)
	reserveRun(t, repo, runID, "itd-10", now, "commands/x.md")
	o, home, _ := stopped(t)
	later := now.Add(3 * time.Hour)
	o.Now = func() time.Time { return later }
	wantStopRefusal(t, checkOnce(repo.Root(), runID, o))
	if _, err := runstop.Lift(home, later); err != nil {
		t.Fatal(err)
	}
	q := otherCopy(t, bare, quinnEmail)
	if _, err := claim.Reserve(q, claim.Request{Record: "itd-10", Footprint: []string{"commands/x.md"}}, as(t, q, clockAt(later))); err != nil {
		t.Fatal(err)
	}
	procsOf(t, hostTwo, hostTwo)
	_, err := TakeRun(repo.Root(), runID, o)
	r, ok := AsRefusal(err)
	if !ok || !strings.Contains(r.Wrote, "driver") {
		t.Fatalf("the refusal: %+v, %v", r, err)
	}
}

// TestAHaltedRunIsNotHeldByItsIdleDriver: every lane has settled, so the listing names
// the take-over and an idle driver's session does not stand in its way.
func TestAHaltedRunIsNotHeldByItsIdleDriver(t *testing.T) {
	repo, runID, _, _, _ := deadLane(t)
	o, home, _ := stopped(t)
	later := time.Now().Add(3 * time.Hour)
	o.Now = func() time.Time { return later }
	procsOf(t, hostOne, hostOne, hostTwo)
	if err := claimDriver(repo.Root(), runID, Options{stopHome: home, Session: "idle", Now: o.Now}, false); err != nil {
		t.Fatal(err)
	}
	if err := mutate(repo.Root(), runID, func(_ *os.Root, st *State) (bool, error) {
		st.Lanes[0].Awaits[0].Kept = "gone"
		return true, nil
	}); err != nil {
		t.Fatal(err)
	}
	wantStopRefusal(t, checkOnce(repo.Root(), runID, o))
	if c := controlOf(t, repo.Root(), runID); c.Stop == nil || c.Stop.HaltedAt == nil {
		t.Fatalf("the run did not halt: %+v", c.Stop)
	}
	if _, err := runstop.Lift(home, later); err != nil {
		t.Fatal(err)
	}
	procsOf(t, hostTwo, hostOne, hostTwo)
	l, err := ListRuns(repo.Root(), runID, o)
	if err != nil {
		t.Fatal(err)
	}
	if next := theRun(t, l, runID).Next; !strings.Contains(next, "--take") {
		t.Fatalf("the listing says %q", next)
	}
	if _, err := TakeRun(repo.Root(), runID, o); err != nil {
		t.Fatalf("the take-over: %v", err)
	}
	st, err := ReadState(repo.Root(), runID)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range st.Record {
		if e.Stage == StageDriver && strings.Contains(e.Note, "is gone") {
			t.Fatalf("the record calls a live driver gone: %q", e.Note)
		}
	}
}

// TestAClaimGivesWayOnlyOverAHaltedRun: what lets an idle driver give way is read under
// the lock, so a run no stop has halted keeps its driver.
func TestAClaimGivesWayOnlyOverAHaltedRun(t *testing.T) {
	repo, runID, _, _ := awaitingLane(t)
	home, _ := stopWorld(t, false)
	procsOf(t, hostOne, hostOne, hostTwo)
	if err := ClaimDriver(repo.Root(), runID, Options{stopHome: home}); err != nil {
		t.Fatal(err)
	}
	procsOf(t, hostTwo, hostOne, hostTwo)
	err := claimDriver(repo.Root(), runID, Options{stopHome: home}, true)
	if r, ok := AsRefusal(err); !ok || r.Stage != StageDriver {
		t.Fatalf("over a run no stop has halted: %v", err)
	}
}

// TestTheListingSanitisesWhatItReads: a control record's text is made safe before it is
// written out, in JSON too.
func TestTheListingSanitisesWhatItReads(t *testing.T) {
	repo, runID, _, _ := awaitingLane(t)
	home, _ := stopWorld(t, false)
	if err := mutate(repo.Root(), runID, func(r *os.Root, _ *State) (bool, error) {
		at := time.Now()
		return false, writeControl(r, runID, control{
			Driver: &LiveDriver{Session: "s\x1b[31m", PID: 1, At: at},
			Stop:   &RunStop{SetBy: "p\x1b[31m", SetAt: at, NotedAt: at, Released: "r\x1b[31m", Left: []StopLeft{{Lane: "l\x1b[31m", Stage: "s\x1b[31m", Note: "n\x1b[31m"}}},
		})
	}); err != nil {
		t.Fatal(err)
	}
	dir, err := laneRel(runID, "lane-1", stageRestart)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(repo.Root(), filepath.FromSlash(dir), AsideDirName, "a\x1b[31m"), 0o700); err != nil {
		t.Fatal(err)
	}
	l, err := ListRuns(repo.Root(), runID, Options{stopHome: home})
	if err != nil {
		t.Fatal(err)
	}
	r := theRun(t, l, runID)
	if len(r.Lanes) == 0 || len(r.Lanes[0].Asides) != 1 {
		t.Fatalf("the listing's lanes: %+v", r.Lanes)
	}
	for _, got := range []string{r.Driver.Session, r.Stop.SetBy, r.Stop.Released, r.Stop.Left[0].Note, r.Stop.Left[0].Lane, string(r.Stop.Left[0].Stage), r.Lanes[0].Asides[0]} {
		if strings.ContainsRune(got, 0x1b) {
			t.Fatalf("%q went out as it was read", got)
		}
	}
}

// TestTheListingNamesAStopAnotherAccountLeft: which is not honoured, and is said.
func TestTheListingNamesAStopAnotherAccountLeft(t *testing.T) {
	repo, _, _, _ := awaitingLane(t)
	home, _ := stopWorld(t, false)
	folder := t.TempDir()
	dir := filepath.Join(folder, runstop.StopDirName+"-"+strconv.Itoa(os.Getuid()+1))
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, runstop.StopFileName), []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	old := runstop.WellKnownFolder
	runstop.WellKnownFolder = func() string { return folder }
	t.Cleanup(func() { runstop.WellKnownFolder = old })
	l, err := ListRuns(repo.Root(), "", Options{stopHome: home})
	if err != nil {
		t.Fatal(err)
	}
	if len(l.Notices) != 1 || l.Stop != nil {
		t.Fatalf("the listing: %+v", l)
	}
}

// TestASecondTakerDoesNotDisplaceTheFirst: only a driver that claimed before the halt
// gives way. One that claimed since is a take-over at work, however halted the run
// still reads while it reserves.
func TestASecondTakerDoesNotDisplaceTheFirst(t *testing.T) {
	repo, runID, _, _ := awaitingLane(t)
	home, _ := stopWorld(t, false)
	halted := time.Now().UTC().Truncate(time.Second)
	if err := mutate(repo.Root(), runID, func(r *os.Root, _ *State) (bool, error) {
		c, err := readControl(r, runID)
		if err != nil {
			return false, err
		}
		c.Stop = &RunStop{SetBy: "a person", SetAt: halted, NotedAt: halted, HaltedAt: &halted, FromFile: true}
		return false, writeControl(r, runID, c)
	}); err != nil {
		t.Fatal(err)
	}
	procsOf(t, hostOne, hostOne, hostTwo)
	first := Options{stopHome: home, Session: "first", Now: func() time.Time { return halted.Add(time.Minute) }}
	if err := claimDriver(repo.Root(), runID, first, true); err != nil {
		t.Fatal(err)
	}
	procsOf(t, hostTwo, hostOne, hostTwo)
	second := Options{stopHome: home, Session: "second", Now: func() time.Time { return halted.Add(2 * time.Minute) }}
	err := claimDriver(repo.Root(), runID, second, true)
	if r, ok := AsRefusal(err); !ok || r.Stage != StageDriver {
		t.Fatalf("the second taker: %v", err)
	}
	if c := controlOf(t, repo.Root(), runID); c.Driver == nil || c.Driver.Session != "first" {
		t.Fatalf("the driver: %+v", c.Driver)
	}
}

// TestAStepAfterTheLiftLeavesTheTakeOverOpen: a step cannot move a run a stop halted, so
// the session that tried claims nothing and another may take the run over.
func TestAStepAfterTheLiftLeavesTheTakeOverOpen(t *testing.T) {
	repo, runID, _, _, _ := deadLane(t)
	o, home, _ := stopped(t)
	later := time.Now().Add(3 * time.Hour)
	o.Now = func() time.Time { return later }
	wantStopRefusal(t, checkOnce(repo.Root(), runID, o))
	if c := controlOf(t, repo.Root(), runID); c.Stop == nil || c.Stop.HaltedAt == nil {
		t.Fatalf("the run did not halt: %+v", c.Stop)
	}
	if _, err := runstop.Lift(home, later); err != nil {
		t.Fatal(err)
	}
	procsOf(t, hostOne, hostOne, hostTwo)
	step := Options{stopHome: home, Session: "stepping", Now: func() time.Time { return later.Add(time.Minute) }}
	if err := ClaimDriver(repo.Root(), runID, step); err != nil {
		t.Fatalf("the step's claim: %v", err)
	}
	_, err := advance(repo.Root(), runID, DefaultStages(), step)
	if r := wantStopRefusal(t, err); !strings.Contains(r.Remedy, "implement resume --take") {
		t.Fatalf("the step's refusal names %q", r.Remedy)
	}
	if c := controlOf(t, repo.Root(), runID); c.Driver != nil && c.Driver.Session == "stepping" {
		t.Fatalf("the step claimed a run it cannot move: %+v", c.Driver)
	}
	procsOf(t, hostTwo, hostOne, hostTwo)
	take := Options{stopHome: home, Session: "taking", Now: func() time.Time { return later.Add(2 * time.Minute) }}
	if _, err := TakeRun(repo.Root(), runID, take); err != nil {
		t.Fatalf("the take-over: %v", err)
	}
	if c := controlOf(t, repo.Root(), runID); c.Driver == nil || c.Driver.Session != "taking" {
		t.Fatalf("the driver: %+v", c.Driver)
	}
}

// TestATakerThatWasRefusedIsTheOneToRetryFrom: it holds others out, so the listing names
// it rather than sending them to a take-over they would be refused; it can retry.
func TestATakerThatWasRefusedIsTheOneToRetryFrom(t *testing.T) {
	repo, runID, _, _ := awaitingLane(t)
	home, _ := stopWorld(t, false)
	halted := time.Now().UTC().Truncate(time.Second)
	if err := mutate(repo.Root(), runID, func(r *os.Root, _ *State) (bool, error) {
		c, err := readControl(r, runID)
		if err != nil {
			return false, err
		}
		c.Stop = &RunStop{SetBy: "a person", SetAt: halted, NotedAt: halted, HaltedAt: &halted, FromFile: true}
		return false, writeControl(r, runID, c)
	}); err != nil {
		t.Fatal(err)
	}
	procsOf(t, hostOne, hostOne, hostTwo)
	first := Options{stopHome: home, Session: "first", Now: func() time.Time { return halted.Add(time.Minute) }}
	if err := claimDriver(repo.Root(), runID, first, true); err != nil {
		t.Fatal(err)
	}
	procsOf(t, hostTwo, hostOne, hostTwo)
	l, err := ListRuns(repo.Root(), runID, Options{stopHome: home})
	if err != nil {
		t.Fatal(err)
	}
	next := theRun(t, l, runID).Next
	for _, want := range []string{"session first", "again from that session", "end that session"} {
		if !strings.Contains(next, want) {
			t.Fatalf("the listing says %q, lacking %q", next, want)
		}
	}
	procsOf(t, hostOne, hostOne, hostTwo)
	if err := claimDriver(repo.Root(), runID, first, true); err != nil {
		t.Fatalf("its retry: %v", err)
	}
}

// TestWhatARunFileNamesIsMadeSafe: lane ids and roles are not validated when a state is
// read, so what a listing and a take-over say of them is made safe first.
func TestWhatARunFileNamesIsMadeSafe(t *testing.T) {
	const bad = "x\x1b[31m"
	d := listDrain("/tmp/"+bad, DrainState{Judging: &DrainJudging{Issue: bad}})
	res := safeTake(TakeResult{
		Collected: map[string]string{bad: bad},
		Left:      []string{bad},
		Saved:     []RestartAside{{Lane: bad, Path: bad}},
		Restarted: []RestartResult{{StepResult: StepResult{Lane: bad}, Aside: RestartAside{Lane: bad, Path: bad}}},
	})
	refused, ok := AsRefusal(takeFailed(bad, []string{bad + "'s work could not be saved"}))
	if !ok {
		t.Fatal("no refusal")
	}
	said := []string{d.Checkout, d.Judging, d.Next, res.Left[0], res.Saved[0].Lane, res.Saved[0].Path, res.Restarted[0].Lane, res.Restarted[0].Aside.Path, refused.Reason, refused.Remedy}
	for k, v := range res.Collected {
		said = append(said, k, v)
	}
	for _, got := range said {
		if strings.ContainsRune(got, 0x1b) {
			t.Fatalf("%q went out as it was read", got)
		}
	}
}

// The launch ledger's word on an agent (spc-2611050505250705 step 2): an agent a
// step started through a runner is alive or lost by the process its ledger entry
// names, not by the step that started it.

// harnessPID is the fixture's harness: the agent's own process.
const harnessPID = 5200

// runnerClaimed gives the lane's await a runner's claim, inside its term, with the
// launch the run recorded for it: started at at by a step whose process, hostOne,
// leads its own group apart from the agent's. stepPID 0 is a launch whose step's
// process could not be read.
func runnerClaimed(t *testing.T, repoRoot, runID, receipt string, at time.Time, stepPID int) {
	t.Helper()
	if err := mutate(repoRoot, runID, func(root *os.Root, st *State) (bool, error) {
		i, k := st.findAwait(repoRoot, receipt)
		if i < 0 {
			t.Fatalf("no await at %s", receipt)
		}
		st.Lanes[i].Awaits[k].Running = &RunnerClaim{Runner: runner.Claude, Session: "the-claim", Until: at.Add(time.Hour)}
		c, err := readControl(root, runID)
		if err != nil {
			return false, err
		}
		l := Launch{Lane: st.Lanes[i].ID, Role: st.Lanes[i].Awaits[k].Role, Runner: runner.Claude, Session: "the-claim", At: at}
		if stepPID > 0 {
			l.PID, l.StartUnix = stepPID, at.Add(-time.Duration(stepPID)*time.Second).Unix()
		}
		c.Launches = append(c.Launches, l)
		return true, writeControl(root, runID, c)
	}); err != nil {
		t.Fatal(err)
	}
}

// ledgerHolds writes one launch's entry into home's ledger, as the runner's
// launch does: e's run, lane and await name the work, and its id says when it
// started.
func ledgerHolds(t *testing.T, home string, e runner.LedgerEntry) string {
	t.Helper()
	e.Schema = runner.LedgerSchema
	e.ID = fmt.Sprintf("lch-%s%06d-%08x", e.StartedAt.UTC().Format("060102150405"), e.StartedAt.Nanosecond()/1000, e.PID)
	if e.Role == "" {
		e.Role, e.Runner, e.Route = RoleImplementer, runner.Claude, runner.RouteUnknown
	}
	line, err := json.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}
	dir := abcdhome.Path(home, "agents")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, e.ID+".json"), append(line, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	return e.ID
}

// harnessRunning adds the agent's own process, started at started, to the table
// procsOf stood in.
func harnessRunning(t *testing.T, started time.Time) {
	t.Helper()
	prev := readProcs
	readProcs = func() (sessionname.Table, error) {
		tab, err := prev()
		tab.Procs = append(slices.Clone(tab.Procs), machineload.Proc{PID: harnessPID, PPID: 1, Name: "claude", Age: tab.ReadAt.Sub(started)})
		return tab, err
	}
	t.Cleanup(func() { readProcs = prev })
}

// TestTheListingJudgesARunnersAgentByItsOwnProcess: the step that started the agent
// is gone and the claim's term runs, which the run's own record can only call
// unknown; the ledger's entry for the launch names the agent's own process, alive
// while that process runs and lost once it has gone. An entry an earlier attempt
// left speaks for that attempt alone.
func TestTheListingJudgesARunnersAgentByItsOwnProcess(t *testing.T) {
	repo, runID, l, _ := awaitingLane(t)
	home, _ := stopWorld(t, false)
	at := time.Now().Add(-time.Minute)
	receipt := l.awaiting().Receipt
	runnerClaimed(t, repo.Root(), runID, receipt, at, hostOne)
	procsOf(t, hostTwo, hostTwo)
	agent := func() ListedAgent {
		t.Helper()
		got, err := ListRuns(repo.Root(), runID, Options{stopHome: home})
		if err != nil {
			t.Fatal(err)
		}
		r := theRun(t, got, runID)
		if len(r.Lanes) != 1 || len(r.Lanes[0].Agents) != 1 {
			t.Fatalf("the lanes: %+v", r.Lanes)
		}
		return r.Lanes[0].Agents[0]
	}
	if a := agent(); a.State != AgentUnknown || a.Process != 0 || a.Launch != "" {
		t.Fatalf("with no entry in the ledger, the run's record judges it: %+v", a)
	}

	started := at.Add(2 * time.Second)
	work := runner.LedgerEntry{Repo: repo.Root(), Run: runID, Lane: l.ID, Await: receipt, Outcome: runner.OutcomeRunning}
	earlier := work
	earlier.StartedAt, earlier.PID, earlier.PIDStartUnix = at.Add(-time.Hour), harnessPID, started.Unix()
	ledgerHolds(t, home, earlier)
	harnessRunning(t, started)
	if a := agent(); a.State != AgentUnknown || a.Launch != "" {
		t.Fatalf("an earlier attempt's entry judged this one's agent: %+v", a)
	}

	work.StartedAt, work.PID, work.PIDStartUnix = at.Add(time.Second), harnessPID, started.Unix()
	id := ledgerHolds(t, home, work)
	if a := agent(); a.State != AgentAlive || a.Process != harnessPID || a.Launch != id {
		t.Fatalf("its own process runs: %+v", a)
	}
	procsOf(t, hostTwo, hostTwo)
	if a := agent(); a.State != AgentLost || a.Launch != id {
		t.Fatalf("its own process has gone: %+v", a)
	}
}

// TestATakeOverRestartsAnAgentWhoseOwnProcessIsGone: with the step gone inside the
// claim's term, a take-over restarts the lane once the ledger says the agent's own
// process has gone, saving its work aside first, and leaves the lane exactly as it
// is while that process runs.
func TestATakeOverRestartsAnAgentWhoseOwnProcessIsGone(t *testing.T) {
	for _, alive := range []bool{false, true} {
		t.Run(map[bool]string{false: "gone", true: "running"}[alive], func(t *testing.T) {
			repo, runID, l, dir, head := deadLane(t)
			home, _ := stopWorld(t, false)
			at := time.Now().Add(-time.Minute)
			runnerClaimed(t, repo.Root(), runID, l.awaiting().Receipt, at, hostOne)
			driverGone(t, repo.Root(), runID, home)
			started := at.Add(2 * time.Second)
			ledgerHolds(t, home, runner.LedgerEntry{Repo: repo.Root(), Run: runID, Lane: l.ID, Await: l.awaiting().Receipt,
				Outcome: runner.OutcomeRunning, StartedAt: at.Add(time.Second), PID: harnessPID, PIDStartUnix: started.Unix()})
			if alive {
				harnessRunning(t, started)
			}

			res, err := TakeRun(repo.Root(), runID, Options{stopHome: home})
			if err != nil {
				t.Fatal(err)
			}
			if alive {
				if len(res.Restarted) != 0 || len(res.Left) != 0 {
					t.Fatalf("an agent whose own process runs was touched: %+v", res)
				}
				noAside(t, dir)
				editsIntact(t, repo, l)
				return
			}
			if len(res.Restarted) != 1 || res.Restarted[0].Aside.Head != head || len(res.Restarted[0].Aside.Files) != 3 {
				t.Fatalf("the restart: %+v (left %v)", res.Restarted, res.Left)
			}
			asideOf(t, dir)
		})
	}
}

// TestATakeOverCollectsWhatTheLedgerSaysHasEnded: the step's process was never
// read, so the run's own record takes the agent to be running for its claim's
// term; the ledger says its launch ended, so its receipt is collected and no fresh
// agent is told for it.
func TestATakeOverCollectsWhatTheLedgerSaysHasEnded(t *testing.T) {
	repo, runID, l, dir := awaitingLane(t)
	home, _ := stopWorld(t, false)
	writeReceipt(t, dir, goodReceipt(t, runID, l, dir, laneCommit(t, repo, l, "one.txt")))
	at := time.Now().Add(-time.Minute)
	runnerClaimed(t, repo.Root(), runID, l.awaiting().Receipt, at, 0)
	driverGone(t, repo.Root(), runID, home)
	ended := time.Now().Add(-time.Second)
	ledgerHolds(t, home, runner.LedgerEntry{Repo: repo.Root(), Run: runID, Lane: l.ID, Await: l.awaiting().Receipt,
		Outcome: runner.OutcomeFinished, StartedAt: at.Add(time.Second), EndedAt: &ended, PID: harnessPID})

	res, err := TakeRun(repo.Root(), runID, Options{stopHome: home})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(res.Collected[l.ID], "collected") || len(res.Restarted) != 0 {
		t.Fatalf("the take-over: %+v", res)
	}
	st, err := ReadState(repo.Root(), runID)
	if err != nil {
		t.Fatal(err)
	}
	if st.Lanes[0].Stage != StageValidate {
		t.Fatalf("the lane: %+v", st.Lanes[0])
	}
	noAside(t, dir)
}

// forgeLists stands the forge's one listing of pull requests in, counting each
// read, so a test sees what the take-over read and how often.
func forgeLists(t *testing.T, listed map[int]listedPR, unread string) *int {
	t.Helper()
	calls := 0
	old := takeForgePRs
	takeForgePRs = func(string) (map[int]listedPR, string) {
		calls++
		return listed, unread
	}
	t.Cleanup(func() { takeForgePRs = old })
	return &calls
}

// prLane gives the run's one lane pull request #7 on its branch, as a landing
// that opened it and then lost its driver leaves it.
func prLane(t *testing.T, repoRoot, runID string) string {
	t.Helper()
	branch := ""
	if err := mutate(repoRoot, runID, func(_ *os.Root, st *State) (bool, error) {
		st.Lanes[0].PR = 7
		branch = st.Lanes[0].Branch
		return true, nil
	}); err != nil {
		t.Fatal(err)
	}
	return branch
}

// TestATakeOverReadsEachPullRequestFromTheForge: a pull request a crash left
// open drifts while nobody looks (iss-2610241414414823), so the take-over reads
// its state from the forge once and names one that conflicts with the default
// branch, where the bare listing reads nothing over the network.
func TestATakeOverReadsEachPullRequestFromTheForge(t *testing.T) {
	repo, runID, l, dir := awaitingLane(t)
	home, _ := stopWorld(t, false)
	writeReceipt(t, dir, goodReceipt(t, runID, l, dir, laneCommit(t, repo, l, "one.txt")))
	branch := prLane(t, repo.Root(), runID)
	driverGone(t, repo.Root(), runID, home)
	calls := forgeLists(t, map[int]listedPR{7: {Number: 7, HeadRefName: branch, State: "OPEN", Mergeable: MergeableConflicting, MergeStateStatus: "DIRTY"}}, "")

	listed, err := ListRuns(repo.Root(), "", Options{stopHome: home})
	if err != nil {
		t.Fatal(err)
	}
	if *calls != 0 || theRun(t, listed, runID).Lanes[0].Forge != nil {
		t.Fatalf("the bare listing read the forge: %d call(s), %+v", *calls, theRun(t, listed, runID).Lanes[0])
	}

	res, err := TakeRun(repo.Root(), runID, Options{stopHome: home})
	if err != nil {
		t.Fatal(err)
	}
	if *calls != 1 {
		t.Fatalf("the forge was listed %d time(s), not once", *calls)
	}
	if res.Run == nil || len(res.Run.Lanes) != 1 {
		t.Fatalf("the run as it stands: %+v", res.Run)
	}
	f := res.Run.Lanes[0].Forge
	if f == nil || f.State != ForgeConflicting || !strings.Contains(f.Note, "pull request #7 conflicts with the default branch") {
		t.Fatalf("the lane's pull request: %+v", f)
	}
	if !slices.ContainsFunc(res.Left, func(s string) bool { return strings.Contains(s, "lane-1's pull request #7 conflicts") }) {
		t.Fatalf("the take-over does not name the drifting pull request: %+v", res.Left)
	}
}

// TestATakeOverThatCannotReadTheForgeSaysSo: a forge that cannot be read takes
// nothing over less: the lane says why its pull request is unread, the take-over
// names it among what it left, and goes on.
func TestATakeOverThatCannotReadTheForgeSaysSo(t *testing.T) {
	repo, runID, l, dir := awaitingLane(t)
	home, _ := stopWorld(t, false)
	writeReceipt(t, dir, goodReceipt(t, runID, l, dir, laneCommit(t, repo, l, "one.txt")))
	prLane(t, repo.Root(), runID)
	driverGone(t, repo.Root(), runID, home)
	forgeLists(t, nil, "the forge client `gh` is not installed")

	res, err := TakeRun(repo.Root(), runID, Options{stopHome: home})
	if err != nil {
		t.Fatal(err)
	}
	f := res.Run.Lanes[0].Forge
	if f == nil || f.State != ForgeUnread || !strings.Contains(f.Note, "`gh` is not installed") {
		t.Fatalf("the lane's pull request: %+v", f)
	}
	if !slices.ContainsFunc(res.Left, func(s string) bool { return strings.Contains(s, "lane-1's pull request #7 could not be read") }) {
		t.Fatalf("the take-over does not name the unread pull request: %+v", res.Left)
	}
	if !strings.Contains(res.Next, "abcd implement step") {
		t.Fatalf("the move: %q", res.Next)
	}
}

// TestATakeOverWithNoPullRequestReadsNoForge: a run whose lanes name no pull
// request costs no forge read.
func TestATakeOverWithNoPullRequestReadsNoForge(t *testing.T) {
	repo, runID, l, dir := awaitingLane(t)
	home, _ := stopWorld(t, false)
	writeReceipt(t, dir, goodReceipt(t, runID, l, dir, laneCommit(t, repo, l, "one.txt")))
	driverGone(t, repo.Root(), runID, home)
	calls := forgeLists(t, nil, "")

	if _, err := TakeRun(repo.Root(), runID, Options{stopHome: home}); err != nil {
		t.Fatal(err)
	}
	if *calls != 0 {
		t.Fatalf("the forge was read %d time(s) for no pull request", *calls)
	}
}

// TestEachForgeStateIsNamed: what the forge says of a lane's pull request, by
// the listing's one reading, with a pull request the listing does not show, or
// shows on another branch, read as unread.
func TestEachForgeStateIsNamed(t *testing.T) {
	lane := Lane{ID: "lane-1", Branch: "build/b", PR: 7}
	for _, c := range []struct {
		pr   listedPR
		want ForgeState
		note string
	}{
		{listedPR{Number: 7, HeadRefName: "build/b", State: "OPEN", Mergeable: "MERGEABLE", MergeStateStatus: "CLEAN"}, ForgeOpen, "waiting on a person's merge"},
		{listedPR{Number: 7, HeadRefName: "build/b", State: "OPEN", Mergeable: "MERGEABLE", MergeStateStatus: "CLEAN", AutoMergeRequest: []byte(`{"mergeMethod":"MERGE"}`)}, ForgeArmed, "armed"},
		{listedPR{Number: 7, HeadRefName: "build/b", State: "MERGED"}, ForgeMerged, "merged"},
		{listedPR{Number: 7, HeadRefName: "build/b", State: "CLOSED"}, ForgeClosed, "closed without merging"},
		{listedPR{Number: 7, HeadRefName: "build/other", State: "OPEN"}, ForgeUnread, "not among"},
		{listedPR{Number: 8, HeadRefName: "build/b", State: "OPEN"}, ForgeUnread, "not among"},
	} {
		got := laneForge(lane, map[int]listedPR{c.pr.Number: c.pr}, "")
		if got == nil || got.State != c.want || !strings.Contains(got.Note, c.note) {
			t.Errorf("%+v: %+v, want %s naming %q", c.pr, got, c.want, c.note)
		}
	}
	if got := laneForge(Lane{ID: "lane-1"}, nil, ""); got != nil {
		t.Errorf("a lane with no pull request: %+v", got)
	}
}
