package loop

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/gittest"
)

// shortCommitTimeout bounds commitGit at d for the rest of the test.
func shortCommitTimeout(t *testing.T, d time.Duration) {
	t.Helper()
	was := commitTimeout
	commitTimeout = d
	t.Cleanup(func() { commitTimeout = was })
}

// stalledCommitRepo is a repository with an identity, one commit and a change
// to a.md ready for the pick commit.
func stalledCommitRepo(t *testing.T) *gittest.Repo {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("POSIX stub script")
	}
	withoutPersonalConfig(t)
	r := gittest.NewRepo(t)
	r.Git("config", "user.name", "Fixture")
	r.Git("config", "user.email", "fixture@example.invalid")
	r.Write("a.md", "a\n")
	r.Commit("seed")
	return r
}

// sleeper writes a program that waits far past any test's deadline, as a GUI
// pinentry nobody answers or a hook that blocks does.
func sleeper(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, []byte("#!/bin/sh\nexec sleep 60\n"), 0o755); err != nil {
		t.Fatal(err)
	}
}

// assertStopped checks that a commitGit-run command came back within the
// deadline's reach, refused with the reason, and left HEAD where it was.
func assertStopped(t *testing.T, r *gittest.Repo, head string, began time.Time, err error) {
	t.Helper()
	if took := time.Since(began); took > 20*time.Second {
		t.Fatalf("the command took %s; the deadline did not stop it", took)
	}
	if err == nil {
		t.Fatal("a commit whose signing program or hook never finished was not refused")
	}
	if !strings.Contains(err.Error(), "did not finish within 1s") {
		t.Fatalf("the refusal does not name the deadline: %v", err)
	}
	if got := r.Git("rev-parse", "HEAD"); got != head {
		t.Fatalf("the stopped command moved HEAD to %s", got)
	}
}

// TestAPickCommitWhoseSigningNeverFinishesIsRefused is the deadline half of
// iss-2610111729399920: the repository's signing program runs on the pick
// commit, and one that never answers (a GUI pinentry nobody is at) must not
// hold `abcd implement step`, an unattended resume included, forever.
func TestAPickCommitWhoseSigningNeverFinishesIsRefused(t *testing.T) {
	r := stalledCommitRepo(t)
	signer := filepath.Join(t.TempDir(), "stalled-gpg.sh")
	sleeper(t, signer)
	r.Git("config", "commit.gpgsign", "true")
	r.Git("config", "gpg.program", signer)
	shortCommitTimeout(t, time.Second)

	head := r.Git("rev-parse", "HEAD")
	r.Write("a.md", "picked\n")
	began := time.Now()
	_, err := commitGit(r.Root(), "commit", "-q", "-m", "pick", "--", "a.md")
	assertStopped(t, r, head, began, err)
}

// TestAPickCommitAndASyncMergeWhoseHookNeverFinishesAreRefused: a hook that
// blocks is stopped at the deadline on the pick commit and on a sync's merge,
// and git is given the chance to remove its lock, so the next step is not
// refused over a stale index.lock.
func TestAPickCommitAndASyncMergeWhoseHookNeverFinishesAreRefused(t *testing.T) {
	r := stalledCommitRepo(t)
	r.Git("branch", "side")
	r.Git("checkout", "-q", "side")
	r.Write("b.txt", "b\n")
	r.Commit("side")
	r.Git("checkout", "-q", "main")
	for _, h := range []string{"pre-commit", "pre-merge-commit"} {
		sleeper(t, filepath.Join(r.Root(), ".git", "hooks", h))
	}
	shortCommitTimeout(t, time.Second)

	head := r.Git("rev-parse", "HEAD")
	r.Write("a.md", "picked\n")
	began := time.Now()
	_, err := commitGit(r.Root(), "commit", "-q", "-m", "pick", "--", "a.md")
	assertStopped(t, r, head, began, err)
	if _, err := os.Lstat(filepath.Join(r.Root(), ".git", "index.lock")); err == nil {
		t.Fatal("the stopped commit left index.lock behind")
	}

	r.Git("checkout", "-q", "--", "a.md")
	began = time.Now()
	err = syncMerge(r.Root(), "sync", "side")
	assertStopped(t, r, head, began, err)
}
