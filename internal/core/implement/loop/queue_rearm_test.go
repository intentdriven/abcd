package loop

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/core/landing"
)

// The watch arms again a pull request the merge queue removed
// (itd-2611141610201875, spc-2611251808377435 step 1): only the queue's own
// removal, only one later than the loop's own last arming of the pushed head
// and than the removal it last armed again for, at most maxRearms times at one
// head, and only past the fence and the merge gate. Past the bound the lane is
// handed back naming the failed check, the command that re-runs it and the
// command that arms it again.

// rearmArmedAt is when the fixture's landing armed its pull request: before
// the queue's removal of #1011 as the forge recorded it.
var rearmArmedAt = time.Date(2026, 10, 15, 7, 0, 0, 0, time.UTC)

// mergeQueueBot is the actor the forge names on its merge queue's removals.
var mergeQueueBot = map[string]any{"__typename": "Bot", "login": "github-merge-queue"}

// capturedRemoval is the merge queue's removal of pull request pr for its
// failed checks, as the forge answered it (testdata/queue-removals.json).
func capturedRemoval(t *testing.T, pr string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "queue-removals.json"))
	if err != nil {
		t.Fatal(err)
	}
	var all map[string]json.RawMessage
	if err := json.Unmarshal(data, &all); err != nil {
		t.Fatal(err)
	}
	var nodes []map[string]any
	if err := json.Unmarshal(all[pr], &nodes); err != nil {
		t.Fatal(err)
	}
	for _, n := range nodes {
		if n["reason"] == "failed_checks" {
			return n
		}
	}
	t.Fatalf("no failed_checks removal of #%s was captured", pr)
	return nil
}

// removal is a removal node the forge answers: at, reason and actor (nil for
// a removal naming none).
func removal(at time.Time, reason string, actor map[string]any) map[string]any {
	n := map[string]any{"createdAt": at.UTC().Format(time.RFC3339), "reason": reason}
	if actor != nil {
		n["actor"] = actor
	}
	return n
}

// rearmFixture is an armed lane whose landing armed it at rearmArmedAt, and
// whose last merge group failed one check, in workflow run 4242.
func rearmFixture(t *testing.T) (*landFixture, string) {
	t.Helper()
	f := newLandFixture(t, queueRuleset("MERGE"))
	l := f.landedToArmedOnce(t)
	f.setLanding(t, func(ld *Landing) {
		at := rearmArmedAt
		ld.ArmedAt = &at
	})
	group := strings.Repeat("ab", 20)
	f.writeGH(t, "runs.json", map[string]any{"workflow_runs": []map[string]any{
		{"id": 4242, "name": "ci", "head_branch": "gh-readonly-queue/main/pr-7-" + strings.Repeat("0", 40), "head_sha": group, "conclusion": "failure", "created_at": "2026-10-15T07:40:00Z"},
		{"id": 1111, "name": "ci", "head_branch": "gh-readonly-queue/main/pr-7-" + strings.Repeat("1", 40), "head_sha": strings.Repeat("cd", 20), "conclusion": "failure", "created_at": "2026-10-14T07:10:00Z"},
	}})
	f.writeGH(t, "checkruns.json", map[string]any{"check_runs": []map[string]any{{"name": "check (ubuntu-latest)", "conclusion": "failure"}}})
	return f, l.Landing.Pushed
}

// setLanding changes the run's lane's landing as edit says.
func (f *landFixture) setLanding(t *testing.T, edit func(ld *Landing)) {
	t.Helper()
	if err := mutate(f.repo.Root(), f.runID, func(_ *os.Root, st *State) (bool, error) {
		edit(st.Lanes[0].Landing)
		return true, nil
	}); err != nil {
		t.Fatal(err)
	}
}

// landingNow is the run's lane's landing as its state file holds it.
func (f *landFixture) landingNow(t *testing.T) Landing {
	t.Helper()
	st, err := ReadState(f.repo.Root(), f.runID)
	if err != nil {
		t.Fatal(err)
	}
	return *st.Lanes[0].Landing
}

// removedBy makes the forge answer that pull request #7, at the pushed head,
// is open, outside the queue and disarmed, its last removal node.
func (f *landFixture) removedBy(t *testing.T, pushed string, node map[string]any) {
	t.Helper()
	f.queueAnswer(t, stubQueue{head: pushed, status: "BLOCKED", removal: node})
}

// stepAt advances the run once on a clock reading at.
func (f *landFixture) stepAt(at time.Time) (StepResult, error) {
	return advance(f.repo.Root(), f.runID, f.stages, Options{Now: func() time.Time { return at }})
}

// armings counts the arming calls in a stretch of the forge's log.
func armings(log string) int { return strings.Count(log, "pr merge 7 --auto ") }

// wantDroppedUnarmed is a step refused as the dropped report, with no arming
// call made.
func (f *landFixture) wantDroppedUnarmed(t *testing.T, at time.Time) *Refusal {
	t.Helper()
	before := f.ghLog(t)
	_, err := f.stepAt(at)
	r := mustRefusal(t, err)
	if r.Contention || !r.outOfQueue || !strings.Contains(r.Reason, "left main's merge queue and is no longer armed") {
		t.Fatalf("the step is the dropped report: %+v", r)
	}
	if n := armings(strings.TrimPrefix(f.ghLog(t), before)); n != 0 {
		t.Fatalf("the step armed the pull request %d times:\n%s", n, f.ghLog(t))
	}
	return r
}

func TestAPullRequestTheQueueRemovedIsArmedAgain(t *testing.T) {
	f, pushed := rearmFixture(t)
	node := capturedRemoval(t, "1011")
	f.removedBy(t, pushed, node)
	before := f.ghLog(t)
	now := time.Date(2026, 10, 15, 7, 50, 0, 0, time.UTC)
	res, err := f.stepAt(now)
	if err != nil || res.Stage != StageLand {
		t.Fatalf("a pull request the queue removed is armed again and the lane stays at land: %+v %v", res, err)
	}
	log := strings.TrimPrefix(f.ghLog(t), before)
	if armings(log) != 1 || !strings.Contains(log, "pr merge 7 --auto --merge --match-head-commit="+pushed) {
		t.Fatalf("it is armed again once, by the mirror's method pinned to %s:\n%s", pushed, log)
	}
	ld := f.landingNow(t)
	removedAt, _ := time.Parse(time.RFC3339, node["createdAt"].(string))
	if ld.Rearms != 1 || ld.RearmedRemoval == nil || !ld.RearmedRemoval.Equal(removedAt) || ld.ArmedAt == nil || !ld.ArmedAt.Equal(now) || !ld.Armed {
		t.Fatalf("the landing counts the arming, keeps the removal's time and the new arming's: %+v", ld)
	}
	note := f.lastNote(t)
	for _, want := range []string{"pull request #7", "merge queue", "failed_checks", "armed again, 1 of 3 at this head"} {
		if !strings.Contains(note, want) {
			t.Fatalf("the note names %q: %q", want, note)
		}
	}
	if strings.Contains(note, "; ") || strings.Contains(strings.TrimSuffix(note, "."), ". ") {
		t.Fatalf("the note is one sentence: %q", note)
	}
}

// The forge's reason for the removal chooses nothing: one the code has never
// seen is armed again as failed_checks is.
func TestTheRemovalsReasonChangesNothing(t *testing.T) {
	for _, reason := range []string{"failed_checks", "a_reason_never_seen"} {
		t.Run(reason, func(t *testing.T) {
			f, pushed := rearmFixture(t)
			f.removedBy(t, pushed, removal(rearmArmedAt.Add(30*time.Minute), reason, mergeQueueBot))
			before := f.ghLog(t)
			if _, err := f.stepAt(rearmArmedAt.Add(40 * time.Minute)); err != nil {
				t.Fatal(err)
			}
			if n := armings(strings.TrimPrefix(f.ghLog(t), before)); n != 1 {
				t.Fatalf("a removal by the queue is armed again whatever its reason: %d armings", n)
			}
		})
	}
}

func TestAFourthRemovalAtOneHeadIsHandedBackAfterThreeArmings(t *testing.T) {
	f, pushed := rearmFixture(t)
	before := f.ghLog(t)
	at := rearmArmedAt
	for i := 1; i <= maxRearms; i++ {
		at = at.Add(10 * time.Minute)
		f.removedBy(t, pushed, removal(at, "failed_checks", mergeQueueBot))
		if _, err := f.stepAt(at.Add(2 * time.Minute)); err != nil {
			t.Fatalf("removal %d is armed again: %v", i, err)
		}
		if note := f.lastNote(t); !strings.Contains(note, "armed again, "+strconv.Itoa(i)+" of 3") {
			t.Fatalf("removal %d's note counts it: %q", i, note)
		}
	}
	at = at.Add(10 * time.Minute)
	f.removedBy(t, pushed, removal(at, "failed_checks", mergeQueueBot))
	r := f.wantDroppedUnarmed(t, at.Add(2*time.Minute))
	if n := armings(strings.TrimPrefix(f.ghLog(t), before)); n != maxRearms {
		t.Fatalf("the forge received exactly %d arming calls at the head: %d", maxRearms, n)
	}
	if ld := f.landingNow(t); ld.Rearms != maxRearms {
		t.Fatalf("the count stays at the bound: %+v", ld)
	}
	if !strings.Contains(r.Reason, "armed it again 3 times at this head") {
		t.Fatalf("the hand-back says the loop armed it again three times: %s", r.Reason)
	}
	check := strings.Index(r.Remedy, "check (ubuntu-latest)")
	rerun := strings.Index(r.Remedy, "`gh run rerun 4242 --failed`")
	arm := strings.Index(r.Remedy, "`gh pr merge 7 --auto --merge --match-head-commit="+pushed+"`")
	step := strings.Index(r.Remedy, "`abcd implement step`")
	drive := strings.Index(r.Remedy, "`abcd drain --drive`")
	if check < 0 || rerun < check || arm < rerun || step < arm || drive < arm {
		t.Fatalf("the remedy names the check, the re-run, the arming, then the step: %s", r.Remedy)
	}
	if strings.Contains(r.Remedy, "1111") {
		t.Fatalf("an older group's run is not re-run: %s", r.Remedy)
	}

	// A person arms it by hand past the bound and withdraws it again: the
	// forge's last removal is the same, and nothing is armed.
	f.wantDroppedUnarmed(t, at.Add(20*time.Minute))
}

// Only the merge queue's own removal is armed again: a person's, another
// application's, one naming no actor, and a pull request with no removal on
// record (its arming withdrawn) are each the dropped report.
func TestARemovalNotTheQueuesOwnIsNotArmedAgain(t *testing.T) {
	after := rearmArmedAt.Add(30 * time.Minute)
	for name, node := range map[string]map[string]any{
		"a person":          removal(after, "dequeued", map[string]any{"__typename": "User", "login": "github-merge-queue"}),
		"another app":       removal(after, "failed_checks", map[string]any{"__typename": "Bot", "login": "some-other-app"}),
		"a lookalike":       removal(after, "failed_checks", map[string]any{"__typename": "Bot", "login": "github-merge-queue[bot]"}),
		"no actor":          removal(after, "failed_checks", nil),
		"no removal at all": nil,
	} {
		t.Run(name, func(t *testing.T) {
			f, pushed := rearmFixture(t)
			f.removedBy(t, pushed, node)
			f.wantDroppedUnarmed(t, after.Add(5*time.Minute))
		})
	}
}

// A removal by the queue is armed again only when it is later than the loop's
// own last arming of the pushed head: one from before it (a removal at an
// earlier head, followed by a resync and a new arming that a person withdrew),
// one at the same second, and one on a landing that keeps no arming time.
func TestARemovalFromBeforeTheLoopsArmingIsNotArmedAgain(t *testing.T) {
	t.Run("before the arming", func(t *testing.T) {
		f, pushed := rearmFixture(t)
		f.removedBy(t, pushed, removal(rearmArmedAt.Add(-time.Minute), "failed_checks", mergeQueueBot))
		f.wantDroppedUnarmed(t, rearmArmedAt.Add(10*time.Minute))
	})
	t.Run("at the arming's second", func(t *testing.T) {
		f, pushed := rearmFixture(t)
		f.removedBy(t, pushed, removal(rearmArmedAt, "failed_checks", mergeQueueBot))
		f.wantDroppedUnarmed(t, rearmArmedAt.Add(10*time.Minute))
	})
	t.Run("no arming time", func(t *testing.T) {
		f, pushed := rearmFixture(t)
		f.setLanding(t, func(ld *Landing) { ld.ArmedAt = nil })
		f.removedBy(t, pushed, removal(rearmArmedAt.Add(time.Minute), "failed_checks", mergeQueueBot))
		f.wantDroppedUnarmed(t, rearmArmedAt.Add(10*time.Minute))
	})
	t.Run("a time that does not parse", func(t *testing.T) {
		f, pushed := rearmFixture(t)
		f.removedBy(t, pushed, map[string]any{"createdAt": "yesterday", "reason": "failed_checks", "actor": mergeQueueBot})
		f.wantDroppedUnarmed(t, rearmArmedAt.Add(10*time.Minute))
	})
}

// A pull request found disarmed after a re-arm, with no later removal on
// record, was disarmed by a person, and that stands: held by the removal the
// landing last armed again for, even where the run's clock reads its own
// arming as earlier than the forge's removal.
func TestAWithdrawalAfterAReArmStands(t *testing.T) {
	f, pushed := rearmFixture(t)
	node := removal(rearmArmedAt.Add(30*time.Minute), "failed_checks", mergeQueueBot)
	f.removedBy(t, pushed, node)
	if _, err := f.stepAt(rearmArmedAt.Add(40 * time.Minute)); err != nil {
		t.Fatal(err)
	}
	f.setLanding(t, func(ld *Landing) {
		at := rearmArmedAt
		ld.ArmedAt = &at
	})
	f.wantDroppedUnarmed(t, rearmArmedAt.Add(50*time.Minute))
}

// The landing's own arming and a toggle each keep the run's time of the arming.
func TestTheLandingsArmingAndAToggleSetArmedAt(t *testing.T) {
	f := newLandFixture(t, queueRuleset("MERGE"))
	from := time.Now().Add(-time.Second)
	f.landedToArmed(t)
	ld := f.landingNow(t)
	if ld.ArmedAt == nil || ld.ArmedAt.Before(from) || ld.ArmedAt.After(time.Now()) {
		t.Fatalf("the landing's arming keeps its time: %+v", ld)
	}

	if err := os.WriteFile(filepath.Join(f.gh, "refuse-enqueue"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	f.queueAnswer(t, stubQueue{head: ld.Pushed, enabledAt: rearmArmedAt.Format(time.RFC3339), status: "CLEAN"})
	late := rearmArmedAt.Add(45 * time.Minute)
	if _, err := f.stepAt(late); err != nil {
		t.Fatal(err)
	}
	if ld := f.landingNow(t); ld.Toggles != 1 || ld.ArmedAt == nil || !ld.ArmedAt.Equal(late) {
		t.Fatalf("a toggle keeps the time it armed again: %+v", ld)
	}
}

// While a release being cut holds every merge, the lane waits on the hold and
// nothing is armed.
func TestARemovedPullRequestWaitsOnAReleasesHold(t *testing.T) {
	f, pushed := rearmFixture(t)
	now := time.Now().UTC().Truncate(time.Second)
	q := otherCopy(t, f.bare, quinnEmail)
	reserveRelease(t, q, "rel-v0.16.0", now.Add(-time.Minute))
	f.removedBy(t, pushed, removal(rearmArmedAt.Add(30*time.Minute), "failed_checks", mergeQueueBot))
	before := f.ghLog(t)
	_, err := f.stepAt(now)
	if r := mustRefusal(t, err); !r.Contention || !r.releaseHeld {
		t.Fatalf("under a release's hold the lane waits on it: %+v", r)
	}
	if n := armings(strings.TrimPrefix(f.ghLog(t), before)); n != 0 {
		t.Fatalf("nothing is armed under the hold: %d", n)
	}
}

// Where the release holds cannot be read, a release's withdrawal is not ruled
// out: nothing is armed, and the report says the holds were not read.
func TestARemovalWithTheHoldsUnreadArmsNothing(t *testing.T) {
	f, pushed := rearmFixture(t)
	lane := currentLane(t, f.repo, f.runID)
	v := rearmableView(pushed, rearmArmedAt.Add(30*time.Minute))
	if err := os.Rename(f.bare, f.bare+".gone"); err != nil {
		t.Fatal(err)
	}
	st, err := ReadState(f.repo.Root(), f.runID)
	if err != nil {
		t.Fatal(err)
	}
	before := f.ghLog(t)
	c := Context{RepoRoot: f.repo.Root(), Now: rearmArmedAt.Add(40 * time.Minute), State: st}
	out, _, err := rearmDropped(c, &lane, "main", landing.MergeRule{Method: "MERGE"}, v)
	r := mustRefusal(t, err)
	if out != nil || !r.outOfQueue || !strings.Contains(r.Reason, "the release holds could not be read") {
		t.Fatalf("unread holds give today's report with its clause: %+v %+v", out, r)
	}
	if n := armings(strings.TrimPrefix(f.ghLog(t), before)); n != 0 || lane.Landing.Rearms != 0 {
		t.Fatalf("nothing is armed with the holds unread: %d, %+v", n, lane.Landing)
	}
}

// rearmableView is the forge's answer for a pull request at pushed that the
// queue removed at at, disarmed.
func rearmableView(pushed string, at time.Time) queueView {
	var v queueView
	v.ID, v.State, v.HeadRefOid, v.MergeStateStatus = "PR_kwDOstub7", "OPEN", pushed, "BLOCKED"
	v.TimelineItems.Nodes = append(v.TimelineItems.Nodes, queueRemovalNode{CreatedAt: at.Format(time.RFC3339), Reason: "failed_checks",
		Actor: &queueActor{Typename: "Bot", Login: "github-merge-queue"}})
	return v
}

// A mirror naming no merge method, and a pushed head that is not a full object
// name, arm nothing.
func TestARemovalArmsNothingWithoutAMethodOrAFullHead(t *testing.T) {
	f, pushed := rearmFixture(t)
	st, err := ReadState(f.repo.Root(), f.runID)
	if err != nil {
		t.Fatal(err)
	}
	c := Context{RepoRoot: f.repo.Root(), Now: rearmArmedAt.Add(40 * time.Minute), State: st}
	for name, c2 := range map[string]struct {
		rule   landing.MergeRule
		pushed string
	}{
		"no method":      {landing.MergeRule{Method: "NO_SUCH_METHOD"}, pushed},
		"a partial head": {landing.MergeRule{Method: "MERGE"}, pushed[:12]},
	} {
		t.Run(name, func(t *testing.T) {
			lane := currentLane(t, f.repo, f.runID)
			lane.Landing.Pushed = c2.pushed
			before := f.ghLog(t)
			out, _, err := rearmDropped(c, &lane, "main", c2.rule, rearmableView(c2.pushed, rearmArmedAt.Add(30*time.Minute)))
			if r := mustRefusal(t, err); out != nil || !r.outOfQueue {
				t.Fatalf("it is the dropped report: %+v %+v", out, r)
			}
			if n := armings(strings.TrimPrefix(f.ghLog(t), before)); n != 0 || lane.Landing.Rearms != 0 {
				t.Fatalf("nothing is armed: %d, %+v", n, lane.Landing)
			}
		})
	}
}

// The merge gate is asked as the watch asks it before it queues or toggles: a
// head it refuses is left open with its reason, a gate it cannot read is a
// wait, and neither arms anything.
func TestARemovalPassesTheMergeGate(t *testing.T) {
	t.Run("refused", func(t *testing.T) {
		f, pushed := rearmFixture(t)
		f.removedBy(t, pushed, removal(rearmArmedAt.Add(30*time.Minute), "failed_checks", mergeQueueBot))
		if err := os.Remove(filepath.Join(f.gh, "approved")); err != nil {
			t.Fatal(err)
		}
		before := f.ghLog(t)
		if _, err := f.stepAt(rearmArmedAt.Add(40 * time.Minute)); err != nil {
			t.Fatal(err)
		}
		if n := armings(strings.TrimPrefix(f.ghLog(t), before)); n != 0 {
			t.Fatalf("a head the gate refuses is not armed again: %d", n)
		}
		if ld := f.landingNow(t); ld.Armed || !strings.HasPrefix(ld.Merge, "left open") || ld.Rearms != 0 {
			t.Fatalf("the lane is left open with the gate's reason: %+v", ld)
		}
	})
	t.Run("unread", func(t *testing.T) {
		f, pushed := rearmFixture(t)
		f.removedBy(t, pushed, removal(rearmArmedAt.Add(30*time.Minute), "failed_checks", mergeQueueBot))
		if err := os.WriteFile(filepath.Join(f.gh, "role-author-a"), []byte("Bad Gateway"), 0o644); err != nil {
			t.Fatal(err)
		}
		before := f.ghLog(t)
		_, err := f.stepAt(rearmArmedAt.Add(40 * time.Minute))
		if r := mustRefusal(t, err); !r.Contention || !strings.Contains(r.Reason, "merge gate could not be read") {
			t.Fatalf("an unread gate is a wait: %+v", r)
		}
		if n := armings(strings.TrimPrefix(f.ghLog(t), before)); n != 0 {
			t.Fatalf("an unread gate arms nothing: %d", n)
		}
	})
}

// A reservation no longer this person's refuses the arming, as the fence
// refuses every act of the landing.
func TestARemovalIsNotArmedAgainOverAnotherPersonsReservation(t *testing.T) {
	f, pushed := rearmFixture(t)
	now := time.Now().UTC().Truncate(time.Second)
	reserveRun(t, f.repo, f.runID, "itd-10", now.Add(-130*time.Minute), "internal/widget")
	q := otherCopy(t, f.bare, quinnEmail)
	if res, err := claim.Reserve(q, claim.Request{Record: "itd-10", Footprint: []string{"internal/gadget"}}, as(t, q, clockAt(now.Add(-time.Minute)))); err != nil || res.Act != claim.ActTakenOver {
		t.Fatalf("Q takes the lapsed reservation over: %+v %v", res, err)
	}
	f.removedBy(t, pushed, removal(now.Add(-30*time.Minute), "failed_checks", mergeQueueBot))
	f.setLanding(t, func(ld *Landing) {
		at := now.Add(-40 * time.Minute)
		ld.ArmedAt = &at
	})
	before := f.ghLog(t)
	_, err := f.stepAt(now)
	if r := mustRefusal(t, err); !strings.Contains(r.Reason, "so this lane does not arm its pull request again") {
		t.Fatalf("the fence refuses the arming: %+v", r)
	}
	if n := armings(strings.TrimPrefix(f.ghLog(t), before)); n != 0 {
		t.Fatalf("nothing is armed over another person's reservation: %d", n)
	}
}

// An arming the network loses is the network's refusal and is not counted; one
// the forge refuses is the dropped report with the forge's reason, and is not
// counted either.
func TestAFailedArmingIsNotCounted(t *testing.T) {
	t.Run("the network", func(t *testing.T) {
		f, pushed := rearmFixture(t)
		f.losesTheNetworkAtTheArming(t)
		f.removedBy(t, pushed, removal(rearmArmedAt.Add(30*time.Minute), "failed_checks", mergeQueueBot))
		_, err := f.stepAt(rearmArmedAt.Add(40 * time.Minute))
		if r := mustRefusal(t, err); !r.Contention || !strings.Contains(r.Reason, "lost the network at gh pr merge") {
			t.Fatalf("a lost network at the arming is the network's wait: %+v", r)
		}
		if ld := f.landingNow(t); ld.Rearms != 0 || ld.RearmedRemoval != nil || !ld.ArmedAt.Equal(rearmArmedAt) {
			t.Fatalf("the count stays: %+v", ld)
		}
	})
	t.Run("the forge", func(t *testing.T) {
		f, pushed := rearmFixture(t)
		if err := os.WriteFile(filepath.Join(f.gh, "refuse-rearm"), nil, 0o644); err != nil {
			t.Fatal(err)
		}
		f.removedBy(t, pushed, removal(rearmArmedAt.Add(30*time.Minute), "failed_checks", mergeQueueBot))
		_, err := f.stepAt(rearmArmedAt.Add(40 * time.Minute))
		r := mustRefusal(t, err)
		if r.Contention || !r.outOfQueue || !strings.Contains(r.Reason, "the forge refused the arming") {
			t.Fatalf("a refused arming is the dropped report with the forge's reason: %+v", r)
		}
		if ld := f.landingNow(t); ld.Rearms != 0 || ld.RearmedRemoval != nil {
			t.Fatalf("the count stays: %+v", ld)
		}
	})
}

// The count is raised only once the forge took the arming: the landing
// rearmDropped is handed is left uncounted by an arming the network lost or the
// forge refused, as well as the state the step leaves on disk.
func TestARefusedArmingLeavesTheLandingInHandUncounted(t *testing.T) {
	for name, fail := range map[string]func(f *landFixture, t *testing.T){
		"the network": func(f *landFixture, t *testing.T) { f.losesTheNetworkAtTheArming(t) },
		"the forge": func(f *landFixture, t *testing.T) {
			if err := os.WriteFile(filepath.Join(f.gh, "refuse-rearm"), nil, 0o644); err != nil {
				t.Fatal(err)
			}
		},
	} {
		t.Run(name, func(t *testing.T) {
			f, pushed := rearmFixture(t)
			fail(f, t)
			st, err := ReadState(f.repo.Root(), f.runID)
			if err != nil {
				t.Fatal(err)
			}
			lane := st.Lanes[0]
			c := Context{RepoRoot: f.repo.Root(), Now: rearmArmedAt.Add(40 * time.Minute), State: st}
			out, _, err := rearmDropped(c, &lane, "main", landing.MergeRule{Method: "MERGE"}, rearmableView(pushed, rearmArmedAt.Add(30*time.Minute)))
			if out != nil || err == nil {
				t.Fatalf("a failed arming is refused: %+v %v", out, err)
			}
			if !strings.Contains(f.ghLog(t), "pr merge 7 --auto --merge --match-head-commit="+pushed) {
				t.Fatalf("premise: the arming was tried:\n%s", f.ghLog(t))
			}
			if ld := lane.Landing; ld.Rearms != 0 || ld.RearmedRemoval != nil || !ld.ArmedAt.Equal(rearmArmedAt) {
				t.Fatalf("the landing in hand is left uncounted: %+v", ld)
			}
		})
	}
}

// losesTheNetworkAtTheArming puts a forge client before the stub that fails
// every arming as a lost network does.
func (f *landFixture) losesTheNetworkAtTheArming(t *testing.T) {
	t.Helper()
	stub := filepath.Join(f.gh, "gh")
	if err := os.Rename(stub, filepath.Join(f.gh, "gh.stub")); err != nil {
		t.Fatal(err)
	}
	wrap := "#!/bin/sh\n" +
		`case "$*" in "pr merge "*"--auto "*) printf '%s\n' "$*" >> "$(dirname "$0")/gh.log"; echo "dial tcp 140.82.112.6:443: connect: network is unreachable" >&2; exit 1 ;; esac` + "\n" +
		`exec "$(dirname "$0")/gh.stub" "$@"` + "\n"
	if err := os.WriteFile(stub, []byte(wrap), 0o755); err != nil {
		t.Fatal(err)
	}
}

// The hand-back names a run id only where it is 1 to 20 digits, and where the
// forge names no failed run it says so and gives the arming command alone.
func TestTheHandBacksReRunIsNamedOnlyByADigitRunID(t *testing.T) {
	for name, c := range map[string]struct {
		id   any
		want string
	}{
		"digits":     {4242, "`gh run rerun 4242 --failed`"},
		"a string":   {"4242; rm -rf /", "the forge names no failed run to re-run"},
		"negative":   {-5, "the forge names no failed run to re-run"},
		"too long":   {json.Number(strings.Repeat("9", 21)), "the forge names no failed run to re-run"},
		"a fraction": {json.Number("1.5"), "the forge names no failed run to re-run"},
		"none":       {nil, "the forge names no failed run to re-run"},
	} {
		t.Run(name, func(t *testing.T) {
			f, pushed := rearmFixture(t)
			run := map[string]any{"name": "ci", "head_branch": "gh-readonly-queue/main/pr-7-" + strings.Repeat("0", 40),
				"head_sha": strings.Repeat("ab", 20), "conclusion": "failure", "created_at": "2026-10-15T07:40:00Z"}
			if c.id != nil {
				run["id"] = c.id
			}
			f.writeGH(t, "runs.json", map[string]any{"workflow_runs": []any{run}})
			f.removedBy(t, pushed, removal(rearmArmedAt.Add(30*time.Minute), "failed_checks", nil))
			r := f.wantDroppedUnarmed(t, rearmArmedAt.Add(40*time.Minute))
			if !strings.Contains(r.Remedy, c.want) || !strings.Contains(r.Remedy, "`gh pr merge 7 --auto --merge --match-head-commit="+pushed+"`") {
				t.Fatalf("the remedy says %q and gives the arming command: %s", c.want, r.Remedy)
			}
			if c.want != "`gh run rerun 4242 --failed`" && strings.Contains(r.Remedy, "gh run rerun") {
				t.Fatalf("no run id that is not 1 to 20 digits reaches a command: %s", r.Remedy)
			}
		})
	}
}

// A resync's new head is a new count: it clears the re-arms and keeps the
// removal the landing last armed again for, and the arming that follows sets a
// new arming time.
func TestAResyncClearsTheCountAndKeepsTheRemoval(t *testing.T) {
	removed, armed := rearmArmedAt.Add(30*time.Minute), rearmArmedAt.Add(40*time.Minute)
	ld := Landing{Pushed: strings.Repeat("ab", 20), Armed: true, Merge: "armed", Toggles: 2, Rearms: 3, RearmedRemoval: &removed, ArmedAt: &armed}
	again := resyncedLanding(ld)
	if again.Rearms != 0 || again.Toggles != 0 || again.Armed || again.Pushed != "" {
		t.Fatalf("a resync starts a new count at a new head: %+v", again)
	}
	if again.RearmedRemoval == nil || !again.RearmedRemoval.Equal(removed) {
		t.Fatalf("a resync keeps the removal last armed again for: %+v", again)
	}
	if ld.Rearms != 3 {
		t.Fatalf("the landing resynced from is left as it was: %+v", ld)
	}
}

// A failed check's name is the forge's text: in the remedy it stands quoted in
// a clause of its own before the commands, on one line, and a back-quote or a
// newline in it never reaches the remedy as it is, so every back-quoted span
// of the remedy is one of the loop's own commands.
func TestAFailedChecksNameCannotReadAsACommand(t *testing.T) {
	f, pushed := rearmFixture(t)
	hostile := "x` then `gh pr merge 7 --admin`\n`rm -rf ~\"\u2028y"
	f.writeGH(t, "checkruns.json", map[string]any{"check_runs": []map[string]any{{"name": hostile, "conclusion": "failure"}}})
	f.removedBy(t, pushed, removal(rearmArmedAt.Add(30*time.Minute), "failed_checks", nil))
	r := f.wantDroppedUnarmed(t, rearmArmedAt.Add(40*time.Minute))
	for _, text := range []string{r.Reason, r.Remedy} {
		if strings.ContainsAny(text, "\n\r\u2028") || strings.Contains(text, "`gh pr merge 7 --admin`") || strings.Contains(text, "`rm") {
			t.Fatalf("the check's name reaches the hand-back as it is: %q", text)
		}
	}
	if !strings.Contains(r.Reason, `: "x' then 'gh pr merge 7 --admin'?'rm -rf ~'?y (failure)"`) {
		t.Fatalf("the reason names the check quoted, as the remedy does: %s", r.Reason)
	}
	named := strings.Index(r.Remedy, `is "x' then 'gh pr merge 7 --admin'?'rm -rf ~'?y (failure)"`)
	if named < 0 || named > strings.Index(r.Remedy, "`gh run rerun 4242 --failed`") {
		t.Fatalf("the name stands quoted, its quotes made single, before the commands: %s", r.Remedy)
	}
	for _, m := range regexp.MustCompile("`([^`]*)`").FindAllStringSubmatch(r.Remedy, -1) {
		if m[1] != "gh run rerun 4242 --failed" && m[1] != "gh pr merge 7 --auto --merge --match-head-commit="+pushed &&
			m[1] != "abcd implement step" && m[1] != "abcd drain --drive" {
			t.Fatalf("a back-quoted span of the remedy is not the loop's own command (%q): %s", m[1], r.Remedy)
		}
	}
}
