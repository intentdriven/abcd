package loop

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/implement"
	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// deadLane is a lane whose implementer committed once and then died with work
// uncommitted: a staged new file, an unstaged change to the committed file, an
// untracked file in a new directory, an ignored file, and a partial receipt.
// It returns the commit, the lane, and the lane's directory.
func deadLane(t *testing.T) (*gittest.Repo, string, Lane, string, string) {
	t.Helper()
	repo, runID, l, dir := awaitingLane(t)
	head := laneCommit(t, repo, l, "one.txt")
	write := func(rel, body string) {
		t.Helper()
		p := filepath.Join(l.Worktree, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("staged.txt", "staged\n")
	repo.Git("-C", l.Worktree, "add", "--", "staged.txt")
	write("one.txt", "one\nunstaged edit\n")
	write("new/untracked.txt", "untracked\n")
	write(".abcd/.work.local/ignored.txt", "ignored\n")
	if err := os.WriteFile(filepath.Join(dir, ReceiptFileName), []byte(`{"schema_version": 1, "run_id": "`), 0o600); err != nil {
		t.Fatal(err)
	}
	return repo, runID, l, dir, head
}

// asideOf reads the one aside a restart saved under the lane's directory.
func asideOf(t *testing.T, dir string) (string, map[string]any) {
	t.Helper()
	stamps, err := os.ReadDir(filepath.Join(dir, "aside"))
	if err != nil || len(stamps) != 1 {
		t.Fatalf("one aside is saved under the lane's directory: %v %v", stamps, err)
	}
	at := filepath.Join(dir, "aside", stamps[0].Name())
	data, err := os.ReadFile(filepath.Join(at, AsideFileName))
	if err != nil {
		t.Fatal(err)
	}
	var meta map[string]any
	if err := json.Unmarshal(data, &meta); err != nil {
		t.Fatalf("aside.json is JSON: %v\n%s", err, data)
	}
	return at, meta
}

// noAside fails the test when a restart saved anything aside.
func noAside(t *testing.T, dir string) {
	t.Helper()
	if _, err := os.Lstat(filepath.Join(dir, "aside")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("a refused restart saves nothing aside (%v)", err)
	}
}

// editsIntact fails the test unless the dead agent's edits are where it left them.
func editsIntact(t *testing.T, repo *gittest.Repo, l Lane) {
	t.Helper()
	status := repo.Git("-C", l.Worktree, "status", "--porcelain", "--untracked-files=all")
	// The fixture's output is trimmed, so the first line loses its leading
	// space: the unstaged change reads "M one.txt" where it leads.
	for _, want := range []string{"A  staged.txt", "M one.txt", "?? new/untracked.txt"} {
		if !strings.Contains(status, want) || strings.Contains(status, "M  one.txt") {
			t.Fatalf("a refused restart leaves the worktree as it was; status lacks %q:\n%s", want, status)
		}
	}
}

// TestRestartSavesAsideThenResetsToLastCommit: a lane whose implementer died
// restarts from its last commit. Everything uncommitted — staged, unstaged and
// untracked — is saved aside as one patch that applies to that commit, with
// aside.json naming the head, the files and why, and the partial receipt copied
// beside it; the worktree is then clean at the commit, an ignored file kept;
// the implementer await is re-told for a fresh agent; the run record names the
// aside, and the brief does not.
func TestRestartSavesAsideThenResetsToLastCommit(t *testing.T) {
	repo, runID, l, dir, head := deadLane(t)

	res, err := Restart(repo.Root(), runID, "lane-1", "", Options{})
	if err != nil {
		t.Fatal(err)
	}

	at, meta := asideOf(t, dir)
	if meta["head"] != head || meta["why"] != "agent died" || meta["lane"] != "lane-1" || meta["run_id"] != runID {
		t.Fatalf("aside.json names the head, the lane and why: %v", meta)
	}
	var files []string
	for _, f := range meta["files"].([]any) {
		files = append(files, f.(string))
	}
	for _, want := range []string{"new/untracked.txt", "one.txt", "staged.txt"} {
		if !slices.Contains(files, want) {
			t.Fatalf("aside.json lists %s among the files saved: %v", want, files)
		}
	}
	if slices.Contains(files, ".abcd/.work.local/ignored.txt") {
		t.Fatalf("an ignored file is not saved aside: %v", files)
	}
	if got, err := os.ReadFile(filepath.Join(at, ReceiptFileName)); err != nil || !strings.Contains(string(got), `"run_id"`) {
		t.Fatalf("the partial receipt is copied aside: %q %v", got, err)
	}
	if _, err := os.Lstat(filepath.Join(dir, ReceiptFileName)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("the partial receipt no longer waits at the await's path for the fresh agent (%v)", err)
	}

	if got := strings.TrimSpace(repo.Git("-C", l.Worktree, "rev-parse", "HEAD")); got != head {
		t.Fatalf("the lane stays at its last commit: %s, want %s", got, head)
	}
	if status := repo.Git("-C", l.Worktree, "status", "--porcelain", "--untracked-files=all"); strings.TrimSpace(status) != "" {
		t.Fatalf("the worktree is clean at its last commit:\n%s", status)
	}
	if _, err := os.Stat(filepath.Join(l.Worktree, ".abcd", ".work.local", "ignored.txt")); err != nil {
		t.Fatalf("an ignored file is left where it is: %v", err)
	}
	patch := filepath.Join(at, AsidePatchName)
	repo.Git("-C", l.Worktree, "apply", "--check", "--", patch)
	if body, _ := os.ReadFile(patch); !strings.Contains(string(body), "unstaged edit") || !strings.Contains(string(body), "untracked") {
		t.Fatalf("the patch carries the unstaged and untracked edits:\n%s", body)
	}

	if res.Awaiting == nil || res.Awaiting.Role != RoleImplementer || res.Awaiting.Receipt != l.awaiting().Receipt || res.Awaiting.Brief != l.Brief {
		t.Fatalf("the implementer await is re-told: %+v", res.Awaiting)
	}
	if !strings.Contains(res.Next, "start a fresh implementer") {
		t.Fatalf("the lead is told to start a fresh agent: %q", res.Next)
	}
	wantAside := RunRelDir + "/" + runID + "/lane-1/aside/" + filepath.Base(at)
	if res.Aside.Path != wantAside {
		t.Fatalf("the result names the aside: %q, want %q", res.Aside.Path, wantAside)
	}
	st, err := ReadState(repo.Root(), runID)
	if err != nil {
		t.Fatal(err)
	}
	if a := st.Lanes[0].awaiting(); a == nil || a.Role != RoleImplementer || len(st.Lanes[0].Awaits) != 1 {
		t.Fatalf("the lane still awaits one implementer: %+v", st.Lanes[0].Awaits)
	}
	rec, err := ReadRecord(repo.Root(), runID)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, e := range rec.Record {
		if e.Stage == "restart" && e.Lane == "lane-1" && strings.Contains(e.Note, wantAside) && strings.Contains(e.Note, "agent died") {
			found = true
		}
	}
	if !found {
		t.Fatalf("the run record names the aside for review: %+v", rec.Record)
	}
	brief, err := os.ReadFile(filepath.Join(repo.Root(), filepath.FromSlash(l.Brief)))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(brief), "aside") {
		t.Fatal("the implementer's brief never mentions the aside")
	}
}

// TestRestartRecordsYieldReason: an agent that yielded on a network failure
// names its NETWORK: line, and the aside and the record carry it as the why. A
// yield that is not a NETWORK: line is refused, changing nothing.
func TestRestartRecordsYieldReason(t *testing.T) {
	repo, runID, l, dir, _ := deadLane(t)

	if _, err := Restart(repo.Root(), runID, "lane-1", "it felt like stopping", Options{}); err == nil {
		t.Fatal("a yield that is not a NETWORK: line is refused")
	} else {
		mustRefusal(t, err)
	}
	noAside(t, dir)
	editsIntact(t, repo, l)

	// A yield that is blank is no yield: it is refused, not read as a death.
	if _, err := Restart(repo.Root(), runID, "lane-1", " \t\n", Options{}); err == nil {
		t.Fatal("a blank yield is refused")
	} else {
		mustRefusal(t, err)
	}
	noAside(t, dir)
	editsIntact(t, repo, l)

	if _, err := Restart(repo.Root(), runID, "lane-1", "  NETWORK: git push origin build/x \n", Options{}); err != nil {
		t.Fatal(err)
	}
	_, meta := asideOf(t, dir)
	want := "agent yielded: NETWORK: git push origin build/x"
	if meta["why"] != want {
		t.Fatalf("aside.json's why is %q, want %q", meta["why"], want)
	}
	rec, err := ReadRecord(repo.Root(), runID)
	if err != nil {
		t.Fatal(err)
	}
	if e := rec.Record[len(rec.Record)-1]; e.Stage != "restart" || !strings.Contains(e.Note, want) {
		t.Fatalf("the record names the yield: %+v", e)
	}
}

// TestRestartRefusedWhileOutageOpen: while the run's outage is open, a restart
// is refused as a wait and changes nothing.
func TestRestartRefusedWhileOutageOpen(t *testing.T) {
	repo, runID, l, dir, _ := deadLane(t)
	run, err := implement.Open(gitutil.RootCommit(repo.Root()))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := run.Join("lead", implement.RoleFirst, "", "", 0); err != nil {
		t.Fatal(err)
	}
	if _, err := run.Outage().Record("lead", implement.ServiceNetwork, implement.OutageAgent, "lane-1", "git push"); err != nil {
		t.Fatal(err)
	}
	before := stateBytes(t, repo.Root(), runID)

	_, err = Restart(repo.Root(), runID, "lane-1", "", Options{})
	r := mustRefusal(t, err)
	if !r.Contention || !strings.Contains(r.Reason, "outage") {
		t.Fatalf("an open outage is a wait, named: %+v", r)
	}
	noAside(t, dir)
	editsIntact(t, repo, l)
	if after := stateBytes(t, repo.Root(), runID); string(before) != string(after) {
		t.Fatal("a refused restart leaves the state unchanged")
	}
}

// TestRestartRefusedAfterOutageGaveUp: once the run has given up on an
// outage, a restart is refused (not a wait), changes nothing, and names the
// way on: clearing the outage once the connection is back.
func TestRestartRefusedAfterOutageGaveUp(t *testing.T) {
	repo, runID, l, dir, _ := deadLane(t)
	run, err := implement.Open(gitutil.RootCommit(repo.Root()))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	run.Now = func() time.Time { return now }
	if _, err := run.Join("lead", implement.RoleFirst, "", "", 0); err != nil {
		t.Fatal(err)
	}
	giveUp(t, run, "lead", func(d time.Duration) { now = now.Add(d) })
	if cur, err := run.Outage().Current(); err != nil || cur == nil || cur.Status != implement.OutageGaveUp {
		t.Fatalf("the outage gave up: %+v, %v", cur, err)
	}
	before := stateBytes(t, repo.Root(), runID)

	_, err = Restart(repo.Root(), runID, "lane-1", "", Options{})
	r := mustRefusal(t, err)
	if r.Contention || !strings.Contains(r.Reason, "gave up") || !strings.Contains(r.Remedy, "implement outage clear") {
		t.Fatalf("a given-up outage refuses the restart, naming the clear: %+v", r)
	}
	noAside(t, dir)
	editsIntact(t, repo, l)
	if after := stateBytes(t, repo.Root(), runID); string(before) != string(after) {
		t.Fatal("a refused restart leaves the state unchanged")
	}
}

// TestRestartRefusedWithoutImplementerAwait: a lane with no implementer out —
// and a lane the run does not have — is refused, changing nothing.
func TestRestartRefusedWithoutImplementerAwait(t *testing.T) {
	repo, runID, l, dir, _ := deadLane(t)
	if _, err := Restart(repo.Root(), runID, "lane-9", "", Options{}); err == nil {
		t.Fatal("a lane the run does not have is refused")
	} else {
		mustRefusal(t, err)
	}
	if err := mutate(repo.Root(), runID, func(_ *os.Root, st *State) (bool, error) {
		st.Lanes[0].Awaits = nil
		return true, nil
	}); err != nil {
		t.Fatal(err)
	}
	before := stateBytes(t, repo.Root(), runID)
	_, err := Restart(repo.Root(), runID, "lane-1", "", Options{})
	if r := mustRefusal(t, err); !strings.Contains(r.Reason, "implementer") {
		t.Fatalf("the refusal names the missing implementer: %+v", r)
	}
	noAside(t, dir)
	editsIntact(t, repo, l)
	if after := stateBytes(t, repo.Root(), runID); string(before) != string(after) {
		t.Fatal("a refused restart leaves the state unchanged")
	}
}

// TestRestartRefusedForForeignWorktreePath: a lane whose state names a worktree
// other than the one the loop derives for it — here the main checkout — is
// refused before anything is saved or reset there.
func TestRestartRefusedForForeignWorktreePath(t *testing.T) {
	repo, runID, _, dir, _ := deadLane(t)
	repo.Write("precious.txt", "the main checkout's own work\n")
	if err := mutate(repo.Root(), runID, func(_ *os.Root, st *State) (bool, error) {
		st.Lanes[0].Worktree = repo.Root()
		return true, nil
	}); err != nil {
		t.Fatal(err)
	}
	_, err := Restart(repo.Root(), runID, "lane-1", "", Options{})
	if r := mustRefusal(t, err); !strings.Contains(r.Reason, "worktree") {
		t.Fatalf("the refusal names the worktree: %+v", r)
	}
	noAside(t, dir)
	if got, err := os.ReadFile(filepath.Join(repo.Root(), "precious.txt")); err != nil || !strings.Contains(string(got), "own work") {
		t.Fatalf("the foreign checkout is untouched: %q %v", got, err)
	}
}

// TestRestartChangesNothingWhenPatchFailsCheck: a patch that does not apply to
// the lane's last commit is never trusted as the saved work: the restart is
// refused, nothing is saved aside, and the worktree, the receipt and the state
// are as they were.
func TestRestartChangesNothingWhenPatchFailsCheck(t *testing.T) {
	repo, runID, l, dir, _ := deadLane(t)
	orig := checkAsidePatch
	t.Cleanup(func() { checkAsidePatch = orig })
	checkAsidePatch = func(worktree, patch string) error { return errors.New("error: patch failed: one.txt:1") }
	before := stateBytes(t, repo.Root(), runID)

	_, err := Restart(repo.Root(), runID, "lane-1", "", Options{})
	if r := mustRefusal(t, err); !strings.Contains(r.Reason, "patch failed") {
		t.Fatalf("the refusal carries git's reason: %+v", r)
	}
	noAside(t, dir)
	editsIntact(t, repo, l)
	if _, err := os.Stat(filepath.Join(dir, ReceiptFileName)); err != nil {
		t.Fatalf("the partial receipt is left where it was: %v", err)
	}
	if after := stateBytes(t, repo.Root(), runID); string(before) != string(after) {
		t.Fatal("a refused restart leaves the state unchanged")
	}
}
