package loop

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/claim"
)

// An intent's run widens its hold as an issue's does (iss-2611101316537446):
// the paths its lane's commits change beyond the footprint its reservation
// holds, a decision record the lane wrote and the records its landing changes
// among them, are reserved by the loop once the receipt is handed back, before
// the validators and before the landing's push, so the spec is never edited to
// widen it. Another copy's live hold on such a path stops the push, naming it,
// and no widening drops a path the hold already names.

// intentHandBack drives the intent run's lane to its implementer, commits
// names in its worktree and hands the receipt back.
func (f *landFixture) intentHandBack(t *testing.T, names ...string) {
	t.Helper()
	stepTo(t, f.repo, f.runID, f.stages, StageImplement)
	if res, err := advance(f.repo.Root(), f.runID, f.stages, Options{}); err != nil || res.Awaiting == nil {
		t.Fatalf("the implement stage awaits an implementer: %+v %v", res, err)
	}
	l := currentLane(t, f.repo, f.runID)
	dir := filepath.Join(f.repo.Root(), filepath.FromSlash(RunRelDir), f.runID, l.ID)
	sha := commitIn(t, f.repo, l, names...)
	rc := goodReceipt(t, f.runID, l, dir, sha)
	rc.Model = f.model
	if _, err := Receipt(f.repo.Root(), f.runID, writeReceipt(t, dir, rc), f.stages, Options{}); err != nil {
		t.Fatalf("the receipt is taken: %v", err)
	}
}

// intentToThePush takes the intent run's lane from its validators to the push,
// returning the first contention met.
func (f *landFixture) intentToThePush(t *testing.T, atLand func()) (Lane, error) {
	t.Helper()
	passRound(t, f.repo, f.runID, f.stages, RoleRuthless, RoleSecurity, RoleAuditor)
	stepTo(t, f.repo, f.runID, f.stages, StageLand)
	if atLand != nil {
		atLand()
	}
	for range 4 * len(Sequence) {
		l := currentLane(t, f.repo, f.runID)
		if l.Landing != nil && l.Landing.Pushed != "" {
			return l, nil
		}
		if l.Landing != nil && l.Landing.RecordsDone {
			preflighted(t, l, l.HeadSHA)
		}
		if _, err := advance(f.repo.Root(), f.runID, f.stages, Options{}); err != nil {
			if rf, ok := AsRefusal(err); ok && rf.Contention {
				return currentLane(t, f.repo, f.runID), err
			}
			t.Fatalf("stepping to the push: %v", err)
		}
	}
	t.Fatal("the lane never reached its push")
	return Lane{}, nil
}

// branchChanges is each path lane's branch changes since its merge base with
// origin's default branch.
func (f *landFixture) branchChanges(t *testing.T, l Lane) []string {
	t.Helper()
	base := strings.TrimSpace(f.repo.Git("merge-base", l.Branch, "refs/remotes/origin/main"))
	return strings.Fields(f.repo.Git("diff", "--no-renames", "--name-only", base, l.Branch, "--"))
}

// intentFootprint is the footprint origin's reservation of the intent names.
func (f *landFixture) intentFootprint(t *testing.T) []string {
	t.Helper()
	return originFootprint(t, f.bare, "itd-10")
}

// reservedIntentLand is the landing fixture with its run holding the intent,
// internal/widget its footprint.
func reservedIntentLand(t *testing.T) *landFixture {
	t.Helper()
	f := newLandFixture(t, queueRuleset("MERGE"))
	reserveRun(t, f.repo, f.runID, "itd-10", time.Now().UTC().Truncate(time.Second).Add(-30*time.Minute), "internal/widget")
	return f
}

// TestAnIntentLaneFindsWhatItCommitsReservedBeforeThePush: the lane commits
// one.txt beyond its spec's footprint; the push goes ahead holding every path
// the branch carries, the records its landing changed among them, and the
// footprint it held at the start.
func TestAnIntentLaneFindsWhatItCommitsReservedBeforeThePush(t *testing.T) {
	f := reservedIntentLand(t)
	f.intentHandBack(t, "one.txt")
	if fp := f.intentFootprint(t); !covers(fp, []string{"one.txt"}) {
		t.Fatalf("the receipt's widening reserves what the lane committed: %q", fp)
	}
	l, err := f.intentToThePush(t, nil)
	if err != nil {
		t.Fatal(err)
	}
	if f.remoteBranch(t, l.Branch) == "" {
		t.Fatalf("the lane's branch is pushed: %+v", l.Landing)
	}
	carried := f.branchChanges(t, l)
	if len(carried) < 2 {
		t.Fatalf("the branch carries the lane's file and its landing's records: %q", carried)
	}
	fp := f.intentFootprint(t)
	for _, p := range append(carried, "internal/widget") {
		if !covers(fp, []string{p}) {
			t.Fatalf("the reservation covers %s before the push: %q", p, fp)
		}
	}
}

// TestAnIntentRunsDecisionRecordIsCoveredBeforeItsValidators: a decision
// record the lane wrote is reserved by name once the receipt is handed back.
func TestAnIntentRunsDecisionRecordIsCoveredBeforeItsValidators(t *testing.T) {
	f := reservedIntentLand(t)
	decision := ".abcd/work/decisions/dcn-2611160000000001-a-judgement-call.md"
	f.intentHandBack(t, decision)
	fp := f.intentFootprint(t)
	if !slices.Contains(fp, decision) || !slices.Contains(fp, "internal/widget") {
		t.Fatalf("the decision record is held by name beside the footprint: %q", fp)
	}
	if slices.Contains(fp, ".abcd/work/decisions") {
		t.Fatalf("the hold never names the decisions directory, which would hold off every other run's: %q", fp)
	}
}

// TestAnIntentLaneDoesNotLandAPathAPeerHolds: the hold came to name the
// spec's footprint alone again by the landing, and another account holds
// one.txt live; the lane waits at the fence before its push, naming the path
// and the hold, with nothing pushed and its reservation as it was.
func TestAnIntentLaneDoesNotLandAPathAPeerHolds(t *testing.T) {
	f := reservedIntentLand(t)
	f.intentHandBack(t, "one.txt")
	l, err := f.intentToThePush(t, func() {
		if _, err := claim.Reserve(f.repo.Root(), claim.Request{Record: "itd-10", Footprint: []string{"internal/widget"}}, claim.Options{}); err != nil {
			t.Fatal(err)
		}
		(&narrowFixture{bare: f.bare}).reserveAs(t, "iss-99", "quinn", "one.txt", time.Now().Add(-time.Minute))
	})
	rf, ok := AsRefusal(err)
	if !ok || !rf.Contention || !waitByDesign(rf) {
		t.Fatalf("the lane waits by design: %v (%+v)", err, l.Landing)
	}
	if !strings.Contains(rf.Reason, "iss-99") || !strings.Contains(rf.Reason, "quinn") || !strings.Contains(rf.Reason, "one.txt") {
		t.Fatalf("the stop names the path, the hold and its holder: %s", rf.Reason)
	}
	if l.Stage != StageLand || l.Landing == nil || l.Landing.Pushed != "" || f.remoteBranch(t, l.Branch) != "" {
		t.Fatalf("nothing is pushed: %s %+v", l.Stage, l.Landing)
	}
	if fp := f.intentFootprint(t); !slices.Equal(fp, []string{"internal/widget"}) {
		t.Fatalf("the reservation is left as it was: %q", fp)
	}
}

// TestAnIntentRunsWideningDropsNothingItsHoldNames: `implement widen` widens
// an intent's run, keeping a path the hold names beyond its spec, and a resume
// whose spec has since gained a package reserves again with every path the
// hold named beside it.
func TestAnIntentRunsWideningDropsNothingItsHoldNames(t *testing.T) {
	repo := loopRepo(t, readyIntent("", settledQuestions), footprintedSpec)
	repo.Write("AGENTS.md", agentsMarked)
	repo.Commit("the conventions")
	bare := onOrigin(t, repo)
	res, err := Start(repo.Root(), "itd-10", Options{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := claim.Reserve(repo.Root(), claim.Request{Record: "itd-10", Footprint: []string{"internal/widget", "commands", "docs/extra.md"}}, claim.Options{}); err != nil {
		t.Fatal(err)
	}
	stepTo(t, repo, res.RunID, DefaultStages(), StageImplement)
	l := currentLane(t, repo, res.RunID)
	w, err := Widen(repo.Root(), res.RunID, l.ID, []string{"README.txt"}, Options{})
	if err != nil {
		t.Fatalf("an intent's run widens its hold: %v", err)
	}
	if !slices.Equal(w.Added, []string{"README.txt"}) {
		t.Fatalf("the widening adds README.txt: %+v", w)
	}
	want := []string{"README.txt", "commands", "docs/extra.md", "internal/widget"}
	sorted := func() []string {
		fp := slices.Clone(footprintOf(t, bare, "itd-10"))
		slices.Sort(fp)
		return fp
	}
	if fp := sorted(); !slices.Equal(fp, want) {
		t.Fatalf("the widening keeps every path the hold named: %q", fp)
	}

	spec, err := os.ReadFile(filepath.Join(repo.Root(), filepath.FromSlash(specRel)))
	if err != nil {
		t.Fatal(err)
	}
	repo.Write(specRel, strings.Replace(string(spec), "- packages: internal/widget, commands/", "- packages: internal/widget, commands/, cmd", 1))
	repo.Commit("the spec gains a package")
	if _, err := Start(repo.Root(), "itd-10", Options{}); err != nil {
		t.Fatalf("the resume reserves again: %v", err)
	}
	if fp := sorted(); !slices.Equal(fp, append([]string{"README.txt", "cmd"}, want[1:]...)) {
		t.Fatalf("the resume's widening drops nothing the hold named: %q", fp)
	}
}
