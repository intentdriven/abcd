package gitutil_test

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// iss-2610051434156079: the root-commit lookup read every failed git call as
// "no root commit", so a caller keying a store on the identity took a real
// repository for none. These pin which answers are definite ("", nil) and
// which are git not answering (an error). They ask RootCommitContext, whose
// signature the change kept, so each was run against the old body too.

// TestRootCommitOfAMissingDirectoryIsNone: a directory that is not there has
// no root commit, definitely.
func TestRootCommitOfAMissingDirectoryIsNone(t *testing.T) {
	gittest.Env(t)
	got, err := gitutil.RootCommitContext(context.Background(), filepath.Join(t.TempDir(), "gone"))
	if got != "" || err != nil {
		t.Errorf("RootCommitContext(missing directory) = %q, %v; want \"\", nil", got, err)
	}
}

// TestRootCommitOutsideAnyRepositoryIsNone: a directory with no .git entry at
// or above it has no root commit, definitely, and git is not asked.
func TestRootCommitOutsideAnyRepositoryIsNone(t *testing.T) {
	gittest.Env(t)
	gittest.ShimGit(t, []string{"--max-parents=0"}, "", 128)
	got, err := gitutil.RootCommitContext(context.Background(), t.TempDir())
	if got != "" || err != nil {
		t.Errorf("RootCommitContext(no .git above) = %q, %v; want \"\", nil", got, err)
	}
}

// TestRootCommitOfAnEmptyRepositoryIsNone: a repository with no commits yet
// has no root commit, definitely — git's own exit 128 for an unborn HEAD is
// told apart from git failing.
func TestRootCommitOfAnEmptyRepositoryIsNone(t *testing.T) {
	r := gittest.NewRepo(t)
	got, err := gitutil.RootCommitContext(context.Background(), r.Root())
	if got != "" || err != nil {
		t.Errorf("RootCommitContext(no commits) = %q, %v; want \"\", nil", got, err)
	}
}

// TestRootCommitOfARepositoryIsItsSHA: a repository with history answers
// with its root commit.
func TestRootCommitOfARepositoryIsItsSHA(t *testing.T) {
	r := gittest.NewRepo(t)
	r.Write("a.txt", "a\n")
	r.Commit("one")
	r.Write("b.txt", "b\n")
	r.Commit("two")
	want := r.Git("rev-list", "--max-parents=0", "HEAD")
	got, err := gitutil.RootCommitContext(context.Background(), r.Root())
	if got != want || err != nil {
		t.Errorf("RootCommitContext = %q, %v; want %q, nil", got, err, want)
	}
}

// TestRootCommitNamesAGitThatFailsForARepository: git exiting 128 for a
// repository that has commits (a fork refused on a loaded machine, a git
// killed early) is ErrNoRootAnswer, never ("", nil).
func TestRootCommitNamesAGitThatFailsForARepository(t *testing.T) {
	r := gittest.NewRepo(t)
	r.Write("a.txt", "a\n")
	r.Commit("one")
	gittest.ShimGit(t, []string{"--max-parents=0"}, "", 128)
	got, err := gitutil.RootCommitContext(context.Background(), r.Root())
	if got != "" || !errors.Is(err, gitutil.ErrNoRootAnswer) {
		t.Errorf("RootCommitContext(git exits 128) = %q, %v; want \"\" and ErrNoRootAnswer", got, err)
	}
}

// TestRootCommitNamesAGitThatCannotStart: a git that never runs — no
// executable git on PATH, or one killed by a signal — is ErrNoRootAnswer for
// a repository-shaped directory.
func TestRootCommitNamesAGitThatCannotStart(t *testing.T) {
	shaped := func(t *testing.T) string {
		t.Helper()
		dir := t.TempDir()
		if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
			t.Fatal(err)
		}
		return dir
	}
	t.Run("no executable git on PATH", func(t *testing.T) {
		gittest.Env(t)
		bin := t.TempDir()
		if err := os.WriteFile(filepath.Join(bin, "git"), []byte("#!/bin/sh\nexit 0\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		t.Setenv("PATH", bin)
		got, err := gitutil.RootCommitContext(context.Background(), shaped(t))
		if got != "" || !errors.Is(err, gitutil.ErrNoRootAnswer) {
			t.Errorf("RootCommitContext(git not runnable) = %q, %v; want \"\" and ErrNoRootAnswer", got, err)
		}
	})
	t.Run("a git that kills itself", func(t *testing.T) {
		gittest.Env(t)
		if _, err := exec.LookPath("sh"); err != nil {
			t.Skip("no sh")
		}
		bin := t.TempDir()
		if err := os.WriteFile(filepath.Join(bin, "git"), []byte("#!/bin/sh\nkill -9 $$\n"), 0o700); err != nil {
			t.Fatal(err)
		}
		t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
		got, err := gitutil.RootCommitContext(context.Background(), shaped(t))
		if got != "" || !errors.Is(err, gitutil.ErrNoRootAnswer) {
			t.Errorf("RootCommitContext(git killed) = %q, %v; want \"\" and ErrNoRootAnswer", got, err)
		}
	})
}

// TestRootCommitThroughASymlinkIsItsSHA: a directory reached through a symlink
// into a repository's subdirectory carries no .git entry on the path as
// written, and is still a repository; git is asked, not the path's shape.
func TestRootCommitThroughASymlinkIsItsSHA(t *testing.T) {
	r := gittest.NewRepo(t)
	r.Write("svc/api/main.txt", "x\n")
	r.Commit("one")
	want := r.Git("rev-list", "--max-parents=0", "HEAD")
	link := filepath.Join(t.TempDir(), "api")
	if err := os.Symlink(filepath.Join(r.Root(), "svc", "api"), link); err != nil {
		t.Skipf("symlink: %v", err)
	}
	got, err := gitutil.RootCommit(link)
	if got != want || err != nil {
		t.Errorf("RootCommit through a symlink = %q, %v; want %q, nil", got, err, want)
	}
}

// TestRootCommitOfABareRepositoryIsItsSHA: a bare repository has no .git
// entry at all, and still has a root commit.
func TestRootCommitOfABareRepositoryIsItsSHA(t *testing.T) {
	r := gittest.NewRepo(t)
	r.Write("a.txt", "a\n")
	r.Commit("one")
	want := r.Git("rev-list", "--max-parents=0", "HEAD")
	bare := filepath.Join(t.TempDir(), "bare.git")
	gittest.CloneBare(t, r.Root(), bare)
	got, err := gitutil.RootCommit(bare)
	if got != want || err != nil {
		t.Errorf("RootCommit of a bare repository = %q, %v; want %q, nil", got, err, want)
	}
}

// TestRootCommitOutsideAnyRepositoryWithNoGitIsNone: a plain folder is no
// repository whether or not git can run there, so git missing from PATH does
// not turn it into a repository git did not answer for.
func TestRootCommitOutsideAnyRepositoryWithNoGitIsNone(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PATH", t.TempDir())
	if got, err := gitutil.RootCommit(dir); got != "" || err != nil {
		t.Errorf("RootCommit of a plain folder with no git = %q, %v; want \"\", nil", got, err)
	}
}
