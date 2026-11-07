package gitutil_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// gitLogShim puts a git on PATH that notes each command line in a private file
// and then runs the real one, and returns a reader of the lines noted so far,
// leaving out the version probe a process makes once.
func gitLogShim(t *testing.T) func() []string {
	t.Helper()
	real, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git not on PATH")
	}
	dir := t.TempDir()
	log := filepath.Join(dir, "git.log")
	shim := "#!/bin/sh\nprintf '%s\\n' \"$*\" >> '" + log + "'\nexec '" + real + "' \"$@\"\n"
	if err := os.WriteFile(filepath.Join(dir, "git"), []byte(shim), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return func() []string {
		data, err := os.ReadFile(log)
		if err != nil && !os.IsNotExist(err) {
			t.Fatal(err)
		}
		var lines []string
		for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
			if line != "" && !strings.HasSuffix(line, "version") {
				lines = append(lines, line)
			}
		}
		return lines
	}
}

// TestATreeListingStartsOneGit: the listing every state read makes starts one
// git, not a question whether this is a working tree and then the listing. The
// loop package spent 44% of its git processes here.
func TestATreeListingStartsOneGit(t *testing.T) {
	repo := newRepo(t, "")
	if err := os.WriteFile(filepath.Join(repo, "a.md"), []byte("a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	commitAll(t, repo)
	started := gitLogShim(t)
	got, err := gitutil.TrackedInTreeContext(context.Background(), repo)
	if err != nil || !slices.Contains(got, "a.md") {
		t.Fatalf("TrackedInTreeContext = %q, %v; want a list naming a.md", got, err)
	}
	lines := started()
	if len(lines) != 1 || !strings.Contains(lines[0], "ls-files") {
		t.Fatalf("the listing started %d gits, want one ls-files:\n%s", len(lines), strings.Join(lines, "\n"))
	}
}

// TestATreeListingAnswersAsTheTwoQuestionsDid holds the one-process listing to
// the answers of git asked twice, in every shape a tree can have. The dangerous
// way to be wrong is a "nothing tracked" over a tree git would not answer for,
// since a tracked file would then read as this checkout's own, so each shape
// also names its answer outright: a not-a-repository lists nothing and refuses
// nothing, and a refused or unreadable repository is an error.
func TestATreeListingAnswersAsTheTwoQuestionsDid(t *testing.T) {
	const couldNotRead = "git could not read the repository here"
	tracked := func(t *testing.T) string {
		t.Helper()
		repo := newRepo(t, "")
		if err := os.WriteFile(filepath.Join(repo, "a.md"), []byte("a\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		commitAll(t, repo)
		return repo
	}
	git := func(t *testing.T, dir string, args ...string) {
		t.Helper()
		if out, err := runGit(t, dir, args...); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	for _, tc := range []struct {
		name string
		top  func(t *testing.T) string
		// files is the list wanted, by what it must hold; empty wants none.
		files []string
		// failure is a word the error must carry; "" wants no error.
		failure string
	}{
		{name: "a plain repository", top: tracked, files: []string{"a.md"}},
		{name: "not a repository", top: func(t *testing.T) string { return t.TempDir() }},
		{name: "a directory that is not there", top: func(t *testing.T) string { return filepath.Join(t.TempDir(), "gone") }},
		{name: "a .git folder holding nothing", top: func(t *testing.T) string {
			dir := t.TempDir()
			if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
				t.Fatal(err)
			}
			return dir
		}, failure: couldNotRead},
		{name: "a .git file naming nothing", top: func(t *testing.T) string {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, ".git"), []byte("gitdir: /nowhere\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			return dir
		}, failure: couldNotRead},
		{name: "a repository git refuses over its owner", top: func(t *testing.T) string {
			repo := tracked(t)
			t.Setenv("GIT_TEST_ASSUME_DIFFERENT_OWNER", "1")
			return repo
		}, failure: couldNotRead},
		{name: "git absent from PATH in a repository", top: func(t *testing.T) string {
			repo := tracked(t)
			t.Setenv("PATH", t.TempDir())
			return repo
		}, failure: couldNotRead},
		{name: "git absent from PATH in no repository", top: func(t *testing.T) string {
			dir := t.TempDir()
			t.Setenv("PATH", t.TempDir())
			return dir
		}},
		{name: "an index git cannot read", top: func(t *testing.T) string {
			repo := tracked(t)
			if err := os.WriteFile(filepath.Join(repo, ".git", "index"), []byte("garbage"), 0o644); err != nil {
				t.Fatal(err)
			}
			return repo
		}, failure: "exit status"},
		{name: "a linked worktree", top: func(t *testing.T) string {
			repo := tracked(t)
			wt := filepath.Join(t.TempDir(), "linked")
			git(t, repo, "worktree", "add", "-q", "-b", "lane", wt)
			if err := os.WriteFile(filepath.Join(wt, "only-here.md"), []byte("b\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			commitAll(t, wt)
			return wt
		}, files: []string{"a.md", "only-here.md"}},
		{name: "a linked worktree whose repository is gone", top: func(t *testing.T) string {
			repo := tracked(t)
			wt := filepath.Join(t.TempDir(), "linked")
			git(t, repo, "worktree", "add", "-q", "-b", "lane", wt)
			if err := os.RemoveAll(filepath.Join(repo, ".git")); err != nil {
				t.Fatal(err)
			}
			return wt
		}, failure: couldNotRead},
		{name: "core.bare set in a checkout", top: func(t *testing.T) string {
			repo := tracked(t)
			git(t, repo, "config", "core.bare", "true")
			return repo
		}, files: []string{"a.md"}},
		{name: "core.worktree set elsewhere", top: func(t *testing.T) string {
			repo := tracked(t)
			git(t, repo, "config", "core.worktree", t.TempDir())
			return repo
		}, files: []string{"a.md"}},
		{name: "a bare repository", top: func(t *testing.T) string {
			dir := t.TempDir()
			gittest.InitBare(t, dir)
			return dir
		}},
		{name: "a repository with no commits", top: func(t *testing.T) string {
			dir := t.TempDir()
			git(t, dir, "init", "-q")
			return dir
		}},
		{name: "a subdirectory of a checkout", top: func(t *testing.T) string {
			repo := tracked(t)
			sub := filepath.Join(repo, "sub")
			if err := os.Mkdir(sub, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(sub, "b.md"), []byte("b\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			commitAll(t, repo)
			return sub
		}, files: []string{"a.md", "sub/b.md"}},
		{name: "the .git directory itself", top: func(t *testing.T) string {
			return filepath.Join(tracked(t), ".git")
		}, files: []string{"a.md"}},
		{name: "a relative path", top: func(t *testing.T) string {
			repo := tracked(t)
			t.Chdir(filepath.Dir(repo))
			return filepath.Base(repo)
		}, failure: couldNotRead},
	} {
		t.Run(tc.name, func(t *testing.T) {
			top := tc.top(t)
			ctx := context.Background()
			got, err := gitutil.TrackedInTreeContext(ctx, top)
			before, beforeErr := gitutil.TrackedInTreeAskedTwice(ctx, top)
			if !slices.Equal(got, before) || (got == nil) != (before == nil) || errText(err) != errText(beforeErr) {
				t.Errorf("one question answered %q, %v; two questions answered %q, %v", got, err, before, beforeErr)
			}
			if tc.failure == "" && err != nil {
				t.Errorf("error %v; want none", err)
			}
			if tc.failure != "" && (err == nil || !strings.Contains(err.Error(), tc.failure)) {
				t.Errorf("error %v; want one saying %q", err, tc.failure)
			}
			if len(tc.files) == 0 && len(got) != 0 {
				t.Errorf("listed %q; want nothing", got)
			}
			for _, want := range tc.files {
				if !slices.Contains(got, want) {
					t.Errorf("listed %q; want it to name %s", got, want)
				}
			}
		})
	}
}

// TestATreeListingOverAnEndedContext: a context that ended is git not
// answering, the same either way: an error in a repository, nothing outside one.
func TestATreeListingOverAnEndedContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	repo := newRepo(t, "")
	if _, err := gitutil.TrackedInTreeContext(ctx, repo); err == nil || !strings.Contains(err.Error(), "git could not read") {
		t.Errorf("in a repository: error %v; want git could not read it", err)
	}
	if got, err := gitutil.TrackedInTreeContext(ctx, t.TempDir()); got != nil || err != nil {
		t.Errorf("outside a repository: %q, %v; want nothing and no error", got, err)
	}
}

func errText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
