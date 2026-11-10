package loop

// terminal_receipt_test.go proves iss-2611071242184002 in the drive: a lane whose
// implementer's receipts are refused for the work is not held on the outage's
// wait. Each refused receipt is set aside and the role started again, and the
// lane goes back at the bound of failed launches or the drain's stuck bounds,
// with no wait that has no deadline and no turn between the drive and the drain
// that has no wait.

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/core/layered"
)

// TestADriveHandsBackALaneWhoseReceiptsAreRefusedForTheWork: every receipt the
// implementer writes is refused for the work. The drive starts it three times,
// keeps each receipt aside, and the lane goes back with the reason; the run goes
// on to its cap. With no bound the drain's stuck bounds hand it back; under a
// bound the third failed launch does.
func TestADriveHandsBackALaneWhoseReceiptsAreRefusedForTheWork(t *testing.T) {
	for _, bound := range []bool{false, true} {
		name := "with no bound"
		if bound {
			name = "under a bound"
		}
		t.Run(name, func(t *testing.T) {
			w := newTerminalWorld(t)
			for i := range w.stages {
				if w.stages[i].Name == StageImplement {
					w.stages[i].Verify = func(c Context, l *Lane, receipt string) error {
						r := refuse("receipt", "", l.ID, "the receipt is missing a passing definition of done (it exited 1)", "finish the lane")
						r.work = true
						return r
					}
				}
			}
			o := w.options(t)
			o.Drain.Max = 1
			if bound {
				o.Until = w.clock.now().Add(72 * time.Hour)
			}
			moves := 0
			o.Moved = func() {
				if moves++; moves > 20 {
					t.Fatalf("the drive and the drain turn with no end: %d moves\n%s", moves, strings.Join(w.lines, "\n"))
				}
			}
			sleep := o.Sleep
			o.Sleep = func(ctx context.Context, d time.Duration) {
				if len(w.sleeps) > 100 {
					t.Fatalf("the lane waits with no deadline: %v\n%s", w.sleeps[:8], strings.Join(w.lines, "\n"))
				}
				sleep(ctx, d)
			}
			res, err := RunTerminal(context.Background(), w.repo.Root(), o)
			if err != nil {
				t.Fatal(err)
			}
			if res.Ended != DrainStoppedCap {
				t.Fatalf("the lane goes back and the run reaches its cap: %+v\n%s", res, strings.Join(w.lines, "\n"))
			}
			// The judge, and the implementer three times.
			if n := w.launches(t); n != 4 {
				t.Fatalf("the implementer is started three times: %d launches\n%s", n, strings.Join(w.lines, "\n"))
			}
			st, _, err := readDrain(w.repo.Root())
			if err != nil {
				t.Fatal(err)
			}
			if len(st.HandBacks) != 1 || st.HandBacks[0].Kind != claim.HandBackStuck {
				t.Fatalf("the lane is handed back as stuck: %+v", st.HandBacks)
			}
			why := "refused the same way 3 times"
			if bound {
				why = "failed 3 times in a row"
			}
			if !strings.Contains(st.HandBacks[0].Reason, why) {
				t.Fatalf("the hand-back says why (%q): %s", why, st.HandBacks[0].Reason)
			}
			kept := 0
			_ = filepath.WalkDir(filepath.Join(w.repo.Root(), filepath.FromSlash(RunRelDir), st.Lanes[0].RunID), func(p string, d os.DirEntry, err error) error {
				if err == nil && !d.IsDir() && strings.Contains(filepath.ToSlash(p), "/"+AsideDirName+"/") {
					kept++
				}
				return nil
			})
			if kept != 3 {
				t.Fatalf("each refused receipt is kept: %d", kept)
			}
		})
	}
}

// TestADriveHandsBackALaneHeldOnAReceiptItCannotConsume: a receipt refused for
// something other than the work stays at its path, and no runner is started over
// it; each step that reads it counts, and under a bound, where the outage's wait
// has no end, the drain's stuck bounds reach the lane the drive holds and hand it
// back.
func TestADriveHandsBackALaneHeldOnAReceiptItCannotConsume(t *testing.T) {
	w := newTerminalWorld(t)
	for i := range w.stages {
		if w.stages[i].Name == StageImplement {
			w.stages[i].Verify = func(c Context, l *Lane, receipt string) error {
				return refuse("receipt", "", l.ID, "the receipt names another run", "write this run's")
			}
		}
	}
	o := w.options(t)
	o.Drain.Max = 1
	o.Until = w.clock.now().Add(72 * time.Hour)
	sleep := o.Sleep
	o.Sleep = func(ctx context.Context, d time.Duration) {
		if len(w.sleeps) > 100 {
			t.Fatalf("the lane waits with no deadline: %v\n%s", w.sleeps[:8], strings.Join(w.lines, "\n"))
		}
		sleep(ctx, d)
	}
	res, err := RunTerminal(context.Background(), w.repo.Root(), o)
	if err != nil {
		t.Fatal(err)
	}
	if res.Ended != DrainStoppedCap {
		t.Fatalf("the lane goes back and the run reaches its cap: %+v\n%s", res, strings.Join(w.lines, "\n"))
	}
	if n := w.launches(t); n != 2 {
		t.Fatalf("the judge and one implementer, none over the receipt: %d launches", n)
	}
	st, _, err := readDrain(w.repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	if len(st.HandBacks) != 1 || st.HandBacks[0].Kind != claim.HandBackStuck || !strings.Contains(st.HandBacks[0].Reason, "names another run") {
		t.Fatalf("the lane is handed back as stuck, naming the refusal: %+v", st.HandBacks)
	}
}

// TestAStuckLaneGivenBackTwiceWaitsAtTheFloor: the drive gives a stuck lane back
// to the next move at once; given the same lane again with no other between, the
// move having not handed it back, it waits at the floor before it gives it back,
// so the two never turn with no wait. A lane that is not stuck is not given back.
func TestAStuckLaneGivenBackTwiceWaitsAtTheFloor(t *testing.T) {
	c := &clock{t: time.Date(2026, 11, 10, 9, 0, 0, 0, time.UTC)}
	var sleeps []time.Duration
	tm := newTerminal(t.TempDir(), TerminalOptions{
		Options: Options{Now: c.now, Roots: &layered.Roots{Repo: t.TempDir(), Home: t.TempDir()}},
		Sleep:   func(_ context.Context, d time.Duration) { sleeps = append(sleeps, d); c.t = c.t.Add(d) },
	})
	run := State{Key: secondIssue, Lanes: []Lane{{ID: "lane-1", Stage: StageImplement,
		Refused: &LaneRefused{Stage: "receipt", Reason: "refused", Count: BundledStuckRefusals, Since: c.now(), Last: c.now()}}}}
	if back, ended := tm.stuckBack(context.Background(), "run-1", run); !back || ended != "" || len(sleeps) != 0 {
		t.Fatalf("a stuck lane is given back at once: %v %q %v", back, ended, sleeps)
	}
	if back, ended := tm.stuckBack(context.Background(), "run-1", run); !back || ended != "" || len(sleeps) != 1 || sleeps[0] != TerminalWaitFloor {
		t.Fatalf("the same lane given back again waits at the floor first: %v %q %v", back, ended, sleeps)
	}
	moving := run
	moving.Lanes = []Lane{{ID: "lane-1", Stage: StageImplement}}
	if back, _ := tm.stuckBack(context.Background(), "run-1", moving); back {
		t.Fatal("a lane that is not stuck is driven")
	}
	if back, _ := tm.stuckBack(context.Background(), "run-1", run); !back || len(sleeps) != 1 {
		t.Fatalf("once a lane moved, a stuck one is given back at once again: %v", sleeps)
	}
}
