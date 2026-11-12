package loop

import (
	"bytes"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/runstop"
)

// A save of a run's whole state takes the file to this abcd's version, so a session
// that is not the run's driver leaves a run whose file states another version as that
// driver wrote it (iss-2611110933196955). The run's driver is the live session its
// control record names, which only a claim records: a step claims, and a drain claims
// one of its own lanes before it saves one whose file states another version. Each
// route below is driven over a file on disk, and each leaves the file's bytes as they
// were and says what it left and why.

// claimedBy records driver as the run's live driver, as a step's claim records it, and
// leaves here as this session, with alive the hosts the process table holds.
func claimedBy(t *testing.T, repoRoot, runID string, o Options, driver, here int, alive ...int) {
	t.Helper()
	procsOf(t, driver, driver)
	if err := ClaimDriver(repoRoot, runID, o); err != nil {
		t.Fatal(err)
	}
	if c := controlOf(t, repoRoot, runID); c.Driver == nil || c.Driver.PID != driver {
		t.Fatalf("premise: the run's driver is %d: %+v", driver, c.Driver)
	}
	procsOf(t, here, alive...)
}

// leftLine is what a route says of a run it leaves at version.
func leftLine(runID string, version int) string {
	return fmt.Sprintf("%s states schema version %d and this abcd writes %d", StateRelPath(runID), version, SchemaVersion)
}

// wantLeft fails unless said names the run left at version, and its file is still
// before.
func wantLeft(t *testing.T, repoRoot, runID string, version int, before []byte, said string) {
	t.Helper()
	want := leftLine(runID, version)
	if version > SchemaVersion {
		want = StateRelPath(runID) + " is schema version " + strconv.Itoa(version) + ", which a newer abcd wrote"
	}
	if !strings.Contains(said, want) {
		t.Fatalf("the route does not say %q of the run it left:\n%s", want, said)
	}
	after, err := os.ReadFile(runStatePath(repoRoot, runID))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatalf("the route rewrote a run whose file states version %d:\n%s", version, after)
	}
}

// readAsParked makes one drain move over the parked lane l, its pull request open and
// waiting, so the drain's own state reads the lane as parked on its pull request
// rather than in progress, which the move would admit first.
func readAsParked(t *testing.T, f *landFixture, l Lane) {
	t.Helper()
	f.listPRs(t, stubPR{Number: 7, HeadRefName: l.Branch, State: "OPEN", Mergeable: "MERGEABLE", MergeStateStatus: "BLOCKED"})
	res, err := Drain(f.repo.Root(), Options{}, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	i := slices.IndexFunc(res.Lanes, func(dl DrainLane) bool { return dl.RunID == f.runID })
	if i < 0 || res.Lanes[i].Outcome != DrainLanePullRequest {
		t.Fatalf("premise: the drain reads its lane as parked on its pull request: %+v", res.Lanes)
	}
}

// stateVersionOf is the version the run's file states.
func stateVersionOf(t *testing.T, repoRoot, runID string) string {
	t.Helper()
	data, err := os.ReadFile(runStatePath(repoRoot, runID))
	if err != nil {
		t.Fatal(err)
	}
	return string(topLevelFields(t, data)["schema_version"])
}

// TestAStepUnderAStopLeavesAnotherVersionsRunToItsDriver: a step under a stop claims
// nothing, and its move's check settles the run only where this session is the run's
// driver or the file states this abcd's version. A run another live session drives, or
// whose driver is gone, is left as its file stands, the refusal at the stage `stop`
// naming the file and both versions; a newer abcd's file is refused by its read.
func TestAStepUnderAStopLeavesAnotherVersionsRunToItsDriver(t *testing.T) {
	for _, tc := range []struct {
		name          string
		version       int
		driver, here  int
		alive         []int
		halts         bool
		leftByTheRead bool
	}{
		{name: "another live session drives it, previous version", version: schemaVersionPrevious, driver: hostTwo, here: hostOne, alive: []int{hostOne, hostTwo}},
		{name: "another live session drives it, version 24", version: schemaVersionUnrenamed, driver: hostTwo, here: hostOne, alive: []int{hostOne, hostTwo}},
		{name: "its driver is gone", version: schemaVersionPrevious, driver: hostTwo, here: hostOne, alive: []int{hostOne}},
		{name: "a newer abcd wrote it", version: SchemaVersion + 1, driver: hostTwo, here: hostOne, alive: []int{hostOne, hostTwo}, leftByTheRead: true},
		{name: "this session drives it", version: schemaVersionPrevious, driver: hostOne, here: hostOne, alive: []int{hostOne, hostTwo}, halts: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo, runID, _, _, _ := deadLane(t)
			home, _ := stopWorld(t, true)
			o := Options{stopHome: home}
			claimedBy(t, repo.Root(), runID, o, tc.driver, tc.here, tc.alive...)
			if _, err := runstop.Set(home, time.Now()); err != nil {
				t.Fatal(err)
			}
			before := stateAtVersion(t, repo.Root(), runID, tc.version)

			// The step's own route: its claim, which claims nothing under a stop, then
			// its move's check.
			if err := ClaimDriver(repo.Root(), runID, o); err != nil {
				t.Fatalf("the claim under a stop: %v", err)
			}
			err := checkOnce(repo.Root(), runID, o)

			if tc.halts {
				wantStopRefusal(t, err)
				if c := controlOf(t, repo.Root(), runID); c.Stop == nil {
					t.Fatal("the run's own driver did not take the stop")
				}
				if v := stateVersionOf(t, repo.Root(), runID); v != strconv.Itoa(SchemaVersion) {
					t.Fatalf("the driver's settle saved version %s, not its own %d", v, SchemaVersion)
				}
				return
			}
			r, ok := AsRefusal(err)
			if !ok {
				t.Fatalf("want a refusal, got %v", err)
			}
			if !tc.leftByTheRead && (r.Stage != StageStop || !r.Contention) {
				t.Fatalf("the step's refusal is not the stop's, at exit 3: %+v", r)
			}
			wantLeft(t, repo.Root(), runID, tc.version, before, r.Error())
			if c := controlOf(t, repo.Root(), runID); c.Stop != nil {
				t.Fatalf("the step marked another driver's run: %+v", c.Stop)
			}
		})
	}
}

// TestADrainsStopHaltsItsOwnLaneOnlyAsItsDriver: a drain's move under a stop halts a lane
// of its own whose file states another version only where this session is the lane's
// driver; one another live session drives is left as its file stands, and the move's
// refusal names it.
func TestADrainsStopHaltsItsOwnLaneOnlyAsItsDriver(t *testing.T) {
	for _, tc := range []struct {
		name  string
		here  int
		halts bool
	}{
		{name: "another live session drives the lane", here: hostOne},
		{name: "this session drives the lane", here: hostTwo, halts: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo, runID, _, _, _ := deadLane(t)
			home, _ := stopWorld(t, true)
			o := Options{stopHome: home}
			claimedBy(t, repo.Root(), runID, o, hostTwo, tc.here, hostOne, hostTwo)
			if _, err := runstop.Set(home, time.Now()); err != nil {
				t.Fatal(err)
			}
			before := stateAtVersion(t, repo.Root(), runID, schemaVersionPrevious)
			st := DrainState{Lanes: []DrainLane{{Issue: secondIssue, RunID: runID, Outcome: DrainLaneInProgress}}}

			_, err := drainStop(repo.Root(), &st, o)
			r := wantStopRefusal(t, err)

			if halted := controlOf(t, repo.Root(), runID).Stop != nil; halted != tc.halts {
				t.Fatalf("the drain's stop halted its lane: %v, want %v", halted, tc.halts)
			}
			if tc.halts {
				return
			}
			if !strings.Contains(r.Wrote, "run "+runID) {
				t.Fatalf("the move's refusal does not name the lane it left: %q", r.Wrote)
			}
			wantLeft(t, repo.Root(), runID, schemaVersionPrevious, before, r.Wrote)
		})
	}
}

// TestADrainsMoveLeavesItsMarkedParkedLaneAnotherSessionDrives: a drain takes up the
// mark a lifted stop left on a parked lane of its own only as that lane's driver; while
// another live session drives it, the mark stays and the file with it, and the parked
// reading names why.
func TestADrainsMoveLeavesItsMarkedParkedLaneAnotherSessionDrives(t *testing.T) {
	f, l := parkDrainLane(t)
	readAsParked(t, f, l)
	home, _ := stopWorld(t, false)
	claimedBy(t, f.repo.Root(), f.runID, Options{stopHome: home}, hostTwo, hostOne, hostOne, hostTwo)
	markUnsettled(t, f.repo.Root(), f.runID)
	before := stateAtVersion(t, f.repo.Root(), f.runID, schemaVersionPrevious)

	res, err := Drain(f.repo.Root(), Options{stopHome: home}, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if c := controlOf(t, f.repo.Root(), f.runID); c.Stop == nil {
		t.Fatal("the drain's move took up the mark of a lane another live session drives")
	}
	p := parkedOf(t, res, f.runID)
	if p.State != ParkedStopHolds {
		t.Fatalf("the lane is not named as its mark holding it: %+v", p)
	}
	wantLeft(t, f.repo.Root(), f.runID, schemaVersionPrevious, before, p.Note)
}

// TestADrainsLandingStepLeavesAMergedRunToItsDriver: the landing's last step over a
// merged lane is taken only by the run's driver where its file states another version.
// A lane of the drain's own that another live session drives, and a person's own build
// whose driver is gone, are left as their files stand, named among what the move
// finished, worktree and all; a newer abcd's file is refused by its read. A lane of the
// drain's own whose driver is gone is claimed by the drain, with a line in its record,
// and finished, as a drain resumed by a newer abcd finishes its lanes.
func TestADrainsLandingStepLeavesAMergedRunToItsDriver(t *testing.T) {
	for _, tc := range []struct {
		name     string
		version  int
		own      bool
		alive    []int
		finishes bool
	}{
		{name: "its own lane another live session drives", version: schemaVersionPrevious, own: true, alive: []int{hostOne, hostTwo}},
		{name: "a person's own build whose driver is gone", version: schemaVersionPrevious, alive: []int{hostOne}},
		{name: "a newer abcd's file", version: SchemaVersion + 1, own: true, alive: []int{hostOne, hostTwo}},
		{name: "its own lane whose driver is gone", version: schemaVersionPrevious, own: true, alive: []int{hostOne}, finishes: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := drainLandFixture(t)
			if tc.own {
				first := f.firstMove(t)
				f.runID = first.Start.RunID
			} else {
				start, err := f.startOnce(t, secondIssue)
				if err != nil {
					t.Fatal(err)
				}
				f.runID = start.RunID
			}
			f.issue = secondIssue
			l := f.armIssueLaneOnce(t)
			f.merged(t, l.Landing.Pushed)
			home, _ := stopWorld(t, false)
			o := Options{stopHome: home}
			claimedBy(t, f.repo.Root(), f.runID, o, hostTwo, hostOne, tc.alive...)
			before := stateAtVersion(t, f.repo.Root(), f.runID, tc.version)

			res, err := Drain(f.repo.Root(), o, DrainOptions{})

			if tc.finishes {
				if err != nil {
					t.Fatal(err)
				}
				i := slices.IndexFunc(res.Finished, func(d DrainFinished) bool { return d.RunID == f.runID })
				if i < 0 || !res.Finished[i].Done {
					t.Fatalf("the drain did not finish its own merged lane: %+v", res.Finished)
				}
				if v := stateVersionOf(t, f.repo.Root(), f.runID); v != strconv.Itoa(SchemaVersion) {
					t.Fatalf("the drain finished its lane at version %s, not its own %d", v, SchemaVersion)
				}
				st, err := ReadState(f.repo.Root(), f.runID)
				if err != nil {
					t.Fatal(err)
				}
				if !slices.ContainsFunc(st.Record, func(e Entry) bool { return e.Stage == StageDriver && strings.Contains(e.Note, "gone") }) {
					t.Fatalf("the run's record does not say the drain took the run over: %+v", st.Record)
				}
				return
			}
			said := ""
			if err != nil {
				said = err.Error()
			} else {
				i := slices.IndexFunc(res.Finished, func(d DrainFinished) bool { return d.RunID == f.runID })
				if i < 0 || res.Finished[i].Done {
					t.Fatalf("the merged run is not named as left: %+v", res.Finished)
				}
				said = res.Finished[i].Note
			}
			wantLeft(t, f.repo.Root(), f.runID, tc.version, before, said)
			if _, err := os.Stat(l.Worktree); err != nil {
				t.Fatalf("the left lane's worktree went: %v", err)
			}
		})
	}
}

// TestADrainsParkedRoutesLeaveALaneAnotherSessionDrives: entering a conflicting lane's
// sync again, watching an armed pull request in the merge queue and handing back a
// closed one each save the run whole, so each leaves a lane of the drain's own that
// another live session drives as its file stands, and the parked reading names why.
func TestADrainsParkedRoutesLeaveALaneAnotherSessionDrives(t *testing.T) {
	for _, tc := range []struct {
		name  string
		armed bool
		pr    func(branch string) stubPR
	}{
		{name: "the sync entered again", pr: func(b string) stubPR {
			return stubPR{Number: 7, HeadRefName: b, State: "OPEN", Mergeable: "CONFLICTING", MergeStateStatus: "DIRTY"}
		}},
		{name: "the merge-queue watch", armed: true, pr: func(b string) stubPR {
			return stubPR{Number: 7, HeadRefName: b, State: "OPEN", Mergeable: "MERGEABLE", MergeStateStatus: "CLEAN"}
		}},
		{name: "the closed pull request's hand-back", pr: func(b string) stubPR {
			return stubPR{Number: 7, HeadRefName: b, State: "CLOSED", Mergeable: "UNKNOWN", MergeStateStatus: "UNKNOWN"}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var f *landFixture
			var l Lane
			if tc.armed {
				f = drainLandFixture(t)
				first := f.firstMove(t)
				f.runID, f.issue = first.Start.RunID, secondIssue
				l = f.armIssueLaneOnce(t)
			} else {
				f, l = parkDrainLane(t)
			}
			readAsParked(t, f, l)
			f.listPRs(t, tc.pr(l.Branch))
			home, _ := stopWorld(t, false)
			o := Options{stopHome: home}
			claimedBy(t, f.repo.Root(), f.runID, o, hostTwo, hostOne, hostOne, hostTwo)
			before := stateAtVersion(t, f.repo.Root(), f.runID, schemaVersionPrevious)

			res, err := Drain(f.repo.Root(), o, DrainOptions{})
			if err != nil {
				t.Fatalf("the move: %v", err)
			}
			wantLeft(t, f.repo.Root(), f.runID, schemaVersionPrevious, before, parkedOf(t, res, f.runID).Note)
		})
	}
}

// TestARecordOfTranscriptsLeavesARunAnotherSessionDrives: implement record makes no
// claim, and its capture saves the run whole, so a finished run whose file states
// another version and which another live session drives is left as it stands, with
// nothing captured, and the refusal names it.
func TestARecordOfTranscriptsLeavesARunAnotherSessionDrives(t *testing.T) {
	f := newLandFixture(t, queueRuleset("MERGE"))
	l := f.landedToArmedOnce(t)
	f.merged(t, l.HeadSHA)
	if res := f.step(t); !res.Complete {
		t.Fatalf("premise: the run completes once its lane lands: %+v", res)
	}
	claimedBy(t, f.repo.Root(), f.runID, Options{}, hostTwo, hostOne, hostOne, hostTwo)
	before := stateAtVersion(t, f.repo.Root(), f.runID, schemaVersionPrevious)
	var calls []string

	_, err := CaptureTranscripts(f.repo.Root(), f.runID, []string{"/t/main.jsonl"}, fakeCapture(&calls), Options{})

	if _, ok := AsRefusal(err); !ok {
		t.Fatalf("want a refusal, got %v", err)
	}
	if len(calls) != 0 {
		t.Fatalf("a transcript was captured for a run left as it stands: %v", calls)
	}
	wantLeft(t, f.repo.Root(), f.runID, schemaVersionPrevious, before, err.Error())
}

// TestADrainsMoveLeavesItsLaneInProgressAnotherSessionDrives: a drain's move admits a
// lane of its own that names no admission, which saves its run whole, so a lane whose
// file states another version and which another live session drives is left as it
// stands, and the move's refusal names it.
func TestADrainsMoveLeavesItsLaneInProgressAnotherSessionDrives(t *testing.T) {
	f := drainLandFixture(t)
	first := f.firstMove(t)
	f.runID, f.issue = first.Start.RunID, secondIssue
	home, _ := stopWorld(t, false)
	o := Options{stopHome: home}
	claimedBy(t, f.repo.Root(), f.runID, o, hostTwo, hostOne, hostOne, hostTwo)
	before := stateAtVersion(t, f.repo.Root(), f.runID, schemaVersionPrevious)

	_, err := Drain(f.repo.Root(), o, DrainOptions{})
	if err == nil {
		t.Fatal("the move went on over a lane it may not save")
	}
	wantLeft(t, f.repo.Root(), f.runID, schemaVersionPrevious, before, err.Error())
}
