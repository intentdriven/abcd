package loop

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// A drain lane merges origin's default branch in its sync (spc-2610161044461448
// step 13; iss-2610161015410527; dcn-2610161040384068): an issue lane whose
// default branch moved since its base merges it before its records and its
// push, the merge head judged by a fresh round; a conflict confined to files
// origin's default branch declares linguist-generated goes back to an
// implementer, any other conflict hands the lane back naming the files with
// nothing pushed, and a pull request that starts to conflict while it is
// armed or waiting re-enters the sync rather than being refused for a person.

// genAttributes declares gen.txt generated, as abcd declares its CLI
// reference.
const genAttributes = "gen.txt linguist-generated\n"

// syncFixture is drainLandFixture with gen.txt and plain.txt at the base, the
// former declared generated on origin's default branch, and the drain's
// documentation lane opened.
func syncFixture(t *testing.T) *landFixture {
	t.Helper()
	return syncFixtureWith(t, queueRuleset("MERGE"))
}

// syncFixtureWith is syncFixture with the ruleset mirror given.
func syncFixtureWith(t *testing.T, ruleset string) *landFixture {
	t.Helper()
	f := drainLandFixtureWith(t, ruleset)
	f.repo.Write(".gitattributes", genAttributes)
	f.repo.Write("gen.txt", "generated at the base\n")
	f.repo.Write("plain.txt", "written at the base\n")
	f.repo.Commit("a generated file and a plain one")
	f.repo.Git("push", "-q", "origin", "main")
	f.repo.Git("fetch", "-q", "origin")
	first := drainToLane(t, f.repo, Options{}, DrainOptions{})
	if first.Start == nil || first.Lane.Issue != secondIssue {
		t.Fatalf("the drain opens the documentation issue's lane: %+v", first)
	}
	f.runID, f.issue = first.Start.RunID, secondIssue
	return f
}

// passToLand drives the lane through its implementer, who changes names, and
// a passing round, to its landing's first step.
func (f *landFixture) passToLand(t *testing.T, names ...string) Lane {
	t.Helper()
	repo := f.repo
	stepTo(t, repo, f.runID, f.stages, StageImplement)
	if res, err := advance(repo.Root(), f.runID, f.stages, Options{}); err != nil || res.Awaiting == nil {
		t.Fatalf("the implement stage awaits an implementer: %+v %v", res, err)
	}
	l := currentLane(t, repo, f.runID)
	dir := filepath.Join(repo.Root(), filepath.FromSlash(RunRelDir), f.runID, l.ID)
	widenAsImplementer(t, repo, l, names...)
	sha := commitIn(t, repo, l, names...)
	rc := goodReceipt(t, f.runID, l, dir, sha)
	rc.Model = "claude-test-5"
	rc.Resolves = []Resolution{{Issue: f.issue, Commit: sha, Note: "the page names the flag", Impact: "fix",
		Grounds: "pursued: the page names the flag; shown wrong if it still does not"}}
	if _, err := Receipt(repo.Root(), f.runID, writeReceipt(t, dir, rc), f.stages, Options{}); err != nil {
		t.Fatal(err)
	}
	passRound(t, repo, f.runID, f.stages, RoleRuthless, RoleSecurity)
	return stepTo(t, repo, f.runID, f.stages, StageLand)
}

// moveMain lands another account's change of names on origin's default
// branch, as a merge there does, and returns the new tip.
func (f *landFixture) moveMain(t *testing.T, content string, names ...string) string {
	t.Helper()
	for _, name := range names {
		f.repo.Write(name, content)
	}
	f.repo.Commit("another account's change of " + strings.Join(names, ", "))
	f.repo.Git("push", "-q", "origin", "main")
	return strings.TrimSpace(f.repo.Git("rev-parse", "HEAD"))
}

// TestAnIssueLaneMergesTheMovedDefaultBranchInItsSync: origin's default
// branch moved while the lane was validated, so its landing's first step
// fetches it and merges it into the lane's branch with a merge commit whose
// parents are the judged head and the default branch's tip, and a fresh round
// judges that head before anything is recorded or pushed.
func TestAnIssueLaneMergesTheMovedDefaultBranchInItsSync(t *testing.T) {
	f := syncFixture(t)
	judged := f.passToLand(t, "fix.txt").HeadSHA
	tip := f.moveMain(t, "another account's file\n", "other.txt")

	f.step(t)
	l := currentLane(t, f.repo, f.runID)
	if l.Stage != StageValidate || len(l.Syncs) != 1 || l.Landing != nil {
		t.Fatalf("the sync merges the default branch and sends the lane to a fresh round: %+v", l)
	}
	s := l.Syncs[0]
	if s.Merged != tip || s.Conflicted || s.Head != l.HeadSHA || len(s.Siblings) != 0 {
		t.Fatalf("the sync names the default branch's tip it merged and the head it made: %+v (tip %s)", s, tip)
	}
	if parents := strings.Fields(f.repo.Git("rev-list", "--parents", "-n", "1", l.HeadSHA)); len(parents) != 3 || parents[1] != judged || parents[2] != tip {
		t.Fatalf("the merge commit's parents are the judged head and the default branch's tip: %q", parents)
	}
	if !l.syncRoundDue() {
		t.Fatalf("a fresh round judges the merge head: %+v", l)
	}
	if f.remoteBranch(t, l.Branch) != "" {
		t.Fatal("nothing is pushed before the fresh round passes")
	}

	passRound(t, f.repo, f.runID, f.stages, RoleRuthless, RoleSecurity)
	stepTo(t, f.repo, f.runID, f.stages, StageLand)
	f.step(t)
	if l := currentLane(t, f.repo, f.runID); l.Landing == nil || len(l.Syncs) != 1 {
		t.Fatalf("with the default branch merged, the landing goes on without a second sync: %+v", l)
	}
}

// TestAConflictOnADeclaredGeneratedFileGoesToTheImplementer: the default
// branch changed gen.txt, which origin declares generated, and so did the
// lane; the merge is aborted with the branch unchanged, and the conflict goes
// to an implementer with a brief that says the file is generated, nothing
// pushed.
func TestAConflictOnADeclaredGeneratedFileGoesToTheImplementer(t *testing.T) {
	f := syncFixture(t)
	judged := f.passToLand(t, "fix.txt", "gen.txt").HeadSHA
	f.moveMain(t, "generated again on main\n", "gen.txt")

	f.step(t)
	l := currentLane(t, f.repo, f.runID)
	if len(l.Syncs) != 1 || !l.Syncs[0].Conflicted || !slices.Equal(l.Syncs[0].Paths, []string{"gen.txt"}) || l.HeadSHA != judged {
		t.Fatalf("the conflict on the generated file is recorded, the branch unchanged: %+v", l)
	}
	if l.Stage != StageValidate || l.pendingSync() == nil {
		t.Fatalf("the conflict goes to an implementer: %+v", l)
	}
	brief, err := os.ReadFile(filepath.Join(f.repo.Root(), filepath.FromSlash(l.Syncs[0].Brief)))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(brief), "generated") || !strings.Contains(string(brief), "gen.txt") {
		t.Fatalf("the brief names the generated file and says to regenerate it:\n%s", brief)
	}
	if f.remoteBranch(t, l.Branch) != "" {
		t.Fatal("nothing is pushed")
	}
}

// TestARegeneratedFileIsPushedAsAPullRequestThatMergesCleanly is
// iss-2610252224311175: the default branch moves gen.txt, which origin
// declares generated, under a live lane that also changed it; the sync's
// conflict goes to a fresh implementer, who merges the default branch's tip
// and commits the regenerated file in the merge; once a fresh round judges
// that head, the lane's records, push, pull request and arming follow, and
// the pushed head holds the default branch's tip, so the pull request merges
// into it cleanly, as a fast-forward, carrying the regenerated file.
func TestARegeneratedFileIsPushedAsAPullRequestThatMergesCleanly(t *testing.T) {
	f := syncFixture(t)
	judged := f.passToLand(t, "fix.txt", "gen.txt").HeadSHA
	tip := f.moveMain(t, "generated again on main\n", "gen.txt")

	// The sync conflicts on the generated file and goes to an implementer.
	f.step(t)
	res := f.step(t)
	l := currentLane(t, f.repo, f.runID)
	s := l.pendingSync()
	if s == nil || s.Merged != tip || !slices.Equal(s.Paths, []string{"gen.txt"}) || l.HeadSHA != judged {
		t.Fatalf("the generated file's conflict waits on an implementer, the branch at the judged head: %+v", l)
	}
	if res.Awaiting == nil || res.Awaiting.Role != RoleImplementer || res.Awaiting.Receipt != s.Receipt {
		t.Fatalf("a fresh implementer is handed the sync brief: %+v", res)
	}

	// The implementer merges the default branch's tip, regenerates gen.txt
	// over the conflict, and commits the merge.
	const regenerated = "regenerated over both sides\n"
	_ = gitErr(f.repo, "-C", l.Worktree, "merge", "--no-edit", s.Merged)
	if err := os.WriteFile(filepath.Join(l.Worktree, "gen.txt"), []byte(regenerated), 0o600); err != nil {
		t.Fatal(err)
	}
	f.repo.Git("-C", l.Worktree, "add", "--", "gen.txt")
	f.repo.Git("-C", l.Worktree, "commit", "-q", "--no-edit")
	merge := strings.TrimSpace(f.repo.Git("-C", l.Worktree, "rev-parse", "HEAD"))
	if parents := strings.Fields(f.repo.Git("rev-list", "--parents", "-n", "1", merge)); len(parents) != 3 || parents[1] != judged || parents[2] != tip {
		t.Fatalf("the implementer's merge commit joins the judged head and the default branch's tip: %q", parents)
	}

	laneDir := filepath.Join(f.repo.Root(), filepath.FromSlash(RunRelDir), f.runID, l.ID)
	receipt := filepath.Join(f.repo.Root(), filepath.FromSlash(res.Awaiting.Receipt))
	rel, err := filepath.Rel(laneDir, filepath.Dir(receipt))
	if err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{ReportFileName: "regenerated gen.txt in the merge\n", DoDFileName: "ok\n"} {
		if err := os.WriteFile(filepath.Join(laneDir, rel, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	rc := LaneReceipt{SchemaVersion: ReceiptSchemaVersion, RunID: f.runID, Lane: l.ID, Branch: l.Branch, Commits: []string{merge},
		DefinitionOfDone: &DoDRun{Command: "make check", ExitCode: zero(), Output: filepath.ToSlash(filepath.Join(rel, DoDFileName))},
		Report:           filepath.ToSlash(filepath.Join(rel, ReportFileName)), Model: "claude-test-5"}
	data, err := json.Marshal(rc)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(receipt, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Receipt(f.repo.Root(), f.runID, receipt, f.stages, Options{}); err != nil {
		t.Fatalf("the implementer's receipt carrying the merged sha verifies: %v", err)
	}
	if l := currentLane(t, f.repo, f.runID); l.HeadSHA != merge || l.pendingSync() != nil || !l.syncRoundDue() {
		t.Fatalf("the resolved sync names the merge head, and a fresh round is due over it: %+v", l)
	}

	// A fresh round judges the merge head; the landing goes on without a
	// second sync, through its records, push, pull request and arming.
	passRound(t, f.repo, f.runID, f.stages, RoleRuthless, RoleSecurity)
	stepTo(t, f.repo, f.runID, f.stages, StageLand)
	for range 8 {
		f.step(t)
		l = currentLane(t, f.repo, f.runID)
		if l.Landing != nil && l.Landing.Pushed == "" && l.Landing.RecordsDone {
			preflighted(t, l, l.HeadSHA)
		}
		if l.Landing != nil && l.Landing.Armed {
			break
		}
	}
	l = currentLane(t, f.repo, f.runID)
	if l.Landing == nil || !l.Landing.Armed || l.PR != 7 || len(l.Syncs) != 1 {
		t.Fatalf("the merge head's pull request is opened and armed, with no second sync: %+v %+v", l.Landing, l.Syncs)
	}
	pushed := l.Landing.Pushed
	if got := f.remoteBranch(t, l.Branch); got != pushed || pushed != l.HeadSHA {
		t.Fatalf("origin holds the lane's head: %s, pushed %s, head %s", got, pushed, l.HeadSHA)
	}
	if on, err := gitutil.IsAncestor(f.repo.Root(), merge, pushed); err != nil || !on {
		t.Fatalf("the pushed head carries the implementer's merge %s: %v", merge, err)
	}
	if !strings.Contains(f.ghLog(t), "pr merge 7 --auto --merge --match-head-commit="+pushed) {
		t.Fatalf("the pull request is armed at the pushed head:\n%s", f.ghLog(t))
	}

	// The pull request merges cleanly: origin's default branch, unmoved since
	// the sync, is an ancestor of the pushed head, so the merge is a
	// fast-forward that brings the regenerated file.
	f.repo.Git("fetch", "-q", "origin")
	onMain := strings.TrimSpace(f.repo.Git("rev-parse", "refs/remotes/origin/main"))
	if onMain != tip {
		t.Fatalf("origin's default branch has not moved since the sync: %s, want %s", onMain, tip)
	}
	if on, err := gitutil.IsAncestor(f.repo.Root(), onMain, pushed); err != nil || !on {
		t.Fatalf("the pull request's head holds the default branch's tip %s, so it merges without a conflict: %v", onMain, err)
	}
	// The merge, as the queue makes it: a push of the pushed head to origin's
	// default branch, which git refuses unless it is a fast-forward.
	f.merged(t, pushed)
	if got := f.repo.Git("--git-dir", f.bare, "show", "main:gen.txt"); got+"\n" != regenerated && got != regenerated {
		t.Fatalf("the default branch carries the regenerated file: %q", got)
	}
	if res := f.step(t); res.PerformedStage != StageLand || !res.Complete {
		t.Fatalf("once merged, the landing completes: %+v", res)
	}
	if st, err := ReadState(f.repo.Root(), f.runID); err != nil || st.Lanes[0].Stage != StageDone {
		t.Fatalf("the lane is done: %+v %v", st.Lanes, err)
	}
}

// TestAConflictOnAnUndeclaredFileHandsBackAndPushesNothing: a conflict on
// plain.txt, which nothing declares generated, refuses the step naming it,
// in a way that cannot change, so the next drain move hands the lane back
// as stuck naming the file; nothing is pushed.
func TestAConflictOnAnUndeclaredFileHandsBackAndPushesNothing(t *testing.T) {
	f := syncFixture(t)
	judged := f.passToLand(t, "fix.txt", "plain.txt").HeadSHA
	f.moveMain(t, "rewritten on main\n", "plain.txt")

	_, err := advance(f.repo.Root(), f.runID, f.stages, Options{})
	r := mustRefusal(t, err)
	if !strings.Contains(r.Reason, "plain.txt") || r.final == "" {
		t.Fatalf("the step is refused naming the conflicting file, in a way that cannot change: %+v", r)
	}
	l := currentLane(t, f.repo, f.runID)
	if l.HeadSHA != judged || len(l.Syncs) != 0 || f.repo.Git("rev-parse", "refs/heads/"+l.Branch) != judged {
		t.Fatalf("the refused sync leaves the branch at the judged head: %+v", l)
	}
	if f.remoteBranch(t, l.Branch) != "" {
		t.Fatal("nothing is pushed")
	}

	res, err := Drain(f.repo.Root(), Options{}, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	route := routeOf(t, res.Routed, secondIssue)
	if route.Kind != claim.HandBackStuck || !strings.Contains(route.Reason, "plain.txt") {
		t.Fatalf("the next drain move hands the lane back naming the file: %+v", route)
	}
	if f.remoteBranch(t, l.Branch) != "" {
		t.Fatal("the hand-back pushes nothing")
	}
}

// TestAPullRequestThatStartsToConflictReentersTheSync: a lane's pull request
// that the forge reports conflicting on a generated file goes back through
// the sync, its push and its arming to be made again over the merge, rather
// than being refused for a person: at its arming, from the run's own step,
// and while it waits armed, from the drain's next move.
func TestAPullRequestThatStartsToConflictReentersTheSync(t *testing.T) {
	conflicting := func(t *testing.T, f *landFixture) {
		t.Helper()
		f.moveMain(t, "generated again on main\n", "gen.txt")
		if err := os.WriteFile(filepath.Join(f.gh, "mergeable.json"), []byte(`{"mergeable":"CONFLICTING","mergeStateStatus":"DIRTY"}`), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	reentered := func(t *testing.T, f *landFixture, pushed string) {
		t.Helper()
		l := currentLane(t, f.repo, f.runID)
		if len(l.Syncs) != 1 || !l.Syncs[0].Conflicted || !slices.Equal(l.Syncs[0].Paths, []string{"gen.txt"}) ||
			l.Stage != StageValidate || l.pendingSync() == nil {
			t.Fatalf("the lane re-enters its sync, the generated file's conflict going to an implementer: %+v", l)
		}
		if ld := l.Landing; ld == nil || ld.Pushed != "" || ld.Merge != "" || ld.Armed || l.PR != 7 || !ld.RecordsDone {
			t.Fatalf("the push and the arming are to be made again, the records and the pull request kept: %+v", l.Landing)
		}
		if got := f.remoteBranch(t, l.Branch); got != pushed {
			t.Fatalf("nothing more is pushed until a fresh round passes: %s, pushed %s", got, pushed)
		}
	}

	t.Run("at its arming", func(t *testing.T) {
		f := syncFixture(t)
		f.passToLand(t, "fix.txt", "gen.txt")
		for range 4 {
			f.step(t)
			if l := currentLane(t, f.repo, f.runID); l.Landing != nil && l.Landing.Pushed == "" && l.Landing.RecordsDone {
				preflighted(t, l, l.HeadSHA)
			}
			if l := currentLane(t, f.repo, f.runID); l.PR == 7 && l.Landing.BodyChecked {
				break
			}
		}
		l := currentLane(t, f.repo, f.runID)
		if l.PR != 7 || l.Landing.Merge != "" {
			t.Fatalf("the pull request is open and not yet armed: %+v", l.Landing)
		}
		conflicting(t, f)
		f.step(t)
		reentered(t, f, l.Landing.Pushed)
		if strings.Contains(f.ghLog(t), "--auto") {
			t.Fatalf("a conflicting pull request is not armed:\n%s", f.ghLog(t))
		}
	})

	t.Run("and lands again once a fresh round judges the merge", func(t *testing.T) {
		f := syncFixture(t)
		f.passToLand(t, "fix.txt")
		f.step(t)
		f.step(t)
		l := currentLane(t, f.repo, f.runID)
		preflighted(t, l, l.HeadSHA)
		f.step(t)
		f.step(t)
		f.step(t)
		l = currentLane(t, f.repo, f.runID)
		if !l.Landing.Armed {
			t.Fatalf("the pull request is armed: %+v", l.Landing)
		}
		first := l.Landing.Pushed
		// The default branch moves with no conflict here, and the forge
		// reports the pull request conflicting all the same.
		tip := f.moveMain(t, "another account's file\n", "other.txt")
		if err := os.WriteFile(filepath.Join(f.gh, "mergeable.json"), []byte(`{"mergeable":"CONFLICTING","mergeStateStatus":"DIRTY"}`), 0o644); err != nil {
			t.Fatal(err)
		}
		// The withdrawal of the armed merge, then the sync.
		f.step(t)
		if l := currentLane(t, f.repo, f.runID); l.Landing.Armed || l.Landing.Merge != "" || l.Landing.Pushed != first || len(l.Syncs) != 0 {
			t.Fatalf("the armed merge is withdrawn in a step of its own: %+v", l.Landing)
		}
		f.step(t)
		l = currentLane(t, f.repo, f.runID)
		if l.Stage != StageValidate || len(l.Syncs) != 1 || l.Syncs[0].Merged != tip || l.Syncs[0].Conflicted || !l.syncRoundDue() {
			t.Fatalf("the lane merges the default branch and a fresh round judges it: %+v", l)
		}
		if err := os.Remove(filepath.Join(f.gh, "mergeable.json")); err != nil {
			t.Fatal(err)
		}
		passRound(t, f.repo, f.runID, f.stages, RoleRuthless, RoleSecurity)
		stepTo(t, f.repo, f.runID, f.stages, StageLand)
		l = currentLane(t, f.repo, f.runID)
		preflighted(t, l, l.HeadSHA)
		// The push, then the arming: the pull request is kept.
		f.step(t)
		f.step(t)
		l = currentLane(t, f.repo, f.runID)
		if l.Landing.Pushed != l.HeadSHA || l.Landing.Pushed == first || !l.Landing.Armed || l.PR != 7 {
			t.Fatalf("the merge head is pushed to the same pull request and armed again: %+v", l.Landing)
		}
		if got := f.remoteBranch(t, l.Branch); got != l.HeadSHA {
			t.Fatalf("origin holds the merge head: %s, want %s", got, l.HeadSHA)
		}
	})

	t.Run("while it waits armed", func(t *testing.T) {
		f := syncFixture(t)
		f.passToLand(t, "fix.txt", "gen.txt")
		f.step(t)
		f.step(t)
		l := currentLane(t, f.repo, f.runID)
		preflighted(t, l, l.HeadSHA)
		f.step(t)
		f.step(t)
		f.step(t)
		l = currentLane(t, f.repo, f.runID)
		if !l.Landing.Armed || l.PR != 7 {
			t.Fatalf("the pull request is armed: %+v", l.Landing)
		}
		conflicting(t, f)
		f.listPRs(t, stubPR{Number: 7, HeadRefName: l.Branch, State: "OPEN", Mergeable: "CONFLICTING", MergeStateStatus: "DIRTY"})

		res, err := Drain(f.repo.Root(), Options{}, DrainOptions{})
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(f.ghLog(t), "--disable-auto") {
			t.Fatalf("the armed merge is withdrawn before the branch moves:\n%s", f.ghLog(t))
		}
		if ld := currentLane(t, f.repo, f.runID).Landing; ld.Armed || ld.Merge != "" || ld.Pushed != l.Landing.Pushed {
			t.Fatalf("the withdrawal is recorded before any merge is tried: %+v", ld)
		}
		if p := parkedOf(t, res, f.runID); p.State != ParkedSyncing {
			t.Fatalf("the drain reads the lane as syncing again: %+v", p)
		}
		i := slices.IndexFunc(res.Lanes, func(dl DrainLane) bool { return dl.RunID == f.runID })
		if i < 0 || res.Lanes[i].Outcome != DrainLaneInProgress {
			t.Fatalf("the drain drives the lane again: %+v", res.Lanes)
		}
		if st, err := ReadState(f.repo.Root(), f.runID); err != nil || st.Lanes[0].HandBack != nil {
			t.Fatalf("the lane is not handed back: %+v %v", st.Lanes[0], err)
		}
		// The lane's next step finds the conflict at the arming and enters
		// the sync, nothing armed.
		f.step(t)
		reentered(t, f, l.Landing.Pushed)
	})
}

// TestADisarmedLaneWhoseSyncRefusesIsNotLeftArmed: the drain's move withdraws
// an armed merge over a conflicting pull request, and the lane's state then
// says so, whatever the merge that follows does. A person's uncommitted edit
// in the lane's worktree refuses that merge; the lane is left disarmed with
// its arming to be made again, never recorded armed while the forge holds
// nothing armed, and the refusal is named, not dropped. Once the edit is gone
// the lane's next step merges the default branch.
func TestADisarmedLaneWhoseSyncRefusesIsNotLeftArmed(t *testing.T) {
	f := syncFixture(t)
	f.passToLand(t, "fix.txt", "gen.txt")
	f.step(t)
	f.step(t)
	l := currentLane(t, f.repo, f.runID)
	preflighted(t, l, l.HeadSHA)
	f.step(t)
	f.step(t)
	f.step(t)
	l = currentLane(t, f.repo, f.runID)
	if !l.Landing.Armed || l.PR != 7 {
		t.Fatalf("armed: %+v", l.Landing)
	}
	pushed := l.Landing.Pushed
	f.moveMain(t, "generated again on main\n", "gen.txt")
	if err := os.WriteFile(filepath.Join(f.gh, "mergeable.json"), []byte(`{"mergeable":"CONFLICTING","mergeStateStatus":"DIRTY"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	f.listPRs(t, stubPR{Number: 7, HeadRefName: l.Branch, State: "OPEN", Mergeable: "CONFLICTING", MergeStateStatus: "DIRTY"})
	// A person starts resolving the conflict in the lane's worktree.
	edit := filepath.Join(l.Worktree, "fix.txt")
	if err := os.WriteFile(edit, []byte("a person's edit\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	res, err := Drain(f.repo.Root(), Options{}, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(f.ghLog(t), "--disable-auto") {
		t.Fatalf("the armed merge is withdrawn:\n%s", f.ghLog(t))
	}
	st, err := ReadState(f.repo.Root(), f.runID)
	if err != nil {
		t.Fatal(err)
	}
	ld := st.Lanes[0].Landing
	if ld == nil || ld.Armed || ld.Merge != "" || ld.Pushed != pushed || st.Lanes[0].HandBack != nil {
		t.Fatalf("the withdrawn arming is recorded, the push kept for the next step: %+v", ld)
	}
	if p := parkedOf(t, res, f.runID); p.State != ParkedSyncing {
		t.Fatalf("the drain reads the lane as syncing: %+v", p)
	}
	// The lane's next step reaches the arming, finds the conflict, and its
	// merge is refused over the person's edit, naming it; nothing is armed.
	_, err = advance(f.repo.Root(), f.runID, f.stages, Options{})
	if r := mustRefusal(t, err); !strings.Contains(r.Reason, "uncommitted changes") {
		t.Fatalf("the merge's refusal names the worktree's changes: %+v", r)
	}
	if ld := currentLane(t, f.repo, f.runID).Landing; ld.Armed || ld.Merge != "" || ld.Pushed != pushed {
		t.Fatalf("the refused merge leaves the lane disarmed: %+v", ld)
	}
	if strings.Count(f.ghLog(t), "--auto ") != 1 {
		t.Fatalf("nothing arms the conflicting pull request again:\n%s", f.ghLog(t))
	}

	if err := os.Remove(edit); err != nil {
		t.Fatal(err)
	}
	f.repo.Git("-C", l.Worktree, "checkout", "--", "fix.txt")
	f.step(t)
	l = currentLane(t, f.repo, f.runID)
	if len(l.Syncs) != 1 || !l.Syncs[0].Conflicted || l.Stage != StageValidate || l.Landing == nil || l.Landing.Pushed != "" || l.Landing.Armed {
		t.Fatalf("with the edit gone, the next step merges the default branch, nothing armed: %+v", l)
	}
}

// TestAParkedLaneWhoseSyncIsRefusedNamesTheRefusal: a lane left open for a
// person's merge, its pull request now conflicting on a generated file, goes
// back through its sync from the drain's move; when that merge is refused
// (here over a person's uncommitted edit in the lane's worktree), the move
// names the refusal beside the forge's reading rather than dropping it.
func TestAParkedLaneWhoseSyncIsRefusedNamesTheRefusal(t *testing.T) {
	f := syncFixtureWith(t, unreviewedQueueRuleset("MERGE"))
	f.passToLand(t, "fix.txt", "gen.txt")
	for range 6 {
		f.step(t)
		l := currentLane(t, f.repo, f.runID)
		if l.Landing != nil && l.Landing.Pushed == "" && l.Landing.RecordsDone {
			preflighted(t, l, l.HeadSHA)
		}
		if l.Landing != nil && l.Landing.Merge != "" {
			break
		}
	}
	l := currentLane(t, f.repo, f.runID)
	if l.PR != 7 || l.Landing.Merge == "" || l.Landing.Armed {
		t.Fatalf("the pull request is left open for a person: %+v", l.Landing)
	}
	f.moveMain(t, "generated again on main\n", "gen.txt")
	if err := os.WriteFile(filepath.Join(f.gh, "mergeable.json"), []byte(`{"mergeable":"CONFLICTING","mergeStateStatus":"DIRTY"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	f.listPRs(t, stubPR{Number: 7, HeadRefName: l.Branch, State: "OPEN", Mergeable: "CONFLICTING", MergeStateStatus: "DIRTY"})
	if err := os.WriteFile(filepath.Join(l.Worktree, "fix.txt"), []byte("a person's edit\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	res, err := Drain(f.repo.Root(), Options{}, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	p := parkedOf(t, res, f.runID)
	if p.State != ParkedConflicting || !strings.Contains(p.Note, "uncommitted changes") {
		t.Fatalf("the move names why the sync was not entered: %+v", p)
	}
	if ld := currentLane(t, f.repo, f.runID).Landing; ld.Merge != l.Landing.Merge || ld.Pushed != l.Landing.Pushed {
		t.Fatalf("the refused step changes nothing: %+v", ld)
	}
}
