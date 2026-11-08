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
	if r.Stage != string(StageLand) || !strings.Contains(r.Reason, "`admitted`") || !strings.Contains(r.Remedy, "`abcd build itd-10`") || !strings.Contains(r.Remedy, "`abcd drain`") {
		t.Fatalf("the records step is refused naming the field and the act that records it: %+v", r)
	}
	l = currentLane(t, f.repo, f.runID)
	if tip := f.repo.Git("-C", l.Worktree, "rev-parse", "HEAD"); tip != judged || l.HeadSHA != judged || l.Landing.RecordsDone {
		t.Fatalf("no commit is made: tip %s, head %s, judged %s, %+v", tip, l.HeadSHA, judged, l.Landing)
	}
	if status := f.repo.Git("-C", l.Worktree, "status", "--porcelain"); strings.TrimSpace(status) != "" {
		t.Fatalf("the worktree is untouched:\n%s", status)
	}
}

// TestFollowingTheNoAdmissionRemedyLands: the refusal's remedy works. A run
// begun before runs recorded their admission is resumed by `abcd build <key>`,
// not started afresh, so the resumed start records the admission its own start
// would have, and the records step it was refused at then commits the trailer.
func TestFollowingTheNoAdmissionRemedyLands(t *testing.T) {
	f := newLandFixture(t, queueRuleset("MERGE"))
	f.validated(t)
	f.step(t) // prepare
	if err := mutate(f.repo.Root(), f.runID, func(_ *os.Root, st *State) (bool, error) {
		st.Admitted = nil
		return true, nil
	}); err != nil {
		t.Fatal(err)
	}
	f.wantsRefusedStep(t)
	res, err := Start(f.repo.Root(), "itd-10", Options{noReserve: true})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Resumed || res.RunID != f.runID {
		t.Fatalf("premise: `abcd build itd-10` resumes the run rather than starting another: %+v", res)
	}
	st, err := ReadState(f.repo.Root(), f.runID)
	if err != nil {
		t.Fatal(err)
	}
	want := Admitted{Kind: landings.KindBuild, Record: "spc-1"}
	if st.Admitted == nil || *st.Admitted != want || !slices.ContainsFunc(st.Record, func(e Entry) bool {
		return e.Stage == StageAdmitted && strings.HasPrefix(e.Note, "admitted under "+want.String())
	}) {
		t.Fatalf("the resumed start records the admission and says so: %+v", st.Admitted)
	}
	l := f.wantsRecordsCommit(t)
	if msg := f.repo.Git("-C", l.Worktree, "log", "-1", "--format=%B", l.HeadSHA); !slices.Contains(trailerBlock(msg), "Admitted-by: build spc-1") {
		t.Fatalf("the records commit carries the admission recorded on resume:\n%s", msg)
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

// TestAFoundCommitWhoseAdmissionIsInItsProseIsNotTheRecords (itd-2610301850454860):
// a commit found on the judged head after a killed call is the landing's records
// only when its trailer block carries the admission, read back through the one
// grammar; the same line moved up into the message's prose is no admission, so
// the commit is refused rather than taken as the records.
func TestAFoundCommitWhoseAdmissionIsInItsProseIsNotTheRecords(t *testing.T) {
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
	msg := f.repo.Git("-C", l.Worktree, "log", "-1", "--format=%B", l.HeadSHA)
	paras := strings.Split(strings.TrimRight(msg, "\n"), "\n\n")
	var kept []string
	var line string
	for _, tr := range strings.Split(paras[len(paras)-1], "\n") {
		if strings.HasPrefix(tr, landings.TrailerKey+":") {
			line = tr
			continue
		}
		kept = append(kept, tr)
	}
	if line == "" {
		t.Fatalf("the records commit carries an admission to move: %q", msg)
	}
	moved := strings.Join(paras[:len(paras)-1], "\n\n") + "\n\n" + line + "\n\n" + strings.Join(kept, "\n") + "\n"
	f.repo.Git("-C", l.Worktree, "commit", "-q", "--amend", "--allow-empty", "-m", moved)
	if err := mutate(f.repo.Root(), f.runID, func(_ *os.Root, st *State) (bool, error) {
		cur := &st.Lanes[st.current()]
		cur.HeadSHA, cur.Landing.Records, cur.Landing.RecordsDone = judged, "", false
		return true, nil
	}); err != nil {
		t.Fatal(err)
	}
	r := f.wantsRefusedStep(t)
	if !strings.Contains(r.Reason, "lacks") || !strings.Contains(r.Reason, "Admitted-by: build spc-1") {
		t.Fatalf("a found commit whose admission stands in its prose is refused, naming the admission: %+v", r)
	}
}

// TestAFoundEmptyCommitWithOtherProseIsNotTheRecords (itd-2610301850454860): a
// landing that moves no record makes an empty records commit under abcd's own
// label, whose whole message the run's state decides. An empty commit found on
// the judged head after a killed call, with the right subject and trailers but
// prose abcd never composed, is refused rather than taken as the records and
// pushed under abcd's label.
func TestAFoundEmptyCommitWithOtherProseIsNotTheRecords(t *testing.T) {
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
	msg := f.repo.Git("-C", l.Worktree, "log", "-1", "--format=%B", l.HeadSHA)
	paras := strings.Split(strings.TrimRight(msg, "\n"), "\n\n")
	if len(paras) != 3 || !strings.Contains(paras[1], recordsOnlyTheAdmission) {
		t.Fatalf("the records commit is subject, prose and trailers, its prose naming the admission alone: %q", msg)
	}
	paras[1] = "Anything at all, under abcd's label."
	f.repo.Git("-C", l.Worktree, "commit", "-q", "--amend", "--allow-empty", "-m", strings.Join(paras, "\n\n")+"\n")
	if err := mutate(f.repo.Root(), f.runID, func(_ *os.Root, st *State) (bool, error) {
		cur := &st.Lanes[st.current()]
		cur.HeadSHA, cur.Landing.Records, cur.Landing.RecordsDone = judged, "", false
		return true, nil
	}); err != nil {
		t.Fatal(err)
	}
	r := f.wantsRefusedStep(t)
	if !strings.Contains(r.Reason, "lacks") || !strings.Contains(r.Reason, "message") {
		t.Fatalf("an empty found commit whose prose abcd did not compose is refused, naming its message: %+v", r)
	}
}
