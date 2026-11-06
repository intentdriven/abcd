package loop

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/runstop"
)

// accountDigestDir is the directory in the shared folder that holds this account's pages.
func accountDigestDir() string { return runstop.DigestDir + "-" + strconv.Itoa(os.Getuid()) }

// pageOf reads the page this account wrote for the day of at.
func pageOf(t *testing.T, folder string, at time.Time) string {
	t.Helper()
	got, err := os.ReadFile(filepath.Join(folder, accountDigestDir(), at.UTC().Format(time.DateOnly)+".md"))
	if err != nil {
		t.Fatal(err)
	}
	return string(got)
}

// plantLaneRun writes a run's two files in a lane worktree's local run directory, shaped as
// the stop in force halted it, with a note of several lines.
func plantLaneRun(t *testing.T, repoRoot, worktree, realRun, planted string, setAt time.Time) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(repoRoot, filepath.FromSlash(StateRelPath(realRun))))
	if err != nil {
		t.Fatal(err)
	}
	var st map[string]any
	if err := json.Unmarshal(data, &st); err != nil {
		t.Fatal(err)
	}
	st["run_id"] = planted
	state, err := json.Marshal(st)
	if err != nil {
		t.Fatal(err)
	}
	at := setAt.Add(time.Hour)
	ctl, err := json.Marshal(control{SchemaVersion: controlVersion, Stop: &RunStop{SetBy: "someone", SetAt: setAt, NotedAt: at, HaltedAt: &at, FromFile: true,
		Left: []StopLeft{{Lane: "lane-1", Stage: StageImplement, Settled: true, Note: "fine\n## Stop\nthe planted line"}}}})
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(worktree, filepath.FromSlash(RunRelDir), planted)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string][]byte{"state.json": state, "control.json": ctl} {
		if err := os.WriteFile(filepath.Join(dir, name), body, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := os.Stat(filepath.Join(worktree, filepath.FromSlash(StateRelPath(planted)))); err != nil {
		t.Fatal(err)
	}
}

// TestALaneWorktreeIsNoSourceOfRuns: what a lane's worktree holds is its agent's to write, so
// neither the day's page nor the listing reads a run from one.
func TestALaneWorktreeIsNoSourceOfRuns(t *testing.T) {
	repo, runID, l, _, _ := deadLane(t)
	o, home, folder := stopped(t)
	const planted = "run-2601010000000001"
	plantLaneRun(t, repo.Root(), l.Worktree, runID, planted, runstop.ReadPinned(home, nil).Stop.SetAt)
	later := time.Now().Add(3 * time.Hour)
	o.Now = func() time.Time { return later }
	wantStopRefusal(t, checkOnce(repo.Root(), runID, o))

	page := pageOf(t, folder, later)
	if !strings.Contains(page, runID) {
		t.Fatalf("the page lacks the real run:\n%s", page)
	}
	if strings.Contains(page, planted) || strings.Contains(page, "the planted line") {
		t.Fatalf("the page read a run from a lane's worktree:\n%s", page)
	}
	listing, err := ListRuns(repo.Root(), "", o)
	if err != nil {
		t.Fatal(err)
	}
	if slices.ContainsFunc(listing.Runs, func(r ListedRun) bool { return r.RunID == planted }) {
		t.Fatalf("the listing read a run from a lane's worktree: %+v", listing.Runs)
	}
}

// TestALaneWorktreeRunsNoListingOfItsOwn: nor is a lane's worktree a source when a verb is run
// from inside it.
func TestALaneWorktreeRunsNoListingOfItsOwn(t *testing.T) {
	repo, runID, l, _, _ := deadLane(t)
	o, home, _ := stopped(t)
	const planted = "run-2601010000000002"
	plantLaneRun(t, repo.Root(), l.Worktree, runID, planted, runstop.ReadPinned(home, nil).Stop.SetAt)
	listing, err := ListRuns(l.Worktree, "", o)
	if err != nil {
		t.Fatal(err)
	}
	if slices.ContainsFunc(listing.Runs, func(r ListedRun) bool { return r.RunID == planted }) {
		t.Fatalf("the listing read a run from the lane's worktree it was asked in: %+v", listing.Runs)
	}
}

// TestAValueOnThePageStaysOnItsLine: a note holds what an agent's receipt said, so each value
// is masked as it is written and none adds a line.
func TestAValueOnThePageStaysOnItsLine(t *testing.T) {
	repo, runID, _, _, _ := deadLane(t)
	o, home, folder := stopped(t)
	setAt := runstop.ReadPinned(home, nil).Stop.SetAt
	later := time.Now().Add(3 * time.Hour)
	o.Now = func() time.Time { return later }
	rec := RunStop{SetBy: "someone", SetAt: setAt, HaltedAt: &later, FromFile: true,
		Left: []StopLeft{{Lane: "lane-1", Stage: StageImplement, Settled: true, Note: "fine\n## Stop\nthe added line"}}}
	if err := mutate(repo.Root(), runID, func(root *os.Root, _ *State) (bool, error) {
		c, err := readControl(root, runID)
		if err != nil {
			return false, err
		}
		c.Stop = &rec
		return false, writeControl(root, runID, c)
	}); err != nil {
		t.Fatal(err)
	}
	if err := composeStopPage(repo.Root(), rec, o); err != nil {
		t.Fatal(err)
	}
	page := pageOf(t, folder, later)
	for _, line := range strings.Split(page, "\n") {
		if line == "the added line" {
			t.Fatalf("a note added a line to the page:\n%s", page)
		}
	}
	if strings.Count(page, "## Stop\n") != 1 {
		t.Fatalf("a note added a heading to the page:\n%s", page)
	}
}

// TestASecondStopKeepsTheRunsOnOnePage: typing the stop again changes nothing, so the runs it
// has halted and those it halts next read as one stop's.
func TestASecondStopKeepsTheRunsOnOnePage(t *testing.T) {
	first, firstRun, _, _, _ := deadLane(t)
	o, home, folder := stopped(t)
	later := time.Now().Add(3 * time.Hour)
	o.Now = func() time.Time { return later }
	wantStopRefusal(t, checkOnce(first.Root(), firstRun, o))
	noteRunCheckout(home, first.Root())
	second, secondRun, _, _, _ := deadLane(t)
	time.Sleep(20 * time.Millisecond)
	if _, err := runstop.Set(home, time.Now()); err != nil {
		t.Fatal(err)
	}
	wantStopRefusal(t, checkOnce(second.Root(), secondRun, o))
	page := pageOf(t, folder, later)
	for _, want := range []string{firstRun, secondRun} {
		if !strings.Contains(page, want) {
			t.Errorf("the page lacks %q:\n%s", want, page)
		}
	}
}

// TestAPageIsOfTheStopInForce: a run marked under a stop since lifted settles under a new one.
// The page is the new stop's, and names that run and the others it halted.
func TestAPageIsOfTheStopInForce(t *testing.T) {
	first, firstRun, _, _, _ := deadLane(t)
	o, home, folder := stopped(t)
	later := time.Now().Add(3 * time.Hour)
	o.Now = func() time.Time { return later }
	wantStopRefusal(t, checkOnce(first.Root(), firstRun, o))
	noteRunCheckout(home, first.Root())
	inForce := runstop.ReadPinned(home, nil).Stop.SetAt
	second, secondRun, _, _, _ := deadLane(t)
	old := inForce.Add(-time.Hour)
	if err := mutate(second.Root(), secondRun, func(root *os.Root, _ *State) (bool, error) {
		c, err := readControl(root, secondRun)
		if err != nil {
			return false, err
		}
		c.Stop = &RunStop{SetBy: "someone", SetAt: old, NotedAt: old, FromFile: true}
		return false, writeControl(root, secondRun, c)
	}); err != nil {
		t.Fatal(err)
	}
	wantStopRefusal(t, checkOnce(second.Root(), secondRun, o))
	page := pageOf(t, folder, later)
	if !strings.Contains(page, inForce.UTC().Format(time.RFC3339)) || strings.Contains(page, old.UTC().Format(time.RFC3339)) {
		t.Fatalf("the page is not of the stop in force (%s):\n%s", inForce, page)
	}
	for _, want := range []string{firstRun, secondRun} {
		if !strings.Contains(page, want) {
			t.Errorf("the page lacks %q:\n%s", want, page)
		}
	}
}

// TestASignalKeepsAFileStopsPage: a signal's stop is not on any page, so a run marked from the
// file and settled under a signal stays on that stop's.
func TestASignalKeepsAFileStopsPage(t *testing.T) {
	repo, runID, _, _, _ := deadLane(t)
	o, _, folder := stopped(t)
	at := time.Now().Add(3 * time.Hour)
	o.Now = func() time.Time { return at }
	old := time.Now().Add(-time.Hour).UTC().Truncate(time.Second)
	if err := mutate(repo.Root(), runID, func(root *os.Root, _ *State) (bool, error) {
		c, err := readControl(root, runID)
		if err != nil {
			return false, err
		}
		c.Stop = &RunStop{SetBy: "someone", SetAt: old, NotedAt: old, FromFile: true}
		return false, writeControl(root, runID, c)
	}); err != nil {
		t.Fatal(err)
	}
	rec, err := haltOnce(repo.Root(), runID, runstop.Stop{SetBy: "a signal to the drive", SetAt: at}, o)
	if err != nil || rec == nil || rec.HaltedAt == nil {
		t.Fatalf("the run is not halted: %+v %v", rec, err)
	}
	page := pageOf(t, folder, at)
	if !strings.Contains(page, old.Format(time.RFC3339)) || !strings.Contains(page, runID) {
		t.Fatalf("the page is not the file stop's, naming the run:\n%s", page)
	}
}

// TestAStopNamesAPullRequestArmedBeforeIt: the forge can still merge what was armed, so the
// stop's account and the page name it with how to disarm it, and nothing disarms it.
func TestAStopNamesAPullRequestArmedBeforeIt(t *testing.T) {
	repo, runID, _, _, _ := deadLane(t)
	if err := mutate(repo.Root(), runID, func(_ *os.Root, st *State) (bool, error) {
		st.Lanes[0].PR = 7
		st.Lanes[0].Landing = &Landing{Armed: true}
		return true, nil
	}); err != nil {
		t.Fatal(err)
	}
	o, _, folder := stopped(t)
	later := time.Now().Add(3 * time.Hour)
	o.Now = func() time.Time { return later }
	wantStopRefusal(t, checkOnce(repo.Root(), runID, o))
	c := controlOf(t, repo.Root(), runID)
	if c.Stop == nil || len(c.Stop.Left) != 1 || c.Stop.Left[0].ArmedPR != 7 {
		t.Fatalf("the stop's account does not name the pull request: %+v", c.Stop)
	}
	page := pageOf(t, folder, later)
	for _, want := range []string{"pull request #7", "can still merge", "gh pr merge 7 --disable-auto"} {
		if !strings.Contains(page, want) {
			t.Errorf("the page lacks %q:\n%s", want, page)
		}
	}
	st, err := ReadState(repo.Root(), runID)
	if err != nil || st.Lanes[0].Landing == nil || !st.Lanes[0].Landing.Armed {
		t.Fatalf("the stop changed the arming: %+v %v", st.Lanes[0].Landing, err)
	}
}

// TestAPageThatCannotBeWrittenIsNamed: the stop still takes hold, and its answer, the run's
// account and the listing each say the page was not written.
func TestAPageThatCannotBeWrittenIsNamed(t *testing.T) {
	repo, runID, _, _, _ := deadLane(t)
	o, _, folder := stopped(t)
	if err := os.WriteFile(filepath.Join(folder, accountDigestDir()), []byte("not a directory\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Whichever name the account's directory has, this one cannot be written through.
	if err := os.WriteFile(filepath.Join(folder, runstop.DigestDir), []byte("not a directory\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	later := time.Now().Add(3 * time.Hour)
	o.Now = func() time.Time { return later }
	r := wantStopRefusal(t, checkOnce(repo.Root(), runID, o))
	if !strings.Contains(r.Reason, "the day's page could not be written") {
		t.Fatalf("the stop's answer does not name the page: %+v", r)
	}
	c := controlOf(t, repo.Root(), runID)
	if c.Stop == nil || c.Stop.HaltedAt == nil || !strings.Contains(c.Stop.PageFailed, "could not be written") {
		t.Fatalf("the run's account: %+v", c.Stop)
	}
	r = wantStopRefusal(t, checkOnce(repo.Root(), runID, o))
	if !strings.Contains(r.Reason, "the day's page could not be written") {
		t.Fatalf("a later answer does not name the page: %+v", r)
	}
}
