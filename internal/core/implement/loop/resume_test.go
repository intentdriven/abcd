package loop

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/core/machineload"
	"github.com/intentdriven/abcd/internal/core/occupancy"
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
	oldProcs, oldFrom := readProcs, hereFrom
	readProcs = func() (sessionname.Table, error) { return tab, nil }
	hereFrom = func() int { return here + shellOf }
	t.Cleanup(func() { readProcs, hereFrom = oldProcs, oldFrom })
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
