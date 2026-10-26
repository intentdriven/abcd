package loop

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/spec"
)

// TestTheLandingMarksTheLanesStepLanded (iss-2610110918418459): a lane that
// lands one step of a spec with more to build writes the step's `- landed:`
// line onto the open spec in its records commit, even when it fixed no
// capture, so a later `abcd build` of the spec, or a --remainder close, reads
// the step as landed rather than opening a lane for it again.
func TestTheLandingMarksTheLanesStepLanded(t *testing.T) {
	f := newLandFixtureWith(t, map[string]string{
		".abcd/work/rulesets/main-protection.json": queueRuleset("MERGE"),
		specRel: specWithSteps("1. The parser\n   - packages: internal/core/spec\n" +
			"2. The loop\n   - packages: internal/core/implement\n"),
	})
	repo, id := f.repo, f.runID
	stepTo(t, repo, id, f.stages, StageImplement)
	if res, err := advance(repo.Root(), id, f.stages, Options{}); err != nil || res.Awaiting == nil {
		t.Fatalf("the implement stage awaits an implementer: %+v %v", res, err)
	}
	l := currentLane(t, repo, id)
	if l.SpecStep != 1 {
		t.Fatalf("the first lane builds step 1, not %d", l.SpecStep)
	}
	dir := filepath.Join(repo.Root(), filepath.FromSlash(RunRelDir), id, l.ID)
	sha := laneCommit(t, repo, l, "one.txt")
	rc := goodReceipt(t, id, l, dir, sha)
	path := writeReceipt(t, dir, map[string]any{
		"schema_version": rc.SchemaVersion, "run_id": rc.RunID, "lane": rc.Lane, "branch": rc.Branch,
		"commits": rc.Commits, "definition_of_done": rc.DefinitionOfDone, "report": rc.Report, "model": f.model,
	})
	if _, err := Receipt(repo.Root(), id, path, f.stages, Options{}); err != nil {
		t.Fatalf("the receipt verifies: %v", err)
	}
	passRound(t, repo, id, f.stages, RoleRuthless, RoleSecurity)
	l = stepTo(t, repo, id, f.stages, StageLand)
	judged := l.HeadSHA

	f.step(t) // prepare
	if l = currentLane(t, repo, id); l.Landing == nil || l.Landing.MarkStep != 1 || l.Landing.RecordsDone {
		t.Fatalf("the prepared landing marks step 1 and has records to make: %+v", l.Landing)
	}
	f.step(t) // records
	l = currentLane(t, repo, id)
	if l.HeadSHA == judged || !l.Landing.RecordsDone {
		t.Fatalf("a lane that lands a step makes a records commit: %+v", l.Landing)
	}
	steps, err := spec.ReadSteps(l.Worktree, spec.Spec{Path: specRel})
	if err != nil {
		t.Fatal(err)
	}
	if steps[0].Landed != shortSHA(judged) || steps[1].Landed != "" {
		t.Fatalf("step 1 is marked landed by the judged head %s and step 2 is not: %+v", shortSHA(judged), steps)
	}
	msg := repo.Git("-C", l.Worktree, "log", "-1", "--format=%B", l.HeadSHA)
	// The mark is abcd's text from the run's state, so a commit that only
	// marks the step declares abcd's label, not the implementer's model.
	if !strings.Contains(msg, "marked step 1 landed") || !strings.Contains(msg, composedAssistedBy()) ||
		strings.Contains(msg, f.model) || strings.Contains(msg, "Delivers:") {
		t.Fatalf("the records commit names the mark, abcd's label, no model and no delivery:\n%s", msg)
	}
	if changed := repo.Git("-C", l.Worktree, "diff", "--name-only", judged, l.HeadSHA); strings.TrimSpace(changed) != specRel {
		t.Fatalf("the records commit changes only the spec: %q", changed)
	}
}

// TestTheClosingLanesMarkSurvivesARefusingHook: on the lane that closes a
// stepped spec, the mark is written before the close moves the spec, so a
// commit-msg hook that refuses the records commit leaves the spec closed with
// the mark on it; once the hook passes, the step resumes over the closed spec
// rather than refusing it as not open, and the closed spec carries the mark.
func TestTheClosingLanesMarkSurvivesARefusingHook(t *testing.T) {
	f := newLandFixtureWith(t, map[string]string{
		".abcd/work/rulesets/main-protection.json": queueRuleset("MERGE"),
		specRel: specWithSteps("1. The parser\n   - packages: internal/core/spec\n"),
	})
	l := f.validated(t)
	judged := l.HeadSHA
	hook := installHook(t, f, "commit-msg", "#!/bin/sh\necho 'commit-msg: refused by the test' >&2\nexit 1\n")
	f.step(t) // prepare
	if l = currentLane(t, f.repo, f.runID); !l.Landing.Closes || l.Landing.MarkStep != 1 {
		t.Fatalf("the closing lane marks its step: %+v", l.Landing)
	}
	_, err := advance(f.repo.Root(), f.runID, f.stages, Options{})
	if r := mustRefusal(t, err); !strings.Contains(r.Reason, "refused by the test") {
		t.Fatalf("the refusing hook stops the records: %+v", r)
	}
	if err := os.WriteFile(hook, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	acceptHook(t, f)
	f.step(t)
	l = currentLane(t, f.repo, f.runID)
	if l.HeadSHA == judged || !l.Landing.RecordsDone {
		t.Fatalf("the landing resumes once the hook passes: %+v", l.Landing)
	}
	closed := strings.Replace(specRel, "/open/", "/closed/", 1)
	steps, err := spec.ReadSteps(l.Worktree, spec.Spec{Path: closed})
	if err != nil || len(steps) != 1 || steps[0].Landed != shortSHA(judged) {
		t.Fatalf("the closed spec carries the step's mark: %+v %v", steps, err)
	}
}
