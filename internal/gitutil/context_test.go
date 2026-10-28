package gitutil_test

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// hangingGit puts a `git` first on PATH that records its pid in the returned
// file and then sleeps far past any deadline a test sets.
func hangingGit(t *testing.T) (pidFile string) {
	t.Helper()
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("no sh")
	}
	bin := t.TempDir()
	pidFile = filepath.Join(t.TempDir(), "pid")
	body := "#!/bin/sh\necho $$ > '" + pidFile + "'\nexec sleep 10\n"
	if err := os.WriteFile(filepath.Join(bin, "git"), []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	return pidFile
}

// TestRootCommitContextKillsGitAtTheDeadline: the context form of the
// root-commit question returns when its context ends, says so with the
// context's own error rather than an empty answer, and the git it started is
// gone — killed, not abandoned to finish on its own.
func TestRootCommitContextKillsGitAtTheDeadline(t *testing.T) {
	// The partial-clone guard asks the real git for its version once per
	// process; asked here first, so the hanging git below is only the
	// root-commit lookup and not the version probe the guard makes before it.
	gitutil.GitAtLeast(2, 0)
	pidFile := hangingGit(t)
	// Repository-shaped, so the lookup asks git: a directory with no .git
	// entry is answered without it.
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	start := time.Now()
	sha, err := gitutil.RootCommitContext(ctx, dir)
	took := time.Since(start)
	if sha != "" || !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("RootCommitContext = %q, %v; want \"\" and context.DeadlineExceeded", sha, err)
	}
	if took > 3*time.Second {
		t.Errorf("returned after %s; the deadline was 500ms", took)
	}
	raw, rerr := os.ReadFile(pidFile)
	if os.IsNotExist(rerr) {
		// Killed before its first line ran (a loaded machine): nothing of it
		// is left to outlive the call.
		return
	}
	if rerr != nil {
		t.Fatal(rerr)
	}
	pid, perr := strconv.Atoi(strings.TrimSpace(string(raw)))
	if perr != nil {
		t.Fatal(perr)
	}
	if err := syscall.Kill(pid, 0); err == nil {
		_ = syscall.Kill(pid, syscall.SIGKILL)
		t.Errorf("git (pid %d) is still running after the deadline", pid)
	}
}

// TestRootCommitContextAnswersAsRootCommitDoes: with time to answer, the
// context form names the same root commit, and "" with no error outside a
// repository — the total answer RootCommit gives.
func TestRootCommitContextAnswersAsRootCommitDoes(t *testing.T) {
	repo := t.TempDir()
	for _, args := range [][]string{
		{"init", "-q"},
		{"-c", "user.name=t", "-c", "user.email=t@example.invalid", "commit", "-q", "--allow-empty", "-m", "one"},
	} {
		cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
		cmd.Env = gittest.Env(t)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Skipf("git %v unavailable: %v (%s)", args, err, out)
		}
	}
	want := gittest.RootCommit(t, repo)
	if !gitutil.IsFullSHA(want) {
		t.Fatalf("fixture: RootCommit = %q", want)
	}
	if got, err := gitutil.RootCommitContext(context.Background(), repo); got != want || err != nil {
		t.Errorf("RootCommitContext = %q, %v; want %q", got, err, want)
	}
	if got, err := gitutil.RootCommitContext(context.Background(), t.TempDir()); got != "" || err != nil {
		t.Errorf("outside a repository: %q, %v; want \"\", nil", got, err)
	}
}

// TestAContextBoundReadEndsAtItsDeadlineWhileTheVersionProbeHangs: the
// partial-clone guard asks git its version before an object read, and that
// probe once ran with no context, so a git that hung held a caller on a
// deadline (the status verb, the prompt hook) for as long as git hung. The
// guard now waits for the probe or the caller's deadline, whichever ends
// first, and a probe no caller waits on any longer is cancelled, so its git
// is gone when the call returns rather than left running after a short-lived
// process exits (iss-2610181802344551).
func TestAContextBoundReadEndsAtItsDeadlineWhileTheVersionProbeHangs(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	pidFile := hangingGit(t)
	gitutil.ColdGitVersionForTest(t)
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	start := time.Now()
	sha, err := gitutil.RootCommitContext(ctx, dir)
	took := time.Since(start)
	if sha != "" || !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("RootCommitContext = %q, %v; want \"\" and context.DeadlineExceeded", sha, err)
	}
	if took > 3*time.Second {
		t.Errorf("returned after %s while git's version probe hung; the deadline was 500ms", took)
	}
	raw, rerr := os.ReadFile(pidFile)
	if os.IsNotExist(rerr) {
		// Killed before its first line ran (a loaded machine): nothing of it
		// is left to outlive the call.
		return
	}
	if rerr != nil {
		t.Fatal(rerr)
	}
	pid, perr := strconv.Atoi(strings.TrimSpace(string(raw)))
	if perr != nil {
		t.Fatal(perr)
	}
	if err := syscall.Kill(pid, 0); err == nil {
		_ = syscall.Kill(pid, syscall.SIGKILL)
		t.Errorf("the version probe's git (pid %d) is still running after the deadline", pid)
	}
}

// TestAContextBoundReadEndsAtItsDeadlineWhileTheConfigReadHangs: on a git
// below the floor the guard reads the repository's promisor config before an
// object read, and that read once ran with no context too; a git that answers
// its version and then hangs must still let the caller go at its deadline
// (iss-2610181802344551).
func TestAContextBoundReadEndsAtItsDeadlineWhileTheConfigReadHangs(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("no sh")
	}
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	bin := t.TempDir()
	body := "#!/bin/sh\nif [ \"$1\" = version ]; then echo 'git version 2.30.0'; exit 0; fi\nexec sleep 10\n"
	if err := os.WriteFile(filepath.Join(bin, "git"), []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	gitutil.ColdGitVersionForTest(t)
	if ok, _ := gitutil.GitAtLeast(2, 44); ok {
		t.Fatal("the fake git read as at or past the floor, so the guard would never read the config")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	start := time.Now()
	sha, err := gitutil.RootCommitContext(ctx, dir)
	took := time.Since(start)
	if sha != "" || !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("RootCommitContext = %q, %v; want \"\" and context.DeadlineExceeded", sha, err)
	}
	if took > 3*time.Second {
		t.Errorf("returned after %s while git's config read hung; the deadline was 500ms", took)
	}
}
