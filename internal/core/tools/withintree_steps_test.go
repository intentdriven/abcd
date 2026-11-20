package tools

import (
	"os"
	"path/filepath"
	"testing"
)

// TestWithinTreeHoldsEachStepAtItsRealLocation pins iss-2611202034242491: a
// PATH directory outside the repository that links into it (~/alias ->
// R/scripts) reaches a committed link (R/scripts/brew) to a program outside the
// repository. Neither the written path nor the final target lies inside R, but
// the link the program is reached through really lies there, so it is
// repository content and the repository chose what runs under the name brew.
func TestWithinTreeHoldsEachStepAtItsRealLocation(t *testing.T) {
	mkdir := func(p string) string {
		t.Helper()
		if err := os.MkdirAll(p, 0o755); err != nil {
			t.Fatal(err)
		}
		return p
	}
	link := func(target, at string) {
		t.Helper()
		if err := os.Symlink(target, at); err != nil {
			t.Fatal(err)
		}
	}
	outside := mkdir(filepath.Join(t.TempDir(), "bin"))
	prog := filepath.Join(outside, "brew")
	if err := os.WriteFile(prog, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	home := t.TempDir()

	t.Run("a repository link reached through an outside directory link", func(t *testing.T) {
		repo := t.TempDir()
		scripts := mkdir(filepath.Join(repo, "scripts"))
		link(prog, filepath.Join(scripts, "brew"))
		alias := filepath.Join(home, "alias")
		link(scripts, alias)
		p := filepath.Join(alias, "brew")
		resolved, err := filepath.EvalSymlinks(p)
		if err != nil {
			t.Fatal(err)
		}
		if !WithinTree(p, resolved, repo) {
			t.Errorf("WithinTree(%q, %q, %q) = false; the link it is reached through lies inside the repository", p, resolved, repo)
		}
	})

	t.Run("an outside directory link to nothing guarded is admitted", func(t *testing.T) {
		repo := t.TempDir()
		elsewhere := mkdir(filepath.Join(t.TempDir(), "scripts"))
		link(prog, filepath.Join(elsewhere, "brew"))
		alias := filepath.Join(home, "other")
		link(elsewhere, alias)
		p := filepath.Join(alias, "brew")
		resolved, err := filepath.EvalSymlinks(p)
		if err != nil {
			t.Fatal(err)
		}
		if WithinTree(p, resolved, repo) {
			t.Errorf("WithinTree(%q, %q, %q) = true; nothing on the way lies inside the repository", p, resolved, repo)
		}
	})
}
