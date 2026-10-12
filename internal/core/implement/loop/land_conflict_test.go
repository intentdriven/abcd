package loop

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The forge's mergeability as `gh pr view <n> --json mergeable,mergeStateStatus`
// prints it.
const (
	forgeConflicting = `{"mergeable":"CONFLICTING","mergeStateStatus":"DIRTY"}`
	forgeUnknown     = `{"mergeable":"UNKNOWN","mergeStateStatus":"UNKNOWN"}`
	forgeMergeable   = `{"mergeable":"MERGEABLE","mergeStateStatus":"BLOCKED"}`
)

// unionOne is the lane's file under the union driver, as .gitattributes names
// CHANGELOG.md and DECISIONS.md in this repository.
var unionOne = map[string]string{".gitattributes": "one.txt merge=union\n"}

// forgeSays sets the stub forge's mergeability of the pull request.
func (f *landFixture) forgeSays(t *testing.T, verdict string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(f.gh, "mergeable.json"), []byte(verdict+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

// mainAdds lands a commit on the remote's default branch that adds the lane's
// own file with other content, as a later merge to main that touched it does.
func (f *landFixture) mainAdds(t *testing.T, name, body string) {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "other")
	f.repo.Git("clone", "-q", f.bare, dir)
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	f.repo.Git("-C", dir, "add", "--", name)
	f.repo.Git("-C", dir, "commit", "-q", "-m", "main: "+name)
	f.repo.Git("-C", dir, "push", "-q", "origin", "HEAD:refs/heads/main")
}

// toArm takes a validated lane through its records, push and pull request,
// stopping before the arm.
func (f *landFixture) toArm(t *testing.T) Lane {
	t.Helper()
	f.validated(t)
	f.step(t)
	f.step(t)
	l := currentLane(t, f.repo, f.runID)
	preflighted(t, l, l.HeadSHA)
	f.step(t)
	f.step(t)
	return currentLane(t, f.repo, f.runID)
}

// stateBytes is the run's state file as it stands.
func (f *landFixture) stateBytes(t *testing.T) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(f.repo.Root(), filepath.FromSlash(StateRelPath(f.runID))))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// refusedUnchanged advances the run once, wants a refusal that is not
// contention (a conflicting pull request needs a person, exit 2), and wants
// the state file as it was.
func (f *landFixture) refusedUnchanged(t *testing.T) *Refusal {
	t.Helper()
	before := f.stateBytes(t)
	_, err := advance(f.repo.Root(), f.runID, f.stages, Options{})
	r := mustRefusal(t, err)
	if r.Contention {
		t.Fatalf("a conflicting pull request is a person's, refused at exit 2, not the merge's wait at exit 3: %+v", r)
	}
	if r.Stage != string(StageLand) {
		t.Fatalf("the refusal names the land stage: %+v", r)
	}
	if !bytes.Equal(before, f.stateBytes(t)) {
		t.Fatalf("a refusal writes nothing: the state file changed")
	}
	return r
}

// TestAConflictingPullRequestIsRefusedBeforeArming is iss-2610110908443341 at
// the arm: a pull request the forge already reports CONFLICTING is not armed,
// since a conflicting pull request never enters the merge queue and nothing
// is pushed to a lane after arming; the refusal names the union-driven file
// the forge ignores the driver of, and the hand merge as the remedy.
func TestAConflictingPullRequestIsRefusedBeforeArming(t *testing.T) {
	f := newLandFixtureWith(t, map[string]string{".abcd/work/rulesets/main-protection.json": queueRuleset("MERGE"),
		".gitattributes": unionOne[".gitattributes"]})
	l := f.toArm(t)
	f.mainAdds(t, "one.txt", "main's own line\n")
	f.forgeSays(t, forgeConflicting)
	r := f.refusedUnchanged(t)
	for _, want := range []string{"pull request #7 conflicts with main", "CONFLICTING", "DIRTY", "does not arm",
		"merge drivers the forge ignores", "one.txt (merge=union)"} {
		if !strings.Contains(r.Reason, want) {
			t.Fatalf("the refusal's reason names %q: %s", want, r.Reason)
		}
	}
	for _, want := range []string{"merges origin/main into " + l.Branch, "make preflight", "pushes the branch", "arms it once the forge reports it mergeable"} {
		if !strings.Contains(r.Remedy, want) {
			t.Fatalf("the remedy names %q: %s", want, r.Remedy)
		}
	}
	if strings.Contains(f.ghLog(t), "pr merge") {
		t.Fatalf("a conflicting pull request is never armed:\n%s", f.ghLog(t))
	}
	if l = currentLane(t, f.repo, f.runID); l.Landing.Merge != "" || l.Landing.Armed {
		t.Fatalf("the landing records no arm: %+v", l.Landing)
	}

	// Once the forge reports it mergeable, the next step arms it.
	f.forgeSays(t, forgeMergeable)
	f.step(t)
	if !strings.Contains(f.ghLog(t), "pr merge 7 --auto --merge") {
		t.Fatalf("a mergeable pull request arms as before:\n%s", f.ghLog(t))
	}
}

// TestAnArmedPullRequestThatTurnsConflictingIsRefusedNamingItsPaths is
// iss-2610110908443341 at the merged check: a merge to main after the arm
// conflicts with the lane on the forge, though the union driver merges it
// here. The step refuses, naming the file, the driver and the hand merge,
// instead of repeating "not merged yet"; nothing is pushed or cleaned up.
func TestAnArmedPullRequestThatTurnsConflictingIsRefusedNamingItsPaths(t *testing.T) {
	f := newLandFixtureWith(t, map[string]string{".abcd/work/rulesets/main-protection.json": queueRuleset("MERGE"),
		".gitattributes": unionOne[".gitattributes"]})
	l := f.landedToArmed(t)
	pushed := f.remoteBranch(t, l.Branch)
	f.mainAdds(t, "one.txt", "main's own line\n")
	f.forgeSays(t, forgeConflicting)
	r := f.refusedUnchanged(t)
	for _, want := range []string{"pull request #7 is armed but conflicts with main", "CONFLICTING",
		"locally these files carry merge drivers the forge ignores", "one.txt (merge=union)"} {
		if !strings.Contains(r.Reason, want) {
			t.Fatalf("the refusal's reason names %q: %s", want, r.Reason)
		}
	}
	if strings.Contains(r.Reason, "not merged yet") {
		t.Fatalf("a conflicting pull request is not reported as an ordinary wait: %s", r.Reason)
	}
	for _, want := range []string{"gh pr merge 7 --disable-auto", "merges origin/main into " + l.Branch, "make preflight",
		"re-arms it (`gh pr merge 7 --auto --merge`)", "pushes nothing to a lane after arming"} {
		if !strings.Contains(r.Remedy, want) {
			t.Fatalf("the remedy names %q: %s", want, r.Remedy)
		}
	}
	if got := f.remoteBranch(t, l.Branch); got != pushed {
		t.Fatalf("nothing is pushed after arming: the remote moved from %s to %s", pushed, got)
	}
	if n := strings.Count(f.ghLog(t), "pr merge"); n != 1 {
		t.Fatalf("the loop neither disarms nor re-arms on its own: %d merge calls\n%s", n, f.ghLog(t))
	}
	if _, err := os.Stat(l.Worktree); err != nil {
		t.Fatalf("nothing is cleaned up: %v", err)
	}
}

// TestAConflictWithoutADriverIsNamedAsOne: a file the lane and main both added
// with no merge driver conflicts here as on the forge, and the refusal says so
// rather than blaming a driver.
func TestAConflictWithoutADriverIsNamedAsOne(t *testing.T) {
	f := newLandFixture(t, queueRuleset("MERGE"))
	f.landedToArmed(t)
	f.mainAdds(t, "one.txt", "main's own line\n")
	f.forgeSays(t, forgeConflicting)
	r := f.refusedUnchanged(t)
	if !strings.Contains(r.Reason, "conflicts here too, in: one.txt") || strings.Contains(r.Reason, "merge drivers") {
		t.Fatalf("a driver-free conflict is named as one: %s", r.Reason)
	}
}

// TestAConflictThisCheckoutCannotReproduceIsSaidSo: the forge reports a
// conflict no local merge shows (a stale verdict, iss-2608290810033503); the
// refusal says it cannot reproduce it rather than guessing at paths.
func TestAConflictThisCheckoutCannotReproduceIsSaidSo(t *testing.T) {
	f := newLandFixture(t, queueRuleset("MERGE"))
	f.landedToArmed(t)
	f.forgeSays(t, forgeConflicting)
	r := f.refusedUnchanged(t)
	if !strings.Contains(r.Reason, "GitHub reports a conflict this checkout cannot reproduce") {
		t.Fatalf("an unreproduced conflict is said to be one: %s", r.Reason)
	}
}

// TestAnUncomputedOrMergeableVerdictKeepsWaiting: UNKNOWN is a verdict the
// forge computes lazily and has not computed yet, and MERGEABLE is the
// ordinary wait for the queue; both stay the merge's contention wait (exit 3),
// and an UNKNOWN verdict at the arm arms as before.
func TestAnUncomputedOrMergeableVerdictKeepsWaiting(t *testing.T) {
	f := newLandFixture(t, queueRuleset("MERGE"))
	f.toArm(t)
	f.forgeSays(t, forgeUnknown)
	f.step(t)
	if !strings.Contains(f.ghLog(t), "pr merge 7 --auto --merge") {
		t.Fatalf("an uncomputed verdict at the arm arms as before:\n%s", f.ghLog(t))
	}
	for verdict, note := range map[string]string{forgeUnknown: "the forge has not yet computed whether it merges",
		forgeMergeable: "the forge reports it mergeable, merge state BLOCKED"} {
		f.forgeSays(t, verdict)
		_, err := advance(f.repo.Root(), f.runID, f.stages, Options{})
		r := mustRefusal(t, err)
		if !r.Contention || !strings.Contains(r.Reason, "pull request #7 is not merged yet") || !strings.Contains(r.Reason, note) {
			t.Fatalf("%s keeps the merge's wait, naming the verdict: %+v", verdict, r)
		}
	}
}
