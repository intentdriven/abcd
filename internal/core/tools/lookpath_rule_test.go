//go:build unix

package tools

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// TestLookPathRefusesAProgramInsideTheWorkingTree pins tools.LookPath, the
// PATH lookup every program abcd runs by name goes through: a result inside
// the working tree of the process's working directory, or of the directory
// the caller names, is refused with ErrInsideWorkingTree, while one outside
// both is returned as PATH gives it.
func TestLookPathRefusesAProgramInsideTheWorkingTree(t *testing.T) {
	r := newRig(t)
	sub, prog := checkoutWithSubdir(t, r, "gh")
	accountHomeIs(t, r.home)
	withHome(t, r.home, func(string) (string, error) { return prog, nil })
	outside := t.TempDir()

	wd := workingDir
	t.Cleanup(func() { workingDir = wd })
	workingDir = func() (string, error) { return sub, nil }
	if _, err := LookPath("gh", ""); !errors.Is(err, ErrInsideWorkingTree) {
		t.Errorf("LookPath from a subdirectory of the checkout = %v, want ErrInsideWorkingTree", err)
	}
	if _, err := Resolve("gh"); !errors.Is(err, ErrInsideWorkingTree) {
		t.Errorf("Resolve from a subdirectory of the checkout = %v, want ErrInsideWorkingTree", err)
	}

	workingDir = func() (string, error) { return outside, nil }
	if _, err := LookPath("gh", sub); !errors.Is(err, ErrInsideWorkingTree) {
		t.Errorf("LookPath for a directory in the checkout = %v, want ErrInsideWorkingTree", err)
	}
	if got, err := LookPath("gh", ""); err != nil || got != prog {
		t.Errorf("LookPath from outside the checkout = %q, %v; want %q", got, err, prog)
	}
}

// TestLookPathRefusesAProgramInTheMainCheckoutFromALinkedWorktree: run from
// a linked worktree, LookPath holds a program to the main checkout of the
// same repository as well, the one the worktree's .git file leads to, so a
// program in the main checkout's bin folder is refused there too.
func TestLookPathRefusesAProgramInTheMainCheckoutFromALinkedWorktree(t *testing.T) {
	r := newRig(t)
	_, prog := checkoutWithSubdir(t, r, "make")
	admin := filepath.Join(r.repo, ".git", "worktrees", "wt")
	wt := filepath.Join(t.TempDir(), "wt")
	for _, d := range []string{admin, wt} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for p, s := range map[string]string{
		filepath.Join(wt, ".git"):         "gitdir: " + admin + "\n",
		filepath.Join(admin, "commondir"): "../..\n",
		filepath.Join(admin, "gitdir"):    filepath.Join(wt, ".git") + "\n",
	} {
		if err := os.WriteFile(p, []byte(s), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	accountHomeIs(t, r.home)
	withHome(t, r.home, func(string) (string, error) { return prog, nil })
	wd := workingDir
	t.Cleanup(func() { workingDir = wd })
	workingDir = func() (string, error) { return t.TempDir(), nil }

	if _, err := LookPath("make", wt); !errors.Is(err, ErrInsideWorkingTree) {
		t.Errorf("LookPath from a linked worktree for a program in its main checkout = %v, want ErrInsideWorkingTree", err)
	}
	workingDir = func() (string, error) { return wt, nil }
	if _, err := LookPath("make", ""); !errors.Is(err, ErrInsideWorkingTree) {
		t.Errorf("LookPath with a linked worktree as the working directory = %v, want ErrInsideWorkingTree", err)
	}
}
