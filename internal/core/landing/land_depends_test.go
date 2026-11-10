package landing

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

// The enqueue step's dependency check (depends.go): a pull request whose body
// names another in a Depends-On line is armed only once that one has merged
// (itd-2610170634394993, spc-2610230758581403 step 2a).

// dependsAt is the clock the tests judge by: an hour after dependsCreated, so
// a wait's deadline is 2026-11-11T08:00:00Z.
var dependsAt = time.Date(2026, 11, 10, 9, 0, 0, 0, time.UTC)

// dependsDeadline is the deadline of a wait that began at dependsCreated.
var dependsDeadline = time.Date(2026, 11, 11, 8, 0, 0, 0, time.UTC)

// dependsFixture is a fixture whose pull request #7 is ready to arm through a
// merge queue, its body body, its dependencies' answers in deps, judged at
// clock.
func dependsFixture(t *testing.T, body string, deps map[int]string, clock time.Time) (*fixture, Options) {
	t.Helper()
	f := newFixture(t, queueRuleset)
	f.forge.prs = f.prFor(f.head())
	f.forge.views = []string{viewBlocked, viewQueued}
	f.forge.depends = map[int]string{7: dependsPRAnswer(7, "OPEN", false, "main", dependsCreated, body)}
	for n, a := range deps {
		f.forge.depends[n] = a
	}
	o := f.options()
	o.Now = func() time.Time { return clock }
	return f, o
}

func openDep(n int, body string) string {
	return dependsPRAnswer(n, "OPEN", false, "main", "2026-11-09T08:00:00Z", body)
}

// dependsReads is the pull request numbers the check read, in order.
func (f *fakeForge) dependsReads() []string {
	var out []string
	for _, c := range f.calls {
		if len(c) > 3 && c[3] == "query="+dependsQuery {
			out = append(out, strings.TrimPrefix(c[len(c)-1], "n="))
		}
	}
	return out
}

// noArming fails the test when the landing armed, toggled or withdrew
// anything.
func noArming(t *testing.T, f *fixture) {
	t.Helper()
	if n := f.forge.called("pr", "merge"); n != 0 {
		t.Fatalf("forge calls %v: want nothing armed, toggled or withdrawn", f.forge.calls)
	}
}

// A body with no Depends-On line lands as it did before the check: one read
// of the pull request itself, then the arming and its confirmation.
func TestLandArmsAPullRequestWithNoDependsOnLineAsBefore(t *testing.T) {
	for name, body := range map[string]string{
		"an empty body": "",
		"prose":         "## Summary\n\nThe order depends on the queue.\n\nRefs: iss-1",
	} {
		t.Run(name, func(t *testing.T) {
			f, o := dependsFixture(t, body, nil, dependsAt)
			res, err := f.land(o)
			if err != nil {
				t.Fatal(err)
			}
			if got := f.forge.dependsReads(); len(got) != 1 || got[0] != "7" {
				t.Fatalf("dependency reads %v, want #7 alone", got)
			}
			if f.forge.called("pr", "merge", "7", "--auto", "--merge", "--match-head-commit="+f.head()) != 1 || f.forge.called("pr", "merge", "7", "--disable-auto") != 0 {
				t.Fatalf("forge calls %v, want one arm pinned to the head and no toggle", f.forge.calls)
			}
			// The one read comes after the first state query and before the
			// arming, and the calls around it are the ones the path made
			// before the check: state, arm, state.
			var order []string
			for _, c := range f.forge.calls {
				switch {
				case len(c) > 3 && c[3] == "query="+dependsQuery:
					order = append(order, "depends")
				case len(c) > 3 && c[3] == "query="+PRStateQuery:
					order = append(order, "state")
				case len(c) > 1 && c[0] == "pr" && c[1] == "merge":
					order = append(order, "merge")
				}
			}
			if got := strings.Join(order, " "); got != "state depends merge state" {
				t.Fatalf("calls in order %q, want %q", got, "state depends merge state")
			}
			if s := step(t, res, "enqueue"); s.Status != StatusDone || !strings.HasSuffix(s.Note, "; it is in the merge queue") {
				t.Fatalf("enqueue step %+v, want armed and confirmed queued", s)
			}
		})
	}
}

// A dependency not yet merged, open or queued, stops the step with nothing
// armed, naming it and the wait's deadline, up to one nanosecond before it.
func TestLandWaitsOnAnUnmergedDependency(t *testing.T) {
	one := "pull request #7 depends on #5, which has not merged, so nothing was armed: run abcd land again after it merges; the wait is refused from 2026-11-11T08:00:00Z, 24 hours after #7 was opened"
	for _, c := range []struct {
		name  string
		body  string
		deps  map[int]string
		clock time.Time
		want  string
	}{
		{"open", "Depends-On: #5", map[int]string{5: openDep(5, "")}, dependsAt, one},
		{"queued", "Depends-On: #5", map[int]string{5: dependsPRAnswer(5, "OPEN", true, "main", dependsCreated, "")}, dependsAt, one},
		{"a nanosecond before the deadline", "Depends-On: #5", map[int]string{5: openDep(5, "")}, dependsDeadline.Add(-time.Nanosecond), one},
		{"two, one merged", "Depends-On: #4, #5, #6", map[int]string{4: dependsPRAnswer(4, "MERGED", false, "main", dependsCreated, ""), 5: openDep(5, ""), 6: openDep(6, "")}, dependsAt,
			"pull request #7 depends on #5, #6, which have not merged, so nothing was armed: run abcd land again after they merge; the wait is refused from 2026-11-11T08:00:00Z, 24 hours after #7 was opened"},
	} {
		t.Run(c.name, func(t *testing.T) {
			f, o := dependsFixture(t, c.body, c.deps, c.clock)
			res, err := f.land(o)
			if err != nil {
				t.Fatal(err)
			}
			noArming(t, f)
			if s := step(t, res, "enqueue"); s.Status != StatusStopped || s.Note != c.want {
				t.Fatalf("enqueue step %+v\n want stopped, saying %q", s, c.want)
			}
		})
	}
}

// Every dependency merged: the step arms as it would with no line.
func TestLandArmsOnceEveryDependencyHasMerged(t *testing.T) {
	f, o := dependsFixture(t, "Depends-On: #5\nDepends-On: #6", map[int]string{
		5: dependsPRAnswer(5, "MERGED", false, "main", dependsCreated, "Depends-On: #4"),
		6: dependsPRAnswer(6, "MERGED", false, "main", dependsCreated, "depends-on #oops"),
	}, dependsAt)
	res, err := f.land(o)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(f.forge.dependsReads(), " "); got != "7 5 6" {
		t.Fatalf("dependency reads %q, want 7 5 6: a merged dependency's own lines are not read", got)
	}
	if f.forge.called("pr", "merge", "7", "--auto", "--merge", "--match-head-commit="+f.head()) != 1 {
		t.Fatalf("forge calls %v, want one arm", f.forge.calls)
	}
	if s := step(t, res, "enqueue"); s.Status != StatusDone {
		t.Fatalf("enqueue step %+v, want armed", s)
	}
}

// What refuses the landing, exit 2, nothing armed, each with the whole text:
// the pass's at-once refusals and its deadline, and what only the forge can
// know, refused before the pass.
func TestLandRefusesWhatNoWaitCanRelease(t *testing.T) {
	reword := "; reword #7's Depends-On line (gh pr edit 7 --body-file <file>) so it names only open or merged pull requests into main with no cycle among them, or remove it, then run abcd land again"
	resolve := func(n string) string {
		return "; take #" + n + " out of #7's Depends-On line (gh pr edit 7 --body-file <file>), or name only pull requests into main, then run abcd land again"
	}
	settle := func(n string) string {
		return "; settle what the forge reports for #" + n + " (gh pr view " + n + "), then run abcd land again"
	}
	const prefix = "pull request #7's dependencies refuse it, so nothing was armed: "
	const bad = `looks like a Depends-On line but is not in the accepted form "Depends-On: #N, #N"; reword that line, or write it in that form`
	notFound := `{"data":{"repository":{"pullRequest":null}},"errors":[{"type":"NOT_FOUND","path":["repository","pullRequest"],"message":"Could not resolve to a PullRequest with the number of 5."}]}`
	for _, c := range []struct {
		name  string
		body  string
		deps  map[int]string
		clock time.Time
		want  string
	}{
		{"closed unmerged", "Depends-On: #5", map[int]string{5: dependsPRAnswer(5, "CLOSED", false, "main", dependsCreated, "")}, dependsAt,
			prefix + "refused at once, since no wait can release it: depends on #5, closed unmerged" + reword},
		{"a cycle", "Depends-On: #5", map[int]string{5: openDep(5, "Depends-On: #7")}, dependsAt,
			prefix + "refused at once, since no wait can release it: is on or reaches a cycle of unmerged changes among #5, #7" + reword},
		{"itself", "Depends-On: #7", nil, dependsAt,
			prefix + "refused at once, since no wait can release it: is on or reaches a cycle of unmerged changes among #7" + reword},
		{"at the deadline", "Depends-On: #5", map[int]string{5: openDep(5, "")}, dependsDeadline,
			prefix + "waited on #5 for the 24h0m0s limit since 2026-11-10T08:00:00Z; deadline 2026-11-11T08:00:00Z reached; merge #5 first and run abcd land again, or remove the Depends-On line from #7's body (gh pr edit 7 --body-file <file>) if #7 need not wait for it"},
		{"a start later than the clock", "Depends-On: #5", map[int]string{5: openDep(5, "")}, time.Date(2026, 11, 10, 7, 59, 59, 0, time.UTC),
			prefix + "waits on #5 from a recorded start of 2026-11-10T08:00:00Z, later than the records' time of 2026-11-10T07:59:59Z, so no deadline is derived from it; check this computer's clock against the forge's, then run abcd land again"},
		{"no pull request", "Depends-On: #5", map[int]string{5: notFound}, dependsAt,
			"the forge has no pull request #5 in o/r, so it cannot be resolved as a dependency of #7, and nothing was armed" + resolve("5")},
		{"no pull request, as the forge client fails on it", "Depends-On: #5", map[int]string{5: "!gh api graphql: exit status 1 (gh: Could not resolve to a PullRequest with the number of 5.)"}, dependsAt,
			"the forge has no pull request #5 in o/r, so it cannot be resolved as a dependency of #7, and nothing was armed" + resolve("5")},
		{"the client's message for another number", "Depends-On: #5", map[int]string{5: "!gh api graphql: exit status 1 (gh: Could not resolve to a PullRequest with the number of 50.)"}, dependsAt,
			"the forge could not be asked for pull request #5, so land cannot judge #7's dependencies, and nothing was armed: gh api graphql: exit status 1 (gh: Could not resolve to a PullRequest with the number of 50.)" + settle("5")},
		{"another base", "Depends-On: #5", map[int]string{5: dependsPRAnswer(5, "OPEN", false, "release", dependsCreated, "")}, dependsAt,
			"pull request #5 merges into another branch than the default branch main, so it cannot be resolved as a dependency of #7, and nothing was armed" + resolve("5")},
		{"an unknown state", "Depends-On: #5", map[int]string{5: dependsPRAnswer(5, "LOCKED", false, "main", dependsCreated, "")}, dependsAt,
			`the forge reports pull request #5 in state "LOCKED", which land cannot read, so it cannot judge #7's dependencies, and nothing was armed` + settle("5")},
		{"a lower-case state is no known one", "Depends-On: #5", map[int]string{5: dependsPRAnswer(5, "merged", false, "main", dependsCreated, "")}, dependsAt,
			`the forge reports pull request #5 in state "merged", which land cannot read, so it cannot judge #7's dependencies, and nothing was armed` + settle("5")},
		{"a state of 32 runes", "Depends-On: #5", map[int]string{5: dependsPRAnswer(5, strings.Repeat("é", 32), false, "main", dependsCreated, "")}, dependsAt,
			`the forge reports pull request #5 in state "` + strings.Repeat("é", 32) + `", which land cannot read, so it cannot judge #7's dependencies, and nothing was armed` + settle("5")},
		{"a state of 33 runes", "Depends-On: #5", map[int]string{5: dependsPRAnswer(5, strings.Repeat("é", 33), false, "main", dependsCreated, "")}, dependsAt,
			`the forge reports pull request #5 in state "` + strings.Repeat("é", 32) + `", cut to its first 32 runes, which land cannot read, so it cannot judge #7's dependencies, and nothing was armed` + settle("5")},
		{"another number answered", "Depends-On: #5", map[int]string{5: openDep(6, "")}, dependsAt,
			"the forge answered pull request #6 when asked for #5, so land cannot judge #7's dependencies, and nothing was armed" + settle("5")},
		{"an answer with errors", "Depends-On: #5", map[int]string{5: `{"data":{"repository":{"pullRequest":null}},"errors":[{"type":"FORBIDDEN","message":"Resource not accessible"}]}`}, dependsAt,
			"the forge answered the query for pull request #5 with errors, so land cannot judge #7's dependencies, and nothing was armed: Resource not accessible" + settle("5")},
		{"a pull request beside a not-found error", "Depends-On: #5", map[int]string{5: `{"data":{"repository":{"pullRequest":{"number":5,"state":"MERGED","isInMergeQueue":false,"baseRefName":"main","createdAt":"2026-11-10T08:00:00Z","body":""}}},"errors":[{"type":"NOT_FOUND","message":"x"}]}`}, dependsAt,
			"the forge answered the query for pull request #5 with errors, so land cannot judge #7's dependencies, and nothing was armed: x" + settle("5")},
		{"an answer that does not parse", "Depends-On: #5", map[int]string{5: `{"data":`}, dependsAt,
			"the forge's answer for pull request #5 does not parse, so land cannot judge #7's dependencies, and nothing was armed" + settle("5")},
		{"no repository", "Depends-On: #5", map[int]string{5: `{"data":{"repository":null}}`}, dependsAt,
			"the forge answered no repository o/r for pull request #5, so land cannot judge #7's dependencies, and nothing was armed" + settle("5")},
		{"its own body in a wrong shape", "Title\n\n  depends-on #5 CANARY-7Q", nil, dependsAt,
			"pull request #7's body cannot be read for its dependencies, so nothing was armed: line 3 " + bad + "; edit #7's body on the forge (gh pr edit 7 --body-file <file>), then run abcd land again"},
		{"its own body over the bound", strings.Repeat("x", 65537), nil, dependsAt,
			"pull request #7's body cannot be read for its dependencies, so nothing was armed: the body is 65537 bytes, above the most of 65536; edit #7's body on the forge (gh pr edit 7 --body-file <file>), then run abcd land again"},
		{"a dependency's body in a wrong shape", "Depends-On: #5", map[int]string{5: openDep(5, "Depends-On: #6 CANARY-5Q")}, dependsAt,
			"pull request #5, which #7 depends on, has a body that cannot be read for its dependencies, so nothing was armed: line 1 " + bad + "; edit #5's body on the forge (gh pr edit 5 --body-file <file>), or take #5 out of #7's Depends-On line, then run abcd land again"},
		{"its own creation time unreadable", "Depends-On: #5", map[int]string{7: dependsPRAnswer(7, "OPEN", false, "main", "yesterday", "Depends-On: #5"), 5: openDep(5, "")}, dependsAt,
			"the forge gives pull request #7 no creation time land can read, so no deadline can be derived for its wait on its dependencies, and nothing was armed" + settle("7")},
		{"its own pull request merged since it was listed", "", map[int]string{7: dependsPRAnswer(7, "MERGED", false, "main", dependsCreated, "")}, dependsAt,
			"the forge's answer for pull request #7 no longer reads it as open into main and outside the merge queue, so land cannot judge its dependencies, and nothing was armed; run abcd land again"},
		{"its own pull request into another base", "", map[int]string{7: dependsPRAnswer(7, "OPEN", false, "release", dependsCreated, "")}, dependsAt,
			"the forge's answer for pull request #7 no longer reads it as open into main and outside the merge queue, so land cannot judge its dependencies, and nothing was armed; run abcd land again"},
	} {
		t.Run(c.name, func(t *testing.T) {
			f, o := dependsFixture(t, c.body, c.deps, c.clock)
			_, err := f.land(o)
			var r *Refusal
			if !errors.As(err, &r) || r.Held {
				t.Fatalf("err = %v, want a refusal (exit 2)", err)
			}
			if r.Error() != c.want {
				t.Fatalf("refusal\n got  %q\n want %q", r.Error(), c.want)
			}
			if strings.Contains(r.Error(), "CANARY") {
				t.Fatalf("refusal %q quotes the body", r.Error())
			}
			noArming(t, f)
		})
	}
}

// chain is #7 depending on #8, #8 on #9, and so on to #last, which has merged.
func chain(last int) map[int]string {
	deps := map[int]string{}
	for n := 8; n < last; n++ {
		deps[n] = openDep(n, fmt.Sprintf("Depends-On: #%d", n+1))
	}
	deps[last] = dependsPRAnswer(last, "MERGED", false, "main", dependsCreated, "")
	return deps
}

// The check reads at most 25 pull requests, its own included: a chain of 25
// waits, one of 26 is refused as unresolvable, naming the bound.
func TestLandReadsAtMostTwentyFivePullRequests(t *testing.T) {
	f, o := dependsFixture(t, "Depends-On: #8", chain(31), dependsAt)
	res, err := f.land(o)
	if err != nil {
		t.Fatal(err)
	}
	if n := len(f.forge.dependsReads()); n != 25 {
		t.Fatalf("%d reads, want 25", n)
	}
	if s := step(t, res, "enqueue"); s.Status != StatusStopped || !strings.HasPrefix(s.Note, "pull request #7 depends on #8, which has not merged") {
		t.Fatalf("enqueue step %+v, want a wait on #8", s)
	}

	f, o = dependsFixture(t, "Depends-On: #8", chain(32), dependsAt)
	_, err = f.land(o)
	var r *Refusal
	want := "pull request #7's dependencies reach more than 25 pull requests, past what land reads, so they cannot be resolved, and nothing was armed; name fewer pull requests in the Depends-On lines #7 reaches (gh pr edit 7 --body-file <file>), then run abcd land again"
	if !errors.As(err, &r) || r.Error() != want {
		t.Fatalf("err = %v\n want %q", err, want)
	}
	if n := len(f.forge.dependsReads()); n != 25 {
		t.Fatalf("%d reads, want 25 before the refusal", n)
	}
	noArming(t, f)
}

// A forge that cannot be asked refuses, nothing armed.
func TestLandRefusesWhenTheForgeCannotBeAskedForDependencies(t *testing.T) {
	f, o := dependsFixture(t, "", nil, dependsAt)
	f.forge.dependsErr = errors.New("gh: the network is down")
	_, err := f.land(o)
	want := "the forge could not be asked for pull request #7, so land cannot judge #7's dependencies, and nothing was armed: gh: the network is down; settle what the forge reports for #7 (gh pr view 7), then run abcd land again"
	var r *Refusal
	if !errors.As(err, &r) || r.Error() != want {
		t.Fatalf("err = %v\n want %q", err, want)
	}
	noArming(t, f)
}

// A pull request already in the merge queue, or armed by another account, is
// reported as before: its dependencies are not read.
func TestLandReportsAQueuedOrOtherArmedPullRequestWithoutReadingDependencies(t *testing.T) {
	for name, view := range map[string]string{"queued": viewQueued, "armed by another account": viewArmedByOther} {
		t.Run(name, func(t *testing.T) {
			f, o := dependsFixture(t, "Depends-On: #5", map[int]string{5: openDep(5, "")}, dependsAt)
			f.forge.views = []string{view}
			res, err := f.land(o)
			if err != nil {
				t.Fatal(err)
			}
			if got := f.forge.dependsReads(); len(got) != 0 {
				t.Fatalf("dependency reads %v, want none", got)
			}
			noArming(t, f)
			if s := step(t, res, "enqueue"); s.Status != StatusDone {
				t.Fatalf("enqueue step %+v, want reported as done", s)
			}
		})
	}
}

// ownArmed is the forge's state answer for #7 armed by this account (alice),
// CLEAN outside the queue.
const ownArmed = `{"data":{"viewer":{"login":"alice"},"repository":{"pullRequest":{"mergeable":"MERGEABLE","mergeStateStatus":"CLEAN","isInMergeQueue":false,"autoMergeRequest":{"mergeMethod":"MERGE","enabledAt":"2026-11-10T08:30:00Z","enabledBy":{"login":"alice"}},"headRefOid":"@HEAD@"}}}}`

// This account's arming already in place is left as it is on a wait or a
// refusal, and the note or the refusal says so and how to withdraw it; nothing
// is toggled.
func TestLandLeavesThisAccountsArmingOnAWaitAndSaysSo(t *testing.T) {
	f, o := dependsFixture(t, "Depends-On: #5", map[int]string{5: openDep(5, "")}, dependsAt)
	f.forge.views = nil
	f.forge.answer = ownArmed
	res, err := f.land(o)
	if err != nil {
		t.Fatal(err)
	}
	noArming(t, f)
	want := "pull request #7 depends on #5, which has not merged, so nothing was armed: run abcd land again after it merges; the wait is refused from 2026-11-11T08:00:00Z, 24 hours after #7 was opened; this account's arming already in place is left as it is: withdraw it (gh pr merge 7 --disable-auto) if #7 must not merge first"
	if s := step(t, res, "enqueue"); s.Status != StatusStopped || s.Note != want {
		t.Fatalf("enqueue step %+v\n want stopped, saying %q", s, want)
	}

	// And on a refusal: the arming is left, and the refusal says so.
	f, o = dependsFixture(t, "Depends-On: #5", map[int]string{5: dependsPRAnswer(5, "CLOSED", false, "main", dependsCreated, "")}, dependsAt)
	f.forge.views = nil
	f.forge.answer = ownArmed
	_, err = f.land(o)
	var r *Refusal
	want = "pull request #7's dependencies refuse it, so nothing was armed: refused at once, since no wait can release it: depends on #5, closed unmerged; this account's arming already in place is left as it is: withdraw it (gh pr merge 7 --disable-auto) if #7 must not merge first; reword #7's Depends-On line (gh pr edit 7 --body-file <file>) so it names only open or merged pull requests into main with no cycle among them, or remove it, then run abcd land again"
	if !errors.As(err, &r) || r.Error() != want {
		t.Fatalf("err = %v\n want %q", err, want)
	}
	noArming(t, f)
}

// The wait limit is the most the pass allows, pinned.
func TestDependsWaitLimitIsADay(t *testing.T) {
	if dependsWaitLimit != 24*time.Hour || maxDependsReads != 25 || maxForgeStateRunes != 32 {
		t.Fatalf("dependsWaitLimit %s, maxDependsReads %d, maxForgeStateRunes %d; want 24h, 25 and 32", dependsWaitLimit, maxDependsReads, maxForgeStateRunes)
	}
}
