package gitutil_test

import (
	"context"
	"errors"
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

// A git can answer and still have its output open: a process it started
// holds the pipe, or, on a heavily loaded machine, the copy of its answer has
// not yet been read. exec's WaitDelay once timed that copy from git's exit,
// and both lookups read the miss as an answer — "git cannot name its root",
// and an empty root commit (iss-2610100846469473) — and then, under a
// deadline, as git not answering (iss-2610272041031418). gittest.SlowPipeGit
// makes the output close late without load, by leaving a process holding
// git's pipe for a second after git exits.

// TestRootCommitWaitsOutAGitThatHoldsItsPipe: RootCommit has no deadline, so
// it waits for git's answer rather than reading a held pipe as a repository
// with no root commit. Every caller that keys a store on it (the site build's
// marker, the lane worktree store) gets the root commit.
func TestRootCommitWaitsOutAGitThatHoldsItsPipe(t *testing.T) {
	repo, want, _ := committedRepo(t)
	gittest.SlowPipeGit(t, "--max-parents=0", -1)

	if got, err := gitutil.RootCommit(repo.Root()); got != want || err != nil {
		t.Errorf("RootCommit = %q, %v; want %q, nil", got, err, want)
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

// TestRootCommitContextReadsAGitWhoseOutputClosesLate: under a deadline the
// answer git gave is read once its output closes, however long after git
// exited, and git is asked once (iss-2610272041031418).
func TestRootCommitContextReadsAGitWhoseOutputClosesLate(t *testing.T) {
	repo, want, _ := committedRepo(t)
	calls := gittest.SlowPipeGit(t, "--max-parents=0", -1)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	got, err := gitutil.RootCommitContext(ctx, repo.Root())
	if got != want || err != nil {
		t.Errorf("RootCommitContext = %q, %v; want %q, nil", got, err, want)
	}
	if n := calls(); n != 1 {
		t.Errorf("git was asked %d times; want once", n)
	}
}

// TestRootCommitContextEndsAtItsDeadlineWhileTheOutputStaysOpen: a process
// git left holding its output cannot hold the caller past its deadline; the
// lookup answers the deadline, never the empty answer a repository with no
// root commit gives.
func TestRootCommitContextEndsAtItsDeadlineWhileTheOutputStaysOpen(t *testing.T) {
	repo, _, _ := committedRepo(t)
	gittest.SlowPipeGit(t, "--max-parents=0", -1)
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	start := time.Now()
	got, err := gitutil.RootCommitContext(ctx, repo.Root())
	if took := time.Since(start); took > 5*time.Second {
		t.Errorf("returned after %s; the deadline was 200ms", took)
	}
	if got != "" || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("RootCommitContext = %q, %v; want \"\" and context.DeadlineExceeded", got, err)
	}
}

// TestToplevelContextReadsAGitWhoseOutputClosesLate: the same read on the
// toplevel question under a deadline.
func TestToplevelContextReadsAGitWhoseOutputClosesLate(t *testing.T) {
	repo, _, want := committedRepo(t)
	calls := gittest.SlowPipeGit(t, "--show-toplevel", -1)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	got, err := gitutil.ToplevelContext(ctx, repo.Root())
	if got != want || err != nil {
		t.Errorf("ToplevelContext = %q, %v; want %q, nil", got, err, want)
	}
	if n := calls(); n != 1 {
		t.Errorf("git was asked %d times; want once", n)
	}
}
