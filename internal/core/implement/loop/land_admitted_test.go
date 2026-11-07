package loop

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/landings"
)

// trailerBlock is a commit message's last paragraph, one line per trailer.
func trailerBlock(msg string) []string {
	paras := strings.Split(strings.TrimRight(msg, "\n"), "\n\n")
	return strings.Split(paras[len(paras)-1], "\n")
}

// TestBuildLaneCommitsItsSpecOrIssue (itd-2610301850454860): an intent lane's
// records commit carries `Admitted-by: build <spec id>` after its delivery and
// resolution and before its disclosure, read back as the run's admission; the
// pull request's body carries the same line among its trailers.
func TestBuildLaneCommitsItsSpecOrIssue(t *testing.T) {
	f := newLandFixture(t, queueRuleset("MERGE"))
	f.validatedOnce(t)
	f.step(t) // prepare
	f.step(t) // records
	l := currentLane(t, f.repo, f.runID)
	msg := f.repo.Git("-C", l.Worktree, "log", "-1", "--format=%B", l.HeadSHA)
	want := []string{"Delivers: itd-10", "Resolves: " + f.issue, "Admitted-by: build spc-1", "Assisted-by: Claude:claude-test-5"}
	if got := trailerBlock(msg); !slices.Equal(got, want) {
		t.Fatalf("the records commit's trailers are %q, want %q:\n%s", got, want, msg)
	}
	if a, ok := landings.ParseAdmitted(msg); !ok || a != (Admitted{Kind: landings.KindBuild, Record: "spc-1"}) {
		t.Fatalf("the trailer reads back as the run's admission: %+v %v", a, ok)
	}
	if strings.Contains(msg, "Refs:") {
		t.Fatalf("an admission naming a spec needs no Refs: line:\n%s", msg)
	}
	st, err := ReadState(f.repo.Root(), f.runID)
	if err != nil {
		t.Fatal(err)
	}
	if body := prBody(st, l, []string{"Assisted-by: Claude:claude-test-5"}); !strings.Contains(body, "\nResolves: "+f.issue+"\nAdmitted-by: build spc-1\nAssisted-by: abcd:") {
		t.Fatalf("the body carries the admission among its trailers:\n%s", body)
	}
}

// TestAdmittedTrailersDeclareAnIssueTheyName: an admission naming an issue the
// lane does not resolve is declared by a Refs: line, which RS004's message half
// asks of every issue id a message names; one it resolves needs none.
func TestAdmittedTrailersDeclareAnIssueTheyName(t *testing.T) {
	a := Admitted{Kind: landings.KindBuild, Record: "iss-7"}
	if got := admittedTrailers(a, nil); !slices.Equal(got, []string{"Admitted-by: build iss-7", "Refs: iss-7"}) {
		t.Fatalf("an unresolved issue is declared: %q", got)
	}
	if got := admittedTrailers(a, []Resolution{{Issue: "iss-7"}}); !slices.Equal(got, []string{"Admitted-by: build iss-7"}) {
		t.Fatalf("a resolved issue is declared by its Resolves: line: %q", got)
	}
	if got := admittedTrailers(Admitted{Kind: landings.KindBuild, Record: "spc-1"}, nil); !slices.Equal(got, []string{"Admitted-by: build spc-1"}) {
		t.Fatalf("a spec needs no declaration: %q", got)
	}
}

// TestRecordsStepRefusesAStateWithoutAdmission: a run whose state names no
// admission, as one begun before runs recorded theirs, is refused at its records
// step naming the field, with the branch unmoved and the worktree untouched.
func TestRecordsStepRefusesAStateWithoutAdmission(t *testing.T) {
	f := newLandFixture(t, queueRuleset("MERGE"))
	l := f.validated(t)
	judged := l.HeadSHA
	f.step(t) // prepare
	if err := mutate(f.repo.Root(), f.runID, func(_ *os.Root, st *State) (bool, error) {
		st.Admitted = nil
		return true, nil
	}); err != nil {
		t.Fatal(err)
	}
	r := f.wantsRefusedStep(t)
	if r.Stage != string(StageLand) || !strings.Contains(r.Reason, "`admitted`") || !strings.Contains(r.Remedy, "start the run again") {
		t.Fatalf("the records step is refused naming the field: %+v", r)
	}
	l = currentLane(t, f.repo, f.runID)
	if tip := f.repo.Git("-C", l.Worktree, "rev-parse", "HEAD"); tip != judged || l.HeadSHA != judged || l.Landing.RecordsDone {
		t.Fatalf("no commit is made: tip %s, head %s, judged %s, %+v", tip, l.HeadSHA, judged, l.Landing)
	}
	if status := f.repo.Git("-C", l.Worktree, "status", "--porcelain"); strings.TrimSpace(status) != "" {
		t.Fatalf("the worktree is untouched:\n%s", status)
	}
}

// TestRecordOnlyLaneStillCommits: a lane whose landing marks no step, closes no
// spec and resolves no capture still makes its records commit, changing no
// file, with abcd's label and the admission; the commit is the lane's head, and
// the head the landing pushes.
func TestRecordOnlyLaneStillCommits(t *testing.T) {
	const second = "2. The loop\n   - packages: internal/core/implement\n"
	f := newLandFixtureWith(t, map[string]string{
		".abcd/work/rulesets/main-protection.json": queueRuleset("MERGE"),
		specRel: specWithSteps("1. The parser\n   - packages: internal/core/spec\n" + second),
	})
	l := f.twoStepLandWithNoResolutions(t)
	// The lane's own change already marks its step, so the landing has no mark
	// to write.
	judged := f.commitAtTheJudgedHead(t, l, func() {
		marked := specWithSteps("1. The parser\n   - packages: internal/core/spec\n   - landed: #1\n" + second)
		if err := os.WriteFile(filepath.Join(l.Worktree, specRel), []byte(marked), 0o644); err != nil {
			t.Fatal(err)
		}
	})
	f.step(t) // prepare
	l = currentLane(t, f.repo, f.runID)
	if ld := l.Landing; ld == nil || ld.MarkStep != 0 || ld.Closes || ld.SpecSettled || len(l.Resolves) != 0 || ld.RecordsDone {
		t.Fatalf("premise: the prepared landing moves no record and still has its commit to make: %+v", ld)
	}
	l = f.wantsRecordsCommit(t)
	if l.HeadSHA == judged || l.Landing.Records != l.HeadSHA {
		t.Fatalf("the records commit is made and is the lane's head: head %s, judged %s, %+v", l.HeadSHA, judged, l.Landing)
	}
	if parent := f.repo.Git("-C", l.Worktree, "rev-parse", l.HeadSHA+"^"); parent != judged {
		t.Fatalf("the records commit sits on the judged head: %s", parent)
	}
	if changed := f.repo.Git("-C", l.Worktree, "diff-tree", "--no-commit-id", "-r", "--name-only", l.HeadSHA); strings.TrimSpace(changed) != "" {
		t.Fatalf("the records commit changes no file: %q", changed)
	}
	msg := f.repo.Git("-C", l.Worktree, "log", "-1", "--format=%B", l.HeadSHA)
	if !strings.Contains(msg, recordsOnlyTheAdmission) || !slices.Equal(trailerBlock(msg), []string{"Admitted-by: build spc-1", composedAssistedBy()}) {
		t.Fatalf("the commit says it records only the admission, under abcd's label:\n%s", msg)
	}
	preflighted(t, l, l.HeadSHA)
	f.step(t) // push
	if l2 := currentLane(t, f.repo, f.runID); l2.Landing.Pushed != l.HeadSHA || f.remoteBranch(t, l.Branch) != l.HeadSHA {
		t.Fatalf("the pushed head is the records commit: %+v, remote %s", l2.Landing, f.remoteBranch(t, l.Branch))
	}
}

// TestAKilledRecordOnlyCommitIsFound: a call killed after it made a records
// commit that changes no file and before it wrote the state finds the commit
// on the next call rather than refusing it.
func TestAKilledRecordOnlyCommitIsFound(t *testing.T) {
	const second = "2. The loop\n   - packages: internal/core/implement\n"
	f := newLandFixtureWith(t, map[string]string{
		".abcd/work/rulesets/main-protection.json": queueRuleset("MERGE"),
		specRel: specWithSteps("1. The parser\n   - packages: internal/core/spec\n" + second),
	})
	l := f.twoStepLandWithNoResolutions(t)
	judged := f.commitAtTheJudgedHead(t, l, func() {
		marked := specWithSteps("1. The parser\n   - packages: internal/core/spec\n   - landed: #1\n" + second)
		if err := os.WriteFile(filepath.Join(l.Worktree, specRel), []byte(marked), 0o644); err != nil {
			t.Fatal(err)
		}
	})
	f.step(t) // prepare
	l = f.wantsRecordsCommit(t)
	made := l.HeadSHA
	// Put the state back as the killed call left it: the commit made, the state
	// not yet written.
	if err := mutate(f.repo.Root(), f.runID, func(_ *os.Root, st *State) (bool, error) {
		cur := &st.Lanes[st.current()]
		cur.HeadSHA, cur.Landing.Records, cur.Landing.RecordsDone = judged, "", false
		return true, nil
	}); err != nil {
		t.Fatal(err)
	}
	l = f.wantsRecordsCommit(t)
	if l.HeadSHA != made || l.Landing.Records != made {
		t.Fatalf("the commit the killed call made is found: head %s, made %s", l.HeadSHA, made)
	}
}
