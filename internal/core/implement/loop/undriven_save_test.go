package loop

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/runstop"
	"github.com/intentdriven/abcd/internal/core/sessionname"
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
// naming the file and both versions.
func TestAStepUnderAStopLeavesAnotherVersionsRunToItsDriver(t *testing.T) {
	for _, tc := range []struct {
		name         string
		version      int
		driver, here int
		alive        []int
		halts        bool
	}{
		{name: "another live session drives it, previous version", version: schemaVersionPrevious, driver: hostTwo, here: hostOne, alive: []int{hostOne, hostTwo}},
		{name: "another live session drives it, version 24", version: schemaVersionUnrenamed, driver: hostTwo, here: hostOne, alive: []int{hostOne, hostTwo}},
		{name: "its driver is gone", version: schemaVersionPrevious, driver: hostTwo, here: hostOne, alive: []int{hostOne}},
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
			if r.Stage != StageStop || !r.Contention {
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
// A run another live session drives, one of the drain's own lanes or not, is left as
// its file stands, named among what the move finished, worktree and all. A run whose
// driver is gone is claimed by the drain, with a line in its record, and finished: a
// drain resumed by a newer abcd finishes its own lanes, and a new drain finishes the
// merged lanes an earlier drain, or a person's build, left.
func TestADrainsLandingStepLeavesAMergedRunToItsDriver(t *testing.T) {
	for _, tc := range []struct {
		name     string
		version  int
		own      bool
		alive    []int
		finishes bool
	}{
		{name: "its own lane another live session drives", version: schemaVersionPrevious, own: true, alive: []int{hostOne, hostTwo}},
		{name: "another run another live session drives", version: schemaVersionPrevious, alive: []int{hostOne, hostTwo}},
		{name: "its own lane whose driver is gone", version: schemaVersionPrevious, own: true, alive: []int{hostOne}, finishes: true},
		{name: "another run whose driver is gone", version: schemaVersionPrevious, alive: []int{hostOne}, finishes: true},
		{name: "another run at this abcd's version", version: SchemaVersion, alive: []int{hostOne, hostTwo}, finishes: true},
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
			before := readStateBytes(t, f.repo.Root(), f.runID)
			if tc.version != SchemaVersion {
				before = stateAtVersion(t, f.repo.Root(), f.runID, tc.version)
			}

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
				if tc.version != SchemaVersion && !slices.ContainsFunc(st.Record, func(e Entry) bool { return e.Stage == StageDriver && strings.Contains(e.Note, "gone") }) {
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
// stands. The move is not refused for it: it names the lane as left to that session,
// once a move, and ends as it does over any lane in progress.
func TestADrainsMoveLeavesItsLaneInProgressAnotherSessionDrives(t *testing.T) {
	f := drainLandFixture(t)
	first := f.firstMove(t)
	f.runID, f.issue = first.Start.RunID, secondIssue
	home, _ := stopWorld(t, false)
	o := Options{stopHome: home}
	claimedBy(t, f.repo.Root(), f.runID, o, hostTwo, hostOne, hostOne, hostTwo)
	before := stateAtVersion(t, f.repo.Root(), f.runID, schemaVersionUnadmitted)

	for range 2 {
		res, err := Drain(f.repo.Root(), o, DrainOptions{})
		if err != nil {
			t.Fatalf("the move was refused over a lane another session drives: %v", err)
		}
		if !strings.Contains(res.Next, "run "+f.runID+", is left as its file stands") {
			t.Fatalf("the move does not name the lane it left: %q", res.Next)
		}
		wantLeft(t, f.repo.Root(), f.runID, schemaVersionUnadmitted, before, res.Next)
	}
}

// TestAReceiptLeavesARunAnotherSessionDrives: implement receipt makes no claim, and its
// hand-in saves the run whole, so a run whose file states another version and which
// another live session drives is left as it stands, nothing handed in, and the refusal
// says how the run is taken up.
func TestAReceiptLeavesARunAnotherSessionDrives(t *testing.T) {
	repo, runID, l, _ := awaitingLane(t)
	home, _ := stopWorld(t, false)
	o := Options{stopHome: home}
	claimedBy(t, repo.Root(), runID, o, hostTwo, hostOne, hostOne, hostTwo)
	before := stateAtVersion(t, repo.Root(), runID, schemaVersionPrevious)

	_, err := Receipt(repo.Root(), runID, l.awaiting().Receipt, DefaultStages(), o)

	r, ok := leftAtVersion(err)
	if !ok {
		t.Fatalf("want the refusal that leaves the run at its version, got %v", err)
	}
	wantLeft(t, repo.Root(), runID, schemaVersionPrevious, before, r.Error())
	if !strings.Contains(r.Remedy, "back off") {
		t.Fatalf("the refusal does not say to back off from the live driver: %s", r.Remedy)
	}
}

// TestAStepWhoseClaimClaimedNothingLeavesAnotherVersionsRun: a step whose claim could
// not read the process table, or could not find the session it runs under, claims
// nothing, so its own save over a run whose file states another version is refused and
// the run is left as its file stands.
func TestAStepWhoseClaimClaimedNothingLeavesAnotherVersionsRun(t *testing.T) {
	for _, tc := range []struct {
		name string
		set  func()
		why  string
	}{
		{"the process table cannot be read", func() {
			readProcs = func() (sessionname.Table, error) { return sessionname.Table{}, errors.New("no table") }
		}, "the process table cannot be read"},
		{"the session it runs under cannot be found", func() { hereFrom = func() int { return 77777 } }, "the session this call runs under cannot be found"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := briefRepo(t, agentsMarked)
			start, err := startedOnce(t, repo, "itd-10")
			if err != nil {
				t.Fatal(err)
			}
			home, _ := stopWorld(t, false)
			o := Options{stopHome: home}
			claimedBy(t, repo.Root(), start.RunID, o, hostTwo, hostOne, hostOne)
			before := stateAtVersion(t, repo.Root(), start.RunID, schemaVersionPrevious)
			tc.set()

			if err := ClaimDriver(repo.Root(), start.RunID, o); err != nil {
				t.Fatalf("premise: the claim claims nothing and refuses nothing: %v", err)
			}
			_, err = advance(repo.Root(), start.RunID, DefaultStages(), o)

			r, ok := leftAtVersion(err)
			if !ok {
				t.Fatalf("want the refusal that leaves the run at its version, got %v", err)
			}
			wantLeft(t, repo.Root(), start.RunID, schemaVersionPrevious, before, r.Error())
			if !strings.Contains(r.Reason, tc.why) {
				t.Fatalf("the refusal does not say %q: %s", tc.why, r.Reason)
			}
		})
	}
}

// TestAResumedStartClaimsARunWhoseFileStatesAnotherVersion: a build that resumes a run
// whose file states another version claims it before anything saves it. Its driver gone,
// the run is this session's, the record naming the driver it replaced, and its late
// admission saved at this abcd's version; another live session driving it, the start is
// refused at exit 3 with the run left as its file stands.
func TestAResumedStartClaimsARunWhoseFileStatesAnotherVersion(t *testing.T) {
	for _, tc := range []struct {
		name  string
		alive []int
		takes bool
	}{
		{"its driver is gone", []int{hostOne}, true},
		{"another live session drives it", []int{hostOne, hostTwo}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo, runID, _, _ := awaitingLane(t)
			home, _ := stopWorld(t, false)
			o := Options{stopHome: home, noReserve: true}
			claimedBy(t, repo.Root(), runID, o, hostTwo, hostOne, tc.alive...)
			before := stateAtVersion(t, repo.Root(), runID, schemaVersionUnadmitted)

			res, err := Start(repo.Root(), "itd-10", o)

			if !tc.takes {
				r, ok := AsRefusal(err)
				if !ok || !r.Contention || r.Stage != StageDriver {
					t.Fatalf("want the claim's refusal at exit 3, got %v", err)
				}
				if !strings.Contains(r.Reason, "is driven by") {
					t.Fatalf("the refusal does not name the live driver: %s", r.Reason)
				}
				after := readStateBytes(t, repo.Root(), runID)
				if !bytes.Equal(before, after) {
					t.Fatalf("the start rewrote a run another session drives:\n%s", after)
				}
				return
			}
			if err != nil || !res.Resumed {
				t.Fatalf("the start resumes the run: %+v %v", res, err)
			}
			if v := stateVersionOf(t, repo.Root(), runID); v != strconv.Itoa(SchemaVersion) {
				t.Fatalf("the resumed run is at version %s, not %d", v, SchemaVersion)
			}
			st, err := ReadState(repo.Root(), runID)
			if err != nil {
				t.Fatal(err)
			}
			if st.Admitted == nil {
				t.Fatal("the resumed run's admission was not recorded")
			}
			if !slices.ContainsFunc(st.Record, func(e Entry) bool { return e.Stage == StageDriver && strings.Contains(e.Note, "is gone") }) {
				t.Fatalf("the run's record does not name the driver the start replaced: %+v", st.Record)
			}
			if c := controlOf(t, repo.Root(), runID); c.Driver == nil || c.Driver.PID != hostOne {
				t.Fatalf("the start did not claim the run: %+v", c.Driver)
			}
		})
	}
}

// TestADrivesDropLeavesARunAnotherSessionDrives: a drive's drop of a lane whose issue
// moved hands the lane back, a whole save, so a run another live session drives at
// another version is left as its file stands and the drop says why.
func TestADrivesDropLeavesARunAnotherSessionDrives(t *testing.T) {
	repo, runID, _, _ := awaitingLane(t)
	home, _ := stopWorld(t, false)
	o := Options{stopHome: home}
	claimedBy(t, repo.Root(), runID, o, hostTwo, hostOne, hostOne, hostTwo)
	before := stateAtVersion(t, repo.Root(), runID, schemaVersionPrevious)
	st, err := ReadState(repo.Root(), runID)
	if err != nil {
		t.Fatal(err)
	}

	for _, drop := range []func() error{
		func() error { return dropRun(repo.Root(), runID, "the record moved", "the person", time.Now(), o) },
		func() error {
			_, err := dropMovedRun(repo.Root(), st, "the record moved", time.Now(), o)
			return err
		},
	} {
		r, ok := leftAtVersion(drop())
		if !ok {
			t.Fatal("want the refusal that leaves the run at its version")
		}
		wantLeft(t, repo.Root(), runID, schemaVersionPrevious, before, r.Error())
	}
}

// TestADrivesLetGoKeepsTheVersionOfItsLanesRun: at a drive's end the release of a lane
// in progress adds lines to its run's record and nothing else, so the run's file keeps
// the version it states, whichever session drives it.
func TestADrivesLetGoKeepsTheVersionOfItsLanesRun(t *testing.T) {
	repo := drainRepo(t)
	first := drainedToLaneOnce(t, repo, DrainOptions{})
	runID := first.Start.RunID
	editState(t, repo.Root(), runID, func(doc map[string]any) {
		if doc["reserved_at"] == nil {
			doc["reserved_at"] = time.Now().UTC().Format(time.RFC3339)
		}
	})
	stateAtVersion(t, repo.Root(), runID, schemaVersionPrevious)

	(&terminal{repoRoot: repo.Root()}).letGo()

	if v := stateVersionOf(t, repo.Root(), runID); v != strconv.Itoa(schemaVersionPrevious) {
		t.Fatalf("the release raised the run's file to version %s", v)
	}
	st, err := ReadState(repo.Root(), runID)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.ContainsFunc(st.Record, func(e Entry) bool { return e.Stage == StageDriveReleased }) {
		t.Fatalf("the release is not in the run's record: %+v", st.Record)
	}
}

// TestARunningDrainsPaceLeavesALaneAnotherSessionDrives: a running drain given a new
// pace saves each lane in progress whole; a lane another live session drives at another
// version keeps its pace and is named, and the move is not refused for it.
func TestARunningDrainsPaceLeavesALaneAnotherSessionDrives(t *testing.T) {
	repo := drainRepo(t)
	first := drainedToLaneOnce(t, repo, DrainOptions{})
	runID := first.Start.RunID
	home, _ := stopWorld(t, false)
	claimedBy(t, repo.Root(), runID, Options{stopHome: home}, hostTwo, hostOne, hostOne, hostTwo)
	before := stateAtVersion(t, repo.Root(), runID, schemaVersionPrevious)

	moved, err := Drain(repo.Root(), Options{Pace: strp("60/15"), SubAgents: strp("3"), stopHome: home}, DrainOptions{})

	if err != nil {
		t.Fatalf("the move was refused over a lane another session drives: %v", err)
	}
	if !strings.Contains(moved.Next, "run "+runID+", keeps its pace") {
		t.Fatalf("the move does not name the lane that kept its pace: %q", moved.Next)
	}
	wantLeft(t, repo.Root(), runID, schemaVersionPrevious, before, moved.Next)
}

// TestTheGuardedRoutesLeaveARunAnotherSessionDrives: the stuck hand-back, the go-on after
// a lift and the mark of an await a stop refused each save the run whole, so each leaves a
// run another live session drives at another version as its file stands and says so.
func TestTheGuardedRoutesLeaveARunAnotherSessionDrives(t *testing.T) {
	stopped := refuse(StageStop, "", "", "a stop holds", "")
	for _, tc := range []struct {
		name  string
		route func(repoRoot, runID, receipt string, o Options) error
	}{
		{"the stuck hand-back", func(repoRoot, runID, _ string, o Options) error {
			_, err := handBackStuck(repoRoot, runID, stuckBounds{}, time.Now(), o.laneSave(true))
			return err
		}},
		{"the go-on after a lift", func(repoRoot, runID, _ string, o Options) error {
			return goOnAfterLift(repoRoot, runID, o)
		}},
		{"the mark of an await a stop refused", func(repoRoot, runID, receipt string, o Options) error {
			return notStartedUnderStop(repoRoot, runID, receipt, DefaultStages(), o, stopped)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo, runID, l, _ := awaitingLane(t)
			home, _ := stopWorld(t, false)
			o := Options{stopHome: home}
			claimedBy(t, repo.Root(), runID, o, hostTwo, hostOne, hostOne, hostTwo)
			before := stateAtVersion(t, repo.Root(), runID, schemaVersionPrevious)

			err := tc.route(repo.Root(), runID, l.awaiting().Receipt, o)

			r, ok := leftAtVersion(err)
			if !ok {
				t.Fatalf("want the refusal that leaves the run at its version, got %v", err)
			}
			wantLeft(t, repo.Root(), runID, schemaVersionPrevious, before, r.Error())
		})
	}
}

// TestTheStopsPageNoteKeepsTheVersionOfTheRun: the note of whether the stop's page was
// written is the control record's mark and a line in the run's record, so it is kept for
// a run whose file states another version, which keeps that version.
func TestTheStopsPageNoteKeepsTheVersionOfTheRun(t *testing.T) {
	repo, runID, _, _ := awaitingLane(t)
	home, _ := stopWorld(t, false)
	o := Options{stopHome: home}
	claimedBy(t, repo.Root(), runID, o, hostTwo, hostOne, hostOne, hostTwo)
	now := time.Now()
	rec := &RunStop{SetBy: "a person", SetAt: now, NotedAt: now, HaltedAt: &now, FromFile: true, PageFailed: "an earlier failure"}
	root, err := os.OpenRoot(repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	c := controlOf(t, repo.Root(), runID)
	c.Stop = rec
	if err := writeControl(root, runID, c); err != nil {
		t.Fatal(err)
	}
	stateAtVersion(t, repo.Root(), runID, schemaVersionPrevious)

	out := notePage(repo.Root(), runID, rec, runstop.Stop{SetBy: "a person", SetAt: now}, o)

	if got := controlOf(t, repo.Root(), runID).Stop.PageFailed; got != out.PageFailed || got == rec.PageFailed {
		t.Fatalf("the control record does not keep what the page note says: %q, answered %q", got, out.PageFailed)
	}
	if v := stateVersionOf(t, repo.Root(), runID); v != strconv.Itoa(schemaVersionPrevious) {
		t.Fatalf("the page note raised the run's file to version %s", v)
	}
	st, err := ReadState(repo.Root(), runID)
	if err != nil {
		t.Fatal(err)
	}
	if n := len(st.Record); n == 0 || st.Record[n-1].Stage != StageStop {
		t.Fatalf("the page note is not in the run's record: %+v", st.Record)
	}
}

// TestASignalsUnsettledMarkOnAnotherVersionsRunIsTakenOver: a signal's stop marked a run
// whose file states another version and whose driver is gone, before its lanes settled.
// A step by a newer abcd claims nothing under the mark, and its check leaves the run as
// its file stands; the refusal names the take-over, since a signal's stop has no file to
// lift, and the take-over claims the run and clears the mark, so its next save is made at
// this version.
func TestASignalsUnsettledMarkOnAnotherVersionsRunIsTakenOver(t *testing.T) {
	repo, runID, _, _, _ := deadLane(t)
	home, _ := stopWorld(t, false)
	o := Options{stopHome: home}
	claimedBy(t, repo.Root(), runID, o, hostTwo, hostOne, hostOne)
	root, err := os.OpenRoot(repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	now := time.Now()
	c := controlOf(t, repo.Root(), runID)
	c.Stop = &RunStop{SetBy: "a signal", SetAt: now, NotedAt: now}
	if err := writeControl(root, runID, c); err != nil {
		t.Fatal(err)
	}
	before := stateAtVersion(t, repo.Root(), runID, schemaVersionPrevious)

	if err := ClaimDriver(repo.Root(), runID, o); err != nil {
		t.Fatalf("the claim under the mark: %v", err)
	}
	r, ok := leftAtVersion(checkOnce(repo.Root(), runID, o))
	if !ok || r.Stage != StageStop {
		t.Fatalf("want the stop's refusal that leaves the run at its version, got %+v", r)
	}
	wantLeft(t, repo.Root(), runID, schemaVersionPrevious, before, r.Error())
	if !strings.Contains(r.Remedy, "`abcd implement resume --take "+runID+"`") || strings.Contains(r.Remedy, "--lift") {
		t.Fatalf("the remedy does not name the take-over alone as the way on: %s", r.Remedy)
	}

	if _, err := TakeRun(repo.Root(), runID, o); err != nil {
		t.Fatalf("the take-over the remedy names: %v", err)
	}
	if c := controlOf(t, repo.Root(), runID); c.Stop != nil || c.Driver == nil || c.Driver.PID != hostOne {
		t.Fatalf("the take-over did not claim the run and clear its mark: %+v %+v", c.Driver, c.Stop)
	}
	// The run is this session's now, so its next whole save is made, at this version.
	if err := saveRun(repo.Root(), runID, o, func(_ *os.Root, st *State) (bool, error) {
		st.Record = append(st.Record, Entry{At: time.Now(), Stage: StageDriver, Note: "a step after the take-over"})
		return true, nil
	}); err != nil {
		t.Fatalf("a save after the take-over: %v", err)
	}
	if v := stateVersionOf(t, repo.Root(), runID); v != strconv.Itoa(SchemaVersion) {
		t.Fatalf("the save after the take-over left the run at version %s", v)
	}
}

// TestADrivesProfileMarkSaysWhatItLeft: a drive marks a run that names no profile with
// its own, a whole save; over a run whose file states another version and that another
// live session drives, the mark is left, the file with it, and the drive says what it
// left and why, once however often it comes back to the lane.
func TestADrivesProfileMarkSaysWhatItLeft(t *testing.T) {
	repo, runID, _, _ := awaitingLane(t)
	home, _ := stopWorld(t, false)
	o := Options{stopHome: home, Profile: "work"}
	claimedBy(t, repo.Root(), runID, o, hostTwo, hostOne, hostOne, hostTwo)
	before := stateAtVersion(t, repo.Root(), runID, schemaVersionPrevious)
	var said []string
	drive := &terminal{repoRoot: repo.Root(), o: TerminalOptions{Options: o, Say: func(s string) { said = append(said, s) }}}

	for range 2 {
		if err := drive.markProfile(runID); err != nil {
			t.Fatalf("the mark refused the drive: %v", err)
		}
	}

	if len(said) != 1 {
		t.Fatalf("the drive says what it left once, not %d times: %q", len(said), said)
	}
	wantLeft(t, repo.Root(), runID, schemaVersionPrevious, before, said[0])
	if !strings.Contains(said[0], "its profile is not marked work") {
		t.Fatalf("the line does not say what was left: %s", said[0])
	}
}

// TestADrivesTakeUpOfARunLeftAtItsVersionEndsTheDrive: a drive's reservation of its lane
// again refused because the run's file states another version and the drive may not save
// it is no other drive taking the issue up: the drive ends on it, saying what it left and
// why, and records nothing on the run, which it may not save.
func TestADrivesTakeUpOfARunLeftAtItsVersionEndsTheDrive(t *testing.T) {
	repo, runID, _, _ := awaitingLane(t)
	home, _ := stopWorld(t, false)
	o := Options{stopHome: home}
	claimedBy(t, repo.Root(), runID, o, hostTwo, hostOne, hostOne, hostTwo)
	before := stateAtVersion(t, repo.Root(), runID, schemaVersionPrevious)
	run, err := ReadState(repo.Root(), runID)
	if err != nil {
		t.Fatal(err)
	}
	var said []string
	drive := &terminal{repoRoot: repo.Root(), o: TerminalOptions{Options: o, Say: func(s string) { said = append(said, s) }}}
	left := wholeSaveLeft(repo.Root(), runID, o)
	if left == nil {
		t.Fatal("premise: the run is left at its version")
	}

	done, ended := drive.tookUp(context.Background(), run, left)

	if !done || ended != TerminalWall {
		t.Fatalf("the drive goes on over a run it may not save: %v %q", done, ended)
	}
	all := strings.Join(said, "\n")
	if strings.Contains(all, "another drive took the issue up") {
		t.Fatalf("the drive says another drive took the issue up:\n%s", all)
	}
	wantLeft(t, repo.Root(), runID, schemaVersionPrevious, before, drive.why)
}

// TestAPageNoteThatCannotBeKeptLeavesTheStopsRefusal: the stop holds whether or not its
// page note is kept, so a note whose save fails ends the move at the stage `stop`, at exit
// 3, as any check under the stop does, and the refusal says the note could not be kept.
func TestAPageNoteThatCannotBeKeptLeavesTheStopsRefusal(t *testing.T) {
	repo, runID, _, _, _ := deadLane(t)
	o, _, folder := stopped(t)
	// The page cannot be written, so there is a note to keep (TestAPageThatCannotBeWrittenIsNamed).
	for _, dir := range []string{accountDigestDir(), runstop.DigestDir} {
		if err := os.WriteFile(filepath.Join(folder, dir), []byte("not a directory\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	later := time.Now().Add(3 * time.Hour)
	o.Now = func() time.Time { return later }
	prev := savePageNote
	savePageNote = func(string, string, func(*os.Root, *State) (bool, error)) error {
		return errors.New("the note's save failed")
	}
	t.Cleanup(func() { savePageNote = prev })

	r := wantStopRefusal(t, checkOnce(repo.Root(), runID, o))

	if !strings.Contains(r.Reason, "the day's page could not be written") || !strings.Contains(r.Reason, "the run's account of it could not be kept: the note's save failed") {
		t.Fatalf("the stop's refusal does not say the page note could not be kept: %s", r.Reason)
	}
}
