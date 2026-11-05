package loop

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"context"
	"github.com/intentdriven/abcd/internal/adapter/repoprompt/mcptest"
	"github.com/intentdriven/abcd/internal/core/machineload"
	"github.com/intentdriven/abcd/internal/core/runstop"
	"github.com/intentdriven/abcd/internal/core/sessionname"
	"github.com/intentdriven/abcd/internal/gittest"
)

// stopFileIn is where the person's stop lives in folder.
func stopFileIn(folder string) string {
	return filepath.Join(folder, runstop.StopDirName, runstop.StopFileName)
}

// checkOnce is the check a move makes, under the default stages.
func checkOnce(repoRoot, runID string, o Options) error {
	_, err := stopCheck(repoRoot, runID, DefaultStages(), o)
	return err
}

// haltOnce applies a stop that did not come from the file.
func haltOnce(repoRoot, runID string, s runstop.Stop, o Options) (*RunStop, error) {
	return haltRun(repoRoot, runID, s, DefaultStages(), o)
}

// stopWorld is a fixture home with a shared folder chosen and the person's stop
// set in it.
func stopWorld(t *testing.T, setting bool) (home, folder string) {
	t.Helper()
	base := t.TempDir()
	home = filepath.Join(base, "home")
	folder = filepath.Join(base, "shared")
	if err := os.Mkdir(home, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(folder, 0o755); err != nil {
		t.Fatal(err)
	}
	if setting {
		if _, err := runstop.WriteSetting(home, folder, ""); err != nil {
			t.Fatal(err)
		}
	}
	return home, folder
}

// stopped sets the person's stop in a fixture home and returns the Options that
// read it.
func stopped(t *testing.T) (Options, string, string) {
	t.Helper()
	home, folder := stopWorld(t, true)
	if _, err := runstop.Set(home, time.Now()); err != nil {
		t.Fatal(err)
	}
	return Options{stopHome: home}, home, folder
}

func wantStopRefusal(t *testing.T, err error) *Refusal {
	t.Helper()
	r, ok := AsRefusal(err)
	if !ok || r.Stage != StageStop || !r.Contention {
		t.Fatalf("want a contention refusal at the stage stop, got %v", err)
	}
	return r
}

func controlOf(t *testing.T, repoRoot, runID string) control {
	t.Helper()
	root, err := os.OpenRoot(repoRoot)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	c, err := readControl(root, runID)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// TestNoStopMovesAsBefore: with nothing set, the check says nothing and writes
// nothing.
func TestNoStopMovesAsBefore(t *testing.T) {
	repo, runID, _, _ := awaitingLane(t)
	home, _ := stopWorld(t, true)
	if err := checkOnce(repo.Root(), runID, Options{stopHome: home}); err != nil {
		t.Fatal(err)
	}
	if c := controlOf(t, repo.Root(), runID); c.Stop != nil {
		t.Fatalf("no stop was set, and the run records one: %+v", c.Stop)
	}
}

// TestAStopHoldsAnAgentInItsCall is the ruling's order: a step under a stop moves
// nothing and records who set it and when, and while the lane's agent may still
// be in its call its uncommitted work stays exactly where it is and the run is
// not halted.
func TestAStopHoldsAnAgentInItsCall(t *testing.T) {
	repo, runID, l, dir, _ := deadLane(t)
	o, _, _ := stopped(t)
	before, err := ReadState(repo.Root(), runID)
	if err != nil {
		t.Fatal(err)
	}

	_, err = advance(repo.Root(), runID, DefaultStages(), o)
	r := wantStopRefusal(t, err)
	if !strings.Contains(r.Reason, "still in its call") {
		t.Fatalf("the refusal says %q", r.Reason)
	}

	editsIntact(t, repo, l)
	noAside(t, dir)
	st, err := ReadState(repo.Root(), runID)
	if err != nil {
		t.Fatal(err)
	}
	if len(st.Lanes) != len(before.Lanes) || st.Lanes[0].Stage != before.Lanes[0].Stage {
		t.Fatalf("a stop moved the lane: %+v", st.Lanes[0])
	}
	if a := st.Lanes[0].Awaits[0]; a.Kept != "" {
		t.Fatalf("an agent in its call was marked lost: %+v", a)
	}
	c := controlOf(t, repo.Root(), runID)
	if c.Stop == nil || c.Stop.HaltedAt != nil || c.Stop.SetBy == "" || c.Stop.SetAt.IsZero() {
		t.Fatalf("the run's stop: %+v", c.Stop)
	}
	var named bool
	for _, e := range st.Record {
		if e.Stage == StageStop && strings.Contains(e.Note, c.Stop.SetBy) && strings.Contains(e.Note, c.Stop.SetAt.UTC().Format(time.RFC3339)) {
			named = true
		}
	}
	if !named {
		t.Fatalf("the run record names no one for the stop: %+v", st.Record)
	}
}

// TestALostAgentsWorkIsSavedAsideThenTheRunHalts: once no agent is live, the
// lane's uncommitted work is saved aside as a restart saves it, the worktree is
// clean at its last commit, the agent is marked lost, and the run is halted with
// what it left named.
func TestALostAgentsWorkIsSavedAsideThenTheRunHalts(t *testing.T) {
	repo, runID, l, dir, head := deadLane(t)
	o, _, _ := stopped(t)
	later := time.Now().Add(3 * time.Hour)
	o.Now = func() time.Time { return later }

	wantStopRefusal(t, checkOnce(repo.Root(), runID, o))

	at, meta := asideOf(t, dir)
	if meta["head"] != head || !strings.Contains(meta["why"].(string), "stopped") {
		t.Fatalf("aside.json: %v", meta)
	}
	if status := repo.Git("-C", l.Worktree, "status", "--porcelain"); strings.TrimSpace(status) != "" {
		t.Fatalf("the worktree is clean after the save:\n%s", status)
	}
	if _, err := os.Lstat(filepath.Join(at, AsidePatchName)); err != nil {
		t.Fatalf("the patch is saved: %v", err)
	}
	c := controlOf(t, repo.Root(), runID)
	if c.Stop == nil || c.Stop.HaltedAt == nil || len(c.Stop.Left) != 1 || c.Stop.Left[0].Aside == "" || c.Stop.Left[0].Files == 0 {
		t.Fatalf("the stop's account: %+v", c.Stop)
	}
	st, err := ReadState(repo.Root(), runID)
	if err != nil {
		t.Fatal(err)
	}
	if st.Lanes[0].Awaits[0].Kept == "" {
		t.Fatalf("a lost agent's await is kept for a restart: %+v", st.Lanes[0].Awaits[0])
	}
	// A second check changes nothing, and saves nothing again.
	wantStopRefusal(t, checkOnce(repo.Root(), runID, o))
	if stamps, _ := os.ReadDir(filepath.Join(dir, "aside")); len(stamps) != 1 {
		t.Fatalf("a second check saved again: %v", stamps)
	}
}

// TestAStopCollectsAnArrivedReceipt: a finished agent's receipt is collected and
// the lane goes no further: no fresh agent is told, and the run then halts.
func TestAStopCollectsAnArrivedReceipt(t *testing.T) {
	repo, runID, l, dir := awaitingLane(t)
	c1 := laneCommit(t, repo, l, "one.txt")
	writeReceipt(t, dir, goodReceipt(t, runID, l, dir, c1))
	o, _, _ := stopped(t)

	_, err := advance(repo.Root(), runID, DefaultStages(), o)
	wantStopRefusal(t, err)

	st, err := ReadState(repo.Root(), runID)
	if err != nil {
		t.Fatal(err)
	}
	if st.Lanes[0].Stage != StageValidate || len(st.Lanes[0].Awaits) != 0 {
		t.Fatalf("the receipt is collected and nothing further handed out: %+v", st.Lanes[0])
	}
	c := controlOf(t, repo.Root(), runID)
	if c.Stop == nil || c.Stop.HaltedAt == nil || !strings.Contains(c.Stop.Left[0].Note, "collected") {
		t.Fatalf("the stop's account: %+v", c.Stop)
	}
}

// TestReceiptUnderAStopGoesNoFurther: `implement receipt` is not refused, and its
// answer tells the caller no one else starts.
func TestReceiptUnderAStopGoesNoFurther(t *testing.T) {
	repo, runID, l, dir := awaitingLane(t)
	c1 := laneCommit(t, repo, l, "one.txt")
	path := writeReceipt(t, dir, goodReceipt(t, runID, l, dir, c1))
	o, _, _ := stopped(t)
	res, err := Receipt(repo.Root(), runID, path, DefaultStages(), o)
	if err != nil {
		t.Fatal(err)
	}
	if res.Awaiting != nil || !strings.Contains(res.Next, "lift") {
		t.Fatalf("the receipt's answer: %+v", res)
	}
}

// TestAStopReleasesTheReservationOnlyAtTheEnd: the reservation stands while an
// agent may be in its call, and is released on origin once the lane has settled,
// so a peer may take the work.
func TestAStopReleasesTheReservationOnlyAtTheEnd(t *testing.T) {
	repo, runID, _, _, _ := deadLane(t)
	bare := onOriginAccepted(t, repo, runID)
	now := time.Now().UTC().Truncate(time.Second)
	reserveRun(t, repo, runID, "itd-10", now, "commands/x.md")
	o, _, _ := stopped(t)
	o.Now = func() time.Time { return now }

	wantStopRefusal(t, checkOnce(repo.Root(), runID, o))
	if got := originReservation(t, bare, reservedRef); got.released {
		t.Fatal("the reservation was released with an agent still in its call")
	}

	later := now.Add(3 * time.Hour)
	o.Now = func() time.Time { return later }
	wantStopRefusal(t, checkOnce(repo.Root(), runID, o))
	if got := originReservation(t, bare, reservedRef); !got.released {
		t.Fatalf("the reservation still stands on origin: %+v", got)
	}
	st, err := ReadState(repo.Root(), runID)
	if err != nil {
		t.Fatal(err)
	}
	if st.ReservedAt != nil {
		t.Fatalf("the run still says it holds a reservation: %v", st.ReservedAt)
	}
}

// TestAHaltedRunWaitsForATakeOverWhenTheStopLifts: a run that let its claims go
// does not step on by itself when the stop lifts.
func TestAHaltedRunWaitsForATakeOverWhenTheStopLifts(t *testing.T) {
	repo, runID, _, _, _ := deadLane(t)
	o, home, _ := stopped(t)
	later := time.Now().Add(3 * time.Hour)
	o.Now = func() time.Time { return later }
	wantStopRefusal(t, checkOnce(repo.Root(), runID, o))
	if _, err := runstop.Lift(home, later); err != nil {
		t.Fatal(err)
	}
	_, err := advance(repo.Root(), runID, DefaultStages(), o)
	r := wantStopRefusal(t, err)
	if !strings.Contains(r.Remedy, "implement resume --take") {
		t.Fatalf("the remedy names %q", r.Remedy)
	}
}

// TestALiftBeforeTheHaltJustGoesOn: lifted while an agent was in its call, the
// run never let its claims go, so it carries on with no take-over.
func TestALiftBeforeTheHaltJustGoesOn(t *testing.T) {
	repo, runID, _, _, _ := deadLane(t)
	o, home, _ := stopped(t)
	wantStopRefusal(t, checkOnce(repo.Root(), runID, o))
	if _, err := runstop.Lift(home, time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := checkOnce(repo.Root(), runID, o); err != nil {
		t.Fatal(err)
	}
	if c := controlOf(t, repo.Root(), runID); c.Stop != nil {
		t.Fatalf("the mark stands: %+v", c.Stop)
	}
}

// TestAStopStartsNoRun: no run starts under a stop, a resumed one included.
func TestAStopStartsNoRun(t *testing.T) {
	repo, _, _, _ := awaitingLane(t)
	o, _, _ := stopped(t)
	_, err := Start(repo.Root(), "itd-10", o)
	wantStopRefusal(t, err)
}

// TestARestartIsRefusedUnderAStop: a restart tells a fresh agent to start, so a
// stop refuses it before anything is saved.
func TestARestartIsRefusedUnderAStop(t *testing.T) {
	repo, runID, l, dir, _ := deadLane(t)
	o, _, _ := stopped(t)
	_, err := restartLane(repo.Root(), runID, "lane-1", "", o)
	wantStopRefusal(t, err)
	noAside(t, dir)
	editsIntact(t, repo, l)
}

// TestARefusedFileIsNoStop: a file another account wrote stops nothing, and the
// step's answer names it.
func TestARefusedFileIsNoStop(t *testing.T) {
	repo, runID, _, _ := awaitingLane(t)
	home, folder := stopWorld(t, true)
	p := stopFileIn(folder)
	if err := os.Symlink("/nonexistent", p); err != nil {
		t.Fatal(err)
	}
	res, err := advance(repo.Root(), runID, DefaultStages(), Options{stopHome: home})
	if err != nil {
		var r *Refusal
		if errors.As(err, &r) && r.Stage == StageStop {
			t.Fatalf("a refused file stopped the run: %v", err)
		}
		t.Fatal(err)
	}
	if !strings.Contains(res.Next, runstop.StopFileName) {
		t.Fatalf("the answer does not name the refused file: %q", res.Next)
	}
}

// TestHaltRunIsTheSameActForASignal: a stop that did not come from the file takes
// the same order, naming who the caller says set it.
func TestHaltRunIsTheSameActForASignal(t *testing.T) {
	repo, runID, _, dir, _ := deadLane(t)
	later := time.Now().Add(3 * time.Hour)
	rec, err := haltOnce(repo.Root(), runID, runstop.Stop{SetBy: "a signal", SetAt: later}, Options{Now: func() time.Time { return later }})
	if err != nil {
		t.Fatal(err)
	}
	if rec == nil || rec.HaltedAt == nil || rec.SetBy != "a signal" {
		t.Fatalf("HaltRun = %+v", rec)
	}
	asideOf(t, dir)
}

// TestTheStopComposesTheDaysPage: the page names the stop and each run with what
// it left, is rewritten whole, and holds no account's name.
func TestTheStopComposesTheDaysPage(t *testing.T) {
	repo, runID, _, _, _ := deadLane(t)
	o, home, folder := stopped(t)
	later := time.Now().Add(3 * time.Hour)
	o.Now = func() time.Time { return later }
	wantStopRefusal(t, checkOnce(repo.Root(), runID, o))

	page := filepath.Join(folder, runstop.DigestDir, later.UTC().Format(time.DateOnly)+".md")
	got, err := os.ReadFile(page)
	if err != nil {
		t.Fatal(err)
	}
	s, _, err := runstop.ReadSetting(home)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"## Stop", runID, "itd-10", "lane-1", "saved aside"} {
		if !strings.Contains(string(got), want) {
			t.Errorf("the page lacks %q:\n%s", want, got)
		}
	}
	if strings.Contains(string(got), s.Person) {
		t.Errorf("the page names the account %q:\n%s", s.Person, got)
	}
}

// TestADrainUnderAStopOpensNothing: a drain's move under a stop refuses, reads no
// plan and lets a judgement it awaited go with its hold.
func TestADrainUnderAStopOpensNothing(t *testing.T) {
	repo, bare := reservingDrainRepo(t, "commands/drain.md")
	_, taken := askFirstOnce(t, repo, bare, DrainOptions{})
	o, _, _ := stopped(t)
	_, err := Drain(repo.Root(), o, DrainOptions{})
	wantStopRefusal(t, err)
	st, _, rerr := readDrain(repo.Root())
	if rerr != nil {
		t.Fatal(rerr)
	}
	if st.Judging != nil {
		t.Fatalf("the judgement still awaited: %+v", st.Judging)
	}
	releasedOver(t, bare, secondIssue, taken)
}

// TestAFailedSaveHoldsTheClaims: uncommitted work that cannot be proved stays
// where it is, the lane does not settle, and the run is not halted, so nothing is
// released over work saved nowhere.
func TestAFailedSaveHoldsTheClaims(t *testing.T) {
	repo, runID, l, dir, _ := deadLane(t)
	old := checkAsidePatch
	checkAsidePatch = func(string, string) error { return errors.New("will not apply") }
	t.Cleanup(func() { checkAsidePatch = old })
	o, _, _ := stopped(t)
	later := time.Now().Add(3 * time.Hour)
	o.Now = func() time.Time { return later }
	wantStopRefusal(t, checkOnce(repo.Root(), runID, o))
	c := controlOf(t, repo.Root(), runID)
	if c.Stop == nil || c.Stop.HaltedAt != nil || c.Stop.Left[0].Settled {
		t.Fatalf("the stop's account: %+v", c.Stop)
	}
	noAside(t, dir)
	editsIntact(t, repo, l)
}

// TestAgentStateErrsTowardsAlive: a live agent's worktree is never reset on a
// guess. A step killed outright can leave its agent running until the claim's term
// ends, and a host's agent is taken to be in its call while its driver lives.
func TestAgentStateErrsTowardsAlive(t *testing.T) {
	now := time.Now()
	tab := &sessionname.Table{ReadAt: now}
	dead := Launch{Session: "s", PID: 9999, StartUnix: now.Unix()}
	running := Await{Running: &RunnerClaim{Session: "s", Until: now.Add(time.Hour)}}
	if got := agentState(running, control{Launches: []Launch{dead}}, tab, now); got != AgentUnknown {
		t.Errorf("a step gone inside its claim's term: %s", got)
	}
	expired := Await{Running: &RunnerClaim{Session: "s", Until: now.Add(-time.Second)}}
	if got := agentState(expired, control{Launches: []Launch{dead}}, tab, now); got != AgentLost {
		t.Errorf("a step gone past its claim's term: %s", got)
	}
	host := machineload.Proc{PID: 4100, Age: time.Hour, Name: "host"}
	alive := &sessionname.Table{ReadAt: now, Procs: []machineload.Proc{host}}
	old := Await{Since: now.Add(-48 * time.Hour)}
	d := &LiveDriver{PID: 4100, StartUnix: now.Add(-time.Hour).Unix()}
	if got := agentState(old, control{Driver: d}, alive, now); got != AgentUnknown {
		t.Errorf("a host's agent under a live driver, past the lease: %s", got)
	}
	if got := agentState(old, control{Driver: d}, tab, now); got != AgentLost {
		t.Errorf("a host's agent whose driver is gone: %s", got)
	}
	if got := agentState(old, control{Driver: d}, nil, now); got != AgentUnknown {
		t.Errorf("a host's agent whose driver cannot be judged: %s", got)
	}
}

// TestAGoneWorktreeHoldsNothingBack: a lane whose worktree no longer stands has
// nothing to save, so it does not hold the run's claims for ever.
func TestAGoneWorktreeHoldsNothingBack(t *testing.T) {
	repo, runID, l, _, _ := deadLane(t)
	o, _, _ := stopped(t)
	later := time.Now().Add(3 * time.Hour)
	o.Now = func() time.Time { return later }
	repo.Git("worktree", "remove", "--force", l.Worktree)

	wantStopRefusal(t, checkOnce(repo.Root(), runID, o))
	c := controlOf(t, repo.Root(), runID)
	if c.Stop == nil || c.Stop.HaltedAt == nil || !c.Stop.Left[0].Settled {
		t.Fatalf("the stop's account: %+v", c.Stop)
	}
	if !strings.Contains(c.Stop.Left[0].Note, "worktree is gone") {
		t.Fatalf("the lane's note: %q", c.Stop.Left[0].Note)
	}
}

// TestAFailedReleaseLapses: a release that cannot reach origin says the claim
// lapses with its lease, so nothing may renew it after the run has halted.
func TestAFailedReleaseLapses(t *testing.T) {
	repo, runID, _, _, _ := deadLane(t)
	onOriginAccepted(t, repo, runID)
	now := time.Now().UTC().Truncate(time.Second)
	reserveRun(t, repo, runID, "itd-10", now, "commands/x.md")
	o, _, _ := stopped(t)
	later := now.Add(3 * time.Hour)
	o.Now = func() time.Time { return later }
	repo.Git("remote", "set-url", "origin", filepath.Join(t.TempDir(), "gone.git"))
	if _, err := AcceptConfigChange(repo.Root(), runID, Options{}); err != nil {
		t.Fatal(err)
	}

	wantStopRefusal(t, checkOnce(repo.Root(), runID, o))
	c := controlOf(t, repo.Root(), runID)
	if c.Stop == nil || c.Stop.HaltedAt == nil || !strings.Contains(c.Stop.Released, "lapses") {
		t.Fatalf("the stop's account: %+v", c.Stop)
	}
	st, err := ReadState(repo.Root(), runID)
	if err != nil {
		t.Fatal(err)
	}
	if st.ReservedAt != nil {
		t.Fatalf("the halted run would still renew its claim: %v", st.ReservedAt)
	}
}

// TestALiftBeforeTheHaltReTellsWhatItSettled: one lane settled while another's
// agent was in its call, then the stop lifted. The run goes on, and the settled
// lane's agent, marked lost, is told again rather than left reading as lost
// behind the fresh agent the next step names.
func TestALiftBeforeTheHaltReTellsWhatItSettled(t *testing.T) {
	repo, runID, _, dir, _ := deadLane(t)
	o, home, _ := stopped(t)
	later := time.Now().Add(3 * time.Hour)
	o.Now = func() time.Time { return later }
	if err := mutate(repo.Root(), runID, func(_ *os.Root, st *State) (bool, error) {
		st.Lanes = append(st.Lanes, Lane{ID: "lane-2", Stage: StageImplement, Awaits: []Await{
			{Role: RoleImplementer, Receipt: runRel(runID) + "/lane-2/implement/receipt.json", Since: later},
		}})
		return true, nil
	}); err != nil {
		t.Fatal(err)
	}

	wantStopRefusal(t, checkOnce(repo.Root(), runID, o))
	if _, at := asideOf(t, dir); at == nil {
		t.Fatal("the settled lane saved nothing aside")
	}
	if c := controlOf(t, repo.Root(), runID); c.Stop == nil || c.Stop.HaltedAt != nil {
		t.Fatalf("the run should still wait on lane-2: %+v", c.Stop)
	}
	if _, err := runstop.Lift(home, later); err != nil {
		t.Fatal(err)
	}
	if err := checkOnce(repo.Root(), runID, o); err != nil {
		t.Fatalf("after the lift: %v", err)
	}
	st, err := ReadState(repo.Root(), runID)
	if err != nil {
		t.Fatal(err)
	}
	if a := st.Lanes[0].Awaits[0]; a.Kept != "" || a.Since.Sub(later).Abs() > time.Second {
		t.Fatalf("the settled lane's await is not told again: %+v", a)
	}
}

// TestAStopTakesNoGitStepOverAChangedConfiguration: a stop saves a lane as a restart
// does, tripwire and all: a changed configuration holds the lane and the claims,
// and says so.
func TestAStopTakesNoGitStepOverAChangedConfiguration(t *testing.T) {
	repo, runID, l, dir, _ := deadLane(t)
	o, _, _ := stopped(t)
	later := time.Now().Add(3 * time.Hour)
	o.Now = func() time.Time { return later }
	repo.Git("config", "core.fsmonitor", "/tmp/planted")

	wantStopRefusal(t, checkOnce(repo.Root(), runID, o))
	noAside(t, dir)
	editsIntact(t, repo, l)
	c := controlOf(t, repo.Root(), runID)
	if c.Stop == nil || c.Stop.HaltedAt != nil || !strings.Contains(c.Stop.Left[0].Note, "configuration") {
		t.Fatalf("the stop's account: %+v", c.Stop)
	}
}

func settingOf(home string) string { return filepath.Join(home, ".abcd.noindex", runstop.SettingFile) }

// TestAPinnedRunTakesTheStopWithTheSettingGone: the folder a run started under is
// read whatever has become of the setting.
func TestAPinnedRunTakesTheStopWithTheSettingGone(t *testing.T) {
	repo, runID, _, _, _ := deadLane(t)
	home, _ := stopWorld(t, true)
	o := Options{stopHome: home}
	if err := checkOnce(repo.Root(), runID, o); err != nil {
		t.Fatal(err)
	}
	if c := controlOf(t, repo.Root(), runID); c.Shared == nil || c.Shared.Folder == "" {
		t.Fatalf("the run pinned nothing: %+v", c)
	}
	if _, err := runstop.Set(home, time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(settingOf(home)); err != nil {
		t.Fatal(err)
	}
	wantStopRefusal(t, checkOnce(repo.Root(), runID, o))
	if c := controlOf(t, repo.Root(), runID); c.Stop == nil {
		t.Fatalf("the run did not take the stop: %+v", c)
	}
}

// TestAGoneSettingHoldsTheRun: with no stop to read, a setting gone during the run
// refuses the move loudly, moves nothing and lets no claim go; put back, the run
// goes on.
func TestAGoneSettingHoldsTheRun(t *testing.T) {
	repo, runID, l, dir, _ := deadLane(t)
	home, _ := stopWorld(t, true)
	o := Options{stopHome: home}
	if err := checkOnce(repo.Root(), runID, o); err != nil {
		t.Fatal(err)
	}
	keep, err := os.ReadFile(settingOf(home))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(settingOf(home)); err != nil {
		t.Fatal(err)
	}
	r := wantStopRefusal(t, checkOnce(repo.Root(), runID, o))
	if !strings.Contains(r.Reason, "setting") {
		t.Fatalf("the refusal says %q", r.Reason)
	}
	if c := controlOf(t, repo.Root(), runID); c.Stop != nil {
		t.Fatalf("a held run took a stop no one set: %+v", c.Stop)
	}
	noAside(t, dir)
	editsIntact(t, repo, l)
	if _, err := TakeRun(repo.Root(), runID, o); err == nil {
		t.Fatal("a take-over passed a run that is held")
	}
	if err := os.WriteFile(settingOf(home), keep, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := checkOnce(repo.Root(), runID, o); err != nil {
		t.Fatalf("with the setting back: %v", err)
	}
}

// TestAnUnreadableControlRecordRefusesTheMove: the record that pins the run does
// not read as no pin.
func TestAnUnreadableControlRecordRefusesTheMove(t *testing.T) {
	repo, runID, _, _, _ := deadLane(t)
	home, _ := stopWorld(t, true)
	p := filepath.Join(repo.Root(), filepath.FromSlash(controlRel(runID)))
	if err := os.WriteFile(p, []byte("not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := checkOnce(repo.Root(), runID, Options{stopHome: home}); err == nil {
		t.Fatal("a move passed a control record that does not parse")
	}
}

// TestARefusedReadClearsNoMark: only a read that saw clearly and found no stop is
// a lift: a file refused where the stop stood clears nothing.
func TestARefusedReadClearsNoMark(t *testing.T) {
	repo, runID, _, _, _ := deadLane(t)
	o, _, folder := stopped(t)
	wantStopRefusal(t, checkOnce(repo.Root(), runID, o))
	p := stopFileIn(folder)
	if err := os.Remove(p); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("/nonexistent", p); err != nil {
		t.Fatal(err)
	}
	wantStopRefusal(t, checkOnce(repo.Root(), runID, o))
	if c := controlOf(t, repo.Root(), runID); c.Stop == nil {
		t.Fatal("a refused read cleared the run's mark")
	}
}

// TestASignalsStopIsNoFileToLift: a stop that came from a signal has no file to
// read, so a read that finds none does not lift it: it settles, and it stays.
func TestASignalsStopIsNoFileToLift(t *testing.T) {
	repo, runID, _, _, _ := deadLane(t)
	home, _ := stopWorld(t, false)
	o := Options{stopHome: home}
	if _, err := haltOnce(repo.Root(), runID, runstop.Stop{SetBy: "a signal", SetAt: time.Now()}, o); err != nil {
		t.Fatal(err)
	}
	wantStopRefusal(t, checkOnce(repo.Root(), runID, o))
	if c := controlOf(t, repo.Root(), runID); c.Stop == nil || c.Stop.SetBy != "a signal" {
		t.Fatalf("the signal's mark: %+v", c.Stop)
	}
}

// TestARunPinsTheFolderAtItsStart: the location is fixed for the run's life from
// the moment it begins.
func TestARunPinsTheFolderAtItsStart(t *testing.T) {
	repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
	home, folder := stopWorld(t, true)
	res, err := Start(repo.Root(), "itd-10", Options{stopHome: home})
	if err != nil {
		t.Fatal(err)
	}
	if c := controlOf(t, repo.Root(), res.RunID); c.Shared == nil || c.Shared.Folder != folder {
		t.Fatalf("the pin at the start: %+v", c.Shared)
	}
}

// TestADrainIsHeldWhenARunsSettingHasGone: a drain reads its runs' pins.
func TestADrainIsHeldWhenARunsSettingHasGone(t *testing.T) {
	repo, runID, _, _, _ := deadLane(t)
	home, _ := stopWorld(t, true)
	o := Options{stopHome: home}
	if err := checkOnce(repo.Root(), runID, o); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(settingOf(home)); err != nil {
		t.Fatal(err)
	}
	wantStopRefusal(t, drainStop(repo.Root(), &DrainState{}, o))
}

// TestTheStopHomeIsTheAccounts: a step under another HOME reads this account's
// stop, not one from wherever the variable points.
func TestTheStopHomeIsTheAccounts(t *testing.T) {
	want, err := runstop.AccountHome()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", filepath.Join(t.TempDir(), "elsewhere"))
	if got := accountStopHome(); got != want {
		t.Fatalf("accountStopHome = %q, want %q", got, want)
	}
}

// stopHomeIs makes home the one every call with no home of its own reads.
func stopHomeIs(t *testing.T, home string) {
	t.Helper()
	oldHome, oldOff := defaultStopHome, stopReadingOff
	defaultStopHome, stopReadingOff = func() string { return home }, false
	t.Cleanup(func() { defaultStopHome, stopReadingOff = oldHome, oldOff })
}

// TestALaterSettleLeavesAHaltAsItStands: two checks that race settle once. What the
// first let go stays said, so a take-over reserves again.
func TestALaterSettleLeavesAHaltAsItStands(t *testing.T) {
	repo, runID, _, _, _ := deadLane(t)
	onOriginAccepted(t, repo, runID)
	now := time.Now().UTC().Truncate(time.Second)
	reserveRun(t, repo, runID, "itd-10", now, "commands/x.md")
	o, _, _ := stopped(t)
	later := now.Add(3 * time.Hour)
	o.Now = func() time.Time { return later }
	real := readProcs
	t.Cleanup(func() { readProcs = real })
	raced := false
	readProcs = func() (sessionname.Table, error) {
		if !raced {
			raced = true
			wantStopRefusal(t, checkOnce(repo.Root(), runID, o))
		}
		return real()
	}
	wantStopRefusal(t, checkOnce(repo.Root(), runID, o))
	c := controlOf(t, repo.Root(), runID)
	if c.Stop == nil || c.Stop.HaltedAt == nil || c.Stop.Released == noReservationHeld {
		t.Fatalf("the later settle rewrote the halt: %+v", c.Stop)
	}
}

// TestAStopKeepsTheClaimOfAnAgentInItsCall: a lane an agent may still be working in
// is not left for a peer to take.
func TestAStopKeepsTheClaimOfAnAgentInItsCall(t *testing.T) {
	repo, runID, _, _, _ := deadLane(t)
	bare := onOriginAccepted(t, repo, runID)
	now := time.Now().UTC().Truncate(time.Second)
	reserveRun(t, repo, runID, "itd-10", now, "commands/x.md")
	before := originReservation(t, bare, reservedRef).commit
	later := now.Add(90 * time.Minute)
	if err := mutate(repo.Root(), runID, func(_ *os.Root, st *State) (bool, error) {
		st.Lanes[0].Awaits[0].Since = later
		return true, nil
	}); err != nil {
		t.Fatal(err)
	}
	home, _ := stopWorld(t, true)
	if _, err := runstop.Set(home, now); err != nil {
		t.Fatal(err)
	}
	stopHomeIs(t, home)
	renewRunsAt(repo.Root(), runID, nil, later)
	if got := originReservation(t, bare, reservedRef).commit; got == before {
		t.Fatal("the claim of an agent in its call was not renewed")
	}
}

// TestATakeOverNeedsAClearRead: a read that refused anything did not see clearly,
// so the take-over neither passes nor clears the run's mark.
func TestATakeOverNeedsAClearRead(t *testing.T) {
	repo, runID, _, _, _ := deadLane(t)
	o, _, folder := stopped(t)
	later := time.Now().Add(3 * time.Hour)
	o.Now = func() time.Time { return later }
	wantStopRefusal(t, checkOnce(repo.Root(), runID, o))
	if err := os.Chmod(filepath.Dir(stopFileIn(folder)), 0o775); err != nil {
		t.Fatal(err)
	}
	_, err := TakeRun(repo.Root(), runID, o)
	if r, ok := AsRefusal(err); !ok || !r.Contention {
		t.Fatalf("a take-over on a read that refused: %v", err)
	}
	if c := controlOf(t, repo.Root(), runID); c.Stop == nil {
		t.Fatal("the take-over cleared the run's mark")
	}
}

// TestAnUnknownHomeIsNoLift: a home that cannot be found is not read as no stop: the
// run's mark stands, and nothing moves.
func TestAnUnknownHomeIsNoLift(t *testing.T) {
	repo, runID, _, _, _ := deadLane(t)
	o, _, _ := stopped(t)
	wantStopRefusal(t, checkOnce(repo.Root(), runID, o))
	stopHomeIs(t, "")
	wantStopRefusal(t, checkOnce(repo.Root(), runID, Options{}))
	if c := controlOf(t, repo.Root(), runID); c.Stop == nil {
		t.Fatal("an unknown home cleared the run's mark")
	}
}

// gonePastItsLease leaves the lane's agent long gone, with a reservation due.
func gonePastItsLease(t *testing.T, repo *gittest.Repo, runID string, now time.Time) {
	t.Helper()
	if err := mutate(repo.Root(), runID, func(_ *os.Root, st *State) (bool, error) {
		st.Lanes[0].Awaits[0].Since = now.Add(-3 * time.Hour)
		return true, nil
	}); err != nil {
		t.Fatal(err)
	}
}

// TestAStatusSettlesARunNoAgentHolds: with no one stepping a run under a stop, the
// renewal that falls due settles it instead, so its work is saved before its claim
// goes.
func TestAStatusSettlesARunNoAgentHolds(t *testing.T) {
	repo, runID, _, dir, _ := deadLane(t)
	bare := onOriginAccepted(t, repo, runID)
	now := time.Now().UTC().Truncate(time.Second)
	reserveRun(t, repo, runID, "itd-10", now, "commands/x.md")
	gonePastItsLease(t, repo, runID, now)
	home, _ := stopWorld(t, true)
	if _, err := runstop.Set(home, now); err != nil {
		t.Fatal(err)
	}
	stopHomeIs(t, home)
	for _, r := range renewRunsAt(repo.Root(), runID, nil, now.Add(90*time.Minute)) {
		if r.Renewed {
			t.Fatalf("renewed under a stop: %+v", r)
		}
	}
	if c := controlOf(t, repo.Root(), runID); c.Stop == nil || c.Stop.HaltedAt == nil {
		t.Fatalf("the status settled nothing: %+v", c.Stop)
	}
	asideOf(t, dir)
	if got := originReservation(t, bare, reservedRef); !got.released {
		t.Fatalf("origin's reservation: %+v", got)
	}
}

// TestAFailedSaveKeepsItsLease: work saved nowhere keeps the claim that guards it, and
// a move's refusal says so.
func TestAFailedSaveKeepsItsLease(t *testing.T) {
	repo, runID, l, dir, _ := deadLane(t)
	bare := onOriginAccepted(t, repo, runID)
	now := time.Now().UTC().Truncate(time.Second)
	reserveRun(t, repo, runID, "itd-10", now, "commands/x.md")
	before := originReservation(t, bare, reservedRef).commit
	gonePastItsLease(t, repo, runID, now)
	old := checkAsidePatch
	checkAsidePatch = func(string, string) error { return errors.New("will not apply") }
	t.Cleanup(func() { checkAsidePatch = old })
	home, _ := stopWorld(t, true)
	if _, err := runstop.Set(home, now); err != nil {
		t.Fatal(err)
	}
	stopHomeIs(t, home)
	renewRunsAt(repo.Root(), runID, nil, now.Add(90*time.Minute))
	got := originReservation(t, bare, reservedRef)
	if got.commit == before || got.released {
		t.Fatalf("the claim over unsaved work was not kept: %+v", got)
	}
	noAside(t, dir)
	editsIntact(t, repo, l)
	r := wantStopRefusal(t, checkOnce(repo.Root(), runID, Options{}))
	if !strings.Contains(r.Reason, "could not be saved") {
		t.Fatalf("the refusal says %q", r.Reason)
	}
}

// TestAReviewRouteTakesAStopSetMidStage: a stop set since the step began sends the
// route nothing.
func TestAReviewRouteTakesAStopSetMidStage(t *testing.T) {
	repo, id, res, stages, srv, rr := handedReview(t)
	o, _, _ := stopped(t)
	_, err := DriveReview(context.Background(), repo.Root(), id, res, stages, o, rr)
	wantStopRefusal(t, err)
	if n := len(srv.Calls()); n != 0 {
		t.Fatalf("the route was sent %d request(s) under a stop", n)
	}
}

// handedReview is a run whose validate stage has handed the ruthless reviewer to the
// host, with a route it could go to.
func handedReview(t *testing.T) (*gittest.Repo, string, StepResult, Stages, *mcptest.Server, *ReviewRoute) {
	t.Helper()
	repo := briefRepo(t, agentsMarked)
	oneReviewerPerRole(t)
	start, err := Start(repo.Root(), "itd-10", Options{})
	if err != nil {
		t.Fatal(err)
	}
	stages := DefaultStages()
	implemented(t, repo, start.RunID, stages, "one.txt")
	res, err := advance(repo.Root(), start.RunID, stages, Options{})
	if err != nil || res.Awaiting == nil {
		t.Fatalf("the validate stage hands out a reviewer: %+v %v", res, err)
	}
	srv := rpServer(func(string) string { return "" })
	return repo, start.RunID, res, stages, srv, rpRoute(t, srv)
}
