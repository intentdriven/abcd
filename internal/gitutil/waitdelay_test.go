package gitutil_test

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// committedRepo is a fixture repository with one commit, and its root commit
// and toplevel as git names them before any shim is on PATH.
func committedRepo(t *testing.T) (repo *gittest.Repo, root, top string) {
	t.Helper()
	repo = gittest.NewRepo(t)
	repo.Commit("one")
	root = repo.Git("rev-list", "--max-parents=0", "HEAD")
	top = repo.Git("rev-parse", "--show-toplevel")
	if !gitutil.IsFullSHA(root) {
		t.Fatalf("fixture: root commit %q", root)
	}
	return repo, root, top
}

// A git can answer and still have its output open when exec's WaitDelay runs
// out: on a heavily loaded machine the copy of the answer misses the delay,
// and here gittest.SlowPipeGit makes that happen without load, by leaving a
// process holding git's pipe. Both lookups once read the miss as an answer —
// "git cannot name its root", and an empty root commit (iss-2610100846469473).

// TestRootCommitWaitsOutAGitThatHoldsItsPipe: RootCommit has no deadline, so
// it waits for git's answer rather than reading a held pipe as a repository
// with no root commit. Every caller that keys a store on it (the site build's
// marker, the lane worktree store) gets the root commit.
func TestRootCommitWaitsOutAGitThatHoldsItsPipe(t *testing.T) {
	repo, want, _ := committedRepo(t)
	gittest.SlowPipeGit(t, "--max-parents=0", -1)

	if got := gitutil.RootCommit(repo.Root()); got != want {
		t.Errorf("RootCommit = %q; want %q", got, want)
	}
}

// TestToplevelWaitsOutAGitThatHoldsItsPipe: the same for the toplevel
// question, which has no deadline either.
func TestToplevelWaitsOutAGitThatHoldsItsPipe(t *testing.T) {
	repo, _, want := committedRepo(t)
	gittest.SlowPipeGit(t, "--show-toplevel", -1)

	if got, err := gitutil.Toplevel(repo.Root()); got != want || err != nil {
		t.Errorf("Toplevel = %q, %v; want %q, nil", got, err, want)
	}
}

// TestRootCommitContextRetriesAGitThatMissedItsWaitDelay: under a deadline
// the WaitDelay stays (it bounds a killed git's pipes), so a miss is asked once
// more, and the second answer is the root commit.
func TestRootCommitContextRetriesAGitThatMissedItsWaitDelay(t *testing.T) {
	repo, want, _ := committedRepo(t)
	calls := gittest.SlowPipeGit(t, "--max-parents=0", 1)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	got, err := gitutil.RootCommitContext(ctx, repo.Root())
	if got != want || err != nil {
		t.Errorf("RootCommitContext = %q, %v; want %q, nil", got, err, want)
	}
	if n := calls(); n != 2 {
		t.Errorf("git was asked %d times; want 2 (the miss and one retry)", n)
	}
}

// TestRootCommitContextNamesAPersistentWaitDelayMissATimeout: a miss on the
// retry too is reported as git not answering in time (gitutil.ErrGitTimedOut),
// never as the empty answer a repository with no root commit gives.
func TestRootCommitContextNamesAPersistentWaitDelayMissATimeout(t *testing.T) {
	repo, _, _ := committedRepo(t)
	calls := gittest.SlowPipeGit(t, "--max-parents=0", -1)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	got, err := gitutil.RootCommitContext(ctx, repo.Root())
	if got != "" || !errors.Is(err, gitutil.ErrGitTimedOut) {
		t.Fatalf("RootCommitContext = %q, %v; want \"\" and ErrGitTimedOut", got, err)
	}
	if !errors.Is(err, exec.ErrWaitDelay) {
		t.Errorf("the timeout lost the exec error it came from: %v", err)
	}
	if !strings.Contains(err.Error(), "did not answer in time") {
		t.Errorf("the error does not say git did not answer in time: %v", err)
	}
	if n := calls(); n != 2 {
		t.Errorf("git was asked %d times; want 2 (one retry, no more)", n)
	}
}

// TestToplevelContextRetriesAGitThatMissedItsWaitDelay: the same retry on the
// toplevel question under a deadline.
func TestToplevelContextRetriesAGitThatMissedItsWaitDelay(t *testing.T) {
	repo, _, want := committedRepo(t)
	calls := gittest.SlowPipeGit(t, "--show-toplevel", 1)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	got, err := gitutil.ToplevelContext(ctx, repo.Root())
	if got != want || err != nil {
		t.Errorf("ToplevelContext = %q, %v; want %q, nil", got, err, want)
	}
	if n := calls(); n != 2 {
		t.Errorf("git was asked %d times; want 2", n)
	}
}
