package loop

// receipt_consumed_test.go proves iss-2611071242184002 in the step: where a drive
// is its own driver, an implementer's receipt refused for the work is set aside,
// kept, and the role is started again, so no step reads the same refused receipt
// twice with no launch between. A receipt refused for anything else, and every
// host-stepped run, keep the stop receipt_standing.go describes.

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/core/runner"
)

// refusedForTheWork is a routed implementer's lane whose fake agent writes, at
// every launch, a receipt that names this run, lane and branch and whose
// definition of done exited 1, with its runner on the claude fake; and the
// receipt's bytes.
func refusedForTheWork(t *testing.T, edit func(*LaneReceipt)) (root, id string, l Lane, rs Runners, env driveEnv, body string) {
	t.Helper()
	repo, id, l, dir := awaitingLane(t)
	c := laneCommit(t, repo, l, "one.txt")
	rc := goodReceipt(t, id, l, dir, c)
	one := 1
	rc.DefinitionOfDone.ExitCode = &one
	if edit != nil {
		edit(&rc)
	}
	path := writeReceipt(t, t.TempDir(), rc)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv(loopFakeReceiptEnv, path)
	env = newDriveEnv(t, "receipt", "claude")
	rs = Runners{Config: runnerConfig(t, claudeImplementer, ""), Transcripts: &memTranscripts{}}
	return repo.Root(), id, l, rs, env, string(raw)
}

// asides counts the receipts kept under the lane's aside directory.
func asides(t *testing.T, root, id, lane string) int {
	t.Helper()
	n := 0
	dir := filepath.Join(root, filepath.FromSlash(RunRelDir), id, lane, AsideDirName)
	_ = filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() && d.Name() == ReceiptFileName {
			n++
		}
		return nil
	})
	return n
}

// stands reports whether anything stands at the lane's receipt path.
func stands(t *testing.T, root string, l Lane) bool {
	t.Helper()
	_, err := os.Lstat(absIn(root, l.awaiting().Receipt))
	return !errors.Is(err, os.ErrNotExist)
}

// TestADrivesReceiptRefusedForTheWorkIsSetAsideAndItsRoleStartedAgain: the
// runner's hand-in is refused for a failing definition of done; a drive sets the
// receipt aside in the same write, kept with its refusal in the record, and the
// next step starts the implementer again, each refusal kept apart from the last.
// A host-stepped run keeps the stop: the receipt stands and no runner is started
// over it.
func TestADrivesReceiptRefusedForTheWorkIsSetAsideAndItsRoleStartedAgain(t *testing.T) {
	t.Run("a drive", func(t *testing.T) {
		root, id, l, rs, env, body := refusedForTheWork(t, nil)
		o := Options{driverIsSelf: true}
		res, err := Drive(context.Background(), root, id, DefaultStages(), o, rs)
		if err != nil {
			t.Fatal(err)
		}
		if res.Paused == nil || res.Paused.Paused == nil || res.Paused.Paused.Reason != runner.ReasonInvalid {
			t.Fatalf("the refused hand-in pauses the role: %+v", res)
		}
		if stands(t, root, l) {
			t.Fatal("nothing stands at the receipt's path once it is refused for the work")
		}
		if !asideHolds(t, root, id, l.ID, body) {
			t.Fatal("the refused receipt is kept in the lane's aside directory")
		}
		if !recordSays(t, root, id, StageRunner, "refused for the work", "it exited 1", "set aside at", "started again", "sha256 "+answerDigest([]byte(body))) {
			t.Fatal("the record says the receipt was set aside, with what refusal and the digest of the bytes judged")
		}
		// The files that go aside with it: TestTheAssignedReportAndOutputGoAsideWithAConsumedReceipt.
		if strings.Contains(res.Next, "no runner is started over") {
			t.Fatalf("the step says nothing false of the lane: %s", res.Next)
		}
		if _, err := Drive(context.Background(), root, id, DefaultStages(), o, rs); err != nil {
			t.Fatal(err)
		}
		if n := env.launches(t, "claude"); n != 2 {
			t.Fatalf("the next step starts the implementer again: %d launches", n)
		}
		if !recordSays(t, root, id, StageRetry, l.ID, RoleImplementer) {
			t.Fatal("the record names the retry")
		}
		if n := asides(t, root, id, l.ID); n != 2 {
			t.Fatalf("each refused receipt is kept apart, none written over: %d kept", n)
		}
	})
	t.Run("a host-stepped run", func(t *testing.T) {
		root, id, l, rs, env, body := refusedForTheWork(t, nil)
		for range 3 {
			if _, err := Drive(context.Background(), root, id, DefaultStages(), Options{}, rs); err != nil {
				t.Fatal(err)
			}
		}
		if n := env.launches(t, "claude"); n != 1 {
			t.Fatalf("no runner is started over a host-stepped run's receipt: %d launches", n)
		}
		if raw, err := os.ReadFile(absIn(root, l.awaiting().Receipt)); err != nil || string(raw) != body {
			t.Fatalf("the receipt stands where it was written: %v", err)
		}
	})
}

// TestAClosedWindowDelaysTheRelaunchOfAConsumedReceipt: a paced run's window
// that closes at the step after the refusal starts nothing, and the step after
// the pause starts the implementer again.
func TestAClosedWindowDelaysTheRelaunchOfAConsumedReceipt(t *testing.T) {
	root, id, _, rs, env, _ := refusedForTheWork(t, nil)
	c := &clock{t: time.Now().UTC()}
	o := Options{driverIsSelf: true, Now: c.now}
	if _, err := Drive(context.Background(), root, id, DefaultStages(), o, rs); err != nil {
		t.Fatal(err)
	}
	if err := mutate(root, id, func(_ *os.Root, st *State) (bool, error) {
		started := c.now().Add(-time.Hour)
		bundled := func(v int) PaceValue { return PaceValue{Value: v, Layer: "bundled"} }
		st.Pace = &Pace{WorkMinutes: bundled(30), PauseMinutes: bundled(10), SubAgents: bundled(1)}
		st.WindowStartedAt = &started
		return true, nil
	}); err != nil {
		t.Fatal(err)
	}
	res, err := Drive(context.Background(), root, id, DefaultStages(), o, rs)
	if err != nil {
		t.Fatal(err)
	}
	if res.NextEligibleAt == nil || env.launches(t, "claude") != 1 {
		t.Fatalf("the closed window starts nothing: %+v, %d launches", res, env.launches(t, "claude"))
	}
	c.t = res.NextEligibleAt.Add(time.Second)
	if _, err := Drive(context.Background(), root, id, DefaultStages(), o, rs); err != nil {
		t.Fatal(err)
	}
	if n := env.launches(t, "claude"); n != 2 {
		t.Fatalf("the step after the pause starts the implementer again: %d launches", n)
	}
}

// TestAReceiptRefusedForAnythingButTheWorkIsNotConsumed: a receipt naming another
// run, or one that is not a receipt at all, is not the lane's work to judge, so a
// drive leaves it standing and starts no runner over it, as a host-stepped run does.
func TestAReceiptRefusedForAnythingButTheWorkIsNotConsumed(t *testing.T) {
	for name, edit := range map[string]func(*LaneReceipt){
		"another run's":    func(rc *LaneReceipt) { rc.RunID = "run-2609300000000000" },
		"another lane's":   func(rc *LaneReceipt) { rc.Lane = "lane-9" },
		"another schema's": func(rc *LaneReceipt) { rc.SchemaVersion = ReceiptSchemaVersion + 1 },
		"another branch's": func(rc *LaneReceipt) { rc.Branch = "build/another" },
	} {
		t.Run(name, func(t *testing.T) {
			root, id, l, rs, env, body := refusedForTheWork(t, edit)
			wantKept(t, root, id, l, rs, env, body)
		})
	}
	t.Run("not a receipt", func(t *testing.T) {
		root, id, l, rs, env, _ := refusedForTheWork(t, nil)
		body := "half a receipt"
		t.Setenv(loopFakeReceiptEnv, writeReceipt(t, t.TempDir(), body))
		wantKept(t, root, id, l, rs, env, body)
	})
}

// wantKept drives three steps and wants the one launch and the receipt standing.
func wantKept(t *testing.T, root, id string, l Lane, rs Runners, env driveEnv, body string) {
	t.Helper()
	for range 3 {
		if _, err := Drive(context.Background(), root, id, DefaultStages(), Options{driverIsSelf: true}, rs); err != nil {
			t.Fatal(err)
		}
	}
	if n := env.launches(t, "claude"); n != 1 {
		t.Fatalf("no runner is started over it: %d launches", n)
	}
	if raw, err := os.ReadFile(absIn(root, l.awaiting().Receipt)); err != nil || string(raw) != body {
		t.Fatalf("the receipt stands where it was written: %q %v", raw, err)
	}
	if asides(t, root, id, l.ID) != 0 {
		t.Fatal("nothing is set aside")
	}
}

// TestAReceiptLeftStandingIsConsumedByADrive: a receipt refused for the work that
// stands from a step that did not consume it (a host's step, a step killed, an
// older abcd) is set aside by the first step a drive makes, which starts the
// implementer again in the same step; here its runner's report names the model
// the standing receipt lacked, so the lane moves on.
func TestAReceiptLeftStandingIsConsumedByADrive(t *testing.T) {
	repo, id, l, dir, stages, rs, env, _ := stalledOnModel(t)
	root := repo.Root()
	body, err := os.ReadFile(absIn(root, l.awaiting().Receipt))
	if err != nil {
		t.Fatal(err)
	}
	rs.Transcripts = &duringRun{fn: func() { goodReceipt(t, id, l, dir) }}
	res, err := Drive(context.Background(), root, id, stages, Options{driverIsSelf: true}, rs)
	if err != nil {
		t.Fatal(err)
	}
	if n := env.launches(t, "claude"); n != 2 {
		t.Fatalf("the drive's step starts the implementer again: %d launches", n)
	}
	if !asideHolds(t, root, id, l.ID, string(body)) {
		t.Fatal("the standing receipt is kept in the lane's aside directory")
	}
	if !recordSays(t, root, id, StageRunner, "refused for the work", "the model your harness runs", "set aside at") {
		t.Fatal("the record says what the standing receipt was refused for, and where it went")
	}
	if res.PerformedStage != StageImplement {
		t.Fatalf("the fresh agent's receipt verifies: %+v", res)
	}
}

// TestASetAsideThatFailsKeepsThePause: a set-aside that cannot be made, the
// lane's aside directory being a link out of the run's folder, follows no link,
// leaves the receipt where it stands and says so, and the pause is written as
// ever; on an issue run the step is counted as a refused one.
func TestASetAsideThatFailsKeepsThePause(t *testing.T) {
	root, id, l, rs, env, body := refusedForTheWork(t, nil)
	asIssueRun(t, root, id)
	away := t.TempDir()
	if err := os.Symlink(away, filepath.Join(root, filepath.FromSlash(RunRelDir), id, l.ID, AsideDirName)); err != nil {
		t.Fatal(err)
	}
	res, err := Drive(context.Background(), root, id, DefaultStages(), Options{driverIsSelf: true}, rs)
	if err != nil {
		t.Fatal(err)
	}
	if res.Paused == nil {
		t.Fatalf("the role pauses: %+v", res)
	}
	st, err := ReadState(root, id)
	if err != nil {
		t.Fatal(err)
	}
	if a := st.Lanes[0].awaiting(); a == nil || a.Paused == nil || a.Paused.Reason != runner.ReasonInvalid {
		t.Fatalf("the pause is written: %+v", st.Lanes[0].Awaits)
	}
	if raw, err := os.ReadFile(absIn(root, l.awaiting().Receipt)); err != nil || string(raw) != body {
		t.Fatalf("the receipt stands where it was written: %v", err)
	}
	if entries, _ := os.ReadDir(away); len(entries) != 0 {
		t.Fatalf("nothing is moved through the link: %v", entries)
	}
	if !recordSays(t, root, id, StageRunner, "could not be set aside", "stuck bounds") {
		t.Fatal("the record says the receipt could not be set aside, and what ends the lane")
	}
	if n := refusedCount(t, root, id); n != 1 {
		t.Fatalf("the step that could not set the receipt aside is counted: %d", n)
	}
	if n := env.launches(t, "claude"); n != 1 {
		t.Fatalf("one launch: %d", n)
	}
}

// TestAReceiptLeftStandingThatMeetsAClosingWindowIsSetAsideInThatStep: the step
// whose window closes still sets aside, and counts, a receipt it found standing
// and refused for the work, so the closed window's answer is true of the lane;
// the step after the pause starts the implementer again.
func TestAReceiptLeftStandingThatMeetsAClosingWindowIsSetAsideInThatStep(t *testing.T) {
	repo, id, l, dir, stages, rs, env, _ := stalledOnModel(t)
	root := repo.Root()
	body, err := os.ReadFile(absIn(root, l.awaiting().Receipt))
	if err != nil {
		t.Fatal(err)
	}
	c := &clock{t: time.Now().UTC()}
	o := Options{driverIsSelf: true, Now: c.now}
	elapsedWindow(t, root, id, c)
	res, err := Drive(context.Background(), root, id, stages, o, rs)
	if err != nil {
		t.Fatal(err)
	}
	if res.NextEligibleAt == nil || env.launches(t, "claude") != 1 {
		t.Fatalf("the closed window starts nothing: %+v, %d launches", res, env.launches(t, "claude"))
	}
	if stands(t, root, l) || !asideHolds(t, root, id, l.ID, string(body)) {
		t.Fatal("the step that closes the window sets the standing receipt aside")
	}
	rs.Transcripts = &duringRun{fn: func() { goodReceipt(t, id, l, dir) }}
	c.t = res.NextEligibleAt.Add(time.Second)
	if _, err := Drive(context.Background(), root, id, stages, o, rs); err != nil {
		t.Fatal(err)
	}
	if n := env.launches(t, "claude"); n != 2 {
		t.Fatalf("the step after the pause starts the implementer again: %d launches", n)
	}
}

// elapsedWindow gives the run a pace whose window opened an hour before c's now.
func elapsedWindow(t *testing.T, root, id string, c *clock) {
	t.Helper()
	if err := mutate(root, id, func(_ *os.Root, st *State) (bool, error) {
		started := c.now().Add(-time.Hour)
		bundled := func(v int) PaceValue { return PaceValue{Value: v, Layer: "bundled"} }
		st.Pace = &Pace{WorkMinutes: bundled(30), PauseMinutes: bundled(10), SubAgents: bundled(1)}
		st.WindowStartedAt = &started
		return true, nil
	}); err != nil {
		t.Fatal(err)
	}
}

// refusedCount is the first lane's count of refused steps under the check every
// consumed receipt is counted under, 0 with none, failing on another check.
func refusedCount(t *testing.T, root, id string) int {
	t.Helper()
	st, err := ReadState(root, id)
	if err != nil {
		t.Fatal(err)
	}
	r := st.Lanes[0].Refused
	if r == nil {
		return 0
	}
	if r.Check != consumedCheck {
		t.Fatalf("the lane's count is of another refusal: %+v", r)
	}
	return r.Count
}

// TestAReceiptChangedSinceItWasJudgedIsLeftStanding: under the run's lock a drive
// sets aside only the bytes it judged, and only while no runner's claim stands on
// the await. A file replaced since, or an await claimed since, is left standing
// for the next step to read, and nothing is counted, on an issue run, where a
// count is kept; at both places a receipt is
// consumed, a step's read of one standing and a runner's hand-in.
func TestAReceiptChangedSinceItWasJudgedIsLeftStanding(t *testing.T) {
	replace := func(t *testing.T, root string, l Lane) string {
		t.Helper()
		body := `{"replaced": true}`
		if err := os.WriteFile(absIn(root, l.awaiting().Receipt), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		return body
	}
	claimed := func(t *testing.T, root, id string) {
		t.Helper()
		if err := mutate(root, id, func(_ *os.Root, st *State) (bool, error) {
			st.Lanes[0].Awaits[0].Running = &RunnerClaim{Runner: "claude", Session: "a-restart", Until: time.Now().Add(time.Hour)}
			return true, nil
		}); err != nil {
			t.Fatal(err)
		}
	}
	wantLeft := func(t *testing.T, root, id string, l Lane, body string) {
		t.Helper()
		if raw, err := os.ReadFile(absIn(root, l.awaiting().Receipt)); err != nil || string(raw) != body {
			t.Fatalf("the file stands for the next step to read: %q %v", raw, err)
		}
		if n := asides(t, root, id, l.ID); n != 0 {
			t.Fatalf("nothing is set aside: %d", n)
		}
		if n := refusedCount(t, root, id); n != 0 {
			t.Fatalf("nothing is counted: %d", n)
		}
		if !recordSays(t, root, id, StageRunner, "left standing for the next step to read") {
			t.Fatal("the record says why it was left")
		}
	}
	standing := func(t *testing.T, change func(t *testing.T, root, id string, l Lane) string) {
		t.Helper()
		repo, id, l, _, stages, _, _, _ := stalledOnModel(t)
		root := repo.Root()
		asIssueRun(t, root, id)
		_, took, stopped := collectStanding(root, id, stages, Options{driverIsSelf: true})
		if took || len(stopped) != 1 || !stopped[0].consume {
			t.Fatalf("the step's read finds the receipt refused for the work: %v %+v", took, stopped)
		}
		body := change(t, root, id, l)
		if err := mutate(root, id, func(r *os.Root, st *State) (bool, error) {
			changed, _ := consumeStopped(root, r, st, stopped, time.Now())
			return changed, nil
		}); err != nil {
			t.Fatal(err)
		}
		wantLeft(t, root, id, l, body)
	}
	handedIn := func(t *testing.T, change func(t *testing.T, root, id string, l Lane) string) {
		t.Helper()
		repo, id, l, _, _, _, _, _ := stalledOnModel(t)
		root := repo.Root()
		asIssueRun(t, root, id)
		judged, err := os.ReadFile(absIn(root, l.awaiting().Receipt))
		if err != nil {
			t.Fatal(err)
		}
		rf := refuse("receipt", "", l.ID, "the receipt is missing a passing definition of done (it exited 1)", "finish the lane")
		rf.work, rf.judged = true, answerDigest(judged)
		body := change(t, root, id, l)
		st, err := ReadState(root, id)
		if err != nil {
			t.Fatal(err)
		}
		fb := runner.FallbackReceipt{Role: RoleImplementer, Asked: "claude", Reason: runner.ReasonInvalid, Detail: "refused"}
		if _, _, err := pauseAwait(root, id, l.ID, st.Lanes[0].Awaits[0], "the-runner", fb, "", rf, Options{driverIsSelf: true}); err != nil {
			t.Fatal(err)
		}
		wantLeft(t, root, id, l, body)
	}
	unchanged := func(t *testing.T, root, id string, l Lane) string {
		t.Helper()
		claimed(t, root, id)
		raw, err := os.ReadFile(absIn(root, l.awaiting().Receipt))
		if err != nil {
			t.Fatal(err)
		}
		return string(raw)
	}
	t.Run("replaced, found standing", func(t *testing.T) {
		standing(t, func(t *testing.T, root, _ string, l Lane) string { return replace(t, root, l) })
	})
	t.Run("replaced, handed in", func(t *testing.T) {
		handedIn(t, func(t *testing.T, root, _ string, l Lane) string { return replace(t, root, l) })
	})
	t.Run("claimed, found standing", func(t *testing.T) { standing(t, unchanged) })
	t.Run("claimed, handed in", func(t *testing.T) { handedIn(t, unchanged) })
}

// laneFileAt writes body at rel inside the lane's directory dir, making its
// parents.
func laneFileAt(t *testing.T, dir, rel, body string) {
	t.Helper()
	p := filepath.Join(dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

// TestTheAssignedReportAndOutputGoAsideWithAConsumedReceipt: the report and the
// definition of done's output, at the names the loop assigns them in the lane's
// own directory, go aside with the consumed receipt, and the record says so;
// each is its own case, so each can be watched failing. A path that cleans to
// the assigned name moves the loop's own file, never one the receipt's text
// chose.
func TestTheAssignedReportAndOutputGoAsideWithAConsumedReceipt(t *testing.T) {
	for _, tc := range []struct {
		row, name, what, body string
		edit                  func(*LaneReceipt)
	}{
		{ReportFileName, ReportFileName, "the report", "built it\n", nil},
		{DoDFileName, DoDFileName, "the definition of done's output", "ok\n", nil},
		{"./" + ReportFileName, ReportFileName, "the report", "built it\n", func(rc *LaneReceipt) { rc.Report = "./" + ReportFileName }},
		{"sub/../" + ReportFileName, ReportFileName, "the report", "built it\n", func(rc *LaneReceipt) { rc.Report = "sub/../" + ReportFileName }},
	} {
		t.Run(tc.row, func(t *testing.T) {
			root, id, l, rs, _, _ := refusedForTheWork(t, tc.edit)
			if _, err := Drive(context.Background(), root, id, DefaultStages(), Options{driverIsSelf: true}, rs); err != nil {
				t.Fatal(err)
			}
			dir := filepath.Join(root, filepath.FromSlash(RunRelDir), id, l.ID)
			if _, err := os.Lstat(filepath.Join(dir, tc.name)); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("%s is moved aside with the receipt: %v", tc.name, err)
			}
			if !asideHolds(t, root, id, l.ID, tc.body) {
				t.Fatalf("%s is kept in the lane's aside directory", tc.name)
			}
			if !recordSays(t, root, id, StageRunner, "set aside at", ", with ", tc.what+" "+tc.name) || recordSays(t, root, id, StageRunner, "left in place") {
				t.Fatalf("the record says %s went aside with the receipt", tc.name)
			}
		})
	}
}

// TestAFileTheReceiptNamesIsMovedOnlyWhenItIsTheOneAssigned: the receipt's text
// never chooses what goes aside. A report or output it names anywhere but at the
// name the loop assigned it, directly in the lane's own directory, is left where
// it is, and the record's line says so: the brief, a validator's return, a sync
// round's file, a file already set aside in any letter case, one in a
// sub-directory, a link.
func TestAFileTheReceiptNamesIsMovedOnlyWhenItIsTheOneAssigned(t *testing.T) {
	earlier := AsideDirName + "/earlier/old.txt"
	for name, tc := range map[string]struct {
		setup func(t *testing.T, dir string)
		edit  func(rc *LaneReceipt)
		left  string
	}{
		"a link": {
			setup: func(t *testing.T, dir string) {
				if err := os.Symlink(filepath.Join(dir, ReportFileName), filepath.Join(dir, "link.md")); err != nil {
					t.Fatal(err)
				}
			},
			edit: func(rc *LaneReceipt) { rc.Report = "link.md" },
			left: "link.md",
		},
		"the assigned name, a link": {
			setup: func(t *testing.T, dir string) {
				away := filepath.Join(t.TempDir(), "elsewhere.md")
				if err := os.WriteFile(away, []byte("elsewhere\n"), 0o600); err != nil {
					t.Fatal(err)
				}
				if err := os.Remove(filepath.Join(dir, ReportFileName)); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(away, filepath.Join(dir, ReportFileName)); err != nil {
					t.Fatal(err)
				}
			},
			edit: func(rc *LaneReceipt) {},
			left: ReportFileName,
		},
		"a file already set aside": {
			setup: func(t *testing.T, dir string) { laneFileAt(t, dir, earlier, "earlier\n") },
			edit:  func(rc *LaneReceipt) { rc.DefinitionOfDone.Output = earlier },
			left:  earlier,
		},
		// On a disk that ignores letter case this name reaches the file set aside;
		// on one that does not, it reaches nothing, so the row proves the point
		// only on the first kind. The exact comparison holds on either.
		"a file already set aside, named in another case": {
			setup: func(t *testing.T, dir string) { laneFileAt(t, dir, earlier, "earlier\n") },
			edit:  func(rc *LaneReceipt) { rc.DefinitionOfDone.Output = "Aside/earlier/old.txt" },
			left:  earlier,
		},
		"the brief": {
			setup: func(t *testing.T, dir string) {},
			edit:  func(rc *LaneReceipt) { rc.Report = BriefFileName },
			left:  BriefFileName,
		},
		"a validator's return": {
			setup: func(t *testing.T, dir string) {
				laneFileAt(t, dir, ValidateDirName+"/round-1/"+RoleRuthless+"/"+ReturnFileName, "a return\n")
			},
			edit: func(rc *LaneReceipt) { rc.Report = ValidateDirName + "/round-1/" + RoleRuthless + "/" + ReturnFileName },
			left: ValidateDirName + "/round-1/" + RoleRuthless + "/" + ReturnFileName,
		},
		"a sync round's file": {
			setup: func(t *testing.T, dir string) { laneFileAt(t, dir, SyncDirName+"/1/"+BriefFileName, "a sync brief\n") },
			edit:  func(rc *LaneReceipt) { rc.DefinitionOfDone.Output = SyncDirName + "/1/" + BriefFileName },
			left:  SyncDirName + "/1/" + BriefFileName,
		},
		"the assigned name in a sub-directory": {
			setup: func(t *testing.T, dir string) { laneFileAt(t, dir, "sub/"+ReportFileName, "another\n") },
			edit:  func(rc *LaneReceipt) { rc.Report = "sub/" + ReportFileName },
			left:  "sub/" + ReportFileName,
		},
		"a directory at the assigned name": {
			setup: func(t *testing.T, dir string) {
				if err := os.Remove(filepath.Join(dir, ReportFileName)); err != nil {
					t.Fatal(err)
				}
				laneFileAt(t, dir, ReportFileName+"/inside.txt", "inside\n")
			},
			edit: func(rc *LaneReceipt) {},
			left: ReportFileName,
		},
		"the assigned output name, a link": {
			setup: func(t *testing.T, dir string) {
				away := filepath.Join(t.TempDir(), "elsewhere.log")
				if err := os.WriteFile(away, []byte("elsewhere\n"), 0o600); err != nil {
					t.Fatal(err)
				}
				if err := os.Remove(filepath.Join(dir, DoDFileName)); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(away, filepath.Join(dir, DoDFileName)); err != nil {
					t.Fatal(err)
				}
			},
			edit: func(rc *LaneReceipt) {},
			left: DoDFileName,
		},
	} {
		t.Run(name, func(t *testing.T) {
			root, id, l, rs, _, _ := refusedForTheWork(t, tc.edit)
			dir := filepath.Join(root, filepath.FromSlash(RunRelDir), id, l.ID)
			tc.setup(t, dir)
			at := filepath.Join(dir, filepath.FromSlash(tc.left))
			before := standsAs(t, at)
			if _, err := Drive(context.Background(), root, id, DefaultStages(), Options{driverIsSelf: true}, rs); err != nil {
				t.Fatal(err)
			}
			if stands(t, root, l) {
				t.Fatal("the receipt is set aside")
			}
			if after := standsAs(t, at); after != before {
				t.Fatalf("%s is left where it is: %q, was %q", tc.left, after, before)
			}
			if !recordSays(t, root, id, StageRunner, "set aside at", "left in place") {
				t.Fatal("the record says what was left in place")
			}
		})
	}
}

// standsAs is what stands at p, without following a link: a link's target, a
// directory, or a file's bytes; "" when nothing does.
func standsAs(t *testing.T, p string) string {
	t.Helper()
	fi, err := os.Lstat(p)
	switch {
	case errors.Is(err, os.ErrNotExist):
		return ""
	case err != nil:
		t.Fatal(err)
	case fi.Mode()&os.ModeSymlink != 0:
		to, err := os.Readlink(p)
		if err != nil {
			t.Fatal(err)
		}
		return "a link to " + to
	case fi.IsDir():
		return "a directory"
	}
	raw, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return "a file of " + string(raw)
}

// TestAReceiptNamingTheBriefLeavesItForTheRelaunchedImplementer: a receipt naming
// the lane's brief as its report is consumed, the brief stays, and the implementer
// started again still has it to read.
func TestAReceiptNamingTheBriefLeavesItForTheRelaunchedImplementer(t *testing.T) {
	root, id, l, rs, env, _ := refusedForTheWork(t, func(rc *LaneReceipt) { rc.Report = BriefFileName })
	brief := filepath.Join(root, filepath.FromSlash(RunRelDir), id, l.ID, BriefFileName)
	want, err := os.ReadFile(brief)
	if err != nil {
		t.Fatal(err)
	}
	o := Options{driverIsSelf: true}
	for range 2 {
		if _, err := Drive(context.Background(), root, id, DefaultStages(), o, rs); err != nil {
			t.Fatal(err)
		}
	}
	if n := env.launches(t, "claude"); n != 2 {
		t.Fatalf("the implementer is started again: %d launches", n)
	}
	if got, err := os.ReadFile(brief); err != nil || string(got) != string(want) {
		t.Fatalf("the relaunched implementer still has its brief: %v", err)
	}
}

// asIssueRun keys the run to an issue, so the drain's count of refused steps
// is kept on its lane (noteRefusal).
func asIssueRun(t *testing.T, root, id string) {
	t.Helper()
	if err := mutate(root, id, func(_ *os.Root, st *State) (bool, error) {
		st.Key = secondIssue
		return true, nil
	}); err != nil {
		t.Fatal(err)
	}
}

// roundConsumed stands rc at receipt, a fix or sync round's await, under a claim
// that has lapsed, takes one drive's step, and wants only the receipt set aside.
// Each file at the names the loop assigned the implement stage, directly in the
// lane's directory, and each in the round's own directory, is left as it was;
// the record's line says only the receipt went at this round and never that the
// receipt named the wrong file; and the lane's implement-stage report is still
// read as the next round reads it.
func roundConsumed(t *testing.T, root, id string, stages Stages, o Options, laneID, receipt string, rc LaneReceipt) {
	t.Helper()
	laneDir := filepath.Join(root, filepath.FromSlash(RunRelDir), id, laneID)
	roundDir := filepath.Dir(absIn(root, receipt))
	before := map[string]string{}
	for _, p := range []string{
		filepath.Join(laneDir, ReportFileName), filepath.Join(laneDir, DoDFileName),
		filepath.Join(roundDir, ReportFileName), filepath.Join(roundDir, DoDFileName),
	} {
		if raw, err := os.ReadFile(p); err == nil {
			before[p] = string(raw)
		}
	}
	if _, ok := before[filepath.Join(laneDir, ReportFileName)]; !ok {
		t.Fatal("the lane's implement-stage report stands before the step")
	}
	claimAwait(t, root, id, receipt, o.now().Add(-time.Minute))
	data, err := json.Marshal(rc)
	if err != nil {
		t.Fatal(err)
	}
	standAt(t, root, receipt, string(data))
	o.driverIsSelf = true
	if _, err := advance(root, id, stages, o); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(absIn(root, receipt)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("the round's receipt is set aside: %v", err)
	}
	for p, want := range before {
		if got, err := os.ReadFile(p); err != nil || string(got) != want {
			t.Fatalf("%s is left where it is: %v", p, err)
		}
	}
	if !recordSays(t, root, id, StageRunner, "set aside at", "only the receipt is set aside at this round") {
		t.Fatal("the record says only the receipt went aside at this round")
	}
	if recordSays(t, root, id, StageRunner, "the name the loop assigned") || recordSays(t, root, id, StageRunner, "left in place:") {
		t.Fatal("the record does not say the receipt named the wrong file")
	}
	st, err := ReadState(root, id)
	if err != nil {
		t.Fatal(err)
	}
	for _, l := range st.Lanes {
		if l.ID == laneID {
			if err := reportsCarryNoVerdict(Context{RepoRoot: root, State: st}, l); err != nil {
				t.Fatalf("the next round still reads the lane's report: %v", err)
			}
			return
		}
	}
	t.Fatalf("no lane %s", laneID)
}

// TestAFixRoundsConsumedReceiptMovesNothingButItself: at a fix round's await, a
// receipt refused for the work is set aside alone. One naming report.md, the
// implement stage's own name, leaves the lane's report for the next round; one
// naming its round's report, as the fix brief says, leaves that too, with a line
// that is true of it.
func TestAFixRoundsConsumedReceiptMovesNothingButItself(t *testing.T) {
	for name, inRound := range map[string]bool{"naming report.md": false, "naming its round's report": true} {
		t.Run(name, func(t *testing.T) {
			repo, id, stages := reviewedRun(t)
			l := currentLane(t, repo, id)
			handBack(t, repo, id, stages, RoleRuthless, reviewerReturn("FIX FIRST"))
			passRound(t, repo, id, stages, RoleSecurity, RoleAuditor)
			res, err := advance(repo.Root(), id, stages, Options{})
			if err != nil || res.Awaiting == nil || res.Awaiting.Role != RoleImplementer {
				t.Fatalf("the fix round's implementer is handed out: %+v %v", res, err)
			}
			laneDir := filepath.Join(repo.Root(), filepath.FromSlash(RunRelDir), id, l.ID)
			prefix := ""
			if inRound {
				rel, err := filepath.Rel(laneDir, filepath.Dir(absIn(repo.Root(), res.Awaiting.Receipt)))
				if err != nil {
					t.Fatal(err)
				}
				prefix = filepath.ToSlash(rel) + "/"
				laneFileAt(t, laneDir, prefix+ReportFileName, "fixed it\n")
				laneFileAt(t, laneDir, prefix+DoDFileName, "failed\n")
			}
			one := 1
			rc := LaneReceipt{SchemaVersion: ReceiptSchemaVersion, RunID: id, Lane: l.ID, Branch: l.Branch,
				Commits:          []string{laneCommit(t, repo, l, "fix.txt")},
				DefinitionOfDone: &DoDRun{Command: "make check", ExitCode: &one, Output: prefix + DoDFileName},
				Report:           prefix + ReportFileName, Model: "a-model"}
			roundConsumed(t, repo.Root(), id, stages, Options{}, l.ID, res.Awaiting.Receipt, rc)
		})
	}
}

// TestASyncRoundsConsumedReceiptMovesNothingButItself: the same at a sync
// round's await, for a receipt naming report.md.
func TestASyncRoundsConsumedReceiptMovesNothingButItself(t *testing.T) {
	f := newParFixture(t, "1. One\n2. Two\n   - needs: none\n", Options{SubAgents: strp("4")})
	f.bothImplementersOut(t)
	f.implement(t, "lane-1", "same.txt")
	l2 := f.lane(t, "lane-2")
	if err := os.WriteFile(filepath.Join(l2.Worktree, "same.txt"), []byte("lane two's words\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	f.repo.Git("-C", l2.Worktree, "add", "--", "same.txt")
	f.repo.Git("-C", l2.Worktree, "commit", "-q", "-m", "lane two")
	f.receipt(t, "lane-2", f.await(t, "lane-2", RoleImplementer), strings.TrimSpace(f.repo.Git("-C", l2.Worktree, "rev-parse", "HEAD")))
	f.roundPassed(t, "lane-1")
	f.roundPassed(t, "lane-2")
	f.landed(t, "lane-1")
	f.step(t)
	res := f.step(t)
	l2 = f.lane(t, "lane-2")
	if res.Awaiting == nil || res.Awaiting.Role != RoleImplementer || len(l2.Syncs) != 1 || res.Awaiting.Receipt != l2.Syncs[0].Receipt {
		t.Fatalf("a fresh implementer is handed the sync brief: %+v", res)
	}
	one := 1
	rc := LaneReceipt{SchemaVersion: ReceiptSchemaVersion, RunID: f.runID, Lane: "lane-2", Branch: l2.Branch,
		Commits:          []string{laneCommit(t, f.repo, l2, "other.txt")},
		DefinitionOfDone: &DoDRun{Command: "make check", ExitCode: &one, Output: DoDFileName},
		Report:           ReportFileName, Model: "a-model"}
	roundConsumed(t, f.repo.Root(), f.runID, f.stages, f.opts(), "lane-2", res.Awaiting.Receipt, rc)
}

// standingUnderALapsedClaim stands body at the lane's receipt path under a
// runner's claim that has lapsed, as a drive killed during the launch leaves it.
func standingUnderALapsedClaim(t *testing.T, root, id string, l Lane, body string) {
	t.Helper()
	claimAwait(t, root, id, l.awaiting().Receipt, time.Now().Add(-time.Minute))
	standAt(t, root, l.awaiting().Receipt, body)
}

// unreadableBounds is roots whose repository layer sets a stuck bound out of
// range, so the drain's stuck bounds cannot be read.
func unreadableBounds(t *testing.T) *layered.Roots {
	t.Helper()
	repo := t.TempDir()
	laneFileAt(t, repo, ".abcd/config.json", `{"pace": {"stuck_refusals": 0}}`)
	if _, err := resolveStuckBounds(layered.Roots{Repo: repo, Home: t.TempDir()}); err == nil {
		t.Fatal("the bounds cannot be read")
	}
	return &layered.Roots{Repo: repo, Home: t.TempDir()}
}

// TestAStandingReceiptBelowTheBoundIsFollowedByOneRelaunch: on an issue run with
// no count yet, a receipt refused for the work standing under a lapsed claim is
// consumed, and the implementer is started again in the same step, once; the
// lane is not held. With bounds that cannot be read, no launch follows on an
// issue run, which counts as at the bound, while an intent's run, which keeps no
// count, still starts the implementer again.
func TestAStandingReceiptBelowTheBoundIsFollowedByOneRelaunch(t *testing.T) {
	for _, tc := range []struct {
		name     string
		issue    bool
		unread   bool
		launches int
		held     bool
	}{
		{"an issue run below the bound", true, false, 1, false},
		{"an issue run whose bounds cannot be read", true, true, 0, true},
		{"an intent's run whose bounds cannot be read", false, true, 1, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root, id, l, rs, env, body := refusedForTheWork(t, nil)
			if tc.issue {
				asIssueRun(t, root, id)
			}
			standingUnderALapsedClaim(t, root, id, l, body)
			o := Options{driverIsSelf: true}
			if tc.unread {
				o.Roots = unreadableBounds(t)
			}
			res, err := Drive(context.Background(), root, id, DefaultStages(), o, rs)
			if err != nil {
				t.Fatal(err)
			}
			if !asideHolds(t, root, id, l.ID, body) {
				t.Fatal("the standing receipt is consumed")
			}
			if n := env.launches(t, "claude"); n != tc.launches {
				t.Fatalf("%d launches, want %d: %s", n, tc.launches, res.Next)
			}
			if held := strings.Contains(res.Next, "is not started again"); held != tc.held {
				t.Fatalf("held %v, want %v: %s", held, tc.held, res.Next)
			}
			if tc.issue && !tc.unread {
				if n := refusedCount(t, root, id); n != 2 {
					t.Fatalf("the standing receipt and the relaunch's refused hand-in are counted: %d", n)
				}
			}
		})
	}
}

// TestADestinationThatCannotBeCheckedIsSaidSo: a file the loop assigned is left
// when what is set aside at its destination cannot be checked, and the line says
// that, not that a file of its name is already there.
func TestADestinationThatCannotBeCheckedIsSaidSo(t *testing.T) {
	repo := t.TempDir()
	laneDir := "run/lane-1"
	laneFileAt(t, repo, laneDir+"/"+ReportFileName, "built it\n")
	// The destination's directory is a file, so its entry cannot be looked up.
	laneFileAt(t, repo, laneDir+"/"+AsideDirName+"/one", "not a directory\n")
	root, err := os.OpenRoot(repo)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	moved, kept := moveNamed(root, laneDir, laneDir+"/"+AsideDirName+"/one",
		[]laneNamed{{what: "the report", rel: ReportFileName, assigned: ReportFileName}})
	if len(moved) != 0 || len(kept) != 1 {
		t.Fatalf("the report is left: moved %v, kept %v", moved, kept)
	}
	if strings.Contains(kept[0], "already set aside") || !strings.Contains(kept[0], "cannot be checked") {
		t.Fatalf("the line says the destination cannot be checked: %s", kept[0])
	}
	if raw, err := os.ReadFile(filepath.Join(repo, laneDir, ReportFileName)); err != nil || string(raw) != "built it\n" {
		t.Fatalf("the report stands: %v", err)
	}
}
