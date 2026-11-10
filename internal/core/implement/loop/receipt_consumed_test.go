package loop

// receipt_consumed_test.go proves iss-2611071242184002 in the step: where a drive
// is its own driver, an implementer's receipt refused for the work is set aside,
// kept, and the role is started again, so no step reads the same refused receipt
// twice with no launch between. A receipt refused for anything else, and every
// host-stepped run, keep the stop receipt_standing.go describes.

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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
		if !recordSays(t, root, id, StageRunner, "refused for the work", "it exited 1", "set aside at", "started again") {
			t.Fatal("the record says the receipt was set aside, and with what refusal")
		}
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
		"another run's":  func(rc *LaneReceipt) { rc.RunID = "run-2609300000000000" },
		"another lane's": func(rc *LaneReceipt) { rc.Lane = "lane-9" },
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
	repo, id, l, _, stages, rs, env, _ := stalledOnModel(t)
	root := repo.Root()
	body, err := os.ReadFile(absIn(root, l.awaiting().Receipt))
	if err != nil {
		t.Fatal(err)
	}
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
// ever.
func TestASetAsideThatFailsKeepsThePause(t *testing.T) {
	root, id, l, rs, env, body := refusedForTheWork(t, nil)
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
	if n := env.launches(t, "claude"); n != 1 {
		t.Fatalf("one launch: %d", n)
	}
}
