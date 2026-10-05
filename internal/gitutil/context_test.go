package gitutil

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
	pidFile := hangingGit(t)
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	start := time.Now()
	sha, err := RootCommitContext(ctx, t.TempDir())
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
		cmd.Env = gitEnv()
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Skipf("git %v unavailable: %v (%s)", args, err, out)
		}
	}
	want := RootCommit(repo)
	if !IsFullSHA(want) {
		t.Fatalf("fixture: RootCommit = %q", want)
	}
	if got, err := RootCommitContext(context.Background(), repo); got != want || err != nil {
		t.Errorf("RootCommitContext = %q, %v; want %q", got, err, want)
	}
	if got, err := RootCommitContext(context.Background(), t.TempDir()); got != "" || err != nil {
		t.Errorf("outside a repository: %q, %v; want \"\", nil", got, err)
	}
}
