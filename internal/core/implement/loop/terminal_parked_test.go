package loop

import (
	"encoding/json"
	"strconv"
	"strings"
	"testing"
	"time"
)

// The drive says a parked lane's line (spc-2611251808377435 step 2,
// itd-2611141610201875 criterion 1): of a move's parked lanes it says the line
// of an act the watch made (an arming made again, a queuing, a toggle) and the
// report of a lane read dropped, each once while it stands, and never a parked
// lane's ordinary reading, whose note changes with the queue's every move.

// parkedRearmDrive is a drain's lane parked at land with its pull request
// armed at armedAt, the forge listing it open and disarmed, its last merge
// group failed in workflow run 4242.
func parkedRearmDrive(t *testing.T, armedAt time.Time) (*landFixture, string) {
	t.Helper()
	f := drainLandFixture(t)
	first := f.firstMove(t)
	if first.Start == nil || first.Lane.Issue != secondIssue {
		t.Fatalf("the drain opens the documentation issue's lane: %+v", first)
	}
	f.runID, f.issue = first.Start.RunID, secondIssue
	l := f.armIssueLaneOnce(t)
	f.setLanding(t, func(ld *Landing) { ld.ArmedAt = &armedAt })
	f.writeGH(t, "pr.json", []any{map[string]any{"number": 7, "headRefName": l.Branch, "state": "OPEN", "mergeable": "MERGEABLE", "mergeStateStatus": "BLOCKED"}})
	f.writeGH(t, "runs.json", map[string]any{"workflow_runs": []map[string]any{
		{"id": 4242, "name": "ci", "head_branch": "gh-readonly-queue/main/pr-7-" + strings.Repeat("0", 40), "head_sha": strings.Repeat("ab", 20), "conclusion": "failure", "created_at": armedAt.Format(time.RFC3339)}}})
	f.writeGH(t, "checkruns.json", map[string]any{"check_runs": []map[string]any{{"name": "check (ubuntu-latest)", "conclusion": "failure"}}})
	return f, l.Landing.Pushed
}

// drainAt is one drain move on a clock reading at.
func (f *landFixture) drainAt(t *testing.T, at time.Time) DrainResult {
	t.Helper()
	res, err := Drain(f.repo.Root(), Options{Now: func() time.Time { return at }}, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	return res
}

// TestADriveSaysAPullRequestArmedAgainOnceAndItsHandBackOnce: a pull request
// the queue removed and the watch armed again is said on one line naming it
// and "armed again, 1 of 3", and not again while the move reads the same;
// each later arming is a line of its own; and past the bound the hand-back is
// said once, naming the failed check, the command that re-runs it and the
// one that arms it again. Watched failing first: the drive said nothing of a
// parked lane.
func TestADriveSaysAPullRequestArmedAgainOnceAndItsHandBackOnce(t *testing.T) {
	base := time.Now().UTC().Truncate(time.Second)
	f, pushed := parkedRearmDrive(t, base.Add(-time.Hour))
	var lines []string
	tm := newTerminal(f.repo.Root(), TerminalOptions{Stages: DefaultStages(), Say: sayInto(&lines)})

	at := base.Add(-30 * time.Minute)
	for i := 1; i <= maxRearms; i++ {
		f.queueAnswer(t, stubQueue{head: pushed, status: "BLOCKED", removal: removal(at, "failed_checks", mergeQueueBot)})
		res := f.drainAt(t, at.Add(5*time.Minute))
		p := parkedOf(t, res, f.runID)
		if !p.acted || !strings.Contains(p.Note, "armed again, "+strconv.Itoa(i)+" of 3") {
			t.Fatalf("removal %d is armed again and marked as the watch's act: %+v", i, p)
		}
		for range 11 {
			tm.sayMove(res)
		}
		if len(lines) != i {
			t.Fatalf("each arming made again is said once, %d by removal %d:\n%s", i, i, strings.Join(lines, "\n"))
		}
		said := lines[i-1]
		for _, want := range []string{"pull request #7", "merge queue", "armed again, " + strconv.Itoa(i) + " of 3 at this head", f.runID} {
			if !strings.Contains(said, want) {
				t.Fatalf("the line names %q: %q", want, said)
			}
		}
		at = at.Add(15 * time.Minute)
	}

	f.queueAnswer(t, stubQueue{head: pushed, status: "BLOCKED", removal: removal(at, "failed_checks", mergeQueueBot)})
	res := f.drainAt(t, at.Add(5*time.Minute))
	if p := parkedOf(t, res, f.runID); p.State != ParkedDropped {
		t.Fatalf("a fourth removal is read as dropped: %+v", p)
	}
	for range 11 {
		tm.sayMove(res)
	}
	if len(lines) != maxRearms+1 {
		t.Fatalf("the hand-back is said once:\n%s", strings.Join(lines, "\n"))
	}
	report := lines[maxRearms]
	for _, want := range []string{`"check (ubuntu-latest) (failure)"`, "gh run rerun 4242 --failed", "gh pr merge 7 --auto --merge --match-head-commit=" + pushed, "armed it again 3 times"} {
		if !strings.Contains(report, want) {
			t.Fatalf("the hand-back's line names %q: %q", want, report)
		}
	}
	if strings.Count(report, "\n") != 0 {
		t.Fatalf("the hand-back is one paragraph: %q", report)
	}
}

// TestADriveNeverSaysAParkedLanesOrdinaryReading: a parked lane that only
// waits, its note changing at every move with its place in the queue, is
// never said, and neither is one read conflicting, syncing or unread. The
// mark decides, not the words: a waiting note that reads as an act is not
// said unmarked.
func TestADriveNeverSaysAParkedLanesOrdinaryReading(t *testing.T) {
	var lines []string
	tm := newTerminal(t.TempDir(), TerminalOptions{Stages: DefaultStages(), Say: sayInto(&lines)})
	for i := range 10 {
		tm.sayMove(DrainResult{Parked: []DrainParked{
			{Issue: "iss-1", RunID: "run-1", PR: 7, State: ParkedWaiting,
				Note: "pull request #7 is open and armed, waiting on its merge; it is in main's merge queue at position " + strconv.Itoa(10-i)},
			{Issue: "iss-2", RunID: "run-2", PR: 8, State: ParkedConflicting, Note: "pull request #8 conflicts with the default branch, move " + strconv.Itoa(i)},
			{Issue: "iss-3", RunID: "run-3", PR: 9, State: ParkedSyncing, Note: "pull request #9 went back through its sync, move " + strconv.Itoa(i)},
			{Issue: "iss-4", RunID: "run-4", PR: 10, State: ParkedUnread, Note: "pull request #10 could not be read from the forge, move " + strconv.Itoa(i)},
			{Issue: "iss-5", RunID: "run-5", PR: 11, State: ParkedWaiting,
				Note: "pull request #11 was removed from main's merge queue by the queue and is armed again, 1 of 3 at this head"},
		}})
	}
	if len(lines) != 0 {
		t.Fatalf("a parked lane's ordinary reading is never said:\n%s", strings.Join(lines, "\n"))
	}
}

// TestTheParkedMarkIsNotInTheDrainsJSON: `abcd drain --json` carries no field
// for the mark; a marked reading marshals as the same reading unmarked.
func TestTheParkedMarkIsNotInTheDrainsJSON(t *testing.T) {
	p := DrainParked{Issue: "iss-1", RunID: "run-1", PR: 7, State: ParkedWaiting, Note: "the loop queued it"}
	plain, err := json.Marshal(DrainResult{Parked: []DrainParked{p}})
	if err != nil {
		t.Fatal(err)
	}
	p.acted = true
	marked, err := json.Marshal(DrainResult{Parked: []DrainParked{p}})
	if err != nil {
		t.Fatal(err)
	}
	if string(plain) != string(marked) {
		t.Fatalf("the mark reaches the JSON:\n%s\n%s", plain, marked)
	}
	var shape struct {
		Parked []map[string]any `json:"parked"`
	}
	if err := json.Unmarshal(marked, &shape); err != nil || len(shape.Parked) != 1 {
		t.Fatalf("the reading decodes: %v %s", err, marked)
	}
	for k := range shape.Parked[0] {
		if !strings.Contains(" issue run_id pr bundle state note ", " "+k+" ") {
			t.Fatalf("a parked reading carries a field it did not before: %q in %s", k, marked)
		}
	}
}

// TestAParkedLanesLineHoldsNoLane: the move that says a parked lane's line,
// an act's or a hand-back's, opens its next lane in the same pass: the lane
// is said opened after it, and the move reads as movement.
func TestAParkedLanesLineHoldsNoLane(t *testing.T) {
	for _, p := range []DrainParked{
		{Issue: "iss-1", RunID: "run-1", PR: 7, State: ParkedWaiting, acted: true,
			Note: "pull request #7 was removed from main's merge queue by the queue and is armed again, 1 of 3 at this head"},
		{Issue: "iss-1", RunID: "run-1", PR: 7, State: ParkedDropped,
			Note: "pull request #7 left main's merge queue and is no longer armed; arm it again with `gh pr merge 7 --auto --merge`"},
	} {
		t.Run(p.State, func(t *testing.T) {
			var lines []string
			tm := newTerminal(t.TempDir(), TerminalOptions{Stages: DefaultStages(), Say: sayInto(&lines)})
			res := DrainResult{Parked: []DrainParked{p}, Start: &StartResult{RunID: "run-2"}, Lane: &DrainLane{Issue: "iss-2", RunID: "run-2"}}
			tm.sayMove(res)
			if len(lines) != 2 || !strings.Contains(lines[0], "run run-1: "+p.Note) || lines[1] != "opened the lane of iss-2: run run-2" {
				t.Fatalf("the parked lane's line and the opening are both said:\n%s", strings.Join(lines, "\n"))
			}
			if !drainMoved(res) {
				t.Fatal("the move that opened a lane reads as movement")
			}
		})
	}
}
