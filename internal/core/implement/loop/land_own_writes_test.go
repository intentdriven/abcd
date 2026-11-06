package loop

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// committedFiles lists what the commit changes, against its first parent.
func committedFiles(f *landFixture, l Lane, sha string) []string {
	return strings.Fields(f.repo.Git("-C", l.Worktree, "diff-tree", "--no-commit-id", "--name-only", "-r", sha))
}

// A stray file names itself and how to take it out, on the first attempt and on a
// retry after a refusing hook; the commit made once it is gone does not hold it.
func TestALandingCommitsOnlyWhatItWrote(t *testing.T) {
	strays := map[string]struct {
		rel    string
		remove func(t *testing.T, f *landFixture, l Lane)
	}{
		"a new markdown file": {"docs/stray.md", func(t *testing.T, _ *landFixture, l Lane) {
			t.Helper()
			if err := os.Remove(filepath.Join(l.Worktree, "docs", "stray.md")); err != nil {
				t.Fatal(err)
			}
		}},
		"a tracked markdown file changed": {"AGENTS.md", func(t *testing.T, f *landFixture, l Lane) {
			t.Helper()
			f.repo.Git("-C", l.Worktree, "checkout", "-q", "--", "AGENTS.md")
		}},
	}
	for name, s := range strays {
		for _, retry := range []bool{false, true} {
			t.Run(name+map[bool]string{false: ", first attempt", true: ", after a refusing hook"}[retry], func(t *testing.T) {
				f := newLandFixture(t, queueRuleset("MERGE"))
				l := f.validatedOnce(t)
				judged := l.HeadSHA
				var hook string
				if retry {
					hook = installHook(t, f, "commit-msg", "#!/bin/sh\necho 'commit-msg: refused by the test' >&2\nexit 1\n")
				}
				f.step(t) // prepare
				if retry {
					if r := f.wantsRefusedStep(t); !strings.Contains(r.Reason, "refused by the test") {
						t.Fatalf("the hook refuses first: %+v", r)
					}
				}
				landWrite(t, l.Worktree, s.rel, "not the landing's\n")
				r := f.wantsRefusedStep(t)
				if !strings.Contains(r.Reason, s.rel) || !strings.Contains(r.Reason, "commits only what it writes") || !strings.Contains(r.Remedy, "remove it") {
					t.Fatalf("the refusal names the file and how to take it out: %+v", r)
				}
				if got := f.repo.Git("-C", l.Worktree, "rev-parse", "HEAD"); got != judged {
					t.Fatalf("nothing is committed: %s, %s", got, judged)
				}
				s.remove(t, f, l)
				if retry {
					if err := os.WriteFile(hook, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
						t.Fatal(err)
					}
					acceptHook(t, f)
				}
				l = f.wantsRecordsCommit(t)
				if slices.Contains(committedFiles(f, l, l.HeadSHA), s.rel) {
					t.Fatalf("the commit holds %s: %v", s.rel, committedFiles(f, l, l.HeadSHA))
				}
			})
		}
	}
}

// A link the close repoints in a file outside the record stores is the landing's own
// write: it is committed, on the attempt after a hook refused it too, and nothing else
// beside it is.
func TestALandingCommitsTheLinksItsCloseRepointed(t *testing.T) {
	const guide = "docs/guide.md"
	f := newLandFixtureWith(t, map[string]string{
		".abcd/work/rulesets/main-protection.json": queueRuleset("MERGE"),
		guide: "See [the intent](../" + plannedIntentRel + ").\n",
	})
	l := f.validated(t)
	hook := installHook(t, f, "commit-msg", "#!/bin/sh\necho 'commit-msg: refused by the test' >&2\nexit 1\n")
	f.step(t) // prepare
	res := f.step(t)
	if !strings.Contains(res.stepNote, guide) {
		t.Fatalf("the run's record names the file the close repointed: %q", res.stepNote)
	}
	if r := f.wantsRefusedStep(t); !strings.Contains(r.Reason, "refused by the test") {
		t.Fatalf("the hook refuses what the landing wrote, and nothing else stops it: %+v", r)
	}
	landWrite(t, l.Worktree, "docs/stray.md", "not the landing's\n")
	if r := f.wantsRefusedStep(t); !strings.Contains(r.Reason, "docs/stray.md") {
		t.Fatalf("a stray file beside the repointed one is refused: %+v", r)
	}
	if err := os.Remove(filepath.Join(l.Worktree, "docs", "stray.md")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(hook, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	acceptHook(t, f)
	l = f.wantsRecordsCommit(t)
	got := committedFiles(f, l, l.HeadSHA)
	if !slices.Contains(got, guide) || slices.Contains(got, "docs/stray.md") {
		t.Fatalf("the commit holds the repointed link and nothing stray: %v", got)
	}
	if body := landRead(t, l.Worktree, guide); !strings.Contains(body, "/shipped/") {
		t.Fatalf("the link follows the move:\n%s", body)
	}
}

// walkIssueLaneToLand takes a drain's issue lane to its first landing step.
func (f *landFixture) walkIssueLaneToLand(t *testing.T) Lane {
	t.Helper()
	first := f.firstMove(t)
	f.runID, f.issue = first.Start.RunID, secondIssue
	repo := f.repo
	stepTo(t, repo, f.runID, f.stages, StageImplement)
	if res, err := advance(repo.Root(), f.runID, f.stages, Options{}); err != nil || res.Awaiting == nil {
		t.Fatalf("the implement stage awaits an implementer: %+v %v", res, err)
	}
	l := currentLane(t, repo, f.runID)
	dir := filepath.Join(repo.Root(), filepath.FromSlash(RunRelDir), f.runID, l.ID)
	sha := laneCommit(t, repo, l, "fix.txt")
	rc := goodReceipt(t, f.runID, l, dir, sha)
	rc.Model = "claude-test-5"
	rc.Resolves = []Resolution{{Issue: f.issue, Commit: sha, Note: "the page names the flag", Impact: "fix",
		Grounds: "pursued: the page names the flag; shown wrong if it still does not"}}
	if _, err := Receipt(repo.Root(), f.runID, writeReceipt(t, dir, rc), f.stages, Options{}); err != nil {
		t.Fatal(err)
	}
	passRound(t, repo, f.runID, f.stages, RoleRuthless, RoleSecurity)
	stepTo(t, repo, f.runID, f.stages, StageLand)
	return currentLane(t, repo, f.runID)
}

// An issue lane moves no intent: a move in its worktree is put back before its records
// commit, which carries its resolution alone.
func TestAnIssueLanesRecordsCarryNoIntentMove(t *testing.T) {
	f := drainLandFixture(t)
	l := f.walkIssueLaneToLand(t)
	planned := readyIntent("impact: additive\n", settledQuestions)
	f.commitAtTheJudgedHead(t, l, func() { landWrite(t, l.Worktree, plannedIntentRel, planned) })
	f.step(t) // prepare
	if l = currentLane(t, f.repo, f.runID); l.Landing == nil {
		t.Fatalf("premise: the landing is prepared: %+v", l)
	}
	landWrite(t, l.Worktree, shippedIntentRel, planned)
	if err := os.Remove(filepath.Join(l.Worktree, filepath.FromSlash(plannedIntentRel))); err != nil {
		t.Fatal(err)
	}
	r := f.wantsRefusedStep(t)
	if !strings.Contains(r.Reason, "itd-10") || !strings.Contains(r.Remedy, "put it back") {
		t.Fatalf("the refusal names the record and how to put it back: %+v", r)
	}
	if err := os.Remove(filepath.Join(l.Worktree, filepath.FromSlash(shippedIntentRel))); err != nil {
		t.Fatal(err)
	}
	landWrite(t, l.Worktree, plannedIntentRel, planned)
	l = f.wantsRecordsCommit(t)
	for _, p := range committedFiles(f, l, l.HeadSHA) {
		if strings.Contains(p, "/intents/") {
			t.Fatalf("the records commit carries %s", p)
		}
	}
	msg := f.repo.Git("-C", l.Worktree, "log", "-1", "--format=%B", l.HeadSHA)
	if !strings.Contains(msg, "Resolves: "+f.issue) || strings.Contains(msg, "Delivers:") {
		t.Fatalf("the commit resolves the issue and delivers nothing:\n%s", msg)
	}
}

// A link a resolution repoints outside the record stores is the landing's own write on
// an issue lane too.
func TestAnIssueLaneCommitsTheLinksItsResolutionRepointed(t *testing.T) {
	const guide = "docs/guide.md"
	f := drainLandFixture(t)
	l := f.walkIssueLaneToLand(t)
	found, err := filepath.Glob(filepath.Join(l.Worktree, ".abcd", "work", "issues", "open", f.issue+"*"))
	if err != nil || len(found) != 1 {
		t.Fatalf("premise: the issue's record: %v %v", found, err)
	}
	f.commitAtTheJudgedHead(t, l, func() {
		landWrite(t, l.Worktree, guide, "See [the issue](../.abcd/work/issues/open/"+filepath.Base(found[0])+").\n")
	})
	f.step(t) // prepare
	if res := f.step(t); !strings.Contains(res.stepNote, guide) {
		t.Fatalf("the run's record names the file the resolution repointed: %q", res.stepNote)
	}
	l = f.wantsRecordsCommit(t)
	if got := committedFiles(f, l, l.HeadSHA); !slices.Contains(got, guide) {
		t.Fatalf("the commit holds the repointed link: %v", got)
	}
	if body := landRead(t, l.Worktree, guide); !strings.Contains(body, "/resolved/") {
		t.Fatalf("the link follows the move:\n%s", body)
	}
}

// The record's line speaks for the lane and the head it was written on.
func TestALandingsRepointedLineHoldsForItsLaneAndHead(t *testing.T) {
	const head, later = "aaaa", "bbbb"
	l := Lane{ID: "lane-1", HeadSHA: head}
	line := repointedNote(head, map[string]bool{"docs/a.md": true})
	for name, c := range map[string]struct {
		entry Entry
		lane  Lane
		want  bool
	}{
		"its own":              {Entry{Lane: "lane-1", Stage: string(StageLand), Note: line}, l, true},
		"another lane's":       {Entry{Lane: "lane-2", Stage: string(StageLand), Note: line}, l, false},
		"another stage's":      {Entry{Lane: "lane-1", Stage: string(StageValidate), Note: line}, l, false},
		"an earlier head's":    {Entry{Lane: "lane-1", Stage: string(StageLand), Note: line}, Lane{ID: "lane-1", HeadSHA: later}, false},
		"a line that is prose": {Entry{Lane: "lane-1", Stage: string(StageLand), Note: "docs/a.md was repointed"}, l, false},
		"a file not markdown":  {Entry{Lane: "lane-1", Stage: string(StageLand), Note: repointedNote(head, map[string]bool{"docs/a.txt": true})}, l, false},
	} {
		t.Run(name, func(t *testing.T) {
			st := State{Record: []Entry{c.entry}}
			got := landingRepointed(st, c.lane)
			if got["docs/a.md"] != c.want || (!c.want && len(got) != 0) {
				t.Fatalf("read %v, want the path admitted = %v", got, c.want)
			}
		})
	}
}

// A hand-back while siblings run does not send the person to delete the directory they
// use, and the discard it names works now.
func TestAHandBackMidRunLeavesTheRunsDirectoryToLater(t *testing.T) {
	f := siblingAtArm(t, false, queueRuleset("MERGE"), func(ld *Landing) bool { return !ld.RecordsDone })
	l := f.lane(t, "lane-2")
	if !strings.Contains(strings.Join(busyLanes(f.state(t)), " "), "lane-1") {
		t.Fatalf("premise: lane 1 is at work: %v", busyLanes(f.state(t)))
	}
	landWrite(t, l.Worktree, ".abcd/development/specs/open/spc-0-twin.md", landRead(t, l.Worktree, specRel))
	f.repo.Git("-C", l.Worktree, "add", "-A")
	f.repo.Git("-C", l.Worktree, "commit", "-q", "-m", "a change at the judged head")
	head := f.repo.Git("-C", l.Worktree, "rev-parse", "HEAD")
	if err := mutate(f.repo.Root(), f.runID, func(_ *os.Root, st *State) (bool, error) {
		for i := range st.Lanes {
			if st.Lanes[i].ID == l.ID {
				st.Lanes[i].HeadSHA = head
			}
		}
		return true, nil
	}); err != nil {
		t.Fatal(err)
	}
	res := f.step(t)
	if res.HandBack == nil || res.HandBack.Kind != HandBackStuck {
		t.Fatalf("the lane is handed back: %+v", res)
	}
	home := res.HandBack.Home
	discard, wait, remove := strings.Index(home, "--discard "+l.ID), strings.Index(home, "once"), strings.Index(home, runRel(f.runID))
	if discard < 0 || wait < discard || remove < wait || !strings.Contains(home, "lane-1") {
		t.Fatalf("the directory waits for the other lanes, which the text names: %s", home)
	}
	if _, err := Discard(f.repo.Root(), f.runID, l.ID, f.opts()); err != nil {
		t.Fatalf("the lane is discarded as the hand-back says, while its sibling works: %v", err)
	}
	if got := f.lane(t, "lane-1"); len(got.Awaits) != 1 {
		t.Fatalf("the sibling stays at work: %+v", got)
	}
}

// An uncommitted change left at a hand-back stops the discard, which says so and says
// to run itself again; putting the change back lets it through.
func TestAHandBackWithAChangeInTheWorktreeHasAWayOut(t *testing.T) {
	f := newLandFixtureWith(t, map[string]string{
		".abcd/work/rulesets/main-protection.json": queueRuleset("MERGE"),
		specRel: specWithSteps("1. The parser\n   - packages: internal/core/spec\n"),
	})
	l := f.validatedOnce(t)
	f.commitAtTheJudgedHead(t, l, func() {
		landWrite(t, l.Worktree, plannedIntentRel, strings.Replace(landRead(t, l.Worktree, plannedIntentRel), "\nslug:", "\nheld: \"a reason\"\nslug:", 1))
	})
	landWrite(t, l.Worktree, specRel, strings.Replace(landRead(t, l.Worktree, specRel), "A written design record.", "A written design record, settled.", 1))
	f.step(t) // prepare
	if got := currentLane(t, f.repo, f.runID); got.Landing == nil || !got.Landing.SpecSettled {
		t.Fatalf("premise: the landing carries a settled spec: %+v", got.Landing)
	}
	res, err := advance(f.repo.Root(), f.runID, f.stages, Options{})
	if err != nil || res.HandBack == nil {
		t.Fatalf("the lane is handed back: %+v %v", res, err)
	}
	if !strings.Contains(res.HandBack.Home, "uncommitted") {
		t.Fatalf("the way out says what to do with what is uncommitted: %s", res.HandBack.Home)
	}
	_, err = Discard(f.repo.Root(), f.runID, l.ID, Options{})
	if r := mustRefusal(t, err); !strings.Contains(r.Remedy, "--discard "+l.ID) {
		t.Fatalf("the refusal says to discard again, not to step: %+v", r)
	}
	f.repo.Git("-C", l.Worktree, "checkout", "-q", "--", specRel)
	if _, err := Discard(f.repo.Root(), f.runID, l.ID, Options{}); err != nil {
		t.Fatalf("the discard goes through once the change is put back: %v", err)
	}
}
