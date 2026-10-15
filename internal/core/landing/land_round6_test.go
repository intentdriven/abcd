package landing

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Round 6: the grace's clock, the holders the listing cannot show, the
// listing's bound, and a moved head on the re-run path.
//
// testdata/pr-state-1018-merged-timed.json and pr-state-1020-pending.json are
// the forge's own answers to PRStateQuery as round 6 asks it (completedAt,
// createdAt, contexts(first:100) and pageInfo{hasNextPage} added), read
// 2026-10-15T14:12:01Z by a read-only `gh api graphql` of intentdriven/abcd:
// #1018 after its merge (its required checks' real completion times, the
// newest check (ubuntu-latest) at 12:59:42Z) and #1020 with its required
// checks in progress (rollup PENDING, BEHIND). The same day the forge refused
// contexts(first:101): "exceeds the `first` limit of 100 records".

// recorded is the recorded answer name with edit applied to its pull request,
// its head replaced by @HEAD@.
func recorded(t *testing.T, name string, edit func(pr map[string]any)) string {
	t.Helper()
	m := recordedAnswer(t, name)
	if edit != nil {
		edit(prOf(m))
	}
	out, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	return string(out)
}

func timed1018(t *testing.T, edit func(pr map[string]any)) string {
	return recorded(t, "pr-state-1018-merged-timed.json", edit)
}

func rollupOf(pr map[string]any) map[string]any {
	return pr["statusCheckRollup"].(map[string]any)
}

func nodesOf(pr map[string]any) []any {
	return rollupOf(pr)["contexts"].(map[string]any)["nodes"].([]any)
}

// completed sets the named check run's completion time.
func completed(name, at string) func(pr map[string]any) {
	return func(pr map[string]any) {
		for _, n := range nodesOf(pr) {
			if c := n.(map[string]any); c["name"] == name {
				c["completedAt"] = at
			}
		}
	}
}

func TestTheRound6AnswersParse(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "pr-state-1018-merged-timed.json"))
	if err != nil {
		t.Fatal(err)
	}
	var a prStateAnswer
	if err := json.Unmarshal(raw, &a); err != nil {
		t.Fatal(err)
	}
	r := a.Data.Repository.PullRequest.StatusCheckRollup
	if r == nil || len(r.Contexts.Nodes) != 13 || r.Contexts.PageInfo.HasNextPage {
		t.Fatalf("#1018 read as %+v", r)
	}
	start, ok := r.newestRequiredPass()
	if !ok || start.UTC().Format("15:04:05") != "12:59:42" {
		t.Fatalf("#1018's newest required pass = %v, %v; want 12:59:42 (check (ubuntu-latest))", start, ok)
	}
}

// S1: the grace runs from the later of the arming and the newest passing
// required check, so the time the pull request sat CLEAN is what is timed.
// Checks that take 25 minutes leave a re-run at arming+31 inside the grace.
func TestTheGraceRunsFromTheLastRequiredCheckNotTheArming(t *testing.T) {
	slow := both(unqueued("CLEAN", "REPPL", armedAt), completed("check (ubuntu-latest)", "2026-10-15T13:17:16Z"))

	at(t, "13:23:16") // arming+31, CLEAN+6
	f := newFixture(t, queueRuleset)
	f.forge.prs = f.prFor(f.head())
	f.forge.answers = []string{timed1018(t, slow)}
	res, err := f.land(f.options())
	if err != nil {
		t.Fatal(err)
	}
	if f.forge.called("pr", "merge") != 0 {
		t.Fatalf("forge calls %v, want no toggle 6 minutes after the checks passed", f.forge.calls)
	}
	if s := step(t, res, "enqueue").Note; !strings.Contains(s, "CLEAN for 6 min") || !strings.Contains(s, "after 13:48 UTC") {
		t.Fatalf("enqueue note %q, want CLEAN for 6 min and the time the grace ends", s)
	}

	at(t, "13:48:17") // CLEAN+31
	f = newFixture(t, queueRuleset)
	f.forge.prs = f.prFor(f.head())
	f.forge.answers = []string{timed1018(t, slow), pr1018(t, nil)}
	res, err = f.land(f.options())
	if err != nil {
		t.Fatal(err)
	}
	if f.forge.called("pr", "merge", "7", "--disable-auto") != 1 {
		t.Fatalf("forge calls %v, want one toggle 31 minutes after the checks passed", f.forge.calls)
	}
	if s := step(t, res, "enqueue").Note; !strings.Contains(s, "sat CLEAN outside the queue for 31 min") {
		t.Fatalf("enqueue note %q", s)
	}
}

// #1018's real numbers: armed 12:52:16Z, its last required check passed at
// 12:59:42Z. At 13:22:30 the arming is 30 minutes old but the pull request
// has been CLEAN for under 23, so nothing is toggled.
func TestTheRealTimelineIsNotToggledThirtyMinutesAfterTheArming(t *testing.T) {
	at(t, "13:22:30")
	f := newFixture(t, queueRuleset)
	f.forge.prs = f.prFor(f.head())
	f.forge.answers = []string{timed1018(t, unqueued("CLEAN", "REPPL", armedAt))}
	res, err := f.land(f.options())
	if err != nil {
		t.Fatal(err)
	}
	if f.forge.called("pr", "merge") != 0 {
		t.Fatalf("forge calls %v, want no toggle", f.forge.calls)
	}
	if s := step(t, res, "enqueue").Note; !strings.Contains(s, "after 13:30 UTC") {
		t.Fatalf("enqueue note %q, want the grace to end at 13:30 (12:59:42 + 30, rounded up)", s)
	}
}

// A check the pull request does not require moves no clock; a required
// commit status does, by its createdAt (a status is created in its state).
func TestOnlyRequiredChecksMoveTheGrace(t *testing.T) {
	at(t, "13:30:00")
	f := newFixture(t, queueRuleset)
	f.forge.prs = f.prFor(f.head())
	late := both(unqueued("CLEAN", "REPPL", armedAt), completed("dependency-review", "2026-10-15T13:20:00Z"))
	f.forge.answers = []string{timed1018(t, late), pr1018(t, nil)}
	if _, err := f.land(f.options()); err != nil {
		t.Fatal(err)
	}
	if f.forge.called("pr", "merge", "7", "--disable-auto") != 1 {
		t.Fatalf("forge calls %v, want a toggle: dependency-review is not required", f.forge.calls)
	}

	status := func(pr map[string]any) {
		c := rollupOf(pr)["contexts"].(map[string]any)
		c["nodes"] = append(nodesOf(pr), map[string]any{"context": "ci/legacy", "state": "SUCCESS", "createdAt": "2026-10-15T13:20:00Z", "isRequired": true})
	}
	f = newFixture(t, queueRuleset)
	f.forge.prs = f.prFor(f.head())
	f.forge.answers = []string{timed1018(t, both(unqueued("CLEAN", "REPPL", armedAt), status))}
	if _, err := f.land(f.options()); err != nil {
		t.Fatal(err)
	}
	if f.forge.called("pr", "merge") != 0 {
		t.Fatalf("forge calls %v, want no toggle 10 minutes after a required status passed", f.forge.calls)
	}
}

// S5: a time from the forge later than the local clock never toggles.
func TestAFutureTimeFromTheForgeNeverToggles(t *testing.T) {
	for _, edit := range []func(pr map[string]any){
		unqueued("CLEAN", "REPPL", "2026-10-15T15:00:00Z"),
		both(unqueued("CLEAN", "REPPL", armedAt), completed("check (ubuntu-latest)", "2026-10-15T15:00:00Z")),
	} {
		at(t, "14:00:00")
		f := newFixture(t, queueRuleset)
		f.forge.prs = f.prFor(f.head())
		f.forge.answers = []string{timed1018(t, edit)}
		if _, err := f.land(f.options()); err != nil {
			t.Fatal(err)
		}
		if f.forge.called("pr", "merge") != 0 {
			t.Fatalf("forge calls %v, want no toggle on a time ahead of the local clock", f.forge.calls)
		}
	}
}

// S2: a rollup the forge reports PENDING or EXPECTED with no required check
// pending in the listing (one with no run yet) is still held: never "does
// not hold it".
func TestAPendingRollupWithNoListedRequiredCheckIsStillAHolder(t *testing.T) {
	for _, state := range []string{"PENDING", "EXPECTED"} {
		at(t, "14:12:30")
		f := newFixture(t, queueRuleset)
		f.forge.prs = f.prFor(f.head())
		empty := func(pr map[string]any) {
			r := rollupOf(pr)
			r["state"] = state
			r["contexts"].(map[string]any)["nodes"] = []any{}
		}
		f.forge.answers = []string{
			recorded(t, "pr-state-1020-pending.json", empty),
			recorded(t, "pr-state-1020-pending.json", both(unqueued("BEHIND", "REPPL", "2026-10-15T14:12:20Z"), empty)),
		}
		res, err := f.land(f.options())
		if err != nil {
			t.Fatal(err)
		}
		s := step(t, res, "enqueue").Note
		if !strings.Contains(s, "checks are still pending (the forge's rollup is "+state+")") || strings.Contains(s, "does not hold it") {
			t.Fatalf("rollup %s: enqueue note %q, want the pending rollup named and BEHIND not called no holder", state, s)
		}
	}
}

// The live #1020 answer names its running required checks, as before.
func TestTheLivePendingAnswerNamesItsRunningRequiredChecks(t *testing.T) {
	at(t, "14:12:30")
	f := newFixture(t, queueRuleset)
	f.forge.prs = f.prFor(f.head())
	f.forge.answers = []string{
		recorded(t, "pr-state-1020-pending.json", nil),
		recorded(t, "pr-state-1020-pending.json", unqueued("BEHIND", "REPPL", "2026-10-15T14:12:20Z")),
	}
	res, err := f.land(f.options())
	if err != nil {
		t.Fatal(err)
	}
	s := step(t, res, "enqueue").Note
	if !strings.Contains(s, "attribution (in progress)") || strings.Contains(s, "screenshots") || strings.Contains(s, "rollup") {
		t.Fatalf("enqueue note %q, want the listed required checks named, not the rollup", s)
	}
}

// S3: the listing reads the forge's page maximum, and a listing with more
// pages says land did not read every check.
func TestAListingWithMorePagesSaysSo(t *testing.T) {
	if !strings.Contains(PRStateQuery, "contexts(first:100){pageInfo{hasNextPage}") {
		t.Fatalf("PRStateQuery %q does not read the forge's page maximum and whether more follow", PRStateQuery)
	}
	at(t, "14:12:30")
	f := newFixture(t, queueRuleset)
	f.forge.prs = f.prFor(f.head())
	more := func(pr map[string]any) {
		rollupOf(pr)["contexts"].(map[string]any)["pageInfo"] = map[string]any{"hasNextPage": true}
	}
	f.forge.answers = []string{
		timed1018(t, both(unqueued("BEHIND", "", ""), more)),
		timed1018(t, both(unqueued("BEHIND", "REPPL", "2026-10-15T14:12:20Z"), more)),
	}
	res, err := f.land(f.options())
	if err != nil {
		t.Fatal(err)
	}
	s := step(t, res, "enqueue").Note
	if !strings.Contains(s, "more checks than land read") || strings.Contains(s, "does not hold it") {
		t.Fatalf("enqueue note %q, want the unread checks said and BEHIND not called no holder", s)
	}
}

// S4: on the re-run path, this account's arming on a head that is not the
// one land gated is refused, naming both, as after a fresh arming.
func TestAnOwnArmingOnAnotherHeadIsRefused(t *testing.T) {
	at(t, "13:30:00")
	f := newFixture(t, queueRuleset)
	f.forge.prs = f.prFor(f.head())
	other := strings.Repeat("b", 40)
	f.forge.answers = []string{timed1018(t, both(unqueued("CLEAN", "REPPL", armedAt), func(pr map[string]any) { pr["headRefOid"] = other }))}
	_, err := f.land(f.options())
	wantRefusal(t, err, short(other))
	wantRefusal(t, err, short(f.head()))
	wantRefusal(t, err, "gh pr merge 7 --disable-auto")
	if f.forge.called("pr", "merge") != 0 {
		t.Fatalf("forge calls %v, want nothing toggled or armed", f.forge.calls)
	}
}
