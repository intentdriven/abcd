package loop

import (
	"bytes"
	"context"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/core/runstop"
)

// Every route by which a drive writes for a run leaves a lane the drive passed over alone
// for the rest of the drive (iss-2611120957377169): the move's new pace, its halt under a
// person's stop, its finish of a merged lane and its renewal; the waits' renewal of a run
// in progress and of a lane parked at its pull request; the halt and the signal's halt at
// the drive's end; and the let-go at its end. Each route runs twice over one fixture: with
// the lane passed over nothing of it is written, and without, the route writes for it, so
// each check stands on a route that would.

// runFiles is what a route could write for a run: its state, its control record, and its
// reservation's tip on origin ("" with no origin, or none there).
type runFiles struct {
	state, control []byte
	tip            string
}

// filesOf reads runFiles for the run of runID, whose issue is key, in root, origin at
// bare ("" for none).
func filesOf(t *testing.T, root, bare, runID, key string) runFiles {
	t.Helper()
	f := runFiles{state: readStateBytes(t, root, runID), control: controlBytes(t, root, runID)}
	if bare != "" {
		f.tip = originReservation(t, bare, claim.RefPrefix+key).commit
	}
	return f
}

// same reports whether nothing of f changed in g, and what did.
func (f runFiles) same(g runFiles) (bool, string) {
	switch {
	case !bytes.Equal(f.state, g.state):
		return false, "its state:\n" + string(g.state)
	case !bytes.Equal(f.control, g.control):
		return false, "its control record:\n" + string(g.control)
	case f.tip != g.tip:
		return false, "its reservation on origin: " + f.tip + " to " + g.tip
	}
	return true, ""
}

// passedRoute is one write route over one fixture: set builds the fixture and returns the
// run's id and issue, the checkout and origin ("" for none); act takes the route with the
// lanes passed over that it is given.
type passedRoute struct {
	name string
	set  func(t *testing.T) (root, bare, runID, key string, act func(t *testing.T, passed []string))
}

// TestEveryWriteRouteOfADriveLeavesALaneItPassedOver is the invariant, route by route.
func TestEveryWriteRouteOfADriveLeavesALaneItPassedOver(t *testing.T) {
	for _, rt := range passedRoutes() {
		t.Run(rt.name, func(t *testing.T) {
			for _, passed := range []bool{false, true} {
				name := "not passed over"
				if passed {
					name = "passed over"
				}
				t.Run(name, func(t *testing.T) {
					root, bare, runID, key, act := rt.set(t)
					before := filesOf(t, root, bare, runID, key)
					var given []string
					if passed {
						given = []string{runID}
					}

					act(t, given)

					same, what := before.same(filesOf(t, root, bare, runID, key))
					if passed && !same {
						t.Fatalf("%s wrote for the lane passed over, %s", rt.name, what)
					}
					if !passed && same {
						t.Fatalf("premise: %s writes for the lane when it is not passed over", rt.name)
					}
				})
			}
		})
	}
}

// passedRoutes are the routes TestEveryWriteRouteOfADriveLeavesALaneItPassedOver takes.
func passedRoutes() []passedRoute {
	// inProgress is the drain's lane in progress, opened by one move.
	inProgress := func(t *testing.T) (string, string, string) {
		repo := drainRepo(t)
		first := drainedToLaneOnce(t, repo, DrainOptions{})
		if first.Start == nil {
			t.Fatalf("premise: the move opens a lane: %+v", first)
		}
		return repo.Root(), first.Start.RunID, first.Lane.Issue
	}
	drive := func(root string, o Options, passed []string) *terminal {
		return newTerminal(root, TerminalOptions{Options: o, Drain: DrainOptions{PassedLanes: passed}})
	}
	return []passedRoute{
		{"the move's new pace", func(t *testing.T) (string, string, string, string, func(*testing.T, []string)) {
			root, runID, key := inProgress(t)
			return root, "", runID, key, func(t *testing.T, passed []string) {
				if _, err := Drain(root, Options{Pace: strp("60/15"), SubAgents: strp("3")}, DrainOptions{PassedLanes: passed}); err != nil {
					t.Fatal(err)
				}
			}
		}},
		{"the move under a person's stop", func(t *testing.T) (string, string, string, string, func(*testing.T, []string)) {
			root, runID, key := inProgress(t)
			o, _, _ := stopped(t)
			return root, "", runID, key, func(t *testing.T, passed []string) {
				_, err := Drain(root, o, DrainOptions{PassedLanes: passed})
				r := wantStopRefusal(t, err)
				if len(passed) > 0 && !strings.Contains(r.Wrote, "run "+runID+", which this drive passed over") {
					t.Fatalf("the refusal does not name the lane passed over among those left: %q", r.Wrote)
				}
			}
		}},
		{"the move's finish of a merged lane", func(t *testing.T) (string, string, string, string, func(*testing.T, []string)) {
			f := drainLandFixture(t)
			first := f.firstMove(t)
			if first.Start == nil || first.Lane.Issue != secondIssue {
				t.Fatalf("premise: the drain opens the documentation issue's lane: %+v", first)
			}
			f.runID, f.issue = first.Start.RunID, secondIssue
			l := f.armIssueLaneOnce(t)
			pauseBoth(t, f)
			f.merged(t, l.Landing.Pushed)
			return f.repo.Root(), f.bare, f.runID, f.issue, func(t *testing.T, passed []string) {
				res, err := Drain(f.repo.Root(), Options{}, DrainOptions{PassedLanes: passed})
				if err != nil {
					t.Fatal(err)
				}
				if done := slices.ContainsFunc(res.Finished, func(d DrainFinished) bool { return d.RunID == f.runID }); done == (len(passed) > 0) {
					t.Fatalf("the merged lane is finished only when it is not passed over: %+v", res.Finished)
				}
			}
		}},
		{"the move's renewal of its lane in progress", func(t *testing.T) (string, string, string, string, func(*testing.T, []string)) {
			t0 := time.Date(2026, 10, 17, 9, 0, 0, 0, time.UTC)
			c := &clock{t: t0}
			o := Options{Now: c.now}
			repo, bare := reservingDrainRepo(t, "commands/drain.md")
			first := drainToLane(t, repo, o, DrainOptions{})
			if first.Start == nil {
				t.Fatalf("premise: the move opens a lane: %+v", first)
			}
			pauseDrain(t, repo.Root(), t0.Add(4*time.Hour))
			c.t = t0.Add(61 * time.Minute)
			return repo.Root(), bare, first.Start.RunID, first.Lane.Issue, func(t *testing.T, passed []string) {
				if _, err := Drain(repo.Root(), o, DrainOptions{PassedLanes: passed}); err != nil {
					t.Fatal(err)
				}
			}
		}},
		{"the wait's renewal of a lane another session reserved again", func(t *testing.T) (string, string, string, string, func(*testing.T, []string)) {
			root, bare, runID, key, c := reservedAgainAfterThePass(t)
			return root, bare, runID, key, func(t *testing.T, passed []string) {
				drive(root, Options{Now: c.now}, passed).renewHeld()
			}
		}},
		{"the wait's renewal of a lane parked at its pull request", func(t *testing.T) (string, string, string, string, func(*testing.T, []string)) {
			f, l := parkDrainLane(t)
			f.listPRs(t, stubPR{Number: 7, HeadRefName: l.Branch, State: "OPEN", Mergeable: "MERGEABLE", MergeStateStatus: "BLOCKED"})
			at := originReservation(t, f.bare, claim.RefPrefix+secondIssue).claimed.Add(12*time.Hour + 30*time.Minute)
			return f.repo.Root(), f.bare, f.runID, secondIssue, func(t *testing.T, passed []string) {
				drive(f.repo.Root(), Options{Now: func() time.Time { return at }}, passed).renewHeld()
			}
		}},
		{"the halt at the drive's end", func(t *testing.T) (string, string, string, string, func(*testing.T, []string)) {
			root, runID, key := inProgress(t)
			o, _, _ := stopped(t)
			return root, "", runID, key, func(t *testing.T, passed []string) {
				drive(root, o, passed).settleHalted()
			}
		}},
		{"the signal's halt at the drive's end", func(t *testing.T) (string, string, string, string, func(*testing.T, []string)) {
			root, runID, key := inProgress(t)
			home, _ := stopWorld(t, false)
			return root, "", runID, key, func(t *testing.T, passed []string) {
				drive(root, Options{stopHome: home}, passed).haltOnSignal()
			}
		}},
		{"the let-go at the drive's end", func(t *testing.T) (string, string, string, string, func(*testing.T, []string)) {
			root, runID, key := inProgress(t)
			editState(t, root, runID, func(doc map[string]any) {
				if doc["reserved_at"] == nil {
					doc["reserved_at"] = time.Now().UTC().Format(time.RFC3339)
				}
			})
			return root, "", runID, key, func(t *testing.T, passed []string) {
				drive(root, Options{}, passed).letGo()
			}
		}},
	}
}

// pauseDrain closes the checkout's drain window until until, so a move waits on it.
func pauseDrain(t *testing.T, root string, until time.Time) {
	t.Helper()
	st, live, err := readDrain(root)
	if err != nil || !live {
		t.Fatalf("premise: a live drain: %v", err)
	}
	st.NextEligibleAt = &until
	if err := writeDrain(root, st); err != nil {
		t.Fatal(err)
	}
}

// reservedAgainAfterThePass is a drain's lane let go as a drive's take-up leaves it before
// it passes the lane over, and then reserved again by another session in the checkout, as
// `abcd build` over the run does, the clock then past half the new lease.
func reservedAgainAfterThePass(t *testing.T) (root, bare, runID, key string, c *clock) {
	t.Helper()
	t0 := time.Date(2026, 10, 17, 9, 0, 0, 0, time.UTC)
	c = &clock{t: t0}
	o := Options{Now: c.now}
	repo, bare := reservingDrainRepo(t, "commands/drain.md")
	first := drainToLane(t, repo, o, DrainOptions{})
	if first.Start == nil {
		t.Fatalf("premise: the move opens a lane: %+v", first)
	}
	root, runID, key = repo.Root(), first.Start.RunID, first.Lane.Issue
	c.t = t0.Add(time.Minute)
	now := c.now()
	clause, failed := releaseReservation(root, key, "", now)
	if err := mutateReservation(root, runID, func(st *State) (bool, error) {
		noteRecordReleased(root, st, failed, now)
		st.Record = append(st.Record, Entry{At: now, Stage: StageDriveReleased, Note: clause})
		st.UpdatedAt = now
		return true, nil
	}); err != nil {
		t.Fatal(err)
	}
	c.t = t0.Add(2 * time.Minute)
	if err := reserveResumed(root, &StartResult{RunID: runID}, o); err != nil {
		t.Fatalf("premise: another session reserves the lane again: %v", err)
	}
	run, err := ReadState(root, runID)
	if err != nil {
		t.Fatal(err)
	}
	if letGoByDrive(run) || run.ReservedAt == nil {
		t.Fatalf("premise: the run holds its reservation again: %+v", run.Record)
	}
	c.t = t0.Add(2*time.Minute + 61*time.Minute)
	return root, bare, runID, key, c
}

// TestADrivesTakeOverOfARunItMayNotSavePassesTheLaneOver: a lane this drive's own signal
// halted is taken over, not taken up, as the drive comes back to it. Where its file states
// another version the drive may not save, the take-over is refused at the whole save, and
// the drive passes the lane over, says so once and waits on nothing, as the take-up does.
// After the pass, the drive's waits, its signal's halt and its let-go write nothing of the
// lane. Of the three, only the let-go can fail here: the run, taken over and not let go by a
// drive, names a reservation, so without the pass the let-go would add its line. The waits
// have no origin here to read a view from, and the judge of the whole save refuses the halt
// with or without the pass, so the paired runs of
// TestEveryWriteRouteOfADriveLeavesALaneItPassedOver hold those two.
func TestADrivesTakeOverOfARunItMayNotSavePassesTheLaneOver(t *testing.T) {
	repo := drainRepo(t)
	first := drainedToLaneOnce(t, repo, DrainOptions{})
	if first.Start == nil {
		t.Fatalf("premise: the move opens a lane: %+v", first)
	}
	root, runID := repo.Root(), first.Start.RunID
	home, _ := stopWorld(t, false)
	o := Options{stopHome: home}
	if _, err := haltOnce(root, runID, runstop.Stop{SetBy: "a signal to the drive", SetAt: time.Now()}, o); err != nil {
		t.Fatalf("premise: the drive's signal halts the run: %v", err)
	}
	if c := controlOf(t, root, runID); c.Stop == nil || c.Stop.HaltedAt == nil || c.Stop.FromFile {
		t.Fatalf("premise: the run is halted by the drive's own signal: %+v", c.Stop)
	}
	// The drive cannot find its own session, so it may not save a file at another version.
	procsOf(t, hostOne, hostTwo)
	stateAtVersion(t, root, runID, schemaVersionPrevious)
	var said []string
	slept := 0
	tm := newTerminal(root, TerminalOptions{Options: o, Stages: DefaultStages(), Say: func(s string) { said = append(said, s) },
		Sleep: func(context.Context, time.Duration) { slept++ }})

	done, ended := tm.takeUp(context.Background(), runID)

	if !done || ended != "" || slept != 0 {
		t.Fatalf("the drive does not pass the lane over and go on: %v %q, %d wait(s)\n%s", done, ended, slept, strings.Join(said, "\n"))
	}
	if !slices.Equal(tm.o.Drain.PassedLanes, []string{runID}) {
		t.Fatalf("the moves after are not given the lane to leave: %v\n%s", tm.o.Drain.PassedLanes, strings.Join(said, "\n"))
	}
	if run, err := ReadState(root, runID); err != nil || letGoByDrive(run) {
		t.Fatalf("premise: the lane taken over is not one a drive let go: %v", err)
	}
	n := 0
	for _, s := range said {
		if strings.Contains(s, "passed over") && strings.Contains(s, runID) {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("the drive says once that it passed the lane over:\n%s", strings.Join(said, "\n"))
	}
	before := filesOf(t, root, "", runID, "")
	tm.renewHeld()
	tm.haltOnSignal()
	tm.letGo()
	if same, what := before.same(filesOf(t, root, "", runID, "")); !same {
		t.Fatalf("the drive wrote for the lane it passed over, %s", what)
	}
}
