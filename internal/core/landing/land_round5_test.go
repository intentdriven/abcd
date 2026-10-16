package landing

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Round 5: the first real runs of abcd land (PR #1018, 2026-10-15). Armed at
// 12:52:16Z with required checks pending and mergeStateStatus BEHIND; the
// checks went green at 12:59:47Z (CLEAN, armed, unqueued); the forge queued
// it by itself at 13:15:01Z; reviewDecision was null throughout.
//
// testdata/pr-state-*.json are the forge's own answers to PRStateQuery, read
// 2026-10-15T13:33:59Z by a read-only `gh api graphql` of intentdriven/abcd:
// #1018 queued (AWAITING_CHECKS, its 13 real check runs) and #1020
// conflicting. The answer land got at 12:52 was not kept, so the timeline's
// earlier states are the recorded #1018 answer with only the fields that
// moved set back (each test says which), every other byte the forge's.

// recordedAnswer is a recorded answer, decoded, its head replaced by @HEAD@
// so the fake forge reports the fixture's head.
func recordedAnswer(t *testing.T, name string) map[string]any {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	prOf(m)["headRefOid"] = "@HEAD@"
	return m
}

func prOf(m map[string]any) map[string]any {
	return m["data"].(map[string]any)["repository"].(map[string]any)["pullRequest"].(map[string]any)
}

// pr1018 is the recorded #1018 answer with edit applied to its pull request.
func pr1018(t *testing.T, edit func(pr map[string]any)) string {
	t.Helper()
	m := recordedAnswer(t, "pr-state-1018-queued.json")
	if edit != nil {
		edit(prOf(m))
	}
	out, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	return string(out)
}

// unqueued sets the pull request outside the queue, in state st, armed by
// login at enabledAt ("" is not armed).
func unqueued(st, login, enabledAt string) func(pr map[string]any) {
	return func(pr map[string]any) {
		pr["isInMergeQueue"], pr["mergeQueueEntry"], pr["mergeStateStatus"] = false, nil, st
		pr["autoMergeRequest"] = nil
		if login != "" {
			pr["autoMergeRequest"] = map[string]any{"mergeMethod": "MERGE", "enabledAt": enabledAt, "enabledBy": map[string]any{"login": login}}
		}
	}
}

// running sets the named check runs back to in progress, as they were at
// 12:52.
func running(names ...string) func(pr map[string]any) {
	return func(pr map[string]any) {
		for _, n := range pr["statusCheckRollup"].(map[string]any)["contexts"].(map[string]any)["nodes"].([]any) {
			c := n.(map[string]any)
			for _, name := range names {
				if c["name"] == name {
					c["status"], c["conclusion"] = "IN_PROGRESS", nil
				}
			}
		}
	}
}

func both(fs ...func(pr map[string]any)) func(pr map[string]any) {
	return func(pr map[string]any) {
		for _, f := range fs {
			f(pr)
		}
	}
}

// at fixes land's clock at the UTC time hhmmss on 2026-10-15.
func at(t *testing.T, hhmmss string) {
	t.Helper()
	tm, err := time.Parse(time.RFC3339, "2026-10-15T"+hhmmss+"Z")
	if err != nil {
		t.Fatal(err)
	}
	prev := now
	now = func() time.Time { return tm }
	t.Cleanup(func() { now = prev })
}

const armedAt = "2026-10-15T12:52:16Z"

// The recorded answers parse as land reads them.
func TestTheRecordedLiveAnswersParse(t *testing.T) {
	var a prStateAnswer
	raw, _ := os.ReadFile(filepath.Join("testdata", "pr-state-1018-queued.json"))
	if err := json.Unmarshal(raw, &a); err != nil || a.Data.Viewer == nil || a.Data.Viewer.Login != "REPPL" {
		t.Fatalf("decode #1018: %v, %+v", err, a.Data.Viewer)
	}
	s := a.Data.Repository.PullRequest
	if !s.IsInMergeQueue || s.StatusCheckRollup == nil || len(s.StatusCheckRollup.Contexts.Nodes) != 13 || s.ReviewDecision != "" {
		t.Fatalf("#1018 read as %+v", s)
	}
	required := 0
	for _, c := range s.StatusCheckRollup.Contexts.Nodes {
		if c.IsRequired {
			required++
		}
	}
	if required != 9 || len(s.heldChecks()) != 0 {
		t.Fatalf("#1018: %d required, held %v; want 9 required, none held (SKIPPED passes)", required, s.heldChecks())
	}
	raw, _ = os.ReadFile(filepath.Join("testdata", "pr-state-1020-conflicting.json"))
	a = prStateAnswer{}
	if err := json.Unmarshal(raw, &a); err != nil || a.Data.Repository.PullRequest.Mergeable != "CONFLICTING" {
		t.Fatalf("decode #1020: %v", err)
	}
}

// R5-1: required checks still running are named before the merge state, and
// BEHIND in a queue that builds on the default branch is no holder.
func TestPendingRequiredChecksAreNamedBeforeTheMergeState(t *testing.T) {
	at(t, "12:52:30")
	f := newFixture(t, queueRuleset)
	f.forge.prs = f.prFor(f.head())
	checks := running("check (ubuntu-latest)", "gitleaks", "dependency-review")
	f.forge.answers = []string{
		pr1018(t, both(unqueued("BEHIND", "", ""), checks)),
		pr1018(t, both(unqueued("BEHIND", "REPPL", armedAt), checks)),
	}
	res, err := f.land(f.options())
	if err != nil {
		t.Fatal(err)
	}
	s := step(t, res, "enqueue").Note
	for _, want := range []string{"check (ubuntu-latest)", "gitleaks", "BEHIND", "the queue tests it on top of main"} {
		if !strings.Contains(s, want) {
			t.Fatalf("enqueue note %q does not say %q", s, want)
		}
	}
	if strings.Index(s, "check (ubuntu-latest)") > strings.Index(s, "BEHIND") {
		t.Fatalf("enqueue note %q names the merge state before the checks", s)
	}
	if strings.Contains(s, "dependency-review") || strings.Contains(s, "behind the default branch") {
		t.Fatalf("enqueue note %q names a check that is not required, or BEHIND as a holder", s)
	}
}

func TestBehindAloneIsNotAHolder(t *testing.T) {
	at(t, "12:52:30")
	f := newFixture(t, queueRuleset)
	f.forge.prs = f.prFor(f.head())
	f.forge.answers = []string{pr1018(t, unqueued("BEHIND", "", "")), pr1018(t, unqueued("BEHIND", "REPPL", armedAt))}
	res, err := f.land(f.options())
	if err != nil {
		t.Fatal(err)
	}
	s := step(t, res, "enqueue").Note
	if !strings.Contains(s, "the queue tests it on top of main") || strings.Contains(s, "required check") {
		t.Fatalf("enqueue note %q, want BEHIND alone named as no holder", s)
	}
}

func TestAtMostEightHeldChecksAreNamedSanitised(t *testing.T) {
	at(t, "12:52:30")
	f := newFixture(t, queueRuleset)
	f.forge.prs = f.prFor(f.head())
	all := []string{"attribution", "external-review", "gitleaks", "record-lint", "cold-reading-evals", "check (ubuntu-latest)", "check (macos-latest)", "zizmor", "smoke"}
	escaped := func(pr map[string]any) {
		c := pr["statusCheckRollup"].(map[string]any)["contexts"].(map[string]any)["nodes"].([]any)[0].(map[string]any)
		c["name"], c["status"], c["conclusion"] = "attribution\x1b[31m", "COMPLETED", "FAILURE"
	}
	f.forge.answers = []string{
		pr1018(t, both(unqueued("BLOCKED", "", ""), running(all...), escaped)),
		pr1018(t, both(unqueued("BLOCKED", "REPPL", armedAt), running(all...), escaped)),
	}
	res, err := f.land(f.options())
	if err != nil {
		t.Fatal(err)
	}
	s := step(t, res, "enqueue").Note
	if !strings.Contains(s, "and 1 more") || strings.Contains(s, "smoke") || strings.Contains(s, "\x1b") || !strings.Contains(s, "failure") {
		t.Fatalf("enqueue note %q, want 8 of 9 held checks named, sanitised, the failure said", s)
	}
}

// R5-2: the review note is the forge's reviewDecision, never the mirror's
// approval rule alone.
func TestTheReviewNoteIsTheForgesReviewDecision(t *testing.T) {
	for _, c := range []struct {
		decision     any
		want, banned string
	}{
		{nil, "requires no review for this pull request", "approval"},
		{"REVIEW_REQUIRED", "holds it for a person's approval", "requires no review"},
		{"APPROVED", "approved", "holds it"},
	} {
		at(t, "12:52:30")
		f := newFixture(t, queueRuleset)
		f.forge.prs = f.prFor(f.head())
		decide := func(pr map[string]any) { pr["reviewDecision"] = c.decision }
		f.forge.answers = []string{pr1018(t, both(unqueued("BLOCKED", "", ""), decide)), pr1018(t, decide)}
		res, err := f.land(f.options())
		if err != nil {
			t.Fatal(err)
		}
		if s := step(t, res, "enqueue").Note; !strings.Contains(s, c.want) || strings.Contains(s, c.banned) {
			t.Fatalf("reviewDecision %v: enqueue note %q, want %q and not %q", c.decision, s, c.want, c.banned)
		}
	}
}

// R5-3: an arming the forge has not queued within the grace is a normal
// wait, not a stuck one: no toggle, and the time to look again.
func TestAFreshArmingIsNotToggled(t *testing.T) {
	at(t, "12:53:16")
	f := newFixture(t, queueRuleset)
	f.forge.prs = f.prFor(f.head())
	f.forge.answers = []string{pr1018(t, unqueued("CLEAN", "", "")), pr1018(t, unqueued("CLEAN", "REPPL", armedAt))}
	res, err := f.land(f.options())
	if err != nil {
		t.Fatal(err)
	}
	if f.forge.called("pr", "merge", "7", "--disable-auto") != 0 || f.forge.called("pr", "merge", "7", "--auto") != 1 {
		t.Fatalf("forge calls %v, want one arm and no toggle", f.forge.calls)
	}
	s := step(t, res, "enqueue").Note
	for _, want := range []string{"armed 1 min ago", "run abcd land again after 13:23 UTC if it is still not queued"} {
		if !strings.Contains(s, want) {
			t.Fatalf("enqueue note %q does not say %q", s, want)
		}
	}
}

// The real window: CLEAN, armed by this account and unqueued at 13:10, which
// the forge resolved by itself at 13:15.
func TestAnOwnArmingInsideTheGraceIsNotToggledOnARerun(t *testing.T) {
	at(t, "13:10:00")
	f := newFixture(t, queueRuleset)
	f.forge.prs = f.prFor(f.head())
	f.forge.answers = []string{pr1018(t, unqueued("CLEAN", "REPPL", armedAt))}
	res, err := f.land(f.options())
	if err != nil {
		t.Fatal(err)
	}
	if f.forge.called("pr", "merge") != 0 {
		t.Fatalf("forge calls %v, want no toggle inside the grace", f.forge.calls)
	}
	if s := step(t, res, "enqueue").Note; !strings.Contains(s, "armed 17 min ago") || !strings.Contains(s, "after 13:23 UTC") {
		t.Fatalf("enqueue note %q", s)
	}
}

func TestAnOwnArmingPastTheGraceIsToggledOnce(t *testing.T) {
	at(t, "13:30:00")
	f := newFixture(t, queueRuleset)
	f.forge.prs = f.prFor(f.head())
	f.forge.answers = []string{pr1018(t, unqueued("CLEAN", "REPPL", armedAt)), pr1018(t, nil)}
	res, err := f.land(f.options())
	if err != nil {
		t.Fatal(err)
	}
	if f.forge.called("pr", "merge", "7", "--disable-auto") != 1 || f.forge.called("pr", "merge", "7", "--auto", "--merge", "--match-head-commit="+f.head()) != 1 {
		t.Fatalf("forge calls %v, want one toggle pinned to the head", f.forge.calls)
	}
	if s := step(t, res, "enqueue").Note; !strings.Contains(s, "toggled once") || !strings.Contains(s, "in the merge queue") {
		t.Fatalf("enqueue note %q", s)
	}

	// Still unqueued after the toggle: once is all, on this run.
	f = newFixture(t, queueRuleset)
	f.forge.prs = f.prFor(f.head())
	f.forge.answers = []string{pr1018(t, unqueued("CLEAN", "REPPL", armedAt)), pr1018(t, unqueued("CLEAN", "REPPL", "2026-10-15T13:30:00Z"))}
	if _, err := f.land(f.options()); err != nil {
		t.Fatal(err)
	}
	if f.forge.called("pr", "merge", "7", "--disable-auto") != 1 {
		t.Fatalf("forge calls %v, want exactly one toggle", f.forge.calls)
	}
}

func TestAnArmingByAnotherAccountPastTheGraceIsLeftAlone(t *testing.T) {
	at(t, "14:00:00")
	f := newFixture(t, queueRuleset)
	f.forge.prs = f.prFor(f.head())
	f.forge.answers = []string{pr1018(t, unqueued("CLEAN", "someone-else", armedAt))}
	res, err := f.land(f.options())
	if err != nil {
		t.Fatal(err)
	}
	if f.forge.called("pr", "merge") != 0 {
		t.Fatalf("forge calls %v, want another account's arming untouched", f.forge.calls)
	}
	if s := step(t, res, "enqueue").Note; !strings.Contains(s, "leaves") || !strings.Contains(s, "gh pr merge 7 --disable-auto") {
		t.Fatalf("enqueue note %q", s)
	}
}

// m6 on the re-run toggle: a re-arm the forge refuses says the pull request
// is now disarmed and gives the exact command that arms it again.
func TestAFailedRearmSaysThePullRequestIsDisarmed(t *testing.T) {
	at(t, "13:30:00")
	f := newFixture(t, queueRuleset)
	f.forge.prs = f.prFor(f.head())
	f.forge.answers = []string{pr1018(t, unqueued("CLEAN", "REPPL", armedAt))}
	f.forge.failArm = 1
	_, err := f.land(f.options())
	wantRefusal(t, err, "DISARMED")
	wantRefusal(t, err, "gh pr merge 7 --auto --merge --match-head-commit="+f.head())
}
