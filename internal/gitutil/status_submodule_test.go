package gitutil_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// submoduleFilterFixture is a superproject with a checked-out submodule whose
// OWN config names a clean filter for every path, the filter a script that
// records each run. The submodule's file has a stale stat, so a status that
// looks inside the submodule re-hashes it through that filter. The
// superproject configures no filter, so FilterOverrides has nothing to blank.
func submoduleFilterFixture(t *testing.T) (r *gittest.Repo, mark string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("POSIX sentinel script")
	}
	t.Setenv("HOME", t.TempDir())
	r = gittest.NewRepo(t)
	r.Write("a.txt", "a\n")
	r.Commit("seed")
	gitDir := r.AddSubmodule("sub")
	mark = filepath.Join(t.TempDir(), "sub-filter-ran")
	script := filepath.Join(t.TempDir(), "sub-clean.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\ntouch "+mark+"\ncat\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	r.Git("config", "--file", filepath.Join(gitDir, "config"), "filter.subevil.clean", script)
	if err := os.MkdirAll(filepath.Join(gitDir, "info"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(gitDir, "info", "attributes"), []byte("* filter=subevil\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// submodule.<name>.ignore=none in the superproject beats a
	// diff.ignoreSubmodules config; the flag beats both.
	r.Git("config", "submodule.sub.ignore", "none")
	staleSub(t, r)
	return r, mark
}

func staleSub(t *testing.T, r *gittest.Repo) {
	t.Helper()
	later := time.Now().Add(time.Hour)
	if err := os.Chtimes(filepath.Join(r.Root(), "sub", "s.txt"), later, later); err != nil {
		t.Fatal(err)
	}
}

// TestStatusRunsNoSubmoduleContentFilter is iss-2610091935327982: without
// --ignore-submodules git status starts a status inside each checked-out
// submodule, which reads the submodule's own config, so a clean filter the
// submodule names runs although Status blanked every superproject filter. A
// changed submodule pointer is still listed.
func TestStatusRunsNoSubmoduleContentFilter(t *testing.T) {
	r, mark := submoduleFilterFixture(t)

	// The fixture is live: a plain git status runs the submodule's filter.
	plain := exec.Command("git", "-C", r.Root(), "status", "--porcelain")
	plain.Env = r.Env()
	if out, err := plain.CombinedOutput(); err != nil {
		t.Fatalf("fixture: git status: %v\n%s", err, out)
	}
	if !ran(mark) {
		t.Fatal("fixture: a plain git status did not run the submodule's clean filter, so this test proves nothing")
	}
	if err := os.Remove(mark); err != nil {
		t.Fatal(err)
	}
	staleSub(t, r)

	entries, err := gitutil.Status(r.Root(), 1<<20, gitutil.StatusOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if ran(mark) {
		t.Fatal("Status ran the submodule's own clean filter")
	}
	if len(entries) != 0 {
		t.Fatalf("an unchanged submodule reads as unchanged: %+v", entries)
	}

	// A new commit checked out inside the submodule moves its pointer, which
	// the superproject still sees.
	sub := exec.Command("git", "-C", filepath.Join(r.Root(), "sub"),
		"-c", "user.name=F", "-c", "user.email=f@example.invalid", "-c", "commit.gpgsign=false",
		"-c", "filter.subevil.clean=", "commit", "-q", "--allow-empty", "-m", "moved")
	sub.Env = r.Env()
	if out, err := sub.CombinedOutput(); err != nil {
		t.Fatalf("moving the submodule: %v\n%s", err, out)
	}
	_ = os.Remove(mark)
	entries, err = gitutil.Status(r.Root(), 1<<20, gitutil.StatusOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if ran(mark) {
		t.Fatal("Status ran the submodule's own clean filter over a moved pointer")
	}
	if len(entries) != 1 || entries[0].Path != "sub" {
		t.Fatalf("a moved submodule pointer must still be listed: %+v", entries)
	}
}

// TestAStatusThroughStatusArgsStartsNothingInASubmodule: a status read that
// takes its command line from StatusArgs, whichever runner runs it: the
// isolated one (a run's view of a lane's worktree), the person's own
// (WorkingTreeCommand: a lane's
// clean-tree checks before a sync, a pick commit and a landing, the land
// verb's), and the one bound to a deadline (WorkingTreeCommandContext: the
// read before a merged lane's worktree is removed). None starts a status
// inside the checked-out submodule, so the filter its own settings name does
// not run, and a submodule whose checked-out commit moved is still listed.
// Watched failing with StatusArgs giving no --ignore-submodules: each runner
// ran the submodule's clean filter.
func TestAStatusThroughStatusArgsStartsNothingInASubmodule(t *testing.T) {
	r, mark := submoduleFilterFixture(t)
	runners := []struct {
		name string
		run  func(args ...string) (string, error)
	}{
		{"Run", func(args ...string) (string, error) {
			return gitutil.Run(r.Root(), append([]string{"--no-optional-locks"}, args...)...)
		}},
		{"WorkingTreeCommand", func(args ...string) (string, error) {
			return gitutil.RunTreeCommand(gitutil.WorkingTreeCommand(r.Root(), args...), args)
		}},
		{"WorkingTreeCommandContext", func(args ...string) (string, error) {
			return gitutil.RunTreeCommand(gitutil.WorkingTreeCommandContext(context.Background(), r.Root(), args...), args)
		}},
	}
	for _, moved := range []bool{false, true} {
		if moved {
			sub := exec.Command("git", "-C", filepath.Join(r.Root(), "sub"),
				"-c", "user.name=F", "-c", "user.email=f@example.invalid", "-c", "commit.gpgsign=false",
				"-c", "filter.subevil.clean=", "commit", "-q", "--allow-empty", "-m", "moved")
			sub.Env = r.Env()
			if out, err := sub.CombinedOutput(); err != nil {
				t.Fatalf("moving the submodule: %v\n%s", err, out)
			}
		}
		for _, run := range runners {
			_ = os.Remove(mark)
			staleSub(t, r)
			out, err := run.run(gitutil.StatusArgs("--porcelain", "-z", "--untracked-files=all")...)
			if err != nil {
				t.Fatalf("%s: %v", run.name, err)
			}
			if ran(mark) {
				t.Errorf("%s: the status started a status inside the submodule, and its own clean filter ran", run.name)
			}
			// Run gives git's answer with its outer space trimmed.
			if want := map[bool]string{false: "", true: "M sub\x00"}[moved]; strings.TrimSpace(out) != want {
				t.Errorf("%s, submodule moved %v: status %q, want %q", run.name, moved, out, want)
			}
		}
	}
}

// TestAWorkingTreeDiffThroughDiffWorktreeArgsStartsNothingInASubmodule: a
// diff of the index with the working tree, like a status, starts a status
// inside each checked-out submodule whose commit has not moved, so the
// filter the submodule's own settings name runs, as the plain diff here
// shows; one whose command line DiffWorktreeArgs gives starts nothing there.
// A moved submodule is still listed. Watched failing with DiffWorktreeArgs
// giving no --ignore-submodules: the diff ran the submodule's filter.
func TestAWorkingTreeDiffThroughDiffWorktreeArgsStartsNothingInASubmodule(t *testing.T) {
	r, mark := submoduleFilterFixture(t)
	plain := exec.Command("git", "-C", r.Root(), "diff", "--name-only", "-z")
	plain.Env = r.Env()
	if out, err := plain.CombinedOutput(); err != nil {
		t.Fatalf("fixture: git diff: %v\n%s", err, out)
	}
	if !ran(mark) {
		t.Fatal("fixture: a plain git diff of the working tree did not run the submodule's clean filter, so this test proves nothing")
	}
	for _, moved := range []bool{false, true} {
		if moved {
			sub := exec.Command("git", "-C", filepath.Join(r.Root(), "sub"),
				"-c", "user.name=F", "-c", "user.email=f@example.invalid", "-c", "commit.gpgsign=false",
				"-c", "filter.subevil.clean=", "commit", "-q", "--allow-empty", "-m", "moved")
			sub.Env = r.Env()
			if out, err := sub.CombinedOutput(); err != nil {
				t.Fatalf("moving the submodule: %v\n%s", err, out)
			}
		}
		_ = os.Remove(mark)
		staleSub(t, r)
		args := gitutil.DiffWorktreeArgs("--name-only", "-z")
		out, err := gitutil.RunTreeCommand(gitutil.WorkingTreeCommand(r.Root(), args...), args)
		if err != nil {
			t.Fatal(err)
		}
		if ran(mark) {
			t.Errorf("submodule moved %v: the diff ran the submodule's own clean filter", moved)
		}
		if want := map[bool]string{false: "", true: "sub\x00"}[moved]; out != want {
			t.Errorf("submodule moved %v: diff %q, want %q", moved, out, want)
		}
	}
}

// TestMovedGitlinksRunsNoProgramAndListsAMovedSubmodule: MovedGitlinks reads
// the whole tree with diff-files, which starts nothing inside a checked-out
// submodule and hashes a racily clean file of the superproject, so a filter
// the superproject marks required is blanked and switched off: neither
// filter runs and the read does not fail. A submodule checked out at its
// recorded commit is not listed; one moved is. Watched failing with
// filterBlanks giving nothing: the superproject's filter ran.
func TestMovedGitlinksRunsNoProgramAndListsAMovedSubmodule(t *testing.T) {
	r, mark := submoduleFilterFixture(t)
	top := filepath.Join(t.TempDir(), "top-filter-ran")
	script := filepath.Join(t.TempDir(), "top-clean.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\ntouch "+top+"\ncat\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	r.Git("config", "filter.top.clean", script)
	r.Git("config", "filter.top.required", "true")
	r.Write(".gitattributes", "a.txt filter=top\n")
	r.Git("-c", "filter.subevil.clean=", "add", "-A")
	r.Git("commit", "-q", "-m", "the superproject's own filter")
	_ = os.Remove(top)
	_ = os.Remove(mark)
	staleSub(t, r)
	racy := func() {
		t.Helper()
		past := time.Now().Add(-48 * time.Hour)
		if err := os.Chtimes(filepath.Join(r.Root(), ".git", "index"), past, past); err != nil {
			t.Fatal(err)
		}
	}
	racy()
	plain := exec.Command("git", "-C", r.Root(), "diff-files", "--ignore-submodules=dirty", "-z")
	plain.Env = r.Env()
	if out, err := plain.CombinedOutput(); err != nil {
		t.Fatalf("fixture: git diff-files: %v\n%s", err, out)
	}
	if !ran(top) {
		t.Fatal("fixture: a plain diff-files did not run the superproject's filter on a racily clean file, so this test proves nothing")
	}
	_ = os.Remove(top)
	_ = os.Remove(mark)
	racy()
	moved, err := gitutil.MovedGitlinks(r.Root(), "")
	if err != nil {
		t.Fatal(err)
	}
	if ran(top) || ran(mark) {
		t.Errorf("MovedGitlinks ran a filter: the superproject's %v, the submodule's %v", ran(top), ran(mark))
	}
	if len(moved) != 0 {
		t.Errorf("a submodule at its recorded commit is listed as moved: %v", moved)
	}
	r.Git("-C", "sub", "-c", "filter.subevil.clean=", "commit", "-q", "--allow-empty", "-m", "moved")
	moved, err = gitutil.MovedGitlinks(r.Root(), "")
	if err != nil {
		t.Fatal(err)
	}
	if len(moved) != 1 || !moved["sub"] {
		t.Errorf("a moved submodule is listed: %v", moved)
	}
}
