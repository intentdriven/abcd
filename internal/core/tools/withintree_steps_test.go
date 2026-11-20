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

// TestWalkRefusesATargetTheSystemReadsOtherwise: /out/tool links to
// d/../x and /out/d links to R/sub, so the system reads the target as R/sub/..
// then x, which is R/x, a committed link to a program outside R. Read
// lexically the target is /out/x, which does not exist, and a walk of that
// reading never reaches R/x. The walk's last step must be where the system's
// own resolution ends, or the steps cannot be examined and the program counts
// as inside. A ".." with no directory link before it is read the same way by
// both and stays admitted.
func TestWalkRefusesATargetTheSystemReadsOtherwise(t *testing.T) {
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
	real := filepath.Join(mkdir(filepath.Join(t.TempDir(), "usr", "bin")), "real")
	if err := os.WriteFile(real, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	t.Run("a dot-dot after an outside directory link into the repository", func(t *testing.T) {
		repo := t.TempDir()
		mkdir(filepath.Join(repo, "sub"))
		link(real, filepath.Join(repo, "x"))
		out := t.TempDir()
		link(filepath.Join(repo, "sub"), filepath.Join(out, "d"))
		p := filepath.Join(out, "tool")
		link("d/../x", p)
		resolved, err := filepath.EvalSymlinks(p)
		if err != nil {
			t.Fatal(err)
		}
		if route, err := Walk(p); err == nil {
			t.Errorf("Walk(%q) = steps %q, nil; the walk does not end where the system's resolution does", p, append(route.Links, route.Real))
		}
		if !WithinTree(p, resolved, repo) {
			t.Errorf("WithinTree(%q, %q, %q) = false; the system reaches it through R/x", p, resolved, repo)
		}
	})

	t.Run("a dot-dot through a real directory is admitted", func(t *testing.T) {
		repo := t.TempDir()
		out := t.TempDir()
		mkdir(filepath.Join(out, "d"))
		link(real, filepath.Join(out, "x"))
		p := filepath.Join(out, "tool")
		link("d/../x", p)
		resolved, err := filepath.EvalSymlinks(p)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Walk(p); err != nil {
			t.Errorf("Walk(%q): %v; both readings end at the same program", p, err)
		}
		if WithinTree(p, resolved, repo) {
			t.Errorf("WithinTree(%q, %q, %q) = true; nothing on the way lies inside the repository", p, resolved, repo)
		}
	})
}
