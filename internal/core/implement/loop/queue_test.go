package loop

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/core/landing"
	"github.com/intentdriven/abcd/internal/ghcontract"
)

// The landing's merge-queue watch (spc-2609212015054359 step 5,
// iss-2610090642376032): an armed pull request a merge queue gates is read in
// the queue on each wait for its merge; one left unqueued at CLEAN is queued,
// and one the queue dropped after a failed merge-group check is reported,
// naming the check.

// stubQueue is the stub forge's answer to the queue state query for pull
// request #7: its head, whether it is queued, its arming (enabledAt "" for
// none), its merge state and the reason of its last removal from the queue
// ("" for none).
type stubQueue struct {
	head      string
	queued    bool
	enabledAt string
	status    string
	removed   string
}

// queueAnswer makes the stub forge answer the queue state query with q.
func (f *landFixture) queueAnswer(t *testing.T, q stubQueue) {
	t.Helper()
	pr := map[string]any{"id": "PR_kwDOstub7", "state": "OPEN", "headRefOid": q.head, "isInMergeQueue": q.queued,
		"mergeQueueEntry": nil, "autoMergeRequest": nil, "mergeStateStatus": q.status,
		"timelineItems": map[string]any{"nodes": []any{}}}
	if q.queued {
		pr["mergeQueueEntry"] = map[string]any{"state": "AWAITING_CHECKS", "position": 2}
	}
	if q.enabledAt != "" {
		pr["autoMergeRequest"] = map[string]any{"mergeMethod": "MERGE", "enabledAt": q.enabledAt}
	}
	if q.removed != "" {
		pr["timelineItems"] = map[string]any{"nodes": []any{map[string]any{"createdAt": "2026-10-15T07:48:00Z", "reason": q.removed}}}
	}
	f.writeGH(t, "queue.json", map[string]any{"data": map[string]any{"repository": map[string]any{"pullRequest": pr}}})
}

// writeGH writes v as JSON into the stub forge's file name.
func (f *landFixture) writeGH(t *testing.T, name string, v any) {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(f.gh, name), data, 0o644); err != nil {
		t.Fatal(err)
	}
}

// lastNote is the run's last record line.
func (f *landFixture) lastNote(t *testing.T) string {
	t.Helper()
	st, err := ReadState(f.repo.Root(), f.runID)
	if err != nil {
		t.Fatal(err)
	}
	return st.Record[len(st.Record)-1].Note
}

// TestALandedPullRequestLeftUnqueuedAtCleanIsEnqueued: an armed pull request
// the forge reports CLEAN and outside the merge queue is queued by the
// forge's enqueue call, pinned to the pushed head; one in the queue is waited
// on, its place named.
func TestALandedPullRequestLeftUnqueuedAtCleanIsEnqueued(t *testing.T) {
	f := newLandFixture(t, queueRuleset("MERGE"))
	l := f.landedToArmed(t)
	if !l.Landing.Armed {
		t.Fatalf("the lane's pull request is armed: %+v", l.Landing)
	}
	pushed := l.Landing.Pushed

	// In the queue: the ordinary wait, naming its place, and nothing queued.
	f.queueAnswer(t, stubQueue{head: pushed, queued: true, enabledAt: "2026-10-15T07:00:00Z", status: "CLEAN"})
	_, err := advance(f.repo.Root(), f.runID, f.stages, Options{})
	if r := mustRefusal(t, err); !r.Contention || !strings.Contains(r.Reason, "in main's merge queue (awaiting_checks, position 2)") {
		t.Fatalf("a queued pull request is waited on, its place named: %+v", r)
	}
	if strings.Contains(f.ghLog(t), "enqueuePullRequest") {
		t.Fatalf("a queued pull request is not queued again:\n%s", f.ghLog(t))
	}

	// Armed, CLEAN and outside the queue: queued, pinned to the pushed head.
	f.queueAnswer(t, stubQueue{head: pushed, enabledAt: "2026-10-15T07:00:00Z", status: "CLEAN"})
	res := f.step(t)
	if res.Stage != StageLand {
		t.Fatalf("the lane stays at land after its pull request is queued: %+v", res)
	}
	log := f.ghLog(t)
	if !strings.Contains(log, "enqueuePullRequest") || !strings.Contains(log, "id=PR_kwDOstub7") || !strings.Contains(log, "h="+pushed) {
		t.Fatalf("the pull request is queued by its node id, pinned to the pushed head %s:\n%s", pushed, log)
	}
	if _, err := os.Stat(filepath.Join(f.gh, "enqueued")); err != nil {
		t.Fatalf("the forge's enqueue call was made: %v", err)
	}
	if note := f.lastNote(t); !strings.Contains(note, "sat armed and CLEAN outside main's merge queue, so the loop queued it") {
		t.Fatalf("the record says the loop queued it: %q", note)
	}
	if strings.Contains(log, "--disable-auto") {
		t.Fatalf("a pull request the forge queued is not toggled:\n%s", log)
	}

	// Armed and not CLEAN (a check still running): waited on, not queued.
	_ = os.Remove(filepath.Join(f.gh, "enqueued"))
	f.queueAnswer(t, stubQueue{head: pushed, enabledAt: "2026-10-15T07:00:00Z", status: "BLOCKED"})
	_, err = advance(f.repo.Root(), f.runID, f.stages, Options{})
	if r := mustRefusal(t, err); !r.Contention || !strings.Contains(r.Reason, "armed and not yet in the merge queue (the forge reports BLOCKED)") {
		t.Fatalf("an armed pull request not yet CLEAN is waited on: %+v", r)
	}
	if _, err := os.Stat(filepath.Join(f.gh, "enqueued")); err == nil {
		t.Fatal("a pull request not yet CLEAN is not queued")
	}
}

// TestAnEnqueueTheForgeRefusesTogglesTheArmingPastTheGrace: where the forge
// refuses the enqueue call, the arming is left inside the grace, saying when
// it is toggled, and toggled once past it: withdrawn, then armed again by
// the queue's method, pinned to the pushed head.
func TestAnEnqueueTheForgeRefusesTogglesTheArmingPastTheGrace(t *testing.T) {
	f := newLandFixture(t, queueRuleset("MERGE"))
	l := f.landedToArmed(t)
	pushed := l.Landing.Pushed
	if err := os.WriteFile(filepath.Join(f.gh, "refuse-enqueue"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	armedAt := time.Date(2026, 10, 15, 7, 0, 0, 0, time.UTC)
	f.queueAnswer(t, stubQueue{head: pushed, enabledAt: armedAt.Format(time.RFC3339), status: "CLEAN"})

	early := Options{Now: func() time.Time { return armedAt.Add(10 * time.Minute) }}
	_, err := advance(f.repo.Root(), f.runID, f.stages, early)
	r := mustRefusal(t, err)
	if !r.Contention || !strings.Contains(r.Reason, "the forge refused to queue it") || !strings.Contains(r.Reason, "after 07:30 UTC") {
		t.Fatalf("inside the grace the refused enqueue is a wait naming when the arming is toggled: %+v", r)
	}
	if strings.Contains(f.ghLog(t), "--disable-auto") {
		t.Fatalf("the arming is not toggled inside the grace:\n%s", f.ghLog(t))
	}

	late := Options{Now: func() time.Time { return armedAt.Add(45 * time.Minute) }}
	if _, err := advance(f.repo.Root(), f.runID, f.stages, late); err != nil {
		t.Fatalf("past the grace the arming is toggled: %v", err)
	}
	log := f.ghLog(t)
	off := strings.Index(log, "pr merge 7 --disable-auto")
	on := strings.LastIndex(log, "pr merge 7 --auto --merge --match-head-commit="+pushed)
	if off < 0 || on < off {
		t.Fatalf("the toggle withdraws the arming, then arms it again by the queue's method pinned to %s:\n%s", pushed, log)
	}
	if note := f.lastNote(t); !strings.Contains(note, "toggled its auto-merge, toggle 1 of") {
		t.Fatalf("the record says the arming was toggled: %q", note)
	}
}

// TestAPullRequestDroppedAfterAFailedMergeGroupCheckIsReportedWithTheCheck:
// an armed pull request the forge reports open, outside the queue and no
// longer armed is refused for a person, naming the forge's reason and the
// check that failed on its last merge group, and is neither queued nor armed
// again.
func TestAPullRequestDroppedAfterAFailedMergeGroupCheckIsReportedWithTheCheck(t *testing.T) {
	f := newLandFixture(t, queueRuleset("MERGE"))
	l := f.landedToArmed(t)
	pushed := l.Landing.Pushed
	f.queueAnswer(t, stubQueue{head: pushed, status: "CLEAN", removed: "failed_checks"})
	group := strings.Repeat("ab", 20)
	older := strings.Repeat("cd", 20)
	f.writeGH(t, "runs.json", map[string]any{"workflow_runs": []map[string]any{
		{"name": "ci", "head_branch": "gh-readonly-queue/main/pr-7-" + strings.Repeat("0", 40), "head_sha": group, "conclusion": "failure", "created_at": "2026-10-15T07:10:00Z"},
		{"name": "ci", "head_branch": "gh-readonly-queue/main/pr-7-" + strings.Repeat("1", 40), "head_sha": older, "conclusion": "failure", "created_at": "2026-10-14T07:10:00Z"},
		{"name": "ci", "head_branch": "gh-readonly-queue/main/pr-70-" + strings.Repeat("2", 40), "head_sha": strings.Repeat("ef", 20), "conclusion": "failure", "created_at": "2026-10-15T09:10:00Z"},
	}})
	f.writeGH(t, "checkruns.json", map[string]any{"check_runs": []map[string]any{
		{"name": "check (ubuntu-latest)", "conclusion": "timed_out"},
		{"name": "lint", "conclusion": "success"},
		{"name": "smoke", "conclusion": "cancelled"},
	}})
	before := f.ghLog(t)

	_, err := advance(f.repo.Root(), f.runID, f.stages, Options{})
	r := mustRefusal(t, err)
	if r.Contention {
		t.Fatalf("a pull request dropped from the queue is a terminal state, not a wait: %+v", r)
	}
	for _, want := range []string{"left main's merge queue and is no longer armed", "its reason: failed_checks",
		"on its last merge group " + group[:12], "check (ubuntu-latest) (timed out)"} {
		if !strings.Contains(r.Reason, want) {
			t.Fatalf("the report names %q: %s", want, r.Reason)
		}
	}
	for _, not := range []string{"lint", "smoke"} {
		if strings.Contains(r.Reason, not) {
			t.Fatalf("a check that passed or was cancelled with the group is not named as failed (%s): %s", not, r.Reason)
		}
	}
	if !strings.Contains(r.Remedy, "`gh pr merge 7 --auto --merge --match-head-commit="+pushed+"`") || !strings.Contains(r.Remedy, "does not re-queue it") {
		t.Fatalf("the remedy names the re-arming a person makes, by the queue's method: %s", r.Remedy)
	}
	log := strings.TrimPrefix(f.ghLog(t), before)
	if !strings.Contains(log, "commits/"+group+"/check-runs") || strings.Contains(log, "commits/"+older) {
		t.Fatalf("the checks read are the newest group's of this pull request:\n%s", log)
	}
	if strings.Contains(log, "enqueuePullRequest") || strings.Contains(log, "pr merge") {
		t.Fatalf("a dropped pull request is neither queued nor armed again:\n%s", log)
	}

	// The check runs unread: the failed workflow is named instead.
	_ = os.Remove(filepath.Join(f.gh, "checkruns.json"))
	_, err = advance(f.repo.Root(), f.runID, f.stages, Options{})
	if r := mustRefusal(t, err); !strings.Contains(r.Reason, "the failed workflow on its last merge group "+group[:12]+": ci (failure)") {
		t.Fatalf("without a failed check run, the failed workflow of the group is named: %s", r.Reason)
	}

	// No merge-queue run of it at all: the drop is still reported, saying so.
	f.writeGH(t, "runs.json", map[string]any{"workflow_runs": []any{}})
	_, err = advance(f.repo.Root(), f.runID, f.stages, Options{})
	if r := mustRefusal(t, err); r.Contention || !strings.Contains(r.Reason, "no merge-queue run of it") {
		t.Fatalf("a drop with no run to read is reported all the same: %+v", r)
	}
}

// TestADroppedPullRequestsReArmingIsQuotedWithTheQueuesMethod
// (iss-2611031337128901): the command the drop's report quotes is the one the
// toggle runs, carrying the merge method of the queue's rule, since gh refuses
// `pr merge --auto` with none when it cannot prompt; a rule naming no method
// quotes no command rather than one gh refuses.
func TestADroppedPullRequestsReArmingIsQuotedWithTheQueuesMethod(t *testing.T) {
	f := newLandFixture(t, queueRuleset("SQUASH"))
	l := f.landedToArmed(t)
	pushed := l.Landing.Pushed
	if !strings.Contains(f.ghLog(t), "pr merge 7 --auto --squash --match-head-commit="+pushed) {
		t.Fatalf("the lane is armed by the queue's method:\n%s", f.ghLog(t))
	}
	f.queueAnswer(t, stubQueue{head: pushed, status: "CLEAN", removed: "failed_checks"})
	f.writeGH(t, "runs.json", map[string]any{"workflow_runs": []any{}})

	_, err := advance(f.repo.Root(), f.runID, f.stages, Options{})
	r := mustRefusal(t, err)
	want := "`gh " + strings.Join(rearmArgs("7", "--squash", pushed), " ") + "`"
	if !strings.Contains(r.Remedy, want) || !strings.Contains(r.Remedy, "--auto --squash --match-head-commit="+pushed) {
		t.Fatalf("the remedy quotes the toggle's own re-arming, by the queue's method (%s): %s", want, r.Remedy)
	}

	lane := currentLane(t, f.repo, f.runID)
	c := Context{RepoRoot: f.repo.Root(), Now: time.Now()}
	err = droppedRefusal(c, lane, "main", landing.MergeRule{}, queueView{})
	r = mustRefusal(t, err)
	if strings.Contains(r.Remedy, "gh pr merge") || !strings.Contains(r.Remedy, pushed+" by the queue's merge method") {
		t.Fatalf("with no method to quote, the remedy names the method to arm by and quotes no command gh refuses: %s", r.Remedy)
	}
}

// TestADroppedPullRequestWaitsWhileAReleaseHoldsEveryMerge: a release being
// cut withdraws armed merges and re-arms them after, so a pull request out of
// the queue and unarmed under its hold is waited on, not reported dropped.
func TestADroppedPullRequestWaitsWhileAReleaseHoldsEveryMerge(t *testing.T) {
	f := newLandFixture(t, queueRuleset("MERGE"))
	l := f.landedToArmed(t)
	f.queueAnswer(t, stubQueue{head: l.Landing.Pushed, status: "CLEAN", removed: "dequeued"})
	q := otherCopy(t, f.bare, quinnEmail)
	reserveRelease(t, q, "rel-v0.16.0", time.Now().UTC().Truncate(time.Second).Add(-time.Minute))
	_, err := advance(f.repo.Root(), f.runID, f.stages, Options{})
	r := mustRefusal(t, err)
	if !r.Contention || !strings.Contains(r.Reason, "report its pull request dropped") {
		t.Fatalf("under a release's hold the drop is a wait on the hold: %+v", r)
	}
}

// The watch's forge reads and writes are the forge's own: the query's fields
// in the pinned schema, the enqueue mutation the release hold sends, and the
// REST fields it decodes.
func TestTheQueueWatchAsksTheForgeOnlyForPinnedFields(t *testing.T) {
	if err := ghcontract.CheckGraphQL(QueueStateQuery); err != nil {
		t.Fatal(err)
	}
	if named, ok := ghcontract.GraphQLField("Mutation", "enqueuePullRequest"); !ok || named != "EnqueuePullRequestPayload" ||
		!strings.Contains(claim.EnqueueMutation, "expectedHeadOid") {
		t.Fatalf("the enqueue call is the forge's, pinned to the head: %q %v", named, ok)
	}
	for _, c := range []struct{ resource, field string }{
		{"workflow-runs", "workflow_runs.name"},
		{"workflow-runs", "workflow_runs.head_branch"},
		{"workflow-runs", "workflow_runs.head_sha"},
		{"workflow-runs", "workflow_runs.conclusion"},
		{"workflow-runs", "workflow_runs.created_at"},
		{"check-runs", "check_runs.name"},
		{"check-runs", "check_runs.conclusion"},
	} {
		if !ghcontract.RESTField(c.resource, c.field) {
			t.Errorf("the watch decodes %s.%s, which is not pinned", c.resource, c.field)
		}
	}
}

// TestADrainMoveWatchesItsParkedLanesInTheMergeQueue: a drain's lane parked at
// land is never stepped by the drain, so its move watches it: one the listing
// shows armed and CLEAN, which the forge never queued, is queued, and one the
// listing shows disarmed after a failed merge-group check is read as dropped,
// naming the check, and is not queued or armed again.
func TestADrainMoveWatchesItsParkedLanesInTheMergeQueue(t *testing.T) {
	f := drainLandFixture(t)
	first := drainToLane(t, f.repo, Options{}, DrainOptions{})
	if first.Start == nil || first.Lane.Issue != secondIssue {
		t.Fatalf("the drain opens the documentation issue's lane: %+v", first)
	}
	f.runID, f.issue = first.Start.RunID, secondIssue
	l := f.armIssueLane(t)
	pushed := l.Landing.Pushed
	listed := map[string]any{"number": 7, "headRefName": l.Branch, "state": "OPEN", "mergeable": "MERGEABLE", "mergeStateStatus": "CLEAN",
		"autoMergeRequest": map[string]any{"mergeMethod": "MERGE"}}
	f.writeGH(t, "pr.json", []any{listed})

	f.queueAnswer(t, stubQueue{head: pushed, enabledAt: "2026-10-15T07:00:00Z", status: "CLEAN"})
	res, err := Drain(f.repo.Root(), Options{}, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if p := parkedOf(t, res, f.runID); p.State != ParkedWaiting || !strings.Contains(p.Note, "so the loop queued it") {
		t.Fatalf("an armed parked lane left unqueued at CLEAN is queued and still waits: %+v", p)
	}
	if _, err := os.Stat(filepath.Join(f.gh, "enqueued")); err != nil || !strings.Contains(f.ghLog(t), "h="+pushed) {
		t.Fatalf("the drain's move queued the pull request, pinned to %s: %v\n%s", pushed, err, f.ghLog(t))
	}

	_ = os.Remove(filepath.Join(f.gh, "enqueued"))
	delete(listed, "autoMergeRequest")
	listed["mergeStateStatus"] = "BLOCKED"
	f.writeGH(t, "pr.json", []any{listed})
	f.queueAnswer(t, stubQueue{head: pushed, status: "BLOCKED", removed: "failed_checks"})
	group := strings.Repeat("9a", 20)
	f.writeGH(t, "runs.json", map[string]any{"workflow_runs": []map[string]any{
		{"name": "ci", "head_branch": "gh-readonly-queue/main/pr-7-" + strings.Repeat("0", 40), "head_sha": group, "conclusion": "failure", "created_at": "2026-10-15T07:10:00Z"}}})
	f.writeGH(t, "checkruns.json", map[string]any{"check_runs": []map[string]any{{"name": "check (macos-latest)", "conclusion": "failure"}}})
	before := f.ghLog(t)
	res, err = Drain(f.repo.Root(), Options{}, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	p := parkedOf(t, res, f.runID)
	if p.State != ParkedDropped || !strings.Contains(p.Note, "check (macos-latest) (failure)") || !strings.Contains(p.Note, "failed_checks") {
		t.Fatalf("a parked lane dropped from the queue is read as dropped, naming the failed check: %+v", p)
	}
	if !strings.Contains(res.Next, "check (macos-latest)") {
		t.Fatalf("next names the failed check: %s", res.Next)
	}
	log := strings.TrimPrefix(f.ghLog(t), before)
	if strings.Contains(log, "enqueuePullRequest") || strings.Contains(log, "pr merge") {
		t.Fatalf("a dropped pull request is neither queued nor armed again:\n%s", log)
	}
	st, err := ReadState(f.repo.Root(), f.runID)
	if err != nil {
		t.Fatal(err)
	}
	if lane := st.Lanes[0]; lane.Stage != StageLand || !lane.Landing.Armed || lane.Landing.Merged != "" {
		t.Fatalf("the dropped lane stays parked at land for a person: %+v", lane)
	}
}

// TestADrainReportsAPullRequestItsToggleLeftDisarmed: where the forge refuses
// the enqueue past the grace and then refuses the arming the toggle makes
// again, the pull request is left disarmed by the loop itself. The step's
// refusal says so, naming the command that re-arms it, and a drain move reads
// the parked lane as dropped with that refusal as its note, not as still
// armed from the listing it read before the toggle.
func TestADrainReportsAPullRequestItsToggleLeftDisarmed(t *testing.T) {
	f := drainLandFixture(t)
	first := drainToLane(t, f.repo, Options{}, DrainOptions{})
	if first.Start == nil || first.Lane.Issue != secondIssue {
		t.Fatalf("the drain opens the documentation issue's lane: %+v", first)
	}
	f.runID, f.issue = first.Start.RunID, secondIssue
	l := f.armIssueLane(t)
	pushed := l.Landing.Pushed
	f.writeGH(t, "pr.json", []any{map[string]any{"number": 7, "headRefName": l.Branch, "state": "OPEN", "mergeable": "MERGEABLE",
		"mergeStateStatus": "CLEAN", "autoMergeRequest": map[string]any{"mergeMethod": "MERGE"}}})
	f.queueAnswer(t, stubQueue{head: pushed, enabledAt: "2026-10-15T07:00:00Z", status: "CLEAN"})
	for _, name := range []string{"refuse-enqueue", "refuse-rearm"} {
		if err := os.WriteFile(filepath.Join(f.gh, name), nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	before := f.ghLog(t)

	res, err := Drain(f.repo.Root(), Options{}, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	rearm := "gh pr merge 7 --auto --merge --match-head-commit=" + pushed
	p := parkedOf(t, res, f.runID)
	if p.State != ParkedDropped || !strings.Contains(p.Note, "now DISARMED") || !strings.Contains(p.Note, rearm) {
		t.Fatalf("a pull request the drain's toggle left disarmed is read as dropped, naming the re-arming: %+v", p)
	}
	if strings.Contains(p.Note, "open and armed") {
		t.Fatalf("the drain does not report as armed a pull request it disarmed: %+v", p)
	}
	if !strings.Contains(res.Next, "now DISARMED") {
		t.Fatalf("next names the pull request the loop disarmed: %s", res.Next)
	}
	log := strings.TrimPrefix(f.ghLog(t), before)
	if off, on := strings.Index(log, "pr merge 7 --disable-auto"), strings.LastIndex(log, "pr merge 7 --auto"); off < 0 || on < off {
		t.Fatalf("the toggle withdrew the arming and tried to arm it again:\n%s", log)
	}

	// The step's own refusal is the same terminal report, not a wait.
	f.queueAnswer(t, stubQueue{head: pushed, enabledAt: "2026-10-15T07:00:00Z", status: "CLEAN"})
	_, err = advance(f.repo.Root(), f.runID, f.stages, Options{})
	r := mustRefusal(t, err)
	if r.Contention || !r.outOfQueue || !strings.Contains(r.Reason, "now DISARMED") || !strings.Contains(r.Remedy, rearm) {
		t.Fatalf("a failed re-arm is refused out of the queue, naming the re-arming: %+v", r)
	}
}

// TestAParkedLaneTheWatchFinishesIsReadAsFinished: a pull request that merged
// after the drain move fetched the default branch is found merged by the
// watch's own landing step, which finishes the lane; the watch says so, so
// the move names it finished rather than parked and waiting.
func TestAParkedLaneTheWatchFinishesIsReadAsFinished(t *testing.T) {
	f := drainLandFixture(t)
	first := drainToLane(t, f.repo, Options{}, DrainOptions{})
	if first.Start == nil || first.Lane.Issue != secondIssue {
		t.Fatalf("the drain opens the documentation issue's lane: %+v", first)
	}
	f.runID, f.issue = first.Start.RunID, secondIssue
	l := f.armIssueLane(t)
	f.merged(t, l.Landing.Pushed)

	note, dropped, finished, err := watchParked(f.repo.Root(), f.runID, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if !finished || dropped || note == "" {
		t.Fatalf("the watch reports the lane its step finished: finished=%v dropped=%v note=%q", finished, dropped, note)
	}
	st, err := ReadState(f.repo.Root(), f.runID)
	if err != nil {
		t.Fatal(err)
	}
	if st.Lanes[0].Stage != StageDone || parkedAtLand(st) >= 0 {
		t.Fatalf("the lane the watch finished is done, not parked: %+v", st.Lanes[0])
	}
}
