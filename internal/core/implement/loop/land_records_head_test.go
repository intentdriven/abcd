package loop

import (
	"strings"
	"testing"
)

// preparedLanding walks a fresh fixture's lane to the call that makes the records commit.
func preparedLanding(t *testing.T) (*landFixture, Lane) {
	t.Helper()
	f := newLandFixture(t, queueRuleset("MERGE"))
	f.validatedOnce(t)
	f.step(t) // prepare
	return f, currentLane(t, f.repo, f.runID)
}

// wantsNoRecordsTaken holds the run's state to the judged head after a refusal: no
// commit is taken for the landing's records.
func (f *landFixture) wantsNoRecordsTaken(t *testing.T, judged string) {
	t.Helper()
	got := currentLane(t, f.repo, f.runID)
	if got.Landing == nil || got.Landing.RecordsDone || got.Landing.Records != "" || got.HeadSHA != judged {
		t.Fatalf("no commit is taken for the landing's records: head %s (judged %s), %+v", got.HeadSHA, judged, got.Landing)
	}
}

// The records commit is made only from the lane's branch at the judged head: a worktree
// whose HEAD is detached, or names another branch, or whose branch moved during the
// close is refused before anything is committed.
func TestALandingRefusesAWorktreeOffItsBranchAtTheJudgedHead(t *testing.T) {
	for name, c := range map[string]struct {
		move  func(t *testing.T, f *landFixture, l Lane)
		names string
		// head and branch are where the worktree's HEAD and the lane's branch stand
		// after the refusal: at the judged head, or at its parent where the case put them.
		head, branch string
	}{
		"HEAD detached at the judged head's parent": {func(t *testing.T, f *landFixture, l Lane) {
			t.Helper()
			f.repo.Git("-C", l.Worktree, "update-ref", "--no-deref", "HEAD", l.HeadSHA+"^")
		}, "naming no branch", "parent", "judged"},
		"HEAD naming another branch at the judged head": {func(t *testing.T, f *landFixture, l Lane) {
			t.Helper()
			f.repo.Git("-C", l.Worktree, "branch", "elsewhere", l.HeadSHA)
			f.repo.Git("-C", l.Worktree, "symbolic-ref", "HEAD", "refs/heads/elsewhere")
		}, `on "refs/heads/elsewhere"`, "judged", "judged"},
		"the branch moved during the close": {func(t *testing.T, f *landFixture, l Lane) {
			t.Helper()
			beforeLandingStage = func(string) {
				f.repo.Git("-C", l.Worktree, "update-ref", "refs/heads/"+l.Branch, l.HeadSHA+"^")
			}
			t.Cleanup(func() { beforeLandingStage = nil })
		}, `on "refs/heads/`, "parent", "parent"},
	} {
		t.Run(name, func(t *testing.T) {
			f, l := preparedLanding(t)
			judged := l.HeadSHA
			at := map[string]string{"judged": judged, "parent": f.repo.Git("rev-parse", judged+"^")}
			c.move(t, f, l)
			r := f.wantsRefusedStep(t)
			if !strings.Contains(r.Reason, "HEAD "+c.names) || !strings.Contains(r.Reason, shortSHA(judged)) {
				t.Fatalf("the refusal names where HEAD stands and the judged head: %+v", r)
			}
			if got := f.repo.Git("-C", l.Worktree, "rev-parse", "HEAD"); got != at[c.head] {
				t.Fatalf("nothing is committed: HEAD %s, want the %s %s", got, c.head, at[c.head])
			}
			if got := f.repo.Git("rev-parse", "refs/heads/"+l.Branch); got != at[c.branch] {
				t.Fatalf("the branch is unchanged: %s, want the %s %s", got, c.branch, at[c.branch])
			}
			if strings.Contains(c.names, "elsewhere") {
				if got := f.repo.Git("rev-parse", "refs/heads/elsewhere"); got != judged {
					t.Fatalf("nothing is committed on the other branch: %s", got)
				}
			}
			f.wantsNoRecordsTaken(t, judged)
		})
	}
}

// The commit git made is adopted as the landing's records only as a commit found would
// be: on the branch at the worktree's HEAD, its one parent the judged head and its change
// what was judged. A hook that moves the branch or HEAD after the commit, or restages the
// index before it, is refused, and the run's state stays at the judged head.
func TestTheRecordsCommitIsCheckedBeforeItIsAdopted(t *testing.T) {
	for name, c := range map[string]struct {
		hook, body string
		names      string
	}{
		"the branch moved after the commit": {"post-commit",
			"#!/bin/sh\ngit update-ref 'refs/heads/{branch}' '{judged}^'\n",
			"which is not the landing's records commit"},
		"HEAD moved off the branch after the commit": {"post-commit",
			"#!/bin/sh\ngit update-ref --no-deref HEAD '{judged}'\n",
			"has HEAD naming no branch"},
		"the index restaged before the commit": {"pre-commit",
			"#!/bin/sh\necho 'Not the landing.' > restaged.txt\ngit add restaged.txt\n",
			"restaged.txt"},
	} {
		t.Run(name, func(t *testing.T) {
			f, l := preparedLanding(t)
			judged := l.HeadSHA
			installHook(t, f, c.hook, strings.NewReplacer("{branch}", l.Branch, "{judged}", judged).Replace(c.body))
			r := f.wantsRefusedStep(t)
			if !strings.Contains(r.Reason, c.names) {
				t.Fatalf("the refusal names what is wrong with the commit: %+v", r)
			}
			f.wantsNoRecordsTaken(t, judged)
		})
	}
}

// A records commit refused because a hook took the worktree's HEAD off the branch stays
// refused: the next call finds the commit on the branch and holds it to the worktree's
// HEAD as the call that made it did, so with nothing repaired it is refused again and
// not adopted.
func TestARecordsCommitWithHEADDetachedIsRefusedAgainWhenFound(t *testing.T) {
	for name, at := range map[string]string{
		"HEAD detached at the judged head": "{judged}",
		"HEAD detached at the new commit":  "HEAD",
	} {
		t.Run(name, func(t *testing.T) {
			f, l := preparedLanding(t)
			judged := l.HeadSHA
			installHook(t, f, "post-commit", "#!/bin/sh\ngit update-ref --no-deref HEAD \"$(git rev-parse '"+strings.ReplaceAll(at, "{judged}", judged)+"')\"\n")
			r := f.wantsRefusedStep(t)
			if !strings.Contains(r.Reason, "has HEAD ") {
				t.Fatalf("the first call refuses the commit it made, naming where HEAD stands: %+v", r)
			}
			f.wantsNoRecordsTaken(t, judged)
			tip := f.repo.Git("rev-parse", "refs/heads/"+l.Branch)
			if tip == judged {
				t.Fatalf("the commit stands on the branch for the next call to find: %s", tip)
			}
			r = f.wantsRefusedStep(t)
			if !strings.Contains(r.Reason, "has HEAD naming no branch") || !strings.Contains(r.Reason, shortSHA(tip)+", the commit found") {
				t.Fatalf("the next call refuses the commit it finds, naming where HEAD stands: %+v", r)
			}
			f.wantsNoRecordsTaken(t, judged)
		})
	}
}

// With the worktree on its branch at the judged head, the records commit is made there
// and adopted.
func TestTheRecordsCommitOnTheJudgedHeadIsAdopted(t *testing.T) {
	f, l := preparedLanding(t)
	judged := l.HeadSHA
	l = f.wantsRecordsCommit(t)
	if l.HeadSHA == judged || l.Landing.Records != l.HeadSHA {
		t.Fatalf("the records commit is adopted as the lane's head: head %s, records %s, judged %s", l.HeadSHA, l.Landing.Records, judged)
	}
	if got := f.repo.Git("rev-parse", "refs/heads/"+l.Branch); got != l.HeadSHA {
		t.Fatalf("the records commit is the branch's tip: %s, %s", got, l.HeadSHA)
	}
	if got := f.repo.Git("-C", l.Worktree, "log", "-1", "--format=%P", l.HeadSHA); got != judged {
		t.Fatalf("the records commit sits on the judged head: parents %q, judged %s", got, judged)
	}
}
