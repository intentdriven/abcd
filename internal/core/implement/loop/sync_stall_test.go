package loop

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// TestASyncMergeStoppedAtItsDeadlineIsUndone is iss-2610111729399920's
// review finding: a sync merge stopped at the commit deadline by a
// pre-merge-commit hook that never finishes leaves the merged result staged
// in the lane's worktree with no merge to abort, and every later step was
// refused over it. The loop undoes its own attempt: the refusal names it, the
// lane's worktree is clean at the judged head, and once the hook is settled a
// retried step syncs.
func TestASyncMergeStoppedAtItsDeadlineIsUndone(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX hook script")
	}
	f := newParFixture(t, "1. One\n2. Two\n   - needs: none\n", Options{SubAgents: strp("4")})
	f.stepUntil(t, "both implementers are out", func(st State) bool {
		return len(st.Lanes) == 2 && len(st.Lanes[0].Awaits) == 1 && len(st.Lanes[1].Awaits) == 1
	})
	f.implement(t, "lane-1", "one.txt")
	f.implement(t, "lane-2", "two.txt")
	f.roundPassed(t, "lane-1")
	f.roundPassed(t, "lane-2")
	f.landed(t, "lane-1")
	l2 := f.lane(t, "lane-2")
	judged := l2.HeadSHA

	hook := filepath.Join(f.repo.Root(), ".git", "hooks", "pre-merge-commit")
	sleeper(t, hook)
	shortCommitTimeout(t, time.Second)
	began := time.Now()
	_, err := advance(f.repo.Root(), f.runID, f.stages, f.opts())
	if took := time.Since(began); took > 30*time.Second {
		t.Fatalf("the step took %s; the deadline did not stop the merge", took)
	}
	r := mustRefusal(t, err)
	if !strings.Contains(r.Reason, "did not finish within 1s") || !strings.Contains(r.Reason, "undone") || !strings.Contains(r.Reason, shortSHA(judged)) {
		t.Fatalf("the refusal names the stalled merge and what was undone: %+v", r)
	}
	if got := f.repo.Git("-C", l2.Worktree, "rev-parse", "HEAD"); got != judged {
		t.Fatalf("the lane stays at its judged head: %s, want %s", got, judged)
	}
	if st := f.repo.Git("-C", l2.Worktree, "status", "--porcelain", "--untracked-files=all"); st != "" {
		t.Fatalf("the loop's merge attempt is undone and the worktree clean:\n%s", st)
	}
	if l := f.lane(t, "lane-2"); len(l.Syncs) != 0 || l.HeadSHA != judged {
		t.Fatalf("the stopped sync records nothing: %+v", l)
	}

	if err := os.Remove(hook); err != nil {
		t.Fatal(err)
	}
	f.step(t)
	l := f.lane(t, "lane-2")
	if len(l.Syncs) != 1 || l.HeadSHA == judged || l.Syncs[0].Conflicted {
		t.Fatalf("a retried step syncs the lane once the hook is settled: %+v", l)
	}
}

// TestASyncOverUncommittedWorkIsRefusedBeforeTheMerge: the undo of a stopped
// merge resets the lane's worktree, so a sync starts only from a clean one; a
// change the loop did not make is never merged over or reset away.
func TestASyncOverUncommittedWorkIsRefusedBeforeTheMerge(t *testing.T) {
	f := newParFixture(t, "1. One\n2. Two\n   - needs: none\n", Options{SubAgents: strp("4")})
	f.stepUntil(t, "both implementers are out", func(st State) bool {
		return len(st.Lanes) == 2 && len(st.Lanes[0].Awaits) == 1 && len(st.Lanes[1].Awaits) == 1
	})
	f.implement(t, "lane-1", "one.txt")
	f.implement(t, "lane-2", "two.txt")
	f.roundPassed(t, "lane-1")
	f.roundPassed(t, "lane-2")
	f.landed(t, "lane-1")
	l2 := f.lane(t, "lane-2")
	if err := os.WriteFile(filepath.Join(l2.Worktree, "two.txt"), []byte("hand edit\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := advance(f.repo.Root(), f.runID, f.stages, f.opts())
	r := mustRefusal(t, err)
	if !strings.Contains(r.Reason, "uncommitted") {
		t.Fatalf("the sync is refused over the uncommitted change: %+v", r)
	}
	if got, _ := os.ReadFile(filepath.Join(l2.Worktree, "two.txt")); string(got) != "hand edit\n" {
		t.Fatalf("the uncommitted change is left where it is: %q", got)
	}
	if got := f.repo.Git("-C", l2.Worktree, "rev-parse", "HEAD"); got != l2.HeadSHA {
		t.Fatalf("no merge was made: HEAD %s", got)
	}
}
